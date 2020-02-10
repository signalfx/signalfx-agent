package jenkins

import (
	"fmt"
	"sort"

	"github.com/bndr/gojenkins"
	"github.com/signalfx/golib/v3/datapoint"
	jc "github.com/signalfx/signalfx-agent/pkg/monitors/jenkins/client"
)

func (m *Monitor) jobMetrics(jkClient jc.JenkinsClient) ([]*datapoint.Datapoint, error) {
	// GetAllJobs returns the first level jobs
	firstLevelJobs, err := jkClient.GoJenkins.GetAllJobs()
	if err != nil {
		return nil, fmt.Errorf("failed to get firstLevelJobs %v", err)
	}

	allJobs := getAllJobs(firstLevelJobs)
	var dps []*datapoint.Datapoint
	// jobsMap is used for jobs state cleanup
	jobsMap := make(map[string]struct{})
	for _, job := range allJobs {
		// jobs in different folders can use the same name,
		// use the FullName for uniqueness
		jobName := job.Raw.FullName
		jobsMap[jobName] = struct{}{}

		builds, err := getBuilds(job)
		if err != nil {
			logger.WithError(err).Debugf("Failed to get builds for job %s", jobName)
			continue
		}

		dps = append(dps, m.getJobDPS(job, builds)...)
	}
	m.jobsBuildsStateCleanup(jobsMap)
	logger.Debugf("jobs build state: %v", m.JobsMetricsState)
	return dps, nil
}

func (m *Monitor) getJobDPS(job *gojenkins.Job, builds []BuildCustomStruct) []*datapoint.Datapoint {
	jobName := job.Raw.FullName
	jMState, ok := m.JobsMetricsState[jobName]
	if !ok {
		jMState = &JobMetricsState{}
		m.JobsMetricsState[jobName] = jMState.initialJobMetricsState(builds)
		return nil
	}
	jMState.updateJobMetricsState(jobName, builds)
	return jMState.buildDPSFromState(jobName, job.Raw.Class)
}

// jobsBuildsStateCleanup remove no longer existing jobs from JobsMetricsState
func (m *Monitor) jobsBuildsStateCleanup(jobsMap map[string]struct{}) {
	for jobName := range m.JobsMetricsState {
		if _, ok := jobsMap[jobName]; !ok {
			delete(m.JobsMetricsState, jobName)
			logger.Debugf("Deleted a staled job: %s in cache", jobName)
		}
	}
}

// getAllJobs identifies folders and recursively get their inner jobs.
// jobs with type folder are skipped
func getAllJobs(jobs []*gojenkins.Job) []*gojenkins.Job {
	var allJobs []*gojenkins.Job
	for _, job := range jobs {
		if job.Raw.Class == "com.cloudbees.hudson.plugins.folder.Folder" {
			innerJobs, err := job.GetInnerJobs()
			if err != nil {
				logger.WithError(err).Debugf("Failed to get inner jobs for %s", job.Raw.FullName)
				continue
			}
			allJobs = append(allJobs, getAllJobs(innerJobs)...)
		} else {
			allJobs = append(allJobs, job)
		}
	}
	return allJobs
}

type BuildCustomStruct struct {
	ID       int64  `json:"id,string"`
	Result   string `json:"result"`
	Building bool   `json:"building"`
	Duration int64  `json:"duration"`
}

// ByBuildID implements sort.Interface based on the
// BuildCustomStruct.ID field in descending order
type ByBuildID []BuildCustomStruct

func (a ByBuildID) Len() int           { return len(a) }
func (a ByBuildID) Less(i, j int) bool { return a[i].ID > a[j].ID }
func (a ByBuildID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

// getBuilds fetches the last 100 builds for a job,
// returns a descending sorted list by ID or an error if no builds are found
func getBuilds(job *gojenkins.Job) ([]BuildCustomStruct, error) {
	var buildsResp struct {
		Builds []BuildCustomStruct `json:"builds"`
	}
	err := job.GetBuildsFields([]string{"id", "result", "duration", "building"}, &buildsResp)
	if err != nil {
		return nil, err
	}

	if len(buildsResp.Builds) == 0 {
		return nil, fmt.Errorf("no builds found")
	}

	sort.Sort(ByBuildID(buildsResp.Builds))
	return buildsResp.Builds, nil
}
