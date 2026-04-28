package collectors

import (
	"context"

	"homer-go/internal/config"
)

type NetAlertx struct{}

func (NetAlertx) Type() string { return "NetAlertx" }

func (NetAlertx) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	headers := map[string]string{}
	if apiKey := stringField(item, "apikey"); apiKey != "" {
		headers["Authorization"] = "Bearer " + apiKey
	}
	var response any
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "devices/totals", Headers: headers}, &response); err != nil {
		return offlineStatus("Error", err)
	}
	total, connected, newDevices, down := netAlertTotals(response)
	return deviceTotalsStatus(total, connected, newDevices, down)
}
