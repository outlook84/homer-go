package collectors

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"homer-go/internal/config"
)

type DockerSocketProxy struct{}

func (DockerSocketProxy) Type() string { return "DockerSocketProxy" }

func (DockerSocketProxy) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	var containers []struct {
		State string `json:"State"`
	}
	if err := collectDockerContainers(ctx, item, proxy, &containers); err != nil {
		return offlineStatus("Error", err)
	}
	running := 0
	exited := 0
	for _, container := range containers {
		switch container.State {
		case "running":
			running++
		case "exited":
			exited++
		}
	}
	return Status{
		Badges: positiveBadges(
			countBadge("Running", running, "running", "info"),
			countBadge("Stopped", exited, "stopped", "warning"),
		),
		Updated: time.Now(),
	}
}

func collectDockerContainers[T any](ctx context.Context, item config.Item, proxy config.Proxy, out *T) error {
	socketPath := dockerSocketPath(item)
	if socketPath == "" {
		return collectJSON(ctx, item, proxy, requestOptions{Path: "containers/json?all=true"}, out)
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}
	defer transport.CloseIdleConnections()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://docker/containers/json?all=true", nil)
	if err != nil {
		return err
	}
	applyHeaders(req, effectiveHeaders(item, proxy))
	resp, err := (&http.Client{Transport: transport}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if !successCode(item, resp.StatusCode) {
		return errors.New(resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func dockerSocketPath(item config.Item) string {
	if socket := stringField(item, "socket"); socket != "" {
		return socket
	}
	endpoint := stringField(item, "endpoint")
	if endpoint == "" {
		endpoint = item.URL
	}
	if strings.HasPrefix(endpoint, "unix://") {
		return strings.TrimPrefix(endpoint, "unix://")
	}
	return ""
}
