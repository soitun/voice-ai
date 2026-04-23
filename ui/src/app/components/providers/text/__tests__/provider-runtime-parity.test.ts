import { Metadata } from '@rapidaai/react';
import {
  GetDefaultTextProviderConfigIfInvalid,
  GetDefaultTextProviderConfigOnProviderSwitch,
  ValidateTextProviderDefaultOptions,
} from '../index';
import { TEXT_PROVIDERS } from '@/providers';
import { loadProviderConfig, loadProviderData } from '@/providers/config-loader';

jest.mock('@/app/components/providers', () => ({}));
jest.mock('@/utils', () => ({
  cn: (...inputs: any[]) => inputs.filter(Boolean).join(' '),
}));
jest.mock('@/app/components/dropdown', () => ({
  Dropdown: () => null,
}));
jest.mock('@/app/components/dropdown/credential-dropdown', () => ({
  CredentialDropdown: () => null,
}));
jest.mock('@/app/components/form/fieldset', () => ({
  FieldSet: ({ children }: any) => children ?? null,
}));
jest.mock('@/app/components/form-label', () => ({
  FormLabel: ({ children }: any) => children ?? null,
}));
jest.mock('@/app/components/providers/config-renderer', () => ({
  ConfigRenderer: () => null,
}));

const createMetadata = (key: string, value: string): Metadata => {
  const m = new Metadata();
  m.setKey(key);
  m.setValue(value);
  return m;
};

const cloneMetadata = (source: Metadata[]): Metadata[] =>
  source.map(m => createMetadata(m.getKey(), m.getValue()));

const normalizeMetadata = (source: Metadata[]): string[] =>
  source
    .map(m => `${m.getKey()}=${m.getValue()}`)
    .sort((a, b) => a.localeCompare(b));

const getMetadataValue = (source: Metadata[], key: string): string => {
  const value = source.find(m => m.getKey() === key)?.getValue();
  return value === undefined || value === null ? '' : String(value);
};

const getModelConfigContext = (provider: string, modelIndex = 0) => {
  const textConfig = loadProviderConfig(provider)?.text;
  expect(textConfig).toBeDefined();

  const modelParam = textConfig!.parameters[0];
  expect(modelParam.data).toBeDefined();

  const catalog = loadProviderData(provider, modelParam.data!);
  expect(catalog.length).toBeGreaterThan(modelIndex);

  const model = catalog[modelIndex];
  const valueField = modelParam.valueField || 'id';
  const nameField = modelParam.linkedField?.sourceField || 'name';

  return {
    model,
    modelId: String(model[valueField]),
    modelName: String(model[nameField] ?? model[valueField]),
    parameterDefaults: Object.fromEntries(
      (model?.config?.parameters ?? [])
        .filter((param: any) => param.default !== undefined)
        .map((param: any) => [param.key, String(param.default)]),
    ) as Record<string, string>,
  };
};

const buildAlternateValue = (param: any): string | null => {
  const current = String(param.default);

  switch (param.type) {
    case 'slider':
    case 'number': {
      const currentNum = Number(current);
      const step = typeof param.step === 'number' ? param.step : 1;
      const candidates = [
        param.min,
        param.max,
        Number.isFinite(currentNum) ? currentNum + step : undefined,
        Number.isFinite(currentNum) ? currentNum - step : undefined,
      ];

      for (const candidate of candidates) {
        if (typeof candidate !== 'number' || Number.isNaN(candidate)) continue;
        if (param.min !== undefined && candidate < param.min) continue;
        if (param.max !== undefined && candidate > param.max) continue;
        const normalized = String(candidate);
        if (normalized !== current) return normalized;
      }
      return null;
    }
    case 'select': {
      const alternative = (param.choices ?? []).find(
        (choice: any) => String(choice.value) !== current,
      );
      return alternative ? String(alternative.value) : null;
    }
    case 'input':
      return current === 'custom-value' ? 'custom-value-2' : 'custom-value';
    case 'json':
      return current === '{"mode":"custom"}'
        ? '{"mode":"custom-2"}'
        : '{"mode":"custom"}';
    default:
      return null;
  }
};

const getOverridableDefaultParam = (provider: string, modelIndex = 1) => {
  const { model } = getModelConfigContext(provider, modelIndex);
  return (model?.config?.parameters ?? []).find(
    (param: any) =>
      param.default !== undefined && buildAlternateValue(param) !== null,
  );
};

