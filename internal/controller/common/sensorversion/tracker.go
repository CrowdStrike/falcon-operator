package sensorversion

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Handler func(context.Context, types.NamespacedName) error
type SensorVersionQuery func(context.Context) (string, error)

type Tracker struct {
	activeTracks    map[types.NamespacedName]*track
	ctx             context.Context
	logger          logr.Logger
	pollingInterval time.Duration
	trackUpdates    chan track
}

type track struct {
	getSensorVersion SensorVersionQuery
	handler          Handler
	name             types.NamespacedName
	priorVersion     string
}

func NewTracker(ctx context.Context, pollingInterval time.Duration) Tracker {
	return Tracker{
		activeTracks:    make(map[types.NamespacedName]*track),
		ctx:             ctx,
		logger:          log.FromContext(ctx).WithName("sensor-version-tracker"),
		pollingInterval: pollingInterval,
		trackUpdates:    make(chan track),
	}
}

func (tracker Tracker) KeepTrackingChanges() {
	const backoffInterval = time.Second * 5

	for {
		err := tracker.TrackChanges()
		if err == nil {
			break
		}

		tracker.logger.Error(err, "change-tracking failed")
		time.Sleep(backoffInterval)
	}
}

func (tracker Tracker) StopTracking(name types.NamespacedName) {
	tracker.trackUpdates <- track{
		name: name,
	}
}

func (tracker Tracker) Track(name types.NamespacedName, getSensorVersion SensorVersionQuery, handler Handler) {
	tracker.trackUpdates <- track{
		getSensorVersion: getSensorVersion,
		handler:          handler,
		name:             name,
	}
}

func (tracker Tracker) TrackChanges() error {
	tracker.logDebug("started tracking changes")

	timer := time.NewTimer(0)

	for {
		select {
		case <-tracker.ctx.Done():
			tracker.logDebug("stopped tracking changes")
			return nil

		case update := <-tracker.trackUpdates:
			if update.getSensorVersion != nil && update.handler != nil {
				if err := tracker.updateTrack(update); err != nil {
					return err
				}
			} else {
				if _, exists := tracker.activeTracks[update.name]; exists {
					delete(tracker.activeTracks, update.name)
					tracker.logDebug("deleted track", "namespace", update.name.Namespace, "name", update.name.Name)
				}
			}

		case <-timer.C:
			if err := tracker.runCycle(); err != nil {
				return err
			}
			timer.Reset(tracker.pollingInterval)
		}
	}
}

func NewTestTracker() (Tracker, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	tracker := NewTracker(ctx, time.Hour)
	go tracker.KeepTrackingChanges()
	return tracker, cancel
}

func (tracker Tracker) logDebug(msg string, keysAndValues ...any) {
	tracker.logger.V(1).Info(msg, keysAndValues...)
}

func (tracker Tracker) runCycle() error {
	tracker.logDebug("started run cycle")

	for name, trk := range tracker.activeTracks {
		currentVersion, err := trk.getSensorVersion(tracker.ctx)
		if err != nil {
			return err
		}
		tracker.logDebug("queried sensor version", "namespace", name.Namespace, "name", name.Name, "version", currentVersion)

		if currentVersion != trk.priorVersion {
			tracker.logDebug("sensor version changed, calling handler", "namespace", name.Namespace, "name", name.Name, "version", currentVersion)
			if err := trk.handler(tracker.ctx, name); err != nil {
				return err
			}
		}

		trk.priorVersion = currentVersion
	}

	return nil
}

func (tracker Tracker) updateTrack(update track) error {
	trk, exists := tracker.activeTracks[update.name]
	if exists {
		trk.getSensorVersion = update.getSensorVersion
		trk.handler = update.handler
		tracker.logDebug("updated track", "namespace", update.name.Namespace, "name", update.name.Name)
		return nil
	}

	initialVersion, err := update.getSensorVersion(tracker.ctx)
	if err != nil {
		return err
	}

	tracker.activeTracks[update.name] = &track{
		getSensorVersion: update.getSensorVersion,
		handler:          update.handler,
		name:             update.name,
		priorVersion:     initialVersion,
	}

	tracker.logDebug("added track", "namespace", update.name.Namespace, "name", update.name.Name, "initialVersion", initialVersion)
	return nil
}
