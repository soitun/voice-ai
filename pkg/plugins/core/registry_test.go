package plugins_core

import (
	"context"
	"testing"

	plugins_types "github.com/rapidaai/pkg/plugins/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubPlugin struct{ code string }

func (s *stubPlugin) Code() string                                 { return s.code }
func (s *stubPlugin) Validate(config map[string]interface{}) error { return nil }
func (s *stubPlugin) Execute(ctx context.Context, req plugins_types.ExecuteRequest, deps plugins_types.ExecuteDeps) (*plugins_types.Result, error) {
	return plugins_types.Success("stub", req.Operation, map[string]interface{}{"ok": true}), nil
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()
	err := r.Register(&stubPlugin{code: "email"})
	require.NoError(t, err)

	plugin, ok := r.Get("email")
	require.True(t, ok)
	assert.Equal(t, "email", plugin.Code())
}

func TestRegistry_RejectDuplicate(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Register(&stubPlugin{code: "email"}))
	err := r.Register(&stubPlugin{code: "email"})
	require.Error(t, err)
}

func TestRegistry_MustGetUnknown(t *testing.T) {
	r := NewRegistry()
	_, err := r.MustGet("missing")
	require.Error(t, err)
}
