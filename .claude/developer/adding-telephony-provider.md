# Adding a New Telephony Provider (Frontend to Backend)

This guide covers the complete end-to-end steps for adding a new telephony provider to Rapida — from the React UI through to the Go backend streamer, telephony webhook handler, and factory registration.

---

## Overview of Touchpoints

| Layer | Files | Purpose |
|-------|-------|---------|
| **Provider Registry** | `ui/src/providers/provider.development.json` + `.production.json` | Registers provider with `"telephony"` + `"external"` feature flags |
| **UI Config Component** | `ui/src/app/components/providers/telephony/<provider>/index.tsx` | Provider-specific configuration form (phone number, app IDs, etc.) |
| **UI Provider Router** | `ui/src/app/components/providers/telephony/index.tsx` | Switch-case routing to config component + validation |
| **Streamer** | `api/assistant-api/internal/channel/telephony/internal/<provider>/websocket.go` | Go: implements `Streamer` — reads/writes audio over the transport |
| **Telephony Handler** | `api/assistant-api/internal/channel/telephony/internal/<provider>/telephony.go` | Go: implements `Telephony` — handles HTTP webhooks (inbound/outbound/status) |
| **Factory** | `api/assistant-api/internal/channel/telephony/telephony.go` | Adds provider constant + cases in `GetTelephony()` and `NewStreamer()` |

> **No router changes needed.** The existing HTTP routes in `api/assistant-api/` already handle any provider code generically:
> - `GET /v1/talk/:telephony/call/:assistantId` → `CallReciever` (inbound webhook)
> - `GET /v1/talk/:telephony/ctx/:contextId` → `CallTalkerByContext` (upgrades to WebSocket, creates streamer)
> - `GET|POST /v1/talk/:telephony/ctx/:contextId/event` → `CallbackByContext` (status callbacks)

---

## Architecture: How a Telephony Call Works

Understanding this flow prevents mistakes in both the Telephony and Streamer implementations.

```
External system places a call
  │
  ▼
GET /v1/talk/<provider>/call/:assistantId   ← HTTP webhook
  │  InboundDispatcher:
  │   1. tel.ReceiveCall(c)          → parses provider payload, returns CallInfo
  │   2. Creates Conversation in DB
  │   3. Saves CallContext to Postgres (status=pending)
  │   4. tel.InboundCall(c, ...)     → provider-specific response
  │                                     (TwiML / NCCO / JSON with WS URL)
  │  Provider follows the URL and connects:
  ▼
GET /v1/talk/<provider>/ctx/:contextId      ← HTTP upgraded to WebSocket
  │  CallTalkerByContext:
  │   1. Upgrades HTTP → WebSocket
  │   2. Atomically claims CallContext (pending → claimed)
  │   3. Fetches VaultCredential via gRPC
  │   4. Telephony(cc.Provider).NewStreamer(logger, cc, vaultCred, {WebSocketConn: ws})
  │   5. GetTalker(PhoneCall, ..., streamer)  → genericRequestor
  │   6. talker.Talk(ctx, auth)
  │       └── Streamer.Recv() loop → OnPacket() → STT → LLM → TTS
  │           Notify() → Streamer.Send() → audio back to provider
  │   7. CompleteCallSession(contextID)
  ▼
GET|POST /v1/talk/<provider>/ctx/:contextId/event   ← status callbacks (async)
```

### Key types

| Type | Package | Purpose |
|------|---------|---------|
| `internal_type.Streamer` | `api/assistant-api/internal/type` | Transport abstraction: `Recv()`, `Send()`, `NotifyMode()`, `Context()` |
| `internal_type.Telephony` | `api/assistant-api/internal/type` | Webhook handler: `ReceiveCall`, `InboundCall`, `OutboundCall`, `StatusCallback` |
| `BaseTelephonyStreamer` | `internal/channel/telephony/internal/base` | Embeds `BaseStreamer`; adds `CallContext`, resampler, encoder |
| `CallContext` | `api/assistant-api/internal/callcontext` | Bridges the HTTP webhook to the WebSocket media session |

