package collectors

import (
	"context"

	"homer-go/internal/config"
)

type Matrix struct{}

func (Matrix) Type() string { return "Matrix" }

func (Matrix) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	var response struct {
		Server struct {
			Version string `json:"version"`
		} `json:"server"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "_matrix/federation/v1/version"}, &response); err != nil {
		return offlineStatus("Offline", err)
	}
	return onlineStatus("Version "+response.Server.Version, response.Server.Version)
}
