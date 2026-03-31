import React from 'react';
import {
  OverflowMenu,
  OverflowMenuItem,
} from '@/app/components/carbon/overflow-menu';

export function UserOption(props: { id: string }) {
  return (
    <OverflowMenu size="md" flipped iconDescription="User actions">
      <OverflowMenuItem
        itemText="Edit user"
        onClick={() => {}}
      />
      <OverflowMenuItem
        itemText="Delete"
        isDelete
        hasDivider
        onClick={() => {}}
      />
    </OverflowMenu>
  );
}
