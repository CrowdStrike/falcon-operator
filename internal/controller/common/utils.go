package common

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var ErrNoWebhookServicePodReady = errors.New("no webhook service pod found in a Ready state")

func Create(r client.Client, sch *runtime.Scheme, ctx context.Context, req ctrl.Request, log logr.Logger, falconObject client.Object, falconStatus *falconv1alpha1.FalconCRStatus, obj runtime.Object) error {
	switch o := obj.(type) {
	case client.Object:
		name := o.GetName()
		namespace := o.GetNamespace()
		gvk := o.GetObjectKind().GroupVersionKind()
		fgvk := falconObject.GetObjectKind().GroupVersionKind()
		condType := fmt.Sprintf("%sReady", strings.ToUpper(gvk.Kind[:1])+gvk.Kind[1:])

		err := ctrl.SetControllerReference(falconObject, o, sch)
		if err != nil {
			log.Error(err, fmt.Sprintf("unable to set controller reference for %s %s", fgvk.Kind, gvk.Kind))
			return err
		}

		log.Info(logMessage("Creating a new", fgvk.Kind, gvk.Kind), oLogMessage(gvk.Kind, "Name"), name, oLogMessage(gvk.Kind, "Namespace"), namespace)
		err = r.Create(ctx, o)
		if err != nil {
			log.Error(err, logMessage("Failed to create", fgvk.Kind, gvk.Kind), oLogMessage(gvk.Kind, "Name"), name, oLogMessage(gvk.Kind, "Namespace"), namespace)

			if err := ConditionsUpdate(r, ctx, req, log, falconObject, falconStatus,
				metav1.Condition{
					Status:             metav1.ConditionFalse,
					Reason:             falconv1alpha1.ReasonInstallFailed,
					Type:               condType,
					Message:            fmt.Sprintf("%s %s creation has failed", fgvk.Kind, gvk.Kind),
					ObservedGeneration: falconObject.GetGeneration(),
				}); err != nil {
				return err
			}

			return err
		}

		if err := ConditionsUpdate(r, ctx, req, log, falconObject, falconStatus,
			metav1.Condition{
				Status:             metav1.ConditionTrue,
				Reason:             falconv1alpha1.ReasonInstallSucceeded,
				Type:               condType,
				Message:            fmt.Sprintf("%s %s has been successfully created", fgvk.Kind, gvk.Kind),
				ObservedGeneration: falconObject.GetGeneration(),
			}); err != nil {
			return err
		}

		return nil
	default:
		return fmt.Errorf("unrecognized kubernetes object type: %T", obj)
	}
}

func Update(r client.Client, ctx context.Context, req ctrl.Request, log logr.Logger, falconObject client.Object, falconStatus *falconv1alpha1.FalconCRStatus, obj runtime.Object) error {
	switch o := obj.(type) {
	case client.Object:
		name := o.GetName()
		namespace := o.GetNamespace()
		gvk := o.GetObjectKind().GroupVersionKind()
		fgvk := falconObject.GetObjectKind().GroupVersionKind()
		condType := fmt.Sprintf("%sReady", strings.ToUpper(gvk.Kind[:1])+gvk.Kind[1:])

		log.Info(logMessage("Updating", fgvk.Kind, gvk.Kind), oLogMessage(gvk.Kind, "Name"), name, oLogMessage(gvk.Kind, "Namespace"), namespace)
		err := r.Update(ctx, o)
		if err != nil {
			log.Error(err, logMessage("Failed to update", fgvk.Kind, gvk.Kind), oLogMessage(gvk.Kind, "Name"), name, oLogMessage(gvk.Kind, "Namespace"), namespace)

			if err := ConditionsUpdate(r, ctx, req, log, falconObject, falconStatus,
				metav1.Condition{
					Status:             metav1.ConditionFalse,
					Reason:             falconv1alpha1.ReasonUpdateFailed,
					Type:               condType,
					Message:            fmt.Sprintf("%s %s update has failed", fgvk.Kind, gvk.Kind),
					ObservedGeneration: falconObject.GetGeneration(),
				}); err != nil {
				return err
			}

			return err
		}

		if err := ConditionsUpdate(r, ctx, req, log, falconObject, falconStatus,
			metav1.Condition{
				Status:             metav1.ConditionTrue,
				Reason:             falconv1alpha1.ReasonUpdateSucceeded,
				Type:               condType,
				Message:            fmt.Sprintf("%s %s has been successfully updated", fgvk.Kind, gvk.Kind),
				ObservedGeneration: falconObject.GetGeneration(),
			}); err != nil {
			return err
		}

		return nil
	default:
		return fmt.Errorf("unrecognized kubernetes object type: %T", obj)
	}
}

