import { getPostsSimilar } from "@app/actions/posts";
import { getSearchSource, searchPosts } from "@app/actions/search";
import type { SearchSource } from "@lib/data/search";
import { retryUnlessLimited, unwrap } from "@lib/rateLimited";
import {
  SearchFlags,
  searchKey,
  searchPageSize,
  similarPageSize,
} from "@lib/searchParams";
import {
  keepPreviousData,
  useInfiniteQuery,
  useQuery,
} from "@tanstack/react-query";
import {
  ServerSearchResponse,
  ServerSimilarPostsResponse,
} from "analogdb-generated";

export default function useSearch(
  q: string | null,
  flags: SearchFlags,
  initialPage?: ServerSearchResponse | null,
  initialKey?: string | null
) {
  const initialData =
    q && initialPage && initialKey === searchKey("q", q, flags)
      ? { pages: [initialPage], pageParams: [undefined] }
      : undefined;

  const query = useInfiniteQuery({
    queryKey: ["search", q, flags],
    queryFn: async ({ pageParam }) =>
      unwrap(
        await searchPosts({
          q,
          pageSize: searchPageSize,
          cursor: pageParam,
          ...flags,
        })
      ),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (response) => response.meta?.nextCursor || undefined,
    initialData,
    placeholderData: keepPreviousData,
    enabled: Boolean(q),
    retry: retryUnlessLimited,
  });

  const pages = query.data?.pages ?? [];

  return {
    pages,
    relatedKeywords: pages[0]?.relatedKeywords ?? [],
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

export function useSimilar(
  id: number | null,
  flags: SearchFlags,
  initialPage?: ServerSimilarPostsResponse | null,
  initialSource?: SearchSource | null,
  initialKey?: string | null
) {
  const matches = id && initialKey === searchKey("similar", id, flags);

  const query = useQuery({
    queryKey: ["similar", id, flags],
    queryFn: async () =>
      unwrap(
        await getPostsSimilar({ id, pageSize: similarPageSize, ...flags })
      ),
    initialData: matches && initialPage ? initialPage : undefined,
    placeholderData: keepPreviousData,
    enabled: Boolean(id),
    retry: retryUnlessLimited,
  });

  const source = useQuery({
    queryKey: ["search-source", id],
    queryFn: async () => unwrap(await getSearchSource(id)),
    initialData:
      initialSource && initialSource.id === id ? initialSource : undefined,
    enabled: Boolean(id),
    retry: retryUnlessLimited,
  });

  return {
    source: source.data ?? null,
    pages: query.data ? [query.data] : [],
    isLoading: query.isPending,
    isError: query.isError,
    error: query.error,
    isPlaceholderData: query.isPlaceholderData,
    hasNextPage: false,
    isFetchingNextPage: false,
    isFetchNextPageError: false,
    fetchNextPage: () => {},
    refetch: query.refetch,
  };
}
