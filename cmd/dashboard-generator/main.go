package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type RoutesResponse struct {
	Routes []string `json:"routes"`
}

type ServiceAccount struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type ServiceAccountsResponse struct {
	ServiceAccounts []ServiceAccount `json:"serviceAccounts"`
}

type TokenResponse struct {
	Key string `json:"key"`
}

var client = &http.Client{Timeout: 10 * time.Second}

func main() {
	grafanaURL := env("GRAFANA_URL", "http://grafana:3000")
	appURL := env("APP_URL", "http://app:8080")
	adminUser := env("GRAFANA_ADMIN_USER", "admin")
	adminPassword := os.Getenv("GRAFANA_ADMIN_PASSWORD")

	if adminPassword == "" {
		log.Fatal("GRAFANA_ADMIN_PASSWORD is required")
	}

	if err := waitForGrafana(grafanaURL); err != nil {
		log.Fatal(err)
	}

	routes, err := getRoutes(appURL)
	if err != nil {
		log.Fatal(err)
	}

	token, err := createServiceAccountToken(
		grafanaURL,
		adminUser,
		adminPassword,
	)
	if err != nil {
		log.Fatal(err)
	}

	dashboard := makeDashboard(routes)

	body := map[string]any{
		"dashboard": dashboard,
		"folderId":  0,
		"overwrite": true,
		"message":   "Generated automatically from application routes",
	}

	status, response, err := grafanaRequest(
		grafanaURL,
		http.MethodPost,
		"/api/dashboards/db",
		body,
		token,
		"", "",
	)
	if err != nil {
		log.Fatal(err)
	}
	if status < 200 || status >= 300 {
		log.Fatalf("create dashboard: HTTP %d: %s", status, response)
	}

	log.Printf("Dashboard created: %s/d/shop-api-auto", grafanaURL)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func waitForGrafana(baseURL string) error {
	for i := 0; i < 60; i++ {
		resp, err := client.Get(baseURL + "/api/health")
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("Grafana did not become healthy")
}

func getRoutes(baseURL string) ([]string, error) {
	var lastErr error

	for i := 0; i < 60; i++ {
		resp, err := client.Get(baseURL + "/routes")
		if err == nil {
			data, readErr := io.ReadAll(resp.Body)
			_ = resp.Body.Close()

			if readErr == nil && resp.StatusCode == http.StatusOK {
				var result RoutesResponse
				if err := json.Unmarshal(data, &result); err == nil {
					routes := make([]string, 0, len(result.Routes))

					for _, route := range result.Routes {
						switch route {
						case "GET /metrics", "GET /routes", "GET /health":
							continue
						default:
							routes = append(routes, route)
						}
					}

					sort.Strings(routes)
					if len(routes) == 0 {
						return nil, fmt.Errorf("no application routes found")
					}
					return routes, nil
				} else {
					lastErr = err
				}
			} else {
				lastErr = fmt.Errorf("GET /routes returned HTTP %d", resp.StatusCode)
			}
		} else {
			lastErr = err
		}

		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("cannot get application routes: %w", lastErr)
}

func grafanaRequest(
	baseURL, method, path string,
	body any,
	token, username, password string,
) (int, []byte, error) {
	var requestBody io.Reader

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		requestBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, baseURL+path, requestBody)
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else if username != "" {
		req.SetBasicAuth(username, password)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, data, nil
	}

	return resp.StatusCode, data, nil
}

func createServiceAccountToken(
	baseURL, username, password string,
) (string, error) {
	const accountName = "shop-dashboard-generator"

	status, data, err := grafanaRequest(
		baseURL,
		http.MethodPost,
		"/api/serviceaccounts",
		map[string]any{
			"name":       accountName,
			"role":       "Editor",
			"isDisabled": false,
		},
		"", username, password,
	)
	if err != nil {
		return "", err
	}

	var account ServiceAccount

	if status >= 200 && status < 300 {
		if err := json.Unmarshal(data, &account); err != nil {
			return "", err
		}
	} else if status == http.StatusConflict {
		// Reuse the service account if the generator has run before.
		status, data, err = grafanaRequest(
			baseURL,
			http.MethodGet,
			"/api/serviceaccounts/search?query="+accountName,
			nil,
			"", username, password,
		)
		if err != nil {
			return "", err
		}
		if status < 200 || status >= 300 {
			return "", fmt.Errorf("search service account: HTTP %d: %s", status, data)
		}

		var result ServiceAccountsResponse
		if err := json.Unmarshal(data, &result); err != nil {
			return "", err
		}

		for _, candidate := range result.ServiceAccounts {
			if candidate.Name == accountName {
				account = candidate
				break
			}
		}
	} else {
		return "", fmt.Errorf("create service account: HTTP %d: %s", status, data)
	}

	if account.ID == 0 {
		return "", fmt.Errorf("service account %q not found", accountName)
	}

	status, data, err = grafanaRequest(
		baseURL,
		http.MethodPost,
		fmt.Sprintf("/api/serviceaccounts/%d/tokens", account.ID),
		map[string]any{
			"name": "dashboard-generator",
		},
		"", username, password,
	)
	if err != nil {
		return "", err
	}
	if status < 200 || status >= 300 {
		return "", fmt.Errorf("create service account token: HTTP %d: %s", status, data)
	}

	var result TokenResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}
	if result.Key == "" {
		return "", fmt.Errorf("Grafana returned an empty service account token")
	}

	return result.Key, nil
}

func makeDashboard(routes []string) map[string]any {
	datasource := map[string]any{
		"type": "prometheus",
		"uid":  "victoriametrics",
	}

	variables := []any{
		map[string]any{
			"name":       "datacenter",
			"label":      "Datacenter",
			"type":       "query",
			"datasource": datasource,
			"definition": "label_values(http_requests_total, datacenter)",
			"query":      map[string]any{"query": "label_values(http_requests_total, datacenter)", "refId": "PrometheusVariableQueryEditor-VariableQuery"},
			"refresh":    1,
			"multi":      true,
			"includeAll": true,
			"allValue":   ".*",
			"current":    map[string]any{"selected": true, "text": "All", "value": "$__all"},
			"options":    []any{},
			"hide":       0,
		},
		map[string]any{
			"name":       "pod",
			"label":      "Pod",
			"type":       "query",
			"datasource": datasource,
			"definition": "label_values(http_requests_total{datacenter=~\"${datacenter:regex}\"}, pod)",
			"query":      map[string]any{"query": "label_values(http_requests_total{datacenter=~\"${datacenter:regex}\"}, pod)", "refId": "PrometheusVariableQueryEditor-VariableQuery"},
			"refresh":    1,
			"multi":      true,
			"includeAll": true,
			"allValue":   ".*",
			"current":    map[string]any{"selected": true, "text": "All", "value": "$__all"},
			"options":    []any{},
			"hide":       0,
		},
	}

	panels := make([]any, 0, len(routes)*3)
	panelID := 1

	for row, route := range routes {
		handler := strconv.Quote(route)
		selector := fmt.Sprintf(
			`datacenter=~"${datacenter:regex}", pod=~"${pod:regex}", handler=%s`,
			handler,
		)
		errorSelector := selector + `, status=~"[45].."`

		queries := []struct {
			title string
			expr  string
			unit  string
		}{
			{
				title: "Requests / second",
				expr:  "sum(rate(http_requests_total{" + selector + "}[1m])) or vector(0)",
				unit:  "reqps",
			},
			{
				title: "Errors / second",
				expr:  "sum(rate(http_requests_total{" + errorSelector + "}[1m])) or vector(0)",
				unit:  "reqps",
			},
			{
				title: "Latency p95",
				expr: "histogram_quantile(0.95, sum by (le) (" +
					"rate(http_request_duration_seconds_bucket{" + selector + "}[5m])))",
				unit: "s",
			},
		}

		for col, query := range queries {
			panels = append(panels, map[string]any{
				"id":         panelID,
				"title":      route + " — " + query.title,
				"type":       "timeseries",
				"datasource": datasource,
				"gridPos": map[string]any{
					"x": col * 8,
					"y": row * 8,
					"w": 8,
					"h": 8,
				},
				"targets": []any{
					map[string]any{
						"refId":        "A",
						"expr":         query.expr,
						"legendFormat": route,
					},
				},
				"fieldConfig": map[string]any{
					"defaults":  map[string]any{"unit": query.unit},
					"overrides": []any{},
				},
				"options": map[string]any{
					"legend": map[string]any{
						"showLegend": false,
					},
					"tooltip": map[string]any{
						"mode": "single",
					},
				},
			})
			panelID++
		}
	}

	return map[string]any{
		"id":            nil,
		"uid":           "shop-api-auto",
		"title":         "Shop API — automatically generated",
		"tags":          []string{"shop-api", "generated"},
		"schemaVersion": 39,
		"version":       0,
		"editable":      false,
		"refresh":       "5s",
		"time": map[string]any{
			"from": "now-15m",
			"to":   "now",
		},
		"templating": map[string]any{
			"list": variables,
		},
		"annotations": map[string]any{
			"list": []any{},
		},
		"panels": panels,
	}
}

var _ = strings.TrimSpace
