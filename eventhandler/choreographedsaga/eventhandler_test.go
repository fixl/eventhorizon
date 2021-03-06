// Copyright (c) 2021 - The Event Horizon authors.
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

package choreographedsaga

import (
	"context"
	"errors"
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
	"reflect"
	"testing"
	"time"
)

func TestEventHandler(t *testing.T) {
	commandHandler := &mocks.CommandHandler{
		Commands: []eh.Command{},
	}

	a := mocks.NewAggregate(uuid.New())
	aggregateStore := &mocks.AggregateStore{
		Aggregates: map[uuid.UUID]eh.Aggregate{
			a.EntityID(): a,
		},
	}
	saga := &TestSaga{}
	handler := NewEventHandler(saga, aggregateStore, commandHandler)

	ctx := context.Background()

	eventData := &TestEventData{
		TestAggregateID: a.EntityID(),
	}
	timestamp := time.Date(2021, time.March, 6, 14, 0, 0, 0, time.UTC)
	event := eh.NewEvent(mocks.EventType, eventData, timestamp,
		eh.ForAggregate(mocks.AggregateType, a.EntityID(), 1))
	saga.commands = []eh.Command{&mocks.Command{ID: event.AggregateID(), Content: "content"}}

	if err := handler.HandleEvent(ctx, event); err != nil {
		t.Error("there should be no error:", err)
	}
	if saga.event != event {
		t.Error("the handled event should be correct: ", saga.event)
	}
	if !reflect.DeepEqual(commandHandler.Commands, saga.commands) {
		t.Error("the produced commands should be correct: ", commandHandler.Commands)
	}
	if saga.aggregate != a {
		t.Error("the loaded aggregate should be correct: ", saga.aggregate)
	}
}

func TestEventProcessFail(t *testing.T) {
	commandHandler := &mocks.CommandHandler{
		Commands: []eh.Command{},
	}

	a := mocks.NewAggregate(uuid.New())
	aggregateStore := &mocks.AggregateStore{
		Aggregates: map[uuid.UUID]eh.Aggregate{
			a.EntityID(): a,
		},
	}
	saga := &TestSaga{}
	handler := NewEventHandler(saga, aggregateStore, commandHandler)

	ctx := context.Background()

	eventData := &mocks.EventData{
		Content: "content",
	}
	timestamp := time.Date(2021, time.March, 6, 14, 0, 0, 0, time.UTC)
	event := eh.NewEvent(mocks.EventType, eventData, timestamp,
		eh.ForAggregate(mocks.AggregateType, a.EntityID(), 1))

	expectedError := Error{
		Err:       errors.New("cannot process event"),
		Saga:      saga.SagaType().String(),
		Namespace: eh.NamespaceFromContext(ctx),
	}
	err := handler.HandleEvent(ctx, event)

	if !errors.Is(err, expectedError) {
		t.Error("there should be an error", err)
	}
}

func TestAggregateLoadFail(t *testing.T) {
	commandHandler := &mocks.CommandHandler{
		Commands: []eh.Command{},
	}

	a := mocks.NewAggregate(uuid.New())
	aggregateStore := &mocks.AggregateStore{
		Err: errors.New("aggregate does not exist"),
	}
	saga := &TestSaga{}
	handler := NewEventHandler(saga, aggregateStore, commandHandler)

	ctx := context.Background()

	eventData := &TestEventData{
		TestAggregateID: a.EntityID(),
	}
	timestamp := time.Date(2021, time.March, 6, 14, 0, 0, 0, time.UTC)
	event := eh.NewEvent(mocks.EventType, eventData, timestamp,
		eh.ForAggregate(mocks.AggregateType, a.EntityID(), 1))

	expectedError := Error{
		Err:       aggregateStore.Err,
		Saga:      saga.SagaType().String(),
		Namespace: eh.NamespaceFromContext(ctx),
	}
	err := handler.HandleEvent(ctx, event)

	if !errors.Is(err, expectedError) {
		t.Error("there should be an error", err)
	}
}

type TestEventData struct {
	TestAggregateID uuid.UUID
	Err             error
}

const (
	TestSagaType Type = "TestSaga"
)

type TestSaga struct {
	event     eh.Event
	aggregate eh.Aggregate
	context   context.Context
	commands  []eh.Command
}

func (s *TestSaga) SagaType() Type {
	return TestSagaType
}

func (s *TestSaga) ForAggregate(event eh.Event) (eh.AggregateType, uuid.UUID, error) {
	if data, ok := event.Data().(*TestEventData); ok {
		return mocks.AggregateType, data.TestAggregateID, nil
	}

	return "", uuid.Nil, errors.New("cannot process event")
}

func (s *TestSaga) RunSaga(ctx context.Context, aggregate eh.Aggregate, event eh.Event, h eh.CommandHandler) error {
	s.context = ctx
	s.aggregate = aggregate
	s.event = event
	for _, cmd := range s.commands {
		return h.HandleCommand(ctx, cmd)
	}

	return nil
}
