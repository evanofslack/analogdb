"use client";

import { postsParsers } from "@lib/searchParams";
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
import styles from "./filterBar.module.css";
import FilterSummary from "./filterSummary";

export default function FilterBar({
  // State values
  sort,
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
}) {
  const iconSize = 18;

  const sizes = {
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

  const inRange = (value, min, max, integer) =>
    typeof value === "number" &&
    Number.isFinite(value) &&
    (!integer || Number.isInteger(value)) &&
    value >= min &&
    value <= max;

  const sizeErrors = (d) => {
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

  const handleSizeChange = (key) => (value) => {
    const next = { ...draftsRef.current, [key]: value };
    draftsRef.current = next;
    setDrafts(next);
    commitSizes();
  };

  const buttonClassNames = {
    root: styles.button,
    section: styles.buttonSection,
  };

  const activeProps = (key) => (active[key] ? { "data-active": true } : {});
  const activeText = (key) => (
    <>
      <span className={styles.dot} aria-hidden="true" />
      {active[key] && <VisuallyHidden>, filter on</VisuallyHidden>}
    </>
  );

  const optionLabel = (options, a, b, keyA, keyB) =>
    options.find((o) => o[keyA] === a && o[keyB] === b)?.label ?? `${a} ${b}`;
  const defaultSizes = {
    widthMin: widthMinLimit,
    widthMax: widthMaxLimit,
    heightMin: heightMinLimit,
    heightMax: heightMaxLimit,
    ratioMin: ratioMinLimit,
    ratioMax: ratioMaxLimit,
  };
  const flagItem = (key, title, value, setValue) =>
    value !== postsParsers[key].defaultValue && {
      key,
      label: `${title}: ${value}`,
      onRemove: () => setValue(postsParsers[key].defaultValue),
    };
  const summaryItems = [
    active.camera && {
      key: "camera",
      label: `camera: ${optionLabel(
        cameraOptions,
        cameraMake,
        cameraModel,
        "make",
        "model"
      )}`,
      onRemove: () => setCamera(null, null),
    },
    active.film && {
      key: "film",
      label: `film: ${optionLabel(
        filmOptions,
        filmMake,
        filmType,
        "make",
        "type"
      )}`,
      onRemove: () => setFilm(null, null),
    },
    active.color && {
      key: "color",
      label: `color: ${color}`,
      onRemove: () => setColor(null),
    },
    active.size && {
      key: "size",
      label: "size: custom",
      onRemove: () => setSizes(defaultSizes),
    },
    flagItem("nsfw", "18+", nsfw, setNsfw),
    flagItem("bw", "b&w", bw, setBw),
    flagItem("sprocket", "sprocket", sprocket, setSprocket),
    active.text && {
      key: "text",
      label: `keyword: ${text}`,
      onRemove: () => setText(null),
    },
  ].filter(Boolean);

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
            <Menu shadow="md" width={170} onClose={() => commitSizes.flush()}>
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

            <Menu shadow="md" width={125}>
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
              </Menu.Dropdown>
            </Menu>
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
