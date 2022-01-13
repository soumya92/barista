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
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHwmonDetection(t *testing.T) {
	fs = afero.NewMemMapFs()
	contents := []struct {
		path string
		data string
	}{
		{"/sys/class/hwmon/hwmon0/name", "k10temp\n"},
		{"/sys/class/hwmon/hwmon0/temp1_label", "Tctl\n"},
		{"/sys/class/hwmon/hwmon0/temp1_input", "63500\n"},
		{"/sys/class/hwmon/hwmon1/name", "amdgpu\n"},
		{"/sys/class/hwmon/hwmon1/temp1_label", "edge\n"},
		{"/sys/class/hwmon/hwmon1/temp2_label", "junction\n"},
		{"/sys/class/hwmon/hwmon1/temp3_label", "mem\n"},
		{"/sys/class/hwmon/hwmon1/temp1_input", "56000\n"},
		{"/sys/class/hwmon/hwmon1/temp2_input", "56000\n"},
		{"/sys/class/hwmon/hwmon1/temp3_input", "56000\n"},
	}
	for _, content := range contents {
		err := afero.WriteFile(fs, content.path, []byte(content.data), 0644)
		require.NoError(t, err)
	}
	// This only tests detection code.
	// functional tests are in thermalzone_test.go
	var m *Module
	m = HwmonOfNameAndLabel("k10temp", "Tctl")
	assert.Equal(t, "/sys/class/hwmon/hwmon0/temp1_input", m.thermalFile)
	m = HwmonOfNameAndLabel("amdgpu", "junction")
	assert.Equal(t, "/sys/class/hwmon/hwmon1/temp2_input", m.thermalFile)
}
