// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rapidaai/pkg/configs"
	"github.com/rapidaai/pkg/exceptions"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

// GetAllAssistantTelemetry queries the observe event and metric OpenSearch indices
// for the requested assistant and returns typed TelemetryRecord entries.
func (assistantApi *assistantGrpcApi) GetAllAssistantTelemetry(
	ctx context.Context,
	request *protos.GetAllAssistantTelemetryRequest,
) (*protos.GetAllAssistantTelemetryResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		assistantApi.logger.Errorf("unauthenticated request for GetAllAssistantTelemetry")
		return exceptions.AuthenticationError[protos.GetAllAssistantTelemetryResponse]()
	}

	if assistantApi.opensearch == nil {
		return &protos.GetAllAssistantTelemetryResponse{Code: 200, Success: true}, nil
	}

	assistantId := request.GetAssistant().GetAssistantId()

	// Gate: only query OpenSearch when telemetry is configured to persist there.
	// Check env config first (global), then DB providers (per-assistant).
	opensearchEnabled := assistantApi.cfg.TelemetryConfig != nil &&
		assistantApi.cfg.TelemetryConfig.Type() == configs.OPENSEARCH
	if !opensearchEnabled && iAuth.HasProject() {
		cnt, _, _ := assistantApi.assistantTelemetryService.GetAll(
			ctx, iAuth, assistantId,
			[]*protos.Criteria{
				{Key: "provider_type", Logic: "=", Value: "opensearch"},
				{Key: "enabled", Logic: "=", Value: "true"},
			},
			&protos.Paginate{Page: 1, PageSize: 1},
		)
		opensearchEnabled = cnt > 0
	}
	if !opensearchEnabled {
		return &protos.GetAllAssistantTelemetryResponse{Code: 200, Success: true}, nil
	}

	page := int(request.GetPaginate().GetPage())
	if page < 1 {
		page = 1
	}
	size := int(request.GetPaginate().GetPageSize())
	if size < 1 || size > 100 {
		size = 20
	}
	from := (page - 1) * size

	criterias := make(map[string]string)
	for _, c := range request.GetCriterias() {
		criterias[c.GetKey()] = c.GetValue()
	}

	evtIdx, metIdx := telemetryIndices(assistantApi.cfg.IsDevelopment())
	evtQuery := buildTelemetryQuery(assistantId, criterias, from, size, "name")
	metQuery := buildTelemetryQuery(assistantId, criterias, from, size, "scope")

	evtHits := assistantApi.opensearch.Search(ctx, []string{evtIdx}, evtQuery)
	metHits := assistantApi.opensearch.Search(ctx, []string{metIdx}, metQuery)

	var records []*protos.TelemetryRecord
	for _, hit := range evtHits.Hits.Hits {
		if rec := eventHitToRecord(hit); rec != nil {
			records = append(records, rec)
		}
	}
	for _, hit := range metHits.Hits.Hits {
		if rec := metricHitToRecord(hit); rec != nil {
			records = append(records, rec)
		}
	}

	total := evtHits.Hits.Total.Value + metHits.Hits.Total.Value
	return &protos.GetAllAssistantTelemetryResponse{
		Code:    200,
		Success: true,
		Data:    records,
		Paginated: &protos.Paginated{
			TotalItem:   uint32(total),
			CurrentPage: uint32(page),
		},
	}, nil
}

func telemetryIndices(_ bool) (evtIdx, metIdx string) {
	return "rapida-events-*", "rapida-metrics-*"
}

