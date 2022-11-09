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

// +build !baristadebuglog

// Package logging provides logging functions for use in the bar and modules.
// It uses build tags to provide nop functions in the default case, and
// actual logging functions when built with `-tags baristadebuglog`.
package logging

import "io"

// SetOutput sets the output stream for logging.
func SetOutput(output io.Writer) {}

// SetFlags sets flags to control logging output.
func SetFlags(flags int) {}

// Log logs a formatted message.
func Log(format string, args ...interface{}) {}

// Fine logs a formatted message if fine logging is enabled for the
// calling module. Enable fine logging using the commandline flag,
// `--finelog=$module1,$module2`. [Requires debug logging].
func Fine(format string, args ...interface{}) {}

// ID returns a unique name for the given value of the form 'type'#'index'
// for addressable types. This provides log statements with additional
// context and separates logs from multiple instances of the same type.
func ID(thing interface{}) string { return "" }

// Label adds an additional label to thing, incorporated as part of its
// identifier, to provide more useful information than just #0, #1, ...
// For example, a diskspace module might use:
//     logging.Label(m, "sda1")
// which would make its ID mod:diskspace.Module#0<sda1>, making it
// easier to track in logs.
func Label(thing interface{}, label string) {}

// Labelf is Label with built-in formatting. Because all logging functions
// are no-ops without baristadebuglog, having the sprintf be part of the Labelf
// function means that it will only be executed if debug logging is on.
func Labelf(thing interface{}, format string, args ...interface{}) {}

// Attach attaches an object as a named member of a different object.
// This is useful when a generic type (e.g. chan) is used within a more
// specific type (e.g. Module). Typical usage would be:
//     logging.Attach(m, m.scheduler, "refresher")
// where m is a module, m.scheduler is a timing.Scheduler.
// This will make subsequent log statements that use that scheduler as a
// context (even from a different package, e.g. timing) print it as
// module#1.refresher instead of timing.Scheduler#45.
func Attach(parent, child interface{}, name string) {}

// Attachf is Attach with built-in formatting.
func Attachf(parent, child interface{}, format string, args ...interface{}) {}

// Register attaches the given fields of a given *struct as '.' + name.
// This is just a shortcut for Register(&thing, &thing.field, ".field")...
// for a set of fields.
func Register(thing interface{}, names ...string) {}
