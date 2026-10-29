package predicates

import (
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func newCM(name string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
	}
}

// --- CrowdStrikeLabel UpdateFunc ---

func TestCrowdStrikeLabelUpdate_GenerationChanged(t *testing.T) {
	p := CrowdStrikeLabel()
	old := newCM("test")
	old.Generation = 1
	nw := newCM("test")
	nw.Generation = 2
	if !p.Update(event.UpdateEvent{ObjectOld: old, ObjectNew: nw}) {
		t.Error("expected Update=true when generation changes")
	}
}

func TestCrowdStrikeLabelUpdate_SpecChanged(t *testing.T) {
	p := CrowdStrikeLabel()
	old := newCM("test")
	nw := newCM("test")
	nw.Data = map[string]string{"key": "value"}
	if !p.Update(event.UpdateEvent{ObjectOld: old, ObjectNew: nw}) {
		t.Error("expected Update=true when spec content changes")
	}
}

func TestCrowdStrikeLabelUpdate_MetadataOnly_OldLabelHasCrowdStrike(t *testing.T) {
	p := CrowdStrikeLabel()
	old := newCM("test")
	old.Labels = map[string]string{"crowdstrike.com/component": "sensor"}
	nw := newCM("test")
	if !p.Update(event.UpdateEvent{ObjectOld: old, ObjectNew: nw}) {
		t.Error("expected Update=true when old labels contain crowdstrike.com")
	}
}

func TestCrowdStrikeLabelUpdate_MetadataOnly_NewLabelHasCrowdStrike(t *testing.T) {
	p := CrowdStrikeLabel()
	old := newCM("test")
	nw := newCM("test")
	nw.Labels = map[string]string{"crowdstrike.com/component": "sensor"}
	if !p.Update(event.UpdateEvent{ObjectOld: old, ObjectNew: nw}) {
		t.Error("expected Update=true when new labels contain crowdstrike.com")
	}
}

func TestCrowdStrikeLabelUpdate_MetadataOnly_OldAnnotationHasCrowdStrike(t *testing.T) {
	p := CrowdStrikeLabel()
	old := newCM("test")
	old.Annotations = map[string]string{"crowdstrike.com/injected": "true"}
	nw := newCM("test")
	if !p.Update(event.UpdateEvent{ObjectOld: old, ObjectNew: nw}) {
		t.Error("expected Update=true when old annotations contain crowdstrike.com")
	}
}

func TestCrowdStrikeLabelUpdate_MetadataOnly_NewAnnotationHasCrowdStrike(t *testing.T) {
	p := CrowdStrikeLabel()
	old := newCM("test")
	nw := newCM("test")
	nw.Annotations = map[string]string{"crowdstrike.com/injected": "true"}
	if !p.Update(event.UpdateEvent{ObjectOld: old, ObjectNew: nw}) {
		t.Error("expected Update=true when new annotations contain crowdstrike.com")
	}
}

func TestCrowdStrikeLabelUpdate_MetadataOnly_NoCrowdStrikeKey(t *testing.T) {
	p := CrowdStrikeLabel()
	old := newCM("test")
	old.Labels = map[string]string{"app": "myapp"}
	nw := newCM("test")
	nw.Labels = map[string]string{"app": "otherapp"}
	if p.Update(event.UpdateEvent{ObjectOld: old, ObjectNew: nw}) {
		t.Error("expected Update=false when metadata-only change has no crowdstrike.com key")
	}
}

func TestCrowdStrikeLabelUpdate_MetadataOnly_UnrelatedAnnotation(t *testing.T) {
	p := CrowdStrikeLabel()
	old := newCM("test")
	nw := newCM("test")
	nw.Annotations = map[string]string{"kubectl.kubernetes.io/last-applied-configuration": "{}"}
	if p.Update(event.UpdateEvent{ObjectOld: old, ObjectNew: nw}) {
		t.Error("expected Update=false when annotation has no crowdstrike.com key")
	}
}

