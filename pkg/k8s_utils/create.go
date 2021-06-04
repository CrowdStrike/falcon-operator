package k8s_utils

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Create(ctx context.Context, cli client.Client, objects []runtime.Object, logger logr.Logger) error {
	for _, obj := range objects {
		switch t := obj.(type) {
		case client.Object:
			logger.Info("Creating Falcon Container object on the cluster", "Kind", t.GetObjectKind().GroupVersionKind().Kind, "Name", t.GetName())
			err := cli.Create(ctx, t)
			if err != nil {
				if errors.IsAlreadyExists(err) {
					logger.Info("Already Exists")
				} else {
					return err
				}
			}
		default:
			return fmt.Errorf("Unrecognized kube object type: %T", obj)
		}
	}

	return nil
}
