package watcher

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/docker/libkv/store"
	. "github.com/signalfx/neo-agent/neotest"
	. "github.com/smartystreets/goconvey/convey"
)

func Test_PollingWatcher_poll(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)

	changed := func(ch <-chan []*store.KVPair, pairs []*store.KVPair) {
		select {
		case changed := <-ch:
			So(changed, ShouldResemble, pairs)
		}
	}

	Convey("Given a cloned version of testdata", t, func() {
		_, cleanup := CloneTestData(t)

		w := NewPollingWatcher(60 * time.Second)
		stop := make(chan struct{})

		Reset(func() {
			cleanup()
			stop <- struct{}{}
		})

		Convey("Given a watched empty directory", func() {
			// Create empty directory since we can't check it into git.
			Must(t, os.Mkdir("dir/empty-dir", 0755))

			ch, err := w.WatchTree("dir/empty-dir", stop)
			So(err, ShouldBeNil)

			Must(t, os.Remove("dir/empty-dir"))

			go w.poll()

			changed(ch, Pairs{})
			changed(ch, Pairs{})
		})

		Convey("Given a watched directory that doesn't exist", func() {
			ch, err := w.WatchTree("does-not-exist", stop)
			So(err, ShouldBeNil)
			changed(ch, Pairs{})

			w.poll()
			So(ch, ShouldBeEmpty)

			Convey("Create non-existent directory", func() {
				Must(t, os.Mkdir("does-not-exist", 0755))
				w.poll()

				changed(ch, Pairs{})

				Convey("Create file in previously missing directory", func() {
					Must(t, ioutil.WriteFile("does-not-exist/new-file", []byte("new file"), 0644))
					w.poll()

					changed(ch, Pairs{
						pair("does-not-exist/new-file", []byte("new file")),
					})
				})
			})
		})

		Convey("Given an unreadable watched directory", func() {
			Must(t, os.Mkdir("unreadable-dir", 0))
			ch, err := w.WatchTree("unreadable-dir", stop)
			So(err, ShouldBeNil)
			changed(ch, Pairs{})
		})

		Convey("Give a watched directory", func() {
			ch, err := w.WatchTree("dir", stop)
			So(err, ShouldBeNil)

			Convey("Contains an unreadable file", func() {
				pairs := <-ch
				So(pairs, ShouldHaveLength, 3)

				Must(t, ioutil.WriteFile("dir/new-file", []byte("unreadable file"), 0))
				// Poll twice to make sure not notified multiple times.
				w.poll()
				pairs = <-ch
				So(pairs, ShouldHaveLength, 3)

				w.poll()
				So(ch, ShouldBeEmpty)
			})

			Convey("A directory is added", func() {
				pairs := <-ch
				So(pairs, ShouldHaveLength, 3)

				Must(t, os.Mkdir("dir/new-dir", 0644))
				w.poll()

				pairs = <-ch

				So(pairs, ShouldHaveLength, 4)

				Convey("A directory is removed", func() {
					Must(t, os.Remove("dir/new-dir"))
					w.poll()
					pairs = <-ch

					So(pairs, ShouldHaveLength, 3)
				})
			})

			Convey("A file is added", func() {
				pairs := <-ch
				Must(t, ioutil.WriteFile("dir/new-file", []byte("new file"), 0644))
				w.poll()
				pairs = <-ch

				So(pairs, ShouldHaveLength, 4)
			})
			Convey("A file is removed", func() {
				pairs := <-ch
				Must(t, os.Remove("dir/file1"))
				w.poll()

				pairs = <-ch
				So(pairs, ShouldHaveLength, 2)

				Convey("Then added back", func() {
					Must(t, ioutil.WriteFile("dir/file1", []byte("new file"), 0644))
					w.poll()

					pairs = <-ch

					So(pairs, ShouldHaveLength, 3)
				})
			})
			Convey("A file is modified", func() {
				Must(t, ioutil.WriteFile("dir/file1", []byte("changed"), 0644))
				// Have to make file change time in the future because at least
				// on Mac the modified resolution is 1 second.
				Must(t, os.Chtimes("dir/file1", future, future))

				go w.poll()
				changed(ch, Pairs{
					pair("dir/file1", []byte("1\n")),
					pair("dir/file2", []byte("2\n")),
					pair("dir/subdir", nil),
				})
				changed(ch, Pairs{
					pair("dir/file1", []byte("changed")),
					pair("dir/file2", []byte("2\n")),
					pair("dir/subdir", nil),
				})
			})
			Convey("Multiple files are modified", func() {
				pairs := <-ch
				for _, file := range []string{"dir/file1", "dir/file2"} {
					Must(t, ioutil.WriteFile(file, []byte("changed"), 0644))
					// Have to make file change time in the future because at least
					// on Mac the modified resolution is 1 second.
					Must(t, os.Chtimes(file, future, future))
				}
				w.poll()

				pairs = <-ch

				So(pairs, ShouldHaveLength, 3)
			})

			Convey("A symlink is added", func() {
				pairs := <-ch
				Must(t, os.Symlink("file1", "dir/symlink"))
				w.poll()

				pairs = <-ch
				So(pairs, ShouldHaveLength, 4)

				changed = nil

				Convey("A symlink is removed", func() {
					Must(t, os.Remove("dir/symlink"))
					w.poll()
					pairs = <-ch

					So(pairs, ShouldHaveLength, 3)
				})

				Convey("A symlinks contents is modified", func() {
					Must(t, ioutil.WriteFile("dir/file1", []byte("changed"), 0644))
					// Have to make file change time in the future because at least
					// on Mac the modified resolution is 1 second.
					Must(t, os.Chtimes("dir/file1", future, future))

					w.poll()
					pairs = <-ch

					So(pairs, ShouldHaveLength, 4)
				})
			})
			Convey("All files are removed", func() {
				pairs := <-ch

				// Remove everything in the directory except the directory itself.
				if files, err := ioutil.ReadDir("dir"); err != nil {
					t.Fatal(err)
				} else {
					for _, file := range files {
						Must(t, os.RemoveAll(path.Join("dir", file.Name())))
					}
				}
				w.poll()
				pairs = <-ch

				So(pairs, ShouldBeEmpty)

				Convey("A file is added back", func() {
					Must(t, ioutil.WriteFile("dir/file1", []byte("changed"), 0644))
					// Have to make file change time in the future because at least
					// on Mac the modified resolution is 1 second.
					Must(t, os.Chtimes("dir/file1", future, future))

					w.poll()
					pairs = <-ch

					So(pairs, ShouldHaveLength, 1)
				})
			})
		})

		Convey("Given a watched file that doesn't exist", func() {
			ch, err := w.Watch("dir/file-does-not-exist", stop)
			So(err, ShouldBeNil)
			pr := <-ch
			So(pr, ShouldResemble, pair("dir/file-does-not-exist", nil))

			w.poll()
			So(ch, ShouldBeEmpty)
		})

		Convey("Given a watched file modified in the past", func() {
			d, err := time.Parse(time.UnixDate, "Mon Jan 2 15:04:05 MST 2006")
			if err != nil {
				t.Fatal(err)
			}
			Must(t, os.Chtimes("dir/file1", time.Now(), d))
			ch, err := w.Watch("dir/file1", stop)
			So(err, ShouldBeNil)

			Convey("And the contents have changed", func() {
				Must(t, ioutil.WriteFile("dir/file1", []byte("changed"), 0644))

				Convey("Modification time before file modification at time of watch", func() {
					// Set mtime to an old date.
					<-ch
					Must(t, os.Chtimes("dir/file1", time.Now(), d))
					w.poll()
					So(ch, ShouldBeEmpty)
				})

				Convey("Modification time is in the future", func() {
					<-ch
					Must(t, os.Chtimes("dir/file1", time.Now(), time.Now().Add(1*time.Hour)))
					w.poll()
					So(ch, ShouldHaveLength, 1)
				})

				Convey("Modification time equal to last polled time", func() {
					<-ch
					w.poll()
					<-ch
					w.poll()
					So(ch, ShouldBeEmpty)
				})

				Convey("The file is considered changed", func() {
					<-ch
					w.poll()
					pr := <-ch

					So(pr, ShouldResemble, pair("dir/file1", []byte("changed")))
				})
			})

			Convey("The file modified time is changed but not its contents", func() {
				<-ch
				Must(t, os.Chtimes("dir/file1", time.Now(), time.Now().Add(1*time.Hour)))
				w.poll()
				So(ch, ShouldBeEmpty)
			})

			Convey("The file is modified once but polled twice", func() {
				<-ch
				Must(t, ioutil.WriteFile("dir/file1", []byte("changed"), 0644))
				w.poll()
				w.poll()
				So(ch, ShouldHaveLength, 1)
			})

			Convey("The file is unmodified", func() {
				<-ch
				w.poll()

				So(ch, ShouldBeEmpty)
			})

			Convey("The file is removed", func() {
				<-ch
				Must(t, os.Remove("dir/file1"))
				w.poll()
				pr := <-ch

				// XXX: not sure this case is the same as libkv
				So(pr, ShouldResemble, pair("dir/file1", nil))

				Convey("Then readded with different contents", func() {
					Must(t, ioutil.WriteFile("dir/file1", []byte("new contents after remove"), 0644))
					w.poll()
					pr = <-ch

					So(pr, ShouldResemble, pair("dir/file1", []byte("new contents after remove")))
				})
				Convey("Then readded with same contents", func() {
					w.poll()
					Must(t, ioutil.WriteFile("dir/file1", []byte("1\n"), 0644))
					w.poll()

					pr = <-ch

					So(pr, ShouldResemble, pair("dir/file1", []byte("1\n")))
				})
			})
		})
	})
}

// These tests are too flaky now. It has a number of data races and is hard to
// fix without changing the way stopCh works and breaking the libkv interface.
// func Test_PollingWatcher_Watch(t *testing.T) {
// 	Convey("Given a watcher instance", t, func() {
// 		w := NewPollingWatcher(time.Second)
// 		w.Start()

// 		Convey("Watch a file", func() {
// 			stopCh := make(chan struct{})
// 			notify, err := w.Watch("foo", stopCh)
// 			So(err, ShouldBeNil)
// 			So(notify, ShouldNotBeNil)
// 			So(w.watches, ShouldHaveLength, 1)

// 			stopCh <- struct{}{}

// 			w.Close()
// 			So(w.watches, ShouldBeEmpty)
// 		})

// 		Convey("Watch a directory", func() {
// 			stopCh := make(chan struct{})
// 			notify, err := w.WatchTree("foo", stopCh)
// 			So(err, ShouldBeNil)
// 			So(notify, ShouldNotBeNil)
// 			So(w.watches, ShouldHaveLength, 1)

// 			stopCh <- struct{}{}

// 			w.Close()
// 			So(w.watches, ShouldBeEmpty)
// 		})
// 	})
// }
