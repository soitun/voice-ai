import React, { useCallback, useContext, useState } from 'react';
import { AddUsersToProject } from '@rapidaai/react';
import { AddUsersToProjectResponse } from '@rapidaai/react';
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
import { Stack, TextInput } from '@/app/components/carbon/form';
import { Dropdown, MultiSelect } from '@carbon/react';
import { connectionConfig } from '@/configs';

const roles = ['super admin', 'admin', 'writer', 'reader'];

interface InviteUserDialogProps extends ModalProps {
  projectId?: string;
  onSuccess?: () => void;
}

export function InviteUserDialog(props: InviteUserDialogProps) {
  const [projects, setProjects] = useState<string[]>([]);
  const [projectRole, setProjectRole] = useState<string>('');
  const [email, setEmail] = useState<string>('');
  const [error, setError] = useState<string>('');
  const { authId, token } = useCurrentCredential();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const { projectRoles } = useContext(AuthContext);

  const afterAddToProject = useCallback(
    (err: ServiceError | null, aur: AddUsersToProjectResponse | null) => {
      hideLoader();
      if (err) {
        toast.error('Unable to process your request. please try again later.');
        setError('Unable to process your request. please try again later.');
        return;
      }
      if (aur?.getSuccess()) {
        setEmail('');
        setProjectRole('');
        setProjects([]);
        props.setModalOpen(false);
        toast.success(
          'The invitation of joining the projects are successfully sent to the user.',
        );
        if (props.onSuccess) props.onSuccess();
      } else {
        toast.error('Unable to process your request. please try again later.');
        setError('Unable to process your request. please try again later.');
        return;
      }
    },
    [],
  );

  const addUserToProject = () => {
    showLoader('overlay');
    if (projectRole === '') {
      setError('Please select a role for the user.');
      return;
    }
    if (email === '') {
      setError('Please provide a valid email to invite user.');
      return;
    }
    if (!props.projectId && projects.length === 0) {
      setError('Please select one or more projects for the user to invite.');
      return;
    }

    AddUsersToProject(
      connectionConfig,
      email,
      projectRole,
      props.projectId ? [props?.projectId] : projects,
      afterAddToProject,
      {
        authorization: token,
        'x-auth-id': authId,
      },
    );
  };

  return (
    <Modal
      open={props.modalOpen}
      onClose={() => props.setModalOpen(false)}
      size="sm"
      selectorPrimaryFocus="#invite-email"
      preventCloseOnClickOutside
    >
      <ModalHeader
        label="User Management"
        title="Invite new user"
        onClose={() => props.setModalOpen(false)}
      />
      <ModalBody hasForm>
        <Stack gap={6}>
          <Dropdown
            id="invite-role"
            titleText="Project Role"
            label="Select a project role"
            items={roles}
            selectedItem={projectRole || null}
            itemToString={(item: string | null) =>
              item ? item.charAt(0).toUpperCase() + item.slice(1) : ''
            }
            onChange={({ selectedItem }) => {
              if (selectedItem) setProjectRole(selectedItem);
            }}
          />
          <TextInput
            id="invite-email"
            labelText="Email Address"
            value={email}
            type="email"
            placeholder="eg: john@deo.io"
            onChange={e => {
              setError('');
              setEmail(e.target.value);
            }}
          />
          {!props.projectId && projectRoles && (
            <MultiSelect
              id="invite-projects"
              titleText="Projects"
              label="Select projects"
              items={projectRoles}
              selectedItems={projectRoles.filter(p =>
                projects.includes(p.projectid),
              )}
              itemToString={(item: any) => item?.projectname || ''}
              onChange={({ selectedItems }) => {
                setProjects(
                  (selectedItems || []).map((p: any) => p.projectid),
                );
              }}
            />
          )}
          <ErrorMessage message={error} />
        </Stack>
      </ModalBody>
      <ModalFooter>
        <SecondaryButton size="lg" onClick={() => props.setModalOpen(false)}>
          Cancel
        </SecondaryButton>
        <PrimaryButton size="lg" onClick={addUserToProject} isLoading={loading}>
          Invite User
        </PrimaryButton>
      </ModalFooter>
    </Modal>
  );
}
