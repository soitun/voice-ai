import type { FC } from 'react';
import React, { useState } from 'react';
import { Modal } from '@carbon/react';
import { TextInput } from '@/app/components/carbon/form';

type ConfirmDeleteDialogProps = {
  showing: boolean;
  title: string;
  content: string;
  confirmText?: string;
  objectName: string;
  onConfirm: () => void;
  cancelText?: string;
  onCancel: () => void;
  onClose: () => void;
};

export const ConfirmDeleteDialog: FC<ConfirmDeleteDialogProps> = ({
  showing,
  title,
  content,
  confirmText = 'Delete',
  cancelText = 'Cancel',
  objectName,
  onClose,
  onConfirm,
  onCancel,
}) => {
  const [inputName, setInputName] = useState('');

  const handleConfirm = () => {
    if (inputName === objectName) {
      onConfirm();
      setInputName('');
    }
  };

  return (
    <Modal
      danger
      open={showing}
      modalHeading={title}
      modalLabel="Confirm action"
      primaryButtonText={confirmText}
      secondaryButtonText={cancelText}
      primaryButtonDisabled={inputName !== objectName}
      onRequestSubmit={handleConfirm}
      onRequestClose={() => {
        setInputName('');
        onClose();
      }}
      onSecondarySubmit={() => {
        setInputName('');
        onCancel();
      }}
      size="sm"
    >
      <p className="text-sm mb-4">{content}</p>
      <TextInput
        id="confirm-delete-input"
        labelText={`Type "${objectName}" to confirm`}
        value={inputName}
        onChange={e => setInputName(e.target.value)}
        placeholder={objectName}
      />
    </Modal>
  );
};
