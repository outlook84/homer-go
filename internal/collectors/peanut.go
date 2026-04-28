package collectors

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"homer-go/internal/config"
)

type PeaNUT struct{}

func (PeaNUT) Type() string { return "PeaNUT" }

func (PeaNUT) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	device := stringField(item, "device")
	var response map[string]any
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api/v1/devices/" + url.PathEscape(device)}, &response); err != nil {
		return offlineStatus("Offline", err)
	}
	code := asCollectorString(response["ups.status"])
	load := asCollectorFloat(response["ups.load"])
	state, tone, label := peanutState(code)
	return Status{
		State:   state,
		Tone:    tone,
		Label:   fmt.Sprintf("%.1f%% UPS Load", load),
		Detail:  label,
		Updated: time.Now(),
	}
}
