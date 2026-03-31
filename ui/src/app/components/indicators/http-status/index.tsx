import { FC } from 'react';
import { unstable__ShapeIndicator as ShapeIndicatorModule } from '@carbon/react';

const ShapeIndicator =
  (ShapeIndicatorModule as unknown as { default?: FC<any> }).default ||
  (ShapeIndicatorModule as unknown as FC<any>);

const getStatusKind = (status: number): { kind: string; label: string } => {
  if (status >= 200 && status < 300) return { kind: 'stable', label: `${status} OK` };
  if (status >= 300 && status < 400) return { kind: 'informative', label: `${status} Redirect` };
  if (status >= 400 && status < 500) return { kind: 'cautious', label: `${status} Client Error` };
  if (status >= 500) return { kind: 'failed', label: `${status} Server Error` };
  return { kind: 'undefined', label: `${status}` };
};

export const HttpStatusSpanIndicator: FC<{
  status: number;
  textSize?: 12 | 14;
}> = ({ status, textSize = 12 }) => {
  const { kind, label } = getStatusKind(status);
  return <ShapeIndicator kind={kind as any} label={label} textSize={textSize} />;
};
