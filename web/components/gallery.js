import Head from "next/head";
import styles from "./gallery.module.css";
import Header from "../components/header";
import Footer from "./footer";
import InfiniteGallery from "../components/infiniteGallery";
import ScrollTop from "../components/scrollTop";
import { useState, useEffect } from "react";
import useKeyPress from "../hooks/useKeyPress";
import {
  IconSearch,
  IconArrowsSort,
  IconClock,
  IconQuestionMark,
  IconTrophy,
  IconAdjustmentsHorizontal,
} from "@tabler/icons";

import {
  TextInput,
  Button,
  SegmentedControl,
  Menu,
  Radio,
} from "@mantine/core";
import { baseURL } from "../constants.ts";

async function makeRequest(queryParams) {
  const url = baseURL + "/posts/" + queryParams;
  const response = await fetch(url);
  const data = await response.json();
  return data;
}

function filterQueryParams(sort, nsfw, bw, sprocket, search) {
  let queryParams = sort + "?";

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
      queryParams = queryParams.concat("&bw=false");
      break;
    case "only":
      queryParams = queryParams.concat("&bw=true");
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
  return queryParams;
}

export default function Gallery(props) {
  const [search, setSearch] = useState("");
  const [sort, setSort] = useState("latest");
  const [nsfw, setNsfw] = useState("exclude");
  const [bw, setBw] = useState("exclude");
  const [sprocket, setSprocket] = useState("include");
  const [response, setResponse] = useState(props.data);

  const updateRequest = async () => {
    let request = filterQueryParams(sort, nsfw, bw, sprocket, search);
    const response = await makeRequest(request);
    setResponse(response);
  };

  const returnPress = useKeyPress("Enter");

  useEffect(() => {
    updateRequest();
  }, [sort, nsfw, bw, sprocket, returnPress]);

  return (
    <div className={styles.main}>
      <Head>
        <title>AnalogDB</title>
        <link rel="icon" href="/favicon.ico" />
      </Head>
      <Header />

      <div className={styles.query}>
        <Menu shadow="md" width={125}>
          <Menu.Target>
            <Button
              variant="outline"
              color="gray"
              leftIcon={<IconArrowsSort size={18} stroke={1.5} />}
              styles={(theme) => ({
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
            <Menu.Label>sort posts by</Menu.Label>
            <div className={styles.radio}>
              <Radio.Group
                value={sort}
                onChange={setSort}
                name="Sort"
                orientation="vertical"
                spacing="sm"
              >
                <Radio value="latest" label="latest" />
                <Radio value="top" label="top" />
                <Radio value="random" label="random" />
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
              styles={(theme) => ({
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
            <Menu.Label>filter posts by</Menu.Label>
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
      <Footer />
    </div>
  );
}