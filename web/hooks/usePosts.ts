import { getPosts } from "@app/actions/posts";
import { postsLimits, postsParsers, toPostsRequest } from "@lib/searchParams";
import { isValidSeed, pickSeed } from "@lib/seed";
import { keepPreviousData, useInfiniteQuery } from "@tanstack/react-query";
import { useQueryStates } from "nuqs";
import { useEffect, useMemo } from "react";

export default function usePosts() {
  const [filters, setFilters] = useQueryStates(postsParsers);

  const needsSeed = filters.sort === "random" && !isValidSeed(filters.seed);
  const request = useMemo(() => toPostsRequest(filters), [filters]);

  useEffect(() => {
    if (needsSeed) {
      setFilters({ seed: pickSeed() }, { history: "replace" });
    }
  }, [needsSeed, setFilters]);

  const query = useInfiniteQuery({
    queryKey: ["posts", request],
    queryFn: ({ pageParam }) => getPosts({ ...request, cursor: pageParam }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (response) => response.meta?.nextCursor || undefined,
    placeholderData: keepPreviousData,
    enabled: !needsSeed,
  });

  const pages = query.data?.pages ?? [];

  return {
    filters,
    setFilters,
    limits: postsLimits,
    pages,
    totalPosts: pages[0]?.meta?.totalPosts ?? 0,
    isLoading: query.isPending,
    isError: query.isError,
    isPlaceholderData: query.isPlaceholderData,
    hasNextPage: query.hasNextPage,
    isFetchingNextPage: query.isFetchingNextPage,
    isFetchNextPageError: query.isFetchNextPageError,
    fetchNextPage: query.fetchNextPage,
    refetch: query.refetch,
  };
}
