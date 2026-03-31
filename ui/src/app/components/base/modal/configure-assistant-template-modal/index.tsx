import React, { FC, useState } from 'react';
import { PrimaryButton, SecondaryButton } from '@/app/components/carbon/button';
import { ModalProps } from '@/app/components/base/modal';
import { Modal, ModalHeader, ModalBody, ModalFooter } from '@/app/components/carbon/modal';
import { cn } from '@/utils';
import assistantTemplates from '@/prompts/assistants/index.json';
import { Checkmark } from '@carbon/icons-react';
import { CornerBorderOverlay } from '@/app/components/base/corner-border';
import { Tag, ContentSwitcher, Switch } from '@carbon/react';

// ── Types ─────────────────────────────────────────────────────────────────────

export interface AssistantTemplate {
  name: string;
  description: string;
  category: string;
  provider: string;
  model: string;
  parameters: {
    temperature: number;
  };
  instruction: {
    role: string;
    content: string;
  }[];
}

interface ConfigureAssistantTemplateDialogProps extends ModalProps {
  onSelectTemplate?: (template: AssistantTemplate) => void;
}

// ── Component ─────────────────────────────────────────────────────────────────

export const ConfigureAssistantTemplateDialog: FC<
  ConfigureAssistantTemplateDialogProps
> = props => {
  const [selectedTemplate, setSelectedTemplate] =
    useState<AssistantTemplate | null>(null);
  const [activeCategory, setActiveCategory] = useState<string>('All');

  const templates = assistantTemplates as AssistantTemplate[];
  const categories = ['All', ...Array.from(new Set(templates.map(t => t.category)))];

  const visible =
    activeCategory === 'All'
      ? templates
      : templates.filter(t => t.category === activeCategory);

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
      containerClassName="!h-[90vh] !w-[90vw] !max-h-[90vh] !max-w-[90vw]"
    >
      <ModalHeader
        label="Assistant"
        title="Select a usecase template"
        onClose={() => props.setModalOpen(false)}
      />

      <ModalBody hasScrollingContent>
        <p className="text-xs text-gray-500 dark:text-gray-400 leading-relaxed mb-4">
          Choose a pre-configured assistant template to auto-fill your model,
          prompt, and parameters. You can customise everything after selecting.
        </p>

        {/* Category filter */}
        <div className="flex items-center gap-2 flex-wrap mb-4">
          <ContentSwitcher
            onChange={({ name }) => {
              setActiveCategory(name as string);
              setSelectedTemplate(null);
            }}
            selectedIndex={categories.indexOf(activeCategory)}
            size="sm"
          >
            {categories.map(cat => (
              <Switch key={cat} name={cat} text={cat} />
            ))}
          </ContentSwitcher>
          {selectedTemplate && (
            <span className="ml-auto text-xs text-gray-500 dark:text-gray-400">
              Selected:{' '}
              <span className="font-medium text-gray-900 dark:text-gray-100">
                {selectedTemplate.name}
              </span>
            </span>
          )}
        </div>

        {/* Tile grid */}
        <div className="grid grid-cols-3 border-l border-t border-gray-200 dark:border-gray-800">
          {visible.map((template, index) => {
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

                <Tag size="sm" type="blue" className="!self-start !mb-2">
                  {template.category}
                </Tag>

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
