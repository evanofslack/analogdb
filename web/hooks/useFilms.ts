import { getFilms } from "@app/actions/films";
import { useQuery } from "@tanstack/react-query";
import { FilmsGetRequest } from "analogdb-generated";

export default function useFilms(count: number, enabled: boolean) {
  const params: FilmsGetRequest = {
    sort: "counts",
    pageSize: count,
    includeCounts: true,
    excludeZeroCounts: true,
  };

  return useQuery({
    queryKey: ["films", params],
    queryFn: () => getFilms(params),
    staleTime: Infinity,
    enabled: enabled,
  });
}
