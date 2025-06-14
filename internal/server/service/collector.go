package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"git.myservermanager.com/varakh/ecolinker/internal/app"
	"git.myservermanager.com/varakh/ecolinker/internal/float"
	jsoninternal "git.myservermanager.com/varakh/ecolinker/internal/json"
	"git.myservermanager.com/varakh/ecolinker/internal/server/constant"
	"git.myservermanager.com/varakh/ecolinker/internal/server/dto"
	"git.myservermanager.com/varakh/ecolinker/internal/server/model"
	"git.myservermanager.com/varakh/ecolinker/internal/server/repository"
	"git.myservermanager.com/varakh/ecolinker/internal/server/service_error"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"
)

type CollectorService struct {
	ecoFlowHttpService *EcoFlowHttpService
	mqttForwardService *MqttForwardService
	taskService        *TaskService
	prometheusService  *PrometheusService
	repo               repository.CollectorRepository
}

const (
	jobTagCollectors = "COLLECTORS"
)

func NewCollectorService(e *EcoFlowHttpService, m *MqttForwardService, t *TaskService, p *PrometheusService, r repository.CollectorRepository) *CollectorService {
	return &CollectorService{
		ecoFlowHttpService: e,
		mqttForwardService: m,
		taskService:        t,
		prometheusService:  p,
		repo:               r,
	}
}

// Init initializes the service, bootstrapping necessary configuration, should be called directly after NewCollectorService
func (s *CollectorService) Init() error {
	var err error
	var collectors []*model.Collector
	if collectors, err = s.GetAll(); err != nil {
		return err
	}

	for _, c := range collectors {
		if err = s.start(c); err != nil {
			zap.L().Sugar().Errorf("Could not initialize collector '%s': %v", c.ID.String(), err)
			continue
		}
	}

	return nil
}

// GetAll retrieves collector information about all collectors
func (s *CollectorService) GetAll() ([]*model.Collector, error) {
	return s.repo.FindAll("")
}

// Get retrieves collector information by device SN, if device SN is blank, all collectors are returned
func (s *CollectorService) Get(deviceSN string) ([]*model.Collector, error) {
	return s.repo.FindAll(deviceSN)
}

// GetById retrieves information by collector ifd
func (s *CollectorService) GetById(id string) (*model.Collector, error) {
	if id == "" {
		return nil, service_error.ErrValidationNotBlank
	}

	e, err := s.repo.FindById(id)

	if err != nil {
		return nil, err
	}

	return e, nil
}

// Create creates a new collector
func (s *CollectorService) Create(deviceSN string, kind constant.CollectorKind, frequency string, parameters []string) (*model.Collector, error) {
	if kind == "" || deviceSN == "" || frequency == "" {
		return nil, service_error.ErrValidationNotBlank
	}

	var err error
	var frequencyParsed time.Duration
	if frequencyParsed, err = time.ParseDuration(frequency); err != nil {
		return nil, service_error.NewServiceError(service_error.ErrCodeIllegalArgument, fmt.Errorf("not a valid frequency: %w", err))
	}

	var p interface{}
	switch kind {
	case constant.CollectorKindDeviceParameters:
		if parameters == nil {
			return nil, service_error.ErrValidationNotEmpty
		}
		p = dto.CollectorEcoFlowHttpDeviceParameterPayload{Parameters: parameters}
	default:
		return nil, service_error.NewServiceError(service_error.ErrCodeGeneral, errors.New("invalid kind provided"))
	}

	var e *model.Collector
	if e, err = s.repo.Create(deviceSN, kind.String(), frequencyParsed.String(), p); err != nil {
		return nil, err
	}

	zap.L().Sugar().Debugf("Created collector '%+v'", e)

	if err = s.start(e); err != nil {
		zap.L().Sugar().Errorf("Could not start collector '%s': %v", e.ID.String(), err)
	}

	return e, nil
}