---

## Step 1: Register the Provider (Frontend)

### 1a. Add to Provider Registry JSON

Edit both `ui/src/providers/provider.development.json` and `ui/src/providers/provider.production.json`. Add an entry:

```json
{
    "code": "myprovider",
    "name": "My Provider",
    "description": "Description of what this telephony provider offers.",
    "image": "https://cdn-01.rapida.ai/partners/myprovider.png",
    "featureList": ["telephony", "external"],
    "configurations": [
        {
            "name": "account_sid",
            "type": "string",
            "label": "Account SID"
        },
        {
            "name": "auth_token",
            "type": "string",
            "label": "Auth Token"
        }
    ],
    "website": "https://myprovider.com"
}
```

Key fields:
- **`code`**: Lowercase unique identifier. Must match the `Telephony` constant in the backend factory and the route param `:telephony`.
- **`featureList`**: Always `["telephony", "external"]` — `"telephony"` shows the provider in the telephony dropdown; `"external"` shows it in vault/credential management.
- **`configurations`**: Credential fields shown in the vault form. The `name` becomes the key in the vault credential value map — it must match what the backend reads from `vaultCredential.GetValue().AsMap()["<name>"]`.

---

## Step 2: Create the UI Config Component

### 2a. Directory structure

```
ui/src/app/components/providers/telephony/myprovider/
└── index.tsx     # Config component + ValidateMyProviderTelephonyOptions
```

### 2b. Implement `index.tsx`

```tsx
// ui/src/app/components/providers/telephony/myprovider/index.tsx
import { Metadata } from '@rapidaai/react';
import { FormLabel } from '@/app/components/form-label';
import { FieldSet } from '@/app/components/form/fieldset';
import { Input } from '@/app/components/form/input';
import { InputHelper } from '@/app/components/input-helper';

export const ValidateMyProviderTelephonyOptions = (
  options: Metadata[],
): boolean => {
  const credentialId = options.find(o => o.getKey() === 'rapida.credential_id');
  if (!credentialId?.getValue()) return false;

  // Validate any provider-specific required fields:
  const phone = options.find(o => o.getKey() === 'phone');
  if (!phone?.getValue()) return false;

  return true;
};

export const ConfigureMyProviderTelephony: React.FC<{
  onParameterChange: (parameters: Metadata[]) => void;
  parameters: Metadata[] | null;
}> = ({ onParameterChange, parameters }) => {
  const getParam = (key: string) =>
    parameters?.find(p => p.getKey() === key)?.getValue() ?? '';

  const updateParameter = (key: string, value: string) => {
    const updated = [...(parameters || [])];
    const idx = updated.findIndex(p => p.getKey() === key);
    const m = new Metadata();
    m.setKey(key);
    m.setValue(value);
    if (idx >= 0) updated[idx] = m;
    else updated.push(m);
    onParameterChange(updated);
  };

  return (
    <>
      <FieldSet className="col-span-2">
        <FormLabel>Phone Number</FormLabel>
        <Input
          className="bg-light-background"
          value={getParam('phone')}
          onChange={e => updateParameter('phone', e.target.value)}
          placeholder="Enter your provider phone number"
        />
        <InputHelper>Phone number for inbound or outbound calls.</InputHelper>
      </FieldSet>
    </>
  );
};
```

**Common parameter keys** (pick what applies):

| Key | Description |
|-----|-------------|
| `rapida.credential_id` | Always required — vault credential reference |
| `phone` | Provider phone number |
| `app_id` | Provider application / trunk ID |
| `webhook_url` | Custom webhook URL override |

---

## Step 3: Wire into the UI Provider Router

Edit `ui/src/app/components/providers/telephony/index.tsx`.

### 3a. Import

```typescript
import {
  ConfigureMyProviderTelephony,
  ValidateMyProviderTelephonyOptions,
} from '@/app/components/providers/telephony/myprovider';
```

