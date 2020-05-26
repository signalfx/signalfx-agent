package measurements

import (
	"fmt"

	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	log "github.com/sirupsen/logrus"
)

// Process is the MongoDB Process identified by the host and port on which the Process is running.
type Process struct {
	Host string // The name of the host in which the MongoDB Process is running
	Port int    // The port number on which the MongoDB Process is running
}

// nextPage gets the next page for pagination request.
func nextPage(resp *mongodbatlas.Response) (bool, int) {
	if resp == nil || len(resp.Links) == 0 || resp.IsLastPage() {
		return false, -1
	}

	currentPage, err := resp.CurrentPage()

	if err != nil {
		log.WithError(err).Error("failed to get the next page")
		return false, -1
	}

	return true, currentPage + 1
}

// optionPT1M sets the granularity and period options for getting measurement datapoints to PT1M. This corresponds to
// to 1 minute granularity the highest resolution supported by Atlas.
func optionPT1M(pageNum int) *mongodbatlas.ProcessMeasurementListOptions {
	return &mongodbatlas.ProcessMeasurementListOptions{
		ListOptions: &mongodbatlas.ListOptions{PageNum: pageNum},
		Granularity: "PT1M", // The interval between measurement datapoints set to 1 minute.
		Period:      "PT2M", // The period to get measurement datapoints set to the past 2 minutes. Setting the period to 1 minute resulted in a lot of measurement datapoints with empty values.
	}
}

func errorMsg(err error, resp *mongodbatlas.Response) (string, error) {
	if err != nil {
		return "request for getting %s failed (Atlas project: %s, host: %s, port: %d)", err
	}

	if resp == nil {
		return "response for getting %s returned empty (Atlas project: %s, host: %s, port: %d)", fmt.Errorf("empty response")
	}

	if err := mongodbatlas.CheckResponse(resp.Response); err != nil {
		return "response for getting %s returned error (Atlas project: %s, host: %s, port: %d)", err
	}

	return "", nil
}
