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
  IconAdjustmentsHorizontal,
} from "@tabler/icons";
import useQuery from "../stores/query";

import {
  TextInput,
  Button,
  SegmentedControl,
  Menu,
  Radio,
} from "@mantine/core";
import { baseURL } from "../constants.ts";

async function makeRequest(queryParams) {
  const url = baseURL + "/posts" + queryParams;
  const response = await fetch(url);
  const data = await response.json();
  return data;
}

function filterQueryParams(sort, nsfw, bw, sprocket, search) {
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

  queryParams = queryParams.concat("&page_size=" + 100);

  return queryParams;
}

export default function Gallery(props) {
  const { search, sort, nsfw, bw, sprocket } = useQuery((store) => ({
    search: store.search,
    sort: store.sort,
    nsfw: store.nsfw,
    bw: store.bw,
    sprocket: store.sprocket,
  }));

  const { setSearch, setSort, setNsfw, setBw, setSprocket } = useQuery(
    (store) => ({
      setSearch: store.setSearch,
      setSort: store.setSort,
      setNsfw: store.setNsfw,
      setBw: store.setBw,
      setSprocket: store.setSprocket,
    })
  );

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
      <Header />
      <div className={styles.margin}>
        <div className={styles.query}>
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
              <Menu.Label>sort posts by</Menu.Label>
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
      </div>
      <Footer />
    </div>
  );
}
