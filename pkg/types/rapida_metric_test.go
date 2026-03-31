// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package types

import (
	"testing"
	"time"

	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

func TestMetric_GetName(t *testing.T) {
	tests := []struct {
		name string
		m    *Metric
		want string
	}{
		{
			name: "nil metric",
			m:    nil,
			want: "",
		},
		{
			name: "valid metric",
			m:    &Metric{Name: "test"},
			want: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.GetName(); got != tt.want {
				t.Errorf("GetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetric_GetValue(t *testing.T) {
	tests := []struct {
		name string
		m    *Metric
		want string
	}{
		{
			name: "nil",
			m:    nil,
			want: "",
		},
		{
			name: "valid",
			m:    &Metric{Value: "val"},
			want: "val",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.GetValue(); got != tt.want {
				t.Errorf("GetValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetric_GetDescription(t *testing.T) {
	tests := []struct {
		name string
		m    *Metric
		want string
	}{
		{
			name: "nil",
			m:    nil,
			want: "",
		},
		{
			name: "valid",
			m:    &Metric{Description: "desc"},
			want: "desc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.GetDescription(); got != tt.want {
				t.Errorf("GetDescription() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetric_ToProto(t *testing.T) {
	m := &Metric{Name: "name", Value: "val", Description: "desc"}
	proto := m.ToProto()
	if proto.Name != "name" {
		t.Errorf("ToProto() Name = %v, want %v", proto.Name, "name")
	}
}

func TestMetrics_ToProto(t *testing.T) {
	metrics := Metrics{
		&Metric{Name: "1"},
		&Metric{Name: "2"},
	}
	protos := metrics.ToProto()
	if len(protos) != 2 {
		t.Errorf("ToProto() length = %v, want %v", len(protos), 2)
	}
}

func TestToMetric(t *testing.T) {
	proto := &protos.Metric{Name: "name", Value: "val"}
	m := ToMetric(proto)
	if m == nil {
		t.Errorf("ToMetric() returned nil")
		return
	}
	if m.Name != "name" {
		t.Errorf("ToMetric() Name = %v, want %v", m.Name, "name")
	}
}

func TestToMetrics(t *testing.T) {
	protos := []*protos.Metric{
		{Name: "1"},
		{Name: "2"},
	}
	metrics := ToMetrics(protos)
	if len(metrics) != 2 {
		t.Errorf("ToMetrics() length = %v, want %v", len(metrics), 2)
	}
}

func TestNewMetric(t *testing.T) {
	desc := "description"
	m := NewMetric("name", "val", &desc)
	if m.Name != "name" {
		t.Errorf("NewMetric() Name = %v, want %v", m.Name, "name")
	}
	if m.Value != "val" {
		t.Errorf("NewMetric() Value = %v, want %v", m.Value, "val")
	}
	if m.Description != "description" {
		t.Errorf("NewMetric() Description = %v, want %v", m.Description, "description")
	}
}

func TestNewTimeTakenMetric(t *testing.T) {
	dur := 5 * time.Second
	m := NewTimeTakenMetric(dur)
	if m.Name != type_enums.TIME_TAKEN.String() {
		t.Errorf("NewTimeTakenMetric() Name = %v, want %v", m.Name, type_enums.TIME_TAKEN.String())
	}
	if m.Value != "5000000000" {
		t.Errorf("NewTimeTakenMetric() Value = %v, want %v", m.Value, "5000000000")
	}
}

func TestNewInputTokenMetric(t *testing.T) {
	m := NewInputTokenMetric(100)
	if m.Name != type_enums.INPUT_TOKEN.String() {
		t.Errorf("NewInputTokenMetric() Name = %v", m.Name)
	}
	if m.Value != "100" {
		t.Errorf("NewInputTokenMetric() Value = %v", m.Value)
	}
}

func TestNewOutputTokenMetric(t *testing.T) {
	m := NewOutputTokenMetric(200)
	if m.Name != type_enums.OUTPUT_TOKEN.String() {
		t.Errorf("NewOutputTokenMetric() Name = %v", m.Name)
	}
	if m.Value != "200" {
		t.Errorf("NewOutputTokenMetric() Value = %v", m.Value)
	}
}

func TestNewTotalTokenMetric(t *testing.T) {
	m := NewTotalTokenMetric(300)
	if m.Name != type_enums.TOTAL_TOKEN.String() {
		t.Errorf("NewTotalTokenMetric() Name = %v", m.Name)
	}
	if m.Value != "300" {
		t.Errorf("NewTotalTokenMetric() Value = %v", m.Value)
	}
}

func TestNewInputCostMetric(t *testing.T) {
	m := NewInputCostMetric(1.234567)
	if m.Name != type_enums.INPUT_COST.String() {
		t.Errorf("NewInputCostMetric() Name = %v", m.Name)
	}
	if m.Value != "1.234567" {
		t.Errorf("NewInputCostMetric() Value = %v", m.Value)
	}
}

func TestNewOutputCostMetric(t *testing.T) {
	m := NewOutputCostMetric(2.345678)
	if m.Name != type_enums.OUTPUT_COST.String() {
		t.Errorf("NewOutputCostMetric() Name = %v", m.Name)
	}
	if m.Value != "2.345678" {
		t.Errorf("NewOutputCostMetric() Value = %v", m.Value)
	}
}

func TestNewTotalCostMetric(t *testing.T) {
	m := NewTotalCostMetric(3.456789)
	if m.Name != type_enums.COST.String() {
		t.Errorf("NewTotalCostMetric() Name = %v", m.Name)
	}
	if m.Value != "3.456789" {
		t.Errorf("NewTotalCostMetric() Value = %v", m.Value)
	}
}

func TestNewStatusMetric(t *testing.T) {
	status := type_enums.RECORD_ACTIVE
	m := NewStatusMetric(status)
	if m.Name != type_enums.STATUS.String() {
		t.Errorf("NewStatusMetric() Name = %v", m.Name)
	}
	if m.Value != status.String() {
		t.Errorf("NewStatusMetric() Value = %v", m.Value)
	}
}

func TestMetricName_String(t *testing.T) {
	if type_enums.TIME_TAKEN.String() != "time_taken" {
		t.Errorf("TIME_TAKEN.String() = %v, want %v", type_enums.TIME_TAKEN.String(), "time_taken")
	}
}
