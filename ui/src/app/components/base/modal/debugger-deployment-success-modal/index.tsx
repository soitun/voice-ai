import { IBlueBGButton, ICancelButton } from '@/app/components/form/button';
import { GenericModal } from '@/app/components/base/modal';
import { ModalFooter } from '@/app/components/base/modal/modal-footer';
import type { FC } from 'react';
import { ModalProps } from '../index';
import { ModalFitHeightBlock } from '@/app/components/blocks/modal-fit-height-block';
import { ModalHeader } from '@/app/components/base/modal/modal-header';
import { ModalBody } from '@/app/components/base/modal/modal-body';
import { ExternalLink } from 'lucide-react';
import { ModalTitleBlock } from '@/app/components/blocks/modal-title-block';

interface DebuggerDeploymentSuccessDialogProps extends ModalProps {
  assistantId: string;
}

export const DebuggerDeploymentSuccessDialog: FC<
  DebuggerDeploymentSuccessDialogProps
> = ({ modalOpen, setModalOpen, assistantId }) => {
  return (
    <GenericModal modalOpen={modalOpen} setModalOpen={setModalOpen}>
      <ModalFitHeightBlock>
        <ModalHeader onClose={() => setModalOpen(false)}>
          <ModalTitleBlock>Deployment completed</ModalTitleBlock>
        </ModalHeader>
        <ModalBody>
          <p className="text-sm text-gray-600 dark:text-gray-400 leading-relaxed">
            Your debugger is ready. Use the preview to test your assistant in a
            sandbox environment before deploying to other channels.
          </p>
        </ModalBody>
        <ModalFooter>
          <ICancelButton onClick={() => setModalOpen(false)}>
            Close
          </ICancelButton>
          <IBlueBGButton
            type="button"
            onClick={() =>
              window.open(`/preview/chat/${assistantId}`, '_blank')
            }
          >
            <span>Preview assistant</span>
            <ExternalLink className="w-4 h-4 ml-1" strokeWidth={1.5} />
          </IBlueBGButton>
        </ModalFooter>
      </ModalFitHeightBlock>
    </GenericModal>
  );
};
