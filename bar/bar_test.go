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

package bar_test

import (
	"encoding/json"
	"fmt"
	"syscall"
	"testing"
	"time"

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/outputs"
	"github.com/soumya92/barista/testing/mockio"
	// testing/module depends on bar, hence the '.' import and package name.
	. "github.com/soumya92/barista/bar"
	testModule "github.com/soumya92/barista/testing/module"
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
	assert.Equal(t, int(syscall.SIGUSR1), int(header["stop_signal"].(float64)), "header stop_signal == USR1")
	assert.Equal(t, int(syscall.SIGUSR2), int(header["cont_signal"].(float64)), "header cont_signal == USR2")
}

func readOutput(t *testing.T, stdout *mockio.Writable) []map[string]interface{} {
	var jsonOutputs []map[string]interface{}
	out, err := stdout.ReadUntil(']', time.Second)
	assert.Nil(t, err, "No error while reading output")
	assert.Nil(t, json.Unmarshal([]byte(out), &jsonOutputs), "Output is valid json")
	_, err = stdout.ReadUntil(',', time.Second)
	assert.Nil(t, err, "outputs a comma after full bar")
	return jsonOutputs
}

func readOutputTexts(t *testing.T, stdout *mockio.Writable) []string {
	jsonOutputs := readOutput(t, stdout)
	outputs := make([]string, len(jsonOutputs))
	for idx, jsonOutput := range jsonOutputs {
		outputs[idx] = jsonOutput["full_text"].(string)
	}
	return outputs
}

func TestSingleModule(t *testing.T) {
	mockStdin := mockio.Stdin()
	mockStdout := mockio.Stdout()
	bar := NewOnIo(mockStdin, mockStdout)

	module := testModule.New(t)

	bar.Add(module)
	go bar.Run()

	_, err := mockStdout.ReadUntil('[', time.Second)
	assert.Nil(t, err, "output array started without any errors")

	_, err = mockStdout.ReadUntil(']', time.Millisecond)
	assert.Error(t, err, "no output until module updates")

	module.Output(outputs.Text("test"))
	out := readOutputTexts(t, mockStdout)
	assert.Equal(t, []string{"test"}, out,
		"output contains an element for the module")

	_, err = mockStdout.ReadUntil(']', time.Millisecond)
	assert.Error(t, err, "no output until module updates")

	module.Output(outputs.Text("other"))
	out = readOutputTexts(t, mockStdout)
	assert.Equal(t, []string{"other"}, out,
		"output updates when module sends an update")

	assert.Panics(t,
		func() { bar.Add(testModule.New(t)) },
		"adding a module to a running bar")
}

func TestMultipleModules(t *testing.T) {
	mockStdin := mockio.Stdin()
	mockStdout := mockio.Stdout()

	module1 := testModule.New(t)
	module2 := testModule.New(t)
	module3 := testModule.New(t)
	go RunOnIo(mockStdin, mockStdout, module1, module2, module3)

	_, err := mockStdout.ReadUntil('[', time.Second)
	assert.Nil(t, err, "output array started without any errors")

	_, err = mockStdout.ReadUntil(']', time.Millisecond)
	assert.Error(t, err, "no output until module updates")

	module1.Output(outputs.Text("test"))

	out := readOutputTexts(t, mockStdout)
	assert.Equal(t, []string{"test"}, out,
		"output contains elements only for modules that have output")

	_, err = mockStdout.ReadUntil(']', time.Millisecond)
	assert.Error(t, err, "no output until module updates")

	module3.Output(outputs.Text("module3"))
	out = readOutputTexts(t, mockStdout)
	assert.Equal(t, []string{"test", "module3"}, out,
		"new output repeats previous value for other modules")

	module3.Output(outputs.Text("new value"))
	out = readOutputTexts(t, mockStdout)
	assert.Equal(t, []string{"test", "new value"}, out,
		"updated output repeats previous value for other modules")

	module2.Output(outputs.Text("middle"))
	out = readOutputTexts(t, mockStdout)
	assert.Equal(t, []string{"test", "middle", "new value"}, out,
		"newly updated module correctly repositions other modules")

	module1.Output(Output{})
	out = readOutputTexts(t, mockStdout)
	assert.Equal(t, []string{"middle", "new value"}, out,
		"nil output correctly repositions other modules")
}

