package collectors

import (
	"context"
	"fmt"
	"time"

	"homer-go/internal/config"
)

type Miniflux struct{}

func (Miniflux) Type() string { return "Miniflux" }

func (Miniflux) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	headers := map[string]string{}
	if apiKey := stringField(item, "apikey"); apiKey != "" {
		headers["X-Auth-Token"] = apiKey
	}
	var counters struct {
		Unreads map[string]int `json:"unreads"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "v1/feeds/counters", Headers: headers}, &counters); err != nil {
		return offlineStatus("Error", err)
	}

	unreadFeeds := len(counters.Unreads)
	unreadEntries := 0
	for _, count := range counters.Unreads {
		unreadEntries += count
	}
	if unreadEntries == 0 {
		status := Status{State: "online", Tone: "success", Label: "Online", Updated: time.Now()}
		if stringField(item, "style") != "counter" {
			status.Indicator = "Online"
		}
		return status
	}
	label := fmt.Sprintf("%d unread", unreadEntries)
	if unreadFeeds >= 2 {
		label = fmt.Sprintf("%d unread in %d feeds", unreadEntries, unreadFeeds)
	}
	if stringField(item, "style") == "counter" {
		return Status{
			State:   "unread",
			Tone:    "info",
			Label:   label,
			Badges:  []Badge{countBadge("Unread", unreadEntries, "unread", "info")},
			Updated: time.Now(),
		}
	}
	return Status{
		State:     "unread",
		Tone:      "info",
		Label:     label,
		Indicator: "Unread",
		Updated:   time.Now(),
	}
}
