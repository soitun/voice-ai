package plugins_calendar

import (
	"context"
	"errors"
	"testing"

	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/commons"
	plugins_types "github.com/rapidaai/pkg/plugins/types"
	rapida_types "github.com/rapidaai/pkg/types"
	vault_api "github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

type mockAuth struct{}

func (m *mockAuth) GetUserId() *uint64                { return nil }
func (m *mockAuth) GetCurrentOrganizationId() *uint64 { v := uint64(1); return &v }
func (m *mockAuth) GetCurrentProjectId() *uint64      { v := uint64(1); return &v }
func (m *mockAuth) HasUser() bool                     { return true }
func (m *mockAuth) HasOrganization() bool             { return true }
func (m *mockAuth) HasProject() bool                  { return true }
func (m *mockAuth) IsAuthenticated() bool             { return true }
func (m *mockAuth) GetCurrentToken() string           { return "t" }
func (m *mockAuth) Type() string                      { return "test" }

type mockVault struct {
	credential *vault_api.VaultCredential
	err        error
}

func (m *mockVault) GetCredential(ctx context.Context, auth rapida_types.SimplePrinciple, vaultId uint64) (*vault_api.VaultCredential, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.credential, nil
}

func (m *mockVault) GetOauth2Credential(ctx context.Context, auth rapida_types.SimplePrinciple, vaultId uint64) (*vault_api.VaultCredential, error) {
	return nil, errors.New("not used")
}

var _ web_client.VaultClient = (*mockVault)(nil)

func testLogger(t *testing.T) commons.Logger {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	require.NoError(t, err)
	return logger
}

func TestPlugin_ExecuteCalComSuccess(t *testing.T) {
	p := NewPlugin()
	st, _ := structpb.NewStruct(map[string]interface{}{"api_key": "cal-secret"})
	vault := &mockVault{credential: &vault_api.VaultCredential{Id: 10, Value: st}}

	result, err := p.Execute(context.Background(), plugins_types.ExecuteRequest{
		Operation: "book_meeting",
		Provider:  "cal.com",
		Input: map[string]interface{}{
			"start_time": "2026-05-01T10:00:00Z",
			"end_time":   "2026-05-01T10:30:00Z",
			"email":      "user@example.com",
		},
		Config: map[string]interface{}{"provider": "cal.com", "credential_id": float64(10)},
	}, plugins_types.ExecuteDeps{VaultClient: vault, Logger: testLogger(t), Auth: &mockAuth{}})
	require.NoError(t, err)
	assert.Equal(t, plugins_types.StatusSuccess, result.Status)
	assert.Equal(t, "cal.com", result.Provider)
}

func TestPlugin_ExecuteCalComMissingInput(t *testing.T) {
	p := NewPlugin()
	st, _ := structpb.NewStruct(map[string]interface{}{"api_key": "cal-secret"})
	vault := &mockVault{credential: &vault_api.VaultCredential{Id: 10, Value: st}}

	result, err := p.Execute(context.Background(), plugins_types.ExecuteRequest{
		Operation: "book_meeting",
		Provider:  "cal.com",
		Input:     map[string]interface{}{"start_time": "2026-05-01T10:00:00Z"},
		Config:    map[string]interface{}{"provider": "cal.com", "credential_id": float64(10)},
	}, plugins_types.ExecuteDeps{VaultClient: vault, Logger: testLogger(t), Auth: &mockAuth{}})
	require.NoError(t, err)
	assert.Equal(t, plugins_types.StatusFail, result.Status)
}
