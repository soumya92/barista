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

package click

import (
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"testing"

	"barista.run/bar"
	"github.com/stretchr/testify/require"
)

func makeFunc() (do func(), check func() bool) {
	result := make(chan bool, 10)
	return func() { result <- true }, func() bool {
		select {
		case val := <-result:
			return val
		default:
			return false
		}
	}
}

const notClicked = bar.Button(-2)

func makeHandler() (do func(bar.Event), check func() bar.Button) {
	result := make(chan bar.Button, 10)
	return func(e bar.Event) { result <- e.Button }, func() bar.Button {
		select {
		case val := <-result:
			return val
		default:
			return notClicked
		}
	}
}

func randomEvent(btn bar.Button) bar.Event {
	e := bar.Event{Button: btn}
	e.Width = rand.Intn(190) + 10
	if rand.Float32() > 0.5 {
		e.Height = 40
	} else {
		e.Height = 20
	}
	e.X = rand.Intn(e.Width)
	e.Y = rand.Intn(e.Height)
	e.ScreenX = rand.Intn(3840-e.Width) + e.X
	if rand.Float32() > 0.5 {
		e.ScreenY = 2160 - e.Height + e.Y
	} else {
		e.ScreenY = e.Y
	}
	return e
}

func triggerHandler(handler func(bar.Event), btns ...bar.Button) {
	if len(btns) == 0 {
		handler(randomEvent(bar.ButtonLeft))
	}
	for _, b := range btns {
		handler(randomEvent(b))
	}
}

func TestDiscard(t *testing.T) {
	do, check := makeFunc()
	triggerHandler(DiscardEvent(do))
	require.True(t, check(), "Original function is triggered")
}

func TestButton(t *testing.T) {
	do, check := makeFunc()
	handler := Button(func(bar.Button) { do() }, bar.ButtonLeft, bar.ButtonRight)

	triggerHandler(handler,
		bar.ScrollUp, bar.ScrollLeft, bar.ButtonMiddle, bar.ButtonForward)
	require.False(t, check(), "Handler should not call original function")

	triggerHandler(handler, bar.ButtonLeft)
	require.True(t, check(), "Handler calls original function on left click")

	triggerHandler(handler, bar.ButtonRight)
	require.True(t, check(), "Handler calls original function on left click")

	handler, checkE := makeHandler()
	triggerHandler(handler, bar.ButtonMiddle)
	require.Equal(t, bar.ButtonMiddle, checkE())

	handler = ButtonE(handler, bar.ScrollUp)
	triggerHandler(handler, bar.ScrollDown)
	require.Equal(t, notClicked, checkE())

	triggerHandler(handler, bar.ScrollUp)
	require.Equal(t, bar.ScrollUp, checkE())
}

func TestRunLeft(t *testing.T) {
	dir, err := ioutil.TempDir("", "scratch")
	if err != nil {
		t.Fatalf("failed to create test directory: %s", err)
	}
	defer os.RemoveAll(dir)
	file := path.Join(dir, "foo")

	handler := RunLeft("touch", file)
	triggerHandler(handler, bar.ButtonRight, bar.ButtonMiddle, bar.ButtonBack)
	triggerHandler(handler, bar.ScrollUp, bar.ScrollRight)

	_, err = os.Stat(file)
	require.Error(t, err, "file not created when not left-clicked")

	triggerHandler(handler, bar.ButtonLeft)
	_, err = os.Stat(file)
	require.NoError(t, err, "file created when left-clicked")
}

func TestClickAndScroll(t *testing.T) {
	do, check := makeFunc()
	handler := Click(do)

	triggerHandler(handler, bar.ScrollUp, bar.ScrollDown, bar.ButtonBack)
	require.False(t, check(), "only left/right/middle click should trigger")

	triggerHandler(handler, bar.ButtonMiddle)
	require.True(t, check())

	handler = Click(do, true)
	triggerHandler(handler, bar.ButtonBack)
	require.True(t, check(), "Back should trigger with includeBackAndForward")

	triggerHandler(handler, bar.ScrollUp)
	require.False(t, check())

	doInc, checkInc := makeFunc()
	doDec, checkDec := makeFunc()
	scrollFunc := func(b bar.Button) {
		switch b {
		case bar.ScrollDown, bar.ScrollRight:
			doInc()
		case bar.ScrollUp, bar.ScrollLeft:
			doDec()
		default:
			require.Fail(t, "Expected only scroll events", "got %d", b)
		}
	}

	handler = Scroll(scrollFunc)
	triggerHandler(handler, bar.ButtonLeft, bar.ButtonMiddle, bar.ButtonBack)

	triggerHandler(handler, bar.ScrollUp, bar.ScrollLeft)
	require.True(t, checkDec())
	require.True(t, checkDec())
	require.False(t, checkInc())

	triggerHandler(handler, bar.ScrollDown, bar.ScrollRight)
	require.True(t, checkInc())
	require.True(t, checkInc())
	require.False(t, checkDec())
}

