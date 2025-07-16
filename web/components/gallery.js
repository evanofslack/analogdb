"use client";

import useKeyPress from "@hooks/useKeyPress";
import { baseURL } from "@lib/constants.js";
import { useBreakpoint } from "@providers/breakpoint.js";
import { useQueryState } from "nuqs";
import { useCallback, useEffect, useRef, useState } from "react";
import FilterBar from "./filterBar";
import Footer from "./footer";
import styles from "./gallery.module.css";
import Header from "./header";
import InfiniteGallery from "./infiniteGallery";
import ScrollTop from "./scrollTop";

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

  return queryParams;
}

const defaultSort = "latest";
const defaultNsfw = "exclude";
const defaultBw = "exclude";
const defaultSprocket = "include";
const defaultColor = "";
const defaultText = "";

export default function Gallery(props) {
  let isAdmin = props.isAdmin;
  let data = props.data;

  const [sort, setSort] = useQueryState("sort", {
    history: "push",
    defaultValue: defaultSort,
  });
  const [nsfw, setNsfw] = useQueryState("nsfw", {
    history: "push",
    defaultValue: defaultNsfw,
  });
  const [bw, setBw] = useQueryState("bw", {
    history: "push",
    defaultValue: defaultBw,
  });
  const [sprocket, setSprocket] = useQueryState("sprocket", {
    history: "push",
    defaultValue: defaultSprocket,
  });

  // Dimension limits
  let widthMinLimit = 600;
  let widthMaxLimit = 15000;
  let heightMinLimit = 400;
  let heightMaxLimit = 12000;
  let ratioMinLimit = 0.3;
  let ratioMaxLimit = 4.8;

  const [widthMin, setWidthMin] = useQueryState("widthMin", {
    defaultValue: widthMinLimit,
  });
  const [widthMax, setWidthMax] = useQueryState("widthMax", {
    defaultValue: widthMaxLimit,
  });
  const [heightMin, setHeightMin] = useQueryState("heightMin", {
    defaultValue: heightMinLimit,
  });
  const [heightMax, setHeightMax] = useQueryState("heightMax", {
    defaultValue: heightMaxLimit,
  });
  const [ratioMin, setRatioMin] = useQueryState("ratioMin", {
    defaultValue: ratioMinLimit,
  });
  const [ratioMax, setRatioMax] = useQueryState("ratioMax", {
    defaultValue: ratioMaxLimit,
  });

  const [color, setColor] = useQueryState("color", { defaultValue: "" });
  const [text, setText] = useQueryState("text", { defaultValue: "" });
  const [textTemp, setTextTemp] = useState(text || "");

  const returnPress = useKeyPress("Enter");
  const breakpoints = useBreakpoint();

  let onlyIcon = false;
  if (breakpoints["xs"] || breakpoints["sm"]) {
    onlyIcon = true;
  }

  const textPlaceholder = onlyIcon
    ? "films, cameras..."
    : "films, cameras, places...";

  const [response, setResponse] = useState(data);
  const isInitialLoad = useRef(true);

  const updateRequest = useCallback(async () => {
    if (textTemp === defaultText) {
      setText(null);
    } else {
      setText(textTemp);
    }

    let request = filterQueryParams(
      sort || defaultSort,
      nsfw || defaultNsfw,
      bw || defaultBw,
      sprocket || defaultSprocket,
      text || defaultText,
      color || defaultColor,
      [widthMin, widthMax],
      [heightMin, heightMax],
      [ratioMin, ratioMax]
    );
    const response = await makeRequest(request);
    setResponse(response);
  }, [
    textTemp,
    sort,
    nsfw,
    bw,
    sprocket,
    text,
    color,
    widthMin,
    widthMax,
    heightMin,
    heightMax,
    ratioMin,
    ratioMax,
  ]);

  useEffect(() => {
    if (isInitialLoad.current) {
      const isUsingDefaults =
        (sort || defaultSort) === defaultSort &&
        (nsfw || defaultNsfw) === defaultNsfw &&
        (bw || defaultBw) === defaultBw &&
        (sprocket || defaultSprocket) === defaultSprocket &&
        (text || defaultText) === defaultText &&
        (color || defaultColor) === defaultColor;

      if (isUsingDefaults) {
        isInitialLoad.current = false;
        return;
      }
    }

    isInitialLoad.current = false;
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
    updateRequest,
  ]);

  return (
    <div className={styles.main}>
      <Header isAdmin={isAdmin} />
      <div className={styles.margin}>
        <FilterBar
          sort={sort}
          nsfw={nsfw}
          bw={bw}
          sprocket={sprocket}
          color={color}
          textTemp={textTemp}
          widthMin={widthMin}
          widthMax={widthMax}
          heightMin={heightMin}
          heightMax={heightMax}
          ratioMin={ratioMin}
          ratioMax={ratioMax}
          setSort={setSort}
          setNsfw={setNsfw}
          setBw={setBw}
          setSprocket={setSprocket}
          setColor={setColor}
          setTextTemp={setTextTemp}
          setWidthMin={setWidthMin}
          setWidthMax={setWidthMax}
          setHeightMin={setHeightMin}
          setHeightMax={setHeightMax}
          setRatioMin={setRatioMin}
          setRatioMax={setRatioMax}
          onlyIcon={onlyIcon}
          textPlaceholder={textPlaceholder}
          widthMinLimit={widthMinLimit}
          widthMaxLimit={widthMaxLimit}
          heightMinLimit={heightMinLimit}
          heightMaxLimit={heightMaxLimit}
          ratioMinLimit={ratioMinLimit}
          ratioMaxLimit={ratioMaxLimit}
        />
        <InfiniteGallery response={response} />
        <ScrollTop />
      </div>
      <Footer />
    </div>
  );
}
