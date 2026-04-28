package controllers

import (
	"context"
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
)

func boolPtr(b bool) *bool {
	return &b
}

func TestNewConfigMapVisibilityVars(t *testing.T) {
	tests := []struct {
		name                           string
		deployWatcher                  *bool
		watcherEnabled                 *bool
		snapshotsEnabled               *bool
		configMapWatcherEnabled        *bool
		wantSnapshotsEnabled           string
		wantWatchEventsEnabled         string
		wantVisibilityConfigMapEnabled string
	}{
		{
			name:                           "watcherEnabled=false only disables watch events, snapshots and configmap watcher are independent",
			watcherEnabled:                 boolPtr(false),
			snapshotsEnabled:               boolPtr(true),
			configMapWatcherEnabled:        boolPtr(true),
			wantSnapshotsEnabled:           "true",
			wantWatchEventsEnabled:         "false",
			wantVisibilityConfigMapEnabled: "true",
		},
		{
			name:                           "watcherEnabled=false with other defaults leaves snapshots and configmap watcher at their defaults",
			watcherEnabled:                 boolPtr(false),
			snapshotsEnabled:               nil, // defaults to true
			configMapWatcherEnabled:        nil, // defaults to true
			wantSnapshotsEnabled:           "true",
			wantWatchEventsEnabled:         "false",
			wantVisibilityConfigMapEnabled: "true",
		},
		{
			name:                           "deploy watcher false disables all visibility vars regardless of watcherEnabled",
			deployWatcher:                  boolPtr(false),
			watcherEnabled:                 nil, // defaults to true
			snapshotsEnabled:               nil, // defaults to true
			configMapWatcherEnabled:        nil, // defaults to true
			wantSnapshotsEnabled:           "false",
			wantWatchEventsEnabled:         "false",
			wantVisibilityConfigMapEnabled: "false",
		},
		{
			name:                           "watcher enabled with all toggles true",
			watcherEnabled:                 boolPtr(true),
			snapshotsEnabled:               boolPtr(true),
			configMapWatcherEnabled:        boolPtr(true),
			wantSnapshotsEnabled:           "true",
			wantWatchEventsEnabled:         "true",
			wantVisibilityConfigMapEnabled: "true",
		},
		{
			name:                           "watcher enabled with individual toggles false",
			watcherEnabled:                 boolPtr(true),
			snapshotsEnabled:               boolPtr(false),
			configMapWatcherEnabled:        boolPtr(false),
			wantSnapshotsEnabled:           "false",
			wantWatchEventsEnabled:         "true",
			wantVisibilityConfigMapEnabled: "false",
		},
		{
			name:                           "all defaults (everything true)",
			watcherEnabled:                 nil, // defaults to true
			snapshotsEnabled:               nil, // defaults to true
			configMapWatcherEnabled:        nil, // defaults to true
			wantSnapshotsEnabled:           "true",
			wantWatchEventsEnabled:         "true",
			wantVisibilityConfigMapEnabled: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cid := "test-cid"
			falconAdmission := &falconv1alpha1.FalconAdmission{}
			falconAdmission.Spec.Falcon.CID = &cid
			falconAdmission.Spec.AdmissionConfig.DeployWatcher = tt.deployWatcher
			falconAdmission.Spec.AdmissionConfig.WatcherEnabled = tt.watcherEnabled
			falconAdmission.Spec.AdmissionConfig.SnapshotsEnabled = tt.snapshotsEnabled
			falconAdmission.Spec.AdmissionConfig.ConfigMapWatcherEnabled = tt.configMapWatcherEnabled

			r := &FalconAdmissionReconciler{}
			cm, err := r.newConfigMap(context.Background(), "test-config", falconAdmission)
			if err != nil {
				t.Fatalf("newConfigMap() returned unexpected error: %v", err)
			}

			if got := cm.Data["__CS_SNAPSHOTS_ENABLED"]; got != tt.wantSnapshotsEnabled {
				t.Errorf("__CS_SNAPSHOTS_ENABLED = %q, want %q", got, tt.wantSnapshotsEnabled)
			}
			if got := cm.Data["__CS_WATCH_EVENTS_ENABLED"]; got != tt.wantWatchEventsEnabled {
				t.Errorf("__CS_WATCH_EVENTS_ENABLED = %q, want %q", got, tt.wantWatchEventsEnabled)
			}
			if got := cm.Data["__CS_VISIBILITY_CONFIGMAPS_ENABLED"]; got != tt.wantVisibilityConfigMapEnabled {
				t.Errorf("__CS_VISIBILITY_CONFIGMAPS_ENABLED = %q, want %q", got, tt.wantVisibilityConfigMapEnabled)
			}
		})
	}
}
