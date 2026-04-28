package collectors

import (
	"context"
	"fmt"
	"time"

	"homer-go/internal/config"
)

type arrOptions struct {
	QueuePath       string
	MissingPath     string
	LegacyQueueList bool
	LegacyQueueKey  string
	SkipMissing     bool
}

func collectArrStatus(ctx context.Context, item config.Item, proxy config.Proxy, api string, opts arrOptions) Status {
	apiKey := stringField(item, "apikey")
	var health []struct {
		Type string `json:"type"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: fmt.Sprintf("%s/health?apikey=%s", api, apiKey)}, &health); err != nil {
		return offlineStatus("Error", err)
	}
	warnings := 0
	errors := 0
	for _, h := range health {
		switch h.Type {
		case "warning":
			warnings++
		case "error", "errors":
			errors++
		}
	}
	activity := 0
	if opts.LegacyQueueList {
		var queue []map[string]any
		if err := collectJSON(ctx, item, proxy, requestOptions{Path: fmt.Sprintf("%s/%s?apikey=%s", api, opts.QueuePath, apiKey)}, &queue); err != nil {
			return offlineStatus("Error", err)
		}
		for _, entry := range queue {
			if entry[opts.LegacyQueueKey] != nil {
				activity++
			}
		}
	} else {
		var queue struct {
			TotalCount   int `json:"totalCount"`
			TotalRecords int `json:"totalRecords"`
		}
		if err := collectJSON(ctx, item, proxy, requestOptions{Path: fmt.Sprintf("%s/%s?apikey=%s", api, opts.QueuePath, apiKey)}, &queue); err != nil {
			return offlineStatus("Error", err)
		}
		activity = queue.TotalRecords
		if activity == 0 {
			activity = queue.TotalCount
		}
	}
	missing := 0
	if !opts.SkipMissing && opts.MissingPath != "" {
		var missingResponse struct {
			TotalRecords int `json:"totalRecords"`
			Records      []struct {
				Monitored   bool  `json:"monitored"`
				IsAvailable *bool `json:"isAvailable"`
				HasFile     bool  `json:"hasFile"`
			} `json:"records"`
		}
		if err := collectJSON(ctx, item, proxy, requestOptions{Path: fmt.Sprintf("%s/%s?apikey=%s", api, opts.MissingPath, apiKey)}, &missingResponse); err != nil {
			return offlineStatus("Error", err)
		}
		missing = missingResponse.TotalRecords
		if len(missingResponse.Records) > 0 {
			hasAvailability := false
			for _, record := range missingResponse.Records {
				if record.IsAvailable != nil {
					hasAvailability = true
					break
				}
			}
			if hasAvailability {
				missing = 0
			}
			for _, record := range missingResponse.Records {
				if hasAvailability && record.Monitored && record.IsAvailable != nil && *record.IsAvailable && !record.HasFile {
					missing++
				}
			}
		}
	}
	return Status{
		Badges: positiveBadges(
			countBadge("Activity", activity, "activity", "info"),
			countBadge("Missing", missing, "missing", "neutral"),
			countBadge("Warning", warnings, "warnings", "warning"),
			countBadge("Error", errors, "errors", "danger"),
		),
		Updated: time.Now(),
	}
}

func mergeCountBadges(existing []Badge, additions ...Badge) []Badge {
	index := map[string]int{}
	for i, badge := range existing {
		index[badge.Label] = i
	}
	out := append([]Badge{}, existing...)
	for _, badge := range additions {
		if badge.Value == "0" {
			continue
		}
		if i, ok := index[badge.Label]; ok {
			current := asCollectorInt(out[i].Value)
			out[i].Value = fmt.Sprintf("%d", current+asCollectorInt(badge.Value))
			continue
		}
		out = append(out, badge)
	}
	return out
}
