package plugins_provider_sms_noop

import (
	"context"

	"github.com/rapidaai/pkg/commons"
)

type Provider struct{}

func New() *Provider { return &Provider{} }

func (n *Provider) Code() string { return "noop" }

func (n *Provider) Send(_ context.Context, input map[string]interface{}, credential map[string]interface{}, logger commons.Logger) (map[string]interface{}, error) {
	logger.Infof("noop sms provider executed")
	return map[string]interface{}{
		"input":      input,
		"credential": credential,
		"message":    "noop sms provider",
	}, nil
}
