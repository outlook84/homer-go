package collectors

import (
	"context"
	"strings"

	"homer-go/internal/config"
)

type Docuseal struct{}

func (Docuseal) Type() string { return "Docuseal" }

func (Docuseal) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	version, err := collectText(ctx, item, proxy, requestOptions{Path: "version"})
	if err != nil {
		return offlineStatus("Offline", err)
	}
	version = strings.TrimSpace(version)
	return onlineStatus("Version "+version, version)
}
