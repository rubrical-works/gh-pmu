package api

// Shared cursor pagination for ProjectV2 item connections (#874).
//
// GetProjectItems, GetProjectItemsMinimal and GetProjectItemsForBoard each
// carried their own copy of the same loop: fetch a page, filter it, append the
// survivors, stop when the connection reports no next page, otherwise advance
// the cursor. Only the item type, the filter predicate and the presence of a
// limit differed.

// collectPages walks a cursor-paginated connection, appending every item for
// which keep returns true. A nil keep accepts everything.
//
// limit caps the number of *kept* items; items rejected by keep do not count
// toward it. A limit of zero or less means unlimited. Once the limit is
// reached, collectPages returns immediately without fetching further pages.
func collectPages[T any](
	fetch func(cursor *string) ([]T, pageInfo, error),
	keep func(T) bool,
	limit int,
) ([]T, error) {
	var all []T
	var cursor *string

	for {
		items, info, err := fetch(cursor)
		if err != nil {
			return nil, err
		}

		for _, item := range items {
			if keep != nil && !keep(item) {
				continue
			}
			all = append(all, item)
			if limit > 0 && len(all) >= limit {
				return all[:limit], nil
			}
		}

		if !info.HasNextPage {
			return all, nil
		}
		cursor = &info.EndCursor
	}
}
