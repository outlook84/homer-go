package collectors

import (
	"bytes"
	"context"
	"net/http"
	"time"

	"homer-go/internal/config"
)

type Tdarr struct{}

func (Tdarr) Type() string { return "Tdarr" }

func (Tdarr) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	body := []byte(`{"headers":{"content-Type":"application/json"},"data":{"collection":"StatisticsJSONDB","mode":"getById","docID":"statistics","obj":{}},"timeout":1000}`)
	headers := map[string]string{"Content-Type": "application/json", "Accept": "application/json"}
	if apiKey := stringField(item, "apikey"); apiKey != "" {
		headers["x-api-key"] = apiKey
	}
	var response struct {
		Queue   int `json:"table1Count"`
		Errored int `json:"table6Count"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{
		Method:  http.MethodPost,
		Path:    "api/v2/cruddb",
		Headers: headers,
		Body:    bytes.NewReader(body),
	}, &response); err != nil {
		return offlineStatus("Error", err)
	}
	return Status{
		Badges: positiveBadges(
			countBadge("Queue", response.Queue, "queue", "info"),
			countBadge("Errored", response.Errored, "errored", "danger"),
		),
		Updated: time.Now(),
	}
}
