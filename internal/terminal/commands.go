package terminal

import (
	"context"
	"errors"
	"fmt"
	"git.myservermanager.com/varakh/ecolinker/api"
	"git.myservermanager.com/varakh/ecolinker/internal/app"
	"git.myservermanager.com/varakh/ecolinker/internal/server/constant"
	"git.myservermanager.com/varakh/ecolinker/internal/str"
	"github.com/BurntSushi/toml"
	"github.com/adrg/xdg"
	"github.com/go-resty/resty/v2"
	"github.com/urfave/cli/v3"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

const (
	envConfig   = "ECOLINKER_CONFIG"
	envUrl      = "ECOLINKER_URL"
	envUser     = "ECOLINKER_USER"
	envPassword = "ECOLINKER_PASSWORD"

	flagConfig  = "config"
	flagUrl     = "url"
	flagUser    = "user"
	flagPass    = "pass"
	flagRaw     = "raw"
	flagTimeout = "timeout"

	flagSerialNumber  = "serial-number"
	flagLabel         = "label"
	flagBeginTime     = "begin-time"
	flagEndTime       = "end-time"
	flagDeviceKind    = "device-kind"
	flagTopicKind     = "topic-kind"
	flagCollectorKind = "collector-kind"
	flagFrequency     = "frequency"
	flagParameters    = "parameter"

	ecoFlowUrlPath           = "/api/v1/ecoflow"
	devicesUrlPath           = "/api/v1/devices"
	mqttSubscriptionsUrlPath = "/api/v1/mqtt-subscriptions"
	collectorsUrlPath        = "/api/v1/collectors"

	errorParse = "error while parsing response: %v"
	errorFlush = "error during while flushing response: %v"
	errorCall  = "error during call: %w"
)

var (
	configPath    string
	instanceUrl   string
	user          string
	password      string
	raw           bool
	timeout       time.Duration
	serialNumber  string
	label         string
	deviceKind    string
	topicKind     string
	collectorKind string
	frequency     time.Duration
	beginTime     string
	endTime       string
	parameters    []string

	configPathFlag = &cli.StringFlag{
		Name:        flagConfig,
		Usage:       "Path to EcoLinker's TOML configuration file",
		Required:    false,
		Aliases:     []string{"c"},
		Sources:     cli.EnvVars(envConfig),
		Destination: &configPath,
	}
	urlFlag = &cli.StringFlag{
		Name:        flagUrl,
		Usage:       "EcoLinker instance URL like http://192.168.0.10:8080 or an explicit domain like https://ecolinker.domain.tld",
		Required:    false,
		Aliases:     []string{"i"},
		Sources:     cli.EnvVars(envUrl),
		Destination: &instanceUrl,
	}
	userFlag = &cli.StringFlag{
		Name:        flagUser,
		Usage:       "EcoLinker instance user",
		Required:    false,
		Aliases:     []string{"u"},
		Sources:     cli.EnvVars(envUser),
		Destination: &user,
	}
	passwordFlag = &cli.StringFlag{
		Name:        flagPass,
		Usage:       "EcoLinker instance password",
		Required:    false,
		Aliases:     []string{"p"},
		Sources:     cli.EnvVars(envPassword),
		Destination: &password,
	}
	rawFlag = &cli.BoolFlag{
		Name:        flagRaw,
		Usage:       "Returns raw JSON data on success",
		Aliases:     []string{"r"},
		Value:       false,
		Destination: &raw,
	}
	timeoutFlag = &cli.DurationFlag{
		Name:        flagTimeout,
		Usage:       "Optional flag to determine maximum timeout to query EcoLinker",
		Aliases:     []string{"to"},
		Required:    false,
		Value:       10 * time.Second,
		Destination: &timeout,
	}
	snFlag = &cli.StringFlag{
		Name:        flagSerialNumber,
		Usage:       "Device's serial number",
		Required:    false,
		Aliases:     []string{"sn"},
		Destination: &serialNumber,
	}
	labelFlag = &cli.StringFlag{
		Name:        flagLabel,
		Usage:       "Device's label",
		Required:    true,
		Aliases:     []string{"l"},
		Destination: &label,
	}
	deviceKindFlag = &cli.StringFlag{
		Name:        flagDeviceKind,
		Usage:       fmt.Sprintf("Device's kind, one of %v", constant.DeviceKindNames()),
		Required:    true,
		Aliases:     []string{"dk"},
		Destination: &deviceKind,
	}
	topicKindFlag = &cli.StringFlag{
		Name:        flagTopicKind,
		Usage:       fmt.Sprintf("Topic kind, one of %v", constant.TopicKindNames()),
		Required:    true,
		Aliases:     []string{"tk"},
		Destination: &topicKind,
	}
	collectorKindFlag = &cli.StringFlag{
		Name:        flagCollectorKind,
		Usage:       fmt.Sprintf("Collector kind, one of %v", constant.CollectorKindNames()),
		Required:    true,
		Aliases:     []string{"ck"},
		Destination: &collectorKind,
	}
	frequencyFlag = &cli.DurationFlag{
		Name:        flagFrequency,
		Usage:       "Optional flag to determine how frequently the collector runs (if collector queries EcoFlow's HTTP API, ensure to pick a reasonable duration to not spam EcoFlow's services)",
		Aliases:     []string{"fq"},
		Required:    false,
		Value:       45 * time.Second,
		Destination: &frequency,
	}
	beginTimeFlag = &cli.StringFlag{
		Name:        flagBeginTime,
		Usage:       fmt.Sprintf("The begin time with layout '%s', total time span cannot exceed one week", time.DateTime),
		Aliases:     []string{"bt"},
		Destination: &beginTime,
	}
	endTimeFlag = &cli.StringFlag{
		Name:        flagEndTime,
		Usage:       fmt.Sprintf("The end time with layout '%s', total time span cannot exceed one week", time.DateTime),
		Aliases:     []string{"et"},
		Destination: &endTime,
	}
	parametersFlag = &cli.StringSliceFlag{
		Name:        flagParameters,
		Usage:       "Parameters to query, if none given, all will be queried, you can provide multiple --parameter or its aliases",
		Aliases:     []string{"pn", "par", "pp"},
		Destination: &parameters,
	}
	EcoFlowDevicesListCmd = &cli.Command{
		Name:  "ls",
		Usage: "Lists devices on your EcoFlow account",
		Flags: []cli.Flag{
			configPathFlag,
			urlFlag,
			userFlag,
			passwordFlag,
			rawFlag,
			timeoutFlag,
		},
		Action: ecoFlowDevicesList,
	}
	EcoFlowDeviceParametersCmd = &cli.Command{
		Name:  "ps",
		Usage: "Device's parameters queried from EcoFlow",
		Flags: []cli.Flag{
			snFlag,
			configPathFlag,
			urlFlag,
			userFlag,
			passwordFlag,
			parametersFlag,
			rawFlag,
			timeoutFlag,
		},
		Action: ecoFlowDeviceParameters,
	}
	EcoFlowDeviceBatteriesCmd = &cli.Command{
		Name:        "bs",
		Usage:       "Device's batteries queried from EcoFlow",
		Description: "Filters specific keys of all parameters for a device",
		Flags: []cli.Flag{
			snFlag,
			configPathFlag,
			urlFlag,
			userFlag,
			passwordFlag,
			rawFlag,
			timeoutFlag,
		},
		Action: ecoFlowDeviceBatteries,
	}
	EcoFlowDeviceHistoryCmd = &cli.Command{
		Name:  "hs",
		Usage: "Device's historical data queried from EcoFlow (PowerOcean only)",
		Flags: []cli.Flag{
			snFlag,
			beginTimeFlag,
			endTimeFlag,
			configPathFlag,
			urlFlag,
			userFlag,
			passwordFlag,
			rawFlag,
			timeoutFlag,
		},
		Action: ecoFlowDeviceHistory,
	}
	EcoFlowStatusCmd = &cli.Command{
		Name:  "status",
		Usage: "EcoLinker's status regarding its connection to EcoFlow's MQTT broker",
		Flags: []cli.Flag{
			snFlag,
			configPathFlag,
			urlFlag,
			userFlag,
			passwordFlag,
			rawFlag,
			timeoutFlag,
		},
		Action: ecoFlowBrokerStatus,
	}
	DevicesListCmd = &cli.Command{
		Name:  "ls",
		Usage: "Lists tracked devices",
		Flags: []cli.Flag{
			configPathFlag,
			urlFlag,
			userFlag,
			passwordFlag,
			rawFlag,
			timeoutFlag,
		},
		Action: devicesList,
	}
	DevicesAddCmd = &cli.Command{
		Name:  "add",
		Usage: "Adds a device which enables you to listen for MQTT messages from EcoFlow",
		Flags: []cli.Flag{
			snFlag,
			labelFlag,
			deviceKindFlag,
			configPathFlag,
			urlFlag,
			userFlag,
			passwordFlag,
			rawFlag,
			timeoutFlag,
		},
		Action: devicesAdd,
	}
	DevicesRmCmd = &cli.Command{
		Name:  "rm",
		Usage: "Removes a device, remember that all associated MQTT subscriptions are deleted with it",
		Flags: []cli.Flag{
			snFlag,
			configPathFlag,
			urlFlag,
			userFlag,
			passwordFlag,
			rawFlag,
			timeoutFlag,
		},
		Action: devicesRemove,
	}
	SubsListCmd = &cli.Command{
		Name:  "ls",
		Usage: "Lists MQTT subscriptions for tracked devices",
		Flags: []cli.Flag{
			configPathFlag,
			urlFlag,
			userFlag,
			passwordFlag,
			rawFlag,
			timeoutFlag,
		},
		Action: subsList,
	}
	SubsAddCmd = &cli.Command{
		Name:  "add",
		Usage: "Adds a subscription for a tracked device",
		Flags: []cli.Flag{
			snFlag,
			topicKindFlag,
			configPathFlag,
			urlFlag,
			userFlag,
			passwordFlag,
			rawFlag,
			timeoutFlag,
		},
		Action: subsAdd,
	}
	SubsRmCmd = &cli.Command{
		Name:  "rm",
		Usage: "Removes a subscription for a tracked device, remember that no MQTT messages from EcoFlow are retrieved anymore",
		Flags: []cli.Flag{
			configPathFlag,
			urlFlag,
			userFlag,
			passwordFlag,
			rawFlag,
			timeoutFlag,
		},
		ArgsUsage: "<id>",
		Action:    subsRemove,
	}
	CollectorsListCmd = &cli.Command{
		Name:  "ls",
		Usage: "Lists collector for tracked devices",
		Flags: []cli.Flag{
			configPathFlag,
			urlFlag,
			userFlag,
			passwordFlag,
			rawFlag,
			timeoutFlag,
		},
		Action: collectorsList,
	}
	CollectorsAddCmd = &cli.Command{
		Name:  "add",
		Usage: "Adds a collector for a tracked device",
		Flags: []cli.Flag{
			snFlag,
			collectorKindFlag,
			frequencyFlag,
			parametersFlag,
			configPathFlag,
			urlFlag,
			userFlag,
			passwordFlag,
			rawFlag,
			timeoutFlag,
		},
		Action: collectorsAdd,
	}
	CollectorsRmCmd = &cli.Command{
		Name:  "rm",
		Usage: "Removes a collector for a tracked device",
		Flags: []cli.Flag{
			configPathFlag,
			urlFlag,
			userFlag,
			passwordFlag,
			rawFlag,
			timeoutFlag,
		},
		ArgsUsage: "<id>",
		Action:    collectorsRemove,
	}
)

func ecoFlowDevicesList(ctx context.Context, cmd *cli.Command) error {
	if err := loadConfigFromToml(cmd); err != nil {
		return cli.Exit(err, 1)
	}
	if err := failIfFlagsNotPresent(cmd, []string{flagUrl}); err != nil {
		return cli.Exit(err, 1)
	}

	client := newClient(cmd)
	url := fmt.Sprintf("%s/devices", ecoFlowUrlPath)

	var successRes api.EcoFlowDeviceListDataResponse
	var errorRes api.ErrorResponse

	res, err := client.R().
		SetContext(ctx).
		SetResult(&successRes).
		SetError(&errorRes).
		Get(url)

	if err != nil {
		return cli.Exit(fmt.Errorf(errorCall, err), 1)
	}
	if !res.IsSuccess() {
		return cli.Exit(fmt.Sprintf("error during call: (%d) %+v", res.StatusCode(), errorRes), 1)
	}

	if cmd.Bool(flagRaw) {
		fmt.Println(string(res.Body()))
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 10, 1, 1, ' ', tabwriter.Debug)
	if _, err = fmt.Fprintf(w, "%v\t %v\n", "Serial Number", "Online"); err != nil {
		return cli.Exit(fmt.Sprintf(errorParse, err), 1)
	}

	for _, d := range successRes.Data.Content {
		if _, err = fmt.Fprintf(w, "%v\t %v\n", d.SN, d.Online); err != nil {
			return cli.Exit(fmt.Sprintf(errorFlush, err), 1)
		}
	}
	if err = w.Flush(); err != nil {
		return cli.Exit(fmt.Sprintf(errorFlush, err), 1)
	}

	return nil
}

func ecoFlowDeviceParameters(ctx context.Context, cmd *cli.Command) error {
	if err := loadConfigFromToml(cmd); err != nil {
		return cli.Exit(err, 1)
	}
	if err := failIfFlagsNotPresent(cmd, []string{flagUrl, flagSerialNumber}); err != nil {
		return cli.Exit(err, 1)
	}

	sn := cmd.String(flagSerialNumber)
	client := newClient(cmd)
	url := fmt.Sprintf("%s/devices/%s", ecoFlowUrlPath, sn)

	var successRes api.EcoFlowDeviceParametersDataResponse
	var errorRes api.ErrorResponse

	params := cmd.StringSlice(flagParameters)

	var err error
	var res *resty.Response
	if params != nil && len(params) > 0 {
		payload := api.EcoFlowDeviceParametersRequest{
			Parameters: parameters,
		}
		res, err = client.R().
			SetContext(ctx).
			SetResult(&successRes).
			SetError(&errorRes).
			SetBody(&payload).
			Post(url)
	} else {
		res, err = client.R().
			SetContext(ctx).
			SetResult(&successRes).
			SetError(&errorRes).
			Get(url)
	}

	if err != nil {
		return cli.Exit(fmt.Errorf(errorCall, err), 1)
	}
	if !res.IsSuccess() {
		return cli.Exit(fmt.Sprintf("error during call: (%d) %+v", res.StatusCode(), errorRes), 1)
	}

	if cmd.Bool(flagRaw) {
		fmt.Println(string(res.Body()))
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 10, 1, 1, ' ', tabwriter.Debug)
	if _, err = fmt.Fprintf(w, "%v\t %v\n", "Attribute", "Value"); err != nil {
		return cli.Exit(fmt.Sprintf(errorParse, err), 1)
	}

	for k, v := range successRes.Data {
		if _, err = fmt.Fprintf(w, "%v\t %+v\n", k, v); err != nil {
			return cli.Exit(fmt.Sprintf(errorFlush, err), 1)
		}
	}
	if err = w.Flush(); err != nil {
		return cli.Exit(fmt.Sprintf(errorFlush, err), 1)
	}

	return nil
}

func ecoFlowDeviceBatteries(ctx context.Context, cmd *cli.Command) error {
	if err := loadConfigFromToml(cmd); err != nil {
		return cli.Exit(err, 1)
	}
	if err := failIfFlagsNotPresent(cmd, []string{flagUrl}); err != nil {
		return cli.Exit(err, 1)
	}

	sn := cmd.String(flagSerialNumber)
	client := newClient(cmd)
	url := fmt.Sprintf("%s/devices/%s/batteries", ecoFlowUrlPath, sn)

	var successRes api.EcoFlowDeviceBatteriesDataResponse
	var errorRes api.ErrorResponse

	res, err := client.R().
		SetContext(ctx).
		SetResult(&successRes).
		SetError(&errorRes).
		Get(url)

	if err != nil {
		return cli.Exit(fmt.Errorf(errorCall, err), 1)
	}
	if !res.IsSuccess() {
		return cli.Exit(fmt.Sprintf("error during call: (%d) %+v", res.StatusCode(), errorRes), 1)
	}

	if cmd.Bool(flagRaw) {
		fmt.Println(string(res.Body()))
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 10, 1, 1, ' ', tabwriter.Debug)
	if _, err = fmt.Fprintf(w, "%v\t %v\t %v\n", "Battery SN", "Attribute", "Value"); err != nil {
		return cli.Exit(fmt.Sprintf(errorParse, err), 1)
	}

	for k, v := range successRes.Data {
		for attrKey, attrVal := range v {
			if _, err = fmt.Fprintf(w, "%v\t %v\t %v\n", k, attrKey, attrVal); err != nil {
				return cli.Exit(fmt.Sprintf(errorFlush, err), 1)
			}
		}
	}
	if err = w.Flush(); err != nil {
		return cli.Exit(fmt.Sprintf(errorFlush, err), 1)
	}

	return nil
}

func ecoFlowDeviceHistory(ctx context.Context, cmd *cli.Command) error {
	if err := loadConfigFromToml(cmd); err != nil {
		return cli.Exit(err, 1)
	}
	if err := failIfFlagsNotPresent(cmd, []string{flagUrl, flagBeginTime, flagEndTime}); err != nil {
		return cli.Exit(err, 1)
	}

	sn := cmd.String(flagSerialNumber)
	client := newClient(cmd)

	url := fmt.Sprintf("%s/devices/%s/history", ecoFlowUrlPath, sn)

	var successRes api.EcoFlowHistoryDataResponse
	var errorRes api.ErrorResponse

	argBeginTime := cmd.String(flagBeginTime)
	argEndTime := cmd.String(flagEndTime)

	var err error
	var beginTimeParsed, endTimeParsed time.Time
	if beginTimeParsed, err = time.Parse(time.DateTime, argBeginTime); err != nil {
		return cli.Exit(fmt.Sprintf("begin time is not a valid time, expecting format '%s'", time.DateTime), 1)
	}
	if endTimeParsed, err = time.Parse(time.DateTime, argEndTime); err != nil {
		return cli.Exit(fmt.Sprintf("end time is not a valid time, expecting format '%s'", time.DateTime), 1)
	}

	res, err := client.R().
		SetContext(ctx).
		SetResult(&successRes).
		SetError(&errorRes).
		SetQueryParam("beginTime", beginTimeParsed.Format(time.DateTime)).
		SetQueryParam("endTime", endTimeParsed.Format(time.DateTime)).
		Get(url)

	if err != nil {
		return cli.Exit(fmt.Errorf(errorCall, err), 1)
	}
	if !res.IsSuccess() {
		return cli.Exit(fmt.Sprintf("error during call: (%d) %+v", res.StatusCode(), errorRes), 1)
	}

	if cmd.Bool(flagRaw) {
		fmt.Println(string(res.Body()))
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 10, 1, 1, ' ', tabwriter.Debug)
	if _, err = fmt.Fprintf(w, "%v\t %v\t %v\n", "Attribute", "Value", "Unit"); err != nil {
		return cli.Exit(fmt.Sprintf(errorParse, err), 1)
	}

	for _, v := range successRes.Data {
		if _, err = fmt.Fprintf(w, "%v\t %v\t %s\n", v.IndexName, *v.IndexValue, v.Unit); err != nil {
			return cli.Exit(fmt.Sprintf(errorFlush, err), 1)
		}
	}
	if err = w.Flush(); err != nil {
		return cli.Exit(fmt.Sprintf(errorFlush, err), 1)
	}

	return nil
}

func ecoFlowBrokerStatus(ctx context.Context, cmd *cli.Command) error {
	if err := loadConfigFromToml(cmd); err != nil {
		return cli.Exit(err, 1)
	}
	if err := failIfFlagsNotPresent(cmd, []string{flagUrl}); err != nil {
		return cli.Exit(err, 1)
	}

	client := newClient(cmd)
	url := fmt.Sprintf("%s/status", ecoFlowUrlPath)

	var successRes api.EcoFlowBrokerStatusDataResponse
	var errorRes api.ErrorResponse

	res, err := client.R().
		SetContext(ctx).
		SetResult(&successRes).
		SetError(&errorRes).
		Get(url)

	if err != nil {
		return cli.Exit(fmt.Errorf(errorCall, err), 1)
	}
	if !res.IsSuccess() {
		return cli.Exit(fmt.Sprintf("error during call: (%d) %+v", res.StatusCode(), errorRes), 1)
	}

	if cmd.Bool(flagRaw) {
		fmt.Println(string(res.Body()))
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 10, 1, 1, ' ', tabwriter.Debug)
	if _, err = fmt.Fprintf(w, "%v\t %v\n", "Enabled", "Connected"); err != nil {
		return cli.Exit(fmt.Sprintf(errorParse, err), 1)
	}

	if _, err = fmt.Fprintf(w, "%v\t %v\n", successRes.Data.Enabled, successRes.Data.Connected); err != nil {
		return cli.Exit(fmt.Sprintf(errorFlush, err), 1)
	}

	if err = w.Flush(); err != nil {
		return cli.Exit(fmt.Sprintf(errorFlush, err), 1)
	}

	return nil
}

func devicesList(ctx context.Context, cmd *cli.Command) error {
	if err := loadConfigFromToml(cmd); err != nil {
		return cli.Exit(err, 1)
	}
	if err := failIfFlagsNotPresent(cmd, []string{flagUrl}); err != nil {
		return cli.Exit(err, 1)
	}

	client := newClient(cmd)

	var successRes api.DevicePageDataResponse
	var errorRes api.ErrorResponse
	url := devicesUrlPath

	res, err := client.R().
		SetContext(ctx).
		SetResult(&successRes).
		SetError(&errorRes).
		Get(url)

	if err != nil {
		return cli.Exit(fmt.Errorf(errorCall, err), 1)
	}
	if !res.IsSuccess() {
		return cli.Exit(fmt.Sprintf("error during call: (%d) %+v", res.StatusCode(), errorRes), 1)
	}

	if cmd.Bool(flagRaw) {
		fmt.Println(string(res.Body()))
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 10, 1, 1, ' ', tabwriter.Debug)
	if _, err = fmt.Fprintf(w, "%v\t %v\t %v\t %v\t %v\n", "Serial Number", "Kind", "Label", "Created", "Updated"); err != nil {
		return cli.Exit(fmt.Sprintf(errorParse, err), 1)
	}

	for _, d := range successRes.Data.Content {
		if _, err = fmt.Fprintf(w, "%v\t %v\t %v\t %v\t %v\n", d.SN, d.Kind, d.Label, d.CreatedAt, d.UpdatedAt); err != nil {
			return cli.Exit(fmt.Sprintf(errorFlush, err), 1)
		}
	}
	if err = w.Flush(); err != nil {
		return cli.Exit(fmt.Sprintf(errorFlush, err), 1)
	}

	return nil
}

func devicesAdd(ctx context.Context, cmd *cli.Command) error {
	if err := loadConfigFromToml(cmd); err != nil {
		return cli.Exit(err, 1)
	}
	if err := failIfFlagsNotPresent(cmd, []string{flagUrl, flagSerialNumber, flagLabel, flagDeviceKind}); err != nil {
		return cli.Exit(err, 1)
	}

	// validate serial number
	sn := cmd.String(flagSerialNumber)
	if sn == "" || len(sn) > 255 {
		return cli.Exit(errors.New("serial number cannot be blank or only be 255 characters long"), 1)
	}

	// validate l
	l := cmd.String(flagLabel)
	if l == "" || len(l) > 255 {
		return cli.Exit(errors.New("label cannot be blank or only be 255 characters long"), 1)
	}

	// validate kind
	kind := cmd.String(flagDeviceKind)
	if !str.FindInSlice(constant.DeviceKindNames(), kind) {
		return cli.Exit(errors.New(fmt.Sprintf("device kind must be one of %v", constant.DeviceKindNames())), 1)
	}

	// fully constructed payload
	payload := api.CreateDeviceRequest{
		SN:    sn,
		Label: l,
		Kind:  kind,
	}

	client := newClient(cmd)

	var successRes api.DeviceSingleResponse
	var errorRes api.ErrorResponse
	url := devicesUrlPath
	res, err := client.R().
		SetContext(ctx).
		SetHeader(api.HeaderContentType, api.HeaderContentTypeApplicationJson).
		SetBody(&payload).
		SetResult(&successRes).
		SetError(&errorRes).
		Post(url)

	if err != nil {
		return cli.Exit(fmt.Errorf(errorCall, err), 1)
	}
	if !res.IsSuccess() {
		return cli.Exit(fmt.Sprintf("error during call: (%d) %+v", res.StatusCode(), errorRes), 1)
	}

	if cmd.Bool(flagRaw) {
		fmt.Println(string(res.Body()))
		return nil
	}

	fmt.Printf("SN\t%v\n", successRes.Data.SN)
	fmt.Printf("Kind\t%v\n", successRes.Data.Kind)
	fmt.Printf("Label\t%v\n", successRes.Data.Label)
	fmt.Printf("Created\t%v\n", successRes.Data.CreatedAt)
	fmt.Printf("Updated\t%v\n", successRes.Data.UpdatedAt)

	return nil
}

func devicesRemove(ctx context.Context, cmd *cli.Command) error {
	if err := loadConfigFromToml(cmd); err != nil {
		return cli.Exit(err, 1)
	}
	if err := failIfFlagsNotPresent(cmd, []string{flagUrl, flagSerialNumber}); err != nil {
		return cli.Exit(err, 1)
	}

	// validate sn
	sn := cmd.String(flagSerialNumber)
	if sn == "" || len(sn) > 255 {
		return cli.Exit(errors.New("sn cannot be blank or only be 255 characters long"), 1)
	}

	client := newClient(cmd)

	var errorRes api.ErrorResponse
	url := fmt.Sprintf("%s/%s", devicesUrlPath, sn)

	res, err := client.R().
		SetContext(ctx).
		SetError(&errorRes).
		Delete(url)

	if err != nil {
		return cli.Exit(fmt.Errorf(errorCall, err), 1)
	}
	if !res.IsSuccess() {
		return cli.Exit(fmt.Sprintf("error during call: (%d) %+v", res.StatusCode(), errorRes), 1)
	}

	return nil
}

func subsList(ctx context.Context, cmd *cli.Command) error {
	if err := loadConfigFromToml(cmd); err != nil {
		return cli.Exit(err, 1)
	}
	if err := failIfFlagsNotPresent(cmd, []string{flagUrl}); err != nil {
		return cli.Exit(err, 1)
	}

	client := newClient(cmd)

	var successRes api.MqttSubscriptionPageDataResponse
	var errorRes api.ErrorResponse
	url := mqttSubscriptionsUrlPath

	res, err := client.R().
		SetContext(ctx).
		SetResult(&successRes).
		SetError(&errorRes).
		Get(url)

	if err != nil {
		return cli.Exit(fmt.Errorf(errorCall, err), 1)
	}
	if !res.IsSuccess() {
		return cli.Exit(fmt.Sprintf("error during call: (%d) %+v", res.StatusCode(), errorRes), 1)
	}

	if cmd.Bool(flagRaw) {
		fmt.Println(string(res.Body()))
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 10, 1, 1, ' ', tabwriter.Debug)
	if _, err = fmt.Fprintf(w, "%v\t %v\t %v\t %v\t %v\n", "ID", "Device SN", "Topic Kind", "Created", "Updated"); err != nil {
		return cli.Exit(fmt.Sprintf(errorParse, err), 1)
	}

	for _, d := range successRes.Data.Content {
		if _, err = fmt.Fprintf(w, "%v\t %v\t %v\t %v\t %v\n", d.ID, d.DeviceSN, d.TopicKind, d.CreatedAt, d.UpdatedAt); err != nil {
			return cli.Exit(fmt.Sprintf(errorFlush, err), 1)
		}
	}
	if err = w.Flush(); err != nil {
		return cli.Exit(fmt.Sprintf(errorFlush, err), 1)
	}

	return nil
}

func subsAdd(ctx context.Context, cmd *cli.Command) error {
	if err := loadConfigFromToml(cmd); err != nil {
		return cli.Exit(err, 1)
	}
	if err := failIfFlagsNotPresent(cmd, []string{flagUrl, flagSerialNumber, flagTopicKind}); err != nil {
		return cli.Exit(err, 1)
	}

	// validate device serial number
	deviceSN := cmd.String(flagSerialNumber)
	if deviceSN == "" || len(deviceSN) > 255 {
		return cli.Exit(errors.New("device serial number cannot be blank or only be 255 characters long"), 1)
	}

	// validate topic kind
	tk := cmd.String(flagTopicKind)
	if !str.FindInSlice(constant.TopicKindNames(), tk) {
		return cli.Exit(errors.New(fmt.Sprintf("topic kind must be one of %v", constant.TopicKindNames())), 1)
	}

	// fully constructed payload
	payload := api.CreateMqttSubscriptionRequest{
		DeviceSN:  deviceSN,
		TopicKind: tk,
	}

	client := newClient(cmd)

	var successRes api.MqttSubscriptionSingleResponse
	var errorRes api.ErrorResponse
	url := mqttSubscriptionsUrlPath

	res, err := client.R().
		SetContext(ctx).
		SetHeader(api.HeaderContentType, api.HeaderContentTypeApplicationJson).
		SetBody(&payload).
		SetResult(&successRes).
		SetError(&errorRes).
		Post(url)

	if err != nil {
		return cli.Exit(fmt.Errorf(errorCall, err), 1)
	}
	if !res.IsSuccess() {
		return cli.Exit(fmt.Sprintf("error during call: (%d) %+v", res.StatusCode(), errorRes), 1)
	}

	if cmd.Bool(flagRaw) {
		fmt.Println(string(res.Body()))
		return nil
	}

	fmt.Printf("ID\t%v\n", successRes.Data.ID)
	fmt.Printf("Device SN\t%v\n", successRes.Data.DeviceSN)
	fmt.Printf("Topic Kind\t%v\n", successRes.Data.TopicKind)
	fmt.Printf("Created\t%v\n", successRes.Data.CreatedAt)
	fmt.Printf("Updated\t%v\n", successRes.Data.UpdatedAt)

	return nil
}

func subsRemove(ctx context.Context, cmd *cli.Command) error {
	if err := loadConfigFromToml(cmd); err != nil {
		return cli.Exit(err, 1)
	}
	if err := failIfFlagsNotPresent(cmd, []string{flagUrl}); err != nil {
		return cli.Exit(err, 1)
	}
	if !cmd.Args().Present() {
		return cli.Exit(errors.New("args required - try 'subs rm help'"), 1)
	}

	// validate id
	id := cmd.Args().First()
	if id == "" || len(id) > 255 {
		return cli.Exit(errors.New("id cannot be blank or only be 255 characters long"), 1)
	}

	var errorRes api.ErrorResponse
	url := fmt.Sprintf("%s/%s", mqttSubscriptionsUrlPath, id)
	client := newClient(cmd)
	res, err := client.R().
		SetContext(ctx).
		SetError(&errorRes).
		Delete(url)

	if err != nil {
		return cli.Exit(fmt.Errorf(errorCall, err), 1)
	}
	if !res.IsSuccess() {
		return cli.Exit(fmt.Sprintf("error during call: (%d) %+v", res.StatusCode(), errorRes), 1)
	}

	return nil
}

func collectorsList(ctx context.Context, cmd *cli.Command) error {
	if err := loadConfigFromToml(cmd); err != nil {
		return cli.Exit(err, 1)
	}
	if err := failIfFlagsNotPresent(cmd, []string{flagUrl}); err != nil {
		return cli.Exit(err, 1)
	}

	client := newClient(cmd)

	var successRes api.CollectorPageDataResponse
	var errorRes api.ErrorResponse
	url := collectorsUrlPath

	res, err := client.R().
		SetContext(ctx).
		SetResult(&successRes).
		SetError(&errorRes).
		Get(url)

	if err != nil {
		return cli.Exit(fmt.Errorf(errorCall, err), 1)
	}
	if !res.IsSuccess() {
		return cli.Exit(fmt.Sprintf("error during call: (%d) %+v", res.StatusCode(), errorRes), 1)
	}

	if cmd.Bool(flagRaw) {
		fmt.Println(string(res.Body()))
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 10, 1, 1, ' ', tabwriter.Debug)
	if _, err = fmt.Fprintf(w, "%v\t %v\t %v\t %v\t %v\t %v\t %v\n", "ID", "Device SN", "Kind", "Frequency", "Payload", "Created", "Updated"); err != nil {
		return cli.Exit(fmt.Sprintf(errorParse, err), 1)
	}

	for _, d := range successRes.Data.Content {
		if _, err = fmt.Fprintf(w, "%v\t %v\t %v\t %v\t %v\t %v\t %v\n", d.ID, d.DeviceSN, d.Kind, d.Frequency, d.Payload, d.CreatedAt, d.UpdatedAt); err != nil {
			return cli.Exit(fmt.Sprintf(errorFlush, err), 1)
		}
	}
	if err = w.Flush(); err != nil {
		return cli.Exit(fmt.Sprintf(errorFlush, err), 1)
	}

	return nil
}

func collectorsAdd(ctx context.Context, cmd *cli.Command) error {
	if err := loadConfigFromToml(cmd); err != nil {
		return cli.Exit(err, 1)
	}
	if err := failIfFlagsNotPresent(cmd, []string{flagUrl, flagSerialNumber, flagCollectorKind}); err != nil {
		return cli.Exit(err, 1)
	}

	// validate device serial number
	deviceSN := cmd.String(flagSerialNumber)
	if deviceSN == "" || len(deviceSN) > 255 {
		return cli.Exit(errors.New("device serial number cannot be blank or only be 255 characters long"), 1)
	}

	// validate collector kind
	tk := cmd.String(flagCollectorKind)
	if !str.FindInSlice(constant.CollectorKindNames(), tk) {
		return cli.Exit(errors.New(fmt.Sprintf("collector kind must be one of %v", constant.CollectorKindNames())), 1)
	}

	fq := cmd.Duration(flagFrequency)

	params := cmd.StringSlice(flagParameters)

	// fully constructed payload
	payload := api.CreateCollectorRequest{
		DeviceSN:   deviceSN,
		Kind:       tk,
		Frequency:  fq.String(),
		Parameters: params,
	}

	client := newClient(cmd)

	var successRes api.CollectorSingleResponse
	var errorRes api.ErrorResponse
	url := collectorsUrlPath

	res, err := client.R().
		SetContext(ctx).
		SetHeader(api.HeaderContentType, api.HeaderContentTypeApplicationJson).
		SetBody(&payload).
		SetResult(&successRes).
		SetError(&errorRes).
		Post(url)

	if err != nil {
		return cli.Exit(fmt.Errorf(errorCall, err), 1)
	}
	if !res.IsSuccess() {
		return cli.Exit(fmt.Sprintf("error during call: (%d) %+v", res.StatusCode(), errorRes), 1)
	}

	if cmd.Bool(flagRaw) {
		fmt.Println(string(res.Body()))
		return nil
	}

	fmt.Printf("ID\t%v\n", successRes.Data.ID)
	fmt.Printf("Device SN\t%v\n", successRes.Data.DeviceSN)
	fmt.Printf("Kind\t%v\n", successRes.Data.Kind)
	fmt.Printf("Frequency\t%v\n", successRes.Data.Frequency)
	fmt.Printf("Payload\t%v\n", successRes.Data.Payload)
	fmt.Printf("Created\t%v\n", successRes.Data.CreatedAt)
	fmt.Printf("Updated\t%v\n", successRes.Data.UpdatedAt)

	return nil
}

func collectorsRemove(ctx context.Context, cmd *cli.Command) error {
	if err := loadConfigFromToml(cmd); err != nil {
		return cli.Exit(err, 1)
	}
	if err := failIfFlagsNotPresent(cmd, []string{flagUrl}); err != nil {
		return cli.Exit(err, 1)
	}
	if !cmd.Args().Present() {
		return cli.Exit(errors.New("args required - try 'subs rm help'"), 1)
	}

	// validate id
	id := cmd.Args().First()
	if id == "" || len(id) > 255 {
		return cli.Exit(errors.New("id cannot be blank or only be 255 characters long"), 1)
	}

	var errorRes api.ErrorResponse
	url := fmt.Sprintf("%s/%s", collectorsUrlPath, id)
	client := newClient(cmd)
	res, err := client.R().
		SetContext(ctx).
		SetError(&errorRes).
		Delete(url)

	if err != nil {
		return cli.Exit(fmt.Errorf(errorCall, err), 1)
	}
	if !res.IsSuccess() {
		return cli.Exit(fmt.Sprintf("error during call: (%d) %+v", res.StatusCode(), errorRes), 1)
	}

	return nil
}

type configToml struct {
	Server struct {
		URL string
	}
	Auth struct {
		User         string
		Password     string
		PasswordFile string
	}
	Parsing struct {
		Raw bool
	}
}

// loadConfigFromToml loads configuration from a TOML file
// looks up default XDG_CONFIG_HOME/ecolinker.toml first
func loadConfigFromToml(cmd *cli.Command) error {
	if configPathFlag == nil {
		return nil
	}

	path := cmd.String(flagConfig)

	if path == "" {
		if foundPath, err := xdg.SearchConfigFile(fmt.Sprintf("%s.toml", app.Name)); err == nil {
			path = foundPath
		} else {
			return nil
		}
	}

	if _, err := os.Stat(path); err != nil {
		return nil
	}

	var err error
	var config configToml
	if _, err = toml.DecodeFile(path, &config); err != nil {
		return fmt.Errorf("cannot read config file '%s': %w", path, err)
	}

	// for each configuration, prioritize externally provided settings
	if cmd.String(flagUrl) == "" {
		if err = cmd.Set(flagUrl, config.Server.URL); err != nil {
			return fmt.Errorf("cannot set config value '%s' from config file '%s': %w", flagUrl, path, err)
		}
	}

	if !cmd.Bool(flagRaw) {
		if err = cmd.Set(flagRaw, strconv.FormatBool(config.Parsing.Raw)); err != nil {
			return fmt.Errorf("cannot set config value '%s' from config file '%s': %w", flagRaw, path, err)
		}
	}

	if cmd.String(flagUser) == "" {
		if err = cmd.Set(flagUser, config.Auth.User); err != nil {
			return fmt.Errorf("cannot set config value '%s' from config file '%s': %w", flagUser, path, err)
		}
	}

	// prioritize passwordFile contents if not externally provided
	if cmd.String(flagPass) != "" {
		return nil
	}

	if config.Auth.PasswordFile != "" {
		var b []byte
		if b, err = os.ReadFile(config.Auth.PasswordFile); err != nil {
			return fmt.Errorf("cannot read password file '%s': %w", config.Auth.PasswordFile, err)
		}

		passwordFromFile := strings.TrimSpace(string(b))
		if err = cmd.Set(flagPass, passwordFromFile); err != nil {
			return fmt.Errorf("cannot set config value '%s' from config file '%s': %w", flagPass, path, err)
		}

		return nil
	}

	if err = cmd.Set(flagPass, config.Auth.Password); err != nil {
		return fmt.Errorf("cannot set config value '%s' from config file '%s': %w", flagPass, path, err)
	}

	return nil
}

// failIfFlagsNotPresent fails if any string flag is required, but not provided
func failIfFlagsNotPresent(cmd *cli.Command, flagKeys []string) error {
	if flagKeys == nil {
		return errors.New("flagKeys cannot be null")
	}

	for _, key := range flagKeys {
		if cmd.String(key) == "" {
			return errors.New(fmt.Sprintf("'%v' is required but blank", key))
		}
	}

	return nil
}

func newClient(cmd *cli.Command) *resty.Client {
	client := resty.New()
	client.SetHeader("User-Agent", fmt.Sprintf("%s/%s", app.Name, app.Version))
	client.SetDisableWarn(true)
	client.SetTimeout(cmd.Duration(flagTimeout))
	client.SetBaseURL(cmd.String(flagUrl))

	username := cmd.String(flagUser)
	pass := cmd.String(flagPass)

	if username != "" && pass != "" {
		client.SetBasicAuth(cmd.String(flagUser), cmd.String(flagPass))
	}

	return client
}
