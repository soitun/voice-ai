import { PrimaryButton, SecondaryButton } from '@/app/components/carbon/button';
import { ModalProps } from '@/app/components/base/modal';
import { Modal, ModalHeader, ModalBody, ModalFooter } from '@/app/components/carbon/modal';
import type { FC } from 'react';
import { ExternalLink } from 'lucide-react';

interface DebuggerDeploymentSuccessDialogProps extends ModalProps {
  assistantId: string;
}

export const DebuggerDeploymentSuccessDialog: FC<
  DebuggerDeploymentSuccessDialogProps
> = ({ modalOpen, setModalOpen, assistantId }) => {
  return (
    <Modal open={modalOpen} onClose={() => setModalOpen(false)} size="sm">
      <ModalHeader title="Deployment completed" onClose={() => setModalOpen(false)} />
      <ModalBody>
        <p className="text-sm text-gray-600 dark:text-gray-400 leading-relaxed">
          Your debugger is ready. Use the preview to test your assistant in a
          sandbox environment before deploying to other channels.
        </p>
      </ModalBody>
      <ModalFooter>
        <SecondaryButton size="lg" onClick={() => setModalOpen(false)}>
          Close
        </SecondaryButton>
        <PrimaryButton
          size="lg"
          type="button"
          onClick={() =>
            window.open(`/preview/chat/${assistantId}`, '_blank')
          }
        >
          <span>Preview assistant</span>
          <ExternalLink className="w-4 h-4 ml-1" strokeWidth={1.5} />
        </PrimaryButton>
      </ModalFooter>
    </Modal>
  );
};