func TestClickMap(t *testing.T) {
	handlerL, checkL := makeHandler()
	handlerUp, checkUp := makeHandler()
	funcBack, checkBack := makeFunc()
	funcElse, checkElse := makeFunc()

	m := Map{bar.ButtonLeft: handlerL}
	handler := m.
		ScrollUpE(handlerUp).
		Back(funcBack).
		Else(DiscardEvent(funcElse)).
		Handle

	triggerHandler(handler, bar.ButtonLeft)
	require.Equal(t, bar.ButtonLeft, checkL(), "specified directly in Map")
	require.False(t, checkElse())

	triggerHandler(handler, bar.ScrollUp)
	require.Equal(t, bar.ScrollUp, checkUp(), "specified as an event handler")
	require.False(t, checkElse())

	triggerHandler(handler, bar.ButtonBack)
	require.True(t, checkBack(), "specified as a simple function")
	require.False(t, checkElse())

	triggerHandler(handler, bar.ButtonRight)
	require.True(t, checkElse(), "fallback")
}

func verifySingleButton(t *testing.T, btn bar.Button, handler func(bar.Event),
	verifyFunc func() interface{}, trueValue interface{}) {

	for _, b := range []bar.Button{
		bar.ButtonLeft, bar.ButtonRight, bar.ButtonMiddle,
		bar.ButtonBack, bar.ButtonForward,
		bar.ScrollUp, bar.ScrollLeft, bar.ScrollRight, bar.ScrollDown,
	} {
		triggerHandler(handler, b)
		if b == btn {
			require.Equal(t, trueValue, verifyFunc(),
				"Expected %d to trigger handler", b)
		} else {
			require.NotEqual(t, trueValue, verifyFunc(),
				"Expected %d not to trigger handler", b)
		}
	}
}

func TestButtonFuncs(t *testing.T) {
	buttonFuncs := map[bar.Button]func(func()) func(bar.Event){
		bar.ButtonLeft:    Left,
		bar.ButtonRight:   Right,
		bar.ButtonMiddle:  Middle,
		bar.ButtonBack:    Back,
		bar.ButtonForward: Forward,
		bar.ScrollUp:      ScrollUp,
		bar.ScrollDown:    ScrollDown,
		bar.ScrollLeft:    ScrollLeft,
		bar.ScrollRight:   ScrollRight,
	}
	fn, check := makeFunc()
	for btn, wrap := range buttonFuncs {
		verifySingleButton(t, btn, wrap(fn),
			func() interface{} { return check() }, true)
	}

	buttonEFuncs := map[bar.Button]func(func(bar.Event)) func(bar.Event){
		bar.ButtonLeft:    LeftE,
		bar.ButtonRight:   RightE,
		bar.ButtonMiddle:  MiddleE,
		bar.ButtonBack:    BackE,
		bar.ButtonForward: ForwardE,
		bar.ScrollUp:      ScrollUpE,
		bar.ScrollDown:    ScrollDownE,
		bar.ScrollLeft:    ScrollLeftE,
		bar.ScrollRight:   ScrollRightE,
	}
	handler, checkE := makeHandler()
	for btn, wrap := range buttonEFuncs {
		verifySingleButton(t, btn, wrap(handler),
			func() interface{} { return checkE() }, btn)
	}
}

func TestMapButtonFuncs(t *testing.T) {
	ch := make(chan string, 10)
	sendCh := func(v string) func() {
		return func() {
			ch <- v
		}
	}
	m := Map{}.
		Left(sendCh("L")).
		Right(sendCh("R")).
		Middle(sendCh("|")).
		Back(sendCh("<-")).
		Forward(sendCh("->")).
		ScrollUp(sendCh("^")).
		ScrollRight(sendCh(">")).
		ScrollLeft(sendCh("<")).
		ScrollDown(sendCh("v")).
		Else(DiscardEvent(sendCh("else")))

	expected := map[bar.Button]string{
		bar.ButtonLeft:    "L",
		bar.ButtonRight:   "R",
		bar.ButtonMiddle:  "|",
		bar.ButtonBack:    "<-",
		bar.ButtonForward: "->",
		bar.ScrollUp:      "^",
		bar.ScrollDown:    "v",
		bar.ScrollLeft:    "<",
		bar.ScrollRight:   ">",
	}
	for btn, str := range expected {
		verifySingleButton(t, btn, m.Handle,
			func() interface{} { return <-ch }, str)
	}
}
