import { getPosts } from "@app/actions/posts";
import { retryUnlessLimited, unwrap } from "@lib/rateLimited";
import {
  PostsFilters,
  postsLimits,
  postsParsers,
  toPostsRequest,
} from "@lib/searchParams";
import { isValidSeed, pickSeed } from "@lib/seed";
import { keepPreviousData, useInfiniteQuery } from "@tanstack/react-query";
import { ServerPostResponse } from "analogdb-generated";
import { useQueryStates } from "nuqs";
import { useEffect, useMemo } from "react";

type UsePostsOptions = {
  parsers?: Partial<typeof postsParsers>;
  fixed?: PostsFilters;
};

export default function usePosts(
  initialPage?: ServerPostResponse | null,
  initialFilters?: PostsFilters,
  { parsers = postsParsers, fixed }: UsePostsOptions = {}
) {
  const [queryFilters, setFilters] = useQueryStates(parsers);
  const urlFilters = useMemo(
    () => (fixed ? { ...fixed, ...queryFilters } : queryFilters),
    [fixed, queryFilters]
  ) as PostsFilters;

  const missingSeed =
    urlFilters.sort === "random" && !isValidSeed(urlFilters.seed);
  const serverSeed =
    initialFilters?.sort === "random" && isValidSeed(initialFilters.seed)
      ? initialFilters.seed
      : null;

  const filters = useMemo(
    () =>
      missingSeed && serverSeed
        ? { ...urlFilters, seed: serverSeed }
        : urlFilters,
    [missingSeed, serverSeed, urlFilters]
  );

  const needsSeed = filters.sort === "random" && !isValidSeed(filters.seed);
  const request = useMemo(() => toPostsRequest(filters), [filters]);

  const initialKey = useMemo(
    () => initialFilters && JSON.stringify(toPostsRequest(initialFilters)),
    [initialFilters]
  );
  const initialData =
    initialPage && initialKey === JSON.stringify(request)
      ? { pages: [initialPage], pageParams: [undefined] }
      : undefined;

  useEffect(() => {
    if (missingSeed) {
      setFilters({ seed: serverSeed ?? pickSeed() }, { history: "replace" });
    }
  }, [missingSeed, serverSeed, setFilters]);

  const query = useInfiniteQuery({
    queryKey: ["posts", request],
    queryFn: async ({ pageParam }) =>
      unwrap(await getPosts({ ...request, cursor: pageParam })),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (response) => response.meta?.nextCursor || undefined,
    initialData,
    placeholderData: keepPreviousData,
    enabled: !needsSeed,
    retry: retryUnlessLimited,
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
    error: query.error,
    isPlaceholderData: query.isPlaceholderData,
    hasNextPage: query.hasNextPage,
    isFetchingNextPage: query.isFetchingNextPage,
    isFetchNextPageError: query.isFetchNextPageError,
    fetchNextPage: query.fetchNextPage,
    refetch: query.refetch,
  };
}
