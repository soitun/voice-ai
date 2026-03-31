import { useState } from 'react';
import { UpdateProjectDialog } from '@/app/components/base/modal/update-project-modal';
import { Project } from '@rapidaai/react';
import {
  OverflowMenu,
  OverflowMenuItem,
} from '@/app/components/carbon/overflow-menu';

export const ProjectOption = (props: {
  project: Project.AsObject;
  afterUpdateProject: () => void;
  onDelete: () => void;
}) => {
  const [projectUpdateModalOpen, setProjectUpdateModalOpen] =
    useState<boolean>(false);
  const [menuOpen, setMenuOpen] = useState(false);

  return (
    <>
      <UpdateProjectDialog
        existingProject={props.project}
        modalOpen={projectUpdateModalOpen}
        setModalOpen={setProjectUpdateModalOpen}
        afterUpdateProject={props.afterUpdateProject}
      />
      <OverflowMenu
        size="md"
        flipped
        iconDescription="Project actions"
        open={menuOpen}
        onOpen={() => setMenuOpen(true)}
        onClose={() => setMenuOpen(false)}
      >
        <OverflowMenuItem
          itemText="Update project details"
          onClick={() => {
            setMenuOpen(false);
            setProjectUpdateModalOpen(true);
          }}
        />
        <OverflowMenuItem
          itemText="Delete project"
          isDelete
          hasDivider
          onClick={() => {
            setMenuOpen(false);
            props.onDelete();
          }}
        />
      </OverflowMenu>
    </>
  );
};
