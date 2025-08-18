import { getCameras } from "@app/actions/cameras";
import {
  CamerasGetRequest,
  CamerasGetSortEnum,
  ServerCamerasResponse,
} from "analogdb-generated";
import { useQueryState } from "nuqs";
import { useCallback, useEffect, useState } from "react";

const DEFAULTS = {
  sort: "counts" as CamerasGetSortEnum,
  make: null as string | null,
  model: null as string | null,
  pageSize: 100,
  includeCounts: true,
  excludeZeroCounts: true,
};

function buildQueryParams(
  count: number | null,
  sort: string,
  make: string,
  model: string
): CamerasGetRequest {
  const params: CamerasGetRequest = {
    sort: sort as CamerasGetSortEnum,
    pageSize: DEFAULTS.pageSize,
    includeCounts: DEFAULTS.includeCounts,
    excludeZeroCounts: DEFAULTS.excludeZeroCounts,
  };

  if (count !== null) params.pageSize = count;
  if (make !== "") params.make = make;
  if (model !== "") params.model = model;
  return params;
}

export default function useCameras(count: number) {
  const [sort, setSort] = useQueryState("camera_sort", {
    defaultValue: DEFAULTS.sort,
  });
  const [make, setMake] = useState(DEFAULTS.make);
  const [model, setModel] = useState(DEFAULTS.model);

  const [response, setResponse] = useState<ServerCamerasResponse | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(true);

  const executeQuery = useCallback(async () => {
    setIsLoading(true);

    try {
      const queryParams = buildQueryParams(
        count,
        sort || DEFAULTS.sort,
        make || DEFAULTS.make,
        model || DEFAULTS.model
      );
      const data = await getCameras(queryParams);
      setResponse(data);
    } catch (error) {
      console.error("Failed to fetch cameras:", error);
    } finally {
      setIsLoading(false);
    }
  }, [sort, make, model]);

  useEffect(() => {
    executeQuery();
  }, [executeQuery]);

  return {
    response,
    isLoading,
    filters: {
      sort,
      make,
      model,
    },
    setters: {
      setSort,
      setMake,
      setModel,
    },
    executeQuery,
  };
}
