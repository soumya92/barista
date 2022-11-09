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

// +build baristadebuglog

package logging // import "barista.run/logging"

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
)

func trimSuffix(s, suffix string) (result string, trimmed bool) {
	return strings.TrimSuffix(s, suffix), strings.HasSuffix(s, suffix)
}

func trimPrefix(s, prefix string) (result string, trimmed bool) {
	return strings.TrimPrefix(s, prefix), strings.HasPrefix(s, prefix)
}

func construct() {
	pc, file, _, ok := runtime.Caller(0)
	if !ok {
		return
	}
	fnName := runtime.FuncForPC(pc).Name()
	if pkg, ok := trimSuffix(fnName, "/logging.construct"); ok {
		baristaPkg = pkg
		goSrcRoot, _ = trimSuffix(file, pkg+"/logging/logging.go")
	}
	logger = log.New(os.Stderr, "", 0)
	SetFlags(log.LstdFlags | log.Lshortfile)
	for _, arg := range os.Args {
		if mods, ok := trimPrefix(arg, "--finelog="); ok {
			fineLogModules = append(fineLogModules, strings.Split(mods, ",")...)
		}
		if mods, ok := trimPrefix(arg, "-finelog="); ok {
			fineLogModules = append(fineLogModules, strings.Split(mods, ",")...)
		}
	}
}

func init() {
	// Cannot call construct init because runtime.Caller(0) behaves differently
	// in init functions than it does in other named functions.
	construct()
}

var baristaPkg = "#unknown#"
var goSrcRoot = os.Getenv("GOPATH")

// shorten shortens a package/function/type for logging. It removes the full path
// to barista packages, and simplifies functions that have a receiver.
func shorten(path string) string {
	// If path is a function, it can be something like some/package.(*Type).fn,
	// but we'll simplify it to some/package.Type.fn for logging.
	path = strings.Replace(path, "*", "", -1)
	path = strings.Replace(path, "(", "", -1)
	path = strings.Replace(path, ")", "", -1)

	if module, ok := trimPrefix(path, baristaPkg+"/modules/"); ok {
		return fmt.Sprintf("mod:%s", module)
	}
	if core, ok := trimPrefix(path, baristaPkg+"/core."); ok {
		return fmt.Sprintf("core:%s", core)
	}
	if bar, ok := trimPrefix(path, baristaPkg+"/"); ok {
		return fmt.Sprintf("bar:%s", bar)
	}
	if main, ok := trimPrefix(path, baristaPkg+"."); ok {
		return fmt.Sprintf("barista:%s", main)
	}
	return path
}

var fineLogModules = []string{}
var fineLogModulesCache sync.Map

// fineLogEnabled returns true if finelog is enabled for the module.
// It caches results in a sync.Map so subsequent lookups can be faster.
func fineLogEnabled(mod string) bool {
	cache, ok := fineLogModulesCache.Load(mod)
	if ok {
		return cache.(bool)
	}
	for _, fineMod := range fineLogModules {
		if strings.HasPrefix(mod, fineMod) {
			fineLogModulesCache.Store(mod, true)
			return true
		}
	}
	fineLogModulesCache.Store(mod, false)
	return false
}

// callingModule returns the calling module's name and source location.
// The name is prefixed based on origin:
//     - mod:$module for modules included with barista (e.g. mod:cpuinfo)
//     - bar:$core for core barista code (e.g. bar:notifier, bar:base)
//     - $package for all other code (e.g. github.com/user/repo/module)
// The source location is empty if neither shortfile nor longfile flags are set,
// otherwise it is the appropriately formatted file name, ":", and line number.
func callingModule() (mod string, loc string) {
	pc, file, line, ok := runtime.Caller(2)
	fFlags := int(atomic.LoadInt64(&fileFlags))
	if fFlags != 0 {
		file, _ = trimPrefix(file, goSrcRoot)
		if fFlags&log.Lshortfile != 0 {
			file = filepath.Base(file)
		}
		loc = fmt.Sprintf("%s:%d", file, line)
	}
	if !ok {
		return "unknown", loc
	}
	fnName := runtime.FuncForPC(pc).Name()
	return shorten(fnName), loc
}

var fileFlags int64
var logger *log.Logger

// doLog actually logs the given statement, with appropriate file information
// depending on the currently set flags.
func doLog(mod, loc string, format string, args ...interface{}) {
	out := fmt.Sprintf(format, args...)
	fFlags := int(atomic.LoadInt64(&fileFlags))
	if fFlags != 0 {
		out = fmt.Sprintf("%s (%s) %s", loc, mod, out)
	}
	logger.Output(3, out)
}

// SetOutput sets the output stream for logging.
func SetOutput(output io.Writer) {
	logger.SetOutput(output)
}

// SetFlags sets flags to control logging output.
func SetFlags(flags int) {
	fFlags := flags & (log.Llongfile | log.Lshortfile)
	atomic.StoreInt64(&fileFlags, int64(fFlags))
	logger.SetFlags(flags &^ fFlags)
}

// Log logs a formatted message.
func Log(format string, args ...interface{}) {
	mod, loc := callingModule()
	doLog(mod, loc, format, args...)
}

// Fine logs a formatted message if fine logging is enabled for the
// calling module. Enable fine logging using the commandline flag,
// `--finelog=$module1,$module2`. [Requires debug logging].
func Fine(format string, args ...interface{}) {
	mod, loc := callingModule()
	if fineLogEnabled(mod) {
		doLog(mod, loc, format, args...)
	}
}
