package core

import (
	"context"
	"errors"
)

// Page is one page of a cursor-paginated Boat list endpoint.
type Page[T any] struct {
	Items      []T
	NextCursor string
	HasMore    bool
}

// Each walks pages from fetch, calling yield for every item until the last page, an error, a
// context cancellation or a yield error.
//
// It guards against a malformed server response that claims more pages without a cursor or that
// repeats a cursor, so a caller cannot loop forever.
func Each[T any](
	ctx context.Context,
	fetch func(context.Context, string) (Page[T], error),
	yield func(T) error,
) error {
	if fetch == nil {
		return errors.New("core: nil page fetcher")
	}
	if yield == nil {
		return errors.New("core: nil page yield")
	}

	cursor := ""
	seen := make(map[string]struct{})
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		page, err := fetch(ctx, cursor)
		if err != nil {
			return err
		}
		for _, item := range page.Items {
			if err := yield(item); err != nil {
				return err
			}
		}
		if !page.HasMore {
			return nil
		}
		if page.NextCursor == "" {
			return errors.New("core: page has more items but no next cursor")
		}
		if _, ok := seen[page.NextCursor]; ok {
			return errors.New("core: page cursor repeated")
		}
		seen[page.NextCursor] = struct{}{}
		cursor = page.NextCursor
	}
}
