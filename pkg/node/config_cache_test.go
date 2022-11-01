package node

import (
	"context"
	"os"
	"testing"

	"github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
)

var falconNode = v1alpha1.FalconNodeSensor{}
var falconCID = "1234567890ABCDEF1234567890ABCDEF-12"
var falconImage = "testMyImage"
var config = ConfigCache{cid: falconCID, imageUri: falconImage, nodesensor: &falconNode}

func TestCID(t *testing.T) {
	if config.CID() != falconCID {
		t.Errorf("CID() = %s, want %s", config.CID(), falconCID)
	}
}

func TestUsingCrowdStrikeRegistry(t *testing.T) {
	if config.UsingCrowdStrikeRegistry() != true {
		t.Errorf("UsingCrowdStrikeRegistry() = %t, want %t", config.UsingCrowdStrikeRegistry(), true)
	}
}

func TestGetImageURI(t *testing.T) {
	want, err := config.GetImageURI(context.Background())
	if err != nil {
		t.Errorf("GetImageURI() error: %v", err)
	}
	if want != falconImage {
		t.Errorf("GetImageURI() = %s, want %s", want, falconImage)
	}
}

func TestGetPullToken(t *testing.T) {
	want, err := config.GetPullToken(context.Background())
	if err != nil {
		if err.Error() != "Missing falcon_api configuration" {
			t.Errorf("GetPullToken() error: %v", err)
		}
	}
	if len(want) != 0 {
		t.Errorf("GetPullToken() = %s, want %s", want, "not empty")
	}
}

func TestSensorEnvVars(t *testing.T) {
	want := make(map[string]string)
	want["FALCONCTL_OPT_CID"] = falconCID

	got := config.SensorEnvVars()
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
}

func TestConfigCacheTest(t *testing.T) {
	want := config

	newCache := ConfigCacheTest(falconCID, falconImage, &falconNode)
	if want != *newCache {
		t.Errorf("ConfigCacheTest() = %v, want %v", newCache, want)
	}
}

func TestGetFalconImage(t *testing.T) {
	falconNode.Spec.FalconAPI = nil
	_, err := getFalconImage(context.Background(), &falconNode)
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
	got, err := getFalconImage(context.Background(), &falconNode)
	if err != nil {
		t.Errorf("getFalconImage() error: %v", err)
	}
	if want != got {
		t.Errorf("getFalconImage() = %s, want %s", got, want)
	}

	want = "TestImageOverride"
	falconNode.Spec.Node.ImageOverride = want

	got, err = getFalconImage(context.Background(), &falconNode)
	if err != nil {
		t.Errorf("getFalconImage() error: %v", err)
	}
	if want != got {
		t.Errorf("getFalconImage() = %s, want %s", got, want)
	}
}
