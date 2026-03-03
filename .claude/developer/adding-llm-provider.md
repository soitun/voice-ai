# Adding a New LLM Provider (Frontend to Backend)

This guide covers the complete end-to-end steps for adding a new Large Language Model (LLM) provider to Rapida — from the React UI through to the Go backend caller and gRPC service registration.

---

## Overview of Touchpoints

| Layer | Files | Purpose |
|-------|-------|---------|
| **Provider Registry** | `ui/src/providers/provider.production.json` + `.development.json` | Registers the provider with feature flags (`text`, `external`) |
| **Model Constants** | `ui/src/app/components/providers/text/<provider>/constants.ts` | Model list, default options, validation |
| **UI Config Component** | `ui/src/app/components/providers/text/<provider>/index.tsx` | Model dropdown + advanced parameter form |
| **UI Provider Router** | `ui/src/app/components/providers/text/index.tsx` | Switch-case routing to config component + defaults + validation |
| **Backend Caller** | `api/integration-api/internal/caller/<provider>/` | Go: base struct, LLM streaming, credential verifier |
| **API Handler** | `api/integration-api/api/<provider>.go` | gRPC service implementation (Chat, StreamChat, VerifyCredential) |
| **Service Router** | `api/integration-api/router/provider.go` | Registers the gRPC server |
| **Integration Client** | `pkg/clients/integration/integration_client.go` | Routes `providerName` to the correct gRPC client |
| **Proto** | `protos/artifacts/` (submodule) | Service definition — only needed for entirely new proto services |

> **Note on protos**: Existing providers (OpenAI, Anthropic, Gemini, etc.) each have their own proto service. If you are adding a provider that is API-compatible with an existing one (e.g., a hosted model behind an OpenAI-compatible endpoint), you may be able to reuse the existing proto and add only a new `case` in the integration client switch. If the provider has a genuinely distinct API surface, you will need a new proto service definition.

---

## Step 1: Register the Provider (Frontend)

### 1a. Add to Provider Registry JSON

Edit both `ui/src/providers/provider.development.json` and `ui/src/providers/provider.production.json`. Add an entry:

```json
{
    "code": "myprovider",
    "name": "My Provider",
    "description": "Description of what this provider offers.",
    "image": "https://cdn-01.rapida.ai/partners/myprovider.png",
    "featureList": ["text", "external"],
    "configurations": [
        {
            "name": "key",
            "type": "string",
            "label": "API Key"
        }
    ],
    "humanname": "My Provider",
    "website": "https://myprovider.com"
}
```

Key fields:
- **`code`**: Unique lowercase identifier. Must match the string used in the integration client switch and in the backend caller constant (case-insensitive — the client does `strings.ToLower`).
- **`featureList`**:
  - `"text"` — shows in the LLM/chat provider dropdown
  - `"embedding"` — shows in embedding provider dropdown
  - `"external"` — shows in Integrations / Vault credential management
- **`configurations`**: Credential fields shown in the vault form. The `name` field becomes the key in the credential value map. Most providers use a single `"key"` entry for the API key; use additional entries for `"endpoint"`, `"organization"`, etc.

---

## Step 2: Create the UI Model Constants

Create `ui/src/app/components/providers/text/<provider>/constants.ts`:

