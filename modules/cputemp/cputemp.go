// Copyright 2017 Google Inc.
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

// Package cputemp implements an i3bar module that shows the CPU temperature.
package cputemp

import (
	"github.com/soumya92/barista/modules/internal/temperature"
)

// Module represents a cputemp bar module. It supports setting the output
// format, click handler, update frequency, and urgency/colour functions.
type Module = temperature.Module

// Zone constructs an instance of the cputemp module for the specified zone.
// The file /sys/class/thermal/<zone>/temp should return cpu temp in 1/1000 deg C.
func Zone(thermalZone string) *Module {
	return temperature.ThermalZone(thermalZone)
}

// OfType constructs an instance of the cputemp module for the *first* available
// sensor of the given type. "x86_pkg_temp" usually represents the temperature
// of the actual CPU package, while others may be available depending on the
// system, e.g. "iwlwifi" for wifi, or "acpitz" for the motherboard.
func OfType(typ string) *Module {
	return temperature.ThermalOfType(typ)
}

// New constructs an instance of the cputemp module for zone type "x86_pkg_temp".
// Returns nil of the x86_pkg_temp thermal zone is unavailable.
func New() *Module {
	return temperature.NewDefaultThermal()
}
