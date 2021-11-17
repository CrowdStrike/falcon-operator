package falcon_container_deployer

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (d *FalconContainerDeployer) InjectorPod() (*corev1.Pod, error) {
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace("falcon-system"),
		client.MatchingLabels(map[string]string{"app": "injector", "crowdstrike.com/component": "crowdstrike-falcon-injector"}),
	}
	if err := d.Client.List(d.Ctx, podList, listOpts...); err != nil {
		d.Log.Error(err, "Failed to list pods")
		return nil, err
	}
	if len(podList.Items) != 1 {
		return nil, fmt.Errorf("Found %d relevant pods, expected 1 pod", len(podList.Items))
	}
	return &podList.Items[0], nil
}
