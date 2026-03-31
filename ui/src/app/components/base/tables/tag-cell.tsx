import { TableCell } from '@/app/components/base/tables/table-cell';
import { Tag } from '@carbon/react';
import React from 'react';

export function TagCell(props: { tags?: string[] }) {
  return (
    <TableCell>
      {props.tags && props.tags.length > 0 ? (
        <div className="flex flex-wrap gap-1">
          {props.tags.map((tag, i) => (
            <Tag key={i} type="cool-gray" size="sm">
              {tag}
            </Tag>
          ))}
        </div>
      ) : (
        <span className="text-xs text-gray-400">no tags</span>
      )}
    </TableCell>
  );
}
