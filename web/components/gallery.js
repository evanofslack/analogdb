import styles from "./gallery.module.css";
import Header from "../components/header";
import Footer from "./footer";
import InfiniteGallery from "../components/infiniteGallery";
import ScrollTop from "../components/scrollTop";
import { useState, useEffect } from "react";
import useKeyPress from "../hooks/useKeyPress";
import { useQueryState, queryTypes } from "next-usequerystate";
import {
  IconSearch,
  IconArrowsSort,
  IconAdjustmentsHorizontal,
  IconPalette,
  IconCheck,
} from "@tabler/icons";

import {
  TextInput,
  Button,
  SegmentedControl,
  Menu,
  Radio,
  Checkbox,
} from "@mantine/core";

import { baseURL } from "../constants.js";

async function makeRequest(queryParams) {
  const route = "/posts" + queryParams;
  const response = await fetch(baseURL + route);
  const data = await response.json();
  return data;
}

function filterQueryParams(sort, nsfw, bw, sprocket, search, color) {
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

  if (search !== "") {
    queryParams = queryParams.concat("&title=" + search);
  }

  if (color !== "") {
    queryParams = queryParams.concat("&color=" + color);
    if (color === "black" || color === "gray") {
      queryParams = queryParams.concat("&min_color=" + "0.8");
    }
    if (color === "white") {
      queryParams = queryParams.concat("&min_color=" + "0.6");
    }
    if (color === "teal") {
      queryParams = queryParams.concat("&min_color=" + "0.25");
    }
    if (color === "navy" || color === "green") {
      queryParams = queryParams.concat("&min_color=" + "0.15");
    }
  }

  queryParams = queryParams.concat("&page_size=" + 100);

  return queryParams;
}

export default function Gallery(props) {
  // querystate
  const [sort, setSort] = useQueryState(
    "sort",
    queryTypes.string.withDefault("latest")
  );
  const [nsfw, setNsfw] = useQueryState(
    "nsfw",
    queryTypes.string.withDefault("exclude")
  );
  const [bw, setBw] = useQueryState(
    "bw",
    queryTypes.string.withDefault("exclude")
  );
  const [sprocket, setSprocket] = useQueryState(
    "sprocket",
    queryTypes.string.withDefault("include")
  );
  const [search, setSearch] = useQueryState(
    "text",
    queryTypes.string.withDefault("")
  );
  const [color, setColor] = useQueryState(
    "color",
    queryTypes.string.withDefault("")
  );

  const handleColorClick = (event) => {
    let clickedColor = event.target.id;
    if (clickedColor === color) {
      setColor(null);
    } else {
      setColor(clickedColor);
    }
  };

  const blackCheck = () => {
    <IconCheck color="#000" />;
  };

  const [response, setResponse] = useState(props.data);

  const updateRequest = async () => {
    let request = filterQueryParams(sort, nsfw, bw, sprocket, search, color);
    const response = await makeRequest(request);
    setResponse(response);
  };

  const returnPress = useKeyPress("Enter");

  useEffect(() => {
    updateRequest();
  }, [sort, nsfw, bw, sprocket, color, returnPress]);

  return (
    <div className={styles.main}>
      <Header />
      <div className={styles.margin}>
        <div className={styles.query}>
          <Menu shadow="md" width={100}>
            <Menu.Target>
              <Button
                variant="outline"
                color="gray"
                leftIcon={<IconPalette size={18} stroke={1.5} />}
                styles={() => ({
                  root: {
                    marginRight: 10,
                    paddingLeft: 10,
                    paddingRight: 10,
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
                color
              </Button>
            </Menu.Target>
            <Menu.Dropdown>
              <Menu.Label>with color</Menu.Label>
              <div className={styles.colors}>
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
                />
                <Checkbox
                  styles={{
                    input: { backgroundColor: "#ffd43b", border: "None" },
                  }}
                  size="md"
                  color="yellow.5"
                  checked={color === "yellow"}
                  onChange={handleColorClick}
                  key={2}
                  id={"yellow"}
                  className={styles.colorButton}
                />
                <Checkbox
                  styles={{
                    input: { backgroundColor: "#2f9e44", border: "None" },
                  }}
                  size="md"
                  color="green.9"
                  checked={color === "green"}
                  onChange={handleColorClick}
                  key={3}
                  id={"green"}
                  className={styles.colorButton}
                />
                <Checkbox
                  styles={{
                    input: { backgroundColor: "#22b8cf", border: "None" },
                  }}
                  size="md"
                  color="cyan.6"
                  checked={color === "teal"}
                  onChange={handleColorClick}
                  key={4}
                  id={"teal"}
                  className={styles.colorButton}
                />
                <Checkbox
                  styles={{
                    input: { backgroundColor: "#1971c2", border: "None" },
                  }}
                  size="md"
                  color="blue.9"
                  checked={color === "navy"}
                  onChange={handleColorClick}
                  key={5}
                  id={"navy"}
                  className={styles.colorButton}
                />
                <Checkbox
                  styles={{
                    input: { backgroundColor: "#9c36b5", border: "None" },
                  }}
                  size="md"
                  color="grape.9"
                  checked={color === "purple"}
                  onChange={handleColorClick}
                  key={6}
                  id={"purple"}
                  className={styles.colorButton}
                />
                <Checkbox
                  styles={{
                    input: { backgroundColor: "#868e96", border: "None" },
                  }}
                  size="md"
                  color="dark.3"
                  checked={color === "gray"}
                  onChange={handleColorClick}
                  key={7}
                  id={"gray"}
                  className={styles.colorButton}
                />
                <Checkbox
                  styles={{
                    input: { backgroundColor: "#141517", border: "None" },
                  }}
                  size="md"
                  color="dark.9"
                  checked={color === "black"}
                  onChange={handleColorClick}
                  key={8}
                  id={"black"}
                  className={styles.colorButton}
                />
                <Checkbox
                  styles={{
                    input: { backgroundColor: "#e9ecef", border: "None" },
                  }}
                  size="md"
                  color="gray.3"
                  checked={color === "white"}
                  onChange={handleColorClick}
                  key={9}
                  id={"white"}
                  className={styles.colorButton}
                />
              </div>
            </Menu.Dropdown>
          </Menu>

          <Menu shadow="md" width={125}>
            <Menu.Target>
              <Button
                variant="outline"
                color="gray"
                leftIcon={<IconArrowsSort size={18} stroke={1.5} />}
                styles={() => ({
                  root: {
                    marginRight: 10,
                    paddingLeft: 10,
                    paddingRight: 10,
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
                sort
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
                leftIcon={<IconAdjustmentsHorizontal size={18} stroke={1.5} />}
                styles={() => ({
                  root: {
                    marginRight: 10,
                    paddingLeft: 10,
                    paddingRight: 10,
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
                filter
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
            value={search}
            onChange={(event) => setSearch(event.currentTarget.value)}
            icon={<IconSearch size={18} />}
            placeholder="films, cameras, places..."
          />
        </div>

        <InfiniteGallery response={response} />
        <ScrollTop />
      </div>
      <Footer />
    </div>
  );
}
