package cluster

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/sirupsen/logrus"
)

type MemcachedElector struct {
	client          *memcache.Client
	Expiration      time.Duration
	RetryTimeout    time.Duration
	RenewalDeadline time.Duration
}

func NewMemcachedElector(client *memcache.Client) *MemcachedElector {
	return &MemcachedElector{
		client:          client,
		Expiration:      60 * time.Second,
		RetryTimeout:    30 * time.Second,
		RenewalDeadline: 40 * time.Second,
	}
}

var _ Elector = &MemcachedElector{}

func (me *MemcachedElector) RunSelection(ctx context.Context, agentID string, key Key, callback ChangeCallback) error {
	if me.RenewalDeadline >= me.Expiration {
		return errors.New("RenewalDeadline must be < Expiration time")
	}

	exptime := int32(me.Expiration.Seconds())

	item := &memcache.Item{
		Key:        fmt.Sprintf("signalfx-agent-%s-%d", key.Name, key.Index),
		Value:      []byte(agentID),
		Expiration: exptime,
	}

MAIN:
	for {
		addErr := me.client.Add(item)

		it, err := me.client.Get(item.Key)
		if err != nil {
			logrus.Warnf("Could not determine current leader from memcached: %v", err)
			select {
			case <-time.After(me.RetryTimeout):
				continue
			case <-ctx.Done():
				return nil
			}
		}
		// We need to let the elector know about all changes to the
		// selected agent instance, whether we got it or not.
		callback(key, string(it.Value))

		if addErr == memcache.ErrNotStored {
			// Another agent has claimed the key.  Wait for a while and
			// retry in case that agent fails.
			jitter := time.Duration(rand.Int63n(5000)) * time.Millisecond
			select {
			case <-time.After(me.RetryTimeout + jitter):
				continue
			case <-ctx.Done():
				return nil
			}
		}
		logrus.Debugf("Successfully added memcache key %v", key)

		// This instance got selected, keep it renewed as long as possible
		for {
			select {
			case <-time.Tick(me.RenewalDeadline):
				it.Expiration = exptime
				err := me.client.CompareAndSwap(it)
				if err != nil {
					logrus.Warnf("Failed to renew memcache lease for key %v: %v", key, err)
					continue MAIN
				}
				it, err = me.client.Get(item.Key)
				if err != nil {
					logrus.Warnf("Could not refresh current leader from memcached: %v", err)
					select {
					case <-time.After(me.RetryTimeout):
						continue MAIN
					case <-ctx.Done():
						return nil
					}
				}
				continue
			case <-ctx.Done():
				_ = me.client.Delete(item.Key)
				return nil
			}
		}
	}
}
