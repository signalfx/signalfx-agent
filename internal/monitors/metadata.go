package monitors

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
)

const monitorMetadataFile = "metadata.yaml"

// MetricMetadata contains a metric's metadata.
type MetricMetadata struct {
	Name        string  `json:"name"`
	Alias       string  `json:"alias,omitempty"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Group       *string `json:"group"`
	Included    bool    `json:"included" default:"false"`
}

// PropMetadata contains a property's metadata.
type PropMetadata struct {
	Name        string `json:"name"`
	Dimension   string `json:"dimension"`
	Description string `json:"description"`
}

// GroupMetadata contains a group's metadata.
type GroupMetadata struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// MonitorMetadata contains a monitor's metadata.
type MonitorMetadata struct {
	MonitorType string           `json:"monitorType" yaml:"monitorType"`
	SendAll     bool             `json:"sendAll" yaml:"sendAll"`
	Dimensions  []DimMetadata    `json:"dimensions"`
	Doc         string           `json:"doc"`
	Groups      []GroupMetadata  `json:"groups"`
	Metrics     []MetricMetadata `json:"metrics"`
	Properties  []PropMetadata   `json:"properties"`
}

// PackageMetadata describes a package directory that may have one or more monitors.
type PackageMetadata struct {
	Monitors []MonitorMetadata
	Package  string `yaml:"-"`
	Path     string `json:"-" yaml:"-"`
}

// DimMetadata contains a dimension's metadata.
type DimMetadata struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CollectMetadata loads metadata for all monitors located in root as well as any subdirectories.
func CollectMetadata(root string) ([]PackageMetadata, error) {
	var packages []PackageMetadata

	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || info.Name() != monitorMetadataFile {
			return nil
		}

		var monitorDocs []MonitorMetadata

		if bytes, err := ioutil.ReadFile(path); err != nil {
			return errors.Errorf("unable to read metadata file %s: %s", path, err)
		} else if err := yaml.UnmarshalStrict(bytes, &monitorDocs); err != nil {
			return errors.Wrapf(err, "unable to unmarshal file %s", path)
		}

		packages = append(packages, PackageMetadata{
			Monitors: monitorDocs,
			Package:  filepath.Dir(path),
			Path:     path,
		})

		return nil
	}); err != nil {
		return nil, err
	}

	return packages, nil
}
