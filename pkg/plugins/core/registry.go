package plugins_core

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	plugins_types "github.com/rapidaai/pkg/plugins/types"
)

type Plugin interface {
	Code() string
	Validate(config map[string]interface{}) error
	Execute(ctx context.Context, req plugins_types.ExecuteRequest, deps plugins_types.ExecuteDeps) (*plugins_types.Result, error)
}

type Registry struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
}

func NewRegistry() *Registry {
	return &Registry{plugins: make(map[string]Plugin)}
}

func normalize(code string) string {
	return strings.TrimSpace(strings.ToLower(code))
}

func (r *Registry) Register(plugin Plugin) error {
	if plugin == nil {
		return fmt.Errorf("plugin is nil")
	}
	code := normalize(plugin.Code())
	if code == "" {
		return fmt.Errorf("plugin code is empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.plugins[code]; exists {
		return fmt.Errorf("plugin %q already registered", code)
	}
	r.plugins[code] = plugin
	return nil
}

func (r *Registry) Get(code string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	plugin, ok := r.plugins[normalize(code)]
	return plugin, ok
}

func (r *Registry) MustGet(code string) (Plugin, error) {
	plugin, ok := r.Get(code)
	if !ok {
		return nil, fmt.Errorf("plugin %q not found", code)
	}
	return plugin, nil
}

func (r *Registry) Codes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.plugins))
	for code := range r.plugins {
		out = append(out, code)
	}
	sort.Strings(out)
	return out
}
