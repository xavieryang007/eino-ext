package metrics

import (
	"code.byted.org/gopkg/env"
	"code.byted.org/gopkg/metrics/v4"
)

const metricsPrefix = "eino"

const (
	metricsNameGraph             = "graph"
	metricsNameCommon            = "common"
	metricsNameOutputStreamStart = "output_stream_start"
)

const (
	tagNameSDKVersion = "_sdk_version"
	tagNameIsError    = "_is_error"

	tagNameRunInfoType      = "_run_info_type"
	tagNameRunInfoComponent = "_run_info_component"
	tagNameRunInfoName      = "_run_info_name"
)

var globalCli metrics.Client
var cli metrics.Client
var graphMetric metrics.Metric
var commonMetric metrics.Metric
var outputStreamStartMetric metrics.Metric

func initMetrics() error {
	var err error

	psm := env.PSM()
	if psm == env.PSMUnknown {
		psm = "unknown"
	}

	cli, err = metrics.NewClient(metricsPrefix+"."+psm, metrics.SetDiscardInvalidTag(), metrics.SetReportInitialCounter())
	if err != nil {
		return err
	}

	globalCli, err = metrics.NewClient(metricsPrefix, metrics.SetDiscardInvalidTag(), metrics.SetReportInitialCounter())
	if err != nil {
		return err
	}

	graphMetric, err = globalCli.NewMetricWithOps(
		metricsNameGraph,
		[]string{tagNameSDKVersion},
		metrics.SetMultiFieldTimer())
	if err != nil {
		return err
	}
	commonMetric, err = cli.NewMetricWithOps(
		metricsNameCommon,
		[]string{tagNameRunInfoComponent,
			tagNameRunInfoType,
			tagNameRunInfoName,
			tagNameIsError},
		metrics.SetMultiFieldTimer())
	if err != nil {
		return err
	}
	outputStreamStartMetric, err = cli.NewMetricWithOps(
		metricsNameOutputStreamStart,
		[]string{tagNameRunInfoComponent,
			tagNameRunInfoType,
			tagNameRunInfoName},
		metrics.SetMultiFieldTimer())
	if err != nil {
		return err
	}
	return nil
}
