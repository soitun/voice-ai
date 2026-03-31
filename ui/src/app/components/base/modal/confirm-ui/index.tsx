import type { FC } from 'react';
import React from 'react';
import { Modal } from '@carbon/react';

export type ConfirmDialogProps = {
  showing: boolean;
  type: 'info' | 'warning';
  title: string;
  content: string;
  confirmText?: string;
  onConfirm: () => void;
  cancelText?: string;
  onCancel: () => void;
  onClose: () => void;
};

const ConfirmDialog: FC<ConfirmDialogProps> = ({
  showing,
  type,
  title,
  content,
  confirmText = 'Confirm',
  cancelText = 'Cancel',
  onClose,
  onConfirm,
  onCancel,
}) => {
  return (
    <Modal
      danger={type === 'warning'}
      open={showing}
      modalHeading={title}
      modalLabel={type === 'warning' ? 'Warning' : 'Confirm'}
      primaryButtonText={confirmText}
      secondaryButtonText={cancelText}
      onRequestSubmit={onConfirm}
      onRequestClose={onClose}
      onSecondarySubmit={onCancel}
      size="xs"
    >
      <p className="text-sm">{content}</p>
    </Modal>
  );
};
export default React.memo(ConfirmDialog);
