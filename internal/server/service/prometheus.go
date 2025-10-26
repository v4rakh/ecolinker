package service

import (
	"errors"
	"fmt"
	"git.myservermanager.com/varakh/ecolinker/internal/server/config"
	"git.myservermanager.com/varakh/ecolinker/internal/service_error"
	"git.myservermanager.com/varakh/ecolinker/internal/str"
	"github.com/Depado/ginprom"
	"github.com/gin-gonic/gin"
	"sync"
)

type customGauges struct {
	sync.RWMutex
	values map[string][]string
}
type customCounters struct {
	sync.RWMutex
	values map[string][]string
}

var (
	ErrPrometheusMetricAlreadyRegistered = service_error.NewServiceError(service_error.ErrCodeConflict, errors.New("metric already exists"))
)

type PrometheusService struct {
	router           *gin.Engine
	prometheus       *ginprom.Prometheus
	prometheusConfig *config.Prometheus
	customGauges     customGauges
	customCounters   customCounters
}

func NewPrometheusService(e *gin.Engine, c *config.Prometheus) *PrometheusService {
	var p *ginprom.Prometheus

	if !c.Enabled {
		return &PrometheusService{
			prometheus:       p,
			prometheusConfig: c,
		}
	}

	path := fmt.Sprintf("%s%s", c.BasePath, c.Path)
	if c.SecureTokenEnabled {
		p = ginprom.New(
			ginprom.Engine(e),
			ginprom.Namespace(""),
			ginprom.Subsystem(""),
			ginprom.Path(path),
			ginprom.Ignore(path),
			ginprom.Token(c.SecureToken),
		)
	} else {
		p = ginprom.New(
			ginprom.Engine(e),
			ginprom.Namespace(""),
			ginprom.Subsystem(""),
			ginprom.Ignore(path),
			ginprom.Path(path),
		)
	}

	s := &PrometheusService{
		prometheus:       p,
		prometheusConfig: c,
	}

	s.customGauges.values = make(map[string][]string)
	s.customCounters.values = make(map[string][]string)

	return s
}

// GetProm returns the to be instrumented prometheus registry for gin
func (s *PrometheusService) GetProm() *ginprom.Prometheus {
	return s.prometheus
}

// RegisterGaugeNoLabels registers a metric
func (s *PrometheusService) RegisterGaugeNoLabels(name string, help string) error {
	return s.RegisterGauge(name, help, make([]string, 0))
}

// RegisterGauge registers a metric
func (s *PrometheusService) RegisterGauge(name string, help string, labels []string) error {
	if !s.prometheusConfig.Enabled {
		return nil
	}

	if name == "" || help == "" {
		return service_error.ErrValidationNotBlank
	}

	s.customGauges.Lock()
	defer s.customGauges.Unlock()

	if _, exists := s.customGauges.values[name]; exists && str.AllContained(s.customGauges.values[name], labels) {
		return ErrPrometheusMetricAlreadyRegistered
	}

	s.prometheus.AddCustomGauge(name, help, labels)
	s.customGauges.values[name] = labels

	return nil
}

// RegisterCounterNoLabels registers a metric
func (s *PrometheusService) RegisterCounterNoLabels(name string, help string) error {
	return s.RegisterCounter(name, help, make([]string, 0))
}

// RegisterCounter registers a metric
func (s *PrometheusService) RegisterCounter(name string, help string, labels []string) error {
	if !s.prometheusConfig.Enabled {
		return nil
	}

	if name == "" || help == "" {
		return service_error.ErrValidationNotBlank
	}

	s.customCounters.Lock()
	defer s.customCounters.Unlock()

	if _, exists := s.customCounters.values[name]; exists && str.AllContained(s.customCounters.values[name], labels) {
		return ErrPrometheusMetricAlreadyRegistered
	}

	s.prometheus.AddCustomCounter(name, help, labels)
	s.customCounters.values[name] = labels

	return nil
}

// SetGaugeNoLabels sets a metric
func (s *PrometheusService) SetGaugeNoLabels(name string, value float64) error {
	return s.SetGauge(name, make([]string, 0), value)
}

// SetGauge sets a metric
func (s *PrometheusService) SetGauge(name string, labelValues []string, value float64) error {
	if !s.prometheusConfig.Enabled {
		return nil
	}

	if name == "" {
		return service_error.ErrValidationNotBlank
	}

	if err := s.prometheus.SetGaugeValue(name, labelValues, value); err != nil {
		return err
	}

	return nil
}

// IncreaseCounterNoLabels sets a metric
func (s *PrometheusService) IncreaseCounterNoLabels(name string) error {
	return s.IncreaseCounter(name, make([]string, 0))
}

// IncreaseCounter sets a metric
func (s *PrometheusService) IncreaseCounter(name string, labelValues []string) error {
	if !s.prometheusConfig.Enabled {
		return nil
	}

	if name == "" {
		return service_error.ErrValidationNotBlank
	}

	if err := s.prometheus.IncrementCounterValue(name, labelValues); err != nil {
		return err
	}

	return nil
}
