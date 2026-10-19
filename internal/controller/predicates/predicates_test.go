package predicates

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
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

func TestHasCrowdStrikeKey_Nil(t *testing.T) {
	if hasCrowdStrikeKey(nil) {
		t.Error("expected false for nil map")
	}
}

func TestHasCrowdStrikeKey_Empty(t *testing.T) {
	if hasCrowdStrikeKey(map[string]string{}) {
		t.Error("expected false for empty map")
	}
}

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
