import React, { FC, useCallback, useEffect, useState } from 'react';
import { PrimaryButton, SecondaryButton } from '@/app/components/carbon/button';
import {
  Modal,
  ModalHeader,
  ModalBody,
  ModalFooter,
} from '@/app/components/carbon/modal';
import { ModalProps } from '@/app/components/base/modal';
import { ManualFile } from '@/app/pages/knowledge-base/action/components/datasource-uploader/manual-file';
import { KnowledgeDocument } from '@rapidaai/react';
import { useCreateKnowledgeDocumentPageStore } from '@/hooks/use-create-knowledge-document-page-store';
import { useCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/hooks/use-rapida-store';
import { Notification } from '@/app/components/carbon/notification';

interface CreateKnowledgeDocumentDialogProps extends ModalProps {
  knowledgeId: string;
  onReload: () => void;
}

export const CreateKnowledgeDocumentDialog: FC<
  CreateKnowledgeDocumentDialogProps
> = props => {
  const [errorMessage, setErrorMessage] = useState('');
  const { clear } = useCreateKnowledgeDocumentPageStore();

  useEffect(() => {
    clear();
  }, [props.knowledgeId]);

  const [userId, token, projectId] = useCredential();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const knowledgeDocumentAction = useCreateKnowledgeDocumentPageStore();

  const onSuccess = useCallback(
    (d: KnowledgeDocument[]) => {
      hideLoader();
      props.setModalOpen(false);
      props.onReload();
    },
    [props.knowledgeId],
  );

  const onError = useCallback(
    (e: string) => {
      hideLoader();
      setErrorMessage(e);
    },
    [props.knowledgeId],
  );

  const onCreateKnowledgeDocument = () => {
    showLoader('overlay');
    knowledgeDocumentAction.onCreateKnowledgeDocument(
      props.knowledgeId!,
      projectId,
      token,
      userId,
      onSuccess,
      onError,
    );
  };

  return (
    <Modal
      open={props.modalOpen}
      onClose={() => props.setModalOpen(false)}
      size="lg"
      containerClassName="!w-[1000px] !max-w-[1000px]"
      preventCloseOnClickOutside
    >
      <ModalHeader
        label="Knowledge"
        title="Add document to knowledge"
        onClose={() => props.setModalOpen(false)}
      />
      <ModalBody hasForm hasScrollingContent>
        <ManualFile />
        {errorMessage && (
          <Notification kind="error" title="Error" subtitle={errorMessage} />
        )}
      </ModalBody>
      <ModalFooter>
        <SecondaryButton
          size="lg"
          onClick={() => props.setModalOpen(false)}
        >
          Cancel
        </SecondaryButton>
        <PrimaryButton
          size="lg"
          isLoading={loading}
          onClick={onCreateKnowledgeDocument}
        >
          Create Document
        </PrimaryButton>
      </ModalFooter>
    </Modal>
  );
};