### 3b. `ValidateTelephonyOptions` switch — add case

```typescript
case 'myprovider':
  return ValidateMyProviderTelephonyOptions(parameters);
```

### 3c. `ConfigureTelephonyComponent` switch — add case

```typescript
case 'myprovider':
  return (
    <ConfigureMyProviderTelephony
      parameters={parameters || []}
      onParameterChange={onChangeParameter}
    />
  );
```

The provider automatically appears in the `TELEPHONY_PROVIDER` dropdown because it filters `allProvider()` by `featureList.includes('telephony')`.

---

## Step 4: Implement the Backend Streamer

### 4a. Directory structure

```
api/assistant-api/internal/channel/telephony/internal/myprovider/
├── websocket.go     # Streamer (or transport-specific file)
└── telephony.go     # Telephony (webhook handler)
```

### 4b. Decide on audio format

The first thing to determine is the wire audio format your provider uses:

| Provider | Format | `WithSourceAudioConfig(...)` |
|----------|--------|------------------------------|
| Vonage | Linear16 16kHz (= Rapida internal) | `nil` (no resampling) |
| Exotel | Linear16 8kHz | `internal_audio.NewLinear8khzMonoAudioConfig()` |
| Twilio | mulaw 8kHz | `internal_audio.NewMulaw8khzMonoAudioConfig()` |

When `sourceAudioConfig` is non-nil, `BaseTelephonyStreamer.CreateVoiceRequest(audio)` automatically resamples from the provider format to Rapida internal (16kHz linear16) before dispatching the packet. When the provider needs resampled _output_ audio, call `tws.Resampler().Resample(content.Audio, RAPIDA_AUDIO_CONFIG, PROVIDER_AUDIO_CONFIG)` in `Send()`.

### 4c. Implement `websocket.go`

