// Copyright 2022 Google Inc.
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
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

// HwmonOfNameAndLabel finds a sensor given hwmon name and label.
//
// For example, if you have /sys/class/hwmon/hwmon4/name containing
// "k10temp", and /sys/class/hwmon/hwmon4/temp1_label containing
// "Tctl", the former would be name, and the latter would be label.
func HwmonOfNameAndLabel(name string, label string) *Module {
	baseDir := "/sys/class/hwmon"
	files, _ := afero.ReadDir(fs, baseDir)
	for _, file := range files {
		n, _ := afero.ReadFile(fs, filepath.Join(baseDir, file.Name(), "name"))
		if strings.TrimSpace(string(n)) == name {
			baseDir := filepath.Join(baseDir, file.Name())
			files, _ := afero.ReadDir(fs, filepath.Join(baseDir))
			for _, file := range files {
				if strings.HasSuffix(file.Name(), "_label") {
					l, _ := afero.ReadFile(fs, filepath.Join(baseDir, file.Name()))
					if strings.TrimSpace(string(l)) == label {
						filename := file.Name()
						filename = strings.TrimSuffix(filename, "_label") + "_input"
						return newModule(filepath.Join(baseDir, filename))
					}
				}
			}
		}
	}
	return newModule("")
}
