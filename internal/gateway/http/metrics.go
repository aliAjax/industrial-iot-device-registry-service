package http

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	registry      *prometheus.Registry
	requests      *prometheus.CounterVec
	requestBytes  *prometheus.CounterVec
	responseBytes *prometheus.CounterVec
	latency       *prometheus.HistogramVec
}

func NewMetrics(registry *prometheus.Registry) *Metrics {
	requests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "iot",
		Subsystem: "http",
		Name:      "requests_total",
		Help:      "Total HTTP requests.",
	}, []string{"method", "path", "status"})
	requestBytes := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "iot",
		Subsystem: "http",
		Name:      "request_bytes_total",
		Help:      "Total HTTP request body bytes.",
	}, []string{"method", "path"})
	responseBytes := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "iot",
		Subsystem: "http",
		Name:      "response_bytes_total",
		Help:      "Total HTTP response body bytes.",
	}, []string{"method", "path", "status"})
	latency := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "iot",
		Subsystem: "http",
		Name:      "request_duration_seconds",
		Help:      "HTTP request latency in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "path"})
	registry.MustRegister(requests, requestBytes, responseBytes, latency)
	return &Metrics{registry: registry, requests: requests, requestBytes: requestBytes, responseBytes: responseBytes, latency: latency}
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		status := strconv.Itoa(recorder.status)
		m.requests.WithLabelValues(r.Method, r.URL.Path, status).Inc()
		m.requestBytes.WithLabelValues(r.Method, r.URL.Path).Add(float64(r.ContentLength))
		m.responseBytes.WithLabelValues(r.Method, r.URL.Path, status).Add(float64(recorder.bytes))
		m.latency.WithLabelValues(r.Method, r.URL.Path).Observe(time.Since(start).Seconds())
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not implement http.Hijacker")
	}
	return hijacker.Hijack()
}

func (r *statusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(payload []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(payload)
	r.bytes += n
	return n, err
}
