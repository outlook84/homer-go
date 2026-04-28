package collectors

import (
	"context"
	"fmt"

	"homer-go/internal/config"
)

type SpeedtestTracker struct{}

func (SpeedtestTracker) Type() string { return "SpeedtestTracker" }

func (SpeedtestTracker) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	metrics, err := collectSpeedtestTrackerMetrics(ctx, item, proxy, "api/v1/results/latest", true)
	if err != nil {
		metrics, err = collectSpeedtestTrackerMetrics(ctx, item, proxy, "api/speedtest/latest", false)
	}
	if err != nil {
		return offlineStatus("Error", err)
	}
	return onlineStatus(fmt.Sprintf("Down %s Mbit/s | Up %s Mbit/s | Ping %s ms", formatMetric(metrics.Download), formatMetric(metrics.Upload), formatMetric(metrics.Ping)), "")
}

type speedtestTrackerMetrics struct {
	Download     float64 `json:"download"`
	Upload       float64 `json:"upload"`
	Ping         float64 `json:"ping"`
	DownloadBits float64 `json:"download_bits"`
	UploadBits   float64 `json:"upload_bits"`
}

func collectSpeedtestTrackerMetrics(ctx context.Context, item config.Item, proxy config.Proxy, path string, currentAPI bool) (speedtestTrackerMetrics, error) {
	headers := map[string]string{}
	if currentAPI {
		headers["Accept"] = "application/json"
		if apiKey := stringField(item, "apikey"); apiKey != "" {
			headers["Authorization"] = "Bearer " + apiKey
		}
	}
	var response struct {
		speedtestTrackerMetrics
		Data speedtestTrackerMetrics `json:"data"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: path, Headers: headers}, &response); err != nil {
		return speedtestTrackerMetrics{}, err
	}
	metrics := response.speedtestTrackerMetrics
	if metrics.Download == 0 && metrics.Upload == 0 && metrics.Ping == 0 {
		metrics = response.Data
	}
	metrics.Download = speedtestTrackerMbit(metrics.Download, metrics.DownloadBits)
	metrics.Upload = speedtestTrackerMbit(metrics.Upload, metrics.UploadBits)
	return metrics, nil
}

func speedtestTrackerMbit(value, bits float64) float64 {
	if bits > 0 {
		return bits / 1000 / 1000
	}
	if value > 10000 {
		return value * 8 / 1000 / 1000
	}
	return value
}
