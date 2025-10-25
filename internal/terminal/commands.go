package terminal

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"git.myservermanager.com/varakh/ecolinker/api"
	"git.myservermanager.com/varakh/ecolinker/internal/meta"
	"git.myservermanager.com/varakh/ecolinker/internal/server/constant"
	"git.myservermanager.com/varakh/ecolinker/internal/str"
	"git.myservermanager.com/varakh/ecolinker/internal/tm"
	"github.com/BurntSushi/toml"
	"github.com/adrg/xdg"
	"github.com/go-resty/resty/v2"
	"github.com/urfave/cli/v3"
	"os"
	"slices"
	"sort"
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

	flagConfig       = "config"
	flagUrl          = "url"
	flagUser         = "user"
	flagPass         = "pass"
	flagPrintRaw     = "raw"
	flagTimeout      = "timeout"
	flagPrintCsv     = "csv"
	flagPrintRowWise = "row-wise"

	flagSerialNumber = "serial-number"
	flagLabel        = "label"
	flagBeginTime    = "begin-time"
	flagEndTime      = "end-time"
	flagDeviceKind   = "device-kind"
	flagTopicKind    = "topic-kind"
	flagFrequency    = "frequency"
	flagParameters   = "parameter"
	flagStep         = "step"
	flagInterval     = "interval"

	ecoFlowUrlPath           = "/api/v1/ecoflow"
	devicesUrlPath           = "/api/v1/devices"
	mqttSubscriptionsUrlPath = "/api/v1/mqtt-subscriptions"
	collectorsUrlPath        = "/api/v1/collectors"

	errorParse = "error while parsing response: %v"
	errorFlush = "error during while flushing response: %v"
	errorCall  = "error during call: %w"

	collectorAddRequestPayloadParameters = "parameters"
	collectorAddRequestPayloadStep       = "step"
)

