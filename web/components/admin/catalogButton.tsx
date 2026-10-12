"use client";

import { addCatalogCamera, addCatalogFilm } from "@app/actions/admin";
import {
  Alert,
  Button,
  Group,
  Modal,
  NumberInput,
  SegmentedControl,
  Stack,
  Textarea,
  TextInput,
} from "@mantine/core";
import { useRouter } from "next/navigation";
import React, { useState } from "react";

interface CameraValues {
  make: string;
  model: string;
  description: string;
}

interface FilmValues {
  make: string;
  type: string;
  speed: number | string;
  colorType: string;
  description: string;
}

type CatalogValues = CameraValues & FilmValues;

interface CatalogButtonProps {
  kind: "camera" | "film";
  initial: CameraValues | FilmValues;
}

// CatalogButton adds an unmatched camera or film to the catalog
export default function CatalogButton({ kind, initial }: CatalogButtonProps) {
  const router = useRouter();
  const [opened, setOpened] = useState(false);
  const [values, setValues] = useState<CatalogValues>(initial as CatalogValues);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const set = (key: keyof CatalogValues) => (value: string | number) =>
    setValues((current) => ({ ...current, [key]: value }));
  const setText =
    (key: keyof CatalogValues) =>
    (event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
      set(key)(event.currentTarget.value);

  function open() {
    setValues(initial as CatalogValues);
    setError(null);
    setOpened(true);
  }

  async function save(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    setError(null);
    const result =
      kind === "camera"
        ? await addCatalogCamera(values)
        : await addCatalogFilm({ ...values, speed: Number(values.speed) });
    setSaving(false);
    if (result.ok === false) {
      setError(result.error);
      return;
    }
    setOpened(false);
    router.refresh();
  }

  return (
    <>
      <Button size="xs" variant="light" onClick={open}>
        Add
      </Button>
      <Modal
        opened={opened}
        onClose={() => setOpened(false)}
        title={kind === "camera" ? "Add camera" : "Add film"}
      >
        <form onSubmit={save}>
          <Stack gap="md">
            <TextInput
              label="Make"
              required
              value={values.make}
              onChange={setText("make")}
            />
            {kind === "camera" ? (
              <TextInput
                label="Model"
                required
                value={values.model}
                onChange={setText("model")}
              />
            ) : (
              <>
                <TextInput
                  label="Type"
                  required
                  value={values.type}
                  onChange={setText("type")}
                />
                <NumberInput
                  label="Speed"
                  required
                  min={1}
                  allowDecimal={false}
                  allowNegative={false}
                  value={values.speed}
                  onChange={set("speed")}
                />
                <SegmentedControl
                  data={[
                    { label: "Color", value: "color" },
                    { label: "Black and white", value: "bw" },
                  ]}
                  value={values.colorType}
                  onChange={set("colorType")}
                />
              </>
            )}
            <Textarea
              label="Description"
              autosize
              minRows={2}
              value={values.description}
              onChange={setText("description")}
            />
            {error && (
              <Alert color="red" variant="light">
                {error}
              </Alert>
            )}
            <Group justify="flex-end">
              <Button variant="default" onClick={() => setOpened(false)}>
                Cancel
              </Button>
              <Button type="submit" loading={saving}>
                Add to catalog
              </Button>
            </Group>
          </Stack>
        </form>
      </Modal>
    </>
  );
}
