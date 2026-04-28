package plugins_provider_sms_twilio

import (
	"context"
	"fmt"
	"strings"

	"github.com/rapidaai/pkg/commons"
)

type Provider struct{}

func New() *Provider { return &Provider{} }

func (t *Provider) Code() string { return "twilio" }

func (t *Provider) Send(_ context.Context, input map[string]interface{}, credential map[string]interface{}, logger commons.Logger) (map[string]interface{}, error) {
	logger.Infof("twilio provider executed")
	accountSID := strings.TrimSpace(fmt.Sprintf("%v", credential["account_sid"]))
	authToken := strings.TrimSpace(fmt.Sprintf("%v", credential["auth_token"]))
	from := strings.TrimSpace(fmt.Sprintf("%v", credential["from_number"]))
	if accountSID == "" || authToken == "" || from == "" ||
		accountSID == "<nil>" || authToken == "<nil>" || from == "<nil>" {
		return nil, fmt.Errorf("twilio credential missing account_sid/auth_token/from_number")
	}
	to := strings.TrimSpace(fmt.Sprintf("%v", input["to"]))
	body := strings.TrimSpace(fmt.Sprintf("%v", input["body"]))
	if to == "" || body == "" || to == "<nil>" || body == "<nil>" {
		return nil, fmt.Errorf("send_sms requires to and body")
	}

	return map[string]interface{}{
		"accepted": true,
		"provider_request": map[string]interface{}{
			"provider": "twilio",
			"to":       to,
			"from":     from,
			"body":     body,
		},
	}, nil
}
