package collectors

import (
	"context"

	"homer-go/internal/config"
)

type Gitea struct{}

func (Gitea) Type() string { return "Gitea" }

func (Gitea) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	var swagger struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "swagger.v1.json"}, &swagger); err != nil {
		return offlineStatus("Offline", err)
	}
	return onlineStatus("Version "+swagger.Info.Version, swagger.Info.Version)
}
