import React, { FC, useState } from 'react';
import { DismissibleTag, Tag } from '@carbon/react';
import { TextInput } from '@/app/components/carbon/form';

interface TagInputProps {
  tags: string[];
  addTag: (tag: string) => void;
  removeTag: (tag: string) => void;
  allTags: Array<string>;
  className?: string;
}

export const TagInput: FC<TagInputProps> = ({
  tags,
  addTag,
  removeTag,
  allTags,
}) => {
  const [inputValue, setInputValue] = useState('');

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      const value = inputValue.trim();
      if (value && !tags.includes(value)) {
        addTag(value);
        setInputValue('');
      }
    }
  };

  const suggestedTags = allTags.filter(t => !tags.includes(t));

  return (
    <div>
      <TextInput
        id="tag-input"
        labelText="Tags (Optional)"
        placeholder="Type a tag and press Enter"
        value={inputValue}
        onChange={e => setInputValue(e.target.value)}
        onKeyDown={handleKeyDown}
        helperText="Add tags to organize and locate items more efficiently."
      />
      {tags.length > 0 && (
        <div className="flex flex-wrap gap-1 mt-3">
          {tags.map(t => (
            <DismissibleTag
              key={t}
              text={t}
              type="blue"
              size="md"
              onClose={() => removeTag(t)}
            />
          ))}
        </div>
      )}
      {suggestedTags.length > 0 && (
        <div className="mt-3">
          <p className="text-[10px] font-medium uppercase tracking-wider text-gray-500 dark:text-gray-400 mb-1.5">
            Suggestions
          </p>
          <div className="flex flex-wrap gap-1">
            {suggestedTags.slice(0, 12).map(t => (
              <Tag
                key={t}
                type="outline"
                size="md"
                className="cursor-pointer"
                onClick={() => addTag(t)}
              >
                {t}
              </Tag>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};
