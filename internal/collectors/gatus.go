package collectors

import (
	"context"
	"fmt"
	"math"
	"time"

	"homer-go/internal/config"
)

type Gatus struct{}

func (Gatus) Type() string { return "Gatus" }

func (Gatus) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	var endpoints []struct {
		Group   string `json:"group"`
		Results []struct {
			Success  bool  `json:"success"`
			Duration int64 `json:"duration"`
		} `json:"results"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api/v1/endpoints/statuses"}, &endpoints); err != nil {
		return offlineStatus("Error", err)
	}
	groups := stringSet(stringSliceField(item, "groups"))
	total := 0
	up := 0
	totalDuration := float64(0)
	totalResults := 0
	hideAverages := boolField(item, "hideaverages")
	for _, endpoint := range endpoints {
		if len(groups) > 0 && !groups[endpoint.Group] {
			continue
		}
		total++
		if len(endpoint.Results) > 0 && endpoint.Results[len(endpoint.Results)-1].Success {
			up++
		}
		if hideAverages {
			continue
		}
		for _, result := range endpoint.Results {
			totalDuration += float64(result.Duration) / 1000000
			totalResults++
		}
	}
	down := total - up
	percentage := 0
	if total > 0 {
		percentage = int(math.Round(float64(up) / float64(total) * 100))
	}
	state, tone := gatusState(up, down, total)
	label := fmt.Sprintf("%d/%d up", up, total)
	if !hideAverages && totalResults > 0 {
		label += fmt.Sprintf(" | %.2f ms avg.", totalDuration/float64(totalResults))
	}
	return Status{
		State:     state,
		Tone:      tone,
		Label:     label,
		Detail:    fmt.Sprintf("%d%%", percentage),
		Indicator: fmt.Sprintf("%d%%", percentage),
		Updated:   time.Now(),
	}
}
