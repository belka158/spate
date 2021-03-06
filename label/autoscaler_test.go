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

package label_test

import (
	"math"
	"testing"
	"time"

	"github.com/mtneug/spate/autoscaler"
	"github.com/mtneug/spate/label"
	"github.com/stretchr/testify/require"
)

func TestParseAutoscaler(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		labels     map[string]string
		err        error
		autoscaler *autoscaler.Autoscaler
	}{
		{
			labels: map[string]string{"autoscaler.period": "abc"},
			err:    label.ErrInvalidDuration,
		},
		{
			labels: map[string]string{"autoscaler.cooldown.scaled_up": "abc"},
			err:    label.ErrInvalidDuration,
		},
		{
			labels: map[string]string{"autoscaler.cooldown.scaled_down": "abc"},
			err:    label.ErrInvalidDuration,
		},
		{
			labels: map[string]string{"autoscaler.cooldown.service_added": "abc"},
			err:    label.ErrInvalidDuration,
		},
		{
			labels: map[string]string{"autoscaler.cooldown.service_updated": "abc"},
			err:    label.ErrInvalidDuration,
		},
		{
			labels: map[string]string{"replica.min": "abc"},
			err:    label.ErrInvalidUint,
		},
		{
			labels: map[string]string{"replica.max": "abc"},
			err:    label.ErrInvalidUint,
		},
		{
			labels: nil,
			err:    nil,
			autoscaler: &autoscaler.Autoscaler{
				Period:                    30 * time.Second,
				CooldownServiceScaledUp:   3 * time.Minute,
				CooldownServiceScaledDown: 5 * time.Minute,
				CooldownServiceCreated:    0 * time.Second,
				CooldownServiceUpdated:    0 * time.Second,
				MinReplicas:               1,
				MaxReplicas:               math.MaxUint64,
			},
		},
		{
			labels: map[string]string{
				"autoscaler.period":                   "1m",
				"autoscaler.cooldown.scaled_up":       "2m",
				"autoscaler.cooldown.scaled_down":     "3m",
				"autoscaler.cooldown.service_added":   "4m",
				"autoscaler.cooldown.service_updated": "5m",
				"replica.min":                         "6",
				"replica.max":                         "7",
			},
			err: nil,
			autoscaler: &autoscaler.Autoscaler{
				Period:                    1 * time.Minute,
				CooldownServiceScaledUp:   2 * time.Minute,
				CooldownServiceScaledDown: 3 * time.Minute,
				CooldownServiceCreated:    4 * time.Minute,
				CooldownServiceUpdated:    5 * time.Minute,
				MinReplicas:               6,
				MaxReplicas:               7,
			},
		},
	}

	for _, c := range testCases {
		a := autoscaler.Autoscaler{}
		err := label.ParseAutoscaler(&a, c.labels)
		require.Equal(t, c.err, err)
		if c.err == nil {
			require.Equal(t, c.autoscaler, &a)
		}
	}
}
