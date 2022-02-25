package k8s_utils

import (
	"bytes"
	"context"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func GetPodLog(ctx context.Context, config *rest.Config, pod *corev1.Pod, containerName string) (string, error) {
	podLogOpts := corev1.PodLogOptions{
		Container: containerName,
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", fmt.Errorf("Cannot build kubernetes go client: %w", err)
	}
	req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("Error in opening pod log stream: %w", err)
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", fmt.Errorf("Error in copy information from podLogs to buffer: %w", err)
	}
	str := buf.String()

	return str, nil
}