// Update updates an existing collector
func (s *CollectorService) Update(id string, deviceSN string, kind constant.CollectorKind, frequency string, parameters []string) (*model.Collector, error) {
	if id == "" || kind == "" || deviceSN == "" || frequency == "" {
		return nil, service_error.ErrValidationNotBlank
	}

	var err error
	var oldEntity *model.Collector

	if oldEntity, err = s.GetById(id); err != nil {
		return nil, err
	}

	var frequencyParsed time.Duration
	if frequencyParsed, err = time.ParseDuration(frequency); err != nil {
		return nil, service_error.NewServiceError(service_error.ErrCodeIllegalArgument, fmt.Errorf("not a valid frequency: %w", err))
	}

	var p interface{}
	switch kind {
	case constant.CollectorKindDeviceParameters:
		if parameters == nil {
			return nil, service_error.ErrValidationNotEmpty
		}
		p = dto.CollectorEcoFlowHttpDeviceParameterPayload{Parameters: parameters}
	default:
		return nil, service_error.NewServiceError(service_error.ErrCodeGeneral, errors.New("invalid kind provided"))
	}

	var newEntity *model.Collector
	if newEntity, err = s.repo.Update(id, deviceSN, kind.String(), frequencyParsed.String(), p); err != nil {
		return nil, err
	}

	s.taskService.CancelByTag(s.runIdentifier(oldEntity.ID))

	zap.L().Sugar().Debugf("Modified collector '%v'", id)

	if err = s.start(newEntity); err != nil {
		zap.L().Sugar().Errorf("Could not start collector '%s': %v", newEntity.ID.String(), err)
	}

	return newEntity, nil
}

// Delete deletes an existing collector
func (s *CollectorService) Delete(id string) error {
	if id == "" {
		return service_error.ErrValidationNotBlank
	}

	e, err := s.GetById(id)
	if err != nil {
		return err
	}

	if _, err = s.repo.Delete(e.ID.String()); err != nil {
		return err
	}

	s.taskService.CancelByTag(s.runIdentifier(e.ID))

	zap.L().Sugar().Debugf("Deleted collector '%v'", id)

	return nil
}

// start starts a collector
func (s *CollectorService) start(c *model.Collector) error {
	var err error
	var duration time.Duration
	if duration, err = time.ParseDuration(c.Frequency); err != nil {
		return service_error.NewServiceError(service_error.ErrCodeGeneral, fmt.Errorf("could not parse frequency collector '%s': %w", c.ID, err))
	}

	collectorIdentifier := s.runIdentifier(c.ID)
	if _, err = s.taskService.Enqueue(gocron.DurationJob(duration), gocron.NewTask(s.run, c), collectorIdentifier, gocron.WithDisabledDistributedJobLocker(true), gocron.WithTags(jobTagCollectors, collectorIdentifier)); err != nil {
		return service_error.NewServiceError(service_error.ErrCodeGeneral, fmt.Errorf("could not start collector '%s': %w", c.ID, err))
	}

	return nil
}