```typescript
// ui/src/app/components/providers/text/myprovider/constants.ts
import { SetMetadata } from '@/utils/metadata';
import { Metadata } from '@rapidaai/react';

export const MYPROVIDER_TEXT_MODEL = [
  {
    id: 'myprovider/model-v1',
    name: 'model-v1',
    human_name: 'Model V1',
    description: 'Standard model.',
    category: 'text',
  },
  {
    id: 'myprovider/model-v2',
    name: 'model-v2',
    human_name: 'Model V2 (Recommended)',
    description: 'Latest and most capable model.',
    category: 'text',
  },
];

/**
 * Returns a complete Metadata[] with sensible defaults for any missing keys.
 * Called whenever the user switches to this provider to initialise state.
 */
export const GetMyProviderTextProviderDefaultOptions = (
  current: Metadata[],
): Metadata[] => {
  const mtds: Metadata[] = [];

  const addMetadata = (
    key: string,
    defaultValue?: string,
    validationFn?: (value: string) => boolean,
  ) => {
    const metadata = SetMetadata(current, key, defaultValue, validationFn);
    if (metadata) mtds.push(metadata);
  };

  // Required: credential reference
  addMetadata('rapida.credential_id');

  // Required: model selection (id and display name stored separately)
  addMetadata(
    'model.id',
    MYPROVIDER_TEXT_MODEL[0].id,
    value => MYPROVIDER_TEXT_MODEL.some(m => m.id === value),
  );
  addMetadata(
    'model.name',
    MYPROVIDER_TEXT_MODEL[0].name,
    value => MYPROVIDER_TEXT_MODEL.some(m => m.name === value),
  );

  // Optional advanced parameters — add as needed:
  addMetadata('model.temperature', '0.7');
  addMetadata('model.max_tokens', '2048');

  return mtds;
};

/**
 * Returns an error message string if options are invalid, or undefined if valid.
 * Called before saving an assistant configuration.
 */
export const ValidateMyProviderTextProviderDefaultOptions = (
  options: Metadata[],
): string | undefined => {
  const credentialId = options.find(o => o.getKey() === 'rapida.credential_id');
  if (!credentialId?.getValue()) {
    return 'Please provide a valid My Provider credential.';
  }

  const modelId = options.find(o => o.getKey() === 'model.id');
  if (!modelId || !MYPROVIDER_TEXT_MODEL.some(m => m.id === modelId.getValue())) {
    return 'Please select a valid My Provider model.';
  }

  return undefined;
};
```

**Model parameter key conventions** (must match the backend `switch` in the caller):

| UI Key | Backend `switch` case | Description |
|--------|-----------------------|-------------|
| `model.id` | `"model.id"` | Full provider/model-name ID (stored but often unused by backend — the caller uses `model.name`) |
| `model.name` | `"model.name"` | The raw model name sent to the provider SDK |
| `model.temperature` | `"model.temperature"` | Sampling temperature (float) |
| `model.top_p` | `"model.top_p"` | Top-p / nucleus sampling |
| `model.max_tokens` | `"model.max_tokens"` | Max completion tokens |
| `model.stop` | `"model.stop"` | Comma-separated stop sequences |
| `model.response_format` | `"model.response_format"` | JSON blob (e.g., `{"type":"json_object"}`) |
| `rapida.credential_id` | (resolved before call — not a model param) | Vault credential ID |

---

## Step 3: Create the UI Config Component

Create `ui/src/app/components/providers/text/<provider>/index.tsx`:

```tsx
// ui/src/app/components/providers/text/myprovider/index.tsx
import React, { FC } from 'react';
import { Metadata } from '@rapidaai/react';
import { Dropdown } from '@/app/components/dropdown';
import { FormLabel } from '@/app/components/form-label';
import { FieldSet } from '@/app/components/form/fieldset';
import {
  MYPROVIDER_TEXT_MODEL,
  GetMyProviderTextProviderDefaultOptions,
  ValidateMyProviderTextProviderDefaultOptions,
} from './constants';

export {
  GetMyProviderTextProviderDefaultOptions,
  ValidateMyProviderTextProviderDefaultOptions,
};

type Props = {
  onParameterChange: (parameters: Metadata[]) => void;
  parameters: Metadata[] | null;
};

const renderOption = (item: { human_name: string; description?: string }) => (
  <span className="inline-flex flex-col text-sm">
    <span className="font-medium">{item.human_name}</span>
    {item.description && (
      <span className="text-xs text-gray-500 truncate">{item.description}</span>
    )}
  </span>
);

export const ConfigureMyProviderTextProviderModel: FC<Props> = ({
  onParameterChange,
  parameters,
}) => {
  const getParam = (key: string) =>
    parameters?.find(p => p.getKey() === key)?.getValue() ?? '';

  const updateParam = (key: string, value: string) => {
    const updated = parameters ? parameters.map(p => p.clone()) : [];
    const idx = updated.findIndex(p => p.getKey() === key);
    if (idx !== -1) {
      updated[idx].setValue(value);
    } else {
      const m = new Metadata();
      m.setKey(key);
      m.setValue(value);
      updated.push(m);
    }
    onParameterChange(updated);
  };

  const handleModelChange = (model: (typeof MYPROVIDER_TEXT_MODEL)[0]) => {
    const updated = parameters ? parameters.map(p => p.clone()) : [];
    const setOrAdd = (key: string, value: string) => {
      const idx = updated.findIndex(p => p.getKey() === key);
      if (idx !== -1) {
        updated[idx].setValue(value);
      } else {
        const m = new Metadata();
        m.setKey(key);
        m.setValue(value);
        updated.push(m);
      }
    };
    setOrAdd('model.id', model.id);
    setOrAdd('model.name', model.name);
    onParameterChange(updated);
  };

  const currentModel =
    MYPROVIDER_TEXT_MODEL.find(m => m.id === getParam('model.id')) ??
    MYPROVIDER_TEXT_MODEL[0];

  return (
    <>
      <FieldSet className="col-span-2 h-fit">
        <FormLabel>Model</FormLabel>
        <Dropdown
          className="bg-light-background max-w-full dark:bg-gray-950"
          currentValue={currentModel}
          setValue={handleModelChange}
          allValue={MYPROVIDER_TEXT_MODEL}
          placeholder="Select model"
          option={renderOption}
          label={renderOption}
        />
      </FieldSet>

      {/* Add sliders / inputs for temperature, max_tokens, etc. following
          the pattern used in ui/src/app/components/providers/text/openai/index.tsx */}
    </>
  );
};
```

