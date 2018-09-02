---
title: Writing a Custom Module
pkg: none
---

In this guide we will write a custom module that shows the title of the currently focussed i3
window. This will demonstrate channels, using external go packages, and  some building blocks
available in `base/`.

## Define a Module struct

Most modules are implemented as simple structs that store state and configuration. Our module's
configuration is fairly simple, just a formatting function that receives the window title `string`
and returns a `bar.Output`.

```go
type Module struct {
	formatFunc value.Value // of func(string) bar.Output
}
```

Although new(Module) will work, the `formatFunc` value will be empty. So it's preferable to provide
a `New() *Module` function that fills in a sensible default:

```go
func New() *Module {
	m := new(Module)
	m.formatFunc.Set(func(in string) bar.Output {
		return outputs.Text(in)
	})
	return m
}
```

## Implement the Stream method

This is the only required method for a module, so let's start by implementing a version that simply
listens for window updates and sends output to the bar:

```go
func (m *Module) Stream(s bar.Sink) {
	recv := i3.Subscribe(i3.WindowEventType)
	for recv.Next() {
	    ev := recv.Event().(*i3.WindowEvent)
	    if ev.Change != "focus" {
	    	continue
	    }
	    format := m.formatFunc.Get().(func(string) bar.Output)
	    s.Output(format(ev.Container.Name))
	}
	s.Error(recv.Close())
}
```

## Provide Output customisation

At this point the module simply prints the currently focussed window's complete title. To allow
customising the display, let's add a function that stores new format functions:

```go
func (m *Module) Output(format func(string) bar.Output) *Module {
	m.formatFunc.Set(format)
	return m
}
```

At this point, everything *almost* works. The only remaining problem is that format functions do not
become effective until the next update. This is where using a base/value is helpful, because it
provides notifications whenever the value changes.

Now we need to `select`: any time either the format function or the active window title changes, we
need to send a new output to the bar. Unfortunately the i3 subscription is not a channel, so let's
wrap it in a function that does output to a channel:

```go
func (m *Module) windowTitles(title chan<- string, err chan<- error) {
	recv := i3.Subscribe(i3.WindowEventType)
	for recv.Next() {
	    ev := recv.Event().(*i3.WindowEvent)
	    if ev.Change != "focus" {
	    	continue
	    }
	    title <- ev.Container.Name
	}
	err <- recv.Close()
}
```

and update the Stream function to select:

```go
func (m *Module) Stream(s bar.Sink) {
	titles := make(chan string)
	errs := make(chan error)
	go m.windowTitles(titles, errs)
	var title string
	format := m.formatFunc.Get().(func(string) bar.Output)
	for {
		select {
			case title = <-titles:
			case <-m.formatFunc.Next():
				format = m.formatFunc.Get().(func(string) bar.Output)
			case e := <-errs:
				s.Error(e)
				return
		}
		s.Output(format(title))
	}
}
```

## Handling the initial state

Optionally, let's add a function to get the current title when the module starts, which avoids the
empty `var title string` (and moves the Output to before the select):

```go
func getCurrentTitle() string {
	tree, _ := i3.GetTree()
	focussed := tree.Root.FindFocused(func(n *Node) bool {
		return n.Window != 0
	})
	if focussed != nil {
		return focussed.Name
	}
	return ""
}

func (m *Module) Stream(s bar.Sink) {
	// ...
	title := getCurrentTitle()
	// ...
	for {
		s.Output(format(title))
		select {
			// ...
		}
	}
}
```

## Publishing a Module

All this code can now be placed in its own `go get`table package, which makes it available for
anyone to use. The usage is now as simple as:

```go
import (
	"barista.run"
	"barista.run/bar"
	"example.org/barista/i3window"
)

func main() {
	windowModule := i3window.New().Output(func(title string) bar.Output {
		if len(title) < 20 {
			return outputs.Text(title)
		}
		return outputs.Textf("%s...", title[0:17])
	})
	barista.Run(windowModule)
}
```

## Next Steps

Left as an exercise to the reader: Try adding a 'Controller' interface to the module, allowing click
actions to interact with the focussed window (e.g. allowing right-click to close).

This will require changing the output function to be something that takes a Node (or custom type)
rather than just a string, and provides methods to perform actions on the window.

# Further Reading

The source of the built-in modules have attempted to be good examples of how to write modules. They
cover a wide variety of module types and capabilities, so when writing a new module, it's worth
looking at existing ones to get an idea of how to implement common patterns. For example, any
repeated task should use a Scheduler (mentioned below), which you can see in most built-in modules.

## The `timing` Package

By default, barista handles pause/resume signals sent by i3bar when its visibility changes, and uses
these to pause/resume schedulers created by `timing.NewScheduler()`. This suspends processing while
the bar is hidden, and coalesces all updates to when the bar is next visible. For this reason,
whenever possible, use the timing package for scheduling over `time.Sleep` or `time.After/Func`
since those will fire even when the bar is hidden.

See the [timing package docs](/timing) for more details.

## The `base/*` Packages

The base package provides some building blocks that can be useful when writing modules:

- [`base/Notifier`](/base/notifier): Communicate changed values while discarding intermediate
  updates if the receiver is not ready.

- [`base/Sink`](/base/sink): Create sinks backed by a channel of outputs, or a sink that discards
  everything sent to it.

- [`base/Value`](/base/value): Provides value.Value and value.ErrorValue to store configuration
  or data values and listen for changes.
