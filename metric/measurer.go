// Copyright (c) 2016 Matthias Neugebauer <mtneug@mailbox.org>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metric

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"sync"

	"docker.io/go-docker/api/types"
	"docker.io/go-docker/api/types/filters"
	"github.com/mtneug/pkg/reducer"
	"github.com/mtneug/spate/docker"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

// CriticalFailurePercentage indicates for a replica metric how many
// measurements are allowed to fail.
const CriticalFailurePercentage = 0.5

var (
	// ErrUnknownType indicates that the type is unknown.
	ErrUnknownType = errors.New("metric: unknown type")

	// ErrMetricNotFound indicates that the metric was not found.
	ErrMetricNotFound = errors.New("metric: Metric not found")

	// ErrTooManyFailedMeasurements indicates that too many measurements failed.
	ErrTooManyFailedMeasurements = errors.New("metric: Too many failed measurements")
)

// Measurer measures a metric for a given service.
type Measurer interface {
	Measure(ctx context.Context) (float64, error)
}

// NewMeasurer creates the right measurer for given metric.
func NewMeasurer(serviceID, serviceName string, metric Metric) (measurer Measurer, err error) {
	switch metric.Type {
	case TypeCPU:
		measurer = &CPUMeasurer{ServiceID: serviceID, ServiceName: serviceName, Metric: metric}
	case TypeMemory:
		measurer = &MemoryMeasurer{ServiceID: serviceID, ServiceName: serviceName, Metric: metric}
	case TypePrometheus:
		measurer = &PrometheusMeasurer{ServiceID: serviceID, ServiceName: serviceName, Metric: metric}
	default:
		err = ErrUnknownType
	}
	return
}

// CPUMeasurer measures the CPU utilization.
type CPUMeasurer struct {
	ServiceID   string
	ServiceName string
	Metric      Metric
}

// Measure the CPU utilization.
func (m *CPUMeasurer) Measure(ctx context.Context) (float64, error) {
	// TODO: implement
	return 0, errors.New("not implemented")
}

// MemoryMeasurer measures the memory utilization.
type MemoryMeasurer struct {
	ServiceID   string
	ServiceName string
	Metric      Metric
}

// Measure the memory utilization.
func (m *MemoryMeasurer) Measure(ctx context.Context) (float64, error) {
	// TODO: implement
	return 0, errors.New("not implemented")
}

// PrometheusMeasurer measures the Prometheus metric.
type PrometheusMeasurer struct {
	ServiceID   string
	ServiceName string
	Metric      Metric
	client      http.Client
}

// Measure the Prometheus metric.
func (m *PrometheusMeasurer) Measure(ctx context.Context) (float64, error) {
	// Determine expected number of measurements
	var expectedNMeasurements int

	switch m.Metric.Kind {
	case KindSystem:
		expectedNMeasurements = 1

	case KindReplica:
		args := filters.NewArgs()
		args.Add("service", m.ServiceID)
		args.Add("desired-state", "running")

		tasks, err := docker.C.TaskList(ctx, types.TaskListOptions{Filters: args})
		if err != nil {
			return 0, err
		}

		expectedNMeasurements = len(tasks)
	}

	// Lookup IP addresses to replace localhost
	var addrs []string
	if m.Metric.Prometheus.Endpoint.Host == "localhost" {
		// We use Swarm mode service discovery to get the IPs of all containers.
		// See https://docs.docker.com/engine/swarm/networking/#/use-swarm-mode-service-discovery
		var err error
		addrs, err = net.LookupHost("tasks." + m.ServiceName)
		if err != nil {
			return 0, err
		}

		if len(addrs) < expectedNMeasurements {
			// TODO: should probably tell the user about it, but this package should
			//       not depend on logrus.
			expectedNMeasurements = len(addrs)
		}
	}

	// Measure
	wg := &sync.WaitGroup{}
	mutex := &sync.Mutex{}
	measures := make([]float64, 0, len(addrs))

	for _i := 0; _i < expectedNMeasurements; _i++ {
		i := _i

		wg.Add(1)
		go func() {
			defer wg.Done()

			url := m.Metric.Prometheus.Endpoint
			if url.Host == "localhost" {
				url.Host = addrs[i]
			}

			req, err2 := http.NewRequest("GET", url.String(), nil)
			if err2 != nil {
				return
			}

			resp, err2 := m.client.Do(req.WithContext(ctx))
			if err2 != nil {
				return
			}
			defer drainAndCloseReader(resp.Body)

			sample, err2 := decodeAndFindPrometheusSample(resp.Body, m.Metric.Prometheus.Name)
			if err2 != nil {
				return
			}

			mutex.Lock()
			measures = append(measures, float64(sample.Value))
			mutex.Unlock()
		}()
	}
	wg.Wait()

	// Correct missing measurements
	if len(measures) < expectedNMeasurements {
		if float64(len(measures)) < float64(expectedNMeasurements)*CriticalFailurePercentage {
			return 0, ErrTooManyFailedMeasurements
		}

		// TODO: should probably tell the user about it, but this package should
		//       not depend on logrus.

		skipped := float64(expectedNMeasurements - len(measures))
		avg, err := reducer.Avg().Reduce(measures)
		if err != nil {
			return 0, err
		}
		measures = append(measures, skipped*avg)
	}

	// Aggregate
	sum, err := reducer.Sum().Reduce(measures)
	if err != nil {
		return 0, err
	}

	return sum, nil
}

func decodeAndFindPrometheusSample(r io.Reader, metricName string) (*model.Sample, error) {
	dec := expfmt.SampleDecoder{
		Dec:  expfmt.NewDecoder(r, expfmt.FmtText),
		Opts: &expfmt.DecodeOptions{},
	}

	for {
		var vec model.Vector

		err := dec.Decode(&vec)
		if err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		if len(vec) == 0 {
			continue
		}

		sample := vec[0]
		name := sample.Metric[model.MetricNameLabel]
		if name == model.LabelValue(metricName) {
			return sample, nil
		}
	}

	return nil, ErrMetricNotFound
}

func drainAndCloseReader(r io.ReadCloser) {
	_, _ = io.CopyN(ioutil.Discard, r, 1024)
	_ = r.Close()
}
