package controllers

import (
	"context"
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	releaseRepo = "registry.crowdstrike.com/falcon-sensor/release/falcon-sensor"
	stageRepo   = "registry.crowdstrike.com/falcon-sensor/stage/falcon-sensor"
)

func TestSyncDiscoveredImage(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name         string
		storedURI    string // ImageReady message persisted at discovery; empty means no condition
		storedSensor string
		memorySensor string // status.sensor after image.URI resolved imageURI
		imageURI     string
		wantUpdated  bool
		wantReady    string // expected persisted ImageReady message; empty means condition removed
		wantSensor   *string
	}{
		{
			name:         "unchanged image is a no-op",
			storedURI:    releaseRepo + ":7.39.0-1111",
			storedSensor: "7.39.0-1111",
			memorySensor: "7.39.0-1111",
			imageURI:     releaseRepo + ":7.39.0-1111",
			wantUpdated:  false,
			wantReady:    releaseRepo + ":7.39.0-1111",
			wantSensor:   strPtr("7.39.0-1111"),
		},
		{
			name:         "newly discovered tag is persisted",
			storedURI:    releaseRepo + ":7.39.0-1111",
			storedSensor: "7.39.0-1111",
			memorySensor: "7.40.0-2222",
			imageURI:     releaseRepo + ":7.40.0-2222",
			wantUpdated:  true,
			wantReady:    releaseRepo + ":7.40.0-2222",
			wantSensor:   strPtr("7.40.0-2222"),
		},
		{
			name:         "repository change clears the locked tag",
			storedURI:    releaseRepo + ":7.39.0-1111",
			storedSensor: "7.39.0-1111",
			memorySensor: "7.39.0-1111",
			imageURI:     stageRepo + ":7.39.0-1111",
			wantUpdated:  true,
			wantReady:    "",
			wantSensor:   nil,
		},
		{
			name:        "missing ImageReady condition is a no-op",
			imageURI:    releaseRepo + ":7.39.0-1111",
			wantUpdated: false,
			wantReady:   "",
			wantSensor:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			scheme := runtime.NewScheme()
			_ = falconv1alpha1.AddToScheme(scheme)

			stored := &falconv1alpha1.FalconClusterGuard{ObjectMeta: metav1.ObjectMeta{Name: "test-fcg"}}
			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(stored).WithStatusSubresource(stored).Build()
			r := &FalconClusterGuardReconciler{Client: c, Reader: c, RuntimeScheme: scheme, log: logr.Discard()}

			fcg := &falconv1alpha1.FalconClusterGuard{}
			if err := c.Get(ctx, types.NamespacedName{Name: "test-fcg"}, fcg); err != nil {
				t.Fatal(err)
			}
			if tt.storedURI != "" {
				fcg.Status.Sensor = strPtr(tt.storedSensor)
				meta.SetStatusCondition(&fcg.Status.Conditions, metav1.Condition{
					Type:    falconv1alpha1.ConditionImageReady,
					Status:  metav1.ConditionTrue,
					Reason:  falconv1alpha1.ReasonDiscovered,
					Message: tt.storedURI,
				})
				if err := c.Status().Update(ctx, fcg); err != nil {
					t.Fatal(err)
				}
				fcg.Status.Sensor = strPtr(tt.memorySensor)
			}

			updated, err := r.syncDiscoveredImage(ctx, fcg, tt.imageURI)
			if err != nil {
				t.Fatalf("syncDiscoveredImage() error = %v", err)
			}
			if updated != tt.wantUpdated {
				t.Errorf("syncDiscoveredImage() updated = %v, want %v", updated, tt.wantUpdated)
			}

			persisted := &falconv1alpha1.FalconClusterGuard{}
			if err := c.Get(ctx, types.NamespacedName{Name: "test-fcg"}, persisted); err != nil {
				t.Fatal(err)
			}
			cond := meta.FindStatusCondition(persisted.Status.Conditions, falconv1alpha1.ConditionImageReady)
			switch {
			case tt.wantReady == "" && cond != nil:
				t.Errorf("ImageReady = %q, want condition removed", cond.Message)
			case tt.wantReady != "" && (cond == nil || cond.Message != tt.wantReady):
				t.Errorf("ImageReady = %v, want message %q", cond, tt.wantReady)
			}
			switch {
			case tt.wantSensor == nil && persisted.Status.Sensor != nil:
				t.Errorf("status.sensor = %q, want nil", *persisted.Status.Sensor)
			case tt.wantSensor != nil && (persisted.Status.Sensor == nil || *persisted.Status.Sensor != *tt.wantSensor):
				t.Errorf("status.sensor = %v, want %q", persisted.Status.Sensor, *tt.wantSensor)
			}
		})
	}
}

func TestClearDiscoveredImage(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name        string
		storedURI   string // ImageReady message persisted at discovery; empty means no condition
		wantCleared bool
	}{
		{name: "discovered image is cleared", storedURI: releaseRepo + ":7.39.0-1111", wantCleared: true},
		{name: "no discovered image is a no-op"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			scheme := runtime.NewScheme()
			_ = falconv1alpha1.AddToScheme(scheme)

			stored := &falconv1alpha1.FalconClusterGuard{ObjectMeta: metav1.ObjectMeta{Name: "test-fcg"}}
			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(stored).WithStatusSubresource(stored).Build()
			r := &FalconClusterGuardReconciler{Client: c, Reader: c, RuntimeScheme: scheme, log: logr.Discard()}

			fcg := &falconv1alpha1.FalconClusterGuard{}
			if err := c.Get(ctx, types.NamespacedName{Name: "test-fcg"}, fcg); err != nil {
				t.Fatal(err)
			}
			fcg.Status.Sensor = strPtr("7.39.0-1111")
			if tt.storedURI != "" {
				meta.SetStatusCondition(&fcg.Status.Conditions, metav1.Condition{
					Type:    falconv1alpha1.ConditionImageReady,
					Status:  metav1.ConditionTrue,
					Reason:  falconv1alpha1.ReasonDiscovered,
					Message: tt.storedURI,
				})
			}
			if err := c.Status().Update(ctx, fcg); err != nil {
				t.Fatal(err)
			}

			cleared, err := r.clearDiscoveredImage(ctx, fcg)
			if err != nil {
				t.Fatalf("clearDiscoveredImage() error = %v", err)
			}
			if cleared != tt.wantCleared {
				t.Errorf("clearDiscoveredImage() cleared = %v, want %v", cleared, tt.wantCleared)
			}

			persisted := &falconv1alpha1.FalconClusterGuard{}
			if err := c.Get(ctx, types.NamespacedName{Name: "test-fcg"}, persisted); err != nil {
				t.Fatal(err)
			}
			if cond := meta.FindStatusCondition(persisted.Status.Conditions, falconv1alpha1.ConditionImageReady); cond != nil {
				t.Errorf("ImageReady = %q, want condition removed", cond.Message)
			}
			if tt.wantCleared && persisted.Status.Sensor != nil {
				t.Errorf("status.sensor = %q, want nil", *persisted.Status.Sensor)
			}
			if !tt.wantCleared && (persisted.Status.Sensor == nil || *persisted.Status.Sensor != "7.39.0-1111") {
				t.Errorf("status.sensor = %v, want unchanged %q", persisted.Status.Sensor, "7.39.0-1111")
			}
		})
	}
}
