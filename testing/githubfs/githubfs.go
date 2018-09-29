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

// Package githubfs provides an afero FS that's backed by github.com.
// Useful for testing against master for a repository, especially in cron.
package githubfs // import "barista.run/testing/githubfs"

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/afero"
)

var root = "https://raw.githubusercontent.com"

// Fs represents an in-memory filesystem backed by GitHub.
type Fs struct {
	// readonly view into the backing fs.
	afero.Fs
	// backing mem-mapped fs.
	backingFs afero.Fs
}

// New constructs an instance of GitHubFs.
// This is a readonly Fs, and calls to Read/Stat will fetch the file from github,
// before returning a readonly view into the newly fetched files.
func New() afero.Fs {
	// Using a backing mem-map fs means we only need to handle fetching files
	// from GitHub, then we can just dump contents and chtimes and delegate all
	// calls to the backing filesystem, making this Fs much simpler.
	backingFs := afero.NewMemMapFs()
	// Although the external view of this Fs is readonly, we need a reference
	// to the actual mem-map Fs so that we can write file contents.
	return &Fs{afero.NewReadOnlyFs(backingFs), backingFs}
}

func (f *Fs) fetch(name string) error {
	file, err := f.backingFs.Create(name)
	if err != nil {
		return err
	}
	r, err := http.Get(fmt.Sprintf("%s/%s", root, strings.TrimPrefix(name, "/")))
	if err != nil {
		return err
	}
	if r.StatusCode != 200 {
		return errors.New(r.Status)
	}
	defer r.Body.Close()
	_, err = io.Copy(file, r.Body)
	modTime := r.Header.Get("Last-Modified")
	if parsed, err := http.ParseTime(modTime); err == nil {
		local := parsed.Local()
		f.backingFs.Chtimes(name, local, local)
	}
	return err
}

// Open opens a file, returning it or an error, if any happens.
func (f *Fs) Open(name string) (afero.File, error) {
	err := f.fetch(name)
	if err != nil {
		return nil, err
	}
	return f.Fs.Open(name)
}

// OpenFile opens a file using the given flags and the given mode.
func (f *Fs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	err := f.fetch(name)
	if err != nil {
		return nil, err
	}
	return f.Fs.OpenFile(name, flag, perm)
}

// Stat returns a FileInfo describing the named file, or an error, if any
// happens.
func (f *Fs) Stat(name string) (os.FileInfo, error) {
	err := f.fetch(name)
	if err != nil {
		return nil, err
	}
	return f.Fs.Stat(name)
}

// Name returns the name of this FileSystem
func (f *Fs) Name() string {
	return fmt.Sprintf("GitHubFS/backed by %s", f.Fs.Name())
}
