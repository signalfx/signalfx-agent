package watchers

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"sort"

	. "github.com/signalfx/neo-agent/neotest"
	. "github.com/smartystreets/goconvey/convey"
)

func Test_PollingwatchFileser_Poll(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)

	Convey("Given a cloned version of testdata", t, func() {
		_, cleanup := CloneTestData(t)
		changed := [][]string{}

		// Create empty directory since we can't check it into git.
		Must(t, os.Mkdir("dir/empty-dir", 0))

		cb := func(files []string) error {
			sort.Strings(files)
			changed = append(changed, files)
			return nil
		}

		w := NewPollingWatcher(cb, 50*time.Millisecond)

		Reset(func() {
			cleanup()
		})

		Convey("Given a watched empty directory", func() {
			w.watchDirs("dir/empty-dir")
			Must(t, os.Remove("dir/empty-dir"))
			w.poll()

			So(changed, ShouldResemble, [][]string{
				[]string{"dir/empty-dir"},
			})
		})

		Convey("Give a watched directory", func() {
			w.watchDirs("dir")

			Convey("A directory is added", func() {
				Must(t, os.Mkdir("dir/new-dir", 0644))
				w.poll()

				So(changed, ShouldResemble, [][]string{
					[]string{"dir/new-dir"},
				})

				Convey("A directory is removed", func() {
					Must(t, os.Remove("dir/new-dir"))
					w.poll()

					So(changed, ShouldResemble, [][]string{
						[]string{"dir/new-dir"},
						[]string{"dir/new-dir"},
					})
				})
			})

			Convey("A file is added", func() {
				Must(t, ioutil.WriteFile("dir/new-file", []byte("new file"), 0644))
				w.poll()

				So(changed, ShouldResemble, [][]string{
					[]string{"dir/new-file"},
				})
			})
			Convey("A file is removed", func() {
				Must(t, os.Remove("dir/file1"))
				w.poll()

				So(changed, ShouldResemble, [][]string{
					[]string{"dir/file1"},
				})

				Convey("Then added back", func() {
					Must(t, ioutil.WriteFile("dir/file1", []byte("new file"), 0644))
					w.poll()

					So(changed, ShouldResemble, [][]string{
						[]string{"dir/file1"},
						[]string{"dir/file1"},
					})
				})
			})
			Convey("A file is modified", func() {
				Must(t, ioutil.WriteFile("dir/file1", []byte("changed"), 0644))
				// Have to make file change time in the future because at least
				// on Mac the modified resolution is 1 second.
				Must(t, os.Chtimes("dir/file1", future, future))

				w.poll()

				So(changed, ShouldResemble, [][]string{
					[]string{"dir/file1"},
				})
			})
			Convey("Multiple files are modified", func() {
				for _, file := range []string{"dir/file1", "dir/file2"} {
					Must(t, ioutil.WriteFile(file, []byte("changed"), 0644))
					// Have to make file change time in the future because at least
					// on Mac the modified resolution is 1 second.
					Must(t, os.Chtimes(file, future, future))
				}
				w.poll()

				So(changed, ShouldResemble, [][]string{
					[]string{"dir/file1", "dir/file2"},
				})
			})

			Convey("A symlink is added", func() {
				Must(t, os.Symlink("file1", "dir/symlink"))
				w.poll()

				So(changed, ShouldResemble, [][]string{
					[]string{"dir/symlink"},
				})

				changed = nil

				Convey("A symlink is removed", func() {
					Must(t, os.Remove("dir/symlink"))
					w.poll()

					So(changed, ShouldResemble, [][]string{
						[]string{"dir/symlink"},
					})
				})

				Convey("A symlinks contents is modified", func() {
					Must(t, ioutil.WriteFile("dir/file1", []byte("changed"), 0644))
					// Have to make file change time in the future because at least
					// on Mac the modified resolution is 1 second.
					Must(t, os.Chtimes("dir/file1", future, future))

					w.poll()

					So(changed, ShouldResemble, [][]string{
						[]string{"dir/file1", "dir/symlink"},
					})
				})
			})

			Convey("All files are removed", func() {
				// Remove everything in the directory except the directory itself.
				if files, err := ioutil.ReadDir("dir"); err != nil {
					t.Fatal(err)
				} else {
					for _, file := range files {
						Must(t, os.RemoveAll(path.Join("dir", file.Name())))
					}
				}
				w.poll()

				So(changed, ShouldResemble, [][]string{
					[]string{"dir/empty-dir", "dir/file1", "dir/file2", "dir/subdir"},
				})

				changed = nil

				Convey("A file is added back", func() {
					Must(t, ioutil.WriteFile("dir/file1", []byte("changed"), 0644))
					// Have to make file change time in the future because at least
					// on Mac the modified resolution is 1 second.
					Must(t, os.Chtimes("dir/file1", future, future))

					w.poll()

					So(changed, ShouldResemble, [][]string{
						[]string{"dir/file1"},
					})
				})
			})
		})

		Convey("Given a watched file that doesn't exist", func() {
			w.watchFiles("dir/file-does-not-exist")
			So(changed, ShouldBeEmpty)
		})

		Convey("Given a watched file modified in the past", func() {
			d, err := time.Parse(time.UnixDate, "Mon Jan 2 15:04:05 MST 2006")
			if err != nil {
				t.Fatal(err)
			}
			Must(t, os.Chtimes("dir/file1", time.Now(), d))
			w.watchFiles("dir/file1")

			Convey("And the contents have changed", func() {
				Must(t, ioutil.WriteFile("dir/file1", []byte("changed"), 0644))

				Convey("Modification time before file modification at time of watch", func() {
					// Set mtime to an old date.
					Must(t, os.Chtimes("dir/file1", time.Now(), d))
					w.poll()
					So(changed, ShouldBeEmpty)
				})

				Convey("Modification time is in the future", func() {
					Must(t, os.Chtimes("dir/file1", time.Now(), time.Now().Add(1*time.Hour)))
					w.poll()
					So(changed, ShouldHaveLength, 1)
				})

				Convey("Modification time equal to last polled time", func() {
					Must(t, os.Chtimes("dir/file1", time.Now(), w.files["dir/file1"].modifiedTime))
					w.poll()
					So(changed, ShouldBeEmpty)
				})

				Convey("The file is considered changed", func() {
					w.poll()

					So(changed, ShouldResemble, [][]string{
						[]string{"dir/file1"},
					})
				})
			})

			Convey("The file modified time is changed but not its contents", func() {
				Must(t, os.Chtimes("dir/file1", time.Now(), time.Now().Add(1*time.Hour)))
				So(changed, ShouldBeEmpty)
			})

			Convey("The file is modified once but polled twice", func() {
				Must(t, ioutil.WriteFile("dir/file1", []byte("changed"), 0644))
				w.poll()
				w.poll()
				So(changed, ShouldHaveLength, 1)
			})

			Convey("The file is unmodified", func() {
				w.poll()

				So(changed, ShouldBeEmpty)
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

func Test_PollingWatcher_watchFiles(t *testing.T) {
	Convey("Given a watcher instance", t, func() {
		w := NewPollingWatcher(func(changed []string) error { return nil }, 1*time.Second)

		Reset(func() {
			w.Close()
		})

		// Test watching files.
		Convey("Given some files already being watched", func() {
			w.watchFiles("foo", "bar")

			So(w.files, ShouldHaveLength, 2)
			So(w.dirs, ShouldBeEmpty)

			Convey("Unwatching a file should remove monitor", func() {
				w.watchFiles("foo")

				So(w.files, ShouldHaveLength, 1)
				So(w.files, ShouldContainKey, "foo")
			})

			Convey("Unwatching all files should result in zero monitors", func() {
				w.watchFiles()

				So(w.files, ShouldBeEmpty)
			})
		})

		Convey("A watched file should be monitored", func() {
			w.watchFiles("foo")

			So(w.files, ShouldHaveLength, 1)
		})

		// Test watching directories.
		Convey("Watch a directory that exists", func() {
			w.watchDirs("dir")

			So(w.files, ShouldBeEmpty)
			So(w.dirs, ShouldHaveLength, 1)

			Convey("Unwatch a directory", func() {
				w.watchDirs()
				So(w.files, ShouldBeEmpty)
				So(w.dirs, ShouldBeEmpty)
			})

			Convey("Watch same directory", func() {
				w.watchDirs("dir")
				So(w.files, ShouldBeEmpty)
				So(w.dirs, ShouldHaveLength, 1)
			})
		})
	})
}
