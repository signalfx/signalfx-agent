package hostid

import log "github.com/sirupsen/logrus"

// Dimensions returns a map of host-specific dimensions that are derived from
// the environment.
func Dimensions(sendMachineID bool) map[string]string {
	log.Info("Fetching host id dimensions")
	// Fire off both AWS and GCP requests simultaneously so we delay agent
	// startup as little as possible.
	awsRes := callConcurrent(AWSUniqueID)
	gcpRes := callConcurrent(GoogleComputeID)
	machineIDRes := callConcurrent(MachineID)
	azureRes := callConcurrent(AzureUniqueID)

	dims := make(map[string]string)
	insertNextChanValue(dims, "AWSUniqueId", awsRes)
	insertNextChanValue(dims, "gcp_id", gcpRes)
	if sendMachineID {
		insertNextChanValue(dims, "machine_id", machineIDRes)
	}
	insertNextChanValue(dims, "azure_resource_id", azureRes)

	log.Infof("Using host id dimensions %v", dims)
	return dims
}

func callConcurrent(f func() string) <-chan string {
	res := make(chan string)
	go func() {
		res <- f()
	}()
	return res
}

func insertNextChanValue(m map[string]string, k string, ch <-chan string) {
	select {
	case val := <-ch:
		// Don't insert blank values
		if val != "" {
			m[k] = val
		}
	}
}
