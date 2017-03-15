package watchers

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	. "github.com/signalfx/neo-agent/neotest"
	. "github.com/smartystreets/goconvey/convey"
)

func Test_PollingWatcher_Poll(t *testing.T) {
	Convey("Given a cloned version of testdata", t, func() {
		_, cleanup := CloneTestData(t)
		changed := [][]string{}

		cb := func(files []string) error {
			changed = append(changed, files)
			return nil
		}

		w := NewPollingWatcher(cb, 50*time.Millisecond)

		Reset(func() {
			cleanup()
		})

		Convey("Give a watched directory", func() {
			Convey("A file is added", nil)
			Convey("A file is removed", nil)
			Convey("A file is modified", nil)

			Convey("A symlink is added", nil)
			Convey("A symlink is removed", nil)
			Convey("A symlink is modified", nil)

			Convey("All files are removed", nil)
		})

		Convey("Given a watched double symlink", func() {
			w.Watch("symlinks/link1")

			Convey("The link destination contents are modified", nil)
			Convey("The link destination content is unmodified", nil)
			Convey("The destination file is removed", nil)
		})

		Convey("Given a watched file that doesn't exist", func() {
			w.Watch("dir/file-does-not-exist")
			So(changed, ShouldHaveLength, 0)
		})

		Convey("Given a watched file", func() {
			w.Watch("dir/file1")

			Convey("The file is truncated with new contents", func() {
				Must(t, ioutil.WriteFile("dir/file1", []byte("changed"), 0644))
				w.poll()

				So(changed, ShouldResemble, [][]string{
					[]string{"dir/file1"},
				})
			})

			Convey("The file is modified once but polled twice", func() {
				Must(t, ioutil.WriteFile("dir/file1", []byte("changed"), 0644))
				w.poll()
				w.poll()
				So(changed, ShouldHaveLength, 1)
			})

			Convey("The file is unmodified", func() {
				w.poll()

				So(changed, ShouldHaveLength, 0)
			})

			Convey("The file is removed", func() {
				Must(t, os.Remove("dir/file1"))
				w.poll()

				So(changed, ShouldResemble, [][]string{
					[]string{"dir/file1"},
				})

				Convey("Then readded with different contents", func() {
					Must(t, ioutil.WriteFile("dir/file1", []byte("new contents after remove"), 0644))
					w.poll()

					So(changed, ShouldHaveLength, 2)
				})
				Convey("Then readded with same contents", func() {
					w.poll()
					Must(t, ioutil.WriteFile("dir/file1", []byte("1\n"), 0644))
					w.poll()

					So(changed, ShouldHaveLength, 2)
				})
			})
		})
	})
}

func Test_PollingWatcher_Watch(t *testing.T) {
	Convey("Given a watcher instance", t, func() {
		w := NewPollingWatcher(func(changed []string) error { return nil }, 1*time.Second)

		Reset(func() {
			w.Close()
		})

		Convey("Given some files already being watched", func() {
			w.Watch("foo", "bar")

			So(w.files, ShouldHaveLength, 2)

			Convey("Unwatching a file should remove monitor", func() {
				w.Watch("foo")

				So(w.files, ShouldHaveLength, 1)
				So(w.files, ShouldContainKey, "foo")
			})

			Convey("Unwatching all files should result in zero monitors", func() {
				w.Watch()

				So(w.files, ShouldHaveLength, 0)
			})
		})

		Convey("A watched file should be monitored", func() {
			w.Watch("foo")

			So(w.files, ShouldHaveLength, 1)
		})

		Convey("File monitored but not watched should be removed", nil)
	})
}