func multiOutput(texts ...string) Output {
	m := outputs.Multi()
	for idx, text := range texts {
		m.AddText(fmt.Sprintf("instance_%d", idx), text)
	}
	return m.Build()
}

func TestMultiSegmentModule(t *testing.T) {
	mockStdin := mockio.Stdin()
	mockStdout := mockio.Stdout()

	module := testModule.New(t)
	go RunOnIo(mockStdin, mockStdout, module)

	_, err := mockStdout.ReadUntil('[', time.Second)
	assert.Nil(t, err, "output array started without any errors")

	_, err = mockStdout.ReadUntil(']', time.Millisecond)
	assert.Error(t, err, "no output until module updates")

	module.Output(multiOutput("1", "2", "3"))
	out := readOutputTexts(t, mockStdout)
	assert.Equal(t, []string{"1", "2", "3"}, out,
		"output contains an element for each segment")

	// Implicit in the previous assertion is the fact that all segments
	// are output together, not one at a time. That is, only one array is
	// output with all three segments, rather than an array with 1, then
	// with 1,2, then with 1,2,3, which is what would happen if we had
	// three modules each output 1,2,3 respectively.

	module.Output(multiOutput("2", "3"))
	out = readOutputTexts(t, mockStdout)
	assert.Equal(t, []string{"2", "3"}, out,
		"bar handles a disappearing segment correctly")

	module.Output(multiOutput("2", "3", "4", "5", "6"))
	out = readOutputTexts(t, mockStdout)
	assert.Equal(t, []string{"2", "3", "4", "5", "6"}, out,
		"bar handles additional segments correctly")
}

func TestPauseResume(t *testing.T) {
	mockStdin := mockio.Stdin()
	mockStdout := mockio.Stdout()

	module1 := testModule.New(t)
	module2 := testModule.New(t)
	go RunOnIo(mockStdin, mockStdout, module1, module2)

	// When the infinite array starts, we know the bar is ready.
	_, err := mockStdout.ReadUntil('[', time.Second)
	assert.Nil(t, err, "output array started without any errors")

	syscall.Kill(syscall.Getpid(), syscall.SIGUSR1)
	module1.AssertPaused("on sigusr1")
	module2.AssertPaused("on sigusr1")

	syscall.Kill(syscall.Getpid(), syscall.SIGUSR2)
	module1.AssertResumed("on sigusr2")
	module2.AssertResumed("on sigusr2")

	module1.AssertNoPauseResume("when bar receives no signals")
	module2.AssertNoPauseResume("when bar receives no signals")
}

