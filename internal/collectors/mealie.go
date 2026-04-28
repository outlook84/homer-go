package collectors

import (
	"context"
	"fmt"

	"homer-go/internal/config"
)

type Mealie struct{}

func (Mealie) Type() string { return "Mealie" }

func (Mealie) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
	apiKey := stringField(item, "apikey")
	headers := map[string]string{
		"Accept": "application/json",
	}
	if apiKey != "" {
		headers["Authorization"] = "Bearer " + apiKey
	}

	var meals []struct {
		Recipe struct {
			Name string `json:"name"`
		} `json:"recipe"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api/groups/mealplans/today", Headers: headers}, &meals); err == nil && len(meals) > 0 && meals[0].Recipe.Name != "" {
		return onlineStatus("Today: "+meals[0].Recipe.Name, "")
	}

	var stats struct {
		TotalRecipes int `json:"totalRecipes"`
	}
	if err := collectJSON(ctx, item, proxy, requestOptions{Path: "api/admin/about/statistics", Headers: headers}, &stats); err != nil {
		return offlineStatus("Error", err)
	}
	return onlineStatus(fmt.Sprintf("Happily keeping %d recipes organized", stats.TotalRecipes), "")
}
