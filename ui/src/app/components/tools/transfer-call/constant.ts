import { Metadata } from '@rapidaai/react';
import { getOptionValue, buildDefaultMetadata } from '../common';

// ============================================================================
// Constants
// ============================================================================

export const SEPARATOR = '<|||>';
const REQUIRED_KEYS = ['tool.transfer_to'];
const ALL_KEYS = [
  ...REQUIRED_KEYS,
  'tool.transfer_message',
  'tool.transfer_delay',
  'tool.post_transfer_action',
  'tool.ringtone',
];
const ALLOWED_POST_TRANSFER_ACTIONS = ['end_call', 'resume_ai'];
const ALLOWED_RINGTONES = ['ringtone_in', 'ringtone_uk', 'ringtone_us'];

// ============================================================================
// Default Options
// ============================================================================

export const GetTransferCallDefaultOptions = (
  current: Metadata[],
): Metadata[] =>
  buildDefaultMetadata(
    current,
    [
      { key: 'tool.transfer_to' },
      { key: 'tool.transfer_message' },
      { key: 'tool.transfer_delay', defaultValue: '500' },
      { key: 'tool.post_transfer_action', defaultValue: 'end_call' },
      { key: 'tool.ringtone', defaultValue: 'ringtone_us' },
    ],
    ALL_KEYS,
  );

// ============================================================================
// Validation
// ============================================================================

export const ValidateTransferCallDefaultOptions = (
  options: Metadata[],
): string | undefined => {
  const transferTo = getOptionValue(options, 'tool.transfer_to');
  if (!transferTo || !transferTo.trim()) {
    return 'Please provide at least one phone number or SIP URI to transfer calls to.';
  }

  const postTransferAction = getOptionValue(
    options,
    'tool.post_transfer_action',
  );
  if (
    postTransferAction &&
    !ALLOWED_POST_TRANSFER_ACTIONS.includes(postTransferAction)
  ) {
    return `Post transfer action must be one of: ${ALLOWED_POST_TRANSFER_ACTIONS.join(', ')}.`;
  }

  const ringtone = getOptionValue(options, 'tool.ringtone');
  if (ringtone && !ALLOWED_RINGTONES.includes(ringtone)) {
    return `Ringtone must be one of: ${ALLOWED_RINGTONES.join(', ')}.`;
  }

  return undefined;
};
