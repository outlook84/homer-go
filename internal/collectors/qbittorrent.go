package collectors

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"homer-go/internal/config"
)

type QBittorrent struct{}

func (QBittorrent) Type() string { return "qBittorrent" }

func (QBittorrent) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	headers, err := qBittorrentHeaders(ctx, item, proxy)
	if err != nil {
		return offlineStatus("Error", err)
	}
	var torrents []struct{}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api/v2/torrents/info", Headers: headers}, &torrents); err != nil {
		return offlineStatus("Error", err)
	}
	var transfer struct {
		Download float64 `json:"dl_info_speed"`
		Upload   float64 `json:"up_info_speed"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api/v2/transfer/info", Headers: headers}, &transfer); err != nil {
		return offlineStatus("Error", err)
	}
	return Status{
		Label: fmt.Sprintf("Down %s | Up %s", humanizeRate(transfer.Download, []string{"B", "KB", "MB", "GB"}), humanizeRate(transfer.Upload, []string{"B", "KB", "MB", "GB"})),
		Badges: positiveBadges(
			countBadge("Torrents", len(torrents), "torrents", "info"),
		),
		Updated: time.Now(),
	}
}

func qBittorrentHeaders(ctx context.Context, item config.Item, proxy config.Proxy) (map[string]string, error) {
	username := stringField(item, "username")
	password := stringField(item, "password")
	if username == "" && password == "" {
		return nil, nil
	}
	form := url.Values{}
	form.Set("username", username)
	form.Set("password", password)
	resp, err := doCollectorRequest(ctx, item, proxy, requestOptions{
		Method:  http.MethodPost,
		Path:    "api/v2/auth/login",
		Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		Body:    strings.NewReader(form.Encode()),
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		return nil, nil
	}
	values := make([]string, 0, len(cookies))
	for _, cookie := range cookies {
		values = append(values, cookie.Name+"="+cookie.Value)
	}
	return map[string]string{"Cookie": strings.Join(values, "; ")}, nil
}
