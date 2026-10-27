package node

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	internalErrors "github.com/crowdstrike/falcon-operator/internal/errors"
	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

var falconNode = falconv1alpha1.FalconNodeSensor{}
var falconCID = "1234567890ABCDEF1234567890ABCDEF-12"
var falconImage = "testMyImage"
var falconApiConfig = falcon.ApiConfig{}
var config = ConfigCache{
	cid:             falconCID,
	imageUri:        falconImage,
	nodesensor:      &falconNode,
	falconApiConfig: nil,
}

func TestCID(t *testing.T) {
	got := config.cid
	want := falconCID
	if got != want {
		t.Errorf("CID() = %s, want %s", got, want)
	}
}

func TestUsingCrowdStrikeRegistry(t *testing.T) {
	got := config.UsingCrowdStrikeRegistry()
	want := true
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("UsingCrowdStrikeRegistry() mismatch (-want +got): %s", diff)
	}

	// Test with imageOverride
	config.nodesensor.Spec.Node.Image = falconImage
	got = config.UsingCrowdStrikeRegistry()
	want = false
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("UsingCrowdStrikeRegistry() mismatch (-want +got): %s", diff)
	}

	// Reset imageOverride
	config.nodesensor.Spec.Node.Image = ""
}

func TestGetImageURI(t *testing.T) {
	var logger logr.Logger

	config.imageUri = ""
	got, err := config.GetImageURI(context.Background(), logger)
	if err != nil {
		if err != ErrFalconAPINotConfigured {
			t.Errorf("GetImageURI() error: %v", err)
		}
	}

	want := ""
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("GetImageURI() mismatch (-want +got): %s", diff)
	}

	config.imageUri = falconImage
	got, err = config.GetImageURI(context.Background(), logger)
	if err != nil {
		t.Errorf("GetImageURI() error: %v", err)
	}
	want = falconImage
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("GetImageURI() mismatch (-want +got): %s", diff)
	}

}

func TestGetPullToken(t *testing.T) {
	testConfig := config
	got, err := testConfig.GetPullToken(context.Background())
	if err != nil {
		if err != ErrFalconAPINotConfigured {
			t.Errorf("GetPullToken() error: %v", err)
		}
	}
	if len(got) != 0 {
		t.Errorf("GetPullToken() = %s, want %s", got, "not empty")
	}

	var noCID *string
	testConfig.nodesensor.Spec.FalconAPI = newTestFalconAPI(noCID)
	testConfig.falconApiConfig = newTestApiConfig()
	got, err = testConfig.GetPullToken(context.Background())
	if err != nil {
		if strings.Contains(err.Error(), "401 Unauthorized") {
			got = []byte("testToken")
		}
	}
	if len(got) == 0 {
		t.Errorf("GetPullToken() = %s, want %v", "empty", got)
	}
}

func TestSensorEnvVars(t *testing.T) {
	testConfig := config
	want := make(map[string]string)
	want["FALCONCTL_OPT_CID"] = falconCID
	want["OPERATOR_MANIFEST"] = ""
	want["OPERATOR_VERSION"] = ""

	got := testConfig.SensorEnvVars()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("SensorEnvVars() mismatch (-want +got): %s", diff)
	}

	// Backend field is deprecated; setting it must not affect the sensor env vars
	testConfig.nodesensor.Spec.Node.Backend = "kernel"
	got = testConfig.SensorEnvVars()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("SensorEnvVars() mismatch after setting deprecated Backend field (-want +got): %s", diff)
	}

	cloudRegion := "us-2"
	testConfig.nodesensor.Spec.Falcon.Cloud = cloudRegion
	want["FALCONCTL_OPT_CLOUD"] = cloudRegion
	got = testConfig.SensorEnvVars()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("SensorEnvVars() mismatch (-want +got): %s", diff)
	}
}

func TestNewConfigCache(t *testing.T) {
	want := ConfigCache{cid: falconCID, nodesensor: &falconNode}

	falconNode.Spec.FalconAPI = nil
	falconNode.Spec.Falcon.CID = &falconCID

	newCache, err := NewConfigCache(context.Background(), &falconNode)
	if err != nil {
		t.Errorf("NewConfigCache() error: %v", err)
	}

	if want != *newCache {
		t.Errorf("NewConfigCache() = %v, want %v", newCache, want)
	}

	config.nodesensor.Spec.FalconAPI = newTestFalconAPI(&falconCID)
	newCache, err = NewConfigCache(context.Background(), &falconNode)
	if err != nil {
		t.Errorf("NewConfigCache() error: %v", err)
	}

	if config.cid != newCache.cid {
		t.Errorf("NewConfigCache() = %v, want %v", newCache, want)
	}
}

