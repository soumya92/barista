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

package temperature

import (
	"fmt"
	"strings"

	"github.com/spf13/afero"
)

// ThermalZone constructs an instance of the temperature module for the specified zone.
// The file /sys/class/thermal/<zone>/temp should return cpu temp in 1/1000 deg C.
func ThermalZone(thermalZone string) *Module {
	return newModule(fmt.Sprintf("/sys/class/thermal/%s/temp", thermalZone))
}

// ThermalOfType constructs an instance of the cputemp module for the *first*
// available sensor of the given type. "x86_pkg_temp" usually represents the temperature
// of the actual CPU package, while others may be available depending on the
// system, e.g. "iwlwifi" for wifi, or "acpitz" for the motherboard.
func ThermalOfType(typ string) *Module {
	files, _ := afero.ReadDir(fs, "/sys/class/thermal")
	for _, file := range files {
		name := file.Name()
		typFile := fmt.Sprintf("/sys/class/thermal/%s/type", name)
		typBytes, _ := afero.ReadFile(fs, typFile)
		if strings.TrimSpace(string(typBytes)) == typ {
			return ThermalZone(name)
		}
	}
	return ThermalZone("")
}

// NewDefaultThermal constructs an instance of the cputemp module for zone type
// "x86_pkg_temp". Returns nil of the x86_pkg_temp thermal zone is unavailable.
func NewDefaultThermal() *Module {
	return ThermalOfType("x86_pkg_temp")
}
