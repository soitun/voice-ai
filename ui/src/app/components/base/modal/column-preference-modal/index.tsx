import React, { useEffect, useState } from 'react';
import { ErrorMessage } from '@/app/components/form/error-message';
import {
  Modal,
  ModalHeader,
  ModalBody,
  ModalFooter,
} from '@/app/components/carbon/modal';
import {
  PrimaryButton,
  SecondaryButton,
} from '@/app/components/carbon/button';
import { Stack, Checkbox, FormGroup } from '@/app/components/carbon/form';
import { RadioButton, RadioButtonGroup } from '@carbon/react';

interface TablePreferenceModalProps {
  open: boolean;
  setOpen: (open: boolean) => void;
  defaultPageSize: number[];
  columns: { name: string; key: string; visible: boolean }[];
  onChangeColumns: (
    clmns: { name: string; key: string; visible: boolean }[],
  ) => void;
  pageSize: number;
  onChangePageSize: (size: number) => void;
}

export function ColumnPreferencesDialog(props: TablePreferenceModalProps) {
  const [pgs, setPgs] = useState(props.pageSize);
  const [clmns, setClmns] = useState<
    { name: string; key: string; visible: boolean }[]
  >([]);
  const [error, setError] = useState('');

  useEffect(() => {
    setPgs(props.pageSize);
  }, [props.pageSize]);

  useEffect(() => {
    setClmns(props.columns);
  }, [props.columns]);

  const changeVisibility = (k: string) => {
    setClmns(prevClmns =>
      prevClmns.map(column =>
        column.key === k ? { ...column, visible: !column.visible } : column,
      ),
    );
  };

  const onAction = () => {
    const cnt = clmns.filter(x => x.visible);
    if (cnt.length < 1 && clmns.length > 0) {
      setError('Please have 2 or more column visibility selected');
      return;
    }
    props.onChangePageSize(pgs);
    props.onChangeColumns(clmns);
    props.setOpen(false);
  };

  return (
    <Modal
      open={props.open}
      onClose={() => props.setOpen(false)}
      size="sm"
    >
      <ModalHeader
        label="Table Settings"
        title="Column Preferences"
        onClose={() => props.setOpen(false)}
      />
      <ModalBody>
        <Stack gap={6}>
          {clmns.length > 0 && (
            <FormGroup legendText="Visible Columns">
              <Stack gap={3}>
                {clmns.map((cl, idx) => (
                  <Checkbox
                    key={cl.key}
                    id={`col-pref-${cl.key}`}
                    labelText={cl.name}
                    checked={cl.visible}
                    onChange={() => changeVisibility(cl.key)}
                  />
                ))}
              </Stack>
            </FormGroup>
          )}
          <FormGroup legendText="Page Size">
            <RadioButtonGroup
              name="page-size"
              valueSelected={String(pgs)}
              onChange={(value: string) => setPgs(Number(value))}
              orientation="vertical"
            >
              {props.defaultPageSize.map(sz => (
                <RadioButton
                  key={sz}
                  id={`page-size-${sz}`}
                  value={String(sz)}
                  labelText={`${sz} Items`}
                />
              ))}
            </RadioButtonGroup>
          </FormGroup>
          <ErrorMessage message={error} />
        </Stack>
      </ModalBody>
      <ModalFooter>
        <SecondaryButton size="lg" onClick={() => props.setOpen(false)}>
          Cancel
        </SecondaryButton>
        <PrimaryButton size="lg" onClick={onAction}>
          Save Preference
        </PrimaryButton>
      </ModalFooter>
    </Modal>
  );
}
