package api

import (
	"errors"
	"testing"
)

// Tests for the shared cursor-pagination helper (#874). GetProjectItems,
// GetProjectItemsMinimal and GetProjectItemsForBoard previously each carried
// their own copy of the fetch/filter/append/advance loop.

// pageSource builds a fetch func that serves the given pages in order and
// records the cursor it was called with each time.
func pageSource(pages [][]int) (func(*string) ([]int, pageInfo, error), *[]string) {
	var seen []string
	call := 0
	fetch := func(cursor *string) ([]int, pageInfo, error) {
		if cursor == nil {
			seen = append(seen, "<nil>")
		} else {
			seen = append(seen, *cursor)
		}
		page := pages[call]
		info := pageInfo{HasNextPage: call < len(pages)-1}
		if info.HasNextPage {
			info.EndCursor = string(rune('a' + call))
		}
		call++
		return page, info, nil
	}
	return fetch, &seen
}

func TestCollectPages_SinglePage(t *testing.T) {
	fetch, seen := pageSource([][]int{{1, 2, 3}})

	got, err := collectPages(fetch, nil, 0)
	if err != nil {
		t.Fatalf("Expected no error; got %v", err)
	}
	assertInts(t, got, []int{1, 2, 3})
	if len(*seen) != 1 || (*seen)[0] != "<nil>" {
		t.Errorf("Expected one fetch with a nil cursor; got %v", *seen)
	}
}

func TestCollectPages_FollowsCursorAcrossPages(t *testing.T) {
	fetch, seen := pageSource([][]int{{1, 2}, {3, 4}, {5}})

	got, err := collectPages(fetch, nil, 0)
	if err != nil {
		t.Fatalf("Expected no error; got %v", err)
	}
	assertInts(t, got, []int{1, 2, 3, 4, 5})
	want := []string{"<nil>", "a", "b"}
	if len(*seen) != len(want) {
		t.Fatalf("Expected %d fetches; got %d (%v)", len(want), len(*seen), *seen)
	}
	for i := range want {
		if (*seen)[i] != want[i] {
			t.Errorf("Fetch %d: expected cursor %q; got %q", i, want[i], (*seen)[i])
		}
	}
}

func TestCollectPages_KeepFiltersItems(t *testing.T) {
	fetch, _ := pageSource([][]int{{1, 2, 3}, {4, 5, 6}})

	got, err := collectPages(fetch, func(n int) bool { return n%2 == 0 }, 0)
	if err != nil {
		t.Fatalf("Expected no error; got %v", err)
	}
	assertInts(t, got, []int{2, 4, 6})
}

// The limit must truncate *and* stop paging — otherwise a bounded list still
// walks every page of a large project.
func TestCollectPages_LimitTruncatesAndStopsFetching(t *testing.T) {
	fetch, seen := pageSource([][]int{{1, 2}, {3, 4}, {5, 6}})

	got, err := collectPages(fetch, nil, 3)
	if err != nil {
		t.Fatalf("Expected no error; got %v", err)
	}
	assertInts(t, got, []int{1, 2, 3})
	if len(*seen) != 2 {
		t.Errorf("Expected paging to stop after 2 fetches; got %d (%v)", len(*seen), *seen)
	}
}

// Filtered-out items must not count toward the limit.
func TestCollectPages_LimitCountsKeptItemsOnly(t *testing.T) {
	fetch, _ := pageSource([][]int{{1, 2, 3, 4}, {5, 6, 7, 8}})

	got, err := collectPages(fetch, func(n int) bool { return n%2 == 0 }, 3)
	if err != nil {
		t.Fatalf("Expected no error; got %v", err)
	}
	assertInts(t, got, []int{2, 4, 6})
}

func TestCollectPages_NonPositiveLimitIsUnlimited(t *testing.T) {
	for _, limit := range []int{0, -1} {
		fetch, _ := pageSource([][]int{{1, 2}, {3, 4}})
		got, err := collectPages(fetch, nil, limit)
		if err != nil {
			t.Fatalf("limit %d: expected no error; got %v", limit, err)
		}
		assertInts(t, got, []int{1, 2, 3, 4})
	}
}

func TestCollectPages_FetchErrorPropagates(t *testing.T) {
	sentinel := errors.New("boom")
	fetch := func(cursor *string) ([]int, pageInfo, error) {
		return nil, pageInfo{}, sentinel
	}

	got, err := collectPages(fetch, nil, 0)
	if !errors.Is(err, sentinel) {
		t.Errorf("Expected the fetch error to propagate; got %v", err)
	}
	if got != nil {
		t.Errorf("Expected no items on error; got %v", got)
	}
}

func TestCollectPages_EmptyResultIsNil(t *testing.T) {
	fetch, _ := pageSource([][]int{{}})

	got, err := collectPages(fetch, nil, 0)
	if err != nil {
		t.Fatalf("Expected no error; got %v", err)
	}
	if got != nil {
		t.Errorf("Expected nil for an empty result; got %v", got)
	}
}

func assertInts(t *testing.T, got, want []int) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("Expected %d items; got %d (%v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Index %d: expected %d; got %d", i, want[i], got[i])
		}
	}
}
