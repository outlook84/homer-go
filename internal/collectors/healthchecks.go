package collectors

import (
	"context"
	"strings"
	"time"

	"homer-go/internal/config"
)

type Healthchecks struct{}

func (Healthchecks) Type() string { return "Healthchecks" }

func (Healthchecks) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	apiKey := stringField(item, "apikey")
	if apiKey == "" {
		return offlineStatus("Missing API key", nil)
	}
	var response struct {
		Checks []struct {
			Status string `json:"status"`
		} `json:"checks"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{
		Path:    "api/v1/checks/",
		Headers: map[string]string{"X-Api-Key": apiKey},
	}, &response); err != nil {
		return offlineStatus("Error", err)
	}

	up := 0
	down := 0
	grace := 0
	for _, check := range response.Checks {
		switch strings.ToLower(check.Status) {
		case "up":
			up++
		case "down":
			down++
		case "grace":
			grace++
		}
	}
	return Status{
		Badges: positiveBadges(
			countBadge("Up", up, "up", "success"),
			countBadge("Down", down, "down", "danger"),
			countBadge("Grace", grace, "grace", "warning"),
		),
		Updated: time.Now(),
	}
}
