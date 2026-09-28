import { getCameras } from "@app/actions/cameras";
import { useQuery } from "@tanstack/react-query";
import { CamerasGetRequest } from "analogdb-generated";

export default function useCameras(count: number, enabled: boolean) {
  const params: CamerasGetRequest = {
    sort: "counts",
    pageSize: count,
    includeCounts: true,
    excludeZeroCounts: true,
  };

  return useQuery({
    queryKey: ["cameras", params],
    queryFn: () => getCameras(params),
    staleTime: Infinity,
    enabled: enabled,
  });
}
