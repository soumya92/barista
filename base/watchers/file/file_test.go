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

package file

import (
	"io/ioutil"
	"os"
	"path"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func testDir(t *testing.T) string {
	dir, err := ioutil.TempDir("", "fsnotify")
	if err != nil {
		t.Fatalf("failed to create test directory: %s", err)
	}
	return dir
}

func assertNotified(t *testing.T, ch <-chan struct{}, formatAndArgs ...interface{}) {
	select {
	case <-ch:
		// test passed.
	case <-time.After(time.Second):
		require.Fail(t, "Was not notified", formatAndArgs...)
	}
	deadline := time.After(5 * time.Millisecond)
	for {
		select {
		case <-ch:
		case <-deadline:
			return
		}
	}
}

func assertNotNotified(t *testing.T, ch <-chan struct{}, formatAndArgs ...interface{}) {
	select {
	case <-time.After(10 * time.Millisecond):
		// test passed.
	case <-ch:
		require.Fail(t, "Unexpectedly notified", formatAndArgs...)
	}
}

func waitForStart(w *Watcher) {
	for !atomic.CompareAndSwapInt32(&w.started, 1, 0) {
		time.Sleep(10 * time.Millisecond)
	}
}

func TestWatchOnExistingFile(t *testing.T) {
	tempDir := testDir(t)
	defer os.RemoveAll(tempDir)
	tmpFile := path.Join(tempDir, "somefile")
	ioutil.WriteFile(tmpFile, []byte(`foo`), 0644)

	w := Watch(tmpFile)
	defer w.Unsubscribe()
	waitForStart(w)
	assertNotNotified(t, w.Updates, "On start")

	ioutil.WriteFile(tmpFile, []byte(`bar`), 0644)
	assertNotified(t, w.Updates, "On write")
}

func TestDeleteAndRecreate(t *testing.T) {
	tempDir := testDir(t)
	defer os.RemoveAll(tempDir)
	tmpFile := path.Join(tempDir, "foo")
	ioutil.WriteFile(tmpFile, []byte(`foo`), 0644)

	w := Watch(tmpFile)
	defer w.Unsubscribe()
	waitForStart(w)

	os.Remove(tmpFile)
	assertNotified(t, w.Updates, "On delete")

	ioutil.WriteFile(tmpFile, []byte(`foo`), 0644)
	assertNotified(t, w.Updates, "On recreate")
}

func TestSubdirectories(t *testing.T) {
	tempDir := testDir(t)
	defer os.RemoveAll(tempDir)
	subdir := path.Join(tempDir, "foo", "bar", "baz")
	target := path.Join(subdir, "afile")
	os.MkdirAll(subdir, 0755)

	w := Watch(target)
	defer w.Unsubscribe()
	waitForStart(w)
	assertNotNotified(t, w.Updates, "on start with non-existent file")

	ioutil.WriteFile(target, []byte(`foo`), 0644)
	assertNotified(t, w.Updates, "on file modification")

	os.RemoveAll(path.Join(tempDir, "foo"))
	assertNotified(t, w.Updates, "on parent deletion")

	ioutil.WriteFile(path.Join(tempDir, "notfoo"), []byte(`bar`), 0644)
	os.MkdirAll(path.Join(tempDir, "baz", "etc"), 0755)
	assertNotNotified(t, w.Updates, "ignores creations in currently watched dir")

	os.MkdirAll(subdir, 0755)
	assertNotNotified(t, w.Updates, "on creation of parent dir")

	ioutil.WriteFile(target, []byte(`foo`), 0644)
	assertNotified(t, w.Updates, "on file modification")

	os.Remove(target)
	assertNotified(t, w.Updates, "on file deletion")

	os.Remove(subdir)
	assertNotNotified(t, w.Updates, "on parent deletion after file is gone")

	os.MkdirAll(target, 0755)
	assertNotified(t, w.Updates, "on full path creation")
}

func TestStress(t *testing.T) {
	tempDir := testDir(t)
	defer os.RemoveAll(tempDir)
	subdir := path.Join(tempDir, "foo", "bar", "baz")
	target := path.Join(subdir, "afile")

	w := Watch(target)
	defer w.Unsubscribe()
	waitForStart(w)

	for i := 0; i < 1000; i++ {
		os.MkdirAll(subdir, 0755)
		ioutil.WriteFile(target, []byte(`xx`), 0644)
		os.RemoveAll(path.Join(tempDir, "foo"))

		os.MkdirAll(path.Join(tempDir, "foo", "bar"), 0755)
		os.Remove(path.Join(tempDir, "foo", "bar"))
		os.MkdirAll(target, 0755)
		os.RemoveAll(path.Join(tempDir, "foo"))
	}

	done := false
	for !done {
		select {
		case <-w.Updates:
		default:
			done = true
			break
		}
	}

	os.MkdirAll(subdir, 0755)
	ioutil.WriteFile(target, []byte(`xx`), 0644)
	assertNotified(t, w.Updates, "after stress test")
}

func TestErrors(t *testing.T) {
	tempDir := testDir(t)
	defer os.RemoveAll(tempDir)
	tmpFile := path.Join(tempDir, "somefile")
	ioutil.WriteFile(tmpFile, []byte(`foo`), 0644)

	w := Watch(path.Join(tmpFile, "/dir/under/file"))
	defer w.Unsubscribe()
	assertNotNotified(t, w.Updates, "On start with error")
	select {
	case <-w.Errors:
		// test passed.
	case <-time.After(time.Second):
		require.Fail(t, "Expected an error", "on start")
	}

	subdir := path.Join(tempDir, "foo", "bar", "baz")
	os.MkdirAll(subdir, 0755)
	w = Watch(subdir)
	defer w.Unsubscribe()
	assertNotNotified(t, w.Updates, "On start")
	waitForStart(w)

	os.RemoveAll(path.Join(tempDir, "foo"))
	assertNotified(t, w.Updates, "On parent deletion")
	os.Create(path.Join(tempDir, "foo"))
	assertNotNotified(t, w.Updates, "On error")
	select {
	case <-w.Errors:
	// test passed.
	case <-time.After(time.Second):
		require.Fail(t, "Expected an error", "on start")
	}
}