// --- Create, Delete, Generic always pass through ---

func TestCrowdStrikeLabelCreate(t *testing.T) {
	if !CrowdStrikeLabel().Create(event.CreateEvent{}) {
		t.Error("expected Create=true")
	}
}

func TestCrowdStrikeLabelDelete(t *testing.T) {
	if !CrowdStrikeLabel().Delete(event.DeleteEvent{}) {
		t.Error("expected Delete=true")
	}
}

func TestCrowdStrikeLabelGeneric(t *testing.T) {
	if !CrowdStrikeLabel().Generic(event.GenericEvent{}) {
		t.Error("expected Generic=true")
	}
}

// --- hasCrowdStrikeKey ---

func TestHasCrowdStrikeKey_Match(t *testing.T) {
	if !hasCrowdStrikeKey(map[string]string{"crowdstrike.com/foo": "bar"}) {
		t.Error("expected true for key containing crowdstrike.com")
	}
}

func TestHasCrowdStrikeKey_NoMatch(t *testing.T) {
	if hasCrowdStrikeKey(map[string]string{"app": "foo", "tier": "backend"}) {
		t.Error("expected false when no key contains crowdstrike.com")
	}
}

// --- onlyMetadataChanged ---

func TestOnlyMetadataChanged_Identical(t *testing.T) {
	if !onlyMetadataChanged(newCM("test"), newCM("test")) {
		t.Error("expected true for identical objects")
	}
}

func TestOnlyMetadataChanged_LabelsDiffer(t *testing.T) {
	a := newCM("test")
	b := newCM("test")
	b.Labels = map[string]string{"app": "foo"}
	if !onlyMetadataChanged(a, b) {
		t.Error("expected true when only labels differ")
	}
}

func TestOnlyMetadataChanged_AnnotationsDiffer(t *testing.T) {
	a := newCM("test")
	b := newCM("test")
	b.Annotations = map[string]string{"note": "yes"}
	if !onlyMetadataChanged(a, b) {
		t.Error("expected true when only annotations differ")
	}
}

func TestOnlyMetadataChanged_ResourceVersionDiffers(t *testing.T) {
	a := newCM("test")
	b := newCM("test")
	b.ResourceVersion = "12345"
	if !onlyMetadataChanged(a, b) {
		t.Error("expected true when only resourceVersion differs")
	}
}

func TestOnlyMetadataChanged_ManagedFieldsDiffer(t *testing.T) {
	a := newCM("test")
	b := newCM("test")
	b.ManagedFields = []metav1.ManagedFieldsEntry{{Manager: "kubectl"}}
	if !onlyMetadataChanged(a, b) {
		t.Error("expected true when only managedFields differ")
	}
}

func TestOnlyMetadataChanged_DataDiffers(t *testing.T) {
	a := newCM("test")
	b := newCM("test")
	b.Data = map[string]string{"key": "value"}
	if onlyMetadataChanged(a, b) {
		t.Error("expected false when data (spec) differs")
	}
}

// --- CrowdStrikeLabel lets workload status changes through ---

func TestCrowdStrikeLabelUpdate_WorkloadStatusChanged(t *testing.T) {
	p := CrowdStrikeLabel()
	old := &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "falcon-sensor", Namespace: "default"}}
	nw := old.DeepCopy()
	nw.Status.NumberReady = 1
	if !p.Update(event.UpdateEvent{ObjectOld: old, ObjectNew: nw}) {
		t.Error("expected Update=true when workload status changes, so readiness is noticed right away")
	}
}

// --- IgnoreStatusUpdates UpdateFunc ---

func newFCG() *falconv1alpha1.FalconClusterGuard {
	return &falconv1alpha1.FalconClusterGuard{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "falconclusterguard-sample",
			Generation: 1,
			Labels:     map[string]string{"crowdstrike.com/component": "clusterguard", "app": "falcon"},
		},
	}
}