```go
package internal_myprovider_telephony

import (
    "bytes"
    "encoding/json"

    "github.com/gorilla/websocket"
    callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
    internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
    internal_type "github.com/rapidaai/api/assistant-api/internal/type"
    "github.com/rapidaai/pkg/commons"
    "github.com/rapidaai/protos"
    "google.golang.org/protobuf/types/known/timestamppb"
)

type myProviderWebsocketStreamer struct {
    internal_telephony_base.BaseTelephonyStreamer

    streamID   string
    connection *websocket.Conn
}

func NewMyProviderWebsocketStreamer(
    logger commons.Logger,
    connection *websocket.Conn,
    cc *callcontext.CallContext,
    vaultCred *protos.VaultCredential,
) internal_type.Streamer {
    s := &myProviderWebsocketStreamer{
        BaseTelephonyStreamer: internal_telephony_base.NewBaseTelephonyStreamer(
            logger, cc, vaultCred,
            // TODO: set source audio config if provider format ≠ 16kHz linear16:
            // internal_telephony_base.WithSourceAudioConfig(internal_audio.NewMulaw8khzMonoAudioConfig()),
        ),
        connection: connection,
    }
    go s.runWebSocketReader()
    return s
}

func (s *myProviderWebsocketStreamer) runWebSocketReader() {
    conn := s.connection
    if conn == nil {
        return
    }
    for {
        msgType, message, err := conn.ReadMessage()
        if err != nil {
            s.PushDisconnection(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER)
            s.BaseStreamer.Cancel()
            return
        }

        switch msgType {
        case websocket.TextMessage:
            // Parse JSON control events from provider
            var event map[string]interface{}
            if err := json.Unmarshal(message, &event); err != nil {
                s.Logger.Error("failed to unmarshal text event", "error", err.Error())
                continue
            }
            switch event["event"] {
            case "connected":
                // Some providers send a "connected" event first (before audio starts)
                s.PushInputLow(&protos.ConversationEvent{
                    Name: "channel",
                    Data: map[string]string{"type": "connected", "provider": "myprovider"},
                    Time: timestamppb.Now(),
                })
            case "start":
                // Provider signals media stream is starting — push connection request
                s.streamID = fmt.Sprintf("%v", event["streamSid"])
                s.PushInput(s.CreateConnectionRequest())
                s.PushInputLow(&protos.ConversationEvent{
                    Name: "channel",
                    Data: map[string]string{
                        "type":      "stream_started",
                        "provider":  "myprovider",
                        "stream_id": s.streamID,
                    },
                    Time: timestamppb.Now(),
                })
            case "media":
                // Base64-encoded audio chunk (Twilio-style)
                if media, ok := event["media"].(map[string]interface{}); ok {
                    if payload, ok := media["payload"].(string); ok {
                        decoded, _ := s.Encoder().DecodeString(payload)
                        s.WithInputBuffer(func(buf *bytes.Buffer) {
                            buf.Write(decoded)
                            if buf.Len() >= s.InputBufferThreshold() {
                                s.PushInput(s.CreateVoiceRequest(buf.Bytes()))
                                buf.Reset()
                            }
                        })
                    }
                }
            case "stop":
                s.Cancel()
                s.PushDisconnection(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER)
                return
            default:
                s.Logger.Debugf("unhandled event: %v", event["event"])
            }

        case websocket.BinaryMessage:
            // Raw binary audio (Vonage-style)
            s.WithInputBuffer(func(buf *bytes.Buffer) {
                buf.Write(message)
                if buf.Len() >= s.InputBufferThreshold() {
                    s.PushInput(s.CreateVoiceRequest(buf.Bytes()))
                    buf.Reset()
                }
            })

        default:
            s.Logger.Warnf("unhandled websocket message type %d", msgType)
        }
    }
}

func (s *myProviderWebsocketStreamer) Send(response internal_type.Stream) error {
    if s.connection == nil {
        return nil
    }
    switch data := response.(type) {
    case *protos.ConversationAssistantMessage:
        switch content := data.Message.(type) {
        case *protos.ConversationAssistantMessage_Audio:
            audioData := content.Audio

            // TODO: if provider format ≠ 16kHz linear16, resample output:
            // audioData, err = s.Resampler().Resample(content.Audio, RAPIDA_AUDIO_CONFIG, PROVIDER_AUDIO_CONFIG)

            var sendErr error
            s.WithOutputBuffer(func(buf *bytes.Buffer) {
                buf.Write(audioData)
                for buf.Len() >= s.OutputFrameSize() {
                    chunk := buf.Next(s.OutputFrameSize())
                    // TODO: wrap in provider envelope if needed (e.g. base64 JSON)
                    if err := s.connection.WriteMessage(websocket.BinaryMessage, chunk); err != nil {
                        s.Logger.Error("failed to send audio chunk", "error", err.Error())
                        sendErr = err
                        return
                    }
                }
                if data.GetCompleted() && buf.Len() > 0 {
                    remaining := buf.Bytes()
                    if err := s.connection.WriteMessage(websocket.BinaryMessage, remaining); err != nil {
                        s.Logger.Errorf("failed to send final audio chunk: %v", err)
                        sendErr = err
                        return
                    }
                    buf.Reset()
                }
            })
            return sendErr
        }

    case *protos.ConversationInterruption:
        if data.Type == protos.ConversationInterruption_INTERRUPTION_TYPE_WORD {
            s.ResetOutputBuffer()
            // TODO: send provider-specific clear/flush command
            // e.g. Twilio: {"event":"clear","streamSid":"..."}
            //      Vonage: {"action":"clear"}
        }

    case *protos.ConversationDirective:
        if data.GetType() == protos.ConversationDirective_END_CONVERSATION {
            // TODO: call provider API to hang up (e.g. Twilio UpdateCall, Vonage Hangup)
            // If no API is needed, just close the WebSocket:
            if err := s.Cancel(); err != nil {
                s.Logger.Errorf("error disconnecting: %v", err)
            }
        }
    }
    return nil
}

func (s *myProviderWebsocketStreamer) Cancel() error {
    if s.connection != nil {
        s.connection.Close()
        s.connection = nil
    }
    s.BaseStreamer.Cancel()
    return nil
}
```

