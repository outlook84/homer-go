package collectors

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"homer-go/internal/config"
)

type SABnzbd struct{}

func (SABnzbd) Type() string { return "SABnzbd" }

func (SABnzbd) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	var response struct {
		Queue struct {
			NoOfSlots int    `json:"noofslots"`
			KBPerSec  any    `json:"kbpersec"`
			Speed     string `json:"speed"`
		} `json:"queue"`
	}
	path := "api?output=json&apikey=" + stringField(item, "apikey") + "&mode=queue"
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: path}, &response); err != nil {
		return offlineStatus("Error", err)
	}
	speed := sabnzbdSpeedKB(response.Queue.KBPerSec, response.Queue.Speed)
	return Status{
		Label: fmt.Sprintf("Down %s", humanizeRate(speed, []string{"KB", "MB", "GB"})),
		Badges: positiveBadges(
			countBadge("Downloads", response.Queue.NoOfSlots, "downloading", "info"),
		),
		Updated: time.Now(),
	}
}

func sabnzbdSpeedKB(kbPerSec any, speed string) float64 {
	if kb := asCollectorFloat(kbPerSec); kb > 0 {
		return kb
	}
	fields := strings.Fields(speed)
	if len(fields) == 0 {
		return 0
	}
	value, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}
	if len(fields) == 1 {
		return value * 1024
	}
	switch strings.TrimSuffix(strings.ToUpper(fields[1]), "/S") {
	case "B":
		return value / 1000
	case "KB", "KIB":
		return value
	case "MB", "MIB":
		return value * 1000
	case "GB", "GIB":
		return value * 1000 * 1000
	default:
		return value * 1024
	}
}
