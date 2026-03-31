import type { FC, ReactNode } from 'react';
import { Link } from '@carbon/react';
import { Launch } from '@carbon/icons-react';

export interface TableLinkProps {
  href: string;
  children: ReactNode;
}

export const TableLink: FC<TableLinkProps> = ({ href, children }) => (
  <Link href={href} className="!text-xs !inline-flex !items-center !gap-1">
    <span>{children}</span>
    <Launch size={12} />
  </Link>
);
