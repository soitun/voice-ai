package plugins_domains

import (
	"context"
	"testing"

	"github.com/rapidaai/pkg/commons"
	plugins_core "github.com/rapidaai/pkg/plugins/core"
	plugins_calendar "github.com/rapidaai/pkg/plugins/domains/calendar"
	plugins_email "github.com/rapidaai/pkg/plugins/domains/email"
	plugins_sms "github.com/rapidaai/pkg/plugins/domains/sms"
	"github.com/stretchr/testify/require"
)

type customEmailProvider struct{}

func (c *customEmailProvider) Code() string { return "custom_email" }
func (c *customEmailProvider) Send(ctx context.Context, input map[string]interface{}, credential map[string]interface{}, logger commons.Logger) (map[string]interface{}, error) {
	return map[string]interface{}{"ok": true}, nil
}

type customSMSProvider struct{}

func (c *customSMSProvider) Code() string { return "custom_sms" }
func (c *customSMSProvider) Send(ctx context.Context, input map[string]interface{}, credential map[string]interface{}, logger commons.Logger) (map[string]interface{}, error) {
	return map[string]interface{}{"ok": true}, nil
}

type customCalendarProvider struct{}

func (c *customCalendarProvider) Code() string { return "custom_calendar" }
func (c *customCalendarProvider) BookMeeting(ctx context.Context, input map[string]interface{}, credential map[string]interface{}, logger commons.Logger) (map[string]interface{}, error) {
	return map[string]interface{}{"ok": true}, nil
}

var _ plugins_email.Provider = (*customEmailProvider)(nil)
var _ plugins_sms.Provider = (*customSMSProvider)(nil)
var _ plugins_calendar.Provider = (*customCalendarProvider)(nil)

func TestRegisterDefaults_WithCustomProviders(t *testing.T) {
	r := plugins_core.NewRegistry()
	err := RegisterDefaults(r,
		WithEmailProviders(&customEmailProvider{}),
		WithSMSProviders(&customSMSProvider{}),
		WithCalendarProviders(&customCalendarProvider{}),
	)
	require.NoError(t, err)

	emailPluginI, ok := r.Get("email")
	require.True(t, ok)
	emailPlugin := emailPluginI.(*plugins_email.Plugin)
	require.NoError(t, emailPlugin.Validate(map[string]interface{}{"provider": "custom_email", "credential_id": float64(1)}))

	smsPluginI, ok := r.Get("sms")
	require.True(t, ok)
	smsPlugin := smsPluginI.(*plugins_sms.Plugin)
	require.NoError(t, smsPlugin.Validate(map[string]interface{}{"provider": "custom_sms", "credential_id": float64(1)}))

	calendarPluginI, ok := r.Get("calendar")
	require.True(t, ok)
	calendarPlugin := calendarPluginI.(*plugins_calendar.Plugin)
	require.NoError(t, calendarPlugin.Validate(map[string]interface{}{"provider": "custom_calendar", "credential_id": float64(1)}))
}
