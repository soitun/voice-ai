import React, { useCallback, useContext, useState } from 'react';
import { CreateProject } from '@rapidaai/react';
import { CreateProjectResponse } from '@rapidaai/react';
import { useForm } from 'react-hook-form';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/hooks';
import { ErrorMessage } from '@/app/components/form/error-message';
import toast from 'react-hot-toast/headless';
import { ModalProps } from '@/app/components/base/modal';
import { ServiceError } from '@rapidaai/react';
import { AuthContext } from '@/context/auth-context';
import {
  Modal,
  ModalHeader,
  ModalBody,
  ModalFooter,
} from '@/app/components/carbon/modal';
import { PrimaryButton, SecondaryButton } from '@/app/components/carbon/button';
import { Form, Stack, TextInput, TextArea } from '@/app/components/carbon/form';
import { connectionConfig } from '@/configs';

interface CreateProjectDialogProps extends ModalProps {
  afterCreateProject: () => void;
}

export const CreateProjectDialog = (props: CreateProjectDialogProps) => {
  const { register, handleSubmit } = useForm();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const { authorize } = useContext(AuthContext);
  const { authId, token } = useCurrentCredential();
  const [error, setError] = useState<string>();

  const afterCreateProject = useCallback(
    async (err: ServiceError | null, cpr: CreateProjectResponse | null) => {
      if (err) {
        hideLoader();
        toast.error('Unable to process your request. please try again later.');
        setError('Unable to process your request. please try again later.');
        return;
      }
      if (cpr?.getSuccess()) {
        if (authorize)
          authorize(
            () => {
              console.log('success');
              hideLoader();
              toast.success('The project has been created successfully.');
              props.setModalOpen(false);
              props.afterCreateProject();
            },
            err => {
              console.log('failure');
              hideLoader();
            },
          );
      } else {
        hideLoader();
        let errorMessage = cpr?.getError();
        if (errorMessage) {
          toast.error(errorMessage.getHumanmessage());
          setError(errorMessage.getHumanmessage());
        } else {
          toast.error(
            'Unable to process your request. please try again later.',
          );
          setError('Unable to process your request. please try again later.');
        }
        return;
      }
    },
    [],
  );

  const onCreateProject = data => {
    showLoader();
    CreateProject(
      connectionConfig,
      data.projectName,
      data.projectDescription,
      {
        authorization: token,
        'x-auth-id': authId,
      },
      afterCreateProject,
    );
  };

  return (
    <Modal
      open={props.modalOpen}
      onClose={() => props.setModalOpen(false)}
      size="sm"
      selectorPrimaryFocus="#projectName"
    >
      <ModalHeader
        label="Project"
        title="Create a project"
        onClose={() => props.setModalOpen(false)}
      />
      <Form onSubmit={handleSubmit(onCreateProject)}>
        <ModalBody hasForm>
          <Stack gap={6}>
            <TextInput
              id="projectName"
              labelText="Project Name"
              placeholder="eg: your favorite project"
              required
              {...register('projectName')}
            />
            <TextArea
              id="projectDescription"
              labelText="Project Description"
              placeholder="An optional description of what this project about..."
              rows={3}
              required
              {...register('projectDescription')}
            />
            <ErrorMessage message={error} />
          </Stack>
        </ModalBody>
        <ModalFooter>
          <SecondaryButton size="lg" onClick={() => props.setModalOpen(false)}>
            Cancel
          </SecondaryButton>
          <PrimaryButton size="lg" type="submit" isLoading={loading}>
            Create Project
          </PrimaryButton>
        </ModalFooter>
      </Form>
    </Modal>
  );
};
