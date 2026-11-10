package falcon_registry

import (
	"testing"

	"github.com/crowdstrike/gofalcon/falcon"
)

func TestClusterGuardImageURI(t *testing.T) {
	tests := []struct {
		name       string
		cloud      falcon.CloudType
		buildStage string
		want       string
	}{
		{
			name:       "us-1 release",
			cloud:      falcon.CloudUs1,
			buildStage: "release",
			want:       "registry.crowdstrike.com/falcon-sensor/release/falcon-sensor",
		},
		{
			name:       "us-1 stage",
			cloud:      falcon.CloudUs1,
			buildStage: "stage",
			want:       "registry.crowdstrike.com/falcon-sensor/stage/falcon-sensor",
		},
		{
			name:       "empty build stage defaults to release",
			cloud:      falcon.CloudEu1,
			buildStage: "",
			want:       "registry.crowdstrike.com/falcon-sensor/release/falcon-sensor",
		},
		{
			name:       "us-gov-1 stage",
			cloud:      falcon.CloudUsGov1,
			buildStage: "stage",
			want:       "registry.laggar.gcw.crowdstrike.com/falcon-sensor/stage/falcon-sensor",
		},
		{
			name:       "us-gov-2 release",
			cloud:      falcon.CloudUsGov2,
			buildStage: "release",
			want:       "registry.us-gov-2.crowdstrike.mil/falcon-sensor/release/falcon-sensor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClusterGuardImageURI(tt.cloud, tt.buildStage); got != tt.want {
				t.Errorf("ClusterGuardImageURI() = %q, want %q", got, tt.want)
			}
		})
	}
}
