// Copyright 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package file uses the fsnotify library to watch for changes to files.
package file // import "barista.run/base/watchers/file"

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	"barista.run/base/notifier"
	l "barista.run/logging"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches for changes to a single named file or directory. It notifies
// the Updates chan on any changes to the watched file, while also handling
// parts of the path hierarchy to the file being removed and recreated.
type Watcher struct {
	Updates <-chan struct{}
	Errors  <-chan error

	fswatcher *fsnotify.Watcher
	// To account for an entire tree being removed, we store successive
	// parent dirs, and keep track of which level we're currently watching.
	// If the current level is removed, we move up a level. If the level
	// below is created, we move down to it.
	hierarchy []string
	filename  string
	// We only notify of a change, we do not spell out the change. This is
	// so that modules can (optionally) defer change detection while hidden,
	// reducing fs calls, and also because for most modules it is irrelevant
	// what the change was, they will get the state by reading the file.
	notifyFn func()
	errorCh  chan error
	done     int32 // atomic bool.
	// For synchronisation.
	started chan struct{}
}

// Unsubscribe stops listening for updates and frees any resources used.
func (w *Watcher) Unsubscribe() {
	if atomic.CompareAndSwapInt32(&w.done, 0, 1) {
		l.Fine("%s done", l.ID(w))
		w.fswatcher.Close()
	}
}

func (w *Watcher) watchLoop() {
	restarted := false
	for {
		l.Fine("%s (re)starting watches", l.ID(w))
		err := w.tryWatch(restarted)
		if err != nil {
			w.Unsubscribe()
			w.errorCh <- err
			return
		}
		if atomic.LoadInt32(&w.done) > 0 {
			return
		}
		restarted = true
	}
}

func (w *Watcher) markStarted() {
	select {
	case w.started <- struct{}{}:
	default:
	}
}

func (w *Watcher) tryWatch(restarted bool) error {
	currentLvl := -1
	for lvl, p := range w.hierarchy {
		err := w.fswatcher.Add(p)
		if err == nil {
			currentLvl = lvl
			l.Fine("%s: Watch added for %s", l.ID(w), p)
			break
		} else if !os.IsNotExist(err) {
			l.Log("%s: %v", l.ID(w), err)
			w.markStarted()
			return err
		}
	}
	if currentLvl == -1 {
		w.markStarted()
		return fmt.Errorf("Unable to add file watch for any of %v", w.hierarchy)
	}
	if restarted {
		if _, e := os.Stat(w.filename); e == nil {
			w.notifyFn()
		}
	}
	w.markStarted()
	for {
		select {
		case event, ok := <-w.fswatcher.Events:
			if !ok {
				return nil
			}
			l.Fine("%s notified: %s", l.ID(w), event)
			if event.Name == w.filename {
				w.notifyFn()
			}
			if event.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
				// TODO: Handle parent being moved. For most modules, this is not
				// a common case. (e.g. the files they watch will be things like
				// ssh control files, or run/pid files, where a parent is not
				// likely to be moved.)
				return nil
			}
			if currentLvl == 0 {
				continue
			}
			if event.Op&fsnotify.Create == 0 {
				continue
			}
			if event.Name != w.hierarchy[currentLvl-1] {
				continue
			}
			newLvl := currentLvl - 1
			for newLvl >= 0 {
				err := w.fswatcher.Add(w.hierarchy[newLvl])
				if err != nil {
					if !os.IsNotExist(err) {
						l.Log("%s: %v", l.ID(w), err)
						return err
					}
					break
				}
				w.fswatcher.Remove(w.hierarchy[currentLvl])
				l.Fine("%s: Watch moved from %s -> %s",
					l.ID(w), w.hierarchy[currentLvl], w.hierarchy[newLvl])
				currentLvl = newLvl
				newLvl--
			}
			if _, e := os.Stat(w.filename); e == nil {
				w.notifyFn()
			}
		case err, ok := <-w.fswatcher.Errors:
			if !ok {
				return nil
			}
			l.Log("%s: %v", l.ID(w), err)
			return err
		}
	}
}

// Watch creates a new file watcher for the given filename.
func Watch(filename string) *Watcher {
	w := &Watcher{filename: filename}
	l.Labelf(w, filename)
	w.errorCh = make(chan error, 1)
	w.Errors = w.errorCh
	watcher, err := fsnotify.NewWatcher()
	w.fswatcher = watcher
	if err != nil {
		w.errorCh <- err
		return w
	}
	for p := filepath.Dir(filename); ; p = filepath.Dir(p) {
		w.hierarchy = append(w.hierarchy, p)
		if p == "/" {
			break
		}
	}
	w.notifyFn, w.Updates = notifier.New()
	w.started = make(chan struct{}, 1)
	l.Register(w, "Updates", "Errors")
	go w.watchLoop()
	<-w.started
	return w
}
