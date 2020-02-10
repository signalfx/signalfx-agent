package jenkins

import (
	"sort"
	"testing"
	"time"

	"github.com/bndr/gojenkins"
	"github.com/signalfx/golib/v3/datapoint"
	"github.com/stretchr/testify/assert"
)

const (
	Success = "SUCCESS"
	Failed  = "FAILED"
	Aborted = "ABORTED"
	JobType = "hudson.model.FreeStyleProject"
)

var (
	job1FullName = "folder/job1"
	job1         = &gojenkins.Job{
		Raw: &gojenkins.JobResponse{
			Class:    JobType,
			FullName: job1FullName,
		},
	}
	job2FullName = "job2"
	job2         = &gojenkins.Job{
		Raw: &gojenkins.JobResponse{
			Class:    JobType,
			FullName: job2FullName,
		},
	}

	monitor = &Monitor{
		JobsMetricsState: make(map[string]*JobMetricsState),
	}

	initialBuilds = []BuildCustomStruct{
		{
			ID:       1,
			Result:   Success,
			Building: false,
			Duration: 8991,
		},
		{
			ID:       2,
			Result:   Success,
			Building: false,
			Duration: 9020,
		},
		{
			ID:       3,
			Result:   "",
			Building: true,
			Duration: 0,
		},
	}

	initialMetricStateResult = &JobMetricsState{
		LastProcessedBuildID: 3,
		RunningBuildIDs:      map[int64]struct{}{3: {}},
		Metrics:              make(map[string]*jobMetrics),
	}

	secondBuilds = append(initialBuilds[0:2], []BuildCustomStruct{
		{
			ID:       3,
			Result:   Failed,
			Building: false,
			Duration: 89,
		},
		{
			ID:       4,
			Result:   Success,
			Building: false,
			Duration: 8991,
		},
		{
			ID:       5,
			Result:   "",
			Building: true,
			Duration: 0,
		},
	}...)

	secondMetricStateResult = &JobMetricsState{
		LastProcessedBuildID: 5,
		RunningBuildIDs:      map[int64]struct{}{5: {}},
		Metrics: map[string]*jobMetrics{Failed: {
			jobTotalTime:  89,
			jobBuildCount: 1,
		}, Success: {
			jobTotalTime:  8991,
			jobBuildCount: 1,
		}},
	}

	thirdBuilds = append(secondBuilds[0:4], []BuildCustomStruct{
		{
			ID:       6,
			Result:   Failed,
			Building: false,
			Duration: 21,
		},
		{
			ID:       7,
			Result:   Success,
			Building: false,
			Duration: 1000,
		},
		{
			ID:       8,
			Result:   Aborted,
			Building: false,
			Duration: 8,
		},
	}...)

	thirdMetricStateResult = &JobMetricsState{
		LastProcessedBuildID: 8,
		RunningBuildIDs:      map[int64]struct{}{},
		Metrics: map[string]*jobMetrics{Failed: {
			jobTotalTime:  110,
			jobBuildCount: 2,
		}, Success: {
			jobTotalTime:  9991,
			jobBuildCount: 2,
		}, Aborted: {
			jobTotalTime:  8,
			jobBuildCount: 1,
		}},
	}

	expectedDPS2 = []*datapoint.Datapoint{
		{
			Metric:     jenkinsJobTotalTime,
			Dimensions: map[string]string{"build_result": Failed, "job_name": job1FullName, "job_type": JobType},
			Meta:       map[interface{}]interface{}{},
			Value:      datapoint.NewIntValue(89),
			MetricType: metricSet[jenkinsJobTotalTime].Type,
			Timestamp:  time.Time{},
		},
		{
			Metric:     jenkinsJobBuildCount,
			Dimensions: map[string]string{"build_result": Failed, "job_name": job1FullName, "job_type": JobType},
			Meta:       map[interface{}]interface{}{},
			Value:      datapoint.NewIntValue(1),
			MetricType: metricSet[jenkinsJobBuildCount].Type,
			Timestamp:  time.Time{},
		},
		{
			Metric:     jenkinsJobTotalTime,
			Dimensions: map[string]string{"build_result": Success, "job_name": job1FullName, "job_type": JobType},
			Meta:       map[interface{}]interface{}{},
			Value:      datapoint.NewIntValue(8991),
			MetricType: metricSet[jenkinsJobTotalTime].Type,
			Timestamp:  time.Time{},
		},
		{
			Metric:     jenkinsJobBuildCount,
			Dimensions: map[string]string{"build_result": Success, "job_name": job1FullName, "job_type": JobType},
			Meta:       map[interface{}]interface{}{},
			Value:      datapoint.NewIntValue(1),
			MetricType: metricSet[jenkinsJobBuildCount].Type,
			Timestamp:  time.Time{},
		},
	}

	expectedDPS3 = []*datapoint.Datapoint{
		{
			Metric:     jenkinsJobTotalTime,
			Dimensions: map[string]string{"build_result": Failed, "job_name": job1FullName, "job_type": JobType},
			Meta:       map[interface{}]interface{}{},
			Value:      datapoint.NewIntValue(110),
			MetricType: metricSet[jenkinsJobTotalTime].Type,
			Timestamp:  time.Time{},
		},
		{
			Metric:     jenkinsJobBuildCount,
			Dimensions: map[string]string{"build_result": Failed, "job_name": job1FullName, "job_type": JobType},
			Meta:       map[interface{}]interface{}{},
			Value:      datapoint.NewIntValue(2),
			MetricType: metricSet[jenkinsJobBuildCount].Type,
			Timestamp:  time.Time{},
		},
		{
			Metric:     jenkinsJobTotalTime,
			Dimensions: map[string]string{"build_result": Success, "job_name": job1FullName, "job_type": JobType},
			Meta:       map[interface{}]interface{}{},
			Value:      datapoint.NewIntValue(9991),
			MetricType: metricSet[jenkinsJobTotalTime].Type,
			Timestamp:  time.Time{},
		},
		{
			Metric:     jenkinsJobBuildCount,
			Dimensions: map[string]string{"build_result": Success, "job_name": job1FullName, "job_type": JobType},
			Meta:       map[interface{}]interface{}{},
			Value:      datapoint.NewIntValue(2),
			MetricType: metricSet[jenkinsJobBuildCount].Type,
			Timestamp:  time.Time{},
		},
		{
			Metric:     jenkinsJobTotalTime,
			Dimensions: map[string]string{"build_result": Aborted, "job_name": job1FullName, "job_type": JobType},
			Meta:       map[interface{}]interface{}{},
			Value:      datapoint.NewIntValue(8),
			MetricType: metricSet[jenkinsJobTotalTime].Type,
			Timestamp:  time.Time{},
		},
		{
			Metric:     jenkinsJobBuildCount,
			Dimensions: map[string]string{"build_result": Aborted, "job_name": job1FullName, "job_type": JobType},
			Meta:       map[interface{}]interface{}{},
			Value:      datapoint.NewIntValue(1),
			MetricType: metricSet[jenkinsJobBuildCount].Type,
			Timestamp:  time.Time{},
		},
	}
)