// run runs a collector
func (s *CollectorService) run(ctx context.Context, c *model.Collector) {
	received := float64(time.Now().Unix())

	var err error
	var pb []byte
	if pb, err = c.Payload.MarshalJSON(); err != nil {
		zap.L().Sugar().Errorf("Could not unmarshal collector payload '%s': %v", c.ID, err)
		return
	}

	generalMetricLabels := []string{"device", "kind", "id"}
	genericMetricLabelValues := []string{c.DeviceSN, c.Kind, c.ID.String()}

	if promErr := s.prometheusService.RegisterCounter(constant.MetricCollectorInvocations, constant.MetricCollectorInvocationsHelp, generalMetricLabels); promErr != nil {
		if !errors.Is(promErr, ErrPrometheusMetricAlreadyRegistered) {
			zap.L().Sugar().Warnf("Unable to register prometheus metric for '%s': %v", constant.MetricCollectorInvocations, promErr)
		}
	}
	if promErr := s.prometheusService.IncreaseCounter(constant.MetricCollectorInvocations, genericMetricLabelValues); promErr != nil {
		zap.L().Sugar().Warnf("Unable to set prometheus metric for '%s': %v", constant.MetricCollectorInvocations, promErr)
	}

	if promErr := s.prometheusService.RegisterGauge(constant.MetricCollectorInvocationLast, constant.MetricCollectorInvocationLastHelp, generalMetricLabels); promErr != nil {
		if !errors.Is(promErr, ErrPrometheusMetricAlreadyRegistered) {
			zap.L().Sugar().Warnf("Unable to register prometheus metric for '%s': %v", constant.MetricCollectorInvocationLast, promErr)
		}
	}
	if promErr := s.prometheusService.SetGauge(constant.MetricCollectorInvocationLast, genericMetricLabelValues, received); promErr != nil {
		zap.L().Sugar().Warnf("Unable to set prometheus metric for '%s': %v", constant.MetricCollectorInvocationLast, promErr)
	}

	switch c.Kind {
	case constant.CollectorKindDeviceParameters.String():
		var p dto.CollectorEcoFlowHttpDeviceParameterPayload
		if p, err = jsoninternal.UnmarshalGenericJSON[dto.CollectorEcoFlowHttpDeviceParameterPayload](pb); err != nil {
			zap.L().Sugar().Errorf("Could not unmarshal collector payload '%s'", c.ID)
			return
		}

		var data map[string]interface{}
		if data, err = s.ecoFlowHttpService.GetParameters(ctx, c.DeviceSN, p.Parameters); err != nil {
			zap.L().Sugar().Errorf("Could not get device parameters for collector '%s'", c.ID)
			return
		}

		zap.L().Sugar().Debugf("Collector's '%s' result: %+v", c.ID, data)

		var dataBytes []byte
		if dataBytes, err = json.Marshal(data); err != nil {
			zap.L().Sugar().Errorf("Could not parse device parameters for collector '%s'", c.ID)
		} else {
			forwardTopic := fmt.Sprintf("/%s/%s/%s", strings.ToLower(app.Name), c.DeviceSN, strings.ToLower(c.Kind))
			if err = s.mqttForwardService.Publish(forwardTopic, 0, true, dataBytes); err != nil {
				zap.L().Sugar().Warnf("Unable to forward collector '%s' device parameters: %v", c.ID, err)
			}
		}

		flattenedPayload := make(map[string]interface{})
		flatten(data, flattenedPayload)

		extracted := extractIndicesAndValueList(flattenedPayload)

		for _, valueMap := range extracted {
			metricValue, ok := float.ToFloat(valueMap.Value)

			if !ok {
				zap.L().Sugar().Warnf("Unable to cast value to prometheus metric type: %v", metricValue)
				continue
			}

			metricKey := fmt.Sprintf("%s_%s", strings.ToLower(app.Name), valueMap.Key)

			metricLabelKeys := []string{"device"}
			metricLabelKeys = append(metricLabelKeys, slices.Collect(maps.Keys(valueMap.Indices))...)
			metricLabelValues := []string{c.DeviceSN}

			indicesValues := slices.Collect(maps.Values(valueMap.Indices))
			for _, v := range indicesValues {
				metricLabelValues = append(metricLabelValues, strconv.Itoa(v))
			}

			if promErr := s.prometheusService.RegisterGauge(metricKey, metricHelp, metricLabelKeys); promErr != nil {
				if !errors.Is(promErr, ErrPrometheusMetricAlreadyRegistered) {
					zap.L().Sugar().Warnf("Unable to register prometheus metric for '%s': %v", valueMap.Key, promErr)
					continue
				}
			}
			if promErr := s.prometheusService.SetGauge(metricKey, metricLabelValues, metricValue); promErr != nil {
				zap.L().Sugar().Warnf("Unable to set prometheus metric for '%s': %v", valueMap.Key, promErr)
				continue
			}
		}

		break
	default:
		zap.L().Sugar().Errorf("No collector kind '%s' found", c.Kind)
		s.taskService.CancelByTag(s.runIdentifier(c.ID))
		return
	}
}

// runIdentifier returns identifier for runs
func (s *CollectorService) runIdentifier(id uuid.UUID) string {
	return fmt.Sprintf("COLLECTOR-%s", id.String())
}
