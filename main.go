package main

import (
	"database/sql"
	"encoding/json"
	"hls-hw-01/internal/db"
	"hls-hw-01/internal/handlers"
	"hls-hw-01/internal/metrics"
	"log"
	"net/http"
	"os"
	"sort"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	reg := prometheus.NewRegistry()
	m := metrics.NewMetrics(reg)
	db.Database = dbInit()
	defer db.Database.Close()
	r := routerInit(reg, m)
	log.Fatal(http.ListenAndServe(":8080", r))
}

func routerInit(reg *prometheus.Registry, m *metrics.Metrics) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(m.Middleware)
	r.Get("/items", handlers.GetItems)
	r.Get("/item", handlers.GetItem)
	r.Delete("/item", handlers.DeleteItem)
	r.Post("/item", handlers.PutItem)
	r.Method(http.MethodGet, "/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	r.Get("/routes", func(w http.ResponseWriter, req *http.Request) {
		routes := make([]string, 0)
		err := chi.Walk(r, func(method string, route string, handler http.Handler,
			middlewares ...func(http.Handler) http.Handler) error {
			routes = append(routes, method+" "+route)
			return nil
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		sort.Strings(routes)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string][]string{
			"routes": routes,
		}); err != nil {
			log.Printf("encode routes: %v", err)
		}
	})
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	return r
}

func dbInit() *sql.DB {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	return db
}
