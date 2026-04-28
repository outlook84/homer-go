package collectors

import (
	"context"
	"fmt"

	"homer-go/internal/config"
)

type Radarr struct{}

func (Radarr) Type() string { return "Radarr" }

func (Radarr) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	api := "api/v3"
	if boolField(item, "legacyApi") {
		api = "api"
		return collectArrStatus(ctx, item, proxy, api, arrOptions{
			QueuePath:       "queue",
			LegacyQueueList: true,
			LegacyQueueKey:  "movie",
			SkipMissing:     true,
		})
	}
	status := collectArrStatus(ctx, item, proxy, api, arrOptions{
		QueuePath: "queue",
	})
	if status.State == "offline" {
		return status
	}
	missing, err := radarrMissing(ctx, item, proxy, api)
	if err != nil {
		return status
	}
	status.Badges = mergeCountBadges(status.Badges, countBadge("Missing", missing, "missing", "neutral"))
	var details []struct {
		TrackedDownloadStatus string `json:"trackedDownloadStatus"`
		TrackedDownloadStaus  string `json:"trackedDownloadStaus"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: fmt.Sprintf("%s/queue/details?apikey=%s", api, stringField(item, "apikey"))}, &details); err != nil {
		return status
	}
	warnings := 0
	errors := 0
	for _, detail := range details {
		if detail.TrackedDownloadStatus == "warning" {
			warnings++
		}
		if detail.TrackedDownloadStatus == "error" || detail.TrackedDownloadStaus == "error" {
			errors++
		}
	}
	status.Badges = mergeCountBadges(status.Badges,
		countBadge("Warning", warnings, "warnings", "warning"),
		countBadge("Error", errors, "errors", "danger"),
	)
	return status
}

func radarrMissing(ctx context.Context, item config.Item, proxy config.Proxy, api string) (int, error) {
	apiKey := stringField(item, "apikey")
	var overview struct {
		TotalRecords int `json:"totalRecords"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: fmt.Sprintf("%s/wanted/missing?pageSize=1&apikey=%s", api, apiKey)}, &overview); err != nil {
		return 0, err
	}
	if overview.TotalRecords == 0 {
		return 0, nil
	}
	var movies struct {
		Records []struct {
			Monitored   bool `json:"monitored"`
			IsAvailable bool `json:"isAvailable"`
			HasFile     bool `json:"hasFile"`
		} `json:"records"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: fmt.Sprintf("%s/wanted/missing?pageSize=%d&apikey=%s", api, overview.TotalRecords, apiKey)}, &movies); err != nil {
		return 0, err
	}
	missing := 0
	for _, movie := range movies.Records {
		if movie.Monitored && movie.IsAvailable && !movie.HasFile {
			missing++
		}
	}
	return missing, nil
}
