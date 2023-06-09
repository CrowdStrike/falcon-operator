package node

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
)

var falconNode = falconv1alpha1.FalconNodeSensor{}
var falconCID = "1234567890ABCDEF1234567890ABCDEF-12"
var falconImage = "testMyImage"
var config = ConfigCache{cid: falconCID, imageUri: falconImage, nodesensor: &falconNode}

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
		if err.Error() != "Missing falcon_api configuration" {
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
	got, err := config.GetPullToken(context.Background())
	if err != nil {
		if err.Error() != "Missing falcon_api configuration" {
			t.Errorf("GetPullToken() error: %v", err)
		}
	}
	if len(got) != 0 {
		t.Errorf("GetPullToken() = %s, want %s", got, "not empty")
	}

	config.nodesensor.Spec.FalconAPI = &falconv1alpha1.FalconAPI{
		ClientId:     "testID",
		ClientSecret: "testSecret",
		CloudRegion:  "testRegion",
	}
	got, err = config.GetPullToken(context.Background())
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
	want := make(map[string]string)
	want["FALCONCTL_OPT_CID"] = falconCID

	got := config.SensorEnvVars()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("SensorEnvVars() mismatch (-want +got): %s", diff)
	}

	config.nodesensor.Spec.Node.Backend = "kernel"
	want["FALCONCTL_OPT_BACKEND"] = "kernel"
	got = config.SensorEnvVars()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("SensorEnvVars() mismatch (-want +got): %s", diff)
	}
}

func TestNewConfigCache(t *testing.T) {
	want := ConfigCache{cid: falconCID, nodesensor: &falconNode}
	var logger logr.Logger

	falconNode.Spec.FalconAPI = nil
	falconNode.Spec.Falcon.CID = &falconCID

	newCache, err := NewConfigCache(context.Background(), logger, &falconNode)
	if err != nil {
		t.Errorf("NewConfigCache() error: %v", err)
	}

	if want != *newCache {
		t.Errorf("NewConfigCache() = %v, want %v", newCache, want)
	}

	config.nodesensor.Spec.FalconAPI = &falconv1alpha1.FalconAPI{
		ClientId:     "testID",
		ClientSecret: "testSecret",
		CloudRegion:  "testRegion",
		CID:          &falconCID,
	}
	newCache, err = NewConfigCache(context.Background(), logger, &falconNode)
	if err != nil {
		t.Errorf("NewConfigCache() error: %v", err)
	}

	if config.cid != newCache.cid {
		t.Errorf("NewConfigCache() = %v, want %v", newCache, want)
	}
}

func TestConfigCacheTest(t *testing.T) {
	want := config

	newCache := ConfigCacheTest(falconCID, falconImage, &falconNode)
	if want != *newCache {
		t.Errorf("ConfigCacheTest() = %v, want %v", newCache, want)
	}
}

func TestGetFalconImage(t *testing.T) {
	falconNode.Spec.FalconAPI = &falconv1alpha1.FalconAPI{
		ClientId:     "testID",
		ClientSecret: "testSecret",
		CloudRegion:  "testRegion",
		CID:          &falconCID,
	}

	testVersion := "testVersion"
	falconNode.Spec.Node.Version = &testVersion
	got, err := getFalconImage(context.Background(), &falconNode)
	if err != nil {
		if strings.Contains(err.Error(), "401 Unauthorized") {
			got = fmt.Sprintf("%s:%s", "TestImageEnv", *falconNode.Spec.Node.Version)
		}
	}

	if len(got) == 0 {
		t.Errorf("getFalconImage() = %s, want %s", "empty", got)
	}

	falconNode.Spec.FalconAPI = nil
	_, err = getFalconImage(context.Background(), &falconNode)
	if err != nil {
		if err.Error() != "Missing falcon_api configuration" {
			t.Errorf("getFalconImage() error: %v", err)
		}
	}

	err = os.Setenv("RELATED_IMAGE_NODE_SENSOR", "TestImageEnv")
	if err != nil {
		t.Errorf("getFalconImage() error: %v", err)
	}

	want := "TestImageEnv"
	got, err = getFalconImage(context.Background(), &falconNode)
	if err != nil {
		t.Errorf("getFalconImage() error: %v", err)
	}
	if want != got {
		t.Errorf("getFalconImage() = %s, want %s", got, want)
	}

	want = "TestImageOverride"
	falconNode.Spec.Node.Image = want

	got, err = getFalconImage(context.Background(), &falconNode)
	if err != nil {
		t.Errorf("getFalconImage() error: %v", err)
	}
	if want != got {
		t.Errorf("getFalconImage() = %s, want %s", got, want)
	}
}
