import styles from "./gallery.module.css";
import Header from "../components/header";
import Footer from "./footer";
import InfiniteGallery from "../components/infiniteGallery";
import ScrollTop from "../components/scrollTop";
import { useState, useEffect } from "react";
import { useBreakpoint } from "../providers/breakpoint.js";
import useKeyPress from "../hooks/useKeyPress";
import { useQueryState, queryTypes } from "next-usequerystate";
import {
  IconArrowAutofitWidth,
  IconSearch,
  IconArrowsSort,
  IconAdjustmentsHorizontal,
  IconPalette,
} from "@tabler/icons";

import {
  TextInput,
  Button,
  SegmentedControl,
  Menu,
  Radio,
  Checkbox,
  Tooltip,
  NumberInput,
} from "@mantine/core";

import { baseURL } from "../constants.js";

async function makeRequest(queryParams) {
  const route = "/posts" + queryParams;
  const response = await fetch(baseURL + route);
  const data = await response.json();
  return data;
}

function filterQueryParams(
  sort,
  nsfw,
  bw,
  sprocket,
  text,
  color,
  width,
  height,
  ratio
) {
  let queryParams = "?" + "sort=" + sort;

  switch (nsfw) {
    case "exclude":
      queryParams = queryParams.concat("&nsfw=false");
      break;
    case "only":
      queryParams = queryParams.concat("&nsfw=true");
      break;
  }

  switch (bw) {
    case "exclude":
      queryParams = queryParams.concat("&grayscale=false");
      break;
    case "only":
      queryParams = queryParams.concat("&grayscale=true");
      break;
  }

  switch (sprocket) {
    case "exclude":
      queryParams = queryParams.concat("&sprocket=false");
      break;
    case "only":
      queryParams = queryParams.concat("&sprocket=true");
      break;
  }

  if (text !== "") {
    let keywords = text.split(/[ ,]+/).filter(Boolean);
    keywords.forEach(
      (word) => (queryParams = queryParams.concat("&keyword=" + word))
    );
  }

  if (color !== "") {
    queryParams = queryParams.concat("&color=" + color);
    if (color === "gray") {
      queryParams = queryParams.concat("&min_color=" + "0.8");
    } else if (color === "black") {
      queryParams = queryParams.concat("&min_color=" + "0.7");
    } else if (color === "white") {
      queryParams = queryParams.concat("&min_color=" + "0.50");
    } else if (color === "teal") {
      queryParams = queryParams.concat("&min_color=" + "0.35");
    } else if (color === "olive" || color === "brown") {
      queryParams = queryParams.concat("&min_color=" + "0.35");
    } else if (color === "tan") {
      queryParams = queryParams.concat("&min_color=" + "0.30");
    } else if (color === "navy" || color === "green") {
      queryParams = queryParams.concat("&min_color=" + "0.25");
    } else {
      queryParams = queryParams.concat("&min_color=" + "0.15");
    }
  }

  queryParams = queryParams.concat("&width_min=" + width[0]);
  queryParams = queryParams.concat("&width_max=" + width[1]);
  queryParams = queryParams.concat("&height_min=" + height[0]);
  queryParams = queryParams.concat("&height_max=" + height[1]);
  queryParams = queryParams.concat("&ratio_min=" + ratio[0]);
  queryParams = queryParams.concat("&ratio_max=" + ratio[1]);

  queryParams = queryParams.concat("&page_size=" + 100);

  // console.log(queryParams);

  return queryParams;
}

const defaultSort = "latest";
const defaultNsfw = "exclude";
const defaultBw = "exclude";
const defaultSprocket = "include";
const defaultColor = "";
const defaultText = "";

