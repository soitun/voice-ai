import {
  Debug,
  RecentlyViewed,
  Activity,
} from '@carbon/icons-react';
import type { ComponentType } from 'react';

export interface EndpointNavChild {
  key: string;
  label: string;
  tabKey: string;
}

export interface EndpointNavItem {
  key: string;
  label: string;
  icon: ComponentType<{ size?: number }>;
  tabKey: string;
  children?: EndpointNavChild[];
}

export interface EndpointNavSection {
  label: string;
  items: EndpointNavItem[];
}

export const endpointNavSections: EndpointNavSection[] = [
  {
    label: '',
    items: [
      {
        key: 'playground',
        label: 'Playground',
        icon: Debug,
        tabKey: 'overview',
      },
      {
        key: 'traces',
        label: 'Traces',
        icon: Activity,
        tabKey: 'Traces',
      },
      {
        key: 'versions',
        label: 'Versions',
        icon: RecentlyViewed,
        tabKey: 'versions',
        children: [
          { key: 'versions-list', label: 'View all', tabKey: 'versions' },
          { key: 'versions-create', label: 'Add new version', tabKey: 'create-version' },
        ],
      },
    ],
  },
];
