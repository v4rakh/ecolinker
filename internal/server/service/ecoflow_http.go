package service

import (
	"context"
	"encoding/json"
	"fmt"
	"git.myservermanager.com/varakh/ecolinker/internal/server/dto"
	"git.myservermanager.com/varakh/ecolinker/internal/service_error"
	"git.myservermanager.com/varakh/go-ecoflow"
	"github.com/rs/zerolog/log"
	"regexp"
	"strings"
	"time"
)

type EcoFlowHttpService struct {
	httpClient *ecoflow.Client
}

func NewEcoFlowHttpService(accessKey string, secretKey string, url string) *EcoFlowHttpService {
	httpClient := ecoflow.NewEcoflowClient(accessKey, secretKey, ecoflow.WithBaseUrl(url))

	return &EcoFlowHttpService{
		httpClient: httpClient,
	}
}

// GetDevices retrieves available devices from EcoFlow
func (s *EcoFlowHttpService) GetDevices(ctx context.Context) (dto.EcoFlowDeviceData, error) {
	var err error
	var deviceList *ecoflow.DeviceListResponse

	if deviceList, err = s.httpClient.GetDeviceList(ctx); err != nil {
		return nil, service_error.NewServiceError(service_error.ErrCodeGeneral, fmt.Errorf("cannot get device list: %w", err))
	}

	data := make(dto.EcoFlowDeviceData, 0)
	for _, entry := range deviceList.Devices {
		data = append(data, dto.EcoFlowDeviceItem{
			SN:     entry.SN,
			Online: entry.Online,
		})
	}

	return data, nil
}

// GetParameters retrieves specific device parameters from EcoFlow
func (s *EcoFlowHttpService) GetParameters(ctx context.Context, sn string, params []string) (map[string]interface{}, error) {
	if sn == "" {
		return nil, service_error.ErrValidationNotBlank
	}
	if params == nil || len(params) == 0 {
		return nil, service_error.ErrValidationNotEmpty
	}

	var err error
	var res *ecoflow.GetCmdResponse
	if res, err = s.httpClient.GetDeviceParameters(ctx, sn, params); err != nil {
		return nil, service_error.NewServiceError(service_error.ErrCodeGeneral, fmt.Errorf("cannot get device parameters: %w", err))
	}

	return res.Data, nil
}

// GetAllParameters retrieves all device parameters from EcoFlow
func (s *EcoFlowHttpService) GetAllParameters(ctx context.Context, sn string) (map[string]interface{}, error) {
	if sn == "" {
		return nil, service_error.ErrValidationNotBlank
	}

	var err error
	var parameters map[string]interface{}
	if parameters, err = s.httpClient.GetDeviceAllParameters(ctx, sn); err != nil {
		return nil, service_error.NewServiceError(service_error.ErrCodeGeneral, fmt.Errorf("cannot get all device parameters: %w", err))
	}

	return parameters, nil
}

// GetBatteries like GetAllParameters, retrieves device battery information which is additionally encoded in response from EcoFlow (scans for bp_addr)
func (s *EcoFlowHttpService) GetBatteries(ctx context.Context, sn string) (map[string]interface{}, error) {
	if sn == "" {
		return nil, service_error.ErrValidationNotBlank
	}

	var err error
	var deviceParams map[string]interface{}
	if deviceParams, err = s.GetAllParameters(ctx, sn); err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`^bp_addr\.\w+$`)
	ignoreKey := "bp_addr.updateTime"
	prefix := "bp_addr."

	filtered := make(map[string]interface{})
	for k, v := range deviceParams {
		if re.MatchString(k) && k != ignoreKey {

			jsonStr, ok := v.(string)
			if !ok {
				log.Warn().Msgf("Battery information for '%s' is not a valid string, skipping...", v)
				continue
			}
			var obj map[string]interface{}
			if err = json.Unmarshal([]byte(jsonStr), &obj); err != nil {
				log.Warn().Msgf("Battery information for '%s' cannot be converted from JSON, skipping...", v)
				continue
			}

			filtered[strings.TrimPrefix(k, prefix)] = obj
		}
	}

	return filtered, nil
}

// GetHistory retrieves historical device data from EcoFlow (PowerOcean only)
func (s *EcoFlowHttpService) GetHistory(ctx context.Context, sn string, beginTime time.Time, endTime time.Time) (dto.EcoFlowHistoryData, error) {
	if sn == "" {
		return nil, service_error.ErrValidationNotBlank
	}
	if beginTime.IsZero() || endTime.IsZero() {
		return nil, service_error.ErrValidationNotEmpty
	}

	var err error
	var historicalDataRes *ecoflow.GetCmdHistoryResponse
	if historicalDataRes, err = s.httpClient.GetDeviceHistory(ctx, sn, beginTime, endTime); err != nil {
		return nil, service_error.NewServiceError(service_error.ErrCodeGeneral, fmt.Errorf("cannot get device history: %w", err))
	}

	data := make(dto.EcoFlowHistoryData, 0)
	for _, entry := range historicalDataRes.Data.Data {
		if entry.IndexValue != nil {
			data = append(data, dto.EcoFlowHistoryItem{
				IndexName:  entry.IndexName,
				IndexValue: entry.IndexValue,
				Unit:       entry.Unit,
			})
		}
	}

	return data, nil
}
