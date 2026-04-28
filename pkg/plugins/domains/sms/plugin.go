package plugins_sms

import (
	"context"
	"fmt"
	"strings"

	"github.com/rapidaai/pkg/commons"
	plugins_provider_sms_noop "github.com/rapidaai/pkg/plugins/providers/sms/noop"
	plugins_provider_sms_twilio "github.com/rapidaai/pkg/plugins/providers/sms/twilio"
	plugins_types "github.com/rapidaai/pkg/plugins/types"
	"github.com/rapidaai/pkg/utils"
)

const Code = "sms"

// Provider is the extension contract for SMS providers.
// Implement this in any package and register it via RegisterProvider
// or domains.WithSMSProviders(...).
type Provider interface {
	Code() string
	Send(ctx context.Context, input map[string]interface{}, credential map[string]interface{}, logger commons.Logger) (map[string]interface{}, error)
}

type Plugin struct {
	providers map[string]Provider
}

func NewPlugin(customProviders ...Provider) *Plugin {
	p := &Plugin{providers: make(map[string]Provider)}
	p.RegisterProvider(plugins_provider_sms_noop.New())
	p.RegisterProvider(plugins_provider_sms_twilio.New())
	for _, provider := range customProviders {
		p.RegisterProvider(provider)
	}
	return p
}

func (p *Plugin) Code() string { return Code }

func (p *Plugin) RegisterProvider(provider Provider) {
	if provider == nil {
		return
	}
	p.providers[strings.ToLower(strings.TrimSpace(provider.Code()))] = provider
}

func (p *Plugin) Validate(config map[string]interface{}) error {
	opts := utils.Option(config)
	provider, err := opts.GetString("provider")
	if err != nil || strings.TrimSpace(provider) == "" {
		return fmt.Errorf("provider is required")
	}
	if _, ok := p.providers[strings.ToLower(strings.TrimSpace(provider))]; !ok {
		return fmt.Errorf("unsupported sms provider %q", provider)
	}
	if _, err := opts.GetUint64("credential_id"); err != nil {
		return fmt.Errorf("credential_id is required and must be uint64: %w", err)
	}
	return nil
}

func (p *Plugin) Execute(ctx context.Context, req plugins_types.ExecuteRequest, deps plugins_types.ExecuteDeps) (*plugins_types.Result, error) {
	if err := p.Validate(req.Config); err != nil {
		return plugins_types.Failure(req.Provider, req.Operation, err.Error(), nil), nil
	}

	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	if provider == "" {
		provider, _ = utils.Option(req.Config).GetString("provider")
		provider = strings.ToLower(strings.TrimSpace(provider))
	}

	operation := strings.TrimSpace(req.Operation)
	if operation == "" {
		operation = "send_sms"
	}
	if operation != "send_sms" {
		return plugins_types.Failure(provider, operation, "unsupported sms operation", nil), nil
	}

	credentialID, _ := utils.Option(req.Config).GetUint64("credential_id")
	vaultCredential, err := deps.VaultClient.GetCredential(ctx, deps.Auth, credentialID)
	if err != nil {
		return plugins_types.Failure(provider, operation, fmt.Sprintf("vault credential lookup failed: %v", err), nil), nil
	}

	client, ok := p.providers[provider]
	if !ok {
		return plugins_types.Failure(provider, operation, "sms provider not found", nil), nil
	}

	result, err := client.Send(ctx, req.Input, vaultCredential.GetValue().AsMap(), deps.Logger)
	if err != nil {
		return plugins_types.Failure(provider, operation, err.Error(), result), nil
	}
	return plugins_types.Success(provider, operation, result), nil
}
