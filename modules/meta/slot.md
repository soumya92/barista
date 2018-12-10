---
title: Slot
---

Slot creates multiple named slots that can hold module output, and allows the active slot to be
selected at runtime, allowing limited repositioning of module output.

Creating a `Slotter`: `slotter := slot.New(existingModule)`.  
Creating named slots: `slot1, slot2 := slotter.Slot("1"), slotter.Slot("2")`.  
Switching slots at runtime: `slotter.Activate("1")`.

## Example

<div class="module-example-out"><span>SEA 20:34</span><span>Tue</span><span>LON 04:34</span></div>
<div class="module-example-out"><span>Mon</span><span>SEA 04:34</span><span>LON 12:34</span></div>
Slotting the weekday to show where the date changes:

```go
sea, _ := time.LoadLocation("America/Los_Angeles")
lon, _ := time.LoadLocation("Europe/London")
seaTime := clock.Zone(sea).OutputFormat("SEA 15:04")
lonTime := clock.Zone(lon).OutputFormat("LON 15:04")
day := clock.Zone(lon).OutputFormat("Mon")

s := slot.New(day)
same, diff := s.Slot("same"), s.Slot("diff")

go func() {
	everyMin := timing.NewScheduler().Every(time.Minute)
	for everyMin.Tick() {
		now := timing.Now()
		if now.In(lon).Format("Mon") == now.In(sea).Format("Mon") {
			s.Activate("same")
		} else {
			s.Activate("diff")
		}
	}
}()

barista.Run(same, seaTime, diff, lonTime)
```
