package util

import (
	"fmt"
	"net/http"
	"strings"
)

type BreadcrumbData struct {
	Breadcrumb     []breadcrumbItem
	LastBreadcrumb string
	ThemeSwitch    Theme
}

type breadcrumbItem struct {
	Title   string
	Address string
}

func PrepareBreadcrumb(req *http.Request) BreadcrumbData {
	result := []breadcrumbItem{
		{
			Title:   req.Host,
			Address: "/",
		},
	}

	allItems := strings.Split(req.URL.Path, "/")
	var items []string
	for _, item := range allItems {
		if len(item) != 0 {
			items = append(items, item)
		}
	}
	for idx, item := range items {
		if len(item) == 0 {
			continue
		}

		address := fmt.Sprintf("/%s/", strings.Join(items[:idx+1], "/"))
		result = append(result, breadcrumbItem{
			Title:   item,
			Address: address,
		})
	}

	return BreadcrumbData{
		Breadcrumb:     result[:len(result)-1],
		LastBreadcrumb: result[len(result)-1].Title,
		ThemeSwitch:    GetTheme(req),
	}
}
