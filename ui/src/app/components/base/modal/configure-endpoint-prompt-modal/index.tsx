import React, { FC, useState } from 'react';
import { PrimaryButton, SecondaryButton } from '@/app/components/carbon/button';
import {
  Modal,
  ModalHeader,
  ModalBody,
  ModalFooter,
} from '@/app/components/carbon/modal';
import { ModalProps } from '@/app/components/base/modal';
import { cn } from '@/utils';
import endpointTemplates from '@/prompts/endpoints/index.json';
import { Checkmark } from '@carbon/icons-react';
import { CornerBorderOverlay } from '@/app/components/base/corner-border';
import { Tag } from '@carbon/react';

interface EndpointTemplate {
  name: string;
  description: string;
  provider: string;
  model: string;
  parameters: {
    temperature: number;
    response_format: string;
  };
  instruction: {
    role: string;
    content: string;
  }[];
}

interface ConfigureEndpointPromptDialogProps extends ModalProps {
  onSelectTemplate?: (template: EndpointTemplate) => void;
}

export const ConfigureEndpointPromptDialog: FC<
  ConfigureEndpointPromptDialogProps
> = props => {
  const [selectedTemplate, setSelectedTemplate] =
    useState<EndpointTemplate | null>(null);

  const handleContinue = () => {
    if (selectedTemplate && props.onSelectTemplate) {
      props.onSelectTemplate(selectedTemplate);
    }
    props.setModalOpen(false);
  };

  return (
    <Modal
      open={props.modalOpen}
      onClose={() => props.setModalOpen(false)}
      size="lg"
      containerClassName="!w-[900px] !max-w-[900px]"
    >
      <ModalHeader
        label="Endpoint"
        title="Select a usecase template"
        onClose={() => props.setModalOpen(false)}
      />

      <ModalBody hasForm hasScrollingContent>
        <p className="text-xs text-gray-500 dark:text-gray-400 leading-relaxed mb-4">
          Choose a pre-configured template to auto-fill your model, prompt,
          and parameters. You can customise everything after selecting.
        </p>

        <div className="grid grid-cols-2 border-l border-t border-gray-200 dark:border-gray-800">
          {(endpointTemplates as EndpointTemplate[]).map((template, index) => {
            const isSelected = selectedTemplate?.name === template.name;
            return (
              <div
                key={index}
                role="button"
                tabIndex={0}
                onClick={() => setSelectedTemplate(template)}
                onKeyDown={e =>
                  (e.key === 'Enter' || e.key === ' ') &&
                  setSelectedTemplate(template)
                }
                className={cn(
                  'relative flex flex-col p-4 border-r border-b border-gray-200 dark:border-gray-800 cursor-pointer transition-colors duration-100 select-none outline-none group',
                  'focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary',
                  isSelected
                    ? 'bg-primary/5 dark:bg-primary/10'
                    : 'hover:bg-gray-100 dark:hover:bg-gray-800',
                )}
              >
                <CornerBorderOverlay className={isSelected ? 'opacity-100' : undefined} />

                <div
                  className={cn(
                    'absolute top-0 right-0 w-6 h-6 flex items-center justify-center transition-colors duration-100 z-20',
                    isSelected ? 'bg-primary' : 'bg-transparent',
                  )}
                >
                  {isSelected && <Checkmark size={14} className="text-white" />}
                </div>

                <h3 className="text-sm font-semibold leading-snug mb-1.5 pr-6">
                  {template.name}
                </h3>

                <p className="text-xs text-gray-500 dark:text-gray-400 leading-relaxed line-clamp-2 mb-4 flex-1">
                  {template.description}
                </p>

                <div className="flex flex-wrap gap-1.5">
                  <Tag size="sm" type="cool-gray">{template.provider}</Tag>
                  <Tag size="sm" type="cool-gray">{template.model}</Tag>
                  <Tag size="sm" type="cool-gray">Temp {template.parameters.temperature}</Tag>
                  {template.parameters.response_format && (
                    <Tag size="sm" type="blue">JSON</Tag>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      </ModalBody>

      <ModalFooter>
        <SecondaryButton size="lg" onClick={() => props.setModalOpen(false)}>
          Cancel
        </SecondaryButton>
        <PrimaryButton
          size="lg"
          disabled={!selectedTemplate}
          onClick={handleContinue}
        >
          Use template
        </PrimaryButton>
      </ModalFooter>
    </Modal>
  );
};
