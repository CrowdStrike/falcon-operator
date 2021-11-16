package falcon_registry

import (
	"context"
	"strings"
)

func (reg *FalconRegistry) LastContainerTag(ctx context.Context, versionRequested *string) (string, error) {
	systemContext, err := reg.systemContext()
	if err != nil {
		return "", err
	}

	return lastTag(ctx, systemContext, reg.imageUri(), func(tag string) bool {
		return (tag[0] >= '0' && tag[0] <= '9' &&
			strings.Contains(tag, ".container.x86_64") &&
			(versionRequested == nil || strings.HasPrefix(tag, *versionRequested)))
	})
}
