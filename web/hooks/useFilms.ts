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
  make: null as string | null,
  type: null as string | null,
  speed: null as number | null,
  colortype: null as string | null,
  pageSize: 100,
  includeCounts: true,
  excludeZeroCounts: true,
};

async function makeRequest(
  params: FilmsGetRequest
): Promise<ServerFilmsResponse> {
  try {
    const response = await filmsApi.filmsGet(params);
    // console.log(response);
    return response;
  } catch (error) {
    console.error("API request failed:", error);
    throw error;
  }
}

function buildQueryParams(
  count: number | null,
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

  if (count !== null) params.pageSize = count;
  if (make !== "") params.make = make;
  if (type !== "") params.type = type;
  if (speed !== null) params.speed = speed;
  if (colortype !== "") params.colortype = colortype;

  return params;
}

export default function useFilms(count: number) {
  const [sort, setSort] = useQueryState("film_sort", {
    defaultValue: DEFAULTS.sort,
  });
  const [make, setMake] = useQueryState("film_make", {
    defaultValue: DEFAULTS.make,
  });
  const [type, setType] = useQueryState("film_type", {
    defaultValue: DEFAULTS.type,
  });
  const [speed, setSpeed] = useQueryState(
    "film_speed",
    parseAsInteger.withDefault(DEFAULTS.speed)
  );
  const [colortype, setColortype] = useQueryState("film_colortype", {
    defaultValue: DEFAULTS.colortype,
  });

  const [response, setResponse] = useState<ServerFilmsResponse | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(true);

  const executeQuery = useCallback(async () => {
    setIsLoading(true);

    try {
      const queryParams = buildQueryParams(
        count,
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
