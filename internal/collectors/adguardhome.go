package collectors

import (
	"context"
	"fmt"
	"time"

	"homer-go/internal/config"
)

type AdGuardHome struct{}

func (AdGuardHome) Type() string { return "AdGuardHome" }

func (AdGuardHome) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	headers := adGuardHomeHeaders(item)
	var status struct {
		ProtectionEnabled bool `json:"protection_enabled"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "control/status", Headers: headers}, &status); err != nil {
		return Status{State: "unknown", Tone: "warning", Label: "Unknown", Detail: err.Error(), Indicator: "unknown", Updated: time.Now()}
	}

	state := "disabled"
	tone := "danger"
	if status.ProtectionEnabled {
		state = "enabled"
		tone = "success"
	}
	out := Status{State: state, Tone: tone, Detail: state, Indicator: state, Updated: time.Now()}
	if item.Subtitle != "" {
		return out
	}

	var stats struct {
		NumBlockedFiltering float64 `json:"num_blocked_filtering"`
		NumDNSQueries       float64 `json:"num_dns_queries"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "control/stats", Headers: headers}, &stats); err != nil {
		return out
	}
	if stats.NumDNSQueries > 0 {
		out.Label = fmt.Sprintf("%.2f%% blocked", stats.NumBlockedFiltering*100/stats.NumDNSQueries)
	}
	return out
}

func adGuardHomeHeaders(item config.Item) map[string]string {
	username := stringField(item, "username")
	password := stringField(item, "password")
	if username == "" && password == "" {
		return nil
	}
	return map[string]string{"Authorization": basicAuth(username + ":" + password)}
}
