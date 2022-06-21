package gin_metrics

import (
	"bytes"
	"context"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/gin-gonic/gin"
	"github.com/no-mole/neptune/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	DefaultPushInterval = 5 * time.Second
	DefaultJobName      = "gin"
	DefaultMetricsPath  = "metrics"
)

//New new gin Prometheus metric collector
func New(opts ...Option) *Prometheus {
	h, _ := os.Hostname()
	p := &Prometheus{
		router:       nil,
		exportRouter: nil,
		metrics:      make([]MetricInterface, 0),
		metricsPath:  DefaultMetricsPath,
		jobName:      DefaultJobName,
		PushInterval: DefaultPushInterval,
		Logger:       logger.GetLogger(),
		LoggerTag:    "gin-metrics",
		InstanceName: h,
	}
	for _, opt := range opts {
		opt(p)
	}

	if p.router != nil {
		p.router.Use(p.HandlerFunc())
	}

	if p.exportRouter != nil {
		if p.accounts != nil {
			p.exportRouter.GET(p.metricsPath, gin.BasicAuth(p.accounts), exportHandle())
			for username, password := range p.accounts {
				p.metricsBasicAuth = base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
				break
			}
		} else {
			p.exportRouter.GET(p.metricsPath, exportHandle())
		}

	}

	p.registerMetrics()

	if p.PushGatewayURL != "" {
		p.startPush()
	}

	return p
}

// Prometheus contains the metrics gathered by the instance and its path
type Prometheus struct {

	//origin router
	router *gin.Engine

	//exportRouter export metric router,default is router
	exportRouter *gin.Engine

	//exportRouter basic auth
	accounts gin.Accounts

	//from accounts
	metricsBasicAuth string

	// Push interval
	PushInterval time.Duration

	//PushGatewayURL Push Gateway URL in format http://domain:port
	// where JOB NAME can be any string of your choice
	PushGatewayURL string

	//metrics
	metrics []MetricInterface

	//metricsPath export metric path,default "metrics"
	metricsPath string

	// jobName push gateway job name, defaults to "gin"
	jobName string

	Logger logger.Logger

	//LoggerTag
	LoggerTag string

	//InstanceName default is hostname
	InstanceName string

	// Local metrics URL where metrics are fetched from, this could be ommited in the future
	MetricsURL string
}

// HandlerFunc defines handler function for middleware
func (p *Prometheus) HandlerFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		//Ignore self
		if c.Request.URL.Path == p.metricsPath {
			return
		}
		start := time.Now()
		c.Next()
		for _, metric := range p.metrics {
			metric.Exec(start, c)
		}
	}
}

func (p *Prometheus) fetchMetrics() []byte {
	req, _ := http.NewRequest(http.MethodGet, p.MetricsURL, nil)
	if p.metricsBasicAuth != "" {
		req.Header.Set("Authorization", p.metricsBasicAuth)
	}
	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	return body
}

func (p *Prometheus) GetPushGatewayURL() string {
	return p.PushGatewayURL + "/metrics/job/" + p.jobName + "/instance/" + p.InstanceName
}

func (p *Prometheus) sendToPushGateway(metrics []byte) {
	if p.MetricsURL == "" {
		return
	}
	req, err := http.NewRequest("POST", p.GetPushGatewayURL(), bytes.NewBuffer(metrics))
	client := &http.Client{}
	if _, err = client.Do(req); err != nil {
		p.Logger.Error(context.Background(), p.LoggerTag, err)
	}
}

func (p *Prometheus) startPush() {
	ticker := time.NewTicker(p.PushInterval)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			p.sendToPushGateway(p.fetchMetrics())
		}
	}()
}

func (p *Prometheus) registerMetrics() {
	for _, metric := range p.metrics {
		if err := prometheus.Register(metric.Collector(p.jobName)); err != nil {
			p.Logger.Error(context.Background(), p.LoggerTag, err)
		}
	}
}

func exportHandle() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
