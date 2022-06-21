package gin_metrics

import (
	"time"

	"github.com/no-mole/neptune/logger"

	"github.com/gin-gonic/gin"
)

type Option func(p *Prometheus)

func WithRouter(r *gin.Engine) Option {
	return func(p *Prometheus) {
		p.router = r
	}
}

func WithExportRouter(r *gin.Engine) Option {
	return func(p *Prometheus) {
		p.exportRouter = r
	}
}

func WithMetricsPath(s string) Option {
	return func(p *Prometheus) {
		p.metricsPath = s
	}
}

func WithMetrics(m ...MetricInterface) Option {
	return func(p *Prometheus) {
		p.metrics = append(p.metrics, m...)
	}
}

func WithPushGatewayUrl(pgw string) Option {
	return func(p *Prometheus) {
		p.PushGatewayURL = pgw
	}
}

func WithExportAccounts(accounts gin.Accounts) Option {
	return func(p *Prometheus) {
		p.accounts = accounts
	}
}

func WithPushInterval(t time.Duration) Option {
	return func(p *Prometheus) {
		p.PushInterval = t
	}
}

func WithPushPushGatewayURL(url string) Option {
	return func(p *Prometheus) {
		p.PushGatewayURL = url
	}
}

func WithJobName(jobName string) Option {
	return func(p *Prometheus) {
		p.jobName = jobName
	}
}

func WithLogger(l logger.Logger) Option {
	return func(p *Prometheus) {
		p.Logger = l
	}
}

func WithLoggerTag(tag string) Option {
	return func(p *Prometheus) {
		p.LoggerTag = tag
	}
}

func WithInstanceName(name string) Option {
	return func(p *Prometheus) {
		p.InstanceName = name
	}
}

func WithMetricsURL(url string) Option {
	return func(p *Prometheus) {
		p.MetricsURL = url
	}
}
