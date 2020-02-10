package jenkins

import (
	"fmt"
	"time"

	"github.com/bndr/gojenkins"
	"github.com/signalfx/golib/v3/datapoint"
	jc "github.com/signalfx/signalfx-agent/pkg/monitors/jenkins/client"
)

// jobMetrics fetches the first jobs level and calls getAllJobsMetrics to get current and sub-jobs builds dps
func (m *Monitor) jobMetrics(jkClient jc.JenkinsClient) ([]*datapoint.Datapoint, error) {
	jobs, err := jkClient.GoJenkins.GetAllJobs()
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs %v", err)
	}
	return m.getAllJobsMetrics(jobs), nil
}

// getAllJobsMetrics iterates through all the jobs, fetches the build metrics,
// identifies subfolders and recursively identifies sub-jobs and fetches build metrics.
// Only process builds that completed after the initial run of the monitor.
// lastBuildSent map (key:job_name and value:build_completion_time) keeps track of the builds to process
// On the initial run, we set the time for every job to the current time
// this function does not break on error, all errors are logged
func (m *Monitor) getAllJobsMetrics(jobs []*gojenkins.Job) []*datapoint.Datapoint {
	var dps []*datapoint.Datapoint
	for _, job := range jobs {
		// jobs in different folders can use the same name,
		// use the FullName for uniqueness
		jobName := job.Raw.FullName
		// If a job is a folder, recursively call getAllJobsMetrics
		if job.Raw.Class == "com.cloudbees.hudson.plugins.folder.Folder" {
			innerJobs, err := job.GetInnerJobs()
			if err != nil {
				logger.WithError(err).Debugf("Failed to get inner jobs for %s", jobName)
				continue
			}
			innerDPS := m.getAllJobsMetrics(innerJobs)
			dps = append(dps, innerDPS...)
			continue
		} else if job.Raw.Class == "org.jenkinsci.plugins.workflow.job.WorkflowJob" {
			// Pipeline jobs will be handled separately
			continue
		}

		// Check if we've sent any datapoints for this job otherwise
		// update using the current time in ms since epoch
		if _, ok := m.lastBuildSent[jobName]; !ok {
			m.lastBuildSent[jobName] = time.Now().UnixNano() / 1000000
		}
		jobDPS, err := m.getJobDPS(job)
		if err != nil {
			logger.WithError(err).Debugf("error getting Job builds data points for %s", jobName)
			continue
		}
		dps = append(dps, jobDPS...)
	}
	return dps
}

// getJobDPS fetches specific fields for all the builds of a job using the custom
// method GetBuildsFields and return a dps slice.
// The time holder lastBuildSent is updated when a job completion date is larger than the store value.
func (m *Monitor) getJobDPS(job *gojenkins.Job) ([]*datapoint.Datapoint, error) {
	type BuildCustom struct {
		Result    string `json:"result"`
		Building  bool   `json:"building"`
		Duration  int64  `json:"duration"`
		Timestamp int64  `json:"timestamp"`
	}
	var buildsResp struct {
		Builds []BuildCustom `json:"builds"`
	}

	var dps []*datapoint.Datapoint
	jobName := job.Raw.FullName

	err := job.GetBuildsFields([]string{"timestamp", "result", "duration", "building"}, &buildsResp)
	if err != nil {
		return nil, err
	}
	if len(buildsResp.Builds) == 0 {
		return nil, fmt.Errorf("no builds for job %s", jobName)
	}

	var dimensions = make(map[string]string)
	dimensions["job_name"] = jobName

	lastBuild := m.lastBuildSent[jobName]
	for _, jobBuild := range buildsResp.Builds {
		if jobBuild.Building {
			continue
		}
		buildCompletion := jobBuild.Timestamp + jobBuild.Duration
		if buildCompletion > lastBuild {
			if buildCompletion > m.lastBuildSent[jobName] {
				m.lastBuildSent[jobName] = buildCompletion
			}
			dimensions["build_result"] = jobBuild.Result
			dps = append(dps, datapoint.New(jenkinsJobDuration, dimensions, datapoint.NewIntValue(jobBuild.Duration), metricSet[jenkinsJobDuration].Type, time.Time{}))
		}
	}
	return dps, nil
}