// buildTelemetryQuery builds an OpenSearch query body for the telemetry indices.
// indexSpecificKey is "name" for the events index and "scope" for the metrics index.
func buildTelemetryQuery(assistantId uint64, criterias map[string]string, from, size int, indexSpecificKey string) string {
	must := []interface{}{
		map[string]interface{}{
			"term": map[string]interface{}{"assistantId": assistantId},
		},
	}

	if v, ok := criterias["conversationId"]; ok && v != "" {
		if convId, err := strconv.ParseUint(v, 10, 64); err == nil {
			must = append(must, map[string]interface{}{
				"term": map[string]interface{}{"assistantConversationId": convId},
			})
		}
	}

	if v, ok := criterias["messageId"]; ok && v != "" {
		// events index uses "messageId"; metrics index uses "contextId"
		fieldName := "messageId"
		if indexSpecificKey == "scope" {
			fieldName = "contextId"
		}
		must = append(must, map[string]interface{}{
			"term": map[string]interface{}{fieldName: v},
		})
	}

	if v, ok := criterias[indexSpecificKey]; ok && v != "" {
		must = append(must, map[string]interface{}{
			"term": map[string]interface{}{indexSpecificKey: v},
		})
	}

	q := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": must,
			},
		},
		"sort": []interface{}{
			map[string]interface{}{"time": map[string]interface{}{"order": "desc"}},
		},
		"from": from,
		"size": size,
	}
	b, _ := json.Marshal(q)
	return string(b)
}

// eventHitToRecord maps a rapida-events-* OpenSearch document to a TelemetryRecord.
func eventHitToRecord(hit map[string]interface{}) *protos.TelemetryRecord {
	src, _ := hit["_source"].(map[string]interface{})
	if src == nil {
		return nil
	}

	data := map[string]string{}
	if raw, ok := src["data"].(map[string]interface{}); ok {
		for k, v := range raw {
			data[k] = fmt.Sprintf("%v", v)
		}
	}

	return &protos.TelemetryRecord{
		Record: &protos.TelemetryRecord_Event{
			Event: &protos.TelemetryEvent{
				MessageId:               telemetryStrVal(src["messageId"]),
				AssistantId:             telemetryUint64(src["assistantId"]),
				AssistantConversationId: telemetryUint64(src["assistantConversationId"]),
				ProjectId:               telemetryUint64(src["projectId"]),
				OrganizationId:          telemetryUint64(src["organizationId"]),
				Name:                    telemetryStrVal(src["name"]),
				Data:                    data,
				Time:                    telemetryTimestamp(src["time"]),
			},
		},
	}
}

// metricHitToRecord maps a rapida-metrics-* OpenSearch document to a TelemetryRecord.
func metricHitToRecord(hit map[string]interface{}) *protos.TelemetryRecord {
	src, _ := hit["_source"].(map[string]interface{})
	if src == nil {
		return nil
	}

	var metrics []*protos.Metric
	if raw, ok := src["metrics"].([]interface{}); ok {
		for _, m := range raw {
			if entry, ok := m.(map[string]interface{}); ok {
				metrics = append(metrics, &protos.Metric{
					Name:  telemetryStrVal(entry["name"]),
					Value: telemetryStrVal(entry["value"]),
				})
			}
		}
	}

	return &protos.TelemetryRecord{
		Record: &protos.TelemetryRecord_Metric{
			Metric: &protos.TelemetryMetric{
				ContextId:               telemetryStrVal(src["contextId"]),
				AssistantId:             telemetryUint64(src["assistantId"]),
				AssistantConversationId: telemetryUint64(src["assistantConversationId"]),
				ProjectId:               telemetryUint64(src["projectId"]),
				OrganizationId:          telemetryUint64(src["organizationId"]),
				Scope:                   telemetryStrVal(src["scope"]),
				Metrics:                 metrics,
				Time:                    telemetryTimestamp(src["time"]),
			},
		},
	}
}

// telemetryStrVal converts a JSON-decoded interface value to string.
// float64 values (JSON numbers) are formatted as integers to preserve large IDs.
func telemetryStrVal(v interface{}) string {
	if v == nil {
		return ""
	}
	if f, ok := v.(float64); ok {
		return strconv.FormatUint(uint64(f), 10)
	}
	return fmt.Sprintf("%v", v)
}

// telemetryUint64 converts a JSON-decoded float64 number to uint64.
func telemetryUint64(v interface{}) uint64 {
	if f, ok := v.(float64); ok {
		return uint64(f)
	}
	return 0
}

// telemetryTimestamp parses a time string from a JSON-decoded OpenSearch document.
func telemetryTimestamp(v interface{}) *timestamppb.Timestamp {
	if s, ok := v.(string); ok {
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return timestamppb.New(t)
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return timestamppb.New(t)
		}
	}
	return timestamppb.Now()
}
