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

package controller

import (
	"context"
	"testing"
	"time"

	"docker.io/go-docker/api/types/swarm"
	"github.com/mtneug/pkg/startstopper/testutils"
	"github.com/mtneug/spate/event"
	"github.com/stretchr/testify/require"
)

func TestEventLoopRun(t *testing.T) {
	t.Parallel()

	eq := make(chan event.Event)
	el := newEventLoop(eq, nil)

	// stopChan
	stopChan := make(chan struct{})
	close(stopChan)
	err := el.run(context.Background(), stopChan)
	require.NoError(t, err)
	stopChan = make(chan struct{})

	// ctx
	ctx, cancle := context.WithCancel(context.Background())
	cancle()
	err = el.run(ctx, stopChan)
	require.EqualError(t, err, "context canceled")

	// handleEvent
	done := make(chan struct{})
	go func() {
		err = el.run(context.Background(), stopChan)
		require.NoError(t, err)
	}()
	go func() {
		eq <- event.New("test_event", swarm.Service{})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Did not stop after 1s")
	}
	close(stopChan)
}

func TestHandleEventUnknownEvent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	e := event.New("test_event", swarm.Service{})
	m := &testutils.MockMap{}

	el := newEventLoop(nil, m)
	el.handleEvent(ctx, e)

	m.AssertExpectations(t)
}

func TestHandleEventServiceCreated(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	srv := swarm.Service{
		ID: "testSrv",
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Labels: map[string]string{
					"de.mtneug.spate.metric.cpu.type": "cpu",
				},
			},
		},
	}
	e := event.New(event.TypeServiceCreated, srv)

	m := &testutils.MockMap{}
	m.On("AddAndStart", ctx, "testSrv").Return(true, nil).Once()

	el := newEventLoop(nil, m)
	el.handleEvent(ctx, e)

	m.AssertExpectations(t)
}

func TestHandleEventServiceUpdated(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	srv := swarm.Service{
		ID: "testSrv",
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Labels: map[string]string{
					"de.mtneug.spate.metric.cpu.type": "cpu",
				},
			},
		},
	}
	e := event.New(event.TypeServiceUpdated, srv)

	m := &testutils.MockMap{}
	m.On("UpdateAndRestart", ctx, "testSrv").Return(true, nil).Once()

	el := newEventLoop(nil, m)
	el.handleEvent(ctx, e)

	m.AssertExpectations(t)
}

func TestHandleEventServiceDeleted(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	srv := swarm.Service{
		ID: "testSrv",
	}
	e := event.New(event.TypeServiceDeleted, srv)

	m := &testutils.MockMap{}
	m.On("DeleteAndStop", ctx, "testSrv").Return(true, nil).Once()

	el := newEventLoop(nil, m)
	el.handleEvent(ctx, e)

	m.AssertExpectations(t)
}
