import { PrimaryButton, SecondaryButton } from '@/app/components/carbon/button';
import { ModalProps } from '@/app/components/base/modal';
import { Modal, ModalHeader, ModalBody, ModalFooter } from '@/app/components/carbon/modal';
import { Notification } from '@/app/components/carbon/notification';
import { Stack, TextInput, TextArea } from '@/app/components/carbon/form';
import { useRapidaStore } from '@/hooks';
import React, { useEffect, useState } from 'react';

interface UpdateDescriptionDialogProps extends ModalProps {
  title?: string;
  name?: string;
  description?: string;
  onUpdateDescription: (
    name: string,
    description: string,
    onError: (err: string) => void,
    onSuccess: () => void,
  ) => void;
}

export function UpdateDescriptionDialog(props: UpdateDescriptionDialogProps) {
  const [error, setError] = useState('');
  const [name, setName] = useState<string>('');
  const [description, setDescription] = useState<string>('');
  const rapidaStore = useRapidaStore();

  useEffect(() => {
    if (props.name) setName(props.name);
    if (props.description) setDescription(props.description);
  }, [props.name, props.description]);

  const onUpdateDescription = () => {
    rapidaStore.showLoader('overlay');
    props.onUpdateDescription(
      name,
      description,
      err => {
        rapidaStore.hideLoader();
        setError(err);
      },
      () => {
        rapidaStore.hideLoader();
        props.setModalOpen(false);
      },
    );
  };

  return (
    <Modal
      open={props.modalOpen}
      onClose={() => props.setModalOpen(false)}
      size="sm"
      selectorPrimaryFocus="#edit-name"
    >
      <ModalHeader
        label="Details"
        title={props.title || 'Edit details'}
        onClose={() => props.setModalOpen(false)}
      />
      <ModalBody hasForm>
        <Stack gap={6}>
          <TextInput
            id="edit-name"
            labelText="Name"
            value={name}
            placeholder="e.g. emotion detector"
            onChange={e => setName(e.target.value)}
          />
          <TextArea
            id="edit-description"
            labelText="Description"
            rows={4}
            value={description}
            placeholder="Provide a readable description and how to use it."
            onChange={e => setDescription(e.target.value)}
          />
          {error && (
            <Notification kind="error" title="Error" subtitle={error} />
          )}
        </Stack>
      </ModalBody>
      <ModalFooter>
        <SecondaryButton size="lg" onClick={() => props.setModalOpen(false)}>
          Cancel
        </SecondaryButton>
        <PrimaryButton
          size="lg"
          onClick={onUpdateDescription}
          isLoading={rapidaStore.loading}
        >
          Save changes
        </PrimaryButton>
      </ModalFooter>
    </Modal>
  );
}
