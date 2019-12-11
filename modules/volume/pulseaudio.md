---
title: PulseAudio
---

Provides volume status/control using the PulseAudio D-Bus API.

## Usage

```go
volume.New(pulseaudio.Sink(sinkName))
```

or

```go
volume.New(pulseaudio.DefaultSink())
```
