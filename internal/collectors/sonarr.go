package collectors

import (
	"context"

	"homer-go/internal/config"
)

type Sonarr struct{}

func (Sonarr) Type() string { return "Sonarr" }

func (Sonarr) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	api := "api/v3"
	legacy := boolField(item, "legacyApi")
	if legacy {
		api = "api"
	}
	return collectArrStatus(ctx, item, proxy, api, arrOptions{
		QueuePath:       "queue",
		MissingPath:     "wanted/missing",
		LegacyQueueList: legacy,
		LegacyQueueKey:  "series",
	})
}
