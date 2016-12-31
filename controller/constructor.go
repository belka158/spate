// Copyright © 2016 Matthias Neugebauer <mtneug@mailbox.org>
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

package controller

import (
	"errors"

	"github.com/docker/docker/api/types/swarm"
	"github.com/mtneug/pkg/reducer"
	"github.com/mtneug/spate/api/types"
	"github.com/mtneug/spate/autoscaler"
	"github.com/mtneug/spate/labels"
	"github.com/mtneug/spate/metric"
)

var (
	// ErrDuplicateMetric indicates that at least two metrics are the same.
	ErrDuplicateMetric = errors.New("controller: duplicate metric")
)

func constructAutoscaler(srv swarm.Service) (*autoscaler.Autoscaler, error) {
	// extract labels
	sl := make(map[string]string, len(srv.Spec.Labels))
	ml := make(map[string]map[string]string)

	err := labels.ExtractSpateLabels(sl, ml, srv.Spec.Labels)
	if err != nil {
		return nil, err
	}

	// construct objects
	haveSeenMetric := make(map[types.Metric]bool, len(ml))
	goals := make([]autoscaler.Goal, 0, len(ml))

	for metricName, metricLabels := range ml {
		m := metric.New(metricName)
		err = labels.ParseMetric(&m, metricLabels)
		if err != nil {
			return nil, err
		}

		normMetric := m
		normMetric.ID = ""
		normMetric.Name = ""
		if haveSeenMetric[normMetric] {
			return nil, ErrDuplicateMetric
		}
		haveSeenMetric[normMetric] = true

		var measurer metric.Measurer
		measurer, err = metric.NewMeasurer(srv.ID, srv.Spec.Name, m)
		if err != nil {
			return nil, err
		}

		var reducer reducer.Reducer
		reducer, err = labels.ParseReducer(metricLabels)
		if err != nil {
			return nil, err
		}

		observer := metric.NewObserver(measurer, reducer)
		err = labels.ParseObserver(observer, metricLabels)
		if err != nil {
			return nil, err
		}

		target := types.Target{}
		err = labels.ParseTarget(&target, metricLabels)
		if err != nil {
			return nil, err
		}

		goals = append(goals, autoscaler.Goal{Observer: observer, Target: target})
	}

	a, err := autoscaler.New(srv, goals)
	if err != nil {
		return nil, err
	}
	err = labels.ParseAutoscaler(a, sl)
	if err != nil {
		return nil, err
	}

	return a, nil
}
