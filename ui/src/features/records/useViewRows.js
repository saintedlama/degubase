import { ref } from "vue";
import { useRowPagination } from "./useRowPagination.js";
import { useTableEvents } from "./useTableEvents.js";

export function useViewRows(wrapper, extraFilters = () => [], skipFn = () => false) {
  const searchDebounced = ref("");

  const { rows, total, hasMore, loadingMore, loadMore, loadRows, rowEvents } =
    useRowPagination(
      () => wrapper()?.workspaceCode,
      () => wrapper()?.tableCode,
      () => {
        const cfgFilters = wrapper()?.viewConfig?.filters ?? [];
        const extra = extraFilters() ?? [];
        return extra.length > 0 ? [...cfgFilters, ...extra] : cfgFilters;
      },
      () => wrapper()?.viewConfig?.sort ?? [],
      () => searchDebounced.value,
      () => !skipFn(),
    );

  useTableEvents(
    () => wrapper()?.workspaceCode,
    () => wrapper()?.tableCode,
    rowEvents,
  );

  return {
    rows,
    total,
    hasMore,
    loadingMore,
    loadMore,
    loadRows,
    searchDebounced,
  };
}
