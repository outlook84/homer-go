package collectors

import (
	"context"

	"homer-go/internal/config"
)

type Lidarr struct{}

func (Lidarr) Type() string { return "Lidarr" }

func (Lidarr) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	return collectArrStatus(ctx, item, proxy, "api/v1", arrOptions{
		QueuePath:   "queue/status",
		MissingPath: "wanted/missing",
	})
}
