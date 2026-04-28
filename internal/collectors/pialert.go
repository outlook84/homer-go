package collectors

import (
	"context"

	"homer-go/internal/config"
)

type PiAlert struct{}

func (PiAlert) Type() string { return "PiAlert" }

func (PiAlert) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	var totals []int
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "php/server/devices.php?action=getDevicesTotals"}, &totals); err != nil {
		return offlineStatus("Error", err)
	}
	return deviceTotalsStatus(totalAt(totals, 0), totalAt(totals, 1), totalAt(totals, 3), totalAt(totals, 4))
}
