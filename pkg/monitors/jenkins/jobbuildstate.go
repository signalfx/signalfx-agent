package jenkins

import (
	"fmt"
	"time"

	"github.com/signalfx/golib/v3/datapoint"
)

type jobMetrics struct {
	jobTotalTime  int64
	jobBuildCount int64
}

type JobMetricsState struct {
	LastProcessedBuildID int64
	RunningBuildIDs      map[int64]struct{}
	Metrics              map[string]*jobMetrics
}

func (jMState *JobMetricsState) initialJobMetricsState(builds []BuildCustomStruct) *JobMetricsState {
	// On initial run, we don't send any data points, just set LastProcessedBuildID
	// to the largest buildID and cache running builds
	jMState.LastProcessedBuildID = builds[0].ID
	jMState.RunningBuildIDs = make(map[int64]struct{})
	jMState.Metrics = make(map[string]*jobMetrics)
	for _, build := range builds {
		if build.Building {
			jMState.RunningBuildIDs[build.ID] = struct{}{}
		}
	}
	return jMState
}

// updateJobMetricsState takes non-empty slice of builds and set the job build state
func (jMState *JobMetricsState) updateJobMetricsState(jobName string, builds []BuildCustomStruct) {
	// if we have builds in running state, check and update their state first
	if len(jMState.RunningBuildIDs) > 0 {
		jMState.checkRunningBuilds(jobName, builds)
	}
	// check if we fetched new builds since last run,
	// user can delete a build so check for less or equal.
	// TODO: A corner case where a job with same name is recreated
	if builds[0].ID <= jMState.LastProcessedBuildID {
		return
	}
	for _, build := range builds {
		// break when we hit a build ID less or equal than
		// the largest build from the previous run
		if build.ID <= jMState.LastProcessedBuildID {
			break
		}
		if build.Building {
			jMState.RunningBuildIDs[build.ID] = struct{}{}
			continue
		}
		jMState.updateJobMetrics(build)
	}
	jMState.LastProcessedBuildID = builds[0].ID
}

func (jMState *JobMetricsState) updateJobMetrics(build BuildCustomStruct) {
	if metric, ok := jMState.Metrics[build.Result]; ok {
		metric.jobBuildCount++
		metric.jobTotalTime += build.Duration
	} else {
		jMState.Metrics[build.Result] = &jobMetrics{
			jobTotalTime:  build.Duration,
			jobBuildCount: 1,
		}
	}
}

// checkRunningBuilds checks if running builds from previous run are completed
// and update accordingly.
// If one or more builds are not in the 100 builds we fetched, then delete and
// log a warning.
func (jMState *JobMetricsState) checkRunningBuilds(jobName string, builds []BuildCustomStruct) {
	// for quick access, convert the slice into a map
	buildsMaps := make(map[int64]BuildCustomStruct)
	for _, value := range builds {
		buildsMaps[value.ID] = value
	}
	for k := range jMState.RunningBuildIDs {
		if build, ok := buildsMaps[k]; ok {
			if !build.Building {
				jMState.updateJobMetrics(build)
				delete(jMState.RunningBuildIDs, k)
			}
		} else {
			delete(jMState.RunningBuildIDs, k)
			logger.Warnf("A previously cached running build: %d is no longer part of the latest 100 builds fetched for job: %s, dropping this build datapoint", k, jobName)
		}
	}
}

func (jMState *JobMetricsState) buildDPSFromState(jobName, jobType string) []*datapoint.Datapoint {
	var dps []*datapoint.Datapoint
	for result, metrics := range jMState.Metrics {
		dimensions := map[string]string{
			"job_name":     jobName,
			"job_type":     jobType,
			"build_result": result,
		}
		dps = append(dps, datapoint.New(jenkinsJobTotalTime, dimensions, datapoint.NewIntValue(metrics.jobTotalTime), metricSet[jenkinsJobTotalTime].Type, time.Time{}),
			datapoint.New(jenkinsJobBuildCount, dimensions, datapoint.NewIntValue(metrics.jobBuildCount), metricSet[jenkinsJobBuildCount].Type, time.Time{}))
	}
	return dps
}

func (jMState *JobMetricsState) String() string {
	return fmt.Sprintf("LastProcessedBuildID: %v, RunningBuildIDs: %v, Metrics: %v", jMState.LastProcessedBuildID, jMState.RunningBuildIDs, jMState.Metrics)
}

func (jMetrics *jobMetrics) String() string {
	return fmt.Sprintf("jobTotalTime: %v, jobBuildCount: %v", jMetrics.jobTotalTime, jMetrics.jobBuildCount)
}
