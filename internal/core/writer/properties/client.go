package properties

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

// DimensionPropertyClient sends updates to dimensions to the SignalFx API
type DimensionPropertyClient struct {
	sync.RWMutex
	ctx           context.Context
	Token         string
	APIURL        *url.URL
	requestSender *reqSender
	sendDelay     time.Duration
	// Keeps track of what has been synced so we don't do unnecessary syncs
	history *lru.Cache
	// Set of dims that have been queued up for sending.  Use map to quickly
	// look up in case we need to replace due to flappy prop generation.
	delayedSet map[types.Dimension]*types.DimProperties
	// Queue of dimensions to update.  The ordering should never change once
	// put in the queue so no need for heap/priority queue.
	delayedQueue      chan *queuedDimension
	PropertyFilterSet *propfilters.FilterSet
	// For easier unit testing
	now func() time.Time

	DimensionsCurrentlyDelayed int64
	TotalDimensionsDropped     int64
	// The number of dimension updates that happened to the same dimension
	// within sendDelay.
	TotalFlappyUpdates int64
}

type queuedDimension struct {
	*types.DimProperties
	TimeToSend time.Time
}

// NewDimensionPropertyClient returns a new client
func NewDimensionPropertyClient(ctx context.Context, conf *config.WriterConfig) (*DimensionPropertyClient, error) {
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

	return &DimensionPropertyClient{
		ctx:               ctx,
		Token:             conf.SignalFxAccessToken,
		APIURL:            conf.ParsedAPIURL(),
		sendDelay:         time.Duration(conf.PropertiesSendDelaySeconds) * time.Second,
		history:           history,
		delayedSet:        make(map[types.Dimension]*types.DimProperties),
		delayedQueue:      make(chan *queuedDimension, conf.PropertiesMaxBuffered),
		requestSender:     sender,
		PropertyFilterSet: propFilters,
		now:               time.Now,
	}, nil
}

// Start the client's processing queue
func (dpc *DimensionPropertyClient) Start() {
	go dpc.processQueue()
}

// AcceptDimProp to be sent to the API.  This will return fairly quickly and
// won't block.  If the buffer is full, the dim update will be dropped.
func (dpc *DimensionPropertyClient) AcceptDimProp(dimProps *types.DimProperties) error {
	filteredDimProps := &(*dimProps)

	filteredDimProps = dpc.PropertyFilterSet.FilterDimProps(filteredDimProps)

	dpc.Lock()
	defer dpc.Unlock()

	if delayedDim := dpc.delayedSet[filteredDimProps.Dimension]; delayedDim != nil {
		dpc.TotalFlappyUpdates++

		delayedDim.Properties = filteredDimProps.Properties
		delayedDim.Tags = filteredDimProps.Tags
		// Don't further delay it if it gets updated so that we are always
		// guaranteed to update a dimension at least some times, even if it
		// continually gets updated more frequently than `sendDelay` seconds
		// (which should be dealt with somewhere else).
	} else {
		if dpc.isDuplicate(filteredDimProps) {
			return nil
		}

		atomic.AddInt64(&dpc.DimensionsCurrentlyDelayed, int64(1))

		dpc.delayedSet[filteredDimProps.Dimension] = filteredDimProps
		select {
		case dpc.delayedQueue <- &queuedDimension{
			DimProperties: filteredDimProps,
			TimeToSend:    dpc.now().Add(dpc.sendDelay),
		}:
			break
		default:
			dpc.TotalDimensionsDropped++
			return errors.New("dropped dimension update, propertiesMaxBuffered exceeded")
		}
	}

	return nil
}

func (dpc *DimensionPropertyClient) processQueue() {
	for delayedDim := range dpc.delayedQueue {
		now := dpc.now()
		if now.Before(delayedDim.TimeToSend) {
			// dims are always in the channel in order of TimeToSend
			time.Sleep(delayedDim.TimeToSend.Sub(now))
		}
		atomic.AddInt64(&dpc.DimensionsCurrentlyDelayed, int64(-1))

		dpc.Lock()
		delete(dpc.delayedSet, delayedDim.DimProperties.Dimension)
		dpc.Unlock()

		if !dpc.isDuplicate(delayedDim.DimProperties) {
			if err := dpc.setPropertiesOnDimension(delayedDim.DimProperties); err != nil {
				log.WithError(err).WithField("dim", delayedDim.DimProperties.Dimension).Error("Could not send dimension update")
			}
		}
	}
}

// setPropertiesOnDimension will set custom properties on a specific dimension
// value.  It will wipe out any description on the dimension.  There is no
// retry logic here so any failures are terminal.
func (dpc *DimensionPropertyClient) setPropertiesOnDimension(dimProps *types.DimProperties) error {
	req, err := dpc.makeRequest(dimProps.Name, dimProps.Value, dimProps.Properties, dimProps.Tags)
	if err != nil {
		return err
	}

	req = req.WithContext(context.WithValue(dpc.ctx, reqDoneCallbackKeyVar, func() {
		// Add it to the history only after successfully propagated so that we
		// will end up retrying updates (monitors should send the property
		// updates through to the writer on the same interval as datapoints).
		// This could lead to some duplicates if there are multiple concurrent
		// calls for the same dim props, but that's ok.
		dpc.history.Add(dimProps.Dimension, dimProps)
	}))

	// This will block if we don't have enough requests
	dpc.requestSender.send(req)
	return nil
}

// isDuplicate returns true if the exact same dimension properties have been
// synced before in the recent past.
func (dpc *DimensionPropertyClient) isDuplicate(dimProps *types.DimProperties) bool {
	prev, ok := dpc.history.Get(dimProps.Dimension)
	return ok && reflect.DeepEqual(prev.(*types.DimProperties), dimProps)
}

func (dpc *DimensionPropertyClient) makeRequest(key, value string, props map[string]string, tags map[string]bool) (*http.Request, error) {
	json, err := json.Marshal(map[string]interface{}{
		"key":              key,
		"value":            value,
		"customProperties": props,
		"tags":             utils.StringSetToSlice(tags),
	})
	if err != nil {
		return nil, err
	}

	url, err := dpc.APIURL.Parse(fmt.Sprintf("/v2/dimension/%s/%s", key, value))
	if err != nil {
		return nil, fmt.Errorf("could not construct dimension property PUT URL with %s / %s: %v", key, value, err)
	}

	req, err := http.NewRequest(
		"PUT",
		url.String(),
		bytes.NewReader(json))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-SF-TOKEN", dpc.Token)

	return req, nil
}
