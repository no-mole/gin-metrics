package gin_metrics

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

type MetricInterface interface {
	Exec(startTime time.Time, ctx *gin.Context)
	Collector(jobName string) prometheus.Collector
}

// Metric is a definition for the name, description, type, ID, and
// prometheus.Collector type (i.e. CounterVec, Summary, etc) of each metric
type Metric struct {
	collector   prometheus.Collector
	Name        string
	Description string
	Type        string
	Args        []string
	ExecFn      func(collector prometheus.Collector, startTime time.Time, ctx *gin.Context)
	once        sync.Once
}

func (m *Metric) Exec(startTime time.Time, ctx *gin.Context) {
	m.ExecFn(m.collector, startTime, ctx)
}

func (m *Metric) Collector(jobName string) prometheus.Collector {
	m.once.Do(func() {
		m.collector = NewMetric(m, jobName)
	})
	return m.collector
}

var _ MetricInterface = &Metric{}

type Mapper func(ctx *gin.Context) string

var (
	RequestTotalWithMapper = func(mapper Mapper) MetricInterface {
		return &Metric{
			Name:        "requests_total",
			Description: "How many HTTP requests processed, partitioned by status code and HTTP method.",
			Type:        "counter_vec",
			Args:        []string{"code", "method", "handler", "host", "url"},
			ExecFn: func(collector prometheus.Collector, startTime time.Time, ctx *gin.Context) {
				c := collector.(*prometheus.CounterVec)
				url := mapper(ctx)
				c.WithLabelValues(strconv.Itoa(ctx.Writer.Status()), ctx.Request.Method, ctx.HandlerName(), ctx.Request.Host, url).Inc()
			},
		}
	}

	RequestTotal MetricInterface = &Metric{
		Name:        "requests_total",
		Description: "How many HTTP requests processed, partitioned by status code and HTTP method.",
		Type:        "counter_vec",
		Args:        []string{"code", "method", "handler", "host", "url"},
		ExecFn: func(collector prometheus.Collector, startTime time.Time, ctx *gin.Context) {
			c := collector.(*prometheus.CounterVec)
			c.WithLabelValues(strconv.Itoa(ctx.Writer.Status()), ctx.Request.Method, ctx.HandlerName(), ctx.Request.Host, ctx.Request.URL.Path).Inc()
		},
	}

	RequestDuration MetricInterface = &Metric{
		Name:        "request_duration_seconds",
		Description: "The HTTP request latencies in Millisecond.",
		Type:        "histogram_vec",
		Args:        []string{"code", "method", "url"},
		ExecFn: func(collector prometheus.Collector, startTime time.Time, ctx *gin.Context) {
			c := collector.(*prometheus.HistogramVec)
			c.WithLabelValues(strconv.Itoa(ctx.Writer.Status()), ctx.Request.Method, ctx.Request.URL.Path).Observe(float64(time.Since(startTime)) / float64(time.Millisecond))
		},
	}

	ResponseSize MetricInterface = &Metric{
		Name:        "response_size_bytes",
		Description: "The HTTP response sizes in bytes.",
		Type:        "summary",
		ExecFn: func(collector prometheus.Collector, startTime time.Time, ctx *gin.Context) {
			c := collector.(prometheus.Summary)
			c.Observe(float64(ctx.Writer.Size()))
		},
	}

	RequestSize MetricInterface = &Metric{
		Name:        "request_size_bytes",
		Description: "The HTTP request sizes in bytes.",
		Type:        "summary",
		ExecFn: func(collector prometheus.Collector, startTime time.Time, ctx *gin.Context) {
			c := collector.(prometheus.Summary)
			c.Observe(float64(computeApproximateRequestSize(ctx.Request)))
		},
	}

	SlowRequestTotal = func(slowTime time.Duration) MetricInterface {
		return &Metric{
			Name:        "slow_request_total",
			Description: "The slow HTTP request total.",
			Type:        "counter",
			ExecFn: func(collector prometheus.Collector, startTime time.Time, ctx *gin.Context) {
				if time.Since(startTime) > slowTime {
					c := collector.(prometheus.Counter)
					c.Inc()
				}
			},
		}
	}
)

// NewMetric associates prometheus.Collector based on Metric.Type
func NewMetric(m *Metric, subsystem string) prometheus.Collector {
	var metric prometheus.Collector
	switch m.Type {
	case "counter_vec":
		metric = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
			m.Args,
		)
	case "counter":
		metric = prometheus.NewCounter(
			prometheus.CounterOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
		)
	case "gauge_vec":
		metric = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
			m.Args,
		)
	case "gauge":
		metric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
		)
	case "histogram_vec":
		metric = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
			m.Args,
		)
	case "histogram":
		metric = prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
		)
	case "summary_vec":
		metric = prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
			m.Args,
		)
	case "summary":
		metric = prometheus.NewSummary(
			prometheus.SummaryOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
		)
	}
	return metric
}

// From https://github.com/DanielHeckrath/gin-prometheus/blob/master/gin_prometheus.go
func computeApproximateRequestSize(r *http.Request) int {
	s := 0
	if r.URL != nil {
		s = len(r.URL.Path)
	}

	s += len(r.Method)
	s += len(r.Proto)
	for name, values := range r.Header {
		s += len(name)
		for _, value := range values {
			s += len(value)
		}
	}
	s += len(r.Host)

	// N.B. r.Form and r.MultipartForm are assumed to be included in r.URL.

	if r.ContentLength != -1 {
		s += int(r.ContentLength)
	}
	return s
}
