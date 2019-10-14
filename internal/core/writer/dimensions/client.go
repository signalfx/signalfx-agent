package dimensions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/core/propfilters"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
)

// DimensionClient sends updates to dimensions to the SignalFx API
type DimensionClient struct {
	sync.RWMutex
	ctx           context.Context
	Token         string
	APIURL        *url.URL
	client        *http.Client
	requestSender *reqSender
	sendDelay     time.Duration
	// Keeps track of what has been synced so we don't do unnecessary syncs
	history *lru.Cache
	// Set of dims that have been queued up for sending.  Use map to quickly
	// look up in case we need to replace due to flappy prop generation.
	delayedSet map[types.DimensionKey]*types.Dimension
	// Queue of dimensions to update.  The ordering should never change once
	// put in the queue so no need for heap/priority queue.
	delayedQueue         chan *queuedDimension
	mergedDimensionQueue chan *types.Dimension
	PropertyFilterSet    *propfilters.FilterSet
	// For easier unit testing
	now        func() time.Time
	logUpdates bool

	DimensionsCurrentlyDelayed int64
	TotalDimensionsDropped     int64
	// The number of dimension updates that happened to the same dimension
	// within sendDelay.
	TotalFlappyUpdates int64
}

type queuedDimension struct {
	*types.Dimension
	TimeToSend time.Time
}

// NewDimensionClient returns a new client
func NewDimensionClient(ctx context.Context, conf *config.WriterConfig) (*DimensionClient, error) {
	history, err := lru.New(int(conf.PropertiesHistorySize))
	if err != nil {
		panic("could not create properties history cache: " + err.Error())
	}

	propFilters, err := conf.PropertyFilters()
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:        int(conf.PropertiesMaxRequests),
			MaxIdleConnsPerHost: int(conf.PropertiesMaxRequests),
			IdleConnTimeout:     30 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}
	sender := newReqSender(ctx, client, conf.PropertiesMaxRequests)

	return &DimensionClient{
		ctx:                  ctx,
		Token:                conf.SignalFxAccessToken,
		APIURL:               conf.ParsedAPIURL(),
		sendDelay:            time.Duration(conf.PropertiesSendDelaySeconds) * time.Second,
		history:              history,
		delayedSet:           make(map[types.DimensionKey]*types.Dimension),
		delayedQueue:         make(chan *queuedDimension, conf.PropertiesMaxBuffered),
		mergedDimensionQueue: make(chan *types.Dimension),
		requestSender:        sender,
		client:               client,
		PropertyFilterSet:    propFilters,
		now:                  time.Now,
		logUpdates:           conf.LogDimensionUpdates,
	}, nil
}

// Start the client's processing queue
func (dc *DimensionClient) Start() {
	go dc.processQueue()
}

// AcceptDimension to be sent to the API.  This will return fairly quickly and
// won't block.  If the buffer is full, the dim update will be dropped.
func (dc *DimensionClient) AcceptDimension(dim *types.Dimension) error {
	filteredDim := &(*dim)

	filteredDim = dc.PropertyFilterSet.FilterDimension(filteredDim)

	dc.Lock()
	defer dc.Unlock()

	if delayedDim := dc.delayedSet[filteredDim.Key()]; delayedDim != nil {
		dc.TotalFlappyUpdates++

		// Don't further delay it if it gets updated so that we are always
		// guaranteed to update a dimension at least some times, even if it
		// continually gets updated more frequently than `sendDelay` seconds
		// (which should be dealt with somewhere else).
		delayedDim.Properties = filteredDim.Properties
		delayedDim.Tags = filteredDim.Tags
	} else {
		if dc.isDuplicate(filteredDim) {
			return nil
		}

		atomic.AddInt64(&dc.DimensionsCurrentlyDelayed, int64(1))

		dc.delayedSet[filteredDim.Key()] = filteredDim
		select {
		case dc.delayedQueue <- &queuedDimension{
			Dimension:  filteredDim,
			TimeToSend: dc.now().Add(dc.sendDelay),
		}:
			break
		default:
			dc.TotalDimensionsDropped++
			return errors.New("dropped dimension update, propertiesMaxBuffered exceeded")
		}
	}

	return nil
}

