package plugins_provider_calendar_calcom

import (
	"context"
	"fmt"
	"strings"

	"github.com/rapidaai/pkg/commons"
)

type Provider struct{}

func New() *Provider { return &Provider{} }

func (c *Provider) Code() string { return "cal.com" }

func (c *Provider) BookMeeting(_ context.Context, input map[string]interface{}, credential map[string]interface{}, logger commons.Logger) (map[string]interface{}, error) {
	logger.Infof("cal.com provider executed")
	apiKey := strings.TrimSpace(fmt.Sprintf("%v", credential["api_key"]))
	if apiKey == "" || apiKey == "<nil>" {
		return nil, fmt.Errorf("cal.com credential missing api_key")
	}
	start := strings.TrimSpace(fmt.Sprintf("%v", input["start_time"]))
	end := strings.TrimSpace(fmt.Sprintf("%v", input["end_time"]))
	email := strings.TrimSpace(fmt.Sprintf("%v", input["email"]))
	if start == "" || end == "" || email == "" || start == "<nil>" || end == "<nil>" || email == "<nil>" {
		return nil, fmt.Errorf("book_meeting requires start_time, end_time, and email")
	}

	return map[string]interface{}{
		"accepted": true,
		"provider_request": map[string]interface{}{
			"provider":      "cal.com",
			"start_time":    start,
			"end_time":      end,
			"email":         email,
			"name":          input["name"],
			"event_type_id": input["event_type_id"],
		},
	}, nil
}
