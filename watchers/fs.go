package watchers

import (
	"crypto/md5"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"bytes"

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
	delay    time.Duration
	stopChan chan struct{}
	cb       func(changed []string) error
	mutex    sync.Mutex
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
	w := &PollingWatcher{files: map[string]*entry{}, cb: cb, delay: delay, stopChan: make(chan struct{}, 1)}
	return w
}

// Start begins the polling process in the background
func (w *PollingWatcher) Start() {
	ticker := time.NewTicker(w.delay)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-w.stopChan:
				return
			case <-ticker.C:
				w.poll()
			}
		}
	}()
}

// Watch sets the list of files to poll
func (w *PollingWatcher) Watch(files ...string) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

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
			// Removed
			delete(w.files, file)
		}
	}
}

// Close stops the watcher
func (w *PollingWatcher) Close() {
	w.stopChan <- struct{}{}
}

// read returns whether the file was changed or not (and updates entry state)
func read(fileName string, entry *entry) bool {
	stat, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		return entry.reset()
	}

	if stat.IsDir() {
		panic("unimplemented")
	} else {
		// If the file modified time is equal to or before our recorded modified
		// skip further checks. This also accounts for the initial case where
		// modifiedTime is initialized to its zero value.
		if !stat.ModTime().After(entry.modifiedTime) {
			return false
		}

		entry.modifiedTime = stat.ModTime()

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

		newHash := ck.Sum(nil)
		if !bytes.Equal(entry.hash, newHash) {
			// If the file hashes differ the file is changed.
			entry.hash = newHash
			return true
		}
	}

	return false
}

// poll checks for file changes
func (w *PollingWatcher) poll() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	var changed []string

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
