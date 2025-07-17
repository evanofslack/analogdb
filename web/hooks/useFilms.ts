import { filmsApi } from "@lib/client";
import {
  FilmsGetRequest,
  FilmsGetSortEnum,
  ServerFilmsResponse,
} from "analogdb-generated";
import { parseAsInteger, useQueryState } from "nuqs";
import { useCallback, useEffect, useState } from "react";

const DEFAULTS = {
  sort: "counts" as FilmsGetSortEnum,
  make: "",
  type: "",
  speed: null as number | null,
  colortype: "",
  pageSize: 100,
  includeCounts: true,
  excludeZeroCounts: true,
};

async function makeRequest(
  params: FilmsGetRequest
): Promise<ServerFilmsResponse> {
  try {
    const response = await filmsApi.filmsGet(params);
    return response;
  } catch (error) {
    console.error("API request failed:", error);
    throw error;
  }
}

function buildQueryParams(
  sort: string,
  make: string,
  type: string,
  speed: number | null,
  colortype: string
): FilmsGetRequest {
  const params: FilmsGetRequest = {
    sort: sort as FilmsGetSortEnum,
    pageSize: DEFAULTS.pageSize,
    includeCounts: DEFAULTS.includeCounts,
    excludeZeroCounts: DEFAULTS.excludeZeroCounts,
  };

  if (make !== "") params.make = make;
  if (type !== "") params.type = type;
  if (speed !== null) params.speed = speed;
  if (colortype !== "") params.colortype = colortype;

  return params;
}

export default function useFilms() {
  const [sort, setSort] = useQueryState("sort", {
    defaultValue: DEFAULTS.sort,
  });
  const [make, setMake] = useQueryState("make", {
    defaultValue: DEFAULTS.make,
  });
  const [type, setType] = useQueryState("type", {
    defaultValue: DEFAULTS.type,
  });
  const [speed, setSpeed] = useQueryState(
    "speed",
    parseAsInteger.withDefault(DEFAULTS.speed)
  );
  const [colortype, setColortype] = useQueryState("colortype", {
    defaultValue: DEFAULTS.colortype,
  });

  const [response, setResponse] = useState<ServerFilmsResponse | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(true);

  const executeQuery = useCallback(async () => {
    setIsLoading(true);

    try {
      const queryParams = buildQueryParams(
        sort || DEFAULTS.sort,
        make || DEFAULTS.make,
        type || DEFAULTS.type,
        speed,
        colortype || DEFAULTS.colortype
      );
      const data = await makeRequest(queryParams);
      setResponse(data);
    } catch (error) {
      console.error("Failed to fetch films:", error);
    } finally {
      setIsLoading(false);
    }
  }, [sort, make, type, speed, colortype]);

  useEffect(() => {
    executeQuery();
  }, [executeQuery]);

  return {
    response,
    isLoading,
    filters: {
      sort,
      make,
      type,
      speed,
      colortype,
    },
    setters: {
      setSort,
      setMake,
      setType,
      setSpeed,
      setColortype,
    },
    executeQuery,
  };
}
