package collectors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"homer-go/internal/config"
)

type requestOptions struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    io.Reader
}

func collectJSON[T any](ctx context.Context, item config.Item, proxy config.Proxy, opts requestOptions, out *T) error {
	resp, err := doCollectorRequest(ctx, item, proxy, opts)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

func collectText(ctx context.Context, item config.Item, proxy config.Proxy, opts requestOptions) (string, error) {
	resp, err := doCollectorRequest(ctx, item, proxy, opts)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func doCollectorRequest(ctx context.Context, item config.Item, proxy config.Proxy, opts requestOptions) (*http.Response, error) {
	method := opts.Method
	if method == "" {
		method = http.MethodGet
	}
	url := collectorURL(item, opts.Path)
	if url == "" {
		return nil, fmt.Errorf("missing URL")
	}
	req, err := http.NewRequestWithContext(ctx, method, url, opts.Body)
	if err != nil {
		return nil, err
	}
	applyHeaders(req, effectiveHeaders(item, proxy))
	applyHeaders(req, opts.Headers)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if !successCode(item, resp.StatusCode) {
		defer resp.Body.Close()
		return nil, errors.New(resp.Status)
	}
	return resp, nil
}

func collectorURL(item config.Item, path string) string {
	endpoint := stringField(item, "endpoint")
	if endpoint == "" {
		endpoint = item.URL
	}
	endpoint = strings.TrimRight(endpoint, "/")
	path = strings.TrimLeft(path, "/")
	if endpoint == "" {
		return ""
	}
	if path == "" {
		return endpoint
	}
	return endpoint + "/" + path
}

func onlineStatus(label, detail string) Status {
	return Status{State: "online", Tone: "success", Label: label, Detail: detail, Updated: time.Now()}
}

func offlineStatus(label string, err error) Status {
	detail := ""
	if err != nil {
		detail = err.Error()
	}
	if label == "" {
		label = "Offline"
	}
	return Status{State: "offline", Tone: "danger", Label: label, Detail: detail, Updated: time.Now()}
}
