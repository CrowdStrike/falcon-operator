package k8s_utils

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RestartDeamonSet restarts all pods that belong to the daemonset
func RestartDeamonSet(ctx context.Context, cli client.Client, dsUpdate *appsv1.DaemonSet) error {
	ds := &appsv1.DaemonSet{}
	err := cli.Get(ctx, types.NamespacedName{Name: dsUpdate.Name, Namespace: dsUpdate.Namespace}, ds)
	if err != nil {
		return err
	}

	if err := deleteDaemonSetPods(cli, ctx, ds); err != nil {
		return err
	}

	return nil
}

// deleteDaemonSetPods deletes the old pods associated with the daemonset to start new pods
func deleteDaemonSetPods(c client.Client, ctx context.Context, ds *appsv1.DaemonSet) error {
	var pods corev1.PodList

	if err := c.List(ctx, &pods, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{"app": ds.Name}),
		Namespace:     ds.Namespace,
	}); err != nil {
		return err
	}

	for podIDs := range pods.Items {
		pod := pods.Items[podIDs]
		err := c.Delete(ctx, &pod, &client.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}
