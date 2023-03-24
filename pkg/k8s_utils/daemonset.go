package k8s_utils

import (
	"context"

	"github.com/crowdstrike/falcon-operator/pkg/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RestartDaemonSet restarts all pods that belong to the daemonset
func RestartDaemonSet(ctx context.Context, cli client.Client, dsUpdate *appsv1.DaemonSet) error {
	return cli.DeleteAllOf(ctx, &corev1.Pod{}, client.InNamespace(dsUpdate.GetNamespace()), client.MatchingLabels{common.FalconComponentKey: common.FalconKernelSensor})
}
