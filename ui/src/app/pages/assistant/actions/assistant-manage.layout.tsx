import { FC, HTMLAttributes } from 'react';
import { Outlet } from 'react-router-dom';

export const AssistantManageLayout: FC<HTMLAttributes<HTMLDivElement>> = () => {
  return <Outlet />;
};
