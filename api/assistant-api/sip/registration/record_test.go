package sip_registration

import (
	"context"
	"fmt"
	"strings"
	"testing"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/pkg/commons"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type testPostgresConnector struct {
	db *gorm.DB
}

func (t *testPostgresConnector) Connect(ctx context.Context) error             { return nil }
func (t *testPostgresConnector) Name() string                                  { return "test-postgres" }
func (t *testPostgresConnector) IsConnected(ctx context.Context) bool          { return t.db != nil }
func (t *testPostgresConnector) Disconnect(ctx context.Context) error          { return nil }
func (t *testPostgresConnector) Query(ctx context.Context, qry string, dest interface{}) error {
	return t.db.WithContext(ctx).Raw(qry).Scan(dest).Error
}
func (t *testPostgresConnector) DB(ctx context.Context) *gorm.DB { return t.db.WithContext(ctx) }

func newTestManager(t *testing.T) (*Manager, *gorm.DB, context.Context) {
	t.Helper()
	ctx := context.Background()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to create sqlite db: %v", err)
	}
	schema := []string{
		`CREATE TABLE assistant_phone_deployments (
			id INTEGER PRIMARY KEY,
			assistant_id BIGINT,
			status TEXT,
			telephony_provider TEXT
		)`,
		`CREATE TABLE assistant_deployment_telephony_options (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_date DATETIME,
			assistant_deployment_telephony_id BIGINT NOT NULL,
			key TEXT NOT NULL,
			value TEXT NOT NULL,
			status TEXT,
			created_by BIGINT,
			updated_by BIGINT,
			updated_date DATETIME
		)`,
		`CREATE UNIQUE INDEX idx_adto_deployment_key
			ON assistant_deployment_telephony_options(assistant_deployment_telephony_id, key)`,
	}
	for _, ddl := range schema {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("failed to initialize schema: %v", err)
		}
	}

	logger, err := commons.NewApplicationLogger(
		commons.EnableConsole(true),
		commons.EnableFile(false),
		commons.Level("error"),
	)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	return &Manager{
		logger:   logger,
		postgres: &testPostgresConnector{db: db},
	}, db, ctx
}

func insertSIPDeployment(t *testing.T, db *gorm.DB, deploymentID, assistantID uint64, phone, sipStatus string) {
	t.Helper()

	if err := db.Exec(
		`INSERT INTO assistant_phone_deployments (id, assistant_id, status, telephony_provider)
		 VALUES (?, ?, ?, ?)`,
		deploymentID, assistantID, "ACTIVE", "sip",
	).Error; err != nil {
		t.Fatalf("failed creating deployment: %v", err)
	}

	options := map[string]string{
		"phone":               phone,
		"rapida.credential_id": "101",
		"rapida.sip_inbound":  "true",
	}
	if sipStatus != "" {
		options["rapida.sip_status"] = sipStatus
	}
	for k, v := range options {
		if err := db.Exec(
			`INSERT INTO assistant_deployment_telephony_options
			 (assistant_deployment_telephony_id, key, value, status, created_by, updated_by)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			deploymentID, k, v, "ACTIVE", 1, 1,
		).Error; err != nil {
			t.Fatalf("failed creating option %s: %v", k, err)
		}
	}
}

func getOptionValue(t *testing.T, db *gorm.DB, deploymentID uint64, key string) string {
	t.Helper()
	var opt internal_assistant_entity.AssistantDeploymentTelephonyOption
	if err := db.Where("assistant_deployment_telephony_id = ? AND key = ?", deploymentID, key).First(&opt).Error; err != nil {
		t.Fatalf("failed loading option %s for deployment %d: %v", key, deploymentID, err)
	}
	return opt.Value
}

func TestLoadRecords_PrePipelineDedupe_PrefersActiveAndMarksDropped(t *testing.T) {
	m, db, ctx := newTestManager(t)

	// Duplicate DID differing only by '+' formatting.
	insertSIPDeployment(t, db, 1001, 501, "+15551234567", StatusActive)
	insertSIPDeployment(t, db, 1002, 502, "15551234567", "")

	records, err := m.loadRecords(ctx)
	if err != nil {
		t.Fatalf("loadRecords returned error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 deduped record, got %d", len(records))
	}
	if records[0].DeploymentID != 1001 {
		t.Fatalf("expected active deployment 1001 to win, got %d", records[0].DeploymentID)
	}

	status := getOptionValue(t, db, 1002, OptKeySIPStatus)
	if status != StatusConfigError {
		t.Fatalf("expected loser status=%s, got %s", StatusConfigError, status)
	}
	reason := getOptionValue(t, db, 1002, OptKeySIPError)
	if !strings.Contains(reason, "Duplicate DID +15551234567") || !strings.Contains(reason, fmt.Sprintf("deployment=%d", uint64(1001))) {
		t.Fatalf("unexpected loser reason: %s", reason)
	}
	retry := getOptionValue(t, db, 1002, OptKeySIPRetry)
	if retry != "0" {
		t.Fatalf("expected loser retry_count=0, got %s", retry)
	}
}

func TestLoadRecords_PrePipelineDedupe_PrefersLatestWhenNoActive(t *testing.T) {
	m, db, ctx := newTestManager(t)

	insertSIPDeployment(t, db, 2001, 601, "+14155550100", "")
	insertSIPDeployment(t, db, 2002, 602, "14155550100", "")

	records, err := m.loadRecords(ctx)
	if err != nil {
		t.Fatalf("loadRecords returned error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 deduped record, got %d", len(records))
	}
	if records[0].DeploymentID != 2002 {
		t.Fatalf("expected latest deployment 2002 to win, got %d", records[0].DeploymentID)
	}
}
