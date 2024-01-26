package version

import (
	"fmt"
	"runtime"
)

var (
	Version   = "" // version will be replaced while building the binary using ldflags
	GoVersion = fmt.Sprintf("%s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
)

// Get() returns the operator version
func Get() string {
	return Version
}
