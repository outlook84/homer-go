package collectors

import (
	"context"
	"time"

	"homer-go/internal/config"
)

type Tautulli struct{}

func (Tautulli) Type() string { return "Tautulli" }

func (Tautulli) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	var response struct {
		Response struct {
			Data struct {
				StreamCount int `json:"stream_count"`
			} `json:"data"`
		} `json:"response"`
	}
	path := "api/v2?apikey=" + stringField(item, "apikey") + "&cmd=get_activity"
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: path}, &response); err != nil {
		return offlineStatus("Error", err)
	}
	return Status{
		Badges:  positiveBadges(countBadge("Playing", response.Response.Data.StreamCount, "playing", "info")),
		Updated: time.Now(),
	}
}