func TestIgnoreStatusUpdates_StatusOnly(t *testing.T) {
	p := IgnoreStatusUpdates()
	old := newFCG()
	nw := old.DeepCopy()
	nw.ResourceVersion = "2"
	meta.SetStatusCondition(&nw.Status.Conditions, metav1.Condition{
		Type: falconv1alpha1.ConditionNodeSensorReady, Status: metav1.ConditionFalse, Reason: "Progressing", Message: "0 of 3 pods ready",
	})
	if p.Update(event.UpdateEvent{ObjectOld: old, ObjectNew: nw}) {
		t.Error("expected Update=false when only status changes, even though crowdstrike.com labels are present")
	}
}

func TestIgnoreStatusUpdates_GenerationChanged(t *testing.T) {
	p := IgnoreStatusUpdates()
	old := newFCG()
	nw := old.DeepCopy()
	nw.Generation = 2
	if !p.Update(event.UpdateEvent{ObjectOld: old, ObjectNew: nw}) {
		t.Error("expected Update=true when generation changes")
	}
}

func TestIgnoreStatusUpdates_DeletionTimestampSet(t *testing.T) {
	p := IgnoreStatusUpdates()
	old := newFCG()
	nw := old.DeepCopy()
	now := metav1.Now()
	nw.DeletionTimestamp = &now
	if !p.Update(event.UpdateEvent{ObjectOld: old, ObjectNew: nw}) {
		t.Error("expected Update=true when the deletion timestamp is set")
	}
}

func TestIgnoreStatusUpdates_CrowdStrikeKeyChanged(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*falconv1alpha1.FalconClusterGuard)
	}{
		{"label", func(o *falconv1alpha1.FalconClusterGuard) { o.Labels["crowdstrike.com/component"] = "other" }},
		{"annotation", func(o *falconv1alpha1.FalconClusterGuard) {
			o.Annotations = map[string]string{"crowdstrike.com/injected": "true"}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := newFCG()
			nw := old.DeepCopy()
			tt.mutate(nw)
			if !IgnoreStatusUpdates().Update(event.UpdateEvent{ObjectOld: old, ObjectNew: nw}) {
				t.Errorf("expected Update=true when a crowdstrike.com %s changes", tt.name)
			}
		})
	}
}

func TestIgnoreStatusUpdates_UnrelatedMetadataChanged(t *testing.T) {
	p := IgnoreStatusUpdates()
	old := newFCG()
	nw := old.DeepCopy()
	nw.Labels["app"] = "other"
	nw.Annotations = map[string]string{"kubectl.kubernetes.io/last-applied-configuration": "{}"}
	if p.Update(event.UpdateEvent{ObjectOld: old, ObjectNew: nw}) {
		t.Error("expected Update=false when only non-crowdstrike.com labels and annotations change")
	}
}

// --- crowdStrikeKeysChanged ---

func TestCrowdStrikeKeysChanged(t *testing.T) {
	tests := []struct {
		name     string
		old, new map[string]string
		want     bool
	}{
		{"both nil", nil, nil, false},
		{"unchanged", map[string]string{"crowdstrike.com/a": "1"}, map[string]string{"crowdstrike.com/a": "1"}, false},
		{"only unrelated key changed", map[string]string{"app": "a"}, map[string]string{"app": "b"}, false},
		{"value changed", map[string]string{"crowdstrike.com/a": "1"}, map[string]string{"crowdstrike.com/a": "2"}, true},
		{"removed", map[string]string{"crowdstrike.com/a": "1"}, nil, true},
		{"added", nil, map[string]string{"crowdstrike.com/a": "1"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := crowdStrikeKeysChanged(tt.old, tt.new); got != tt.want {
				t.Errorf("crowdStrikeKeysChanged() = %v, want %v", got, tt.want)
			}
		})
	}
}
