package main

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	gin_metrics "github.com/no-mole/gin-metrics"
	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	r := gin.New()

	//  Optional custom metrics list
	//  Type Options:
	//	counter, counter_vec, gauge, gauge_vec,
	//	histogram, histogram_vec, summary, summary_vec
	customMetric := &gin_metrics.Metric{
		Name:        "test_metric",
		Description: "Counter test metric",
		Type:        "counter",
		ExecFn: func(collector prometheus.Collector, startTime time.Time, ctx *gin.Context) {
			c := collector.(prometheus.Counter)
			c.Inc()
		},
	}

	gin_metrics.New(
		gin_metrics.WithRouter(r),
		gin_metrics.WithExportRouter(r),
		gin_metrics.WithMetrics(customMetric),
		gin_metrics.WithMetrics(gin_metrics.RequestTotal, gin_metrics.RequestSize, gin_metrics.ResponseSize, gin_metrics.RequestDuration),
		gin_metrics.WithMetrics(gin_metrics.SlowRequestTotal(time.Second)),
		gin_metrics.WithMetrics(gin_metrics.RequestTotalWithMapper(func(ctx *gin.Context) string {
			url := ctx.Request.URL.Path
			for _, p := range ctx.Params {
				if p.Key == "name" {
					url = strings.Replace(url, p.Value, ":name", 1)
					break
				}
			}
			return url
		})),
	)

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, "Hello world!")
	})

	r.GET("/hello", func(c *gin.Context) {
		c.JSON(200, "world!")
	})

	r.Run(":80")
}
