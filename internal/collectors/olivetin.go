package collectors

import (
	"context"

	"homer-go/internal/config"
)

type Olivetin struct{}

func (Olivetin) Type() string { return "Olivetin" }

func (Olivetin) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	var response struct {
		CurrentVersion string `json:"CurrentVersion"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "webUiSettings.json"}, &response); err != nil {
		return offlineStatus("Offline", err)
	}
	return onlineStatus("Version "+response.CurrentVersion, response.CurrentVersion)
}
