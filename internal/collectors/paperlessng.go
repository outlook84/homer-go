package collectors

import (
	"context"
	"fmt"

	"homer-go/internal/config"
)

type PaperlessNG struct{}

func (PaperlessNG) Type() string { return "PaperlessNG" }

func (PaperlessNG) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	apiKey := stringField(item, "apikey")
	if apiKey == "" {
		return offlineStatus("Missing API key", nil)
	}
	var response struct {
		Count int `json:"count"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{
		Path:    "api/documents/",
		Headers: map[string]string{"Authorization": "Token " + apiKey},
	}, &response); err != nil {
		return offlineStatus("Error", err)
	}
	return onlineStatus(fmt.Sprintf("happily storing %d documents", response.Count), "")
}
