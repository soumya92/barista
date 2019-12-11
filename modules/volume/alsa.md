---
title: ALSA
---

Provides volume status/control using the ALSA C API.

## Usage

```go
volume.New(alsa.Mixer(card, mixer))
```

or

```go
volume.New(alsa.DefaultMixer())
```
