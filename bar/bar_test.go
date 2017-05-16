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

package bar

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/testing/mockio"
)

func TestHeader(t *testing.T) {
	mockStdin := mockio.Stdin()
	mockStdout := mockio.Stdout()
	bar := NewOnIo(mockStdin, mockStdout)
	assert.Empty(t, mockStdout.ReadNow(), "Nothing written on construction")
	go bar.Run()

	out, err := mockStdout.ReadUntil('}', time.Second)
	assert.Nil(t, err, "header was written")

	header := make(map[string]interface{})
	assert.Nil(t, json.Unmarshal([]byte(out), &header), "header is valid json")
	// JSON deserialises all numbers as float64.
	assert.Equal(t, 1, int(header["version"].(float64)), "header version == 1")
	assert.Equal(t, true, header["click_events"].(bool), "header click_events == true")
}

type testModule chan Output

func (t testModule) Stream() <-chan Output { return (<-chan Output)(t) }
func (t testModule) Output(o Output)       { t <- o }

func readOneBarOutput(t *testing.T, stdout *mockio.Writable) []string {
	var jsonOutputs []map[string]interface{}
	out, err := stdout.ReadUntil(']', time.Second)
	assert.Nil(t, err, "No error while reading output")
	assert.Nil(t, json.Unmarshal([]byte(out), &jsonOutputs), "Output is valid json")
	outputs := make([]string, len(jsonOutputs))
	for idx, jsonOutput := range jsonOutputs {
		outputs[idx] = jsonOutput["full_text"].(string)
	}
	_, err = stdout.ReadUntil(',', time.Second)
	assert.Nil(t, err, "outputs a comma after full bar")
	return outputs
}

func textOutput(text ...string) Output {
	var out Output
	for _, t := range text {
		out = append(out, NewSegment(t))
	}
	return out
}

func TestSingleModule(t *testing.T) {
	mockStdin := mockio.Stdin()
	mockStdout := mockio.Stdout()
	bar := NewOnIo(mockStdin, mockStdout)

	module := make(testModule)

	bar.Add(module)
	go bar.Run()

	_, err := mockStdout.ReadUntil('[', time.Second)
	assert.Nil(t, err, "output array started without any errors")

	_, err = mockStdout.ReadUntil(']', time.Millisecond)
	assert.Error(t, err, "no output until module updates")

	module.Output(textOutput("test"))
	out := readOneBarOutput(t, mockStdout)
	assert.Equal(t, []string{"test"}, out,
		"output contains an element for the module")

	_, err = mockStdout.ReadUntil(']', time.Millisecond)
	assert.Error(t, err, "no output until module updates")

	module.Output(textOutput("other"))
	out = readOneBarOutput(t, mockStdout)
	assert.Equal(t, []string{"other"}, out,
		"output updates when module sends an update")
}

func TestMultipleModules(t *testing.T) {
	mockStdin := mockio.Stdin()
	mockStdout := mockio.Stdout()
	bar := NewOnIo(mockStdin, mockStdout)

	module1 := make(testModule)
	module2 := make(testModule)
	module3 := make(testModule)

	bar.Add(module1, module2, module3)
	go bar.Run()

	_, err := mockStdout.ReadUntil('[', time.Second)
	assert.Nil(t, err, "output array started without any errors")

	_, err = mockStdout.ReadUntil(']', time.Millisecond)
	assert.Error(t, err, "no output until module updates")

	module1.Output(textOutput("test"))

	out := readOneBarOutput(t, mockStdout)
	assert.Equal(t, []string{"test"}, out,
		"output contains elements only for modules that have output")

	_, err = mockStdout.ReadUntil(']', time.Millisecond)
	assert.Error(t, err, "no output until module updates")

	module3.Output(textOutput("module3"))
	out = readOneBarOutput(t, mockStdout)
	assert.Equal(t, []string{"test", "module3"}, out,
		"new output repeats previous value for other modules")

	module3.Output(textOutput("new value"))
	out = readOneBarOutput(t, mockStdout)
	assert.Equal(t, []string{"test", "new value"}, out,
		"updated output repeats previous value for other modules")

	module2.Output(textOutput("middle"))
	out = readOneBarOutput(t, mockStdout)
	assert.Equal(t, []string{"test", "middle", "new value"}, out,
		"newly updated module correctly repositions other modules")

	module1.Output(Output{})
	out = readOneBarOutput(t, mockStdout)
	assert.Equal(t, []string{"middle", "new value"}, out,
		"nil output correctly repositions other modules")
}

func TestMultiSegmentModule(t *testing.T) {
	mockStdin := mockio.Stdin()
	mockStdout := mockio.Stdout()
	bar := NewOnIo(mockStdin, mockStdout)

	module := make(testModule)

	bar.Add(module)
	go bar.Run()

	_, err := mockStdout.ReadUntil('[', time.Second)
	assert.Nil(t, err, "output array started without any errors")

	_, err = mockStdout.ReadUntil(']', time.Millisecond)
	assert.Error(t, err, "no output until module updates")

	module.Output(textOutput("1", "2", "3"))
	out := readOneBarOutput(t, mockStdout)
	assert.Equal(t, []string{"1", "2", "3"}, out,
		"output contains an element for each segment")

	// Implicit in the previous assertion is the fact that all segments
	// are output together, not one at a time. That is, only one array is
	// output with all three segments, rather than an array with 1, then
	// with 1,2, then with 1,2,3, which is what would happen if we had
	// three modules each output 1,2,3 respectively.

	module.Output(textOutput("2", "3"))
	out = readOneBarOutput(t, mockStdout)
	assert.Equal(t, []string{"2", "3"}, out,
		"bar handles a disappearing segment correctly")

	module.Output(textOutput("2", "3", "4", "5", "6"))
	out = readOneBarOutput(t, mockStdout)
	assert.Equal(t, []string{"2", "3", "4", "5", "6"}, out,
		"bar handles additional segments correctly")
}
