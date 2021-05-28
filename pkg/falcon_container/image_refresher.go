package falcon_container

// TODO: logging

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-logr/logr"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"

	"github.com/crowdstrike/falcon-operator/pkg/falcon_container/falcon_image"
	"github.com/crowdstrike/gofalcon/falcon"
)

type ImageRefresher struct {
	ctx          context.Context
	log          logr.Logger
	falconConfig *falcon.ApiConfig
}

func NewImageRefresher(ctx context.Context, log logr.Logger, falconConfig *falcon.ApiConfig) *ImageRefresher {
	if falconConfig.Context == nil {
		falconConfig.Context = ctx
	}
	return &ImageRefresher{
		ctx:          ctx,
		log:          log,
		falconConfig: falconConfig,
	}
}

func (r *ImageRefresher) Refresh(imageDestination string) error {
	policy := &signature.Policy{Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()}}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return fmt.Errorf("Error loading trust policy: %v", err)
	}
	defer func() { _ = policyContext.Destroy() }()

	dest := fmt.Sprintf("docker://%s", imageDestination)
	destRef, err := alltransports.ParseImageName(dest)
	if err != nil {
		return fmt.Errorf("Invalid destination name %s: %v", dest, err)
	}

	destinationContext, err := r.destinationContext(destRef)
	if err != nil {
		return err
	}

	image, err := falcon_image.Pull(r.falconConfig, r.log)
	if err != nil {
		return err
	}
	defer func() { _ = image.Delete() }()

	ref, err := image.ImageReference()
	if err != nil {
		return fmt.Errorf("Failed to build internal image representation for falcon image: %v", err)
	}

	r.log.Info("Pushing falcon image", "docker", destRef.StringWithinTransport())
	_, err = copy.Image(r.ctx, policyContext, destRef, ref, &copy.Options{
		DestinationCtx: destinationContext,
		ReportWriter:   os.Stdout,
	})
	return wrapWithHint(err)
}

func (r *ImageRefresher) destinationContext(imageRef types.ImageReference) (*types.SystemContext, error) {
	ctx := &types.SystemContext{
		DockerInsecureSkipTLSVerify: 1,
		LegacyFormatAuthFilePath:    "/tmp/.dockercfg",
	}

	return ctx, nil
}

func wrapWithHint(in error) error {
	// Use of credentials store outside of docker command is somewhat limited
	// See https://github.com/moby/moby/issues/39377
	// https://github.com/containers/image/pull/656
	if in == nil {
		return in
	}
	if strings.Contains(in.Error(), "authentication required") {
		return fmt.Errorf("Could not authenticate to the registry: %w", in)
	}
	return in
}