> **Reference implementation**: `ui/src/app/components/providers/text/openai/index.tsx` shows a full implementation with a popover for advanced parameters (temperature, top_p, frequency_penalty, max_completion_tokens, seed, response_format, etc.). Copy and adapt that pattern for your provider's supported parameters.

---

## Step 4: Wire into the UI Provider Router

Edit `ui/src/app/components/providers/text/index.tsx`. Add your provider to **all three** switch blocks and one import:

### 4a. Import

```typescript
import {
  ConfigureMyProviderTextProviderModel,
  GetMyProviderTextProviderDefaultOptions,
  ValidateMyProviderTextProviderDefaultOptions,
} from '@/app/components/providers/text/myprovider';
```

### 4b. `GetDefaultTextProviderConfigIfInvalid`

```typescript
case 'myprovider':
  return GetMyProviderTextProviderDefaultOptions(parameters);
```

### 4c. `ValidateTextProviderDefaultOptions`

```typescript
case 'myprovider':
  return ValidateMyProviderTextProviderDefaultOptions(parameters);
```

### 4d. `TextProviderConfigComponent`

```typescript
case 'myprovider':
  return (
    <ConfigureMyProviderTextProviderModel
      parameters={parameters}
      onParameterChange={onChangeParameter}
    />
  );
```

The provider will automatically appear in the `TEXT_PROVIDERS` dropdown because `allProvider()` is filtered by `featureList.includes("text")`.

---

## Step 5: Implement the Backend Caller

### 5a. Directory Structure

```
api/integration-api/internal/caller/myprovider/
├── myprovider.go          # Base struct, GetClient(), usage metric helper
├── llm.go                 # LargeLanguageCaller: GetChatCompletion + StreamChatCompletion
└── verify-credential.go   # Verifier: CredentialVerifier
```

Add optional `text-embedding.go` if the provider also supports embeddings.

### 5b. Base Struct (`myprovider.go`)

