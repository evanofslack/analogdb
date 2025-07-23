"use client";

import {
  Button,
  Menu,
  NumberInput,
  Radio,
  SegmentedControl,
  Select,
  TextInput,
} from "@mantine/core";
import {
  IconAdjustmentsHorizontal,
  IconArrowAutofitWidth,
  IconArrowsSort,
  IconCamera,
  IconMovie,
  IconSearch,
} from "@tabler/icons-react";
import { useState } from "react";
import ColorFilter from "./colorFilter";
import styles from "./filterBar.module.css";

export default function FilterBar({
  // State values
  sort,
  nsfw,
  bw,
  sprocket,
  color,
  textTemp,
  widthMin,
  widthMax,
  heightMin,
  heightMax,
  ratioMin,
  ratioMax,
  filmMake,
  filmType,
  filmSpeed,

  // State setters
  setSort,
  setNsfw,
  setBw,
  setSprocket,
  setColor,
  setTextTemp,
  setWidthMin,
  setWidthMax,
  setHeightMin,
  setHeightMax,
  setRatioMin,
  setRatioMax,
  setFilmMake,
  setFilmType,
  setFilmSpeed,

  filmOptions,

  // UI state
  onlyIcon,
  textPlaceholder,

  // Limits
  widthMinLimit,
  widthMaxLimit,
  heightMinLimit,
  heightMaxLimit,
  ratioMinLimit,
  ratioMaxLimit,
}) {
  const [searchExpanded, setSearchExpanded] = useState(false);

  const getButtonStyles = (onlyIcon) => ({
    root: {
      marginRight: 10,
      paddingLeft: 10,
      paddingRight: onlyIcon ? 0 : 10,
      color: "#2E2E2E",
      fontWeight: 400,
      borderColor: onlyIcon ? "transparent" : "#CED4DA",
      "&:hover": {
        backgroundColor: onlyIcon ? "transparent" : "#fbfbfc",
      },
      leftSection: {
        marginRight: 5,
      },
    },
  });

  return (
    <div className={`${styles.query} ${onlyIcon ? styles.queryIconMode : ""}`}>
      <div className={styles.filterButtons}>
        <Menu shadow="md" width={220}>
          <Menu.Target>
            <Button
              variant="outline"
              color="gray"
              leftSection={
                <IconCamera size={onlyIcon ? 22 : 18} stroke={1.5} />
              }
              styles={() => getButtonStyles(onlyIcon)}
            >
              {!onlyIcon && <span>camera</span>}
            </Button>
          </Menu.Target>
          <Menu.Dropdown>
            <Menu.Label>select film</Menu.Label>
            <div className={styles.filmSelect}>
              <Select
                value={filmMake && filmType ? `${filmMake} - ${filmType}` : ""}
                onChange={(value) => {
                  if (value) {
                    const [make, type] = value.split(" - ");
                    setFilmMake(make);
                    setFilmType(type);
                  } else {
                    setFilmMake(null);
                    setFilmType(null);
                  }
                }}
                data={filmOptions.map((f) => f.label)}
                placeholder="films..."
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
              leftSection={<IconMovie size={onlyIcon ? 22 : 18} stroke={1.5} />}
              styles={() => getButtonStyles(onlyIcon)}
            >
              {!onlyIcon && <span>film</span>}
            </Button>
          </Menu.Target>
          <Menu.Dropdown>
            <Menu.Label>select film</Menu.Label>
            <div className={styles.filmSelect}>
              <Select
                value={filmMake && filmType ? `${filmMake} - ${filmType}` : ""}
                onChange={(value) => {
                  if (value) {
                    const [make, type] = value.split(" - ");
                    setFilmMake(make);
                    setFilmType(type);
                  } else {
                    setFilmMake(null);
                    setFilmType(null);
                  }
                }}
                data={filmOptions.map((f) => f.label)}
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
          onlyIcon={onlyIcon}
          buttonStyles={getButtonStyles(onlyIcon)}
        />
        <Menu shadow="md" width={170}>
          <Menu.Target>
            <Button
              variant="outline"
              color="gray"
              leftSection={
                <IconArrowAutofitWidth size={onlyIcon ? 22 : 18} stroke={1.6} />
              }
              styles={() => getButtonStyles(onlyIcon)}
            >
              {!onlyIcon && <span>size</span>}
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
                        value={ratioMin}
                        onChange={setRatioMin}
                        min={ratioMinLimit}
                        max={ratioMax}
                        step={0.01}
                        precision={2}
                        size="xs"
                      />
                    </div>
                  </div>
                  <div className={styles.numInputRow}>
                    <span className={styles.numInputLabel}>max</span>
                    <div className={styles.numInput}>
                      <NumberInput
                        value={ratioMax}
                        onChange={setRatioMax}
                        min={ratioMin}
                        max={ratioMaxLimit}
                        step={0.01}
                        precision={2}
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
                        value={widthMin}
                        onChange={setWidthMin}
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
                        value={widthMax}
                        onChange={setWidthMax}
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
                        value={heightMin}
                        onChange={setHeightMin}
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
                        value={heightMax}
                        onChange={setHeightMax}
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

        <Menu shadow="md" width={125}>
          <Menu.Target>
            <Button
              variant="outline"
              color="gray"
              leftSection={
                <IconArrowsSort size={onlyIcon ? 22 : 18} stroke={1.5} />
              }
              styles={() => getButtonStyles(onlyIcon)}
            >
              {!onlyIcon && <span>sort</span>}
            </Button>
          </Menu.Target>
          <Menu.Dropdown>
            <Menu.Label>sort by</Menu.Label>
            <div className={styles.radio}>
              <Radio.Group
                value={sort}
                onChange={setSort}
                name="Sort"
                orientation="vertical"
                spacing="md"
              >
                <Radio
                  value="latest"
                  label="latest"
                  className={styles.radioButton}
                />
                <Radio value="top" label="top" className={styles.radioButton} />
                <Radio
                  value="random"
                  label="random"
                  className={styles.radioButton}
                />
              </Radio.Group>
            </div>
          </Menu.Dropdown>
        </Menu>

        <Menu shadow="md" width={250}>
          <Menu.Target>
            <Button
              variant="outline"
              color="gray"
              leftSection={
                <IconAdjustmentsHorizontal
                  size={onlyIcon ? 22 : 18}
                  stroke={1.5}
                />
              }
              styles={() => getButtonStyles(onlyIcon)}
            >
              {!onlyIcon && <span>filter</span>}
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

        {onlyIcon && (
          <div className={styles.searchContainer}>
            {!searchExpanded ? (
              <Button
                variant="outline"
                color="gray"
                leftSection={<IconSearch size={22} stroke={1.5} />}
                styles={() => getButtonStyles(onlyIcon)}
                onClick={() => setSearchExpanded(true)}
              />
            ) : (
              <div className={styles.expandedSearch}>
                <TextInput
                  value={textTemp}
                  onChange={(event) => setTextTemp(event.currentTarget.value)}
                  leftSection={<IconSearch size={18} />}
                  leftSectionPointerEvents="none"
                  placeholder={textPlaceholder}
                  onBlur={() => setSearchExpanded(false)}
                  autoFocus
                  styles={{
                    input: {
                      minWidth: "250px",
                    },
                  }}
                />
              </div>
            )}
          </div>
        )}
      </div>

      {!onlyIcon && (
        <TextInput
          value={textTemp}
          onChange={(event) => setTextTemp(event.currentTarget.value)}
          leftSection={<IconSearch size={18} />}
          leftSectionPointerEvents="none"
          placeholder={textPlaceholder}
        />
      )}
    </div>
  );
}
