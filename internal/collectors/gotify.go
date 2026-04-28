package collectors

import (
	"context"
	"strconv"
	"time"

	"homer-go/internal/config"
)

type Gotify struct{}

func (Gotify) Type() string { return "Gotify" }

func (Gotify) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	var health struct {
		Health   string `json:"health"`
		Database string `json:"database"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "health"}, &health); err != nil {
		return offlineStatus("Offline", err)
	}

	var messages struct {
		Messages []any `json:"messages"`
	}
	headers := map[string]string{}
	if apiKey := stringField(item, "apikey"); apiKey != "" {
		headers["X-Gotify-Key"] = apiKey
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "message?limit=100", Headers: headers}, &messages); err != nil {
		return offlineStatus("Offline", err)
	}

	state, tone := gotifyState(health.Health, health.Database)
	count := len(messages.Messages)
	label := ""
	if count > 0 {
		if count > 100 {
			label = "100+ messages"
		} else if count == 1 {
			label = "1 message"
		} else {
			label = strconv.Itoa(count) + " messages"
		}
	}
	return Status{State: state, Tone: tone, Label: label, Updated: time.Now()}
}

func gotifyState(statuses ...string) (string, string) {
	for _, status := range statuses {
		if status == "red" {
			return "offline", "danger"
		}
	}
	for _, status := range statuses {
		if status == "orange" {
			return "warning", "warning"
		}
	}
	return "online", "success"
}