```go
package internal_caller_myprovider

import (
    internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
    "github.com/rapidaai/pkg/commons"
    "github.com/rapidaai/protos"
)

var (
    DEFAULT_URL = "https://api.myprovider.com/v1"
    API_KEY     = "key" // must match the credential configuration "name" in provider JSON
)

type MyProvider struct {
    logger     commons.Logger
    credential internal_callers.CredentialResolver
}

func myProvider(logger commons.Logger, credential *protos.Credential) MyProvider {
    _credential := credential.GetValue().AsMap()
    return MyProvider{
        logger:     logger,
        credential: func() map[string]interface{} { return _credential },
    }
}

func (p *MyProvider) GetClient() (*myprovider_sdk.Client, error) {
    credentials := p.credential()
    apiKey, ok := credentials[API_KEY]
    if !ok {
        return nil, fmt.Errorf("myprovider: missing API key credential")
    }
    // Initialize your provider's SDK client here
    client := myprovider_sdk.NewClient(apiKey.(string))
    return client, nil
}

func (p *MyProvider) GetUsageMetrics(usage myprovider_sdk.UsageData) []*protos.Metric {
    return []*protos.Metric{
        {Name: "INPUT_TOKEN",  Value: fmt.Sprintf("%d", usage.PromptTokens)},
        {Name: "OUTPUT_TOKEN", Value: fmt.Sprintf("%d", usage.CompletionTokens)},
        {Name: "TOTAL_TOKEN",  Value: fmt.Sprintf("%d", usage.TotalTokens)},
    }
}
```

### 5c. LLM Caller (`llm.go`)

```go
package internal_caller_myprovider

import (
    "context"
    "fmt"
    "time"

    internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
    internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
    "github.com/rapidaai/pkg/commons"
    "github.com/rapidaai/protos"
)

type largeLanguageCaller struct {
    MyProvider
}

// NewLargeLanguageCaller returns the unexported implementation behind the interface.
func NewLargeLanguageCaller(
    logger commons.Logger,
    credential *protos.Credential,
) internal_callers.LargeLanguageCaller {
    return &largeLanguageCaller{MyProvider: myProvider(logger, credential)}
}

func (c *largeLanguageCaller) GetChatCompletion(
    ctx context.Context,
    allMessages []*protos.Message,
    options *internal_callers.ChatCompletionOptions,
) (*protos.Message, []*protos.Metric, error) {
    metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
    metrics.OnStart()

    client, err := c.GetClient()
    if err != nil {
        metrics.OnFailure()
        return nil, metrics.Build(), err
    }

    // Build request from allMessages + options.ModelParameter
    chatOptions := c.getChatCompletionOptions(options)

    // TODO: call client.Chat(ctx, chatOptions)

    metrics.OnAddMetrics(c.GetUsageMetrics(response.Usage)...)
    metrics.OnSuccess()
    return &protos.Message{/* ... */}, metrics.Build(), nil
}

func (c *largeLanguageCaller) StreamChatCompletion(
    ctx context.Context,
    allMessages []*protos.Message,
    options *internal_callers.ChatCompletionOptions,
    onStream  func(rID string, msg *protos.Message) error,
    onMetrics func(rID string, msg *protos.Message, mtrx []*protos.Metric) error,
    onError   func(rID string, err error),
) error {
    metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
    metrics.OnStart()
    requestId := fmt.Sprintf("%d", options.RequestId)

    client, err := c.GetClient()
    if err != nil {
        metrics.OnFailure()
        onError(requestId, err)
        return err
    }

    chatOptions := c.getChatCompletionOptions(options)
    firstToken := true
    var firstTokenAt time.Time

    // TODO: call client.StreamChat(ctx, chatOptions) and iterate chunks
    // For each chunk:
    //   - If firstToken, record firstTokenAt = time.Now(), firstToken = false
    //   - Call onStream(requestId, &protos.Message{...})
    // After stream ends:
    //   - Add FIRST_TOKEN_RECIEVED_TIME metric
    //   - Call onMetrics(requestId, finalMessage, metrics.Build())

    _ = firstToken
    _ = firstTokenAt

    metrics.OnAddMetrics(c.GetUsageMetrics(response.Usage)...)
    metrics.OnAddMetrics(&protos.Metric{
        Name:  "FIRST_TOKEN_RECIEVED_TIME",
        Value: fmt.Sprintf("%d", firstTokenAt.UnixNano()),
    })
    metrics.OnSuccess()
    return onMetrics(requestId, finalMessage, metrics.Build())
}

// getChatCompletionOptions translates the generic ModelParameter map into
// provider-specific SDK parameters. Keys must match what the UI sends.
func (c *largeLanguageCaller) getChatCompletionOptions(
    options *internal_callers.ChatCompletionOptions,
) *myprovider_sdk.ChatOptions {
    params := options.ModelParameter
    opts := &myprovider_sdk.ChatOptions{}

    if v, ok := internal_callers.GetStringParam(params, "model.name"); ok {
        opts.Model = v
    }
    if v, ok := internal_callers.GetFloat64Param(params, "model.temperature"); ok {
        opts.Temperature = v
    }
    if v, ok := internal_callers.GetInt64Param(params, "model.max_tokens"); ok {
        opts.MaxTokens = v
    }
    if v, ok := internal_callers.GetStringParam(params, "model.stop"); ok {
        opts.StopSequences = strings.Split(v, ",")
    }
    // Add other provider-specific parameters as needed

    return opts
}
```

