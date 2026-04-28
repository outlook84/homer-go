package collectors

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"homer-go/internal/config"
)

type Status struct {
	State     string
	Tone      string
	Label     string
	Detail    string
	Indicator string
	URL       string
	Badges    []Badge
	Updated   time.Time
}

type Badge struct {
	Label  string
	Value  string
	State  string
	Tone   string
	Detail string
}

type Collector interface {
	Type() string
	Collect(context.Context, config.Item, config.Proxy) Status
}

type Registry struct {
	collectors map[string]Collector
}

type UnsupportedCollector struct {
	GroupIndex int
	ItemIndex  int
	GroupName  string
	ItemName   string
	Type       string
}

func NewRegistry() *Registry {
	return &Registry{collectors: map[string]Collector{}}
}

func (r *Registry) Register(c Collector) {
	r.collectors[strings.ToLower(c.Type())] = c
}

func (r *Registry) Has(typeName string) bool {
	_, ok := r.collectors[strings.ToLower(strings.TrimSpace(typeName))]
	return ok
}

func (r *Registry) UnsupportedCollectors(cfg config.Config) []UnsupportedCollector {
	var out []UnsupportedCollector
	for groupIndex, group := range cfg.Services {
		for itemIndex, item := range group.Items {
			itemType := strings.TrimSpace(item.Type)
			if itemType == "" || strings.EqualFold(itemType, "Generic") || r.Has(itemType) {
				continue
			}
			out = append(out, UnsupportedCollector{
				GroupIndex: groupIndex,
				ItemIndex:  itemIndex,
				GroupName:  group.Name,
				ItemName:   item.Name,
				Type:       itemType,
			})
		}
	}
	return out
}

func (r *Registry) Collect(ctx context.Context, cfg config.Config, timeout time.Duration) map[string]Status {
	out := map[string]Status{}
	for groupIndex, group := range cfg.Services {
		for itemIndex, item := range group.Items {
			c, ok := r.collectors[strings.ToLower(item.Type)]
			if !ok {
				continue
			}
			itemCtx, cancel := context.WithTimeout(ctx, timeout)
			out[Key(groupIndex, itemIndex)] = c.Collect(itemCtx, item, cfg.Proxy)
			cancel()
		}
	}
	return out
}

func (r *Registry) CollectItem(ctx context.Context, item config.Item, timeout time.Duration) (Status, bool) {
	c, ok := r.collectors[strings.ToLower(item.Type)]
	if !ok {
		return Status{}, false
	}
	itemCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return c.Collect(itemCtx, item, config.Proxy{}), true
}

func Key(groupIndex, itemIndex int) string {
	return fmt.Sprintf("%d:%d", groupIndex, itemIndex)
}

func effectiveHeaders(item config.Item, proxy config.Proxy) map[string]string {
	headers := proxy.Headers
	if hasRawField(item, "headers") || item.Headers != nil {
		headers = item.Headers
	}
	return headers
}

func applyHeaders(req *http.Request, headers map[string]string) {
	for name, value := range headers {
		req.Header.Set(name, value)
	}
}

func successCode(item config.Item, code int) bool {
	raw, ok := item.Raw["successCodes"].([]any)
	if ok && len(raw) > 0 {
		for _, value := range raw {
			switch v := value.(type) {
			case int:
				if code == v {
					return true
				}
			case int64:
				if code == int(v) {
					return true
				}
			}
		}
		return false
	}
	return code >= 200 && code < 300
}

func stringField(item config.Item, name string) string {
	if value, ok := item.Raw[name].(string); ok {
		return value
	}
	return ""
}

func hasRawField(item config.Item, name string) bool {
	if item.Raw == nil {
		return false
	}
	_, ok := item.Raw[name]
	return ok
}
