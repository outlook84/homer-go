package collectors

import (
	"context"
	"fmt"

	"homer-go/internal/config"
)

type Emby struct{}

func (Emby) Type() string { return "Emby" }

func (Emby) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	var info struct {
		ID string `json:"Id"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "System/info/public"}, &info); err != nil {
		return offlineStatus("dead", err)
	}
	if info.ID == "" {
		return offlineStatus("dead", fmt.Errorf("missing server id"))
	}
	status := onlineStatus("running", "")
	status.State = "running"
	status.Indicator = "running"

	if item.Subtitle != "" {
		return status
	}
	headers := map[string]string{}
	if apiKey := stringField(item, "apikey"); apiKey != "" {
		headers["X-Emby-Token"] = apiKey
	}
	var counts struct {
		AlbumCount   int `json:"AlbumCount"`
		SongCount    int `json:"SongCount"`
		MovieCount   int `json:"MovieCount"`
		SeriesCount  int `json:"SeriesCount"`
		EpisodeCount int `json:"EpisodeCount"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "items/counts", Headers: headers}, &counts); err != nil {
		status.Detail = err.Error()
		return status
	}
	switch stringField(item, "libraryType") {
	case "music":
		status.Label = fmt.Sprintf("%d songs, %d albums", counts.SongCount, counts.AlbumCount)
	case "movies":
		status.Label = fmt.Sprintf("%d movies", counts.MovieCount)
	case "series":
		status.Label = fmt.Sprintf("%d eps, %d series", counts.EpisodeCount, counts.SeriesCount)
	default:
		status.Label = "wrong library type"
	}
	return status
}
