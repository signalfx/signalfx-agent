package hostid

import log "github.com/sirupsen/logrus"

// Dimensions returns a map of host-specific dimensions that are derived from
// the environment.
func Dimensions(sendMachineID bool, hostname string, useFullyQualifiedHost *bool) map[string]string {
	log.Info("Fetching host id dimensions")
	// Fire off all lookups simultaneously so we delay agent startup as little
	// as possible.

	hostProvider := callConcurrent(func() string {
		if hostname != "" {
			return hostname
		}
		// Using the FQDN needs to default to true but the defaults lib that we
		// use can't distinguish between false and unspecified, so figure out
		// if the user specified it explicitly as false with this logic.
		return getHostname(useFullyQualifiedHost == nil || *useFullyQualifiedHost)
	})
	awsProvider := callConcurrent(AWSUniqueID)
	gcpProvider := callConcurrent(GoogleComputeID)
	machineIDProvider := callConcurrent(MachineID)
	azureProvider := callConcurrent(AzureUniqueID)

	dims := make(map[string]string)
	insertNextChanValue(dims, "host", hostProvider)
	insertNextChanValue(dims, "AWSUniqueId", awsProvider)
	insertNextChanValue(dims, "gcp_id", gcpProvider)
	if sendMachineID {
		insertNextChanValue(dims, "machine_id", machineIDProvider)
	}
	insertNextChanValue(dims, "azure_resource_id", azureProvider)

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
