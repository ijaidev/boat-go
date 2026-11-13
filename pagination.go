package boat

import (
	"context"

	"github.com/ijaidev/boat-go/core"
)

// Each walks every page from fetch and yields each item. It stops on the first error, on context
// cancellation, or when the server reports the last page.
func Each[T any](
	ctx context.Context,
	fetch func(context.Context, string) (core.Page[T], error),
	yield func(T) error,
) error {
	return core.Each(ctx, fetch, yield)
}
