export type PromptReservedVariable = {
  key: string;
  variable: string;
  runtimeValue: string;
};

const RESERVED_VARIABLE_KEYS = [
  'system.current_date',
  'system.current_time',
  'system.current_datetime',
  'system.day_of_week',
  'system.date_rfc1123',
  'system.date_unix',
  'system.date_unix_ms',
  'assistant.name',
  'assistant.id',
  'assistant.description',
  'conversation.id',
  'conversation.identifier',
  'conversation.source',
  'conversation.direction',
  'conversation.created_date',
  'message.language',
  'message.text',
] as const;

const RESERVED_VARIABLE_RUNTIME_VALUE: Record<string, string> = {
  'system.current_date': 'UTC date (YYYY-MM-DD)',
  'system.current_time': 'UTC time (HH:MM:SS)',
  'system.current_datetime': 'UTC datetime (RFC3339)',
  'system.day_of_week': 'UTC weekday name',
  'system.date_rfc1123': 'UTC RFC1123 date string',
  'system.date_unix': 'UTC Unix timestamp (seconds)',
  'system.date_unix_ms': 'UTC Unix timestamp (milliseconds)',
  'assistant.name': 'Assistant name',
  'assistant.id': 'Assistant identifier',
  'assistant.description': 'Assistant description',
  'conversation.id': 'Conversation ID',
  'conversation.identifier': 'Conversation identifier',
  'conversation.source': 'Conversation source',
  'conversation.direction': 'Conversation direction',
  'conversation.created_date': 'Conversation created datetime',
  'message.language': 'Current message language',
  'message.text': 'Current message text',
};

export const RAPIDA_RESERVED_RUNTIME_VARIABLES: PromptReservedVariable[] =
  RESERVED_VARIABLE_KEYS.map(key => ({
    key,
    variable: `{{${key}}}`,
    runtimeValue: RESERVED_VARIABLE_RUNTIME_VALUE[key],
  }));

export const RAPIDA_RESERVED_RUNTIME_VARIABLE_KEYS = new Set<string>(
  RESERVED_VARIABLE_KEYS,
);
