package collectors

import (
	"context"
	"time"

	"homer-go/internal/config"
)

type Glances struct{}

func (Glances) Type() string { return "Glances" }

func (Glances) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	var response struct {
		Load float64 `json:"load"`
		CPU  float64 `json:"cpu"`
		Mem  float64 `json:"mem"`
		Swap float64 `json:"swap"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api/4/quicklook"}, &response); err != nil {
		return offlineStatus("Error", err)
	}
	stats := stringSliceField(item, "stats")
	if len(stats) == 0 {
		stats = []string{"load", "cpu", "mem", "swap"}
	}
	available := map[string]Badge{
		"load": {Label: "Load", Value: formatMetric(response.Load) + "%", State: "load", Tone: "neutral"},
		"cpu":  {Label: "CPU", Value: formatMetric(response.CPU) + "%", State: "cpu", Tone: "info"},
		"mem":  {Label: "Mem", Value: formatMetric(response.Mem) + "%", State: "mem", Tone: "warning"},
		"swap": {Label: "Swap", Value: formatMetric(response.Swap) + "%", State: "swap", Tone: "danger"},
	}
	badges := make([]Badge, 0, len(stats))
	for _, stat := range stats {
		if badge, ok := available[stat]; ok {
			badges = append(badges, badge)
		}
	}
	return Status{
		Badges:  positiveBadges(badges...),
		Updated: time.Now(),
	}
}