func (dc *DimensionClient) processQueue() {
	send := func(dim *types.Dimension) {
		if err := dc.setPropertiesOnDimension(dim); err != nil {
			log.WithError(err).WithField("dim", dim.Key()).Error("Could not send dimension update")
		} else if dc.logUpdates {
			log.WithFields(log.Fields{
				"name":       dim.Name,
				"value":      dim.Value,
				"properties": dim.Properties,
				"tags":       dim.Tags,
			}).Info("Updated dimension")
		}
	}

	for {
		select {
		case <-dc.ctx.Done():
			return
		case delayedDim := <-dc.delayedQueue:
			now := dc.now()
			if now.Before(delayedDim.TimeToSend) {
				// dims are always in the channel in order of TimeToSend
				time.Sleep(delayedDim.TimeToSend.Sub(now))
			}

			atomic.AddInt64(&dc.DimensionsCurrentlyDelayed, int64(-1))

			dc.Lock()
			delete(dc.delayedSet, delayedDim.Dimension.Key())
			dc.Unlock()

			if !dc.isDuplicate(delayedDim.Dimension) {
				send(delayedDim.Dimension)
			}
		case dim := <-dc.mergedDimensionQueue:
			send(dim)
		}
	}
}

// setPropertiesOnDimension will set custom properties on a specific dimension
// value.  It will wipe out any description on the dimension.  There is no
// retry logic here so any failures are terminal.
func (dc *DimensionClient) setPropertiesOnDimension(dim *types.Dimension) error {
	var (
		req *http.Request
		err error
	)

	if dim.MergeIntoExisting {
		req, err = dc.makePatchRequest(dim.Name, dim.Value, dim.Properties, dim.Tags)
	} else {
		req, err = dc.makeReplaceRequest(dim.Name, dim.Value, dim.Properties, dim.Tags)
	}

	if err != nil {
		return err
	}

	req = req.WithContext(context.WithValue(dc.ctx, reqDoneCallbackKeyVar, func() {
		// Add it to the history only after successfully propagated so that we
		// will end up retrying updates (monitors should send the property
		// updates through to the writer on the same interval as datapoints).
		// This could lead to some duplicates if there are multiple concurrent
		// calls for the same dim props, but that's ok.
		dc.history.Add(dim.Key(), dim)
	}))

	// This will block if we don't have enough requests
	dc.requestSender.send(req)

	return nil
}

// isDuplicate returns true if the exact same dimension properties have been
// synced before in the recent past.
func (dc *DimensionClient) isDuplicate(dim *types.Dimension) bool {
	prev, ok := dc.history.Get(dim.Key())
	return ok && reflect.DeepEqual(prev.(*types.Dimension), dim)
}

func (dc *DimensionClient) makeDimURL(key, value string) (*url.URL, error) {
	url, err := dc.APIURL.Parse(fmt.Sprintf("/v2/dimension/%s/%s", key, value))
	if err != nil {
		return nil, fmt.Errorf("could not construct dimension property PUT URL with %s / %s: %v", key, value, err)
	}

	return url, nil
}

func (dc *DimensionClient) makePatchRequest(key, value string, props map[string]string, tags map[string]bool) (*http.Request, error) {
	tagsToAdd := []string{}
	tagsToRemove := []string{}

	for tag, shouldAdd := range tags {
		if shouldAdd {
			tagsToAdd = append(tagsToAdd, tag)
		} else {
			tagsToRemove = append(tagsToRemove, tag)
		}
	}

	propsWithNil := map[string]interface{}{}
	// Set any empty string props to nil so they get removed from the
	// dimension altogether.
	for k, v := range props {
		if v == "" {
			propsWithNil[k] = nil
		} else {
			propsWithNil[k] = v
		}
	}

	json, err := json.Marshal(map[string]interface{}{
		"customProperties": propsWithNil,
		"tags":             tagsToAdd,
		"tagsToRemove":     tagsToRemove,
	})
	if err != nil {
		return nil, err
	}

	url, err := dc.makeDimURL(key, value)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		"PATCH",
		strings.TrimRight(url.String(), "/")+"/_/sfxagent",
		bytes.NewReader(json))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-SF-TOKEN", dc.Token)

	return req, nil
}

func (dc *DimensionClient) makeReplaceRequest(key, value string, props map[string]string, tags map[string]bool) (*http.Request, error) {
	json, err := json.Marshal(map[string]interface{}{
		"key":              key,
		"value":            value,
		"customProperties": props,
		"tags":             utils.StringSetToSlice(tags),
	})
	if err != nil {
		return nil, err
	}

	url, err := dc.makeDimURL(key, value)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		"PUT",
		url.String(),
		bytes.NewReader(json))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-SF-TOKEN", dc.Token)

	return req, nil
}
