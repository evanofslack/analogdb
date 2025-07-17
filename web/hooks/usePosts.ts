import { postsApi } from "@lib/client";
import { ServerPostResponse } from "analogdb-generated";
import { useQueryState } from "nuqs";
import { useCallback, useEffect, useState } from "react";

type FilterOption = "include" | "exclude" | "only";
type SortOption = "latest" | "oldest" | "random";

interface QueryParams {
  sort: string;
  pageSize: number;
  widthMin: number;
  widthMax: number;
  heightMin: number;
  heightMax: number;
  ratioMin: number;
  ratioMax: number;
  nsfw?: boolean;
  grayscale?: boolean;
  sprocket?: boolean;
  keyword?: string[];
  color?: string;
  minColor?: string;
}

const DEFAULTS = {
  sort: "latest" as SortOption,
  nsfw: "exclude" as FilterOption,
  bw: "exclude" as FilterOption,
  sprocket: "include" as FilterOption,
  color: "",
  text: "",
  widthMin: 600,
  widthMax: 15000,
  heightMin: 400,
  heightMax: 12000,
  ratioMin: 0.3,
  ratioMax: 4.8,
};

const COLOR_MIN_VALUES: Record<string, string> = {
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

async function makeRequest(params: QueryParams): Promise<ServerPostResponse> {
  try {
    const response = await postsApi.postsGet(params);
    return response;
  } catch (error) {
    console.error("API request failed:", error);
    throw error;
  }
}

function buildQueryParams(
  sort: string,
  nsfw: string,
  bw: string,
  sprocket: string,
  text: string,
  color: string,
  width: [number, number],
  height: [number, number],
  ratio: [number, number]
): QueryParams {
  const params: QueryParams = {
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
    params.keyword = [text];
  }

  if (color !== "") {
    params.color = color;
    params.minColor = COLOR_MIN_VALUES[color] || COLOR_MIN_VALUES.default;
  }

  return params;
}

export default function usePostsQuery() {
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

  const [textTemp, setTextTemp] = useState<string>(text || "");
  const [response, setResponse] = useState<ServerPostResponse | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(true);

  const executeQuery = useCallback(async () => {
    setIsLoading(true);

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
