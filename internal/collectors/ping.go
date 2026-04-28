package collectors

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"homer-go/internal/config"
)

type Ping struct{}

func (Ping) Type() string { return "Ping" }

func (Ping) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	endpoint := stringField(item, "endpoint")
	if endpoint == "" {
		endpoint = item.URL
	}
	if endpoint == "" {
		return Status{State: "unknown", Tone: "warning", Label: "No URL", Indicator: "unknown", Updated: time.Now()}
	}
	method := strings.ToUpper(stringField(item, "method"))
	if method == "" {
		method = http.MethodHead
	}
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return Status{State: "offline", Tone: "danger", Label: "Invalid URL", Detail: err.Error(), Indicator: "offline", Updated: time.Now()}
	}
	applyHeaders(req, effectiveHeaders(item, proxy))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Status{State: "offline", Tone: "danger", Label: "Offline", Detail: err.Error(), Indicator: "offline", Updated: time.Now()}
	}
	defer resp.Body.Close()
	elapsed := time.Since(start)
	if successCode(item, resp.StatusCode) {
		return Status{State: "online", Tone: "success", Label: fmt.Sprintf("%d ms", elapsed.Milliseconds()), Detail: resp.Status, Indicator: "online", Updated: time.Now()}
	}
	return Status{State: "offline", Tone: "danger", Label: resp.Status, Indicator: "offline", Updated: time.Now()}
}