func TestGetDPS(t *testing.T) {
	// Run the initial call twice on the same list of builds and
	// make sure the state matches the initial expected result and
	// no DPS are generated
	for i := 0; i < 2; i++ {
		sort.Sort(ByBuildID(initialBuilds))
		assert.Empty(t, monitor.getJobDPS(job1, initialBuilds))
		assert.Contains(t, monitor.JobsMetricsState, job1FullName)
		assert.EqualValues(t, initialMetricStateResult, monitor.JobsMetricsState[job1FullName])
	}

	// Run twice on a new slice of builds
	for i := 0; i < 2; i++ {
		sort.Sort(ByBuildID(secondBuilds))
		secondDPS := monitor.getJobDPS(job1, secondBuilds)
		assert.Empty(t, monitor.getJobDPS(job2, initialBuilds))
		assert.EqualValues(t, secondMetricStateResult, monitor.JobsMetricsState[job1FullName])
		assert.ElementsMatch(t, expectedDPS2, secondDPS)
	}

	sort.Sort(ByBuildID(thirdBuilds))
	thirdDPS := monitor.getJobDPS(job1, thirdBuilds)
	assert.EqualValues(t, thirdMetricStateResult, monitor.JobsMetricsState[job1FullName])
	assert.ElementsMatch(t, expectedDPS3, thirdDPS)
}

func TestJobsBuildsStateCleanup(t *testing.T) {
	monitor.JobsMetricsState[job1FullName] = thirdMetricStateResult
	monitor.JobsMetricsState[job1FullName] = secondMetricStateResult
	assert.Contains(t, monitor.JobsMetricsState, job1FullName)
	assert.Contains(t, monitor.JobsMetricsState, job1FullName)
	// jobsBuildsStateCleanup with a map missing testJob2
	// should remove testJob2 from and keep testJob1
	monitor.jobsBuildsStateCleanup(map[string]struct{}{job1FullName: {}})
	assert.NotContains(t, monitor.JobsMetricsState, job2FullName)
	assert.Contains(t, monitor.JobsMetricsState, job1FullName)

}
