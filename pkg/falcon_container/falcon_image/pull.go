package falcon_image

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/crowdstrike/gofalcon/falcon/client/sensor_download"
)

type FalconImage struct {
	tar     bytes.Buffer
	tempDir string
}

func Pull(apiCfg *falcon.ApiConfig, stdout io.Writer) (*FalconImage, error) {
	fi := FalconImage{
		tar: bytes.Buffer{},
	}
	return &fi, fi.pull(apiCfg, stdout)
}

func (fi *FalconImage) pull(apiCfg *falcon.ApiConfig, stdout io.Writer) error {
	client, err := falcon.NewClient(apiCfg)
	if err != nil {
		return err
	}
	filter := fmt.Sprintf("os:\"%s\"", "Container")
	fmt.Fprintf(stdout, "Getting list of images from %s\n", apiCfg.Host())
	sensors, err := client.SensorDownload.GetCombinedSensorInstallersByQuery(
		&sensor_download.GetCombinedSensorInstallersByQueryParams{
			Filter:  &filter,
			Context: apiCfg.Context,
		},
	)
	if err != nil {
		return errors.New(falcon.ErrorExplain(err))
	}

	payload := sensors.GetPayload()
	for _, errorMsg := range payload.Errors {
		if errorMsg != nil && errorMsg.Message != nil {
			fmt.Fprintf(stdout, "Received error from the server: %s\n", *errorMsg.Message)
		}
	}

	if len(payload.Resources) < 1 {
		return errors.New("Could not find Falcon Container Sensor in the download section. Please ensure you have valid subscription.")
	}

	sensor := payload.Resources[0]
	fmt.Fprintf(stdout, "Downloading %s\n", *sensor.Name)

	tarWriter := bufio.NewWriter(&fi.tar)
	_, err = client.SensorDownload.DownloadSensorInstallerByID(
		&sensor_download.DownloadSensorInstallerByIDParams{
			Context: apiCfg.Context,
			ID:      *sensor.Sha256,
		}, tarWriter)
	return err
}

func (fi *FalconImage) Delete() error {
	if fi.tempDir != "" {
		err := os.RemoveAll(fi.tempDir)
		if err != nil {
			return err
		}
		fi.tempDir = ""
	}
	return nil
}