const withCredential = (source: Metadata[]): Metadata[] => {
  const cloned = cloneMetadata(source);
  const credential = cloned.find(m => m.getKey() === 'rapida.credential_id');
  if (credential) {
    credential.setValue('test-credential');
    return cloned;
  }
  cloned.push(createMetadata('rapida.credential_id', 'test-credential'));
  return cloned;
};

const withMetadataValue = (
  source: Metadata[],
  key: string,
  value: string,
): Metadata[] => {
  const cloned = cloneMetadata(source);
  const item = cloned.find(m => m.getKey() === key);
  if (item) {
    item.setValue(value);
    return cloned;
  }
  cloned.push(createMetadata(key, value));
  return cloned;
};

describe('Text provider runtime parity', () => {
  const configuredTextProviders = TEXT_PROVIDERS.filter(p =>
    Boolean(loadProviderConfig(p.code)?.text),
  );

  it('all supported text providers are config-driven', () => {
    expect(configuredTextProviders.length).toBeGreaterThan(0);
    for (const provider of configuredTextProviders) {
      expect(loadProviderConfig(provider.code)?.text).toBeDefined();
    }
  });

  it.each(configuredTextProviders.map(p => p.code))(
    '%s keeps text config focused on model selector only',
    provider => {
      const textConfig = loadProviderConfig(provider)?.text;
      expect(textConfig).toBeDefined();
      expect(textConfig?.parameters.length).toBe(1);
      expect(textConfig?.parameters[0].key).toBe('model.id');
      expect(textConfig?.parameters[0].type).toBe('dropdown');
      expect(Boolean(textConfig?.parameters[0].data)).toBe(true);
    },
  );

  it.each(configuredTextProviders.map(p => p.code))(
    '%s model catalog uses per-model config.parameters',
    provider => {
      const textConfig = loadProviderConfig(provider)?.text;
      const dataFile = textConfig?.parameters[0]?.data;
      expect(dataFile).toBeDefined();
      const modelCatalog = loadProviderData(provider, dataFile!);
      expect(modelCatalog.length).toBeGreaterThan(0);
      for (const model of modelCatalog) {
        expect(Array.isArray(model?.config?.parameters)).toBe(true);
        expect(model.config.parameters.length).toBeGreaterThan(0);
      }
    },
  );

  it.each(configuredTextProviders.map(p => p.code))(
    '%s defaults + validation are stable',
    provider => {
      const seed = [
        createMetadata('custom.key', 'custom'),
        createMetadata('rapida.credential_id', 'seed-cred'),
      ];
      const defaults = GetDefaultTextProviderConfigIfInvalid(
        provider,
        cloneMetadata(seed),
      );

      expect(defaults.some(m => m.getKey() === 'model.id')).toBe(true);
      expect(defaults.some(m => m.getKey() === 'rapida.credential_id')).toBe(
        true,
      );

      const validated = ValidateTextProviderDefaultOptions(
        provider,
        withCredential(defaults),
      );
      expect(validated).toBeUndefined();
    },
  );

  it.each(configuredTextProviders.map(p => p.code))(
    '%s hydrates first-time parameters from the default model config',
    provider => {
      const { modelId, modelName, parameterDefaults } =
        getModelConfigContext(provider);

      const defaults = GetDefaultTextProviderConfigIfInvalid(provider, []);

      expect(getMetadataValue(defaults, 'model.id')).toBe(modelId);
      expect(getMetadataValue(defaults, 'model.name')).toBe(modelName);

      for (const [key, value] of Object.entries(parameterDefaults)) {
        expect(getMetadataValue(defaults, key)).toBe(value);
      }
    },
  );

  it.each(configuredTextProviders.map((provider, index) => [provider.code, configuredTextProviders[(index + 1) % configuredTextProviders.length].code]))(
    '%s resets stale text parameters when switching from %s',
    (targetProvider, sourceProvider) => {
      const sourceDefaults = GetDefaultTextProviderConfigIfInvalid(
        sourceProvider,
        [],
      );
      const { modelId, modelName, parameterDefaults } =
        getModelConfigContext(targetProvider);

      const switched = GetDefaultTextProviderConfigOnProviderSwitch(
        targetProvider,
        cloneMetadata(sourceDefaults),
      );

      expect(getMetadataValue(switched, 'model.id')).toBe(modelId);
      expect(getMetadataValue(switched, 'model.name')).toBe(modelName);

      for (const [key, value] of Object.entries(parameterDefaults)) {
        expect(getMetadataValue(switched, key)).toBe(value);
      }

      const sourceOnlyDefaults = Object.keys(
        getModelConfigContext(sourceProvider).parameterDefaults,
      ).filter(key => !(key in parameterDefaults));

      for (const key of sourceOnlyDefaults) {
        expect(getMetadataValue(switched, key)).toBe('');
      }
    },
  );

  it.each(configuredTextProviders.map(p => p.code))(
    '%s preserves explicit existing values and fills any missing defaults',
    provider => {
      const { modelId, modelName, parameterDefaults } = getModelConfigContext(
        provider,
        1,
      );
      const overridableParam = getOverridableDefaultParam(provider, 1);
      expect(overridableParam).toBeDefined();

      const overrideKey = overridableParam!.key;
      const overrideValue = buildAlternateValue(overridableParam!);
      expect(overrideValue).not.toBeNull();

      const seeded = [
        createMetadata('model.id', modelId),
        createMetadata('model.name', modelName),
        createMetadata(overrideKey, overrideValue!),
      ];

      const hydrated = GetDefaultTextProviderConfigIfInvalid(
        provider,
        seeded,
      );

      expect(getMetadataValue(hydrated, 'model.id')).toBe(modelId);
      expect(getMetadataValue(hydrated, 'model.name')).toBe(modelName);
      expect(getMetadataValue(hydrated, overrideKey)).toBe(overrideValue!);

      for (const [key, value] of Object.entries(parameterDefaults)) {
        if (key === overrideKey) continue;
        expect(getMetadataValue(hydrated, key)).toBe(value);
      }
    },
  );

  it('unknown provider remains no-op for defaults and returns validation error', () => {
    const seed = [createMetadata('custom.key', 'custom')];
    expect(
      normalizeMetadata(
        GetDefaultTextProviderConfigIfInvalid(
          'unknown-provider',
          cloneMetadata(seed),
        ),
      ),
    ).toEqual(normalizeMetadata(seed));
    expect(ValidateTextProviderDefaultOptions('unknown-provider', [])).toBe(
      'Please select a valid model and provider.',
    );
  });

  it('normalizes template-style model token values into canonical model id/name', () => {
    const openai = GetDefaultTextProviderConfigIfInvalid('openai', [
      createMetadata('model.id', 'gpt-4o'),
      createMetadata('model.name', 'gpt-4o'),
    ]);
    expect(openai.find(m => m.getKey() === 'model.id')?.getValue()).toBe(
      'openai/gpt-4o',
    );
    expect(openai.find(m => m.getKey() === 'model.name')?.getValue()).toBe(
      'gpt-4o',
    );

    const gemini = GetDefaultTextProviderConfigIfInvalid('gemini', [
      createMetadata('model.id', 'gemini-2.5-flash'),
      createMetadata('model.name', 'gemini-2.5-flash'),
    ]);
    expect(gemini.find(m => m.getKey() === 'model.id')?.getValue()).toBe(
      'gemini/gemini-2.5-flash',
    );
    expect(gemini.find(m => m.getKey() === 'model.name')?.getValue()).toBe(
      'gemini-2.5-flash',
    );
  });

  it('custom-model providers keep explicit custom ids', () => {
    const azure = GetDefaultTextProviderConfigIfInvalid('azure-foundry', [
      createMetadata('model.id', 'my-custom-deployment'),
      createMetadata('model.name', 'my-custom-deployment'),
    ]);
    expect(azure.find(m => m.getKey() === 'model.id')?.getValue()).toBe(
      'my-custom-deployment',
    );
    expect(azure.find(m => m.getKey() === 'model.name')?.getValue()).toBe(
      'my-custom-deployment',
    );
  });

  it('normalizes model tokens during validation', () => {
    const defaults = GetDefaultTextProviderConfigIfInvalid('openai', [
      createMetadata('rapida.credential_id', 'cred-openai'),
    ]);
    const withLegacyModelToken = withMetadataValue(defaults, 'model.id', 'gpt-4o');
    const withLegacyModelName = withMetadataValue(
      withLegacyModelToken,
      'model.name',
      'gpt-4o',
    );
    const err = ValidateTextProviderDefaultOptions(
      'openai',
      withLegacyModelName,
      ['cred-openai'],
    );

    expect(err).toBeUndefined();
  });

  it('rejects stale credential ids that do not belong to selected provider', () => {
    const defaults = GetDefaultTextProviderConfigIfInvalid('openai', [
      createMetadata('rapida.credential_id', 'cred-from-other-provider'),
    ]);
    const err = ValidateTextProviderDefaultOptions(
      'openai',
      defaults,
      ['cred-openai-1', 'cred-openai-2'],
    );

    expect(err).toBe('Please select a valid openai credential.');
  });
});
