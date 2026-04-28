package collectors

import (
	"context"

	"homer-go/internal/config"
)

type Readarr struct{}

func (Readarr) Type() string { return "Readarr" }

func (Readarr) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	return collectArrStatus(ctx, item, proxy, "api/v1", arrOptions{
		QueuePath:   "queue",
		MissingPath: "wanted/missing",
	})
}
