"use client";

import type { CameraOption } from "@lib/data/cameras";
import type { FilmOption } from "@lib/data/films";
import {
  ActivePostsFilters,
  PostsFilters,
  postsParsers,
  ratioShapes,
  selectedShapes,
  toggleShape,
} from "@lib/searchParams";
import {
  Button,
  Menu,
  NumberInput,
  Radio,
  SegmentedControl,
  Select,
  Stack,
  VisuallyHidden,
} from "@mantine/core";
import { useDebouncedCallback } from "@mantine/hooks";
import {
  IconAdjustmentsHorizontal,
  IconArrowAutofitWidth,
  IconArrowsSort,
  IconCamera,
  IconMovie,
} from "@tabler/icons-react";
import { useEffect, useRef, useState } from "react";
import ColorFilter from "./colorFilter";
import DisplayMenu from "./displayMenu";
import styles from "./filterBar.module.css";
import FilterSummary, { FilterSummaryItem } from "./filterSummary";

type SizeKey =
  | "widthMin"
  | "widthMax"
  | "heightMin"
  | "heightMax"
  | "ratioMin"
  | "ratioMax";
type Sizes = Pick<PostsFilters, SizeKey>;
type FlagKey = "nsfw" | "bw" | "sprocket";
type Flag = PostsFilters[FlagKey];

const orderLabels = {
  time: [
    { label: "newest", value: "desc" },
    { label: "oldest", value: "asc" },
  ],
  score: [
    { label: "highest", value: "desc" },
    { label: "lowest", value: "asc" },
  ],
};

interface FilterBarProps {
  sort: PostsFilters["sort"];
  order: PostsFilters["order"];
  nsfw: Flag;
  bw: Flag;
  sprocket: Flag;
  color: string | null;
  text: string | null;
  widthMin: number;
  widthMax: number;
  heightMin: number;
  heightMax: number;
  ratioMin: number;
  ratioMax: number;
  filmMake: string | null;
  filmType: string | null;
  cameraMake: string | null;
  cameraModel: string | null;

  setSort: (sort: PostsFilters["sort"]) => void;
  setOrder: (order: PostsFilters["order"]) => void;
  setNsfw: (nsfw: Flag) => void;
  setBw: (bw: Flag) => void;
  setSprocket: (sprocket: Flag) => void;
  setColor: (color: string | null) => void;
  setText: (text: string | null) => void;
  setSizes: (sizes: Sizes) => void;
  setFilm: (make: string | null, type: string | null) => void;
  setCamera: (make: string | null, model: string | null) => void;

  filmOptions: FilmOption[];
  cameraOptions: CameraOption[];

  textPlaceholder?: string;

  widthMinLimit: number;
  widthMaxLimit: number;
  heightMinLimit: number;
  heightMaxLimit: number;
  ratioMinLimit: number;
  ratioMaxLimit: number;

  active?: Partial<ActivePostsFilters>;
  onClear?: () => void;
  resultCount?: number | null;
  resultPending?: boolean;
  apiUrl?: string;
}