### 4d. Audio framing details

`BaseTelephonyStreamer` helpers used in `Send()`:

| Helper | What it does |
|--------|-------------|
| `s.WithOutputBuffer(fn)` | Thread-safe access to output buffer `*bytes.Buffer` |
| `s.OutputFrameSize()` | Byte size of one 20 ms output frame at the provider's rate |
| `s.ResetOutputBuffer()` | Drains output buffer + output channel (call on interruption) |
| `s.Resampler().Resample(audio, from, to)` | Converts between audio configs |
| `s.Encoder()` | Base64 encoder (for Twilio/Exotel-style base64 payloads) |

`BaseTelephonyStreamer` helpers used in `runWebSocketReader()`:

| Helper | What it does |
|--------|-------------|
| `s.WithInputBuffer(fn)` | Thread-safe access to input buffer `*bytes.Buffer` |
| `s.InputBufferThreshold()` | Byte count at which buffered audio is flushed upstream |
| `s.CreateVoiceRequest(audio)` | Resamples → 16kHz linear16, wraps in `ConversationUserMessage{Audio:...}` |
| `s.CreateConnectionRequest()` | Builds `ConversationInitialization` (assistant + conversation IDs, StreamMode_AUDIO) |
| `s.PushInput(msg)` | Enqueues message to `InputCh` (normal priority) |
| `s.PushInputLow(msg)` | Enqueues message to `LowCh` (low priority, for observability events) |
| `s.PushDisconnection(reason)` | Idempotent disconnect signal to `CriticalCh` |

---

## Step 5: Implement the Telephony Webhook Handler

