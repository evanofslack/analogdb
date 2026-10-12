"use client";

import { loadCatalogOptions, updatePost } from "@app/actions/admin";
import { EditablePost, PostPatch } from "@lib/adminClient";
import { CameraOption } from "@lib/data/cameras";
import { FilmOption } from "@lib/data/films";
import {
  Alert,
  Autocomplete,
  Button,
  Group,
  Modal,
  NumberInput,
  SimpleGrid,
  Stack,
  Switch,
  Textarea,
  TextInput,
} from "@mantine/core";
import React, { useEffect, useMemo, useState } from "react";

const textKeys = [
  "description",
  "camera_make",
  "camera_model",
  "film_make",
  "film_type",
  "aperture",
] as const;
const numberKeys = ["film_speed", "focal_length"] as const;
const boolKeys = ["nsfw", "grayscale", "sprocket"] as const;

const numberLabels = { film_speed: "Film speed", focal_length: "Focal length" };

type TextKey = (typeof textKeys)[number];
type NumberKey = (typeof numberKeys)[number];
type BoolKey = (typeof boolKeys)[number];

type EditorValues = Record<TextKey, string> &
  Record<NumberKey, number | string> &
  Record<BoolKey, boolean>;

type EditorPatch = PostPatch & Record<string, unknown>;

type EditorPost = Partial<EditablePost> & Pick<EditablePost, "id">;

function initialValues(post: EditorPost): EditorValues {
  const values = {} as EditorValues;
  for (const key of textKeys) values[key] = post[key] ?? "";
  for (const key of numberKeys) values[key] = post[key] ?? "";
  for (const key of boolKeys) values[key] = !!post[key];
  return values;
}

function unique(values: string[]): string[] {
  return Array.from(new Set(values.filter(Boolean))).sort();
}

// diff returns only changed fields, or a string error
function diff(
  initial: EditorValues,
  values: EditorValues
): EditorPatch | string {
  const patch: EditorPatch = {};
  for (const key of textKeys) {
    const value = values[key].trim();
    if (value !== initial[key].trim()) patch[key] = value;
  }
  for (const key of numberKeys) {
    const value = values[key];
    if (value === initial[key]) continue;
    if (value === "") {
      return `${numberLabels[key]} can't be cleared, only changed`;
    }
    patch[key] = Number(value);
  }
  for (const key of boolKeys) {
    if (values[key] !== initial[key]) patch[key] = values[key];
  }
  return patch;
}

interface PostEditorProps {
  post: EditorPost;
  opened: boolean;
  onClose: () => void;
  onSaved?: (patch: PostPatch) => void;
}

export default function PostEditor({
  post,
  opened,
  onClose,
  onSaved,
}: PostEditorProps) {
  const initial = useMemo(() => initialValues(post), [post]);
  const [values, setValues] = useState(initial);
  const [cameras, setCameras] = useState<CameraOption[]>([]);
  const [films, setFilms] = useState<FilmOption[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!opened) return;
    setValues(initial);
    setError(null);
    loadCatalogOptions().then((options) => {
      setCameras(options.cameras);
      setFilms(options.films);
    });
  }, [opened, initial]);

  const set = (key: keyof EditorValues) => (value: string | number | boolean) =>
    setValues((current) => ({ ...current, [key]: value }));
  const setText =
    (key: TextKey) =>
    (event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
      set(key)(event.currentTarget.value);
  const setBool =
    (key: BoolKey) => (event: React.ChangeEvent<HTMLInputElement>) =>
      set(key)(event.currentTarget.checked);

  const cameraMakes = unique(cameras.map((c) => c.make));
  const cameraModels = unique(
    cameras
      .filter((c) => !values.camera_make || c.make === values.camera_make)
      .map((c) => c.model)
  );
  const filmMakes = unique(films.map((f) => f.make));
  const filmTypes = unique(
    films
      .filter((f) => !values.film_make || f.make === values.film_make)
      .map((f) => f.type)
  );

  async function save(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const patch = diff(initial, values);
    if (typeof patch === "string") {
      setError(patch);
      return;
    }
    if (Object.keys(patch).length === 0) {
      onClose();
      return;
    }
    setSaving(true);
    setError(null);
    const result = await updatePost(post.id, patch);
    setSaving(false);
    if (result.ok === false) {
      setError(result.error);
      return;
    }
    onSaved?.(patch);
    onClose();
  }

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={`Edit post ${post.id}`}
      size="lg"
    >
      <form onSubmit={save}>
        <Stack gap="md">
          <Textarea
            label="Description"
            autosize
            minRows={2}
            maxRows={8}
            value={values.description}
            onChange={setText("description")}
          />
          <Group gap="lg">
            <Switch
              label="NSFW"
              checked={values.nsfw}
              onChange={setBool("nsfw")}
            />
            <Switch
              label="Grayscale"
              checked={values.grayscale}
              onChange={setBool("grayscale")}
            />
            <Switch
              label="Sprocket"
              checked={values.sprocket}
              onChange={setBool("sprocket")}
            />
          </Group>
          <SimpleGrid cols={{ base: 1, xs: 2 }}>
            <Autocomplete
              label="Camera make"
              data={cameraMakes}
              limit={20}
              value={values.camera_make}
              onChange={set("camera_make")}
            />
            <Autocomplete
              label="Camera model"
              data={cameraModels}
              limit={20}
              value={values.camera_model}
              onChange={set("camera_model")}
            />
            <Autocomplete
              label="Film make"
              data={filmMakes}
              limit={20}
              value={values.film_make}
              onChange={set("film_make")}
            />
            <Autocomplete
              label="Film type"
              data={filmTypes}
              limit={20}
              value={values.film_type}
              onChange={set("film_type")}
            />
            <NumberInput
              label="Film speed"
              min={1}
              allowDecimal={false}
              allowNegative={false}
              value={values.film_speed}
              onChange={set("film_speed")}
            />
            <NumberInput
              label="Focal length (mm)"
              min={1}
              allowDecimal={false}
              allowNegative={false}
              value={values.focal_length}
              onChange={set("focal_length")}
            />
            <TextInput
              label="Aperture"
              placeholder="f/2.8"
              value={values.aperture}
              onChange={setText("aperture")}
            />
          </SimpleGrid>
          {error && (
            <Alert color="red" variant="light">
              {error}
            </Alert>
          )}
          <Group justify="flex-end">
            <Button variant="default" onClick={onClose} disabled={saving}>
              Cancel
            </Button>
            <Button type="submit" loading={saving}>
              Save
            </Button>
          </Group>
        </Stack>
      </form>
    </Modal>
  );
}