export default function Gallery(props) {
  // querystate
  const [sort, setSort] = useQueryState(
    "sort",
    queryTypes.string.withDefault(defaultSort)
  );
  const [nsfw, setNsfw] = useQueryState(
    "nsfw",
    queryTypes.string.withDefault(defaultNsfw)
  );
  const [bw, setBw] = useQueryState(
    "bw",
    queryTypes.string.withDefault(defaultBw)
  );
  const [sprocket, setSprocket] = useQueryState(
    "sprocket",
    queryTypes.string.withDefault(defaultSprocket)
  );

  // handle setting sizes
  let widthMinLimit = 600;
  let widthMaxLimit = 15000;
  const [widthMin, setWidthMin] = useState(widthMinLimit);
  const [widthMax, setWidthMax] = useState(widthMaxLimit);

  let heightMinLimit = 400;
  let heightMaxLimit = 12000;
  const [heightMin, setHeightMin] = useState(heightMinLimit);
  const [heightMax, setHeightMax] = useState(heightMaxLimit);

  let ratioMinLimit = 0.3;
  let ratioMaxLimit = 4.8;
  const [ratioMin, setRatioMin] = useState(ratioMinLimit);
  const [ratioMax, setRatioMax] = useState(ratioMaxLimit);

  // handle setting colors
  const [color, setColor] = useQueryState(
    "color",
    queryTypes.string.withDefault(defaultColor)
  );

  const handleColorClick = (event) => {
    let clickedColor = event.target.id;
    if (clickedColor === color) {
      setColor(null);
    } else {
      setColor(clickedColor);
    }
  };

  // handle setting keywords.
  // hold input text in temp variable and
  // only set query state on updateRequest.
  const [textTemp, setTextTemp] = useState("");
  const [text, setText] = useQueryState(
    "text",
    queryTypes.string.withDefault(defaultText)
  );

  const [response, setResponse] = useState(props.data);

  const updateRequest = async () => {
    if (textTemp == defaultText) {
      setText(null);
    } else {
      setText(textTemp);
    }

    let request = filterQueryParams(
      sort,
      nsfw,
      bw,
      sprocket,
      text,
      color,
      [widthMin, widthMax],
      [heightMin, heightMax],
      [ratioMin, ratioMax]
    );
    const response = await makeRequest(request);
    setResponse(response);
  };

  const returnPress = useKeyPress("Enter");

  const breakpoints = useBreakpoint();

  let onlyIcon = false;
  if (breakpoints["xs"] || breakpoints["sm"]) {
    onlyIcon = true;
  }

  const textPlaceholder = () => {
    const placeholder = onlyIcon
      ? "films, cameras..."
      : "films, cameras, places...";
    return placeholder;
  };

  useEffect(() => {
    updateRequest();
  }, [
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
    returnPress,
  ]);

  return (
    <div className={styles.main}>
      <Header />
      <div className={styles.margin}>
        <div className={styles.query}>
          <Menu shadow="md" width={170}>
            <Menu.Target>
              <Button
                variant="outline"
                color="gray"
                leftIcon={
                  <IconArrowAutofitWidth
                    size={onlyIcon ? 22 : 18}
                    stroke={1.6}
                  />
                }
                styles={() => ({
                  root: {
                    marginRight: 10,
                    paddingLeft: 10,
                    paddingRight: onlyIcon ? 0 : 10,
                    color: "#2E2E2E",
                    fontWeight: 400,
                    borderColor: "#CED4DA",
                    "&:hover": {
                      backgroundColor: "#fbfbfc",
                    },
                    leftIcon: {
                      marginRight: 5,
                    },
                  },
                })}
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

          <Menu shadow="md" width={100}>
            <Menu.Target>
              <Button
                variant="outline"
                color="gray"
                leftIcon={
                  <IconPalette size={onlyIcon ? 22 : 18} stroke={1.5} />
                }
                styles={() => ({
                  root: {
                    marginRight: 10,
                    paddingLeft: 10,
                    paddingRight: onlyIcon ? 0 : 10,
                    color: "#2E2E2E",
                    fontWeight: 400,
                    borderColor: "#CED4DA",
                    "&:hover": {
                      backgroundColor: "#fbfbfc",
                    },
                    leftIcon: {
                      marginRight: 5,
                    },
                  },
                })}
              >
                {!onlyIcon && <span>color</span>}
              </Button>
            </Menu.Target>
            <Menu.Dropdown>
              <Menu.Label>with color</Menu.Label>
              <div className={styles.colors}>
                <Tooltip
                  label="red"
                  color={color === "red" ? "red.8" : "red.7"}
                  position="right"
                  withArrow
                >
                  <Checkbox
                    styles={{
                      input: { backgroundColor: "#f03e3e", border: "None" },
                    }}
                    size="md"
                    color="red.8"
                    checked={color === "red"}
                    onChange={handleColorClick}
                    key={1}
                    id={"red"}
                    className={styles.colorButton}
                    radius="xs"
                  />
                </Tooltip>
                <Tooltip
                  label="orange"
                  color={color === "orange" ? "orange.7" : "orange.6"}
                  position="right"
                  withArrow
                >
                  <Checkbox
                    styles={{
                      input: { backgroundColor: "#fd7e14", border: "None" },
                    }}
                    size="md"
                    color="orange.7"
                    checked={color === "orange"}
                    onChange={handleColorClick}
                    key={2}
                    id={"orange"}
                    className={styles.colorButton}
                    radius="xs"
                  />
                </Tooltip>
                <Tooltip
                  label="beige"
                  color={color === "tan" ? "brown.2" : "brown.1"}
                  position="right"
                  withArrow
                >
                  <Checkbox
                    styles={{
                      input: { backgroundColor: "#ffdcb0", border: "None" },
                    }}
                    size="md"
                    color="brown.2"
                    checked={color === "tan"}
                    onChange={handleColorClick}
                    key={3}
                    id={"tan"}
                    className={styles.colorButton}
                    radius="xs"
                  />
                </Tooltip>
                <Tooltip
                  label="yellow"
                  color={color === "yellow" ? "yellow.5" : "yellow.4"}
                  position="right"
                  withArrow
                >
                  <Checkbox
                    styles={{
                      input: { backgroundColor: "#ffd43b", border: "None" },
                    }}
                    size="md"
                    color="yellow.5"
                    checked={color === "yellow"}
                    onChange={handleColorClick}
                    key={4}
                    id={"yellow"}
                    className={styles.colorButton}
                    radius="xs"
                  />
                </Tooltip>
                <Tooltip
                  label="green"
                  color={color === "green" ? "green.9" : "green.8"}
                  position="right"
                  withArrow
                >
                  <Checkbox
                    styles={{
                      input: { backgroundColor: "#2f9e44", border: "None" },
                    }}
                    size="md"
                    color="green.9"
                    checked={color === "green"}
                    onChange={handleColorClick}
                    key={5}
                    id={"green"}
                    className={styles.colorButton}
                    radius="xs"
                  />
                </Tooltip>
                <Tooltip
                  label="olive"
                  color={color === "olive" ? "olive.9" : "olive.8"}
                  position="right"
                  withArrow
                >
                  <Checkbox
                    styles={{
                      input: { backgroundColor: "#4c4d00", border: "None" },
                    }}
                    size="md"
                    color="olive.9"
                    checked={color === "olive"}
                    onChange={handleColorClick}
                    key={6}
                    id={"olive"}
                    className={styles.colorButton}
                    radius="xs"
                  />
                </Tooltip>
                <Tooltip
                  label="teal"
                  color={color === "teal" ? "cyan.6" : "cyan.5"}
                  position="right"
                  withArrow
                >
                  <Checkbox
                    styles={{
                      input: { backgroundColor: "#22b8cf", border: "None" },
                    }}
                    size="md"
                    color="cyan.6"
                    checked={color === "teal"}
                    onChange={handleColorClick}
                    key={7}
                    id={"teal"}
                    className={styles.colorButton}
                    radius="xs"
                  />
                </Tooltip>
                <Tooltip
                  label="navy"
                  color={color === "navy" ? "navy.8" : "navy.7"}
                  position="right"
                  withArrow
                >
                  <Checkbox
                    styles={{
                      input: { backgroundColor: "#064679", border: "None" },
                    }}
                    size="md"
                    color="navy.8"
                    checked={color === "navy"}
                    onChange={handleColorClick}
                    key={8}
                    id={"navy"}
                    className={styles.colorButton}
                    radius="xs"
                  />
                </Tooltip>
                <Tooltip
                  label="purple"
                  color={color === "purple" ? "grape.9" : "grape.8"}
                  position="right"
                  withArrow
                >
                  <Checkbox
                    styles={{
                      input: { backgroundColor: "#9c36b5", border: "None" },
                    }}
                    size="md"
                    color="grape.9"
                    checked={color === "purple"}
                    onChange={handleColorClick}
                    key={9}
                    id={"purple"}
                    className={styles.colorButton}
                    radius="xs"
                  />
                </Tooltip>
                <Tooltip
                  label="gray"
                  color={color === "gray" ? "dark.3" : "dark.2"}
                  position="right"
                  withArrow
                >
                  <Checkbox
                    styles={{
                      input: { backgroundColor: "#868e96", border: "None" },
                    }}
                    size="md"
                    color="dark.3"
                    checked={color === "gray"}
                    onChange={handleColorClick}
                    key={10}
                    id={"gray"}
                    className={styles.colorButton}
                    radius="xs"
                  />
                </Tooltip>
                <Tooltip
                  label="brown"
                  color={color === "brown" ? "brown.8" : "brown.7"}
                  position="right"
                  withArrow
                >
                  <Checkbox
                    styles={{
                      input: { backgroundColor: "#7d4500", border: "None" },
                    }}
                    size="md"
                    color="brown.8"
                    checked={color === "brown"}
                    onChange={handleColorClick}
                    key={11}
                    id={"brown"}
                    className={styles.colorButton}
                    radius="xs"
                  />
                </Tooltip>
                <Tooltip
                  label="black"
                  color={color === "black" ? "dark.9" : "dark.8"}
                  position="right"
                  withArrow
                >
                  <Checkbox
                    styles={{
                      input: { backgroundColor: "#141517", border: "None" },
                    }}
                    size="md"
                    color="dark.9"
                    checked={color === "black"}
                    onChange={handleColorClick}
                    key={12}
                    id={"black"}
                    className={styles.colorButton}
                    radius="xs"
                  />
                </Tooltip>
                <Tooltip
                  label="white"
                  color={color === "white" ? "gray.3" : "gray.2"}
                  position="right"
                  withArrow
                >
                  <Checkbox
                    styles={{
                      input: { backgroundColor: "#e9ecef", border: "None" },
                    }}
                    size="md"
                    color="gray.3"
                    checked={color === "white"}
                    onChange={handleColorClick}
                    key={13}
                    id={"white"}
                    className={styles.colorButton}
                    radius="xs"
                  />
                </Tooltip>
              </div>
            </Menu.Dropdown>
          </Menu>

          <Menu shadow="md" width={125}>
            <Menu.Target>
              <Button
                variant="outline"
                color="gray"
                leftIcon={
                  <IconArrowsSort size={onlyIcon ? 22 : 18} stroke={1.5} />
                }
                styles={() => ({
                  root: {
                    marginRight: 10,
                    paddingLeft: 10,
                    paddingRight: onlyIcon ? 0 : 10,
                    color: "#2E2E2E",
                    fontWeight: 400,
                    borderColor: "#CED4DA",
                    "&:hover": {
                      backgroundColor: "#fbfbfc",
                    },
                    leftIcon: {
                      marginRight: 5,
                    },
                  },
                })}
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
                  <Radio
                    value="top"
                    label="top"
                    className={styles.radioButton}
                  />
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
                leftIcon={
                  <IconAdjustmentsHorizontal
                    size={onlyIcon ? 22 : 18}
                    stroke={1.5}
                  />
                }
                styles={() => ({
                  root: {
                    marginRight: 10,
                    paddingLeft: 10,
                    paddingRight: onlyIcon ? 0 : 10,
                    color: "#2E2E2E",
                    fontWeight: 400,
                    borderColor: "#CED4DA",
                    "&:hover": {
                      backgroundColor: "#fbfbfc",
                    },
                    leftIcon: {
                      marginRight: 5,
                    },
                  },
                })}
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

          <TextInput
            value={textTemp}
            onChange={(event) => setTextTemp(event.currentTarget.value)}
            icon={<IconSearch size={18} />}
            placeholder={textPlaceholder()}
          />
        </div>

        <InfiniteGallery response={response} />
        <ScrollTop />
      </div>
      <Footer />
    </div>
  );
}
