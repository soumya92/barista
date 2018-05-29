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

package barista

import (
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"os"
	"os/signal"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/outputs"
	"github.com/soumya92/barista/testing/mockio"
	testModule "github.com/soumya92/barista/testing/module"
	"github.com/soumya92/barista/timing"
)

func TestHeader(t *testing.T) {
	mockStdin := mockio.Stdin()
	mockStdout := mockio.Stdout()
	TestMode(mockStdin, mockStdout)
	assert.Empty(t, mockStdout.ReadNow(), "Nothing written before Run")
	go Run()

	out, err := mockStdout.ReadUntil('}', time.Second)
	assert.Nil(t, err, "header was written")

	header := make(map[string]interface{})
	assert.Nil(t, json.Unmarshal([]byte(out), &header), "header is valid json")
	// JSON deserialises all numbers as float64.
	assert.Equal(t, 1, int(header["version"].(float64)), "header version == 1")
	assert.Equal(t, true, header["click_events"].(bool), "header click_events == true")
	assert.Equal(t, int(unix.SIGUSR1), int(header["stop_signal"].(float64)), "header stop_signal == USR1")
	assert.Equal(t, int(unix.SIGUSR2), int(header["cont_signal"].(float64)), "header cont_signal == USR2")
}

func readOutput(t *testing.T, stdout *mockio.Writable) []map[string]interface{} {
	var jsonOutputs []map[string]interface{}
	out, err := stdout.ReadUntil(']', time.Second)
	assert.Nil(t, err, "No error while reading output")
	assert.Nil(t, json.Unmarshal([]byte(out), &jsonOutputs), "Output is valid json")
	_, err = stdout.ReadUntil(',', time.Second)
	assert.Nil(t, err, "outputs a comma after full bar")
	_, err = stdout.ReadUntil('\n', time.Second)
	assert.Nil(t, err, "outputs a newline after full bar")
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

func waitForSignal(sigChan <-chan os.Signal) {
	ch := make(chan bool)
	go func() {
		<-sigChan
		// TODO: Eliminate this sleep.
		time.Sleep(10 * time.Millisecond)
		ch <- true
	}()
	<-ch
}

func TestSingleModule(t *testing.T) {
	mockStdin := mockio.Stdin()
	mockStdout := mockio.Stdout()
	TestMode(mockStdin, mockStdout)

	module := testModule.New(t)

	Add(module)
	go Run()

	_, err := mockStdout.ReadUntil('[', time.Second)
	assert.Nil(t, err, "output array started without any errors")

	_, err = mockStdout.ReadUntil(']', time.Millisecond)
	assert.Error(t, err, "no output until module updates")

	module.AssertStarted()
	module.OutputText("test")
	out := readOutputTexts(t, mockStdout)
	assert.Equal(t, []string{"test"}, out,
		"output contains an element for the module")

	_, err = mockStdout.ReadUntil(']', time.Millisecond)
	assert.Error(t, err, "no output until module updates")

	module.OutputText("other")
	out = readOutputTexts(t, mockStdout)
	assert.Equal(t, []string{"other"}, out,
		"output updates when module sends an update")

	assert.Panics(t,
		func() { Add(testModule.New(t)) },
		"adding a module to a running bar")
}

func TestEmptyOutputs(t *testing.T) {
	mockStdin := mockio.Stdin()
	mockStdout := mockio.Stdout()
	TestMode(mockStdin, mockStdout)

	module1 := testModule.New(t)
	module2 := testModule.New(t)

	go Run(module1, module2)

	_, err := mockStdout.ReadUntil('[', time.Second)
	assert.Nil(t, err, "output array started without any errors")

	_, err = mockStdout.ReadUntil(']', time.Millisecond)
	assert.Error(t, err, "no output until module updates")

	module1.AssertStarted()
	module1.Output(outputs.Empty())
	out := readOutputTexts(t, mockStdout)
	assert.Empty(t, out, "all modules are empty")

	module2.AssertStarted()
	module2.Output(outputs.Empty())
	out = readOutputTexts(t, mockStdout)
	assert.Empty(t, out, "all modules are empty")
}

func TestMultipleModules(t *testing.T) {
	mockStdin := mockio.Stdin()
	mockStdout := mockio.Stdout()
	TestMode(mockStdin, mockStdout)

	module1 := testModule.New(t)
	module2 := testModule.New(t)
	module3 := testModule.New(t)
	go Run(module1, module2, module3)

	_, err := mockStdout.ReadUntil('[', time.Second)
	assert.Nil(t, err, "output array started without any errors")

	_, err = mockStdout.ReadUntil(']', time.Millisecond)
	assert.Error(t, err, "no output until module updates")

	module1.AssertStarted()
	module1.OutputText("test")

	out := readOutputTexts(t, mockStdout)
	assert.Equal(t, []string{"test"}, out,
		"output contains elements only for modules that have output")

	_, err = mockStdout.ReadUntil(']', time.Millisecond)
	assert.Error(t, err, "no output until module updates")

	module3.AssertStarted()
	module3.OutputText("module3")
	out = readOutputTexts(t, mockStdout)
	assert.Equal(t, []string{"test", "module3"}, out,
		"new output repeats previous value for other modules")

	module3.OutputText("new value")
	out = readOutputTexts(t, mockStdout)
	assert.Equal(t, []string{"test", "new value"}, out,
		"updated output repeats previous value for other modules")

	module2.AssertStarted()
	module2.OutputText("middle")
	out = readOutputTexts(t, mockStdout)
	assert.Equal(t, []string{"test", "middle", "new value"}, out,
		"newly updated module correctly repositions other modules")

	module1.Output(outputs.Empty())
	out = readOutputTexts(t, mockStdout)
	assert.Equal(t, []string{"middle", "new value"}, out,
		"nil output correctly repositions other modules")
}

func multiOutput(texts ...string) bar.Output {
	m := outputs.Group()
	for _, text := range texts {
		m.Append(bar.TextSegment(text))
	}
	return m
}

func TestMultiSegmentModule(t *testing.T) {
	mockStdin := mockio.Stdin()
	mockStdout := mockio.Stdout()
	TestMode(mockStdin, mockStdout)

	module := testModule.New(t)
	go Run(module)

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
	TestMode(mockStdin, mockStdout)

	module1 := testModule.New(t)
	module2 := testModule.New(t)
	go Run(module1, module2)

	signalChan := make(chan os.Signal, 2)
	signal.Notify(signalChan, unix.SIGUSR1, unix.SIGUSR2)

	_, err := mockStdout.ReadUntil('[', time.Second)
	assert.Nil(t, err, "output array started without any errors")

	module1.AssertStarted()
	module2.AssertStarted()

	module1.OutputText("1")
	out := readOutputTexts(t, mockStdout)
	assert.Equal(t, []string{"1"}, out, "Outputs before pause")
	sch1 := timing.NewScheduler().After(time.Millisecond)
	time.Sleep(2 * time.Millisecond)
	select {
	case <-sch1.Tick():
	default:
		assert.Fail(t, "Scheduler not triggered while bar is running")
	}

	unix.Kill(unix.Getpid(), unix.SIGUSR1)
	waitForSignal(signalChan)
	module1.OutputText("a")
	module2.OutputText("b")
	assert.False(t, mockStdout.WaitForWrite(10*time.Millisecond),
		"No output while paused")

	sch1.After(time.Millisecond)
	sch2 := timing.NewScheduler().After(time.Millisecond)
	time.Sleep(2 * time.Millisecond)
	select {
	case <-sch1.Tick():
		assert.Fail(t, "Scheduler triggered while bar is paused")
	case <-sch2.Tick():
		assert.Fail(t, "Scheduler triggered while bar is paused")
	default:
	}

	unix.Kill(unix.Getpid(), unix.SIGUSR2)
	waitForSignal(signalChan)
	out = readOutputTexts(t, mockStdout)
	assert.Equal(t, []string{"a", "b"}, out,
		"Outputs while paused printed on resume")
	select {
	case <-sch1.Tick():
	default:
		assert.Fail(t, "Scheduler not triggered after bar is resumed")
	}
	select {
	case <-sch2.Tick():
	default:
		assert.Fail(t, "Scheduler not triggered after bar is resumed")
	}

	unix.Kill(unix.Getpid(), unix.SIGUSR2)
	assert.False(t, mockStdout.WaitForWrite(10*time.Millisecond),
		"Resuming a running bar is a nop")

	unix.Kill(unix.Getpid(), unix.SIGUSR1)
	waitForSignal(signalChan)
	module2.OutputText("c")
	assert.False(t, mockStdout.WaitForWrite(10*time.Millisecond),
		"No output while paused, got %s", mockStdout.ReadNow())

	unix.Kill(unix.Getpid(), unix.SIGUSR2)
	waitForSignal(signalChan)
	out = readOutputTexts(t, mockStdout)
	assert.Equal(t, []string{"a", "c"}, out,
		"Partial updates while paused")

	signal.Stop(signalChan)
}

func TestClickEvents(t *testing.T) {
	mockStdin := mockio.Stdin()
	mockStdout := mockio.Stdout()
	TestMode(mockStdin, mockStdout)

	module1 := testModule.New(t)
	module2 := testModule.New(t)
	go Run(module1, module2)

	_, err := mockStdout.ReadUntil('[', time.Second)
	assert.Nil(t, err, "output array started without any errors")
	mockStdin.WriteString("[")

	module1.AssertStarted()
	module2.AssertStarted()

	module1.Output(multiOutput("1", "2", "3"))
	readOutput(t, mockStdout)

	module2.Output(multiOutput("a", "b", "c", "d"))
	out := readOutput(t, mockStdout)

	assert.Equal(t, 7, len(out), "All segments in output")
	module1Name := out[0]["name"].(string)
	module2Name := out[3]["name"].(string)

	module1.AssertNotClicked("when no click event")
	module2.AssertNotClicked("when no click event")

	mockStdin.WriteString(
		fmt.Sprintf("{\"name\": \"%s\", \"x\": %d, \"y\": %d, \"button\": %d},",
			module1Name, 13, 24, 3))
	evt := module1.AssertClicked("when getting a click event")
	assert.Equal(t, 13, evt.ScreenX, "event value is passed through")
	assert.Equal(t, 24, evt.ScreenY, "event value is passed through")
	assert.Equal(t, bar.ButtonRight, evt.Button, "event value is passed through")
	module2.AssertNotClicked("only target module receives the event")

	mockStdin.WriteString(fmt.Sprintf("{\"name\": \"%s\", ", module2Name))
	module2.AssertNotClicked("until event is completely written")
	mockStdin.WriteString("\"relative_x\": 9, \"relative_y\": 7")
	module2.AssertNotClicked("until event is completely written")
	mockStdin.WriteString("},")
	evt = module2.AssertClicked("when getting a click event")
	assert.Equal(t, bar.Event{X: 9, Y: 7}, evt, "event values are passed through")
	module1.AssertNotClicked("only target module receives the event")

	mockStdin.WriteString("{\"name\":\"m/foo/bar\",\"x\":9},")
	module1.AssertNotClicked("with weird module name")
	module2.AssertNotClicked("with weird module name")

	mockStdin.WriteString(fmt.Sprintf("{\"name\": \"%s\"},", module1Name))
	module1.AssertClicked("events are received after the weird name")

	mockStdin.WriteString(fmt.Sprintf("{\"name\": \"%s\"},", module2Name))
	module2.AssertClicked("events are received after the weird name")

	mockStdin.WriteString("{\"name\":\"m/8\",\"x\":9},")
	module1.AssertNotClicked("with weird module name")
	module2.AssertNotClicked("with weird module name")

	mockStdin.WriteString("{\"name\":\"e/2/-45\",\"x\":9},")
	module1.AssertNotClicked("with weird module name")
	module2.AssertNotClicked("with weird module name")

	mockStdin.WriteString(fmt.Sprintf("{\"name\": \"%s\"},", module1Name))
	module1.AssertClicked("events are received after the weird name")

	mockStdin.WriteString(fmt.Sprintf("{\"name\": \"%s\"},", module2Name))
	module2.AssertClicked("events are received after the weird name")

	module1.Close()
	module1.AssertNotStarted("After Close()")

	mockStdin.WriteString(fmt.Sprintf("{\"name\": \"%s\"},", module2Name))
	module2.AssertClicked()
	module1.AssertNotStarted("When a different module is clicked")

	mockStdin.WriteString(fmt.Sprintf("{\"name\": \"%s\", \"button\": 7},", module1Name))
	module1.AssertNotClicked("After Close()")
	module1.AssertNotStarted("On Button7 click")

	mockStdin.WriteString(fmt.Sprintf("{\"name\": \"%s\", \"button\": 1},", module1Name))
	module1.AssertNotClicked("After Close(), before restart")
	module1.AssertStarted("On Button1 click")
}

func TestSignalHandlingSuppression(t *testing.T) {
	mockStdin := mockio.Stdin()
	mockStdout := mockio.Stdout()
	TestMode(mockStdin, mockStdout)

	module := testModule.New(t)
	Add(module)
	assert.NotPanics(t,
		func() { SuppressSignals(true) },
		"Can suppress signal handling before Run")
	go Run()
	signalChan := make(chan os.Signal, 2)
	signal.Notify(signalChan, unix.SIGUSR1, unix.SIGUSR2)

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

	_, err = mockStdout.ReadUntil('[', time.Second)
	assert.Nil(t, err, "output array started without any errors")
	module.AssertStarted()

	unix.Kill(unix.Getpid(), unix.SIGUSR1)
	waitForSignal(signalChan)
	module.OutputText("a")
	outs := readOutputTexts(t, mockStdout)
	assert.Equal(t, []string{"a"}, outs,
		"Not paused on USR1")

	unix.Kill(unix.Getpid(), unix.SIGUSR2)
	waitForSignal(signalChan)
	assert.False(t, mockStdout.WaitForWrite(10*time.Millisecond),
		"USR2 is a nop")

	unix.Kill(unix.Getpid(), unix.SIGUSR1)
	waitForSignal(signalChan)
	assert.False(t, mockStdout.WaitForWrite(10*time.Millisecond),
		"USR1 is a nop")

	assert.Panics(t,
		func() { SuppressSignals(false) },
		"Cannot suppress signal handling after Run")
	signal.Stop(signalChan)
}

func TestErrorHandling(t *testing.T) {
	mockStdin := mockio.Stdin()
	mockStdout := mockio.Stdout()
	TestMode(mockStdin, mockStdout)
	errChan := make(chan bar.ErrorEvent)
	SetErrorHandler(func(e bar.ErrorEvent) { errChan <- e })

	module := testModule.New(t)
	go Run(module)

	module.AssertStarted()
	mockStdin.WriteString("[")
	mockStdout.ReadUntil('[', time.Second)

	outputsWithError := outputs.Group().
		Append(outputs.Errorf("foo")).
		Append(outputs.Text("regular")).
		Append(outputs.Text("other").Error(errors.New("bar")))

	module.Output(outputsWithError)
	out := readOutput(t, mockStdout)
	assert.Equal(t, 3, len(out), "All segments in output")

	errorSegmentName := out[0]["name"].(string)
	regularSegmentName := out[1]["name"].(string)

	mockStdin.WriteString(fmt.Sprintf(`{"name": "%s"},`, regularSegmentName))
	module.AssertClicked("on click of regular segment")

	mockStdin.WriteString(fmt.Sprintf(`{"name": "%s"},`, errorSegmentName))
	module.AssertClicked("on click of error segment")

	mockStdin.WriteString(fmt.Sprintf(`{"name": "%s", "x": 4, "button": 3},`, errorSegmentName))
	module.AssertNotClicked("on right click of error segment")
	select {
	case e := <-errChan:
		assert.Equal(t, "foo", e.Error.Error())
		assert.Equal(t, bar.Event{ScreenX: 4, Button: bar.ButtonRight}, e.Event)
	case <-time.After(time.Second):
		assert.Fail(t, "should trigger error handler on right click")
	}

	assert.False(t, mockStdout.WaitForWrite(10*time.Millisecond),
		"click events do not cause any updates")

	module.Close()

	mockStdin.WriteString(fmt.Sprintf(`{"name": "%s", "button": 3},`, errorSegmentName))
	module.AssertNotClicked("on right click of error segment")
	select {
	case e := <-errChan:
		assert.Equal(t, "foo", e.Error.Error())
	case <-time.After(time.Second):
		assert.Fail(t, "should trigger error handler even after close")
	}

	mockStdin.WriteString(fmt.Sprintf(`{"name": "%s", "button": 1},`, errorSegmentName))
	assert.Equal(t, []string{"regular"}, readOutputTexts(t, mockStdout),
		"restarting clears error outputs immediately")

	module.Output(outputsWithError)
	module.Close()
	out = readOutput(t, mockStdout)
	assert.Equal(t, 3, len(out), "All segments in output")

	regularSegmentName = out[1]["name"].(string)
	mockStdin.WriteString(fmt.Sprintf(`{"name": "%s", "button": 1},`, regularSegmentName))
	assert.Equal(t, []string{"regular"}, readOutputTexts(t, mockStdout),
		"restarting from regular segment also clears errors")
}

type segmentAssertions struct {
	*testing.T
	actual   bar.Segment
	Expected map[string]string
}

func (s segmentAssertions) AssertEqual(message string) {
	actualMap := make(map[string]string)
	for k, v := range i3map(s.actual) {
		actualMap[k] = fmt.Sprintf("%v", v)
	}
	assert.Equal(s.T, s.Expected, actualMap, message)
}

func TestI3Map(t *testing.T) {
	segment := bar.TextSegment("test")
	a := segmentAssertions{t, segment, make(map[string]string)}

	a.Expected["full_text"] = "test"
	a.Expected["markup"] = "none"
	a.AssertEqual("sets full_text")

	segment2 := segment.ShortText("t")
	a2 := segmentAssertions{t, segment2, make(map[string]string)}
	a2.Expected["full_text"] = "test"
	a2.Expected["short_text"] = "t"
	a2.Expected["markup"] = "none"
	a2.AssertEqual("sets short_text, does not lose full_text")

	segment3 := bar.PangoSegment("<b>bold</b>")
	a3 := segmentAssertions{t, segment3, make(map[string]string)}
	a3.Expected["full_text"] = "<b>bold</b>"
	a3.Expected["markup"] = "pango"
	a3.AssertEqual("markup set for pango segment")

	assert.Equal(t, "test", segment.Text(), "text getter")
	assert.Equal(t, "test", segment2.Text(), "text getter")

	a.Expected["short_text"] = "t"
	a.AssertEqual("mutates in place")

	segment.Color(color.RGBA{0xff, 0x00, 0x00, 0xff})
	a.Expected["color"] = "#ff0000"
	a.AssertEqual("sets color value")

	segment.Color(nil)
	delete(a.Expected, "color")
	a.AssertEqual("clears color value when blank")

	segment.Background(nil)
	a.AssertEqual("clearing unset color works")

	segment.Background(color.RGBA{0x00, 0x77, 0x00, 0x77})
	a.Expected["background"] = "#00ff00"
	a.AssertEqual("sets background color")

	segment.Border(color.Transparent)
	// i3 doesn't support alpha in colours yet.
	a.Expected["border"] = "#000000"
	a.AssertEqual("sets border color")

	segment.Align(bar.AlignStart)
	a.Expected["align"] = "left"
	a.AssertEqual("alignment strings are preserved")

	segment.MinWidth(10)
	a.Expected["min_width"] = "10"
	a.AssertEqual("sets min width in px")

	segment.MinWidthPlaceholder("00:00")
	a.Expected["min_width"] = "00:00"
	a.AssertEqual("sets min width placeholder")

	// sanity check default go values.
	segment.Separator(false)
	a.Expected["separator"] = "false"
	a.AssertEqual("separator = false")

	segment.Padding(0)
	a.Expected["separator_block_width"] = "0"
	a.AssertEqual("separator width = 0")

	segment.Urgent(false)
	a.Expected["urgent"] = "false"
	a.AssertEqual("urgent = false")

	segment.Identifier("ident")
	a.Expected["instance"] = "ident"
	a.AssertEqual("opaque instance")
}