> **Helpers for reading `map[string]*anypb.Any`**: Look at the existing OpenAI or Anthropic caller for the pattern of extracting typed values from `options.ModelParameter`. Each caller does its own inline type-assertion after calling `anypb.UnmarshalTo` or using the `structpb` helpers.

### 5d. Credential Verifier (`verify-credential.go`)

```go
package internal_caller_myprovider

import (
    "context"
    "fmt"

    internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
    "github.com/rapidaai/pkg/commons"
    "github.com/rapidaai/protos"
)

type verifyCredentialCaller struct {
    MyProvider
}

func NewVerifyCredentialCaller(
    logger commons.Logger,
    credential *protos.Credential,
) internal_callers.Verifier {
    return &verifyCredentialCaller{MyProvider: myProvider(logger, credential)}
}

func (v *verifyCredentialCaller) CredentialVerifier(
    ctx context.Context,
    options *internal_callers.CredentialVerifierOptions,
) (*string, error) {
    client, err := v.GetClient()
    if err != nil {
        return nil, err
    }
    // Make a minimal test call (e.g., list models or a zero-token completion)
    // to confirm the key is valid.
    _ = client
    msg := "credential verified"
    return &msg, nil
}
```

---

## Step 6: Create the gRPC API Handler

Create `api/integration-api/api/<provider>.go`:

```go
package integration_api

import (
    "context"

    internal_caller_myprovider "github.com/rapidaai/api/integration-api/internal/caller/myprovider"
    "github.com/rapidaai/api/integration-api/config"
    "github.com/rapidaai/pkg/commons"
    "github.com/rapidaai/pkg/connectors"
    "github.com/rapidaai/protos"
)

const MYPROVIDER_INTEGRATION_NAME = "MYPROVIDER"

type myProviderIntegrationGRPCApi struct {
    protos.UnimplementedMyProviderServiceServer
    integrationApi
}

func NewMyProviderGRPC(
    cfg *config.IntegrationConfig,
    logger commons.Logger,
    postgres connectors.PostgresConnector,
) protos.MyProviderServiceServer {
    return &myProviderIntegrationGRPCApi{
        integrationApi: newIntegrationApi(cfg, logger, postgres),
    }
}

func (iApi *myProviderIntegrationGRPCApi) Chat(
    ctx context.Context,
    req *protos.ChatRequest,
) (*protos.ChatResponse, error) {
    return iApi.integrationApi.Chat(
        ctx, req,
        MYPROVIDER_INTEGRATION_NAME,
        internal_caller_myprovider.NewLargeLanguageCaller(iApi.logger, req.GetCredential()),
    )
}

func (iApi *myProviderIntegrationGRPCApi) StreamChat(
    stream protos.MyProviderService_StreamChatServer,
) error {
    return iApi.integrationApi.StreamChatBidirectional(
        stream.Context(),
        MYPROVIDER_INTEGRATION_NAME,
        func(credential *protos.Credential) internal_callers.LargeLanguageCaller {
            return internal_caller_myprovider.NewLargeLanguageCaller(iApi.logger, credential)
        },
        stream,
    )
}

func (iApi *myProviderIntegrationGRPCApi) VerifyCredential(
    ctx context.Context,
    req *protos.Credential,
) (*protos.VerifyCredentialResponse, error) {
    return iApi.integrationApi.VerifyCredential(
        ctx,
        MYPROVIDER_INTEGRATION_NAME,
        internal_caller_myprovider.NewVerifyCredentialCaller(iApi.logger, req),
        &internal_callers.CredentialVerifierOptions{},
    )
}
```

