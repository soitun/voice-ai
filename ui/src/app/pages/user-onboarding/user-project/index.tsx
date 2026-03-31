import React, { useCallback, useContext, useState } from 'react';
import { Helmet } from '@/app/components/helmet';
import { useNavigate } from 'react-router-dom';
import { useForm } from 'react-hook-form';
import { CreateProject } from '@rapidaai/react';
import { CreateProjectResponse } from '@rapidaai/react';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/hooks';
import { ServiceError } from '@rapidaai/react';
import { AuthContext } from '@/context/auth-context';
import { connectionConfig } from '@/configs';
import { Stack, TextInput, TextArea } from '@/app/components/carbon/form';
import { PrimaryButton } from '@/app/components/carbon/button';
import { Notification } from '@/app/components/carbon/notification';
import { ArrowRight } from '@carbon/icons-react';

export function CreateProjectPage() {
  const navigate = useNavigate();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const { authorize } = useContext(AuthContext);
  const { authId, token, user } = useCurrentCredential();
  const { register, handleSubmit } = useForm();
  const [error, setError] = useState('');

  const afterCreateProject = useCallback(
    async (err: ServiceError | null, cpr: CreateProjectResponse | null) => {
      hideLoader();
      if (err) {
        setError('Unable to process your request. Please try again later.');
        return;
      }
      if (cpr?.getSuccess()) {
        authorize &&
          authorize(
            () => navigate('/dashboard'),
            () => setError('Unable to create project. Please check the details.'),
          );
      } else {
        setError('Unable to create project. Please check the details.');
      }
    },
    [],
  );

  const onCreateProject = data => {
    showLoader('overlay');
    CreateProject(
      connectionConfig,
      data.projectName,
      data.projectDescription,
      { authorization: token, 'x-auth-id': authId },
      afterCreateProject,
    );
  };

  return (
    <>
      <Helmet title="Onboarding: Create a Project" />
      <div className="mb-4">
        <h1 className="text-xl font-light tracking-tight">Create your first project</h1>
      </div>

      <form onSubmit={handleSubmit(onCreateProject)}>
        <Stack gap={5}>
          <TextInput
            id="project-name"
            labelText="Project Name"
            type="text"
            required
            defaultValue={`${user?.name}'s Workspace`}
            placeholder="eg: Customer Support Bot"
            helperText="Choose a name that reflects the purpose or team for this project."
            {...register('projectName')}
          />
          <TextArea
            id="project-description"
            labelText="Project Description"
            rows={3}
            placeholder="eg: Voice assistant for handling customer inquiries 24/7"
            helperText="Optional — helps your team understand the project's goals at a glance."
            {...register('projectDescription')}
          />
          {error && (
            <Notification kind="error" title="Error" subtitle={error} />
          )}
          <PrimaryButton
            size="lg"
            renderIcon={ArrowRight}
            type="submit"
            isLoading={loading}
            className="!w-full !max-w-none !justify-between"
          >
            Go to dashboard
          </PrimaryButton>
        </Stack>
      </form>
    </>
  );
}
