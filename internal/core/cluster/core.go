package cluster

import (
	"context"
	"errors"
	"sync"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

type NoticeCh chan ElectionResult
type ElectionResult struct {
	IsElected bool
	Key       Key
}

type UnsubscribeFunc func()
type Key struct {
	Name  string
	Index uint
}

type ChangeCallback func(key Key, newLeaderId string)

type Elector interface {
	RunSelection(ctx context.Context, agentID string, key Key, callback ChangeCallback) error
}

// MultiElector wraps a cluster election implementation and can use it to elect
// multiple keys simultaneously.
type MultiElector struct {
	sync.Mutex
	ctx              context.Context
	agentID          string
	noticeChans      map[Key][]NoticeCh
	leaderIdentities map[Key]string
	cancelers        map[Key]context.CancelFunc
	elector          Elector
}

func NewMultiElector(ctx context.Context, agentID string, elector Elector) *MultiElector {
	return &MultiElector{
		ctx:              ctx,
		agentID:          agentID,
		noticeChans:      make(map[Key][]NoticeCh),
		leaderIdentities: make(map[Key]string),
		cancelers:        make(map[Key]context.CancelFunc),
		elector:          elector,
	}
}

// RegisterCandidateInstance subscribes this agent instance to the election
// process.
func (c *MultiElector) RegisterCandidateInstance(keyName string, maxInstances uint) (NoticeCh, UnsubscribeFunc, error) {
	if maxInstances == 0 {
		return nil, nil, errors.New("maxInstances must be > 0")
	}

	noticeCh := make(NoticeCh, maxInstances)
	var unsubs []UnsubscribeFunc
	for i := uint(0); i < maxInstances; i++ {
		unsub, err := c.registerKey(Key{Name: keyName, Index: i}, noticeCh)
		if err != nil {
			return nil, nil, err
		}
		unsubs = append(unsubs, unsub)
	}
	return noticeCh, func() {
		for _, unsub := range unsubs {
			unsub()
		}
	}, nil
}

func (c *MultiElector) registerKey(key Key, noticeCh NoticeCh) (UnsubscribeFunc, error) {
	c.Lock()
	defer c.Unlock()

	// Prime it with the fact that we are the leader if we are -- this
	// guarantees that the first value sent to the chan will always be true.
	if c.IsLeader(key) {
		go func() { noticeCh <- ElectionResult{true, key} }()
	}

	if len(c.noticeChans[key]) == 0 {
		var selectionCtx context.Context
		selectionCtx, c.cancelers[key] = context.WithCancel(c.ctx)
		go func() {
			err := c.elector.RunSelection(selectionCtx, c.agentID, key, c.handleSelectionChange)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"err": err,
					"key": key,
				}).Error("Election process failed")
			}
		}()
	}

	c.noticeChans[key] = append(c.noticeChans[key], noticeCh)

	return func() {
		c.Lock()
		defer c.Unlock()

		log.Info("Unsubscribing leader notice channel")
		chans := c.noticeChans[key]
		for i := range chans {
			if chans[i] == noticeCh {
				c.noticeChans[key] = append(chans[:i], chans[i+1:]...)
				return
			}
		}

		if len(c.noticeChans[key]) == 0 {
			// Shut it down if nothing needs it anymore.  This doesn't actually
			// do anything on the K8s leaderelection implementation since it is
			// unstoppable.
			c.cancelers[key]()
		}

		log.Error("Could not find leader notice channel to unsubscribe")
	}, nil
}

func (c *MultiElector) handleSelectionChange(key Key, newLeaderID string) {
	c.Lock()
	defer c.Unlock()

	if newLeaderID == c.leaderIdentities[key] {
		return
	}

	log.Infof("Elected '%s' for key '%v'", newLeaderID, key)
	if newLeaderID == c.agentID && c.leaderIdentities[key] != c.agentID {
		for i := range c.noticeChans[key] {
			c.noticeChans[key][i] <- ElectionResult{true, key}
		}
	} else if newLeaderID != c.agentID && c.leaderIdentities[key] == c.agentID {
		for i := range c.noticeChans[key] {
			c.noticeChans[key][i] <- ElectionResult{false, key}
		}
	}
	c.leaderIdentities[key] = newLeaderID
}

func (c *MultiElector) IsLeader(key Key) bool {
	return c.leaderIdentities[key] == c.agentID
}
