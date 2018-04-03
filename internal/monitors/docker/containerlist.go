package docker

import (
	"context"
	"sync"
	"time"

	dtypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	docker "github.com/docker/docker/client"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"github.com/signalfx/signalfx-agent/internal/utils/filter"
)

// Returns a map of containers by id and ensures that the map is kept up to
// date as containers come and go.  Access to the map should be done while
// holding the provided lock.
func listAndWatchContainers(ctx context.Context, client *docker.Client, lock *sync.Mutex, imageFilter filter.StringFilter) (map[string]*dtypes.ContainerJSON, error) {
	containers := make(map[string]*dtypes.ContainerJSON)

	// Make sure you hold the lock before calling this
	updateContainer := func(id string) {
		inspectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		c, err := client.ContainerInspect(inspectCtx, id)
		if err != nil {
			logger.WithError(err).Errorf("Could not inspect updated container %s", id)
		} else if !imageFilter.Matches(c.Config.Image) {
			logger.Debugf("Monitoring docker container %s", id)
			containers[id] = &c
		}
		cancel()
	}

	watchStarted := make(chan struct{})
	go func() {
		// This pattern is taken from
		// https://github.com/docker/cli/blob/master/cli/command/container/stats.go
		f := filters.NewArgs()
		f.Add("type", "container")
		f.Add("event", "destroy")
		f.Add("event", "die")
		f.Add("event", "pause")
		f.Add("event", "stop")
		f.Add("event", "start")
		f.Add("event", "unpause")
		f.Add("event", "update")
		lastTime := time.Now()

	START_STREAM:
		for {
			since := lastTime.Format(time.RFC3339Nano)
			options := dtypes.EventsOptions{
				Filters: f,
				Since:   since,
			}

			logger.Infof("Watching for Docker events since %s", since)
			eventCh, errCh := client.Events(ctx, options)

			if !utils.IsSignalChanClosed(watchStarted) {
				close(watchStarted)
			}

			for {
				select {
				case event := <-eventCh:
					lock.Lock()

					switch event.Action {
					case "destroy", "die", "pause", "stop":
						logger.Debugf("No longer monitoring container %s", event.ID)
						delete(containers, event.ID)
					case "start", "unpause", "update":
						updateContainer(event.ID)
					}

					lock.Unlock()

					lastTime = time.Unix(0, event.TimeNano)

				case err := <-errCh:
					logger.WithError(err).Error("Error watching docker container events")
					time.Sleep(3 * time.Second)
					continue START_STREAM

				case <-ctx.Done():
					// Event stream is tied to the same context and will quit
					// also.
					return
				}
			}
		}
	}()

	<-watchStarted

	f := filters.NewArgs()
	f.Add("status", "running")
	options := dtypes.ContainerListOptions{
		Filters: f,
	}
	containerList, err := client.ContainerList(ctx, options)
	if err != nil {
		return nil, err
	}

	wg := sync.WaitGroup{}
	for i := range containerList {
		wg.Add(1)
		// The Docker API has a different return type for list vs. inspect, and
		// no way to get the return type of list for individual containers,
		// which makes this harder than it should be.
		go func(id string) {
			lock.Lock()
			defer lock.Unlock()
			updateContainer(id)
			wg.Done()
		}(containerList[i].ID)
	}

	wg.Wait()

	return containers, nil
}
