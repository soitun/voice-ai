package plugins_provider_email_sendgrid

import (
	"context"
	"fmt"
	"strings"

	"github.com/rapidaai/pkg/commons"
)

type Provider struct{}

func New() *Provider { return &Provider{} }

func (s *Provider) Code() string { return "sendgrid" }

func (s *Provider) Send(_ context.Context, input map[string]interface{}, credential map[string]interface{}, logger commons.Logger) (map[string]interface{}, error) {
	logger.Infof("sendgrid provider executed")
	apiKey := strings.TrimSpace(fmt.Sprintf("%v", credential["api_key"]))
	if apiKey == "" || apiKey == "<nil>" {
		return nil, fmt.Errorf("sendgrid credential missing api_key")
	}
	to := strings.TrimSpace(fmt.Sprintf("%v", input["to"]))
	subject := strings.TrimSpace(fmt.Sprintf("%v", input["subject"]))
	text := strings.TrimSpace(fmt.Sprintf("%v", input["text"]))
	if to == "" || subject == "" || text == "" || to == "<nil>" || subject == "<nil>" || text == "<nil>" {
		return nil, fmt.Errorf("send_email requires to, subject, and text")
	}

	return map[string]interface{}{
		"accepted": true,
		"provider_request": map[string]interface{}{
			"provider": "sendgrid",
			"to":       to,
			"subject":  subject,
			"text":     text,
			"html":     input["html"],
		},
	}, nil
}
