package collectors

import (
	"context"
	"time"

	"homer-go/internal/config"
)

type Immich struct{}

func (Immich) Type() string { return "Immich" }

func (Immich) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	headers := map[string]string{}
	if apiKey := stringField(item, "apikey"); apiKey != "" {
		headers["x-api-key"] = apiKey
	}
	var stats struct {
		Photos      int   `json:"photos"`
		Videos      int   `json:"videos"`
		Usage       int64 `json:"usage"`
		UsageByUser []any `json:"usageByUser"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api/server/statistics", Headers: headers}, &stats); err != nil {
		return offlineStatus("Error", err)
	}
	return Status{
		Badges: positiveBadges(
			countBadge("Users", len(stats.UsageByUser), "users", "success"),
			countBadge("Photos", stats.Photos, "photos", "info"),
			countBadge("Videos", stats.Videos, "videos", "warning"),
			Badge{Label: "Usage", Value: humanizeBytes(stats.Usage), State: "usage", Tone: "danger"},
		),
		Updated: time.Now(),
	}
}