func Delete(r client.Client, ctx context.Context, req ctrl.Request, log logr.Logger, falconObject client.Object, falconStatus *falconv1alpha1.FalconCRStatus, obj runtime.Object) error {
	switch o := obj.(type) {
	case client.Object:
		name := o.GetName()
		namespace := o.GetNamespace()
		gvk := o.GetObjectKind().GroupVersionKind()
		fgvk := falconObject.GetObjectKind().GroupVersionKind()
		condType := fmt.Sprintf("%sReady", strings.ToUpper(gvk.Kind[:1])+gvk.Kind[1:])

		log.Info(logMessage("Deleting", fgvk.Kind, gvk.Kind), oLogMessage(gvk.Kind, "Name"), name, oLogMessage(gvk.Kind, "Namespace"), namespace)
		err := r.Delete(ctx, o)
		if err != nil {
			log.Error(err, logMessage("Failed to delete", fgvk.Kind, gvk.Kind), oLogMessage(gvk.Kind, "Name"), name, oLogMessage(gvk.Kind, "Namespace"), namespace)

			if err := ConditionsUpdate(r, ctx, req, log, falconObject, falconStatus,
				metav1.Condition{
					Status:             metav1.ConditionFalse,
					Reason:             falconv1alpha1.ReasonDeleteFailed,
					Type:               condType,
					Message:            fmt.Sprintf("%s %s deletion has failed", fgvk.Kind, gvk.Kind),
					ObservedGeneration: falconObject.GetGeneration(),
				}); err != nil {
				return err
			}

			return err
		}

		if err := ConditionsUpdate(r, ctx, req, log, falconObject, falconStatus,
			metav1.Condition{
				Status:             metav1.ConditionTrue,
				Reason:             falconv1alpha1.ReasonDeleteSucceeded,
				Type:               condType,
				Message:            fmt.Sprintf("%s %s has been successfully deleted", fgvk.Kind, gvk.Kind),
				ObservedGeneration: falconObject.GetGeneration(),
			}); err != nil {
			return err
		}

		return nil
	default:
		return fmt.Errorf("unrecognized kubernetes object type: %T", obj)
	}
}

// ConditionsUpdate updates the Falcon Object CR conditions
func ConditionsUpdate(r client.Client, ctx context.Context, req ctrl.Request, log logr.Logger, falconObject client.Object, falconStatus *falconv1alpha1.FalconCRStatus, falconCondition metav1.Condition) error {
	if !meta.IsStatusConditionPresentAndEqual(falconStatus.Conditions, falconCondition.Type, falconCondition.Status) {
		fgvk := falconObject.GetObjectKind().GroupVersionKind()

		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			// Re-fetch the Custom Resource before update the status
			// so that we have the latest state of the resource on the cluster and we will avoid
			// raise the issue "the object has been modified, please apply
			// your changes to the latest version and try again" which would re-trigger the reconciliation
			err := r.Get(ctx, req.NamespacedName, falconObject)
			if err != nil {
				log.Error(err, fmt.Sprintf("Failed to re-fetch %s for status update", fgvk.Kind))
				return err
			}

			// The following implementation will update the status
			meta.SetStatusCondition(&falconStatus.Conditions, falconCondition)
			return r.Status().Update(ctx, falconObject)
		})
		if err != nil {
			log.Error(err, fmt.Sprintf("Failed to update %s status", fgvk.Kind))
			return err
		}
	}

	return nil
}

