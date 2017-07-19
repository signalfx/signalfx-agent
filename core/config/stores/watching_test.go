package stores

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"

	"github.com/docker/libkv/store"
	. "github.com/signalfx/neo-agent/neotest"
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

func Test_AssetSyncer(t *testing.T) {
	Convey("Test start stop", t, func() {
		assets := NewAssetSyncer()
		assets.Start(func(view *AssetsView) {
		})
		assets.Stop()
	})

	Convey("Watch directory", t, func() {
		assets := NewAssetSyncer()
		ch := make(chan *AssetsView, 3)
		assets.Start(func(view *AssetsView) {
			log.Printf("view: %+v", view)
			ch <- view
		})

		tmpdir1, err := ioutil.TempDir("", "asset-sync")
		Must(t, err)

		tmpdir2, err := ioutil.TempDir("", "asset-sync")
		Must(t, err)

		ws := NewAssetSpec()
		ws.Dirs["first"] = tmpdir1
		Must(t, assets.Update(ws))

		So(assets.dirs, ShouldHaveLength, 1)

		newFile := path.Join(tmpdir1, "new-file")

		// It will first send an empty set for the empty directory.
		view := <-ch
		So(view.Dirs["first"], ShouldBeEmpty)

		Must(t, ioutil.WriteFile(newFile, []byte("bar"), 0644))

		view = <-ch
		So(view.Dirs["first"], ShouldResemble, []*store.KVPair{
			&store.KVPair{Key: newFile, Value: []byte("bar"), LastIndex: 0},
		})

		Reset(func() {
			assets.Stop()
			Must(t, os.RemoveAll(tmpdir1))
			Must(t, os.RemoveAll(tmpdir2))
		})

		Convey("Test directory changed", func() {
			ws.Dirs["first"] = tmpdir2
			assets.Update(ws)
			So(assets.dirs, ShouldHaveLength, 1)
			So(assets.dirs["first"].uri, ShouldEqual, tmpdir2)
		})

		Convey("Watch another directory", func() {
			ws.Dirs["second"] = tmpdir2
			Must(t, assets.Update(ws))
			So(assets.dirs, ShouldHaveLength, 2)

			Convey("Unwatch directory", func() {
				delete(ws.Dirs, "second")
				Must(t, assets.Update(ws))
				So(assets.dirs, ShouldHaveLength, 1)
			})

			Convey("Unwatch all directories", func() {
				assets.Update(nil)
				So(assets.dirs, ShouldBeEmpty)
			})
		})
	})
}
