package watchers

import (
	"crypto/md5"
	"io"
	"log"
	"os"
	"path"
	"sync"
	"time"

	"bytes"

	"io/ioutil"

	"github.com/signalfx/neo-agent/utils"
)

type entry struct {
	callback     func() error
	hash         []byte
	modifiedTime time.Time
}

// PollingWatcher watches for changes to files by polling
type PollingWatcher struct {
	files    map[string]*entry
	dirs     map[string]map[string]*entry
	delay    time.Duration
	stopChan chan struct{}
	cb       func(changed []string) error
	mutex    sync.Mutex
	wg       sync.WaitGroup
}

// reset is called when a file is not found or unable to be read during polling.
// Returns true if the entry was reset or false if it was already zeroed out.
func (entry *entry) reset() bool {
	changed := false

	if entry.hash != nil {
		entry.hash = nil
		changed = true
	}

	if !entry.modifiedTime.IsZero() {
		entry.modifiedTime = time.Time{}
		changed = true
	}

	return changed
}

// NewPollingWatcher creates a new polling watcher instance
func NewPollingWatcher(cb func(changed []string) error, delay time.Duration) *PollingWatcher {
	// Make it buffered by one so sending will never be blocked in case polling
	// stopped before send to stopChan.
	w := &PollingWatcher{files: map[string]*entry{}, dirs: map[string]map[string]*entry{},
		cb: cb, delay: delay, stopChan: make(chan struct{}, 1)}
	return w
}

// Start begins the polling process in the background
func (w *PollingWatcher) Start() {
	w.wg.Add(1)
	ticker := time.NewTicker(w.delay)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-w.stopChan:
				w.wg.Done()
				return
			case <-ticker.C:
				w.poll()
			}
		}
	}()
}

// watchDirs watches directories for changes
func (w *PollingWatcher) watchDirs(dirs ...string) {
	dirSet := utils.StringSliceToMap(dirs)

	for dir := range dirSet {
		if _, ok := w.dirs[dir]; !ok {
			// Added
			files := map[string]*entry{}
			dircon, err := ioutil.ReadDir(dir)

			if err == nil {
				for _, file := range dircon {
					e := &entry{}
					read(path.Join(dir, file.Name()), e)
					files[file.Name()] = e
				}
				w.dirs[dir] = files
			} else {
				// Nothing to do about it, just initializes to empty.
				log.Printf("watch directory didn't exist: %s", err)
				w.dirs[dir] = nil
			}

		}
	}

	for dir := range w.dirs {
		if !dirSet[dir] {
			// Directory removed.
			delete(w.dirs, dir)
		}
	}
}

// Watch watches files and directories for changes
func (w *PollingWatcher) Watch(dirs []string, files []string) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.watchDirs(dirs...)
	w.watchFiles(files...)
}

// watchFiles sets the list of paths to poll
func (w *PollingWatcher) watchFiles(files ...string) {
	fileSet := utils.StringSliceToMap(files)

	for file := range fileSet {
		if _, ok := w.files[file]; !ok {
			// Added, initialize hash.
			e := &entry{}
			read(file, e)
			w.files[file] = e
		}
	}

	for file := range w.files {
		if !fileSet[file] {
			// File removed.
			delete(w.files, file)
		}
	}
}

// Close stops the watcher
func (w *PollingWatcher) Close() {
	w.stopChan <- struct{}{}
	// Wait for poll to stop.
	w.wg.Wait()
}

// read returns whether the file was changed or not (and updates entry state)
func read(fileName string, entry *entry) bool {
	stat, err := os.Stat(fileName)
	if err != nil {
		return entry.reset()
	}

	if stat.IsDir() {
		return entry.reset()
	}

	// If the file modified time is equal to or before our recorded modified
	// skip further checks. This also accounts for the initial case where
	// modifiedTime is initialized to its zero value.
	if !stat.ModTime().After(entry.modifiedTime) {
		return false
	}

	ck := md5.New()

	f, err := os.OpenFile(fileName, os.O_RDONLY, 0)
	if err != nil {
		// File was possibly removed between stat and open.
		return entry.reset()
	}
	defer f.Close()

	// Compute file hash.
	if _, err := io.Copy(ck, f); err != nil {
		log.Printf("failed to copy file %s to buffer: %s", fileName, err)
		return entry.reset()
	}

	// Don't set modified time until *after* the hash has been successfully
	// computed. In the case where a file is unreadable (e.g. permission denied)
	// we don't want to set modified time earlier then reset. In that case reset
	// would always return true, signalling that the file has changed when
	// really we just couldn't read it.
	entry.modifiedTime = stat.ModTime()

	newHash := ck.Sum(nil)
	if !bytes.Equal(entry.hash, newHash) {
		// If the file hashes differ the file is changed.
		entry.hash = newHash
		return true
	}

	return false
}

// poll checks for file changes
func (w *PollingWatcher) poll() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	var changed []string

	for dirName, entries := range w.dirs {
		files, err := ioutil.ReadDir(dirName)

		if err != nil {
			// Directory doesn't exist or can't be read.

			if entries != nil {
				// Directory was just removed, report as change and null out.
				changed = append(changed, dirName)
				w.dirs[dirName] = nil
			}
			continue
		}

		// Index directory file names.
		fileSet := map[string]bool{}
		for _, file := range files {
			fileSet[file.Name()] = true
		}

		if entries == nil {
			entries = map[string]*entry{}
			w.dirs[dirName] = entries
			changed = append(changed, dirName)
		}

		// Files added to directory.
		for file := range fileSet {
			dirPath := path.Join(dirName, file)

			if _, ok := entries[file]; ok {
				// File entry already present, update.
				if read(dirPath, entries[file]) {
					changed = append(changed, dirPath)
				}
			} else {
				// File added.
				changed = append(changed, dirPath)
				e := &entry{}
				entries[file] = e
				read(dirPath, e)
			}
		}

		// Files removed from directory.
		for file := range entries {
			if _, ok := fileSet[file]; !ok {
				changed = append(changed, path.Join(dirName, file))
				delete(entries, file)
			}
		}
	}

	// Check watched files for changes.
	for fileName, entry := range w.files {
		if read(fileName, entry) {
			changed = append(changed, fileName)
		}
	}

	if len(changed) > 0 {
		if err := w.cb(changed); err != nil {
			log.Printf("error during change callback of files %s: %s", changed, err)
		}
	}
}
