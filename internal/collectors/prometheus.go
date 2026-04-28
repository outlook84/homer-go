package collectors

import (
	"context"
	"strconv"
	"time"

	"homer-go/internal/config"
)

type Prometheus struct{}

func (Prometheus) Type() string { return "Prometheus" }

func (Prometheus) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	var response struct {
		Data struct {
			Alerts []struct {
				State string `json:"state"`
			} `json:"alerts"`
		} `json:"data"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api/v1/alerts"}, &response); err != nil {
		return offlineStatus("Error", err)
	}

	firing := 0
	pending := 0
	inactive := 0
	for _, alert := range response.Data.Alerts {
		switch alert.State {
		case "firing":
			firing++
		case "pending":
			pending++
		case "inactive":
			inactive++
		}
	}
	level := "inactive"
	count := inactive
	tone := "success"
	if firing > 0 {
		level = "firing"
		count = firing
		tone = "danger"
	} else if pending > 0 {
		level = "pending"
		count = pending
		tone = "warning"
	}
	return Status{
		State:     level,
		Tone:      tone,
		Label:     strconv.Itoa(count) + " " + level + " alerts",
		Indicator: strconv.Itoa(count),
		Updated:   time.Now(),
	}
}
