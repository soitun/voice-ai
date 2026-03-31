import { Tag } from '@rapidaai/react';
import { Modal, ModalHeader, ModalBody, ModalFooter } from '@/app/components/carbon/modal';
import { PrimaryButton, SecondaryButton } from '@/app/components/carbon/button';
import { Notification } from '@/app/components/carbon/notification';
import { TagInput } from '@/app/components/form/tag-input';
import { KnowledgeTags } from '@/app/components/form/tag-input/knowledge-tags';
import { ModalProps } from '@/app/components/base/modal';
import { useRapidaStore } from '@/hooks';
import React, { FC, memo, useEffect, useState } from 'react';

interface CreateTagDialogProps extends ModalProps {
  title: string;
  tags?: string[];
  allTags?: string[];
  onCreateTag: (
    tags: string[],
    onError: (err: string) => void,
    onSuccess: (e: Tag) => void,
  ) => void;
}

export const CreateTagDialog: FC<CreateTagDialogProps> = memo(
  ({ title, tags, allTags, onCreateTag, setModalOpen, modalOpen }) => {
    const [error, setError] = useState('');
    const [_tags, _setTags] = useState<string[]>([]);
    const rapidaStore = useRapidaStore();

    const addTag = (tag: string) => {
      _setTags([..._tags, tag]);
    };

    const removeTag = (index: number) => {
      const newTags = [..._tags];
      newTags.splice(index, 1);
      _setTags(newTags);
    };

    useEffect(() => {
      if (tags) _setTags(tags);
    }, [tags]);

    const createTag = () => {
      rapidaStore.showLoader('overlay');
      onCreateTag(
        _tags,
        (err: string) => {
          rapidaStore.hideLoader();
          setError(err);
        },
        (_rc: Tag) => {
          rapidaStore.hideLoader();
          setModalOpen(false);
        },
      );
    };

    return (
      <Modal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        size="sm"
        preventCloseOnClickOutside
      >
        <ModalHeader
          label="Tags"
          title={title}
          onClose={() => setModalOpen(false)}
        />
        <ModalBody hasForm>
          <TagInput
            tags={_tags}
            addTag={addTag}
            removeTag={removeTag}
            allTags={allTags ?? KnowledgeTags}
          />
          {error && (
            <Notification kind="error" title="Error" subtitle={error} />
          )}
        </ModalBody>
        <ModalFooter>
          <SecondaryButton size="lg" onClick={() => setModalOpen(false)}>
            Cancel
          </SecondaryButton>
          <PrimaryButton
            size="lg"
            onClick={createTag}
            isLoading={rapidaStore.loading}
          >
            Save tags
          </PrimaryButton>
        </ModalFooter>
      </Modal>
    );
  },
);