```go
package internal_myprovider_telephony

import (
    "fmt"
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/rapidaai/api/assistant-api/config"
    internal_type "github.com/rapidaai/api/assistant-api/internal/type"
    "github.com/rapidaai/pkg/commons"
    "github.com/rapidaai/pkg/types"
    "github.com/rapidaai/pkg/utils"
    "github.com/rapidaai/protos"
)

const myProviderName = "myprovider"

type myProviderTelephony struct {
    logger commons.Logger
    appCfg *config.AssistantConfig
}

func NewMyProviderTelephony(
    cfg *config.AssistantConfig,
    logger commons.Logger,
) (internal_type.Telephony, error) {
    return &myProviderTelephony{logger: logger, appCfg: cfg}, nil
}

// ReceiveCall is called for every inbound call webhook.
// Parse the provider payload and return a CallInfo.
// The route handler uses CallInfo to create the conversation and CallContext in DB.
func (t *myProviderTelephony) ReceiveCall(c *gin.Context) (*internal_type.CallInfo, error) {
    // TODO: parse provider-specific query params or request body
    callerNumber := c.Query("caller_number") // example

    if callerNumber == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "missing caller_number"})
        return nil, fmt.Errorf("missing caller_number")
    }

    info := &internal_type.CallInfo{
        Provider:     myProviderName,
        Status:       "SUCCESS",
        CallerNumber: callerNumber,
        StatusInfo:   internal_type.StatusInfo{Event: "webhook"},
    }
    // Set ChannelUUID from provider's call ID if available:
    if callSid := c.Query("call_sid"); callSid != "" {
        info.ChannelUUID = callSid
    }
    return info, nil
}

// InboundCall is called immediately after ReceiveCall.
// Respond to the provider with instructions to open the WebSocket stream.
// The contextId gin.Context key holds the CallContext ID set by the route handler.
func (t *myProviderTelephony) InboundCall(
    c *gin.Context,
    auth types.SimplePrinciple,
    assistantId uint64,
    clientNumber string,
    assistantConversationId uint64,
) error {
    contextID, _ := c.Get("contextId")
    ctxID := fmt.Sprintf("%v", contextID)

    // Build the WebSocket URL the provider should connect to:
    wsURL := fmt.Sprintf("wss://%s/%s",
        t.appCfg.PublicAssistantHost,
        internal_type.GetContextAnswerPath(myProviderName, ctxID),
    )

    // TODO: respond in the format your provider expects.
    // For a simple JSON response:
    c.JSON(http.StatusOK, gin.H{"websocket_url": wsURL})

    // For TwiML (Twilio):
    // c.Header("Content-Type", "application/xml")
    // c.String(http.StatusOK, `<Response><Connect><Stream url="%s"/></Connect></Response>`, wsURL)

    // For NCCO (Vonage):
    // c.JSON(http.StatusOK, []gin.H{{"action":"connect","endpoint":[{"type":"websocket","uri":wsURL,...}]}})
    return nil
}

// StatusCallback handles asynchronous call-status webhooks (ringing, answered, ended, etc.).
func (t *myProviderTelephony) StatusCallback(
    c *gin.Context,
    auth types.SimplePrinciple,
    assistantId uint64,
    assistantConversationId uint64,
) (*internal_type.StatusInfo, error) {
    // TODO: parse provider status payload
    return &internal_type.StatusInfo{Event: c.Query("CallStatus")}, nil
}

// CatchAllStatusCallback handles provider webhooks that do not carry a conversation ID
// (e.g. Twilio's account-level status callbacks).
func (t *myProviderTelephony) CatchAllStatusCallback(
    c *gin.Context,
) (*internal_type.StatusInfo, error) {
    return nil, nil
}

// OutboundCall initiates an outbound call from the provider.
// The CallContext ID is available in opts via opts.GetString("rapida.context_id").
func (t *myProviderTelephony) OutboundCall(
    auth types.SimplePrinciple,
    toPhone string,
    fromPhone string,
    assistantId uint64,
    assistantConversationId uint64,
    vaultCredential *protos.VaultCredential,
    opts utils.Option,
) (*internal_type.CallInfo, error) {
    // TODO: use provider SDK to initiate call
    // Refer to exotel/telephony.go or twilio/telephony.go for complete examples

    contextID, _ := opts.GetString("rapida.context_id")
    statusCallbackURL := fmt.Sprintf("https://%s/%s",
        t.appCfg.PublicAssistantHost,
        internal_type.GetContextEventPath(myProviderName, contextID),
    )
    wsURL := fmt.Sprintf("wss://%s/%s",
        t.appCfg.PublicAssistantHost,
        internal_type.GetContextAnswerPath(myProviderName, contextID),
    )
    t.logger.Infof("outbound call: to=%s, ws=%s, status=%s", toPhone, wsURL, statusCallbackURL)

    return &internal_type.CallInfo{
        Provider: myProviderName,
        Status:   "SUCCESS",
    }, nil
}
```

### URL helpers (`internal_type`)

| Helper | Result |
|--------|--------|
| `GetContextAnswerPath(provider, ctxID)` | `"v1/talk/<provider>/ctx/<ctxID>"` |
| `GetContextEventPath(provider, ctxID)` | `"v1/talk/<provider>/ctx/<ctxID>/event"` |

Use `t.appCfg.PublicAssistantHost` (the publicly reachable hostname) when constructing absolute URLs returned to the provider.

---

## Step 6: Register in the Factory

Edit `api/assistant-api/internal/channel/telephony/telephony.go`.

### 6a. Add the constant

```go
const (
    Twilio      Telephony = "twilio"
    Exotel      Telephony = "exotel"
    Vonage      Telephony = "vonage"
    Asterisk    Telephony = "asterisk"
    SIP         Telephony = "sip"
    MyProvider  Telephony = "myprovider"   // ← add this
)
```

### 6b. Add import

```go
internal_myprovider_telephony "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/myprovider"
```

### 6c. `GetTelephony()` — add case

