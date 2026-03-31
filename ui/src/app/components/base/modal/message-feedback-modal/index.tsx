import { ModalProps } from '@/app/components/base/modal';
import { Modal, ModalHeader, ModalBody, ModalFooter } from '@/app/components/carbon/modal';
import { PrimaryButton, SecondaryButton } from '@/app/components/carbon/button';
import { Textarea } from '@/app/components/form/textarea';
import { Check } from 'lucide-react';
import { FC, useState } from 'react';

export const MessageFeedbackDialog: FC<
  ModalProps & { onSubmitFeedback: (feedback: string) => void }
> = props => {
  const [feedbackText, setFeedbackText] = useState('');
  return (
    <Modal open={props.modalOpen} onClose={() => props.setModalOpen(false)} size="sm">
      <ModalHeader
        title="What can be improved?"
        onClose={() => {
          props.setModalOpen(false);
        }}
      />
      <ModalBody hasForm>
        <div className="px-4 py-6">
          <p className="font-semibold text-base mt-1">
            Tell us what went wrong or how we can make this answer more
            helpful.
          </p>
          <div className="mt-4">
            <Textarea
              required
              rows={3}
              placeholder="Your feedback..."
              value={feedbackText}
              onChange={e => setFeedbackText(e.target.value)}
            />
          </div>
        </div>
      </ModalBody>
      <ModalFooter>
        <SecondaryButton size="lg" onClick={() => props.setModalOpen(false)}>
          Cancel
        </SecondaryButton>
        <PrimaryButton
          size="lg"
          type="button"
          onClick={() => {
            props.setModalOpen(false);
            props.onSubmitFeedback(feedbackText);
          }}
          disabled={!feedbackText.trim()}
        >
          Submit feedback
          <Check className="ml-2" strokeWidth={1.5} />
        </PrimaryButton>
      </ModalFooter>
    </Modal>
  );
};
