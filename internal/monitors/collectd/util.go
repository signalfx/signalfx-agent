package collectd

import (
	"os"
	"path/filepath"

	"github.com/signalfx/signalfx-agent/internal/core/common/constants"
)

// MakePath takes file path components below the BundleDirectory/plugins/collectd path and
// returns an os appropriate file path.  The environment variable SIGNALFX_BUNDLE_DIR is
// used as the root of the path
func MakePath(components ...string) string {
	components = append([]string{os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd"}, components...)
	return filepath.Join(components...)
}
