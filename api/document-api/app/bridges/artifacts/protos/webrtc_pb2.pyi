import app.bridges.artifacts.protos.talk_api_pb2 as _talk_api_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ICEServer(_message.Message):
    __slots__ = ("urls", "username", "credential")
    URLS_FIELD_NUMBER: _ClassVar[int]
    USERNAME_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_FIELD_NUMBER: _ClassVar[int]
    urls: _containers.RepeatedScalarFieldContainer[str]
    username: str
    credential: str
    def __init__(self, urls: _Optional[_Iterable[str]] = ..., username: _Optional[str] = ..., credential: _Optional[str] = ...) -> None: ...

class ICECandidate(_message.Message):
    __slots__ = ("candidate", "sdpMid", "sdpMLineIndex", "usernameFragment")
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    SDPMID_FIELD_NUMBER: _ClassVar[int]
    SDPMLINEINDEX_FIELD_NUMBER: _ClassVar[int]
    USERNAMEFRAGMENT_FIELD_NUMBER: _ClassVar[int]
    candidate: str
    sdpMid: str
    sdpMLineIndex: int
    usernameFragment: str
    def __init__(self, candidate: _Optional[str] = ..., sdpMid: _Optional[str] = ..., sdpMLineIndex: _Optional[int] = ..., usernameFragment: _Optional[str] = ...) -> None: ...

class WebRTCSDP(_message.Message):
    __slots__ = ("type", "sdp")
    class SDPType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        SDP_TYPE_UNSPECIFIED: _ClassVar[WebRTCSDP.SDPType]
        OFFER: _ClassVar[WebRTCSDP.SDPType]
        ANSWER: _ClassVar[WebRTCSDP.SDPType]
    SDP_TYPE_UNSPECIFIED: WebRTCSDP.SDPType
    OFFER: WebRTCSDP.SDPType
    ANSWER: WebRTCSDP.SDPType
    TYPE_FIELD_NUMBER: _ClassVar[int]
    SDP_FIELD_NUMBER: _ClassVar[int]
    type: WebRTCSDP.SDPType
    sdp: str
    def __init__(self, type: _Optional[_Union[WebRTCSDP.SDPType, str]] = ..., sdp: _Optional[str] = ...) -> None: ...

class ClientSignaling(_message.Message):
    __slots__ = ("sessionId", "sdp", "iceCandidate", "disconnect")
    SESSIONID_FIELD_NUMBER: _ClassVar[int]
    SDP_FIELD_NUMBER: _ClassVar[int]
    ICECANDIDATE_FIELD_NUMBER: _ClassVar[int]
    DISCONNECT_FIELD_NUMBER: _ClassVar[int]
    sessionId: str
    sdp: WebRTCSDP
    iceCandidate: ICECandidate
    disconnect: bool
    def __init__(self, sessionId: _Optional[str] = ..., sdp: _Optional[_Union[WebRTCSDP, _Mapping]] = ..., iceCandidate: _Optional[_Union[ICECandidate, _Mapping]] = ..., disconnect: bool = ...) -> None: ...

class ServerSignaling(_message.Message):
    __slots__ = ("sessionId", "config", "sdp", "iceCandidate", "ready", "clear", "error")
    SESSIONID_FIELD_NUMBER: _ClassVar[int]
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    SDP_FIELD_NUMBER: _ClassVar[int]
    ICECANDIDATE_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    CLEAR_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    sessionId: str
    config: WebRTCConfig
    sdp: WebRTCSDP
    iceCandidate: ICECandidate
    ready: bool
    clear: bool
    error: str
    def __init__(self, sessionId: _Optional[str] = ..., config: _Optional[_Union[WebRTCConfig, _Mapping]] = ..., sdp: _Optional[_Union[WebRTCSDP, _Mapping]] = ..., iceCandidate: _Optional[_Union[ICECandidate, _Mapping]] = ..., ready: bool = ..., clear: bool = ..., error: _Optional[str] = ...) -> None: ...

class WebRTCConfig(_message.Message):
    __slots__ = ("iceServers", "audioCodec", "sampleRate")
    ICESERVERS_FIELD_NUMBER: _ClassVar[int]
    AUDIOCODEC_FIELD_NUMBER: _ClassVar[int]
    SAMPLERATE_FIELD_NUMBER: _ClassVar[int]
    iceServers: _containers.RepeatedCompositeFieldContainer[ICEServer]
    audioCodec: str
    sampleRate: int
    def __init__(self, iceServers: _Optional[_Iterable[_Union[ICEServer, _Mapping]]] = ..., audioCodec: _Optional[str] = ..., sampleRate: _Optional[int] = ...) -> None: ...

