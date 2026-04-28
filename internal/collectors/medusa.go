package collectors

import (
	"context"
	"time"

	"homer-go/internal/config"
)

type Medusa struct{}

func (Medusa) Type() string { return "Medusa" }

func (Medusa) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	headers := map[string]string{}
	if apiKey := stringField(item, "apikey"); apiKey != "" {
		headers["X-Api-Key"] = apiKey
	}
	var response struct {
		System struct {
			News struct {
				Unread int `json:"unread"`
			} `json:"news"`
		} `json:"system"`
		Main struct {
			Logs struct {
				NumWarnings int `json:"numWarnings"`
				NumErrors   int `json:"numErrors"`
			} `json:"logs"`
		} `json:"main"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api/v2/config", Headers: headers}, &response); err != nil {
		return offlineStatus("Error", err)
	}
	return Status{
		Badges: positiveBadges(
			countBadge("News", response.System.News.Unread, "news", "neutral"),
			countBadge("Warning", response.Main.Logs.NumWarnings, "warnings", "warning"),
			countBadge("Error", response.Main.Logs.NumErrors, "errors", "danger"),
		),
		Updated: time.Now(),
	}
}