var (
	configPath                 string
	instanceUrl                string
	user                       string
	password                   string
	timeout                    time.Duration
	printRaw                   bool
	printCsv                   bool
	printRowwise               bool
	serialNumber               string
	label                      string
	deviceKind                 string
	topicKind                  string
	frequency                  time.Duration
	beginTime                  string
	endTime                    string
	interval                   string
	collectorPayloadParameters []string
	collectorPayloadStep       string

	configPathFlag = &cli.StringFlag{
		Name:        flagConfig,
		Usage:       "EcoLinker's TOML configuration file path",
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
	printRawFlag = &cli.BoolFlag{
		Name:        flagPrintRaw,
		Usage:       "Prints JSON",
		Aliases:     []string{"r"},
		Value:       false,
		Destination: &printRaw,
	}
	printCsvFlag = &cli.BoolFlag{
		Name:        flagPrintCsv,
		Usage:       "Prints CSV",
		Value:       false,
		Destination: &printCsv,
	}
	printRowFlag = &cli.BoolFlag{
		Name:        flagPrintRowWise,
		Usage:       "Prints data row-wise",
		Value:       false,
		Destination: &printRowwise,
	}
	timeoutFlag = &cli.DurationFlag{
		Name:        flagTimeout,
		Usage:       "Maximum timeout to query EcoLinker",
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
	frequencyFlag = &cli.DurationFlag{
		Name:        flagFrequency,
		Usage:       "Frequency for collector invocations (if collector queries EcoFlow's HTTP API, ensure to pick a reasonable duration to not spam EcoFlow's services). Pick the frequency according to the task being executed to deliver data.",
		Aliases:     []string{"fq"},
		Required:    false,
		Value:       45 * time.Second,
		Destination: &frequency,
	}
	beginTimeFlag = &cli.StringFlag{
		Name:        flagBeginTime,
		Usage:       fmt.Sprintf("Begin time with layout '%s', total time span cannot exceed one week if interval is not provided", time.DateTime),
		Aliases:     []string{"bt"},
		Destination: &beginTime,
	}
	endTimeFlag = &cli.StringFlag{
		Name:        flagEndTime,
		Usage:       fmt.Sprintf("End time with layout '%s', total time span cannot exceed one week if interval is not provided", time.DateTime),
		Aliases:     []string{"et"},
		Destination: &endTime,
	}
	parametersFlag = &cli.StringSliceFlag{
		Name:        flagParameters,
		Usage:       "Parameters to query when collector runs, if none given, all will be queried, you can provide multiple --parameter or using its aliases",
		Aliases:     []string{"pn", "par", "pp"},
		Destination: &collectorPayloadParameters,
	}
	stepFlag = &cli.StringFlag{
		Name:  flagStep,
		Usage: fmt.Sprintf("Time range to query when collector runs which uses full PAST period (last week starting from Monday to Sunday or yesterday), one of: %s", strings.Join(constant.HistoricalDataStepNames(), " ")),
		Validator: func(v string) error {
			_, err := constant.ParseHistoricalDataStep(v)
			return err
		},
		Destination: &collectorPayloadStep,
	}
	intervalFlag = &cli.StringFlag{
		Name:     flagInterval,
		Required: false,
		Usage:    fmt.Sprintf("Splits begin to end time range into individual distinct sub time ranges with a given interval step size ('%s') between them (inclusive) which allows to collect daily or weekly aggregated data", strings.Join(constant.HistoricalDataStepNames(), " ")),
		Validator: func(v string) error {
			_, err := constant.ParseHistoricalDataStep(v)
			return err
		},
		Destination: &interval,
	}
	EcoFlowDevicesListCmd = &cli.Command{
		Name:  "ls",
		Usage: "Lists devices on your EcoFlow account",
		Flags: []cli.Flag{
			configPathFlag,
			urlFlag,
			userFlag,
			passwordFlag,
			timeoutFlag,
			printRawFlag,
		},
		Action: ecoFlowDevicesList,
	}
	EcoFlowDeviceParametersCmd = &cli.Command{
		Name:  "ps",
		Usage: "Device's parameters queried from EcoFlow",
		Flags: []cli.Flag{
			configPathFlag,
			urlFlag,
			userFlag,
			passwordFlag,
			timeoutFlag,
			snFlag,
			parametersFlag,
			printRawFlag,
		},
		Action: ecoFlowDeviceParameters,
	}
	EcoFlowDeviceBatteriesCmd = &cli.Command{
		Name:        "bs",
		Usage:       "Device's batteries queried from EcoFlow",
		Description: "Filters specific keys of all parameters for a device",
		Flags: []cli.Flag{
			configPathFlag,
			urlFlag,
			userFlag,
			passwordFlag,
			timeoutFlag,
			snFlag,
			printRawFlag,
		},
		Action: ecoFlowDeviceBatteries,
	}
	EcoFlowDeviceHistoryCmd = &cli.Command{
		Name:  "hs",
		Usage: "Device's historical data queried from EcoFlow (PowerOcean only)",
		Flags: []cli.Flag{
			configPathFlag,
			urlFlag,
			userFlag,
			passwordFlag,
			timeoutFlag,
			snFlag,
			beginTimeFlag,
			endTimeFlag,
			intervalFlag,
			printCsvFlag,
			printRowFlag,
		},
		Action: ecoFlowDeviceHistory,
	}
	EcoFlowStatusCmd = &cli.Command{
		Name:  "status",
		Usage: "EcoLinker's status regarding its connection to EcoFlow's MQTT broker",
		Flags: []cli.Flag{
			configPathFlag,
			urlFlag,
			userFlag,
			passwordFlag,
			timeoutFlag,
			snFlag,
			printRawFlag,
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
			timeoutFlag,
			printRawFlag,
		},
		Action: devicesList,
	}
	DevicesAddCmd = &cli.Command{
		Name:  "add",
		Usage: "Adds a device which enables you to listen for MQTT messages from EcoFlow",
		Flags: []cli.Flag{
			configPathFlag,
			urlFlag,
			userFlag,
			passwordFlag,
			timeoutFlag,
			snFlag,
			labelFlag,
			deviceKindFlag,
			printRawFlag,
		},
		Action: devicesAdd,
	}
	DevicesRmCmd = &cli.Command{
		Name:  "rm",
		Usage: "Removes a device, remember that all associated MQTT subscriptions are deleted with it",
		Flags: []cli.Flag{
			configPathFlag,
			urlFlag,
			userFlag,
			passwordFlag,
			timeoutFlag,
			snFlag,
			printRawFlag,
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
			timeoutFlag,
			printRawFlag,
		},
		Action: subsList,
	}
	SubsAddCmd = &cli.Command{
		Name:  "add",
		Usage: "Adds a subscription for a tracked device",
		Flags: []cli.Flag{
			configPathFlag,
			urlFlag,
			userFlag,
			passwordFlag,
			timeoutFlag,
			snFlag,
			topicKindFlag,
			printRawFlag,
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
			timeoutFlag,
			printRawFlag,
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
			timeoutFlag,
			printRawFlag,
		},
		Action: collectorsList,
	}
	CollectorsAddCmd = &cli.Command{
		Name:  "add",
		Usage: "Adds a collector for a tracked device",
		Commands: []*cli.Command{
			{
				Name:  "device-parameters",
				Usage: "Adds a collector for a tracked device to query parameters",
				Flags: []cli.Flag{
					configPathFlag,
					urlFlag,
					userFlag,
					passwordFlag,
					timeoutFlag,
					snFlag,
					frequencyFlag,
					parametersFlag,
					printRawFlag,
				},
				Action: collectorsAddDeviceParameters,
			},
			{
				Name:  "device-historical-data",
				Usage: "Adds a collector for a tracked device to query historical data (PowerOcean only)",
				Flags: []cli.Flag{
					configPathFlag,
					urlFlag,
					userFlag,
					passwordFlag,
					timeoutFlag,
					snFlag,
					frequencyFlag,
					stepFlag,
					printRawFlag,
				},
				Action: collectorsAddDeviceHistoricalData,
			},
		},
	}
	CollectorsRmCmd = &cli.Command{
		Name:  "rm",
		Usage: "Removes a collector for a tracked device",
		Flags: []cli.Flag{
			configPathFlag,
			urlFlag,
			userFlag,
			passwordFlag,
			timeoutFlag,
			printRawFlag,
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

	if cmd.Bool(flagPrintRaw) {
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
			Parameters: collectorPayloadParameters,
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

	if cmd.Bool(flagPrintRaw) {
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

	if cmd.Bool(flagPrintRaw) {
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

	var timeRanges []tm.TimeRange
	intervalStep := cmd.String(flagInterval)
	if intervalStep != "" {
		stepSize := constant.MustParseHistoricalDataStep(intervalStep)
		skipInterval := time.Hour * 24
		if constant.HistoricalDataStepWeekly == stepSize {
			skipInterval = time.Hour * 24 * 7
		}

		timeRanges, _ = tm.TimeRanges(beginTimeParsed, endTimeParsed, skipInterval, -1*time.Second, false)
	} else {
		timeRanges = append(timeRanges, tm.TimeRange{
			Start: beginTimeParsed,
			End:   endTimeParsed,
		})
	}

	rowWise := cmd.Bool(flagPrintRowWise)
	var header []string
	var rows [][]string

	for _, t := range timeRanges {
		var successRes api.EcoFlowHistoryDataResponse
		var errorRes api.ErrorResponse

		res, reqErr := client.R().
			SetContext(ctx).
			SetResult(&successRes).
			SetError(&errorRes).
			SetQueryParam("beginTime", t.Start.Format(time.DateTime)).
			SetQueryParam("endTime", t.End.Format(time.DateTime)).
			Get(url)

		if reqErr != nil {
			return cli.Exit(fmt.Errorf(errorCall, err), 1)
		}
		if !res.IsSuccess() {
			return cli.Exit(fmt.Sprintf("error during call: (%d) %+v", res.StatusCode(), errorRes), 1)
		}

		header = getDeviceHistoryHeader(rowWise, successRes.Data)
		for _, r := range getDeviceHistoryRows(rowWise, successRes.Data, t.Start, t.End) {
			rows = append(rows, r)
		}
	}

	if cmd.Bool(flagPrintCsv) {
		return printDeviceHistoryCsv(header, rows)
	}

	return printDeviceHistoryTabular(header, rows)
}

func getDeviceHistoryHeader(rowWise bool, data []*api.EcoFlowHistoryItemResponse) []string {
	if !rowWise {
		return []string{"Start", "End", "Attribute", "Value", "Unit"}
	}

	header := make([]string, 0)
	header = append(header, "Start")
	header = append(header, "End")

	sort.SliceStable(data, func(i, j int) bool {
		return data[i].IndexName < data[j].IndexName
	})

	for _, v := range data {
		header = append(header, fmt.Sprintf("%s Value", v.IndexName))
		header = append(header, fmt.Sprintf("%s Unit", v.IndexName))
	}

	return header
}

func getDeviceHistoryRows(rowWise bool, data []*api.EcoFlowHistoryItemResponse, start time.Time, end time.Time) [][]string {
	rows := make([][]string, 0)

	if !rowWise {
		for _, v := range data {
			rows = append(rows, []string{start.Format(time.DateTime), end.Format(time.DateTime), v.IndexName, fmt.Sprintf("%v", *v.IndexValue), v.Unit})
		}

		return rows
	}

	row := make([]string, 0)
	row = append(row, start.Format(time.DateTime))
	row = append(row, end.Format(time.DateTime))

	sort.SliceStable(data, func(i, j int) bool {
		return data[i].IndexName < data[j].IndexName
	})

	for _, v := range data {
		row = append(row, fmt.Sprintf("%v", *v.IndexValue))
		row = append(row, v.Unit)
	}

	rows = append(rows, row)

	return rows
}

func printDeviceHistoryCsv(header []string, rows [][]string) cli.ExitCoder {
	var err error
	w := csv.NewWriter(os.Stdout)

	if err = w.Write(header); err != nil {
		return cli.Exit(fmt.Sprintf(errorParse, err), 1)
	}
	for _, v := range rows {
		if err = w.Write(v); err != nil {
			return cli.Exit(fmt.Sprintf(errorParse, err), 1)
		}
	}

	w.Flush()
	if err = w.Error(); err != nil {
		return cli.Exit(fmt.Sprintf(errorFlush, err), 1)
	}

	return nil
}

func printDeviceHistoryTabular(header []string, rows [][]string) cli.ExitCoder {
	var err error
	w := tabwriter.NewWriter(os.Stdout, 10, 1, 1, ' ', tabwriter.Debug)

	headerFormat, headerArgs := getTabularFormatAndArgs(header)
	if _, err = fmt.Fprintf(w, headerFormat, headerArgs...); err != nil {
		return cli.Exit(fmt.Sprintf(errorParse, err), 1)
	}

	for _, row := range rows {
		rowFormat, rowArgs := getTabularFormatAndArgs(row)
		if _, err = fmt.Fprintf(w, rowFormat, rowArgs...); err != nil {
			return cli.Exit(fmt.Sprintf(errorFlush, err), 1)
		}
	}

	if err = w.Flush(); err != nil {
		return cli.Exit(fmt.Sprintf(errorFlush, err), 1)
	}

	return nil
}

func getTabularFormatAndArgs(data []string) (string, []interface{}) {
	format := ""
	for i := range data {
		if len(data)-1 == i {
			format += "%v"
		} else {
			format += "%v\t"
		}
	}
	format += "\n"

	args := make([]interface{}, len(data))
	for i, v := range data {
		args[i] = v
	}

	return format, args
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

	if cmd.Bool(flagPrintRaw) {
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

	if cmd.Bool(flagPrintRaw) {
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

	if cmd.Bool(flagPrintRaw) {
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

	if cmd.Bool(flagPrintRaw) {
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

	if cmd.Bool(flagPrintRaw) {
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

	if cmd.Bool(flagPrintRaw) {
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

func collectorsAddDeviceParameters(ctx context.Context, cmd *cli.Command) error {
	if err := loadConfigFromToml(cmd); err != nil {
		return cli.Exit(err, 1)
	}
	if err := failIfFlagsNotPresent(cmd, []string{flagUrl, flagSerialNumber}); err != nil {
		return cli.Exit(err, 1)
	}

	// validate device serial number
	deviceSN := cmd.String(flagSerialNumber)
	if deviceSN == "" || len(deviceSN) > 255 {
		return cli.Exit(errors.New("device serial number cannot be blank or only be 255 characters long"), 1)
	}

	fq := cmd.Duration(flagFrequency)
	if fq == 0 {
		return cli.Exit(errors.New("frequency must be set to a valid duration"), 1)
	}

	params := cmd.StringSlice(flagParameters)

	// fully constructed payload
	collectorPayload := make(map[string]interface{})
	collectorPayload[collectorAddRequestPayloadParameters] = params

	payload := api.CreateCollectorRequest{
		DeviceSN:  deviceSN,
		Kind:      constant.CollectorKindDeviceParameters.String(),
		Frequency: fq.String(),
		Payload:   collectorPayload,
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

	if cmd.Bool(flagPrintRaw) {
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

func collectorsAddDeviceHistoricalData(ctx context.Context, cmd *cli.Command) error {
	if err := loadConfigFromToml(cmd); err != nil {
		return cli.Exit(err, 1)
	}
	if err := failIfFlagsNotPresent(cmd, []string{flagUrl, flagSerialNumber}); err != nil {
		return cli.Exit(err, 1)
	}

	// validate device serial number
	deviceSN := cmd.String(flagSerialNumber)
	if deviceSN == "" || len(deviceSN) > 255 {
		return cli.Exit(errors.New("device serial number cannot be blank or only be 255 characters long"), 1)
	}

	fq := cmd.Duration(flagFrequency)
	if fq == 0 {
		return cli.Exit(errors.New("frequency must be set to a valid duration"), 1)
	}

	step := cmd.String(flagStep)

	// fully constructed payload
	collectorPayload := make(map[string]interface{})
	collectorPayload[collectorAddRequestPayloadStep] = step

	payload := api.CreateCollectorRequest{
		DeviceSN:  deviceSN,
		Kind:      constant.CollectorKindDeviceHistoricalData.String(),
		Frequency: fq.String(),
		Payload:   collectorPayload,
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

	if cmd.Bool(flagPrintRaw) {
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
	Device struct {
		SerialNumber string
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
		if foundPath, err := xdg.SearchConfigFile(fmt.Sprintf("%s.toml", meta.Name)); err == nil {
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

	flagNames := getAllFlagNames(cmd)

	// for each configuration, prioritize externally provided settings
	if slices.Contains(flagNames, flagUrl) && cmd.String(flagUrl) == "" {
		if err = cmd.Set(flagUrl, config.Server.URL); err != nil {
			return fmt.Errorf("cannot set config value '%s' from config file '%s': %w", flagUrl, path, err)
		}
	}

	if slices.Contains(flagNames, flagPrintRaw) && !cmd.Bool(flagPrintRaw) {
		if err = cmd.Set(flagPrintRaw, strconv.FormatBool(config.Parsing.Raw)); err != nil {
			return fmt.Errorf("cannot set config value '%s' from config file '%s': %w", flagPrintRaw, path, err)
		}
	}

	if slices.Contains(flagNames, flagUser) && cmd.String(flagUser) == "" {
		if err = cmd.Set(flagUser, config.Auth.User); err != nil {
			return fmt.Errorf("cannot set config value '%s' from config file '%s': %w", flagUser, path, err)
		}
	}

	if slices.Contains(flagNames, flagSerialNumber) && cmd.String(flagSerialNumber) == "" {
		if err = cmd.Set(flagSerialNumber, config.Device.SerialNumber); err != nil {
			return fmt.Errorf("cannot set config value '%s' from config file '%s': %w", flagSerialNumber, path, err)
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

// getAllFlagNames gets all active flag names for a command
func getAllFlagNames(cmd *cli.Command) []string {
	flagNames := make([]string, len(cmd.Flags))
	for _, f := range cmd.Flags {
		flagNames = append(flagNames, f.Names()...)
	}
	return flagNames
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
	client.SetHeader("User-Agent", fmt.Sprintf("%s/%s", meta.Name, meta.Version))
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
