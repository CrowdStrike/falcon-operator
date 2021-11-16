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

	"github.com/crowdstrike/falcon-operator/pkg/falcon_container/falcon_registry"
	"github.com/crowdstrike/falcon-operator/pkg/registry_auth"
	"github.com/crowdstrike/gofalcon/falcon"
)

type ImageRefresher struct {
	ctx                   context.Context
	log                   logr.Logger
	falconConfig          *falcon.ApiConfig
	insecureSkipTLSVerify bool
	pushCredentials       registry_auth.Credentials
}

func NewImageRefresher(ctx context.Context, log logr.Logger, falconConfig *falcon.ApiConfig, pushAuth registry_auth.Credentials, insecureSkipTLSVerify bool) *ImageRefresher {
	return &ImageRefresher{
		ctx:                   ctx,
		log:                   log,
		falconConfig:          falconConfig,
		insecureSkipTLSVerify: insecureSkipTLSVerify,
		pushCredentials:       pushAuth,
	}
}

func (r *ImageRefresher) Refresh(imageDestination string, versionRequested *string) (string, error) {
	falconTag, srcRef, sourceCtx, err := r.source(versionRequested)
	if err != nil {
		return "", err
	}
	r.log.Info("Identified the latest Falcon Container image", "reference", srcRef.DockerReference().String())

	policy := &signature.Policy{Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()}}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return "", fmt.Errorf("Error loading trust policy: %v", err)
	}
	defer func() { _ = policyContext.Destroy() }()

	destinationCtx, err := r.destinationContext(r.insecureSkipTLSVerify)
	if err != nil {
		return "", err
	}

	// Push to the registry with the falconTag
	dest := fmt.Sprintf("docker://%s:%s", imageDestination, falconTag)
	destRef, err := alltransports.ParseImageName(dest)
	if err != nil {
		return "", fmt.Errorf("Invalid destination name %s: %v", dest, err)
	}
	r.log.Info("Identified the target location for image push", "reference", destRef.DockerReference().String())
	_, err = copy.Image(r.ctx, policyContext, destRef, srcRef,
		&copy.Options{
			ReportWriter:   os.Stdout,
			SourceCtx:      sourceCtx,
			DestinationCtx: destinationCtx,
		},
	)
	if err != nil {
		return "", wrapWithHint(err)
	}

	// Push to the registry with the latest tag
	dest = fmt.Sprintf("docker://%s", imageDestination)
	destRef, err = alltransports.ParseImageName(dest)
	if err != nil {
		return "", fmt.Errorf("Invalid destination name %s: %v", dest, err)
	}
	r.log.Info("Identified the target location for image push", "reference", destRef.DockerReference().String())
	_, err = copy.Image(r.ctx, policyContext, destRef, srcRef,
		&copy.Options{
			ReportWriter:   os.Stdout,
			SourceCtx:      sourceCtx,
			DestinationCtx: destinationCtx,
		},
	)
	return falconTag, wrapWithHint(err)
}

func (r *ImageRefresher) source(versionRequested *string) (falconTag string, falconImage types.ImageReference, systemContext *types.SystemContext, err error) {
	registry, err := falcon_registry.NewFalconRegistry(r.falconConfig)
	if err != nil {
		return
	}
	return registry.PullInfo(r.ctx, versionRequested)
}

func (r *ImageRefresher) destinationContext(insecureSkipTLSVerify bool) (*types.SystemContext, error) {
	ctx, err := r.pushCredentials.DestinationContext()
	if err != nil {
		return nil, err
	}

	if insecureSkipTLSVerify {
		ctx.DockerInsecureSkipTLSVerify = 1
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
