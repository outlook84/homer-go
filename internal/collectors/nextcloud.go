package collectors

import (
	"context"

	"homer-go/internal/config"
)

type Nextcloud struct{}

func (Nextcloud) Type() string { return "Nextcloud" }

func (Nextcloud) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	var response struct {
		VersionString string `json:"versionstring"`
		Maintenance   bool   `json:"maintenance"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "status.php"}, &response); err != nil {
		return offlineStatus("Offline", err)
	}
	status := onlineStatus("Version "+response.VersionString, response.VersionString)
	status.Indicator = "online"
	if response.Maintenance {
		status.State = "maintenance"
		status.Tone = "warning"
		status.Indicator = "maintenance"
	}
	return status
}
