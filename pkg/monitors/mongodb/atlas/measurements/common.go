package measurements

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
)

// Common configuration for getting metric measurements from Atlas API endpoints.
type config struct {
	projectID   string
	ctx         context.Context
	client      *mongodbatlas.Client
	enableCache bool
}

// nextPage gets the next page for pagination request.
func nextPage(resp *mongodbatlas.Response) (bool, int) {
	if len(resp.Links) == 0 {
		return false, -1
	}

	currentPage, err := resp.CurrentPage()

	if resp.IsLastPage() || err != nil {
		log.WithError(err).Error(fmt.Sprintf("failed to get the next page, response: %+v", resp))
		return false, -1
	}

	return true, currentPage + 1
}

// optionPT1M sets the granularity and period options for getting measurement datapoints to PT1M. This corresponds to
// to 1 minute granularity over a 1 minute period. It is the highest resolution allowed by Atlas.
func optionPT1M(pageNum int) *mongodbatlas.ProcessMeasurementListOptions {
	return &mongodbatlas.ProcessMeasurementListOptions{
		ListOptions: &mongodbatlas.ListOptions{PageNum: pageNum},
		Granularity: "PT1M", // granularity of 1 minute
		Period:      "PT1M", // a period of 1 minute
	}
}

func formatError(err error, resp *mongodbatlas.Response) (string, error) {
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
