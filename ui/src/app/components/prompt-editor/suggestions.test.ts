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

  it('returns no suggestions when not inside variable template', () => {
    expect(getPromptVariableSuggestions('Hello user')).toEqual([]);
    expect(getPromptVariableSuggestions('Hello {user')).toEqual([]);
  });
});
