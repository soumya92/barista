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

// Package hwmon implements an i3bar module that shows the temperature from /sys/class/hwmon
package hwmon

import (
	"github.com/soumya92/barista/modules/internal/temperature"
)

// Module represents a hwmon bar module. It supports setting the output
// format, click handler, update frequency, and urgency/colour functions.
type Module = temperature.Module

// OfNameAndLabel finds a sensor given hwmon name and label.
//
// For example, if you have /sys/class/hwmon/hwmon4/name containing
// "k10temp", and /sys/class/hwmon/hwmon4/temp1_label containing
// "Tctl", the former would be name, and the latter would be label.
func OfNameAndLabel(name string, label string) *Module {
	return temperature.HwmonOfNameAndLabel(name, label)
}
