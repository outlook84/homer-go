package collectors

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"homer-go/internal/config"
)

type FreshRSS struct{}

func (FreshRSS) Type() string { return "FreshRSS" }

func (FreshRSS) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	loginQuery := url.Values{}
	loginQuery.Set("Email", stringField(item, "username"))
	loginQuery.Set("Passwd", stringField(item, "password"))
	authText, err := collectText(ctx, item, proxy, requestOptions{Path: "api/greader.php/accounts/ClientLogin?" + loginQuery.Encode()})
	if err != nil {
		return offlineStatus("Error", err)
	}
	match := regexp.MustCompile(`(?m)^Auth=(.+)$`).FindStringSubmatch(authText)
	if len(match) != 2 {
		return offlineStatus("Error", fmt.Errorf("missing auth token"))
	}
	token := strings.TrimSpace(match[1])
	headers := map[string]string{"Authorization": "GoogleLogin auth=" + token}
	var subscriptions struct {
		Subscriptions []struct{} `json:"subscriptions"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api/greader.php/reader/api/0/subscription/list?output=json", Headers: headers}, &subscriptions); err != nil {
		return offlineStatus("Error", err)
	}
	var unread struct {
		UnreadCounts []struct {
			ID    string `json:"id"`
			Count int    `json:"count"`
		} `json:"unreadcounts"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api/greader.php/reader/api/0/unread-count?output=json", Headers: headers}, &unread); err != nil {
		return offlineStatus("Error", err)
	}
	unreadCount := 0
	for _, count := range unread.UnreadCounts {
		if strings.HasSuffix(count.ID, "/state/com.google/reading-list") {
			unreadCount = count.Count
			break
		}
		if strings.HasPrefix(count.ID, "feed/") {
			unreadCount += count.Count
		}
	}
	return Status{
		Badges: positiveBadges(
			countBadge("Subscriptions", len(subscriptions.Subscriptions), "subscriptions", "info"),
			countBadge("Unread", unreadCount, "unread", "warning"),
		),
		Updated: time.Now(),
	}
}
