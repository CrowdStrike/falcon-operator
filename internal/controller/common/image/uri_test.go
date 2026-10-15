package image

import (
	"context"
	"errors"
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/gofalcon/falcon"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const relatedImageEnvVar = "RELATED_IMAGE_TEST_CLUSTER_GUARD"

type fakeResolver struct {
	client.Client
}

func (f fakeResolver) GetK8sReader() client.Reader { return f.Client }

type lastTagCall struct {
	called     bool
	buildStage string
	version    *string
}

// stubLastTag replaces the registry lookup for the duration of the test and records how it was called.
func stubLastTag(t *testing.T, tag string, err error) *lastTagCall {
	t.Helper()
	call := &lastTagCall{}
	orig := lastTag
	lastTag = func(_ context.Context, _ *falcon.ApiConfig, buildStage string, version *string) (string, error) {
		call.called = true
		call.buildStage = buildStage
		call.version = version
		return tag, err
	}
	t.Cleanup(func() { lastTag = orig })
	return call
}

func newResolver(t *testing.T) (fakeResolver, *falconv1alpha1.FalconClusterGuard) {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := falconv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	fcg := &falconv1alpha1.FalconClusterGuard{ObjectMeta: metav1.ObjectMeta{Name: "test-fcg"}}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(fcg).WithStatusSubresource(fcg).Build()
	return fakeResolver{Client: c}, fcg
}

func credentials() *falconv1alpha1.FalconAPI {
	return &falconv1alpha1.FalconAPI{ClientId: "id", ClientSecret: "secret", CloudRegion: "us-1"}
}

// discoverable returns a Config for which the image is discovered from the CrowdStrike registry.
func discoverable() Config {
	return Config{FalconAPI: credentials(), CrowdStrikeRegistry: true}
}

func TestDiscover(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want bool
	}{
		{name: "crowdstrike registry with credentials", cfg: discoverable(), want: true},
		{name: "image set", cfg: Config{Image: "myregistry.io/fcg:1.2.3", FalconAPI: credentials(), CrowdStrikeRegistry: true}},
		{name: "private registry with credentials", cfg: Config{FalconAPI: credentials()}},
		{name: "crowdstrike registry without credentials", cfg: Config{CrowdStrikeRegistry: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Discover(tt.cfg); got != tt.want {
				t.Errorf("Discover() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestURI(t *testing.T) {
	tests := []struct {
		name         string
		cfg          Config
		relatedImage string
		wantURI      string
		wantSensor   string
		wantRegistry bool
		wantErr      bool
	}{
		{
			name:       "image set takes precedence over registry discovery",
			cfg:        Config{Image: "myregistry.io/fcg:1.2.3", FalconAPI: credentials()},
			wantURI:    "myregistry.io/fcg:1.2.3",
			wantSensor: "1.2.3",
		},
		{
			name:         "related image used when credentials are not configured",
			cfg:          Config{},
			relatedImage: "registry.example.com/fcg:4.5.6",
			wantURI:      "registry.example.com/fcg:4.5.6",
			wantSensor:   "4.5.6",
		},
		{
			name:         "discovery takes precedence over related image",
			cfg:          discoverable(),
			relatedImage: "registry.example.com/fcg:4.5.6",
			wantURI:      "registry.crowdstrike.com/falcon-sensor/release/falcon-sensor:7.40.0-1234",
			wantSensor:   "7.40.0-1234",
			wantRegistry: true,
		},
		{
			name:         "stage build stage discovers from the stage repository",
			cfg:          Config{FalconAPI: credentials(), CrowdStrikeRegistry: true, BuildStage: "stage"},
			wantURI:      "registry.crowdstrike.com/falcon-sensor/stage/falcon-sensor:7.40.0-1234",
			wantSensor:   "7.40.0-1234",
			wantRegistry: true,
		},
		{
			name:         "private registry with credentials uses related image",
			cfg:          Config{FalconAPI: credentials()},
			relatedImage: "registry.example.com/fcg:4.5.6",
			wantURI:      "registry.example.com/fcg:4.5.6",
			wantSensor:   "4.5.6",
		},
		{
			name:    "private registry with credentials requires an image",
			cfg:     Config{FalconAPI: credentials()},
			wantErr: true,
		},
		{
			name:    "crowdstrike registry without credentials requires an image",
			cfg:     Config{CrowdStrikeRegistry: true},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(relatedImageEnvVar, tt.relatedImage)
			tt.cfg.RelatedImageEnvVar = relatedImageEnvVar
			call := stubLastTag(t, "7.40.0-1234", nil)
			r, fcg := newResolver(t)

			uri, err := URI(context.Background(), r, tt.cfg, &fcg.Status, fcg)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("URI() = %q, want error", uri)
				}
				if call.called {
					t.Error("registry queried, want no discovery")
				}
				return
			}
			if err != nil {
				t.Fatalf("URI() error = %v", err)
			}
			if uri != tt.wantURI {
				t.Errorf("URI() = %q, want %q", uri, tt.wantURI)
			}
			if fcg.Status.Sensor == nil || *fcg.Status.Sensor != tt.wantSensor {
				t.Errorf("status.Sensor = %v, want %q", fcg.Status.Sensor, tt.wantSensor)
			}
			if call.called != tt.wantRegistry {
				t.Errorf("registry queried = %v, want %v", call.called, tt.wantRegistry)
			}
			if tt.wantRegistry && call.buildStage != tt.cfg.BuildStage {
				t.Errorf("registry build stage = %q, want %q", call.buildStage, tt.cfg.BuildStage)
			}
		})
	}
}

func TestSetTagPassesVersionToRegistry(t *testing.T) {
	call := stubLastTag(t, "7.38.0-1000", nil)
	r, fcg := newResolver(t)
	cfg := discoverable()
	cfg.Version = strPtr("7.38")

	tag, err := SetTag(context.Background(), r, cfg, &fcg.Status, fcg)
	if err != nil {
		t.Fatalf("SetTag() error = %v", err)
	}
	if tag != "7.38.0-1000" {
		t.Errorf("SetTag() = %q, want %q", tag, "7.38.0-1000")
	}
	if call.version == nil || *call.version != "7.38" {
		t.Errorf("registry version = %v, want %q", call.version, "7.38")
	}
}

func TestSetTagVersionLockSkipsRegistry(t *testing.T) {
	call := stubLastTag(t, "7.40.0-1234", nil)
	r, fcg := newResolver(t)
	fcg.Status.Sensor = strPtr("7.39.0-1111")

	tag, err := SetTag(context.Background(), r, discoverable(), &fcg.Status, fcg)
	if err != nil {
		t.Fatalf("SetTag() error = %v", err)
	}
	if tag != "7.39.0-1111" {
		t.Errorf("SetTag() = %q, want locked tag %q", tag, "7.39.0-1111")
	}
	if call.called {
		t.Error("registry queried, want locked tag reused")
	}
}

func TestSetTagReturnsRegistryError(t *testing.T) {
	stubLastTag(t, "", errRegistry)
	r, fcg := newResolver(t)

	if _, err := SetTag(context.Background(), r, discoverable(), &fcg.Status, fcg); err != errRegistry {
		t.Errorf("SetTag() error = %v, want %v", err, errRegistry)
	}
	if fcg.Status.Sensor != nil {
		t.Errorf("status.Sensor = %q, want nil on registry error", *fcg.Status.Sensor)
	}
}

func TestRepository(t *testing.T) {
	cases := map[string]string{
		"registry.crowdstrike.com/falcon-sensor/release/falcon-sensor:7.40.0-1234": "registry.crowdstrike.com/falcon-sensor/release/falcon-sensor",
		"localhost:5000/fcg:1.0":   "localhost:5000/fcg",
		"localhost:5000/fcg":       "localhost:5000/fcg",
		"registry.example.com/fcg": "registry.example.com/fcg",
		"":                         "",
	}
	for in, want := range cases {
		if got := Repository(in); got != want {
			t.Errorf("Repository(%q) = %q, want %q", in, got, want)
		}
	}
}

var errRegistry = errors.New("registry unavailable")
