import React, { useCallback, useContext, useState } from 'react';
import { UpdateProject } from '@rapidaai/react';
import { Project, UpdateProjectResponse } from '@rapidaai/react';
import toast from 'react-hot-toast/headless';
import { useForm } from 'react-hook-form';
import { useCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/hooks';
import { ErrorMessage } from '@/app/components/form/error-message';
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

interface UpdateProjectDialogProps extends ModalProps {
  afterUpdateProject: () => void;
  existingProject: Project.AsObject;
}

export const UpdateProjectDialog = (props: UpdateProjectDialogProps) => {
  const { register, handleSubmit } = useForm();
  const [project, setProject] = useState<Partial<Project.AsObject>>(
    props.existingProject,
  );
  const [error, setError] = useState<string>();
  const [userId, token] = useCredential();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const { authorize } = useContext(AuthContext);

  const afterUpdateProject = useCallback(
    (err: ServiceError | null, upr: UpdateProjectResponse | null) => {
      if (err) {
        hideLoader();
        toast.error('Unable to process your request. please try again later.');
        setError('Unable to process your request. please try again later.');
        return;
      }
      if (upr?.getSuccess()) {
        if (authorize)
          authorize(
            () => {
              toast.success('Your project has been updated successfully.');
              props.setModalOpen(false);
              props.afterUpdateProject();
            },
            err => {
              props.setModalOpen(false);
            },
          );
      } else {
        hideLoader();
        let errorMessage = upr?.getError();
        if (errorMessage) {
          toast.error(errorMessage.getHumanmessage());
          setError(errorMessage.getHumanmessage());
        } else {
          setError('Unable to process your request. please try again later.');
          toast.error(
            'Unable to process your request. please try again later.',
          );
        }
        return;
      }
    },
    [],
  );

  const onUpdateProject = data => {
    showLoader();
    UpdateProject(
      connectionConfig,
      props.existingProject.id,
      afterUpdateProject,
      {
        authorization: token,
        'x-auth-id': userId,
      },
      data.projectName,
      data.projectDescription,
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
        title="Update the project"
        onClose={() => props.setModalOpen(false)}
      />
      <Form onSubmit={handleSubmit(onUpdateProject)}>
        <input
          {...register('projectId', { value: project?.id })}
          type="hidden"
        />
        <ModalBody hasForm>
          <Stack gap={6}>
            <TextInput
              id="projectName"
              labelText="Project Name"
              placeholder="eg: your favorite project"
              required
              value={project?.name || ''}
              {...register('projectName')}
              onChange={e => {
                setProject({ ...project, name: e.target.value });
              }}
            />
            <TextArea
              id="projectDescription"
              labelText="Project Description"
              placeholder="A description of what this project is about..."
              rows={3}
              required
              value={project?.description || ''}
              {...register('projectDescription')}
              onChange={e => {
                setProject({ ...project, description: e.target.value });
              }}
            />
            <ErrorMessage message={error} />
          </Stack>
        </ModalBody>
        <ModalFooter>
          <SecondaryButton size="lg" onClick={() => props.setModalOpen(false)}>
            Cancel
          </SecondaryButton>
          <PrimaryButton size="lg" type="submit" isLoading={loading}>
            Update
          </PrimaryButton>
        </ModalFooter>
      </Form>
    </Modal>
  );
};
