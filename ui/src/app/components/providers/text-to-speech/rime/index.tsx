/**
 * Rapida – Open Source Voice AI Orchestration Platform
 * Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
 * Licensed under a modified GPL-2.0. See the LICENSE file for details.
 */
import { Metadata } from '@rapidaai/react';
import { FormLabel } from '@/app/components/form-label';
import { FieldSet } from '@/app/components/form/fieldset';
import { useEffect, useState } from 'react';
import { RIME_LANGUAGE, RIME_MODEL, RIME_VOICE } from '@/providers';
import { CustomValueDropdown } from '@/app/components/dropdown/custom-value-dropdown';
import { Dropdown } from '@/app/components/dropdown';
import { Input } from '../../../form/input';
export { GetRimeDefaultOptions, ValidateRimeOptions } from './constant';

const renderOption = c => (
  <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
    <span className="truncate capitalize">{c.name}</span>
  </span>
);

export const ConfigureRimeTextToSpeech: React.FC<{
  onParameterChange: (parameters: Metadata[]) => void;
  parameters: Metadata[] | null;
}> = ({ onParameterChange, parameters }) => {
  const selectedModel =
    parameters?.find(p => p.getKey() === 'speak.model')?.getValue() ?? '';
  const selectedLanguage =
    parameters?.find(p => p.getKey() === 'speak.language')?.getValue() ?? '';

  const [filteredLanguages, setFilteredLanguages] = useState(
    RIME_LANGUAGE(selectedModel || undefined),
  );
  const [filteredVoices, setFilteredVoices] = useState(
    RIME_VOICE(selectedModel || undefined, selectedLanguage || undefined),
  );

  // Sync languages when model changes
  useEffect(() => {
    setFilteredLanguages(RIME_LANGUAGE(selectedModel || undefined));
  }, [selectedModel]);

  // Sync voices when model or language changes
  useEffect(() => {
    setFilteredVoices(
      RIME_VOICE(selectedModel || undefined, selectedLanguage || undefined),
    );
  }, [selectedModel, selectedLanguage]);

  const getParamValue = (key: string) =>
    parameters?.find(p => p.getKey() === key)?.getValue() ?? '';

  const updateParameter = (key: string, value: string) => {
    const updatedParams = [...(parameters || [])];
    const existingIndex = updatedParams.findIndex(p => p.getKey() === key);
    const newParam = new Metadata();
    newParam.setKey(key);
    newParam.setValue(value);
    if (existingIndex >= 0) {
      updatedParams[existingIndex] = newParam;
    } else {
      updatedParams.push(newParam);
    }
    onParameterChange(updatedParams);
  };

  const handleModelChange = (model: { model_id: string }) => {
    const modelLanguages = RIME_LANGUAGE(model.model_id);
    setFilteredLanguages(modelLanguages);
    setFilteredVoices([]);

    // Batch model change, reset language and voice
    const updatedParams = [...(parameters || [])];
    const setParam = (key: string, value: string) => {
      const idx = updatedParams.findIndex(p => p.getKey() === key);
      const param = new Metadata();
      param.setKey(key);
      param.setValue(value);
      if (idx >= 0) updatedParams[idx] = param;
      else updatedParams.push(param);
    };
    setParam('speak.model', model.model_id);
    setParam('speak.language', '');
    setParam('speak.voice.id', '');
    onParameterChange(updatedParams);
  };

  const handleLanguageChange = (language: { language_id: string }) => {
    const langVoices = RIME_VOICE(selectedModel, language.language_id);
    setFilteredVoices(langVoices);

    // Batch language change and reset voice
    const updatedParams = [...(parameters || [])];
    const setParam = (key: string, value: string) => {
      const idx = updatedParams.findIndex(p => p.getKey() === key);
      const param = new Metadata();
      param.setKey(key);
      param.setValue(value);
      if (idx >= 0) updatedParams[idx] = param;
      else updatedParams.push(param);
    };
    setParam('speak.language', language.language_id);
    setParam('speak.voice.id', '');
    onParameterChange(updatedParams);
  };

  return (
    <>
      <FieldSet className="col-span-1">
        <FormLabel>Model</FormLabel>
        <Dropdown
          className="bg-light-background max-w-full dark:bg-gray-950"
          currentValue={RIME_MODEL().find(
            x => x.model_id === getParamValue('speak.model'),
          )}
          setValue={handleModelChange}
          allValue={RIME_MODEL()}
          placeholder={`Select model`}
          option={renderOption}
          label={renderOption}
        />
      </FieldSet>
      <FieldSet className="col-span-1">
        <FormLabel>Language</FormLabel>
        <Dropdown
          className="bg-light-background max-w-full dark:bg-gray-950"
          currentValue={filteredLanguages.find(
            x => x.language_id === getParamValue('speak.language'),
          )}
          setValue={handleLanguageChange}
          allValue={filteredLanguages}
          placeholder={`Select language`}
          option={renderOption}
          label={renderOption}
        />
      </FieldSet>
      <FieldSet className="col-span-1">
        <FormLabel>Voice</FormLabel>
        <CustomValueDropdown
          searchable
          className="bg-light-background max-w-full dark:bg-gray-950"
          currentValue={filteredVoices.find(
            x => x.voice_id === getParamValue('speak.voice.id'),
          )}
          setValue={(v: { voice_id: string }) =>
            updateParameter('speak.voice.id', v.voice_id)
          }
          allValue={filteredVoices}
          customValue
          onSearching={t => {
            const voices = RIME_VOICE(
              selectedModel || undefined,
              selectedLanguage || undefined,
            );
            const v = t.target.value;
            if (v.length > 0) {
              setFilteredVoices(
                voices.filter(
                  voice =>
                    voice.name.toLowerCase().includes(v.toLowerCase()) ||
                    voice.voice_id.toLowerCase().includes(v.toLowerCase()),
                ),
              );
              return;
            }
            setFilteredVoices(voices);
          }}
          onAddCustomValue={vl => {
            setFilteredVoices(prev => [...prev, { voice_id: vl, name: vl }]);
            updateParameter('speak.voice.id', vl);
          }}
          placeholder={`Select voice`}
          option={renderOption}
          label={renderOption}
        />
      </FieldSet>
      <FieldSet className="col-span-1">
        <FormLabel>Speed Alpha</FormLabel>
        <Input
          type="number"
          step="0.1"
          min="0.1"
          max="3.0"
          placeholder="1.0 (default)"
          value={getParamValue('speak.speed_alpha')}
          onChange={e => updateParameter('speak.speed_alpha', e.target.value)}
        />
      </FieldSet>
    </>
  );
};
