package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	monitoring "cloud.google.com/go/monitoring/apiv3"
	"google.golang.org/api/iterator"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

type RequestConfig struct {
	Alignment   int32  `json:"alignment"`
	Reducer     int32  `json:"reducer"`
	Measurement string `json:"measurement"`
}

type Label struct {
	Key         string
	Description string
}

type MeasurementJSON struct {
	PeriodMinutes int                 `json:"period-minutes,omitempty"`
	Filters       []string            `json:"filters,omitempty"`
	Measurements  []MeasurementConfig `json:"measurements,omitempty"`
	ProjectID     string              `json:"project-id"`
}

type MeasurementConfig struct {
	MetricName string          `json:"metric"`
	Kind       string          `json:"kind"`
	Filters    []string        `json:"filters,omitempty"`
	Config     []RequestConfig `json:"config"`
	Labels     []Label         `json:"label"`
}

func main() {
	// import monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
	// import "google.golang.org/api/iterator"
	project := flag.String("p", "", "project id")
	filter := flag.String("f", "", "filter")
	flag.Parse()
	ctx := context.Background()
	c, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		panic(err)
	}

	req := &monitoringpb.ListMetricDescriptorsRequest{
		Name:   "projects/" + *project,
		Filter: `metric.type = starts_with("` + *filter + `")`,
		// TODO: Fill request struct fields.
	}
	it := c.ListMetricDescriptors(ctx, req)
	measurementJSON := MeasurementJSON{}
	for {
		metricDescriptor, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			panic(err)
			// TODO: Handle error.
		}
		measurementConfig := MeasurementConfig{}
		measurementConfig.MetricName = metricDescriptor.Type
		measurementConfig.Kind = metricDescriptor.MetricKind.String()
		for _, l := range metricDescriptor.Labels {
			measurementConfig.Labels = append(
				measurementConfig.Labels,
				Label{
					Key:         l.Key,
					Description: l.Description,
				},
			)
		}
		measurement := strings.TrimPrefix(metricDescriptor.Type, *filter)
		measurement = strings.Replace(measurement, "/", "_", -1)
		measurement = strings.Replace(measurement, ".", "_", -1)
		// TODO: Use resp.
		var config RequestConfig

		switch metricDescriptor.MetricKind.String() {
		case "GAUGE", "DELTA":
			switch metricDescriptor.ValueType.String() {
			case "INT64", "DOUBLE":
				// Align the time series by returning the minimum value in each alignment
				// period. This aligner is valid for `GAUGE` and `DELTA` metrics with
				// numeric values. The `value_type` of the aligned result is the same as
				// the `value_type` of the input.
				// Aggregation_ALIGN_MIN Aggregation_Aligner = 10
				config = RequestConfig{
					Alignment:   10,
					Measurement: measurement + "_min",
				}
				measurementConfig.Config = append(measurementConfig.Config, config)
				// Align the time series by returning the maximum value in each alignment
				// period. This aligner is valid for `GAUGE` and `DELTA` metrics with
				// numeric values. The `value_type` of the aligned result is the same as
				// the `value_type` of the input.
				// Aggregation_ALIGN_MAX Aggregation_Aligner = 11
				config = RequestConfig{
					Alignment:   11,
					Measurement: measurement + "_max",
				}
				measurementConfig.Config = append(measurementConfig.Config, config)
				// Align the time series by returning the mean value in each alignment
				// period. This aligner is valid for `GAUGE` and `DELTA` metrics with
				// numeric values. The `value_type` of the aligned result is `DOUBLE`.
				// Aggregation_ALIGN_MEAN Aggregation_Aligner = 12
				config = RequestConfig{
					Alignment:   12,
					Measurement: measurement + "_mean",
				}
				measurementConfig.Config = append(measurementConfig.Config, config)
				// Align the time series by returning the number of values in each alignment
				// period. This aligner is valid for `GAUGE` and `DELTA` metrics with
				// numeric or Boolean values. The `value_type` of the aligned result is
				// `INT64`.
				//Aggregation_ALIGN_COUNT Aggregation_Aligner = 13
				config = RequestConfig{
					Alignment:   13,
					Measurement: measurement + "_count",
				}
				measurementConfig.Config = append(measurementConfig.Config, config)

				// Align the time series by returning the sum of the values in each
				// alignment period. This aligner is valid for `GAUGE` and `DELTA`
				// metrics with numeric and distribution values. The `value_type` of the
				// aligned result is the same as the `value_type` of the input.
				//Aggregation_ALIGN_SUM Aggregation_Aligner = 14
				config = RequestConfig{
					Alignment:   14,
					Measurement: measurement + "_sum",
				}
				measurementConfig.Config = append(measurementConfig.Config, config)

				// Align the time series by returning the standard deviation of the values
				// in each alignment period. This aligner is valid for `GAUGE` and
				// `DELTA` metrics with numeric values. The `value_type` of the output is
				// `DOUBLE`.
				//Aggregation_ALIGN_STDDEV Aggregation_Aligner = 15
				config = RequestConfig{
					Alignment:   15,
					Measurement: measurement + "_stddev",
				}
				measurementConfig.Config = append(measurementConfig.Config, config)

				// Align the time series by returning the number of `True` values in
				// Align and convert to a percentage change. This aligner is valid for
				// `GAUGE` and `DELTA` metrics with numeric values. This alignment returns
				// `((current - previous)/previous) * 100`, where the value of `previous` is
				// determined based on the `alignment_period`.
				//
				// If the values of `current` and `previous` are both 0, then the returned
				// value is 0. If only `previous` is 0, the returned value is infinity.
				//
				// A 10-minute moving mean is computed at each point of the alignment period
				// prior to the above calculation to smooth the metric and prevent false
				// positives from very short-lived spikes. The moving mean is only
				// applicable for data whose values are `>= 0`. Any values `< 0` are
				// treated as a missing datapoint, and are ignored. While `DELTA`
				// metrics are accepted by this alignment, special care should be taken that
				// the values for the metric will always be positive. The output is a
				// `GAUGE` metric with `value_type` `DOUBLE`.
				//Aggregation_ALIGN_PERCENT_CHANGE Aggregation_Aligner = 23
				/*
					config = RequestConfig{
						Alignment:   23,
						Measurement: measurement + "_percent_change",
					}
					measurementConfig.Config = append(measurementConfig.Config, config)
				*/
			}
			switch metricDescriptor.ValueType.String() {
			case "DISTRIBUTION":
				// Align the time series by using [percentile
				// aggregation](https://en.wikipedia.org/wiki/Percentile). The resulting
				// data point in each alignment period is the 99th percentile of all data
				// points in the period. This aligner is valid for `GAUGE` and `DELTA`
				// metrics with distribution values. The output is a `GAUGE` metric with
				// `value_type` `DOUBLE`.
				//Aggregation_ALIGN_PERCENTILE_99 Aggregation_Aligner = 18
				config = RequestConfig{
					Alignment:   18,
					Measurement: measurement + "_p99",
				}
				measurementConfig.Config = append(measurementConfig.Config, config)

				// Align the time series by using [percentile
				// aggregation](https://en.wikipedia.org/wiki/Percentile). The resulting
				// data point in each alignment period is the 95th percentile of all data
				// points in the period. This aligner is valid for `GAUGE` and `DELTA`
				// metrics with distribution values. The output is a `GAUGE` metric with
				// `value_type` `DOUBLE`.
				//Aggregation_ALIGN_PERCENTILE_95 Aggregation_Aligner = 19
				config = RequestConfig{
					Alignment:   19,
					Measurement: measurement + "_p95",
				}
				measurementConfig.Config = append(measurementConfig.Config, config)

				// Align the time series by using [percentile
				// aggregation](https://en.wikipedia.org/wiki/Percentile). The resulting
				// data point in each alignment period is the 50th percentile of all data
				// points in the period. This aligner is valid for `GAUGE` and `DELTA`
				// metrics with distribution values. The output is a `GAUGE` metric with
				// `value_type` `DOUBLE`.
				//Aggregation_ALIGN_PERCENTILE_50 Aggregation_Aligner = 20
				config = RequestConfig{
					Alignment:   20,
					Measurement: measurement + "_p50",
				}
				measurementConfig.Config = append(measurementConfig.Config, config)

				// Align the time series by using [percentile
				// aggregation](https://en.wikipedia.org/wiki/Percentile). The resulting
				// data point in each alignment period is the 5th percentile of all data
				// points in the period. This aligner is valid for `GAUGE` and `DELTA`
				// metrics with distribution values. The output is a `GAUGE` metric with
				// `value_type` `DOUBLE`.
				//Aggregation_ALIGN_PERCENTILE_05 Aggregation_Aligner = 21
				config = RequestConfig{
					Alignment:   21,
					Measurement: measurement + "_p05",
				}
				measurementConfig.Config = append(measurementConfig.Config, config)

			}
		}
		switch metricDescriptor.MetricKind.String() {
		case "GAUGE":
			switch metricDescriptor.ValueType.String() {
			case "INT64", "DOUBLE":
				// Align by interpolating between adjacent points around the alignment
				// period boundary. This aligner is valid for `GAUGE` metrics with
				// numeric values. The `value_type` of the aligned result is the same as the
				// `value_type` of the input.
				//  Aggregation_ALIGN_INTERPOLATE Aggregation_Aligner = 3
				config = RequestConfig{
					Alignment:   3,
					Measurement: measurement,
				}
				measurementConfig.Config = append(measurementConfig.Config, config)
			case "BOOL":
				// Align the time series by returning the number of `True` values in
				// each alignment period. This aligner is valid for `GAUGE` metrics with
				// Boolean values. The `value_type` of the output is `INT64`.
				//Aggregation_ALIGN_COUNT_TRUE Aggregation_Aligner = 16
				config = RequestConfig{
					Alignment:   16,
					Measurement: measurement + "_true_count",
				}
				measurementConfig.Config = append(measurementConfig.Config, config)

				// Align the time series by returning the number of `False` values in
				// each alignment period. This aligner is valid for `GAUGE` metrics with
				// Boolean values. The `value_type` of the output is `INT64`.
				//Aggregation_ALIGN_COUNT_FALSE Aggregation_Aligner = 24
				config = RequestConfig{
					Alignment:   24,
					Measurement: measurement + "_false_count",
				}
				measurementConfig.Config = append(measurementConfig.Config, config)

				// Align the time series by returning the ratio of the number of `True`
				// values to the total number of values in each alignment period. This
				// aligner is valid for `GAUGE` metrics with Boolean values. The output
				// value is in the range [0.0, 1.0] and has `value_type` `DOUBLE`.
				//Aggregation_ALIGN_FRACTION_TRUE Aggregation_Aligner = 17
				config = RequestConfig{
					Alignment:   17,
					Measurement: measurement + "_percent_true",
				}
				measurementConfig.Config = append(measurementConfig.Config, config)

			}
		case "DELTA":
			switch metricDescriptor.ValueType.String() {
			case "INT64", "DOUBLE":
				// Align and convert to
				// [DELTA][google.api.MetricDescriptor.MetricKind.DELTA].
				// The output is `delta = y1 - y0`.
				//
				// This alignment is valid for
				// [CUMULATIVE][google.api.MetricDescriptor.MetricKind.CUMULATIVE] and
				// `DELTA` metrics. If the selected alignment period results in periods
				// with no data, then the aligned value for such a period is created by
				// interpolation. The `value_type`  of the aligned result is the same as
				// the `value_type` of the input.
				// Aggregation_ALIGN_DELTA Aggregation_Aligner = 1
				config = RequestConfig{
					Alignment:   1,
					Measurement: measurement,
				}
				measurementConfig.Config = append(measurementConfig.Config, config)
				// Align and convert to a rate. The result is computed as
				// `rate = (y1 - y0)/(t1 - t0)`, or "delta over time".
				// Think of this aligner as providing the slope of the line that passes
				// through the value at the start and at the end of the `alignment_period`.
				//
				// This aligner is valid for `CUMULATIVE`
				// and `DELTA` metrics with numeric values. If the selected alignment
				// period results in periods with no data, then the aligned value for
				// such a period is created by interpolation. The output is a `GAUGE`
				// metric with `value_type` `DOUBLE`.
				//
				// If, by "rate", you mean "percentage change", see the
				// `ALIGN_PERCENT_CHANGE` aligner instead.
				//Aggregation_ALIGN_RATE Aggregation_Aligner = 2
				config = RequestConfig{
					Alignment:   2,
					Measurement: measurement + "_rate",
				}
				measurementConfig.Config = append(measurementConfig.Config, config)
			}

		}

		/*
			fmt.Printf(" Type: %v\n  MetricKind: %v\n  ValueType %v\n  Labels: %v\n Measurement:%v\n",
				metricDescriptor.Type,
				metricDescriptor.MetricKind,
				metricDescriptor.ValueType,
				metricDescriptor.Labels,
				measurement,
			)
		*/
		measurementJSON.Measurements = append(measurementJSON.Measurements, measurementConfig)
	}
	output, err := json.MarshalIndent(measurementJSON, "", "    ")

	fmt.Printf("  %+v\n", string(output))
}
