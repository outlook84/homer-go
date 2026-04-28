package collectors

import (
	"context"
	"net/url"
	"time"

	"homer-go/internal/config"
)

type Mylar struct{}

func (Mylar) Type() string { return "Mylar" }

func (Mylar) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	apiKey := url.QueryEscape(stringField(item, "apikey"))
	var upcoming []any
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api?cmd=getUpcoming&apikey=" + apiKey}, &upcoming); err != nil {
		return offlineStatus("Error", err)
	}
	var wanted struct {
		Issues  []any `json:"issues"`
		Annuals []any `json:"annuals"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api?cmd=getWanted&apikey=" + apiKey}, &wanted); err != nil {
		return offlineStatus("Error", err)
	}
	return Status{
		Badges: positiveBadges(
			countBadge("Wanted", len(wanted.Issues)+len(wanted.Annuals), "wanted", "info"),
			countBadge("Upcoming", len(upcoming), "upcoming", "neutral"),
		),
		Updated: time.Now(),
	}
}
