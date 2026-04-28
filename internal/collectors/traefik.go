package collectors

import (
	"context"
	"encoding/base64"

	"homer-go/internal/config"
)

type Traefik struct{}

func (Traefik) Type() string { return "Traefik" }

func (Traefik) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	headers := map[string]string{}
	if basicAuth := stringField(item, "basic_auth"); basicAuth != "" {
		headers["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(basicAuth))
	}
	var version struct {
		Version string
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api/version", Headers: headers}, &version); err != nil {
		return offlineStatus("Offline", err)
	}
	return onlineStatus("Version "+version.Version, version.Version)
}
