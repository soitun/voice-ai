<!--
Copyright (c) 2023-2025 RapidaAI
Author: Prashant Srivastav <prashant@rapida.ai>

Licensed under GPL-2.0 with Rapida Additional Terms.
See LICENSE.md or contact sales@rapida.ai for commercial usage.
-->

# Plugins Package

This package provides an interface-driven plugin system for domain actions.

Current domains:
- `email`
- `sms`
- `calendar`

Current built-in providers:
- Email: `sendgrid`
- SMS: `twilio`
- Calendar: `cal.com`

## Structure

- `core/`:
  - plugin registry
- `types/`:
  - shared execution contracts
- `domains/`:
  - domain plugins and domain provider interfaces
- `providers/`:
  - built-in providers (one provider per package)

## Core Concepts

1. `core.Plugin` interface
- `Code() string`
- `Validate(config map[string]interface{}) error`
- `Execute(ctx, req, deps)`

2. Domain provider interfaces
- Email providers implement `domains/email.Provider`
- SMS providers implement `domains/sms.Provider`
- Calendar providers implement `domains/calendar.Provider`

3. Runtime dependencies
- Every plugin execution gets:
  - `VaultClient` (for credential resolution)
  - `Logger`
  - `Auth`

## Quick Start

```go
import (
    plugins_core "github.com/rapidaai/pkg/plugins/core"
    plugins_domains "github.com/rapidaai/pkg/plugins/domains"
)

registry := plugins_core.NewRegistry()
if err := plugins_domains.RegisterDefaults(registry); err != nil {
    panic(err)
}

emailPlugin, ok := registry.Get("email")
if !ok {
    panic("email plugin not found")
}
_ = emailPlugin
```

## Config Convention

Domain plugins currently validate:
- `provider` (string)
- `credential_id` (uint64-compatible)

Example config:

```go
cfg := map[string]interface{}{
    "provider":      "sendgrid", // twilio, cal.com, etc.
    "credential_id": uint64(1234),
}
```

## Add a New Provider (Different Package)

You can add providers in your own package without changing core plugin code.

### 1) Implement domain provider interface

Example: custom email provider

```go
package myemailprovider

import (
    "context"

    plugins_email "github.com/rapidaai/pkg/plugins/domains/email"
    "github.com/rapidaai/pkg/commons"
)

type Provider struct{}

func New() *Provider { return &Provider{} }

func (p *Provider) Code() string { return "my_email" }

func (p *Provider) Send(
    ctx context.Context,
    input map[string]interface{},
    credential map[string]interface{},
    logger commons.Logger,
) (map[string]interface{}, error) {
    // your implementation
    return map[string]interface{}{"accepted": true}, nil
}

var _ plugins_email.Provider = (*Provider)(nil)
```

### 2) Register it with defaults

```go
import (
    plugins_core "github.com/rapidaai/pkg/plugins/core"
    plugins_domains "github.com/rapidaai/pkg/plugins/domains"

    myemail "your/module/myemailprovider"
)

registry := plugins_core.NewRegistry()
err := plugins_domains.RegisterDefaults(
    registry,
    plugins_domains.WithEmailProviders(myemail.New()),
)
if err != nil {
    panic(err)
}
```

The same pattern applies for SMS and Calendar using:
- `WithSMSProviders(...)`
- `WithCalendarProviders(...)`

## Execute a Plugin

```go
import (
    "context"

    plugins_types "github.com/rapidaai/pkg/plugins/types"
)

result, err := emailPlugin.Execute(ctx, plugins_types.ExecuteRequest{
    Operation: "send_email",
    Provider:  "sendgrid",
    Input: map[string]interface{}{
        "to": "user@example.com",
        "subject": "Hello",
        "text": "Welcome",
    },
    Config: map[string]interface{}{
        "provider": "sendgrid",
        "credential_id": uint64(1234),
    },
}, plugins_types.ExecuteDeps{
    VaultClient: vaultClient,
    Logger:      logger,
    Auth:        auth,
})
if err != nil {
    panic(err)
}
_ = result
```

## Notes

- Domain plugins enforce operation names:
  - Email: `send_email`
  - SMS: `send_sms`
  - Calendar: `book_meeting`
- Providers are intentionally isolated by package to avoid merge conflicts and allow independent extension.
