package collectors

import (
	"context"
	"time"

	"homer-go/internal/config"
)

type Portainer struct{}

func (Portainer) Type() string { return "Portainer" }

func (Portainer) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	headers := map[string]string{}
	if apiKey := stringField(item, "apikey"); apiKey != "" {
		headers["X-Api-Key"] = apiKey
	}
	var version struct {
		Version string `json:"Version"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api/status", Headers: headers}, &version); err != nil {
		return offlineStatus("offline", err)
	}
	status := onlineStatus("", "")
	status.State = "online"
	status.Indicator = "online"
	if version.Version != "" {
		status.Label = "Version " + version.Version
	}

	var endpoints []struct {
		ID   int    `json:"Id"`
		Name string `json:"Name"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api/endpoints", Headers: headers}, &endpoints); err != nil {
		return status
	}
	environments := stringSet(stringSliceField(item, "environments"))
	running := 0
	dead := 0
	misc := 0
	for _, endpoint := range endpoints {
		if len(environments) > 0 && !environments[endpoint.Name] {
			continue
		}
		var containers []struct {
			State string `json:"State"`
		}
		path := "api/endpoints/" + intString(endpoint.ID) + "/docker/containers/json?all=1"
		if err := collectJSON(ctx, item, proxy, requestOptions{Path: path, Headers: headers}, &containers); err != nil {
			continue
		}
		for _, container := range containers {
			switch container.State {
			case "running":
				running++
			case "dead":
				dead++
			default:
				misc++
			}
		}
	}
	status.Badges = positiveBadges(
		countBadge("Running", running, "running", "success"),
		countBadge("Dead", dead, "dead", "danger"),
		countBadge("Other", misc, "misc", "info"),
	)
	status.Updated = time.Now()
	return status
}