> **Reference**: `api/integration-api/api/openai.go` and `api/integration-api/api/anthropic.go` are complete, production implementations to follow.

---

## Step 7: Register the gRPC Service

Edit `api/integration-api/router/provider.go` — add one line:

```go
protos.RegisterMyProviderServiceServer(S, integrationApi.NewMyProviderGRPC(Cfg, Logger, Postgres))
```

---

## Step 8: Add to the Integration Client

Edit `pkg/clients/integration/integration_client.go`.

### 8a. Add the client field

```go
type integrationServiceClientGRPC struct {
    // ... existing fields ...
    myProviderClient protos.MyProviderServiceClient
}
```

### 8b. Initialize in the constructor

```go
func NewIntegrationServiceClientGRPC(conn *grpc.ClientConn) IntegrationServiceClient {
    return &integrationServiceClientGRPC{
        // ... existing initializations ...
        myProviderClient: protos.NewMyProviderServiceClient(conn),
    }
}
```

### 8c. Add cases to all relevant switches

**`Chat` switch:**
```go
case "myprovider":
    response, err := c.myProviderClient.Chat(ctx, chatRequest)
    // ... handle response
```

**`StreamChat` switch:**
```go
case "myprovider":
    stream, err := c.myProviderClient.StreamChat(ctx)
    // ... return stream
```

**`VerifyCredential` switch:**
```go
case "myprovider":
    return c.myProviderClient.VerifyCredential(ctx, credential)
```

**`Embedding` switch** (if applicable):
```go
case "myprovider":
    return c.myProviderClient.GetEmbedding(ctx, embeddingRequest)
```

> **Important**: The switch key is `strings.ToLower(providerName)` — it must exactly match the `code` in the provider JSON.

---

## Step 9: Proto Service Definition (if required)

> Skip this step if reusing an existing proto service (e.g., for OpenAI-compatible providers).

Proto sources live in `protos/artifacts/` (a git submodule). Add `myprovider.proto`:

```protobuf
syntax = "proto3";
package rapida.protos;
option go_package = "github.com/rapidaai/protos";

import "chat.proto";
import "credential.proto";

service MyProviderService {
  rpc Chat(ChatRequest) returns (ChatResponse);
  rpc StreamChat(stream ChatRequest) returns (stream ChatResponse);
  rpc VerifyCredential(Credential) returns (VerifyCredentialResponse);
}
```

Then regenerate:
```bash
buf generate
```

Generated Go files land in `protos/` automatically.

---

## Step 10: Metrics Reference

Every caller **must** use `internal_caller_metrics.NewMetricBuilder` and emit these standard metrics:

```go
metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
metrics.OnStart()       // starts timer; sets STATUS=FAILED by default

// ... perform LLM call ...

metrics.OnAddMetrics(   // add token counts from provider's usage object
    &protos.Metric{Name: "INPUT_TOKEN",  Value: "..."},
    &protos.Metric{Name: "OUTPUT_TOKEN", Value: "..."},
    &protos.Metric{Name: "TOTAL_TOKEN",  Value: "..."},
)
metrics.OnSuccess()     // records TIME_TAKEN; sets STATUS=SUCCESS
// or:
metrics.OnFailure()     // keeps STATUS=FAILED

return nil, metrics.Build(), err
```

For streaming, additionally emit:
```go
&protos.Metric{
    Name:  "FIRST_TOKEN_RECIEVED_TIME",   // note: intentional typo kept for consistency
    Value: fmt.Sprintf("%d", firstTokenAt.UnixNano()),
}
```

**Canonical metric names** (from `pkg/types/enums/metric.go`):

| Name | Description |
|------|-------------|
| `TIME_TAKEN` | Total wall-clock duration in nanoseconds |
| `STATUS` | `"SUCCESS"` or `"FAILED"` |
| `INPUT_TOKEN` | Prompt tokens consumed |
| `OUTPUT_TOKEN` | Completion tokens produced |
| `TOTAL_TOKEN` | Input + output tokens |
| `LLM_REQUEST_ID` | Internal snowflake request ID |
| `FIRST_TOKEN_RECIEVED_TIME` | Unix nanoseconds of first streamed chunk (streaming only) |