func TestConfigCacheTest(t *testing.T) {
	testConfig := config
	testConfig.falconApiConfig = &falconApiConfig
	want := testConfig

	newCache := ConfigCacheTest(falconCID, falconImage, &falconNode, &falconApiConfig)
	if want != *newCache {
		t.Errorf("ConfigCacheTest() = %v, want %v", newCache, want)
	}
}

func TestGetFalconImage(t *testing.T) {
	testConfig := config
	falconNode.Spec.FalconAPI = newTestFalconAPI(&falconCID)
	testConfig.falconApiConfig = newTestApiConfig()

	testVersion := "testVersion"
	falconNode.Spec.Node.Version = &testVersion
	got, err := testConfig.getFalconImage(context.Background(), &falconNode)
	if err != nil {
		if strings.Contains(err.Error(), "401 Unauthorized") {
			got = fmt.Sprintf("%s:%s", "TestImageEnv", *falconNode.Spec.Node.Version)
		}
	}

	if len(got) == 0 {
		t.Errorf("getFalconImage() = %s, want %s", "empty", got)
	}

	falconNode.Spec.FalconAPI = nil
	testConfig.falconApiConfig = nil
	_, err = testConfig.getFalconImage(context.Background(), &falconNode)
	if err != nil {
		if err != ErrFalconAPINotConfigured {
			t.Errorf("getFalconImage() error: %v", err)
		}
	}

	err = os.Setenv("RELATED_IMAGE_NODE_SENSOR", "TestImageEnv")
	if err != nil {
		t.Errorf("getFalconImage() error: %v", err)
	}

	want := "TestImageEnv"
	got, err = testConfig.getFalconImage(context.Background(), &falconNode)
	if err != nil {
		t.Errorf("getFalconImage() error: %v", err)
	}
	if want != got {
		t.Errorf("getFalconImage() = %s, want %s", got, want)
	}

	want = "TestImageOverride"
	falconNode.Spec.Node.Image = want

	got, err = testConfig.getFalconImage(context.Background(), &falconNode)
	if err != nil {
		t.Errorf("getFalconImage() error: %v", err)
	}
	if want != got {
		t.Errorf("getFalconImage() = %s, want %s", got, want)
	}
}

// TestNewConfigCache_EmptyFalconAPI verifies that a non-nil but uncredentialed FalconAPI
// does not populate falconApiConfig, preserving the isolated-environment RELATED_IMAGE path.
func TestNewConfigCache_EmptyFalconAPI_DoesNotSetApiConfig(t *testing.T) {
	node := &falconv1alpha1.FalconNodeSensor{}
	node.Spec.FalconAPI = &falconv1alpha1.FalconAPI{}
	node.Spec.Falcon.CID = &falconCID

	cache, err := NewConfigCache(context.Background(), node)
	if err != nil {
		t.Fatalf("NewConfigCache() unexpected error: %v", err)
	}
	if cache.falconApiConfig != nil {
		t.Errorf("NewConfigCache() falconApiConfig = %v, want nil for uncredentialed FalconAPI", cache.falconApiConfig)
	}
}

// TestNewConfigCache_NoCredentialsNoCID_ReturnsError verifies that a FalconAPI without
// credentials and no CID returns an error instead of panicking on a nil API config.
func TestNewConfigCache_NoCredentialsNoCID_ReturnsError(t *testing.T) {
	node := &falconv1alpha1.FalconNodeSensor{}
	node.Spec.FalconAPI = &falconv1alpha1.FalconAPI{ClientId: "client-id-only"}

	_, err := NewConfigCache(context.Background(), node)
	if !errors.Is(err, internalErrors.ErrMissingCIDWithoutFalconAPI) {
		t.Errorf("NewConfigCache() error = %v, want %v", err, internalErrors.ErrMissingCIDWithoutFalconAPI)
	}
}

// TestNewConfigCache_UncredentialedFalconAPICID_UsesCID verifies that falcon_api.cid is used
// even when FalconAPI has no credentials.
func TestNewConfigCache_UncredentialedFalconAPICID_UsesCID(t *testing.T) {
	node := &falconv1alpha1.FalconNodeSensor{}
	node.Spec.FalconAPI = &falconv1alpha1.FalconAPI{CID: &falconCID}

	cache, err := NewConfigCache(context.Background(), node)
	if err != nil {
		t.Fatalf("NewConfigCache() unexpected error: %v", err)
	}
	if cache.cid != falconCID {
		t.Errorf("NewConfigCache() cid = %q, want %q", cache.cid, falconCID)
	}
	if cache.falconApiConfig != nil {
		t.Errorf("NewConfigCache() falconApiConfig = %v, want nil for uncredentialed FalconAPI", cache.falconApiConfig)
	}
}

