package metrics

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	Latency    *prometheus.HistogramVec
	Throughput *prometheus.CounterVec
	HTTPErrors *prometheus.CounterVec

	POD        string
	DATACENTER string
	DEPLOYMENT string
}

type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *responseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}

	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	labels := []string{
		"pod",
		"datacenter",
		"deployment",
		"handler",
	}

	m := &Metrics{
		Latency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request latency in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			labels,
		),

		Throughput: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests.",
			},
			append(labels, "status"),
		),

		HTTPErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_request_errors_total",
				Help: "Total number of HTTP requests ending with 4xx or 5xx.",
			},
			append(labels, "status"),
		),

		POD:        os.Getenv("POD"),
		DATACENTER: os.Getenv("DATACENTER"),
		DEPLOYMENT: os.Getenv("DEPLOYMENT"),
	}

	reg.MustRegister(m.Latency, m.Throughput, m.HTTPErrors)

	return m
}

func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		start := time.Now()
		next.ServeHTTP(rw, r)
		duration := time.Since(start)

		handler := chi.RouteContext(r.Context()).RoutePattern()
		if handler == "" {
			handler = "unmatched"
		}
		handler = r.Method + " " + handler

		baseLabels := []string{
			m.POD,
			m.DATACENTER,
			m.DEPLOYMENT,
			handler,
		}

		status := strconv.Itoa(rw.status)

		m.Latency.WithLabelValues(baseLabels...).Observe(duration.Seconds())

		requestLabels := append(
			append([]string{}, baseLabels...),
			status,
		)
		m.Throughput.WithLabelValues(requestLabels...).Inc()

		if rw.status >= 400 {
			m.HTTPErrors.WithLabelValues(requestLabels...).Inc()
		}
	})
}
