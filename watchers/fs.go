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

	"github.com/docker/libkv/store"
)

type base struct {
	stopCh <-chan struct{}
}

type entry struct {
	fileName     string
	hash         []byte
	modifiedTime time.Time
}

type dirState struct {
	base
	notifyCh chan<- []*store.KVPair
	dirName  string
	files    map[string]*entry
}

type watcher interface {
	check(first bool)
}

type fileState struct {
	base
	notifyCh chan<- *store.KVPair
	entry    *entry
}

func (f *fileState) check(first bool) {
	if pair, changed := read(f.entry); changed || first {
		f.notifyCh <- pair
	}
}

// Pairs is an array of KVPairs
type Pairs []*store.KVPair

func pair(key string, value []byte) *store.KVPair {
	return &store.KVPair{Key: key, Value: value, LastIndex: 0}
}

func (d *dirState) check(first bool) {
	var changed bool

	files, err := ioutil.ReadDir(d.dirName)

	if err != nil {
		// Directory doesn't exist or can't be read.

		if d.files != nil {
			// Directory was just removed, report as change and null out.
			d.files = nil
			changed = true
		}
		if changed || first {
			d.notifyCh <- Pairs{}
		}
		return
	}

	// Index directory file names.
	fileSet := map[string]bool{}
	for _, file := range files {
		fileSet[file.Name()] = true
	}

	if d.files == nil {
		d.files = map[string]*entry{}
		changed = true
	}

	// Files added to directory.
	for file := range fileSet {
		dirPath := path.Join(d.dirName, file)

		if _, ok := d.files[file]; ok {
			// File entry already present, update.
			_, readChanged := read(d.files[file])
			if readChanged {
				changed = true
			}
		} else {
			// File added.
			changed = true
			e := &entry{fileName: dirPath}
			d.files[file] = e
			read(e)
		}
	}

	// Files removed from directory.
	for file := range d.files {
		if _, ok := fileSet[file]; !ok {
			changed = true
			delete(d.files, file)
		}
	}

	if changed {
		pairs := Pairs{}
		for _, f := range files {
			dirPath := path.Join(d.dirName, f.Name())

			if f.IsDir() {
				pairs = append(pairs, pair(dirPath, nil))
			} else {
				data, err := ioutil.ReadFile(dirPath)
				if err != nil {
					log.Printf("failed to read %s: %s", dirPath, err)
					continue
				}
				pairs = append(pairs, pair(dirPath, data))
			}
		}
		d.notifyCh <- pairs
	}
}

// PollingWatcher watches for changes to files by polling
type PollingWatcher struct {
	watches  map[watcher]bool
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
func NewPollingWatcher(delay time.Duration) *PollingWatcher {
	// Make it buffered by one so sending will never be blocked in case polling
	// stopped before send to stopChan.
	w := &PollingWatcher{delay: delay, stopChan: make(chan struct{}), watches: map[watcher]bool{}}
	return w
}

// Start polling
func (w *PollingWatcher) Start() {
	ticker := time.NewTicker(w.delay)
	w.wg.Add(1)

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

// WatchTree watches directories for changes
func (w *PollingWatcher) WatchTree(dir string, stopCh <-chan struct{}) (<-chan []*store.KVPair, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	notifyCh := make(chan []*store.KVPair, 1)
	dw := &dirState{dirName: dir, base: base{stopCh: stopCh}, notifyCh: notifyCh}
	dw.check(true)

	w.watches[dw] = true

	go func() {
		select {
		case <-stopCh:
			w.mutex.Lock()
			close(notifyCh)
			delete(w.watches, dw)
			w.mutex.Unlock()
		}
	}()

	return notifyCh, nil
}

// Watch sets the list of paths to poll
func (w *PollingWatcher) Watch(file string, stopCh <-chan struct{}) (<-chan *store.KVPair, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	notifyCh := make(chan *store.KVPair, 1)
	fw := &fileState{entry: &entry{fileName: file}, base: base{stopCh: stopCh}, notifyCh: notifyCh}
	fw.check(true)

	w.watches[fw] = true

	go func() {
		select {
		case <-stopCh:
			w.mutex.Lock()
			close(notifyCh)
			delete(w.watches, fw)
			w.mutex.Unlock()
		}
	}()

	return notifyCh, nil
}

// Close stops the watcher
func (w *PollingWatcher) Close() {
	// Waits for polling to finish.
	w.stopChan <- struct{}{}
	w.wg.Wait()
}

// read returns whether the file was changed or not (and updates entry state).
// If it has been updated it returns the file contents.
func read(entry *entry) (*store.KVPair, bool) {
	fileName := entry.fileName

	stat, err := os.Stat(fileName)
	if err != nil {
		return pair(fileName, nil), entry.reset()
	}

	if stat.IsDir() {
		return pair(fileName, nil), entry.reset()
	}

	// If the file modified time is equal to or before our recorded modified
	// skip further checks. This also accounts for the initial case where
	// modifiedTime is initialized to its zero value.
	if !stat.ModTime().After(entry.modifiedTime) {
		return pair(fileName, nil), false
	}

	ck := md5.New()

	contents, err := ioutil.ReadFile(fileName)
	if err != nil {
		// File was possibly removed between stat and open.
		return pair(fileName, nil), entry.reset()
	}

	reader := bytes.NewReader(contents)

	// Compute file hash.
	if _, err := io.Copy(ck, reader); err != nil {
		log.Printf("failed to copy file %s to buffer: %s", fileName, err)
		return pair(fileName, nil), entry.reset()
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
		return pair(fileName, contents), true
	}

	return pair(fileName, nil), false
}

// poll checks for file changes
func (w *PollingWatcher) poll() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	for watch := range w.watches {
		watch.check(false)
	}
}