// TestGetFalconImage_EmptyFalconAPI_UsesRelatedImage verifies that when FalconAPI is set
// but has no credentials and RELATED_IMAGE_NODE_SENSOR is present, the bundled image is used.
func TestGetFalconImage_EmptyFalconAPI_UsesRelatedImage(t *testing.T) {
	if err := os.Setenv("RELATED_IMAGE_NODE_SENSOR", "bundled-node-image:latest"); err != nil {
		t.Fatalf("Setenv error: %v", err)
	}
	t.Cleanup(func() { os.Unsetenv("RELATED_IMAGE_NODE_SENSOR") })

	node := &falconv1alpha1.FalconNodeSensor{}
	node.Spec.FalconAPI = &falconv1alpha1.FalconAPI{}

	testCache := &ConfigCache{nodesensor: node, falconApiConfig: nil}

	got, err := testCache.getFalconImage(context.Background(), node)
	if err != nil {
		t.Fatalf("getFalconImage() unexpected error: %v", err)
	}
	if got != "bundled-node-image:latest" {
		t.Errorf("getFalconImage() = %q, want %q", got, "bundled-node-image:latest")
	}
}

func TestVersionLock_WithAutoUpdateDisabled(t *testing.T) {
	admission := &falconv1alpha1.FalconNodeSensor{}
	admission.Status.Sensor = stringPointer("some sensor")
	admission.Spec.Node.Advanced.AutoUpdate = stringPointer(falconv1alpha1.Off)
	assert.True(t, versionLock(admission))
}

func TestVersionLock_WithForcedAutoUpdate(t *testing.T) {
	admission := &falconv1alpha1.FalconNodeSensor{}
	admission.Status.Sensor = stringPointer("some sensor")
	admission.Spec.Node.Advanced.AutoUpdate = stringPointer(falconv1alpha1.Force)
	assert.False(t, versionLock(admission))
}

func TestVersionLock_WithNormalAutoUpdate(t *testing.T) {
	admission := &falconv1alpha1.FalconNodeSensor{}
	admission.Status.Sensor = stringPointer("some sensor")
	admission.Spec.Node.Advanced.AutoUpdate = stringPointer(falconv1alpha1.Normal)
	assert.False(t, versionLock(admission))
}

func TestVersionLock_WithBlankUpdatePolicy(t *testing.T) {
	sensor := &falconv1alpha1.FalconNodeSensor{}
	sensor.Status.Sensor = stringPointer("some sensor")
	sensor.Spec.Node.Advanced.UpdatePolicy = stringPointer("")
	assert.True(t, versionLock(sensor))
}

func TestVersionLock_WithDifferentVersion(t *testing.T) {
	sensor := &falconv1alpha1.FalconNodeSensor{}
	sensor.Status.Sensor = stringPointer("some sensor")
	sensor.Spec.Node.Version = stringPointer("different version")
	assert.False(t, versionLock(sensor))
}

func TestVersionLock_WithLatestVersion(t *testing.T) {
	sensor := &falconv1alpha1.FalconNodeSensor{}
	sensor.Status.Sensor = stringPointer("some sensor")
	assert.True(t, versionLock(sensor))
}

func TestVersionLock_WithNoCurrentSensor(t *testing.T) {
	sensor := &falconv1alpha1.FalconNodeSensor{}
	assert.False(t, versionLock(sensor))
}

func TestVersionLock_WithSameVersion(t *testing.T) {
	sensor := &falconv1alpha1.FalconNodeSensor{}
	sensor.Status.Sensor = stringPointer("some sensor")
	sensor.Spec.Node.Version = sensor.Status.Sensor
	assert.True(t, versionLock(sensor))
}

func TestVersionLock_WithUpdatePolicy(t *testing.T) {
	sensor := &falconv1alpha1.FalconNodeSensor{}
	sensor.Status.Sensor = stringPointer("some sensor")
	sensor.Spec.Node.Advanced.UpdatePolicy = stringPointer("some policy")
	assert.False(t, versionLock(sensor))
}

func newTestFalconAPI(cid *string) *falconv1alpha1.FalconAPI {
	return &falconv1alpha1.FalconAPI{
		ClientId:     "testID",
		ClientSecret: "testSecret",
		CloudRegion:  "testRegion",
		CID:          cid,
		HostOverride: strings.TrimSpace(os.Getenv("FALCON_API_HOST")),
	}
}

func newTestApiConfig() *falcon.ApiConfig {
	return &falcon.ApiConfig{
		Cloud:        falcon.CloudUs1,
		ClientId:     "testID",
		ClientSecret: "testSecret",
	}
}

func stringPointer(s string) *string {
	return &s
}
