package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Registry struct {
	Registry        *prometheus.Registry
	RegistryOptions *prometheus.Opts
}

func NewRegistry(options *prometheus.Opts) *Registry {
	registry := prometheus.NewRegistry()

	return &Registry{
		Registry:        registry,
		RegistryOptions: options,
	}
}

// MustRegister implements Registerer.
func (reg *Registry) MustRegister(cs ...prometheus.Collector) {
	for _, c := range cs {
		if err := reg.Register(c); err != nil {
			panic(err)
		}
	}
}

func (reg *Registry) Register(collector prometheus.Collector) prometheus.Collector {

	if err := reg.Registry.Register(collector); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			collector = are.ExistingCollector
		} else {
			panic(err)
		}
	}

	return collector
}

// RegisterCounter 注册一个累加器指标, 可用于请求数/网络流量总数
func (reg *Registry) RegisterCounter(name, help string, labels ...string) *prometheus.CounterVec {
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        prometheus.BuildFQName(reg.RegistryOptions.Namespace, reg.RegistryOptions.Subsystem, name),
			Help:        help,
			ConstLabels: reg.RegistryOptions.ConstLabels,
		},
		labels,
	)
	counter = reg.Register(counter).(*prometheus.CounterVec)
	return counter
}

// RegisterGauge 注册一个仪表盘 瞬时指标, 可增可减。 比如并发数
func (reg *Registry) RegisterGauge(name, help string, labels ...string) *prometheus.GaugeVec {
	gauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:        prometheus.BuildFQName(reg.RegistryOptions.Namespace, reg.RegistryOptions.Subsystem, name),
			Help:        help,
			ConstLabels: reg.RegistryOptions.ConstLabels,
		},
		labels,
	)
	gauge = reg.Register(gauge).(*prometheus.GaugeVec)
	return gauge
}

// RegisterHistogram 注册一个累积直方图
func (reg *Registry) RegisterHistogram(name, help string, buckets []float64, labels ...string) *prometheus.HistogramVec {
	histogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        prometheus.BuildFQName(reg.RegistryOptions.Namespace, reg.RegistryOptions.Subsystem, name),
			Help:        help,
			Buckets:     buckets,
			ConstLabels: reg.RegistryOptions.ConstLabels,
		},
		labels,
	)
	histogram = reg.Register(histogram).(*prometheus.HistogramVec)
	return histogram
}

// RegisterSummary 注册一个摘要
func (reg *Registry) RegisterSummary(name, help string, objectives map[float64]float64, labels ...string) *prometheus.SummaryVec {
	summary := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:        prometheus.BuildFQName(reg.RegistryOptions.Namespace, reg.RegistryOptions.Subsystem, name),
			Help:        help,
			Objectives:  objectives,
			ConstLabels: reg.RegistryOptions.ConstLabels,
		},
		labels,
	)
	summary = reg.Register(summary).(*prometheus.SummaryVec)
	return summary
}
