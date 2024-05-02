package common

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func getFakeClient(initObjs ...client.Object) (client.WithWatch, error) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := appsv1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	// ...
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjs...).Build(), nil
}

func TestCheckRunningPodLabels(t *testing.T) {
	ctx := context.Background()

	fakeClient, err := getFakeClient()
	if err != nil {
		t.Fatalf("TestCheckRunningPodLabels getFakeClient() error = %v", err)
	}

	err = fakeClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-namespace"}})
	if err != nil {
		t.Fatalf("TestCheckRunningPodLabels Create() error = %v", err)
	}

	testLabel := map[string]string{"crowdstrike.com/provider": "crowdstrike", "testLabel": "testPod"}
	err = fakeClient.Create(ctx, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "test-namespace", Labels: testLabel}})
	if err != nil {
		t.Fatalf("TestCheckRunningPodLabels Create() error = %v", err)
	}

	matchingLabels := client.MatchingLabels{"testLabel": "testPod"}

	got, err := CheckRunningPodLabels(fakeClient, ctx, "test-namespace", matchingLabels)
	if err != nil {
		t.Errorf("CheckRunningPodLabels() error = %v", err)
	}

	want := true
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("CheckRunningPodLabels() mismatch (-want +got):\n%s", diff)
	}

	// Test with non-matching labels
	matchingLabels = client.MatchingLabels{"testLabel": "nonMatchingValue"}

	got, err = CheckRunningPodLabels(fakeClient, ctx, "test-namespace", matchingLabels)
	if err != nil {
		t.Errorf("CheckRunningPodLabels() error = %v", err)
	}

	want = false
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("CheckRunningPodLabels() mismatch (-want +got):\n%s", diff)
	}
}

func TestGetReadyPod(t *testing.T) {
	ctx := context.Background()

	fakeClient, err := getFakeClient()
	if err != nil {
		t.Errorf("TestGetReadyPod getFakeClient() error = %v", err)
	}

	testLabel := map[string]string{"testLabel": "testPod"}

	wantErr := "No webhook service pod found in a Ready state"
	_, gotErr := GetReadyPod(fakeClient, ctx, "test-namespace", testLabel)
	if diff := cmp.Diff(wantErr, gotErr.Error()); diff != "" {
		t.Errorf("GetReadyPod() mismatch (-want +got):\n%s", diff)
	}

	err = fakeClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-namespace"}})
	if err != nil {
		t.Errorf("TestGetReadyPod Create() error = %v", err)
	}

	err = fakeClient.Create(ctx, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "test-namespace", Labels: testLabel}, Status: corev1.PodStatus{Phase: corev1.PodRunning, Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}}}})
	if err != nil {
		t.Errorf("TestGetReadyPod Create() error = %v", err)
	}

	want := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "test-namespace", Labels: testLabel, ResourceVersion: "1"}, Status: corev1.PodStatus{Phase: corev1.PodRunning, Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}}}}
	got, err := GetReadyPod(fakeClient, ctx, "test-namespace", testLabel)
	if err != nil {
		t.Errorf("GetReadyPod() error = %v", err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("GetReadyPod() mismatch (-want +got):\n%s", diff)
	}

}

func TestGetDeployment(t *testing.T) {
	ctx := context.Background()

	fakeClient, err := getFakeClient()
	if err != nil {
		t.Errorf("TestGetDeployment getFakeClient() error = %v", err)
	}

	testLabel := map[string]string{"testLabel": "test"}

	err = fakeClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-namespace"}})
	if err != nil {
		t.Errorf("TestGetDeployment Create() error = %v", err)
	}
	err = fakeClient.Create(ctx, &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "test-deployment", Namespace: "test-namespace", Labels: testLabel}})
	if err != nil {
		t.Errorf("TestGetDeployment Create() error = %v", err)
	}

	want := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "test-deployment", Namespace: "test-namespace", Labels: testLabel, ResourceVersion: "1"}}
	got, err := GetDeployment(fakeClient, ctx, "test-namespace", testLabel)
	if err != nil {
		t.Errorf("GetDeployment() error = %v", err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("GetDeployment() mismatch (-want +got):\n%s", diff)
	}
}

func TestGetRunningFalconNS(t *testing.T) {
	ctx := context.Background()

	fakeClient, err := getFakeClient()
	if err != nil {
		t.Errorf("TestGetRunningFalconNS getFakeClient() error = %v", err)
	}

	err = fakeClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-namespace"}})
	if err != nil {
		t.Errorf("TestGetRunningFalconNS Create() error = %v", err)
	}

	err = fakeClient.Create(ctx, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "test-namespace", Labels: map[string]string{"crowdstrike.com/provider": "crowdstrike"}}})
	if err != nil {
		t.Errorf("TestGetRunningFalconNS Create() error = %v", err)
	}

	want := []string{"test-namespace"}
	got, err := GetRunningFalconNS(fakeClient, ctx)
	if err != nil {
		t.Errorf("GetRunningFalconNS() error = %v", err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("GetRunningFalconNS() mismatch (-want +got):\n%s", diff)
	}
}

func TestGetOpenShiftNamespaceNamesSort(t *testing.T) {
	ctx := context.Background()

	fakeClient, err := getFakeClient()
	if err != nil {
		t.Errorf("TestGetNamespaceNamesSort getFakeClient() error = %v", err)
	}

	for _, val := range []string{"test2", "defaultz", "kube-systemz", "openshift", "falcon-system"} {
		err = fakeClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: val}})
		if err != nil {
			t.Errorf("TestGetNamespaceNamesSort Create() error = %v", err)
		}
	}

	want := []string{"openshift"}
	got, err := GetOpenShiftNamespaceNamesSort(ctx, fakeClient)
	if err != nil {
		t.Errorf("GetNamespaceNamesSort() error = %v", err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("getNamespaceNamesSort() mismatch (-want +got):\n%s", diff)
	}
}

func TestOLogMessage(t *testing.T) {
	want := "test.test"
	got := oLogMessage("test", "test")

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("oLogMessage() mismatch (-want +got):\n%s", diff)
	}
}

func TestLogMessage(t *testing.T) {
	want := "test test test"
	got := logMessage("test", "test", "test")

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("logMessage() mismatch (-want +got):\n%s", diff)
	}
}
