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

	"github.com/grafana/grafana-foundation-sdk/go/common"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/prometheus"
	"github.com/grafana/grafana-foundation-sdk/go/timeseries"
)

var client = &http.Client{Timeout: 10 * time.Second}

func main() {
	grafanaURL := env("GRAFANA_URL", "http://grafana:3000")
	appURL := env("APP_URL", "http://app:8080")

	if err := waitForGrafana(grafanaURL); err != nil {
		log.Fatal(err)
	}

	routes, err := getRoutes(appURL)
	if err != nil {
		log.Fatal(err)
	}

	data, err := os.ReadFile("/run/grafana-token/token")
	if err != nil {
		log.Fatal(err)
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		log.Fatal("Grafana service account token is empty")
	}

	d, err := makeDashboard(routes)
	if err != nil {
		log.Fatal(err)
	}

	body := map[string]any{
		"dashboard": d,
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
	for range 60 {
		resp, err := client.Get(baseURL + "/api/health")
		if err == nil {
			resp.Body.Close()
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

	for range 60 {
		resp, err := client.Get(baseURL + "/routes")
		if err == nil {
			var result struct {
				Routes []string `json:"routes"`
			}

			if resp.StatusCode == http.StatusOK {
				err = json.NewDecoder(resp.Body).Decode(&result)
				resp.Body.Close()

				if err == nil {
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
				}
				lastErr = err
			} else {
				resp.Body.Close()
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
	token string,
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
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	return resp.StatusCode, data, nil
}

func makeDashboard(routes []string) (dashboard.Dashboard, error) {

	dsType := "prometheus"
	dsUID := "victoriametrics"

	ds := common.DataSourceRef{
		Type: &dsType,
		Uid:  &dsUID,
	}

	d := dashboard.NewDashboardBuilder(
		"Shop API — automatically generated",
	).
		Uid("shop-api-auto").
		Tags([]string{"shop-api", "generated"}).
		Refresh("5s").
		Time("now-15m", "now").
		Readonly()

	variables := []struct {
		name  string
		label string
		query string
	}{
		{
			name:  "datacenter",
			label: "Datacenter",
			query: "label_values(http_requests_total, datacenter)",
		},
		{
			name:  "pod",
			label: "Pod",
			query: `label_values(http_requests_total{datacenter=~"${datacenter:regex}"}, pod)`,
		},
	}

	for _, v := range variables {
		d = d.WithVariable(
			dashboard.NewQueryVariableBuilder(v.name).
				Label(v.label).
				Datasource(ds).
				Definition(v.query).
				Query(dashboard.StringOrMap{
					Map: map[string]any{
						"query": v.query,
						"refId": "PrometheusVariableQueryEditor-VariableQuery",
					},
				}).
				Refresh(dashboard.VariableRefreshOnDashboardLoad).
				Multi(true).
				IncludeAll(true).
				AllValue(".*"),
		)
	}

	type metric struct {
		title string
		expr  string
		unit  string
	}

	panelID := uint32(1)

	for _, route := range routes {
		d = d.WithRow(dashboard.NewRowBuilder(route))

		selector := fmt.Sprintf(
			`datacenter=~"${datacenter:regex}", pod=~"${pod:regex}", handler=%s`,
			strconv.Quote(route),
		)

		queries := []metric{
			{
				title: "Requests / second",
				expr:  "sum(rate(http_requests_total{" + selector + "}[1m])) or vector(0)",
				unit:  "reqps",
			},
			{
				title: "Errors / second",
				expr: "sum(rate(http_requests_total{" + selector +
					`, status=~"[45].."}[1m])) or vector(0)`,
				unit: "reqps",
			},
			{
				title: "Latency p95",
				expr: "histogram_quantile(0.95, sum by (le) (" +
					"rate(http_request_duration_seconds_bucket{" +
					selector + "}[5m])))",
				unit: "s",
			},
		}

		for _, q := range queries {
			d = d.WithPanel(
				timeseries.NewPanelBuilder().
					Id(panelID).
					Title(route + " — " + q.title).
					Datasource(ds).
					Unit(q.unit).
					Height(8).
					WithTarget(
						prometheus.NewDataqueryBuilder().
							Expr(q.expr).
							LegendFormat(route),
					),
			)
			panelID++
		}
	}

	return d.Build()
}