export default function FilterBar({
  // State values
  sort,
  order,
  nsfw,
  bw,
  sprocket,
  color,
  text,
  widthMin,
  widthMax,
  heightMin,
  heightMax,
  ratioMin,
  ratioMax,
  filmMake,
  filmType,
  cameraMake,
  cameraModel,

  // State setters
  setSort,
  setOrder,
  setNsfw,
  setBw,
  setSprocket,
  setColor,
  setText,
  setSizes,
  setFilm,
  setCamera,

  filmOptions,
  cameraOptions,

  // UI state
  textPlaceholder,

  // Limits
  widthMinLimit,
  widthMaxLimit,
  heightMinLimit,
  heightMaxLimit,
  ratioMinLimit,
  ratioMaxLimit,

  active = {},
  onClear,
  resultCount,
  resultPending,
  apiUrl,
}: FilterBarProps) {
  const iconSize = 18;

  const sizes: Sizes = {
    widthMin,
    widthMax,
    heightMin,
    heightMax,
    ratioMin,
    ratioMax,
  };
  const [drafts, setDrafts] = useState(sizes);
  const draftsRef = useRef(drafts);

  useEffect(() => {
    const next = {
      widthMin,
      widthMax,
      heightMin,
      heightMax,
      ratioMin,
      ratioMax,
    };
    draftsRef.current = next;
    setDrafts(next);
  }, [widthMin, widthMax, heightMin, heightMax, ratioMin, ratioMax]);

  const inRange = (value: number, min: number, max: number, integer: boolean) =>
    typeof value === "number" &&
    Number.isFinite(value) &&
    (!integer || Number.isInteger(value)) &&
    value >= min &&
    value <= max;

  const sizeErrors = (d: Sizes): Record<SizeKey, boolean> => {
    const widthOrder = !(d.widthMin <= d.widthMax);
    const heightOrder = !(d.heightMin <= d.heightMax);
    const ratioOrder = !(d.ratioMin <= d.ratioMax);
    return {
      widthMin:
        widthOrder || !inRange(d.widthMin, widthMinLimit, widthMaxLimit, true),
      widthMax:
        widthOrder || !inRange(d.widthMax, widthMinLimit, widthMaxLimit, true),
      heightMin:
        heightOrder ||
        !inRange(d.heightMin, heightMinLimit, heightMaxLimit, true),
      heightMax:
        heightOrder ||
        !inRange(d.heightMax, heightMinLimit, heightMaxLimit, true),
      ratioMin:
        ratioOrder || !inRange(d.ratioMin, ratioMinLimit, ratioMaxLimit, false),
      ratioMax:
        ratioOrder || !inRange(d.ratioMax, ratioMinLimit, ratioMaxLimit, false),
    };
  };

  const errors = sizeErrors(drafts);

  const commitSizes = useDebouncedCallback(() => {
    const d = draftsRef.current;
    if (Object.values(sizeErrors(d)).some(Boolean)) return;
    const changed = Object.keys(d).some((key) => d[key] !== sizes[key]);
    if (changed) setSizes(d);
  }, 500);

  const handleSizeChange = (key: SizeKey) => (value: number | string) => {
    const next = { ...draftsRef.current, [key]: value };
    draftsRef.current = next;
    setDrafts(next);
    commitSizes();
  };

  const shapes = selectedShapes(ratioMin, ratioMax);
  const handleShape = (index: number) => {
    const ratio = toggleShape(shapes, index);
    const next = { ...draftsRef.current, ...ratio };
    draftsRef.current = next;
    setDrafts(next);
    setSizes({ ...sizes, ...ratio });
  };

  const buttonClassNames = {
    root: styles.button,
    section: styles.buttonSection,
  };

  const activeProps = (key: keyof ActivePostsFilters) =>
    active[key] ? { "data-active": true } : {};
  const activeText = (key: keyof ActivePostsFilters) => (
    <>
      <span className={styles.dot} aria-hidden="true" />
      {active[key] && <VisuallyHidden>, filter on</VisuallyHidden>}
    </>
  );

  const optionLabel = <T extends { label: string }>(
    options: T[],
    a: string,
    b: string,
    keyA: keyof T,
    keyB: keyof T
  ) =>
    options.find((o) => o[keyA] === a && o[keyB] === b)?.label ?? `${a} ${b}`;
  const defaultSizes: Sizes = {
    widthMin: widthMinLimit,
    widthMax: widthMaxLimit,
    heightMin: heightMinLimit,
    heightMax: heightMaxLimit,
    ratioMin: ratioMinLimit,
    ratioMax: ratioMaxLimit,
  };
  const sizeLabel = () => {
    const onlyRatio = ["widthMin", "widthMax", "heightMin", "heightMax"].every(
      (key) => sizes[key] === defaultSizes[key]
    );
    if (!onlyRatio || !shapes.length) return "custom";
    const first = ratioShapes[shapes[0]].key;
    const last = ratioShapes[shapes[shapes.length - 1]].key;
    return first === last ? first : `${first} to ${last}`;
  };
  const flagItem = (
    key: FlagKey,
    title: string,
    value: Flag,
    setValue: (value: Flag) => void
  ) =>
    value !== postsParsers[key].defaultValue && {
      key,
      category: title,
      value,
      onRemove: () => setValue(postsParsers[key].defaultValue),
    };
  const summaryItems = [
    active.camera && {
      key: "camera",
      category: "camera",
      value: optionLabel(
        cameraOptions,
        cameraMake,
        cameraModel,
        "make",
        "model"
      ),
      onRemove: () => setCamera(null, null),
    },
    active.film && {
      key: "film",
      category: "film",
      value: optionLabel(filmOptions, filmMake, filmType, "make", "type"),
      onRemove: () => setFilm(null, null),
    },
    active.color && {
      key: "color",
      category: "color",
      value: color,
      onRemove: () => setColor(null),
    },
    active.size && {
      key: "size",
      category: "size",
      value: sizeLabel(),
      onRemove: () => setSizes(defaultSizes),
    },
    flagItem("nsfw", "18+", nsfw, setNsfw),
    flagItem("bw", "b&w", bw, setBw),
    flagItem("sprocket", "sprocket", sprocket, setSprocket),
    active.text && {
      key: "text",
      category: "keyword",
      value: text,
      onRemove: () => setText(null),
    },
  ].filter(Boolean) as FilterSummaryItem[];

  return (
    <>
      <div className={styles.query}>
        <div className={styles.group}>
          <div className={styles.filterButtons}>
            <Menu shadow="md" width={220}>
              <Menu.Target>
                <Button
                  variant="outline"
                  color="gray"
                  leftSection={<IconCamera size={iconSize} stroke={1.5} />}
                  classNames={buttonClassNames}
                  {...activeProps("camera")}
                >
                  <span className={styles.label}>camera</span>
                  {activeText("camera")}
                </Button>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Label>select camera</Menu.Label>
                <div className={styles.filmSelect}>
                  <Select
                    value={
                      cameraMake && cameraModel
                        ? JSON.stringify([cameraMake, cameraModel])
                        : null
                    }
                    onChange={(value) => {
                      if (value) {
                        const [make, model] = JSON.parse(value);
                        setCamera(make, model);
                      } else {
                        setCamera(null, null);
                      }
                    }}
                    data={cameraOptions.map((c) => ({
                      value: JSON.stringify([c.make, c.model]),
                      label: c.label,
                    }))}
                    placeholder="cameras..."
                    searchable
                    clearable
                    size="sm"
                    style={{ marginBottom: 12 }}
                  />
                </div>
              </Menu.Dropdown>
            </Menu>
            <Menu shadow="md" width={220}>
              <Menu.Target>
                <Button
                  variant="outline"
                  color="gray"
                  leftSection={<IconMovie size={iconSize} stroke={1.5} />}
                  classNames={buttonClassNames}
                  {...activeProps("film")}
                >
                  <span className={styles.label}>film</span>
                  {activeText("film")}
                </Button>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Label>select film</Menu.Label>
                <div className={styles.filmSelect}>
                  <Select
                    value={
                      filmMake && filmType
                        ? JSON.stringify([filmMake, filmType])
                        : null
                    }
                    onChange={(value) => {
                      if (value) {
                        const [make, type] = JSON.parse(value);
                        setFilm(make, type);
                      } else {
                        setFilm(null, null);
                      }
                    }}
                    data={filmOptions.map((f) => ({
                      value: JSON.stringify([f.make, f.type]),
                      label: f.label,
                    }))}
                    placeholder="films..."
                    searchable
                    clearable
                    size="sm"
                    style={{ marginBottom: 12 }}
                  />
                </div>
              </Menu.Dropdown>
            </Menu>
            <ColorFilter
              color={color}
              setColor={setColor}
              buttonClassNames={buttonClassNames}
              labelClassName={styles.label}
              dotClassName={styles.dot}
              active={active.color}
            />
            <Menu shadow="md" width={230} onClose={() => commitSizes.flush()}>
              <Menu.Target>
                <Button
                  variant="outline"
                  color="gray"
                  leftSection={
                    <IconArrowAutofitWidth size={iconSize} stroke={1.6} />
                  }
                  classNames={buttonClassNames}
                  {...activeProps("size")}
                >
                  <span className={styles.label}>size</span>
                  {activeText("size")}
                </Button>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Label>with size</Menu.Label>
                <div>
                  <div className={styles.dimension}>
                    <span className={styles.dimensionTitle}>shape</span>
                    <div className={styles.shapes}>
                      {ratioShapes.map((shape, index) => (
                        <button
                          key={shape.key}
                          type="button"
                          className={styles.shape}
                          aria-pressed={shapes.includes(index)}
                          onClick={() => handleShape(index)}
                        >
                          <span
                            className={styles.shapeIcon}
                            data-shape={shape.key}
                            aria-hidden="true"
                          />
                          {shape.key}
                        </button>
                      ))}
                    </div>
                  </div>
                  <div className={styles.dimension}>
                    <span className={styles.dimensionTitle}>aspect ratio</span>
                    <div className={styles.subdimension}>
                      <div className={styles.numInputRow}>
                        <span className={styles.numInputLabel}>min</span>
                        <div className={styles.numInput}>
                          <NumberInput
                            value={drafts.ratioMin}
                            onChange={handleSizeChange("ratioMin")}
                            error={errors.ratioMin}
                            min={ratioMinLimit}
                            max={ratioMax}
                            step={0.01}
                            decimalScale={2}
                            size="xs"
                          />
                        </div>
                      </div>
                      <div className={styles.numInputRow}>
                        <span className={styles.numInputLabel}>max</span>
                        <div className={styles.numInput}>
                          <NumberInput
                            value={drafts.ratioMax}
                            onChange={handleSizeChange("ratioMax")}
                            error={errors.ratioMax}
                            min={ratioMin}
                            max={ratioMaxLimit}
                            step={0.01}
                            decimalScale={2}
                            size="xs"
                          />
                        </div>
                      </div>
                    </div>
                  </div>
                  <div className={styles.dimension}>
                    <span className={styles.dimensionTitle}>width</span>
                    <div className={styles.subdimension}>
                      <div className={styles.numInputRow}>
                        <span className={styles.numInputLabel}>min</span>
                        <div className={styles.numInput}>
                          <NumberInput
                            value={drafts.widthMin}
                            onChange={handleSizeChange("widthMin")}
                            error={errors.widthMin}
                            min={widthMinLimit}
                            max={widthMax}
                            size="xs"
                          />
                        </div>
                      </div>
                      <div className={styles.numInputRow}>
                        <span className={styles.numInputLabel}>max</span>
                        <div className={styles.numInput}>
                          <NumberInput
                            value={drafts.widthMax}
                            onChange={handleSizeChange("widthMax")}
                            error={errors.widthMax}
                            allowNegative={false}
                            min={widthMin}
                            max={widthMaxLimit}
                            size="xs"
                          />
                        </div>
                      </div>
                    </div>
                  </div>
                  <div className={styles.dimension}>
                    <span className={styles.dimensionTitle}>height</span>
                    <div className={styles.subdimension}>
                      <div className={styles.numInputRow}>
                        <span className={styles.numInputLabel}>min</span>
                        <div className={styles.numInput}>
                          <NumberInput
                            value={drafts.heightMin}
                            onChange={handleSizeChange("heightMin")}
                            error={errors.heightMin}
                            allowNegative={false}
                            min={heightMinLimit}
                            max={heightMax}
                            size="xs"
                          />
                        </div>
                      </div>
                      <div className={styles.numInputRow}>
                        <span className={styles.numInputLabel}>max</span>
                        <div className={styles.numInput}>
                          <NumberInput
                            value={drafts.heightMax}
                            onChange={handleSizeChange("heightMax")}
                            error={errors.heightMax}
                            min={heightMin}
                            max={heightMaxLimit}
                            size="xs"
                          />
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </Menu.Dropdown>
            </Menu>

            <Menu shadow="md" width={250}>
              <Menu.Target>
                <Button
                  variant="outline"
                  color="gray"
                  leftSection={
                    <IconAdjustmentsHorizontal size={iconSize} stroke={1.5} />
                  }
                  classNames={buttonClassNames}
                  {...activeProps("flags")}
                >
                  <span className={styles.label}>filters</span>
                  {activeText("flags")}
                </Button>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Label>filter by</Menu.Label>
                <div className={styles.segment}>
                  <div className={styles.segmentGroup}>
                    <h5 className={styles.segmentTitle}>18+</h5>
                    <SegmentedControl
                      value={nsfw}
                      onChange={setNsfw}
                      data={[
                        { label: "exclude", value: "exclude" },
                        { label: "include", value: "include" },
                        { label: "only", value: "only" },
                      ]}
                    />
                  </div>
                  <div className={styles.segmentGroup}>
                    <h5 className={styles.segmentTitle}>b&w</h5>
                    <SegmentedControl
                      value={bw}
                      onChange={setBw}
                      data={[
                        { label: "exclude", value: "exclude" },
                        { label: "include", value: "include" },
                        { label: "only", value: "only" },
                      ]}
                    />
                  </div>
                  <div className={styles.segmentGroup}>
                    <h5 className={styles.segmentTitle}>sprocket</h5>
                    <SegmentedControl
                      value={sprocket}
                      onChange={setSprocket}
                      data={[
                        { label: "exclude", value: "exclude" },
                        { label: "include", value: "include" },
                        { label: "only", value: "only" },
                      ]}
                    />
                  </div>
                </div>
              </Menu.Dropdown>
            </Menu>

            <span className={styles.divider} aria-hidden="true" />

            <Menu shadow="md" width={160}>
              <Menu.Target>
                <Button
                  variant="outline"
                  color="gray"
                  leftSection={<IconArrowsSort size={iconSize} stroke={1.5} />}
                  classNames={buttonClassNames}
                >
                  <span className={styles.label}>sort</span>
                </Button>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Label>sort by</Menu.Label>
                <div className={styles.radio}>
                  <Radio.Group value={sort} onChange={setSort} name="Sort">
                    <Stack gap="xs">
                      <Radio
                        value="time"
                        label="time"
                        className={styles.radioButton}
                      />
                      <Radio
                        value="score"
                        label="score"
                        className={styles.radioButton}
                      />
                      <Radio
                        value="random"
                        label="random"
                        className={styles.radioButton}
                        onClick={() => sort === "random" && setSort("random")}
                      />
                    </Stack>
                  </Radio.Group>
                </div>
                <Menu.Label>order</Menu.Label>
                <div className={styles.order}>
                  <SegmentedControl
                    size="xs"
                    fullWidth
                    value={sort === "random" ? "desc" : order}
                    onChange={(value) =>
                      setOrder(value as PostsFilters["order"])
                    }
                    disabled={sort === "random"}
                    data={orderLabels[sort === "score" ? "score" : "time"]}
                    aria-label="order"
                  />
                </div>
              </Menu.Dropdown>
            </Menu>
            <DisplayMenu
              buttonClassName={styles.button}
              sectionClassName={styles.buttonSection}
              labelClassName={styles.label}
            />
          </div>
          {onClear && (
            <FilterSummary
              count={resultCount}
              pending={resultPending}
              onClear={onClear}
              items={summaryItems}
              apiUrl={apiUrl}
            />
          )}
        </div>
      </div>
    </>
  );
}
