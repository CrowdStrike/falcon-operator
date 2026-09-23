package image

import (
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
)

func strPtr(s string) *string { return &s }

func TestVersionLock(t *testing.T) {
	tests := []struct {
		name    string
		version *string
		sensor  *string
		want    bool
	}{
		{
			name:    "version set and sensor matches: locked",
			version: strPtr("6.50"),
			sensor:  strPtr("6.50.0-1234"),
			want:    true,
		},
		{
			name:    "version set but sensor does not match: not locked",
			version: strPtr("6.51"),
			sensor:  strPtr("6.50.0-1234"),
			want:    false,
		},
		{
			name:    "version set but no sensor in status: not locked",
			version: strPtr("6.50"),
			sensor:  nil,
			want:    false,
		},
		{
			name:    "no version pin and sensor present: locked to current",
			version: nil,
			sensor:  strPtr("6.50.0-1234"),
			want:    true,
		},
		{
			name:    "no version pin and no sensor in status: not locked",
			version: nil,
			sensor:  nil,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{Version: tt.version}
			status := &falconv1alpha1.FalconCRStatus{Sensor: tt.sensor}
			if got := versionLock(cfg, status); got != tt.want {
				t.Errorf("versionLock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetTag(t *testing.T) {
	tests := []struct {
		name    string
		sensor  *string
		wantTag string
		wantErr bool
	}{
		{
			name:    "sensor tag present",
			sensor:  strPtr("6.50.0-1234"),
			wantTag: "6.50.0-1234",
			wantErr: false,
		},
		{
			name:    "sensor tag empty string",
			sensor:  strPtr(""),
			wantTag: "",
			wantErr: true,
		},
		{
			name:    "sensor nil",
			sensor:  nil,
			wantTag: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := &falconv1alpha1.FalconCRStatus{Sensor: tt.sensor}
			tag, err := getTag(status)
			if (err != nil) != tt.wantErr {
				t.Errorf("getTag() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tag != tt.wantTag {
				t.Errorf("getTag() = %q, want %q", tag, tt.wantTag)
			}
		})
	}
}

func TestGetTagUsesImageVersion(t *testing.T) {
	// Verify that common.ImageVersion extracts the tag portion correctly,
	// since SetTag stores common.ImageVersion(image) into status.Sensor.
	cases := []struct {
		image string
		want  string
	}{
		{"registry.crowdstrike.com/falcon-sensor/kac:6.50.0-1234", "6.50.0-1234"},
		{"myregistry.io/kac:latest", "latest"},
		// No tag separator: ImageVersion returns the full image string.
		{"myregistry.io/kac", "myregistry.io/kac"},
	}
	for _, c := range cases {
		got := common.ImageVersion(c.image)
		if got == nil {
			t.Errorf("ImageVersion(%q) = nil, want %q", c.image, c.want)
			continue
		}
		if *got != c.want {
			t.Errorf("ImageVersion(%q) = %q, want %q", c.image, *got, c.want)
		}
	}
}
