package collectors

import (
	"context"

	"homer-go/internal/config"
)

type Wallabag struct{}

func (Wallabag) Type() string { return "Wallabag" }

func (Wallabag) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	var version string
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api/version"}, &version); err != nil {
		return offlineStatus("Offline", err)
	}
	return onlineStatus("Version "+version, version)
}
