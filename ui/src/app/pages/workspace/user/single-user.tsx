import React from 'react';
import { UserOption } from './user-options';
import { TextImage } from '@/app/components/text-image';
import { User } from '@rapidaai/react';
import { RoleIndicator } from '@/app/components/indicators/role';
import { toHumanReadableDate } from '@/utils/date';
import { TableRow, TableCell } from '@carbon/react';
import IconIndicator from '@carbon/react/es/components/IconIndicator';

const statusMap: Record<string, { kind: string; label: string }> = {
  ACTIVE: { kind: 'succeeded', label: 'Active' },
  active: { kind: 'succeeded', label: 'Active' },
  DISABLED: { kind: 'failed', label: 'Disabled' },
  disabled: { kind: 'failed', label: 'Disabled' },
  INACTIVE: { kind: 'incomplete', label: 'Inactive' },
  inactive: { kind: 'incomplete', label: 'Inactive' },
  PENDING: { kind: 'pending', label: 'Pending' },
  pending: { kind: 'pending', label: 'Pending' },
};

const defaultStatus = { kind: 'normal', label: 'Active' };

export function SingleUser(props: { user: User }) {
  const status = props.user.getStatus?.() || '';
  const { kind, label } = statusMap[status] || defaultStatus;

  return (
    <TableRow>
      <TableCell>{props.user.getId()}</TableCell>
      <TableCell>
        <div className="flex items-center gap-3">
          <TextImage size={7} name={props.user.getName()} />
          <span>{props.user.getName()}</span>
        </div>
      </TableCell>
      <TableCell>{props.user.getEmail()}</TableCell>
      <TableCell>
        <RoleIndicator role={'SUPER_ADMIN'} />
      </TableCell>
      <TableCell>
        {props.user.getCreateddate() &&
          toHumanReadableDate(props.user.getCreateddate()!)}
      </TableCell>
      <TableCell>
        <IconIndicator kind={kind as any} label={label} size={16} />
      </TableCell>
      <TableCell>
        <UserOption id={props.user.getId()} />
      </TableCell>
    </TableRow>
  );
}