func CheckRunningPodLabels(r client.Client, ctx context.Context, namespace string, matchingLabels client.MatchingLabels) (bool, error) {
	podList := &corev1.PodList{}

	listOpts := []client.ListOption{
		client.InNamespace(namespace),
	}

	if err := r.List(ctx, podList, listOpts...); err != nil {
		return false, fmt.Errorf("unable to list pods: %v", err)
	}

	for _, pod := range podList.Items {
		if pod.ObjectMeta.Labels != nil {
			for k, v := range matchingLabels {
				if pod.ObjectMeta.Labels[k] != v {
					return false, nil
				}
			}
		}
	}

	return true, nil
}

func GetReadyPod(r client.Client, ctx context.Context, namespace string, matchingLabels client.MatchingLabels) (*corev1.Pod, error) {
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(namespace),
		matchingLabels,
	}

	if err := r.List(ctx, podList, listOpts...); err != nil {
		return nil, fmt.Errorf("unable to list pods: %v", err)
	}

	for _, pod := range podList.Items {
		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
				return &pod, nil
			}
		}
	}

	return &corev1.Pod{}, ErrNoWebhookServicePodReady
}

func GetDeployment(r client.Client, ctx context.Context, namespace string, matchingLabels client.MatchingLabels) (*appsv1.Deployment, error) {
	depList := &appsv1.DeploymentList{}
	listOpts := []client.ListOption{
		client.InNamespace(namespace),
		matchingLabels,
	}

	if err := r.List(ctx, depList, listOpts...); err != nil {
		return nil, fmt.Errorf("unable to list deployments: %v", err)
	}

	return &depList.Items[0], nil
}

func GetRunningFalconNS(r client.Client, ctx context.Context) ([]string, error) {
	podList := &corev1.PodList{}
	falconNamespaces := []string{}
	listOpts := []client.ListOption{
		client.MatchingLabels{"crowdstrike.com/provider": "crowdstrike"},
	}

	if err := r.List(ctx, podList, listOpts...); err != nil {
		return []string{}, fmt.Errorf("unable to list pods: %v", err)
	}

	for _, pod := range podList.Items {
		if !slices.Contains(falconNamespaces, pod.GetNamespace()) {
			falconNamespaces = append(falconNamespaces, pod.GetNamespace())
		}
	}

	slices.Sort(falconNamespaces)

	return falconNamespaces, nil
}

func GetOpenShiftNamespaceNamesSort(ctx context.Context, cli client.Client) ([]string, error) {
	nsList := []string{}
	ns := &corev1.NamespaceList{}
	err := cli.List(ctx, ns)
	if err != nil {
		return []string{}, err
	}

	for _, i := range ns.Items {
		if strings.Contains(i.Name, "openshift") && !slices.Contains(nsList, i.Name) {
			nsList = append(nsList, i.Name)
		}
	}

	sort.Slice(nsList, func(i, j int) bool { return nsList[i] < nsList[j] })
	return nsList, nil
}

func NewReconcileTrigger(c controller.Controller) (func(client.Object), error) {
	channel := make(chan event.GenericEvent)

	eventHandler := handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
		return []reconcile.Request{
			{NamespacedName: types.NamespacedName{
				Name:      obj.GetName(),
				Namespace: obj.GetNamespace(),
			}},
		}
	})

	channelSource := source.Channel(channel, eventHandler)

	// Watch using the channel source
	err := c.Watch(channelSource)

	if err != nil {
		return nil, err
	}

	return func(obj client.Object) {
		channel <- event.GenericEvent{Object: obj}
	}, nil
}

func oLogMessage(kind, obj string) string {
	return fmt.Sprintf("%s.%s", kind, obj)
}

func logMessage(msg, falconKind, kind string) string {
	return fmt.Sprintf("%s %s %s", msg, falconKind, kind)
}
