package collectors

import (
	"context"

	"homer-go/internal/config"
)

type TruenasScale struct{}

func (TruenasScale) Type() string { return "TruenasScale" }

func (TruenasScale) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	headers := map[string]string{}
	if token := stringField(item, "api_token"); token != "" {
		headers["Authorization"] = "Bearer " + token
	}
	var version string
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api/v2.0/system/version", Headers: headers}, &version); err != nil {
		return offlineStatus("Offline", err)
	}
	return onlineStatus("Version "+version, version)
}
