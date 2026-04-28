package collectors

import (
	"context"
	"time"

	"homer-go/internal/config"
)

type Scrutiny struct{}

func (Scrutiny) Type() string { return "Scrutiny" }

func (Scrutiny) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	var summary struct {
		Data struct {
			Summary map[string]struct {
				Device struct {
					Archived     bool `json:"archived"`
					DeletedAt    any
					DeviceStatus int `json:"device_status"`
				} `json:"device"`
			} `json:"summary"`
		} `json:"data"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api/summary"}, &summary); err != nil {
		return offlineStatus("Error", err)
	}

	passed := 0
	failed := 0
	unknown := 0
	for _, entry := range summary.Data.Summary {
		if entry.Device.Archived || entry.Device.DeletedAt != nil {
			continue
		}
		switch status := entry.Device.DeviceStatus; {
		case status == 0:
			passed++
		case status > 0 && status <= 3:
			failed++
		default:
			unknown++
		}
	}
	return Status{
		Badges: positiveBadges(
			countBadge("Passed", passed, "online", "success"),
			countBadge("Failed", failed, "offline", "danger"),
			countBadge("Unknown", unknown, "unknown", "warning"),
		),
		Updated: time.Now(),
	}
}