func TestClickEvents(t *testing.T) {
	mockStdin := mockio.Stdin()
	mockStdout := mockio.Stdout()

	module1 := testModule.New(t)
	module2 := testModule.New(t)
	go RunOnIo(mockStdin, mockStdout, module1, module2)

	_, err := mockStdout.ReadUntil('[', time.Second)
	assert.Nil(t, err, "output array started without any errors")
	mockStdin.WriteString("[")

	module1.Output(multiOutput("1", "2", "3"))
	readOutput(t, mockStdout)

	module2.Output(multiOutput("a", "b", "c", "d"))
	out := readOutput(t, mockStdout)

	assert.Equal(t, 7, len(out), "All segments in output")
	module1_name := out[0]["name"].(string)
	module2_name := out[3]["name"].(string)

	module1.AssertNotClicked("when no click event")
	module2.AssertNotClicked("when no click event")

	mockStdin.WriteString(
		fmt.Sprintf("{\"name\": \"%s\", \"x\": %d, \"y\": %d, \"button\": %d},",
			module1_name, 13, 24, 3))
	evt := module1.AssertClicked("when getting a click event")
	assert.Equal(t, 13, evt.X, "event value is passed through")
	assert.Equal(t, 24, evt.Y, "event value is passed through")
	assert.Equal(t, ButtonRight, evt.Button, "event value is passed through")
	module2.AssertNotClicked("only target module receives the event")

	mockStdin.WriteString(fmt.Sprintf("{\"name\": \"%s\", ", module2_name))
	module2.AssertNotClicked("until event is completely written")
	mockStdin.WriteString("\"X\": 9, \"Y\": 7")
	module2.AssertNotClicked("until event is completely written")
	mockStdin.WriteString("},")
	evt = module2.AssertClicked("when getting a click event")
	assert.Equal(t, Event{X: 9, Y: 7}, evt, "event values are passed through")
	module1.AssertNotClicked("only target module receives the event")

	mockStdin.WriteString("{\"name\":\"blah\",\"x\":9},")
	module1.AssertNotClicked("with weird module name")
	module2.AssertNotClicked("with weird module name")

	mockStdin.WriteString(fmt.Sprintf("{\"name\": \"%s\"},", module1_name))
	module1.AssertClicked("events are received after the weird name")

	mockStdin.WriteString(fmt.Sprintf("{\"name\": \"%s\"},", module2_name))
	module2.AssertClicked("events are received after the weird name")

	mockStdin.WriteString("{\"name\":\"8\",\"x\":9},")
	module1.AssertNotClicked("with weird module name")
	module2.AssertNotClicked("with weird module name")

	mockStdin.WriteString("{\"name\":\"-45\",\"x\":9},")
	module1.AssertNotClicked("with weird module name")
	module2.AssertNotClicked("with weird module name")

	mockStdin.WriteString(fmt.Sprintf("{\"name\": \"%s\"},", module1_name))
	module1.AssertClicked("events are received after the weird name")

	mockStdin.WriteString(fmt.Sprintf("{\"name\": \"%s\"},", module2_name))
	module2.AssertClicked("events are received after the weird name")
}

func TestSignalHandlingSuppression(t *testing.T) {
	mockStdin := mockio.Stdin()
	mockStdout := mockio.Stdout()

	module := testModule.New(t)
	b := NewOnIo(mockStdin, mockStdout).Add(module)
	assert.NotPanics(t,
		func() { b.SuppressSignals(true) },
		"Can suppress signal handling before Run")
	go b.Run()

	out, err := mockStdout.ReadUntil('}', time.Second)
	assert.Nil(t, err, "header was written")

	header := make(map[string]interface{})
	assert.Nil(t, json.Unmarshal([]byte(out), &header), "header is valid json")
	// JSON deserialises all numbers as float64.
	assert.Equal(t, 1, int(header["version"].(float64)), "header version == 1")
	assert.Equal(t, true, header["click_events"].(bool), "header click_events == true")

	// Ensure no signals are written in header.
	_, ok := header["stop_signal"]
	assert.False(t, ok, "No stop_signal in header")
	_, ok = header["cont_signal"]
	assert.False(t, ok, "No cont_signal in header")

	// When the infinite array starts, we know the bar is ready.
	_, err = mockStdout.ReadUntil('[', time.Second)
	assert.Nil(t, err, "output array started without any errors")

	syscall.Kill(syscall.Getpid(), syscall.SIGUSR1)
	module.AssertNoPauseResume("when signal handling is suppressed")

	syscall.Kill(syscall.Getpid(), syscall.SIGUSR2)
	module.AssertNoPauseResume("when signal handling is suppressed")

	assert.Panics(t,
		func() { b.SuppressSignals(false) },
		"Cannot suppress signal handling after Run")
}