```go
case MyProvider:
    return internal_myprovider_telephony.NewMyProviderTelephony(cfg, logger)
```

### 6d. `NewStreamer()` — add case

```go
case MyProvider:
    return internal_myprovider_telephony.NewMyProviderWebsocketStreamer(
        logger, opt.WebSocketConn, cc, vaultCred,
    ), nil
```

If your provider uses a non-WebSocket transport (like Asterisk's AudioSocket or SIP's RTP), add the required fields to `StreamerOption` and use them here.

---

## Step 7: Credential Flow

The credential is resolved at session start, not at webhook time:

1. User adds API credentials in **Integrations > Vault** → stored as `VaultCredential.value = {"account_sid": "...", "auth_token": "..."}`.
2. The `CallContext` stores the `AssistantProviderId` (link to the `AssistantProvider` record that holds `rapida.credential_id`).
3. `CallTalkerByContext` resolves `vaultCred` via `VaultCaller().GetCredential(ctx, auth, credentialId)` (Redis-cached, fetched from web-api gRPC).
4. `vaultCred` is passed to both `NewStreamer(...)` and is available in the streamer via `s.VaultCredential()`.

Inside the streamer, access credentials like:
```go
credentials := s.VaultCredential().GetValue().AsMap()
accountSid  := credentials["account_sid"].(string)
authToken   := credentials["auth_token"].(string)
```

**The credential key names must match the `name` fields in the `configurations` array of the provider JSON.**

---

## Step 8: Required Events (ConversationEventPacket)

Every streamer should emit `ConversationEventPacket` via `PushInputLow()` at lifecycle points:

| When | `Data["type"]` | Additional `Data` fields |
|------|---------------|--------------------------|
| WebSocket connection opens | `"connected"` | `"provider"` |
| Media stream starts | `"stream_started"` | `"provider"`, `"stream_id"` |
| Call ends normally | `"disconnected"` | `"provider"`, `"reason"` |
| Error from provider | `"error"` | `"provider"`, `"error"` |

These power the Debugger UI's channel event timeline.

```go
s.PushInputLow(&protos.ConversationEvent{
    Name: "channel",
    Data: map[string]string{
        "type":     "connected",
        "provider": "myprovider",
    },
    Time: timestamppb.Now(),
})
```

---

## Step 9: Test

### Backend

```bash
# Build assistant-api to catch compile errors
go build ./api/assistant-api/...

# Run existing tests
go test ./api/assistant-api/internal/channel/telephony/...
```

### Frontend

```bash
cd ui && yarn checkTs
cd ui && yarn test
```

### Integration

1. `make up-all`
2. Go to **Integrations > Vault**, add your provider credentials
3. Create/edit a **Phone** assistant deployment
4. Select your provider from the Telephony dropdown, enter config (phone number, etc.), save
5. Use the provider's dashboard to trigger an inbound call to the webhook URL:
   `https://<your-host>/v1/talk/myprovider/call/<assistantId>`
6. Verify the WebSocket upgrade to:
   `wss://<your-host>/v1/talk/myprovider/ctx/<contextId>`
7. Confirm STT → LLM → TTS pipeline produces audio back over the WebSocket
8. Check the **Debugger** for channel events (`connected`, `stream_started`)

---

## Common Patterns by Provider Type

### JSON + base64 audio (Twilio-style)
- Text frames contain JSON with base64-encoded audio chunks in a `media.payload` field
- Use `s.Encoder().DecodeString(payload)` to decode inbound audio
- Use `s.Encoder().EncodeToString(chunk)` to encode outbound audio
- Send a `{"event":"clear","streamSid":"..."}` text frame on interruption
- Resample outbound if needed: `s.Resampler().Resample(audio, RAPIDA_AUDIO_CONFIG, PROVIDER_AUDIO_CONFIG)`
- Reference: `api/assistant-api/internal/channel/telephony/internal/twilio/websocket.go`

