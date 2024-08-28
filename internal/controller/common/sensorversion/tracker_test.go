package sensorversion

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
)

const noPollingInterval = 0

func TestTracker_WhenGettingSensorVersionFails_TrackChangesFailsWithSameError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler := func(_ context.Context, _ types.NamespacedName) error {
		require.Fail(t, "handler unexpectedly called")
		return nil
	}

	expectedError := errors.New("some error")
	alwaysFails := func(_ context.Context) (string, error) {
		return "", expectedError
	}

	tracker := NewTracker(ctx, noPollingInterval)

	done := make(chan any)
	go func() {
		defer close(done)

		actualError := tracker.TrackChanges()
		assert.Equal(t, expectedError, actualError, "wrong error returned from TrackChanges()")
	}()

	name := types.NamespacedName{
		Namespace: "someNamespace",
		Name:      "someName",
	}
	tracker.Track(name, alwaysFails, handler, false)

	select {
	case <-done:
		return

	case <-time.After(time.Second):
		require.Fail(t, "TrackChanges() never returned")
	}
}

func TestTracker_WhenHandlerFails_TrackChangesFailsWithSameError(t *testing.T) {
	expectedContext, cancel := context.WithCancel(context.Background())
	defer cancel()

	getSensorVersion := newIncrementingSensorVersionGenerator(t, expectedContext)

	expectedName := types.NamespacedName{
		Namespace: "someNamespace",
		Name:      "someName",
	}

	expectedError := errors.New("some error")
	handler := func(actualContext context.Context, actualName types.NamespacedName) error {
		assert.Same(t, expectedContext, actualContext, "wrong context passed to handler")
		assert.Equal(t, expectedName, actualName, "wrong name passed to handler")
		return expectedError
	}

	tracker := NewTracker(expectedContext, noPollingInterval)

	done := make(chan any)
	go func() {
		defer close(done)

		actualError := tracker.TrackChanges()
		assert.Equal(t, expectedError, actualError, "wrong error returned from TrackChanges()")
	}()

	tracker.Track(expectedName, getSensorVersion, handler, false)

	select {
	case <-done:
		return

	case <-time.After(time.Second):
		require.Fail(t, "TrackChanges() never returned")
	}
}

func TestTracker_WhenSensorVersionChanges_CallsHandler(t *testing.T) {
	runHandlerTest(t, func(ctx context.Context, tracker Tracker, name types.NamespacedName, handler Handler) {
		getSensorVersion := newIncrementingSensorVersionGenerator(t, ctx)
		tracker.Track(name, getSensorVersion, handler, false)
	})
}

func TestTracker_WhenSensorVersionDoesNotChangeButIsForced_CallsHandler(t *testing.T) {
	runHandlerTest(t, func(ctx context.Context, tracker Tracker, name types.NamespacedName, handler Handler) {
		getSensorVersion := newConstantSensorVersionGenerator(t, ctx)
		tracker.Track(name, getSensorVersion, handler, true)
	})
}

func TestTracker_WhenTrackUpdatedWithForcedHandler_CallsHandler(t *testing.T) {
	runHandlerTest(t, func(ctx context.Context, tracker Tracker, name types.NamespacedName, handler Handler) {
		getSensorVersion := newConstantSensorVersionGenerator(t, ctx)
		tracker.Track(name, getSensorVersion, handler, false)
		tracker.Track(name, getSensorVersion, handler, true)
	})
}

func newConstantSensorVersionGenerator(t *testing.T, expectedContext context.Context) SensorVersionQuery {
	const fixedVersion = "v1.1.1"

	return func(actualContext context.Context) (string, error) {
		assert.Same(t, expectedContext, actualContext, "wrong context passed to getSensorVersion()")
		return fixedVersion, nil
	} //
}

func newIncrementingSensorVersionGenerator(t *testing.T, expectedContext context.Context) SensorVersionQuery {
	lastVersion := 0

	return func(actualContext context.Context) (string, error) {
		assert.Same(t, expectedContext, actualContext, "wrong context passed to getSensorVersion()")

		lastVersion++
		return fmt.Sprintf("v%d.%d.%d", lastVersion, lastVersion, lastVersion), nil
	}
}

func runHandlerTest(t *testing.T, runner func(ctx context.Context, tracker Tracker, name types.NamespacedName, handler Handler)) {
	expectedContext, cancel := context.WithCancel(context.Background())
	defer cancel()

	expectedName := types.NamespacedName{
		Namespace: "someNamespace",
		Name:      "someName",
	}

	done := make(chan any)
	channelOpen := true
	handler := func(actualContext context.Context, actualName types.NamespacedName) error {
		assert.Same(t, expectedContext, actualContext, "wrong context passed to handler")
		assert.Equal(t, expectedName, actualName, "wrong name passed to handler")

		if channelOpen {
			close(done)
			channelOpen = false
		}

		return nil
	}

	tracker := NewTracker(expectedContext, noPollingInterval)

	go func() {
		err := tracker.TrackChanges()
		require.NoError(t, err, "TrackChanges() unexpectedly failed")
	}()

	runner(expectedContext, tracker, expectedName, handler)

	select {
	case <-time.After(time.Second):
		require.Fail(t, "handler never called")

	case <-done:
		break
	}
}
