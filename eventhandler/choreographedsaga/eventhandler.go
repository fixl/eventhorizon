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
	"fmt"
	"github.com/google/uuid"
	eh "github.com/looplab/eventhorizon"
)

// EventHandler is a CQRS saga handler to run a Saga implementation with a fully loaded aggregate
type EventHandler struct {
	saga           Saga
	aggregateStore eh.AggregateStore
	commandHandler eh.CommandHandler
}

// Saga is an interface for a CQRS saga that listens to events and generates commands
// A fully loaded aggregate is passed to the saga in order make decisions about
type Saga interface {
	// SagaType returns the type of the saga.
	SagaType() Type

	// ForAggregate extracts the aggregate type and ID for which aggregate to load.
	// Because a saga can be choreographed across many aggregates, this varies from event to event.
	ForAggregate(event eh.Event) (eh.AggregateType, uuid.UUID, error)

	// RunSaga handles an event in saga that can returns commands.
	// A fully loaded aggregate is passed to the saga to allow making
	// business decisions based on current state.
	RunSaga(context.Context, eh.Aggregate, eh.Event, eh.CommandHandler) error
}

// Type is the type of a sage, used as its unique identifier
type Type string

// String returns the string representation of a saga type.
func (t Type) String() string {
	return string(t)
}

// Error is an error in the saga, with the namespace.
type Error struct {
	// Err is the error that happened when projecting the event.
	Err error
	// Saga is the saga where the error happened.
	Saga string
	// Namespace is the namespace for the error.
	Namespace string
}

// Error implements the Error method of the errors.Error interface.
func (e Error) Error() string {
	return fmt.Sprintf("%s: %s (%s)", e.Saga, e.Err, e.Namespace)
}

// Unwrap implements the errors.Unwrap method.
func (e Error) Unwrap() error {
	return e.Err
}

// Cause implements the github.com/pkg/errors Unwrap method.
func (e Error) Cause() error {
	return e.Unwrap()
}

// NewEventHandler creates a new EventHandler
func NewEventHandler(saga Saga, aggregateStore eh.AggregateStore, commandHandler eh.CommandHandler) *EventHandler {
	return &EventHandler{
		saga:           saga,
		aggregateStore: aggregateStore,
		commandHandler: commandHandler,
	}
}

// HandlerType implements the HandlerType method of the eventhorizon.EventHandler interface.
func (h *EventHandler) HandlerType() eh.EventHandlerType {
	return eh.EventHandlerType("aggregate_saga_" + h.saga.SagaType())
}

// HandleEvent implements the HandleEvent method of the eventhorizon.EventHandler interface.
func (h *EventHandler) HandleEvent(ctx context.Context, event eh.Event) error {
	aggregateType, aggregateID, err := h.saga.ForAggregate(event)
	if err != nil {
		return Error{
			Err:       err,
			Saga:      h.saga.SagaType().String(),
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	aggregate, err := h.aggregateStore.Load(ctx, aggregateType, aggregateID)
	if err != nil {
		return Error{
			Err:       err,
			Saga:      h.saga.SagaType().String(),
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	if err := h.saga.RunSaga(ctx, aggregate, event, h.commandHandler); err != nil {
		return Error{
			Err:       err,
			Saga:      h.saga.SagaType().String(),
			Namespace: eh.NamespaceFromContext(ctx),
		}
	}

	return nil
}
