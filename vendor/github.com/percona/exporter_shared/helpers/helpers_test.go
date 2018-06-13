// Copyright 2017 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helpers

import (
	"reflect"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestReadMetric(t *testing.T) {
	for expected, m := range map[*Metric]prometheus.Metric{
		&Metric{
			"counter",
			"metric description",
			prometheus.Labels{"job": "test1", "instance": "test2"},
			dto.MetricType_COUNTER,
			36.6,
		}: prometheus.MustNewConstMetric(
			prometheus.NewDesc("counter", "metric description", []string{"instance"}, prometheus.Labels{"job": "test1"}),
			prometheus.CounterValue,
			36.6,
			"test2",
		),

		&Metric{
			"gauge",
			"metric description",
			prometheus.Labels{"job": "test1", "instance": "test2"},
			dto.MetricType_GAUGE,
			36.6,
		}: prometheus.MustNewConstMetric(
			prometheus.NewDesc("gauge", "metric description", []string{"instance"}, prometheus.Labels{"job": "test1"}),
			prometheus.GaugeValue,
			36.6,
			"test2",
		),

		&Metric{
			"untyped",
			"metric description",
			prometheus.Labels{"job": "test1", "instance": "test2"},
			dto.MetricType_UNTYPED,
			36.6,
		}: prometheus.MustNewConstMetric(
			prometheus.NewDesc("untyped", "metric description", []string{"instance"}, prometheus.Labels{"job": "test1"}),
			prometheus.UntypedValue,
			36.6,
			"test2",
		),
	} {
		actual := ReadMetric(m)
		if !reflect.DeepEqual(expected, actual) {
			t.Errorf("expected = %+v\nactual = %+v", expected, actual)
		}
	}
}
