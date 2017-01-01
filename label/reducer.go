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

package label

import "github.com/mtneug/pkg/reducer"

// ParseReducer parses the labels and returnes the correct reducer.
func ParseReducer(labels map[string]string) (reducer.Reducer, error) {
	aggregationMethodStr, ok := labels[MetricAggregationMethodSuffix]
	if !ok {
		return reducer.Avg(), nil
	}

	switch aggregationMethodStr {
	case MetricAggregationMethodMax:
		return reducer.Max(), nil
	case MetricAggregationMethodMin:
		return reducer.Min(), nil
	case MetricAggregationMethodAvg:
		return reducer.Avg(), nil
	}

	return nil, ErrUnknownAggregationMethod
}