class WebTalkRequest(_message.Message):
    __slots__ = ("initialization", "configuration", "message", "signaling", "metadata", "metric", "disconnection", "toolCallResult")
    INITIALIZATION_FIELD_NUMBER: _ClassVar[int]
    CONFIGURATION_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SIGNALING_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    METRIC_FIELD_NUMBER: _ClassVar[int]
    DISCONNECTION_FIELD_NUMBER: _ClassVar[int]
    TOOLCALLRESULT_FIELD_NUMBER: _ClassVar[int]
    initialization: _talk_api_pb2.ConversationInitialization
    configuration: _talk_api_pb2.ConversationConfiguration
    message: _talk_api_pb2.ConversationUserMessage
    signaling: ClientSignaling
    metadata: _talk_api_pb2.ConversationMetadata
    metric: _talk_api_pb2.ConversationMetric
    disconnection: _talk_api_pb2.ConversationDisconnection
    toolCallResult: _talk_api_pb2.ConversationToolCallResult
    def __init__(self, initialization: _Optional[_Union[_talk_api_pb2.ConversationInitialization, _Mapping]] = ..., configuration: _Optional[_Union[_talk_api_pb2.ConversationConfiguration, _Mapping]] = ..., message: _Optional[_Union[_talk_api_pb2.ConversationUserMessage, _Mapping]] = ..., signaling: _Optional[_Union[ClientSignaling, _Mapping]] = ..., metadata: _Optional[_Union[_talk_api_pb2.ConversationMetadata, _Mapping]] = ..., metric: _Optional[_Union[_talk_api_pb2.ConversationMetric, _Mapping]] = ..., disconnection: _Optional[_Union[_talk_api_pb2.ConversationDisconnection, _Mapping]] = ..., toolCallResult: _Optional[_Union[_talk_api_pb2.ConversationToolCallResult, _Mapping]] = ...) -> None: ...

class WebTalkResponse(_message.Message):
    __slots__ = ("code", "success", "initialization", "configuration", "interruption", "user", "assistant", "toolCall", "toolCallResult", "error", "signaling", "metadata", "metric", "disconnection", "event")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    INITIALIZATION_FIELD_NUMBER: _ClassVar[int]
    CONFIGURATION_FIELD_NUMBER: _ClassVar[int]
    INTERRUPTION_FIELD_NUMBER: _ClassVar[int]
    USER_FIELD_NUMBER: _ClassVar[int]
    ASSISTANT_FIELD_NUMBER: _ClassVar[int]
    TOOLCALL_FIELD_NUMBER: _ClassVar[int]
    TOOLCALLRESULT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    SIGNALING_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    METRIC_FIELD_NUMBER: _ClassVar[int]
    DISCONNECTION_FIELD_NUMBER: _ClassVar[int]
    EVENT_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    initialization: _talk_api_pb2.ConversationInitialization
    configuration: _talk_api_pb2.ConversationConfiguration
    interruption: _talk_api_pb2.ConversationInterruption
    user: _talk_api_pb2.ConversationUserMessage
    assistant: _talk_api_pb2.ConversationAssistantMessage
    toolCall: _talk_api_pb2.ConversationToolCall
    toolCallResult: _talk_api_pb2.ConversationToolCallResult
    error: _talk_api_pb2.ConversationError
    signaling: ServerSignaling
    metadata: _talk_api_pb2.ConversationMetadata
    metric: _talk_api_pb2.ConversationMetric
    disconnection: _talk_api_pb2.ConversationDisconnection
    event: _talk_api_pb2.ConversationEvent
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., initialization: _Optional[_Union[_talk_api_pb2.ConversationInitialization, _Mapping]] = ..., configuration: _Optional[_Union[_talk_api_pb2.ConversationConfiguration, _Mapping]] = ..., interruption: _Optional[_Union[_talk_api_pb2.ConversationInterruption, _Mapping]] = ..., user: _Optional[_Union[_talk_api_pb2.ConversationUserMessage, _Mapping]] = ..., assistant: _Optional[_Union[_talk_api_pb2.ConversationAssistantMessage, _Mapping]] = ..., toolCall: _Optional[_Union[_talk_api_pb2.ConversationToolCall, _Mapping]] = ..., toolCallResult: _Optional[_Union[_talk_api_pb2.ConversationToolCallResult, _Mapping]] = ..., error: _Optional[_Union[_talk_api_pb2.ConversationError, _Mapping]] = ..., signaling: _Optional[_Union[ServerSignaling, _Mapping]] = ..., metadata: _Optional[_Union[_talk_api_pb2.ConversationMetadata, _Mapping]] = ..., metric: _Optional[_Union[_talk_api_pb2.ConversationMetric, _Mapping]] = ..., disconnection: _Optional[_Union[_talk_api_pb2.ConversationDisconnection, _Mapping]] = ..., event: _Optional[_Union[_talk_api_pb2.ConversationEvent, _Mapping]] = ...) -> None: ...
