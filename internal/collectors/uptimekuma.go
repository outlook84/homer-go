package collectors

import (
	"context"
	"fmt"
	"strings"
	"time"

	"homer-go/internal/config"
)

type UptimeKuma struct{}

func (UptimeKuma) Type() string { return "UptimeKuma" }

func (UptimeKuma) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	slug := stringField(item, "slug")
	if slug == "" {
		slug = "default"
	}
	var page struct {
		Incident *struct {
			Title string `json:"title"`
		} `json:"incident"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api/status-page/" + slug}, &page); err != nil {
		return offlineStatus("Error", err)
	}
	var heartbeat struct {
		HeartbeatList map[string][]struct {
			Status int `json:"status"`
		} `json:"heartbeatList"`
		UptimeList map[string]float64 `json:"uptimeList"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api/status-page/heartbeat/" + slug}, &heartbeat); err != nil {
		return offlineStatus("Error", err)
	}
	state := "good"
	tone := "success"
	label := "All Systems Operational"
	if page.Incident != nil {
		state = "bad"
		tone = "danger"
		label = page.Incident.Title
		if label == "" {
			label = "Incident active"
		}
	} else {
		hasUp := false
		for _, beats := range heartbeat.HeartbeatList {
			if len(beats) == 0 {
				continue
			}
			last := beats[len(beats)-1]
			if last.Status == 1 {
				hasUp = true
			} else {
				state = "warn"
				tone = "warning"
				label = "Partially Degraded Service"
			}
		}
		if !hasUp && len(heartbeat.HeartbeatList) > 0 {
			state = "bad"
			tone = "danger"
			label = "Degraded Service"
		}
	}
	uptime := 0.0
	for _, value := range heartbeat.UptimeList {
		uptime += value
	}
	if len(heartbeat.UptimeList) > 0 {
		uptime = uptime / float64(len(heartbeat.UptimeList)) * 100
	}
	return Status{
		State:     state,
		Tone:      tone,
		Label:     label,
		Indicator: fmt.Sprintf("%.1f%%", uptime),
		Detail:    fmt.Sprintf("%.1f%%", uptime),
		URL:       uptimeKumaStatusURL(item, slug),
		Updated:   time.Now(),
	}
}

func uptimeKumaStatusURL(item config.Item, slug string) string {
	if item.URL == "" {
		return ""
	}
	base := strings.TrimRight(item.URL, "/")
	statusPath := "/status/" + slug
	if strings.HasSuffix(base, statusPath) {
		return base
	}
	return base + statusPath
}
