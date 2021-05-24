package falcon_image

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/containers/image/v5/docker/archive"
	"github.com/containers/image/v5/types"
)

func (fi *FalconImage) ImageReference() (types.ImageReference, error) {
	imageFile, err := fi.imageFilePath()
	if err != nil {
		return nil, err
	}
	return archive.NewReference(imageFile, nil)
}

func (fi *FalconImage) imageFilePath() (string, error) {
	// Unfortunately we have to make a tempfile as docker/archive interface does no allow us
	// to pass in an io.Reader

	var filename string
	if fi.tempDir == "" {
		var err error
		systemTempDir := os.TempDir()
		fi.tempDir, err = ioutil.TempDir(systemTempDir, "crowdstrike")
		if err != nil {
			return "", err
		}
		filename = filepath.Join(fi.tempDir, "falcon-image.tar.bz2")
		f, err := os.Create(filename)
		if err != nil {
			return "", err
		}
		defer f.Close()
		_, err = io.Copy(f, &fi.tar)
		if err != nil {
			return "", err
		}
	} else {
		filename = filepath.Join(fi.tempDir, "falcon-image.tar.bz2")
	}
	return filename, nil

}
