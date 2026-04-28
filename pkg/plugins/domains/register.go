package plugins_domains

import (
	plugins_core "github.com/rapidaai/pkg/plugins/core"
	plugins_calendar "github.com/rapidaai/pkg/plugins/domains/calendar"
	plugins_email "github.com/rapidaai/pkg/plugins/domains/email"
	plugins_sms "github.com/rapidaai/pkg/plugins/domains/sms"
)

type RegisterOption func(*registerConfig)

type registerConfig struct {
	emailProviders    []plugins_email.Provider
	smsProviders      []plugins_sms.Provider
	calendarProviders []plugins_calendar.Provider
}

func WithEmailProviders(providers ...plugins_email.Provider) RegisterOption {
	return func(cfg *registerConfig) {
		cfg.emailProviders = append(cfg.emailProviders, providers...)
	}
}

func WithSMSProviders(providers ...plugins_sms.Provider) RegisterOption {
	return func(cfg *registerConfig) {
		cfg.smsProviders = append(cfg.smsProviders, providers...)
	}
}

func WithCalendarProviders(providers ...plugins_calendar.Provider) RegisterOption {
	return func(cfg *registerConfig) {
		cfg.calendarProviders = append(cfg.calendarProviders, providers...)
	}
}

func RegisterDefaults(registry *plugins_core.Registry, options ...RegisterOption) error {
	cfg := &registerConfig{}
	for _, option := range options {
		option(cfg)
	}

	if err := registry.Register(plugins_email.NewPlugin(cfg.emailProviders...)); err != nil {
		return err
	}
	if err := registry.Register(plugins_sms.NewPlugin(cfg.smsProviders...)); err != nil {
		return err
	}
	if err := registry.Register(plugins_calendar.NewPlugin(cfg.calendarProviders...)); err != nil {
		return err
	}
	return nil
}
