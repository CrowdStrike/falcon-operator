package k8s_utils

import (
	corev1 "k8s.io/api/core/v1"
)

func IsPodRunning(pod *corev1.Pod) bool {
	return pod.Status.Phase == corev1.PodRunning
}