---

## Step 11: Credential Flow (How Keys Reach the Caller)

Understanding this prevents credential-related bugs:

1. User adds API key in **Integrations > Vault** → stored as `VaultCredential.value = {"key": "sk-..."}` in web-api DB.
2. In assistant config, the UI saves `rapida.credential_id = <vault_id>` in the `AssistantProviderModelOption` table.
3. At session start, `pipeline.go` calls `VaultCaller().GetCredential(ctx, auth, credentialId)` → fetches from web-api gRPC (Redis-cached).
4. The returned `*protos.VaultCredential` has `.GetValue().AsMap()` → passed as `*protos.Credential` to `NewLargeLanguageCaller`.
5. In the caller, `credential.GetValue().AsMap()[API_KEY].(string)` extracts the key.

**The `API_KEY` constant in `myprovider.go` must match the `name` field in the `configurations` array of the provider JSON.**

---

## Step 12: Test

### Backend

```bash
# Build the integration-api to catch compile errors
go build ./api/integration-api/...

# Run existing tests
go test ./api/integration-api/...

# Run a specific caller test
go test ./api/integration-api/internal/caller/myprovider/...
```

### Frontend

```bash
cd ui && yarn checkTs   # TypeScript type check
cd ui && yarn test       # Run UI tests
```

### Integration

1. `make up-all`
2. Go to **Integrations > Vault**, add your provider credentials
3. Create or edit an assistant, go to **Model** settings
4. Select your provider from the LLM dropdown
5. Select a model, configure parameters, save
6. Open the **Debugger** and send a test message
7. Verify the response streams back correctly
8. Check **Analytics** for token metrics

---

## Checklist

### Frontend
- [ ] Provider entry added to `provider.development.json` and `provider.production.json` with correct `code` and `featureList: ["text", "external"]`
- [ ] `constants.ts` created with model array, `GetDefaultOptions`, and `ValidateOptions` functions
- [ ] `index.tsx` config component created with model dropdown (and optional advanced parameters popover)
- [ ] Config component, defaults, and validation wired into all **3 switch blocks** in `ui/src/app/components/providers/text/index.tsx`
- [ ] `model.id` and `model.name` both set when user selects a model
- [ ] `rapida.credential_id` required in validation

### Backend (integration-api)
- [ ] Caller package created: `myprovider.go`, `llm.go`, `verify-credential.go`
- [ ] `LargeLanguageCaller` interface implemented: `GetChatCompletion` + `StreamChatCompletion`
- [ ] `Verifier` interface implemented: `CredentialVerifier`
- [ ] `MetricBuilder` used in both `GetChatCompletion` and `StreamChatCompletion`
- [ ] `INPUT_TOKEN`, `OUTPUT_TOKEN`, `TOTAL_TOKEN` emitted from usage data
- [ ] `FIRST_TOKEN_RECIEVED_TIME` emitted in `StreamChatCompletion`
- [ ] `getChatCompletionOptions()` handles all model parameter keys sent by the UI
- [ ] gRPC API handler created (`api/<provider>.go`) with `Chat`, `StreamChat`, `VerifyCredential`
- [ ] Line added to `api/integration-api/router/provider.go`
- [ ] Proto service defined and generated (if new proto required)

### Integration client
- [ ] Client field added to `integrationServiceClientGRPC` struct
- [ ] Client initialized in `NewIntegrationServiceClientGRPC`
- [ ] `case "myprovider":` added to `Chat` switch
- [ ] `case "myprovider":` added to `StreamChat` switch
- [ ] `case "myprovider":` added to `VerifyCredential` switch
- [ ] `case "myprovider":` added to `Embedding` switch (if applicable)

### Cross-cutting
- [ ] Provider `code` string matches **exactly** (case-insensitive) across: JSON file, UI switch-cases, integration client switch, caller `API_KEY` constant
- [ ] `configurations[].name` in provider JSON matches `API_KEY` constant in caller
- [ ] `go build ./...` passes with no errors
- [ ] Integration tested end-to-end with a live assistant conversation
