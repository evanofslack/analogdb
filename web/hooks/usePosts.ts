import { postsApi } from "@lib/client";
import {
  PostsGetRequest,
  PostsGetSortEnum,
  ServerPostResponse,
} from "analogdb-generated";
import { parseAsFloat, parseAsInteger, useQueryState } from "nuqs";
import { useCallback, useEffect, useState } from "react";

type FilterOption = "include" | "exclude" | "only";

const DEFAULTS = {
  sort: "latest" as PostsGetSortEnum,
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
  filmMake: null as string | null,
  filmType: null as string | null,
  cameraMake: null as string | null,
  cameraModel: null as string | null,
};

const COLOR_MIN_VALUES: Record<string, number> = {
  gray: 0.8,
  black: 0.7,
  white: 0.5,
  teal: 0.35,
  olive: 0.35,
  brown: 0.35,
  tan: 0.3,
  navy: 0.25,
  green: 0.25,
  default: 0.15,
};

async function makeRequest(
  params: PostsGetRequest
): Promise<ServerPostResponse> {
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
  ratio: [number, number],
  filmMake: string,
  filmType: string,
  cameraMake: string,
  cameraModel: string
): PostsGetRequest {
  const params: PostsGetRequest = {
    sort: sort as PostsGetSortEnum,
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
    params.color = [color];
    params.minColor = [COLOR_MIN_VALUES[color] || COLOR_MIN_VALUES.default];
  }

  if (filmMake !== "") params.filmMake = filmMake;
  if (filmType !== "") params.filmType = filmType;
  if (cameraMake !== "") params.cameraMake = cameraMake;
  if (cameraModel !== "") params.cameraModel = cameraModel;

  return params;
}

export default function usePosts() {
  const [sort, setSort] = useQueryState("sort", {
    defaultValue: DEFAULTS.sort,
  });
  const [nsfw, setNsfw] = useQueryState("nsfw", {
    defaultValue: DEFAULTS.nsfw,
  });
  const [bw, setBw] = useQueryState("bw", {
    defaultValue: DEFAULTS.bw,
  });
  const [sprocket, setSprocket] = useQueryState("sprocket", {
    defaultValue: DEFAULTS.sprocket,
  });
  const [color, setColor] = useQueryState("color", {
    defaultValue: DEFAULTS.color,
  });
  const [text, setText] = useQueryState("text", {
    defaultValue: DEFAULTS.text,
  });
  const [filmMake, setFilmMake] = useQueryState("film_make", {
    defaultValue: DEFAULTS.filmMake,
  });
  const [filmType, setFilmType] = useQueryState("film_type", {
    defaultValue: DEFAULTS.filmType,
  });
  const [cameraMake, setCameraMake] = useQueryState("camera_make", {
    defaultValue: DEFAULTS.cameraMake,
  });
  const [cameraModel, setCameraModel] = useQueryState("camera_model", {
    defaultValue: DEFAULTS.cameraModel,
  });

  const [widthMin, setWidthMin] = useQueryState(
    "widthMin",
    parseAsInteger.withDefault(DEFAULTS.widthMin)
  );

  const [widthMax, setWidthMax] = useQueryState(
    "widthMax",
    parseAsInteger.withDefault(DEFAULTS.widthMax)
  );
  const [heightMin, setHeightMin] = useQueryState(
    "heightMin",
    parseAsInteger.withDefault(DEFAULTS.heightMin)
  );
  const [heightMax, setHeightMax] = useQueryState(
    "heightMax",
    parseAsInteger.withDefault(DEFAULTS.heightMax)
  );
  const [ratioMin, setRatioMin] = useQueryState(
    "ratioMin",
    parseAsFloat.withDefault(DEFAULTS.ratioMin)
  );
  const [ratioMax, setRatioMax] = useQueryState(
    "ratioMax",
    parseAsFloat.withDefault(DEFAULTS.ratioMax)
  );

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
        [ratioMin, ratioMax],
        filmMake,
        filmType,
        cameraMake,
        cameraModel
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
    filmMake,
    filmType,
    cameraMake,
    cameraModel,
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
    filmMake,
    filmType,
    cameraMake,
    cameraModel,
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
      filmMake,
      filmType,
      cameraMake,
      cameraModel,
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
      setFilmMake,
      setFilmType,
      setCameraMake,
      setCameraModel,
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
