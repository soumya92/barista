---
title: Shell
---

Show the output of a shell command on the bar: `shell.New("sh", "-c", "whoami")`.  
Show the last line of output from a long-running shell command: `shell.Tail("dmesg")`.

## Configuration

* `Output(func(string) bar.Output)`: Sets the output format. For `New()`, this function is given the
  complete, trimmed output from the command. For `Tail()` it is given a single line from the output.

* `Every(time.Duration)`: Only for `New()`, sets the interval at which the command should be
  repeated. If not set, or set to a zero duration, disables automatic refresh.

* `Refresh`: Refreshes the output. For `New()`, also re-runs the command. For `Tail()` re-uses the
  last line of output. Can be used if the output format has a time, or in click handlers, e.g.

  ```go
s.Output(func (in string) bar.Output {
  	return outputs.Text(in).OnClick(ifLeft(s.Refresh))
})
```

## Examples

<div class="module-example-out">8 chrome procs</div>
Show a count of chrome processes, refreshed every second:

```go
chromeCount := shell.New("pgrep", "-c", "chrome").
	Every(time.Second).
	Output(func(count string) bar.Output {
		return outputs.Textf("%s chrome procs", count)
	})
```

<div class="module-example-out">(22m15s ago) wlp60s0: associated</div>
Show the last line of dmesg output, with human readable time deltas:

```go
var dmesgFormat = regexp.MustCompile(`^\[([0-9\.]+)\] (.*)$`)
dmesg := shell.Tail("dmesg", "-w").Output(func(line string) bar.Output {
	res := dmesgFormat.FindStringSubmatch(line)     // res[1] = time, res[2] = message
	uptimeStr, _ := ioutil.ReadFile("/proc/uptime") // uptimeStr = "$uptime $cputime"

	timeOfMsg, _ := strconv.ParseFloat(res[1], 64)
	timeNow, _ := strconv.ParseFloat(strings.Split(string(uptimeStr), " ")[0], 64)

	delta := time.Duration(uint64(timeNow-timeOfMsg)) * time.Second
	outLine := res[2]
	if len(outLine) > 20 {
		outLine = outLine[0:19] + "…"
	}

	return outputs.Textf("(%v ago) %s", delta, outLine)
})
go func() {
	for range timing.NewScheduler().Every(time.Second).Tick() {
		dmesg.Refresh()
	}
}()
```