### Binary audio (Vonage-style)
- Binary frames carry raw PCM directly — no base64, no JSON wrapper
- If provider uses 16kHz linear16: no resampling needed (same as Rapida internal format)
- Send `{"action":"clear"}` text frame on interruption
- Reference: `api/assistant-api/internal/channel/telephony/internal/vonage/websocket.go`

### Non-WebSocket transport (SIP/RTP, Asterisk AudioSocket)
- Add new transport fields to `StreamerOption` in `telephony.go`
- Override `Context()` if the session lifetime is managed externally (see SIP streamer)
- Use a paced writer goroutine for RTP (20 ms intervals) — reference: `internal/sip/streamer.go`

---

## Checklist

### Frontend
- [ ] Provider entry added to `provider.development.json` and `provider.production.json` with `"telephony"` and `"external"` in `featureList`
- [ ] `configurations` keys match credential map keys read by the backend
- [ ] `index.tsx` config component created with `ConfigureMyProviderTelephony` and `ValidateMyProviderTelephonyOptions`
- [ ] `rapida.credential_id` validated in `ValidateMyProviderTelephonyOptions`
- [ ] Config component wired into **both** switches in `ui/src/app/components/providers/telephony/index.tsx`

### Backend — Streamer
- [ ] Struct embeds `BaseTelephonyStreamer`
- [ ] `NewXxxWebsocketStreamer(logger, conn, cc, vaultCred)` returns `internal_type.Streamer`
- [ ] `WithSourceAudioConfig(...)` set if provider format ≠ 16kHz linear16
- [ ] `runWebSocketReader()` started as goroutine in constructor
- [ ] On connection event → `PushInput(s.CreateConnectionRequest())` + `PushInputLow(ConversationEvent{type:"connected"})`
- [ ] On audio frame → `WithInputBuffer` → threshold → `PushInput(s.CreateVoiceRequest(...))`
- [ ] On stop/close → `PushDisconnection(DISCONNECTION_TYPE_USER)` + `s.BaseStreamer.Cancel()`
- [ ] `Send()` handles `ConversationAssistantMessage_Audio`, `ConversationInterruption`, `ConversationDirective`
- [ ] Outbound audio resampled if provider format ≠ 16kHz linear16
- [ ] `OutputFrameSize()` / `WithOutputBuffer()` used for paced audio framing
- [ ] `ResetOutputBuffer()` called on interruption
- [ ] `Cancel()` closes WebSocket and calls `s.BaseStreamer.Cancel()`

### Backend — Telephony
- [ ] `NewXxxTelephony(cfg, logger)` returns `internal_type.Telephony`
- [ ] `ReceiveCall()` parses provider webhook, returns `CallInfo{Provider, CallerNumber, ChannelUUID, Status}`
- [ ] `InboundCall()` responds to provider with WebSocket URL using `GetContextAnswerPath()`
- [ ] `StatusCallback()` parses async status events (can return nil for unhandled)
- [ ] `CatchAllStatusCallback()` implemented (can return nil, nil)
- [ ] `OutboundCall()` implemented (or returns `fmt.Errorf("not supported")` if not needed)

### Factory
- [ ] `MyProvider Telephony = "myprovider"` constant added (matches `code` in provider JSON)
- [ ] Import added for the new package
- [ ] Case added to `GetTelephony()` switch
- [ ] Case added to `NewStreamer()` switch
- [ ] Provider `code` string matches across: JSON file, UI switch-cases, Go constant, route param

### Cross-cutting
- [ ] `configurations` key names in provider JSON match backend `vaultCredential.GetValue().AsMap()["<key>"]` reads
- [ ] `GetContextAnswerPath(provider, ctxID)` used in `InboundCall()` and `OutboundCall()`
- [ ] `GetContextEventPath(provider, ctxID)` used for status callback URL in `OutboundCall()`
- [ ] `go build ./api/assistant-api/...` passes
- [ ] Integration tested end-to-end with a live call
