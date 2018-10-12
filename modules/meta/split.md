---
title: Split
---

Split splits the output from an existing module into two distinct modules,
making it easy to show part of the modules output in a different location or
on-demand instead of always.

Splitting a module output: `first2, remaining := split.New(existingModule, 2)`.

## Example

<div class="module-example-out"><span>Mail:5</span><span>+</span></div>
<div class="module-example-out"><span>Mail:5</span><span>&gt;</span><span>ToDo:2</span><span>Important:1</span><span>Bugs:2</span><span>&lt;</span></div>
Splitting up unread message counts to show inbox in the main bar, and all others in a group:

```go
labels := []string{"INBOX", "ToDo", "Important", "Bugs"}
mail := mailProvider.New(labels...).
	Output(func(m mailProvider.Info) bar.Output {
		o := outputs.Group()
		for _, lbl := range labels {
			o.Append(outputs.Textf("%s:%d", lbl, m[lbl]))
		}
		return o
	})
inbox, others := split.New(mail, 1)

// Add inbox, and hide others behind a collapsible group.
grp, _ := collapsing.Group(others)
barista.Run(inbox, grp)
```
