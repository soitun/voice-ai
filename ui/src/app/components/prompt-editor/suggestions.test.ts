import {
  extractPromptVariableQuery,
  getPromptVariableSuggestions,
} from '@/app/components/prompt-editor/suggestions';

describe('Prompt editor reserved variable suggestions', () => {
  it('detects variable trigger when user types {{', () => {
    expect(extractPromptVariableQuery('You are {{')).toBe('');
    expect(extractPromptVariableQuery('You are {{assistant.')).toBe(
      'assistant.',
    );
    expect(extractPromptVariableQuery('You are {assistant')).toBeNull();
  });

  it('returns all reserved suggestions for empty query after {{', () => {
    const suggestions = getPromptVariableSuggestions('Hello {{');

    expect(suggestions.length).toBeGreaterThan(5);
    expect(
      suggestions.some(item => item.label === '{{assistant.name}}'),
    ).toBeTruthy();
    expect(
      suggestions.some(item => item.description.includes('Assistant name')),
    ).toBeTruthy();
    expect(
      suggestions.some(item => item.label === '{{client.phone}}'),
    ).toBeTruthy();
    expect(
      suggestions.some(item => item.label === '{{client.provider_call_id}}'),
    ).toBeTruthy();
  });

  it('filters suggestions by currently typed variable prefix', () => {
    const assistantSuggestions = getPromptVariableSuggestions(
      'Hello {{assistant.',
    );

    expect(assistantSuggestions.length).toBeGreaterThan(0);
    expect(
      assistantSuggestions.every(item => item.key.startsWith('assistant.')),
    ).toBeTruthy();
    expect(
      assistantSuggestions.some(item => item.label === '{{assistant.id}}'),
    ).toBeTruthy();
  });

  it('returns client-specific suggestions when typing client prefix', () => {
    const clientSuggestions = getPromptVariableSuggestions('Hello {{client.');

    expect(clientSuggestions.length).toBeGreaterThan(0);
    expect(
      clientSuggestions.every(item => item.key.startsWith('client.')),
    ).toBeTruthy();
    expect(
      clientSuggestions.some(item => item.label === '{{client.phone}}'),
    ).toBeTruthy();
  });

  it('returns no suggestions when not inside variable template', () => {
    expect(getPromptVariableSuggestions('Hello user')).toEqual([]);
    expect(getPromptVariableSuggestions('Hello {user')).toEqual([]);
  });
});
