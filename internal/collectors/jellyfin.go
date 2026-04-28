package collectors

import (
	"context"
	"time"

	"homer-go/internal/config"
)

type Jellyfin struct{}

func (Jellyfin) Type() string { return "Jellyfin" }

func (Jellyfin) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	apiKey := stringField(item, "apikey")
	headers := map[string]string{}
	if apiKey != "" {
		headers["X-Emby-Authorization"] = `MediaBrowser Client="homer-go", Device="homer-go", DeviceId="homer-go", Version="1.0.0", Token="` + apiKey + `"`
		headers["X-Emby-Token"] = apiKey
	}
	var sessions []map[string]any
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "Sessions", Headers: headers}, &sessions); err != nil {
		return offlineStatus("Error", err)
	}
	streams := 0
	for _, session := range sessions {
		if _, ok := session["NowPlayingItem"]; ok {
			streams++
		}
	}
	return Status{
		Badges:  positiveBadges(countBadge("Playing", streams, "playing", "info")),
		Updated: time.Now(),
	}
}
