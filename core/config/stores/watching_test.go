package stores

import (
	"testing"

	"github.com/docker/libkv/store"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	reconnectTime = 0
}

type GetSpec struct {
	pair *store.KVPair
	err  error
}

type FakeStore struct {
}

type FakeWatchSuccessStore struct {
	FakeStore
	watchCh chan *store.KVPair
}

type FakeWatchFailStore struct {
	FakeStore
	callNum int
}

func (f *FakeStore) Get(path string) (*store.KVPair, error) {
	panic("Get unimplemented")
}

func (f *FakeStore) Watch(path string, stop <-chan struct{}) (<-chan *store.KVPair, error) {
	panic("Watch unimplemented")
}

func (f *FakeStore) WatchTree(string, <-chan struct{}) (<-chan []*store.KVPair, error) {
	panic("WatchTree unimplemented")
}

func (f *FakeStore) Exists(string) (bool, error) {
	panic("Exists unimplemented")
}

func (f *FakeStore) Put(key string, value []byte, opts *store.WriteOptions) error {
	panic("Put unimplemented")
}

func (f *FakeStore) AtomicPut(key string, value []byte, previous *store.KVPair, _ *store.WriteOptions) (bool, *store.KVPair, error) {
	panic("AtomicPut unimplemented")
}
func (f *FakeStore) List(dir string) ([]*store.KVPair, error) {
	panic("List unimplemented")
}

func (f *FakeStore) Close() {}

func (f *FakeWatchSuccessStore) Watch(path string, stop <-chan struct{}) (<-chan *store.KVPair, error) {
	go func() {
		f.watchCh <- &store.KVPair{Key: path, Value: []byte("bar"), LastIndex: 0}
		f.watchCh <- &store.KVPair{Key: path, Value: []byte("changed"), LastIndex: 0}
	}()
	return f.watchCh, nil
}

func (f *FakeWatchFailStore) Watch(path string, stop <-chan struct{}) (<-chan *store.KVPair, error) {
	ch := make(chan *store.KVPair)
	go func() {
		switch f.callNum {
		case 0:
			ch <- &store.KVPair{Key: path, Value: []byte("bar"), LastIndex: 0}
			f.callNum++
			close(ch)
		case 1:
			ch <- &store.KVPair{Key: path, Value: []byte("bar"), LastIndex: 0}
			ch <- &store.KVPair{Key: path, Value: []byte("changed"), LastIndex: 0}
			f.callNum++
		default:
			panic("unexpected call")
		}
	}()
	return ch, nil
}

func Test_WatchRobust(t *testing.T) {
	Convey("Test that initial watch works", t, func() {
		stop := make(chan struct{})
		source := &FakeWatchSuccessStore{watchCh: make(chan *store.KVPair)}
		ch, err := WatchRobust(source, "foo", stop)
		So(err, ShouldBeNil)

		pair := <-ch
		So(pair, ShouldNotBeNil)
		So(pair.Key, ShouldEqual, "foo")
		So(pair.Value, ShouldResemble, []byte("bar"))

		pair = <-ch
		So(pair.Key, ShouldEqual, "foo")
		So(pair.Value, ShouldResemble, []byte("changed"))

		stop <- struct{}{}
	})

	Convey("Test that when watch fails it reconnects", t, func() {
		stop := make(chan struct{})
		source := &FakeWatchFailStore{}

		ch, err := WatchRobust(source, "foo", stop)
		So(err, ShouldBeNil)

		pair := <-ch
		So(pair, ShouldNotBeNil)
		So(pair.Key, ShouldEqual, "foo")
		So(pair.Value, ShouldResemble, []byte("bar"))

		pair = <-ch
		So(pair.Key, ShouldEqual, "foo")
		So(pair.Value, ShouldResemble, []byte("bar"))

		pair = <-ch
		So(pair.Key, ShouldEqual, "foo")
		So(pair.Value, ShouldResemble, []byte("changed"))

		stop <- struct{}{}
	})
}
