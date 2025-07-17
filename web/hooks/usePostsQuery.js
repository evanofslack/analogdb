import { postsApi } from "@lib/client.js";
import { useQueryState } from "nuqs";
import { useCallback, useEffect, useRef, useState } from "react";

const DEFAULTS = {
  sort: "latest",
  nsfw: "exclude",
  bw: "exclude",
  sprocket: "include",
  color: "",
  text: "",
  widthMin: 600,
  widthMax: 15000,
  heightMin: 400,
  heightMax: 12000,
  ratioMin: 0.3,
  ratioMax: 4.8,
};

const COLOR_MIN_VALUES = {
  gray: "0.8",
  black: "0.7",
  white: "0.50",
  teal: "0.35",
  olive: "0.35",
  brown: "0.35",
  tan: "0.30",
  navy: "0.25",
  green: "0.25",
  default: "0.15",
};

async function makeRequest(params) {
  try {
    const response = await postsApi.postsGet(params);
    return response;
  } catch (error) {
    console.error("API request failed:", error);
    throw error;
  }
}

function buildQueryParams(
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
  const params = {
    sort: sort,
    pageSize: 100,
    widthMin: width[0],
    widthMax: width[1],
    heightMin: height[0],
    heightMax: height[1],
    ratioMin: ratio[0],
    ratioMax: ratio[1],
  };

  if (nsfw === "exclude") params.nsfw = false;
  if (nsfw === "only") params.nsfw = true;

  if (bw === "exclude") params.grayscale = false;
  if (bw === "only") params.grayscale = true;

  if (sprocket === "exclude") params.sprocket = false;
  if (sprocket === "only") params.sprocket = true;

  if (text !== "") {
    params.keyword = text.split(/[ ,]+/).filter(Boolean);
  }

  if (color !== "") {
    params.color = color;
    params.minColor = COLOR_MIN_VALUES[color] || COLOR_MIN_VALUES.default;
  }

  return params;
}

export default function usePostsQuery(initialData) {
  const [sort, setSort] = useQueryState("sort", {
    history: "push",
    defaultValue: DEFAULTS.sort,
    shallow: false,
  });
  const [nsfw, setNsfw] = useQueryState("nsfw", {
    history: "push",
    defaultValue: DEFAULTS.nsfw,
    shallow: false,
  });
  const [bw, setBw] = useQueryState("bw", {
    history: "push",
    defaultValue: DEFAULTS.bw,
    shallow: false,
  });
  const [sprocket, setSprocket] = useQueryState("sprocket", {
    history: "push",
    defaultValue: DEFAULTS.sprocket,
    shallow: false,
  });
  const [color, setColor] = useQueryState("color", {
    defaultValue: DEFAULTS.color,
    shallow: false,
  });
  const [text, setText] = useQueryState("text", {
    defaultValue: DEFAULTS.text,
    shallow: false,
  });

  const [widthMin, setWidthMin] = useQueryState("widthMin", {
    defaultValue: DEFAULTS.widthMin,
    shallow: false,
  });
  const [widthMax, setWidthMax] = useQueryState("widthMax", {
    defaultValue: DEFAULTS.widthMax,
    shallow: false,
  });
  const [heightMin, setHeightMin] = useQueryState("heightMin", {
    defaultValue: DEFAULTS.heightMin,
    shallow: false,
  });
  const [heightMax, setHeightMax] = useQueryState("heightMax", {
    defaultValue: DEFAULTS.heightMax,
    shallow: false,
  });
  const [ratioMin, setRatioMin] = useQueryState("ratioMin", {
    defaultValue: DEFAULTS.ratioMin,
    shallow: false,
  });
  const [ratioMax, setRatioMax] = useQueryState("ratioMax", {
    defaultValue: DEFAULTS.ratioMax,
    shallow: false,
  });

  const [textTemp, setTextTemp] = useState(text || "");
  const [response, setResponse] = useState(initialData);
  const [isLoading, setIsLoading] = useState(false);
  const isInitialLoad = useRef(true);

  const executeQuery = useCallback(async () => {
    setIsLoading(true);

    // Handle textTemp -> text conversion
    if (textTemp === DEFAULTS.text) {
      setText(null);
    } else {
      setText(textTemp);
    }

    const currentText = textTemp === DEFAULTS.text ? DEFAULTS.text : textTemp;

    try {
      const queryParams = buildQueryParams(
        sort || DEFAULTS.sort,
        nsfw || DEFAULTS.nsfw,
        bw || DEFAULTS.bw,
        sprocket || DEFAULTS.sprocket,
        currentText,
        color || DEFAULTS.color,
        [widthMin, widthMax],
        [heightMin, heightMax],
        [ratioMin, ratioMax]
      );
      const data = await makeRequest(queryParams);
      setResponse(data);
    } catch (error) {
      console.error("Failed to fetch posts:", error);
    } finally {
      setIsLoading(false);
    }
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
    setText,
  ]);

  useEffect(() => {
    if (isInitialLoad.current) {
      const isUsingDefaults =
        (sort || DEFAULTS.sort) === DEFAULTS.sort &&
        (nsfw || DEFAULTS.nsfw) === DEFAULTS.nsfw &&
        (bw || DEFAULTS.bw) === DEFAULTS.bw &&
        (sprocket || DEFAULTS.sprocket) === DEFAULTS.sprocket &&
        (text || DEFAULTS.text) === DEFAULTS.text &&
        (color || DEFAULTS.color) === DEFAULTS.color;

      if (isUsingDefaults) {
        isInitialLoad.current = false;
        return;
      }
    }

    isInitialLoad.current = false;
    executeQuery();
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
    executeQuery,
  ]);

  return {
    response,
    isLoading,
    filters: {
      sort,
      nsfw,
      bw,
      sprocket,
      color,
      text,
      textTemp,
      widthMin,
      widthMax,
      heightMin,
      heightMax,
      ratioMin,
      ratioMax,
    },
    setters: {
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
    },
    executeQuery,
    limits: {
      widthMin: DEFAULTS.widthMin,
      widthMax: DEFAULTS.widthMax,
      heightMin: DEFAULTS.heightMin,
      heightMax: DEFAULTS.heightMax,
      ratioMin: DEFAULTS.ratioMin,
      ratioMax: DEFAULTS.ratioMax,
    },
  };
}
