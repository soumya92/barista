---
title: Pango Formatting
---

When using a Pango font, i3bar supports formatting the output using
[Pango Markup](https://docs.gtk.org/Pango/pango_markup.html). This allows rich
in-line formatting in each segment of the output.

Barista provides formatting primitives that allow type-safe construction of markup strings,
wutomatically handling escaping and some unit conversions. A pango "document" is just a set of
`*pango.Node`s, which stringify together to create the final markup. Nodes provide methods to
set formatting and add additional nodes to build up a "document".

## Text Nodes

Created using `pango.Text("...")`, or `pango.Textf("foo is %v", ...)`, text nodes automatically
escape their content to avoid conflicts and invalid pango markup. So `Text("<b>bold</b>")` will
print `<b>bold</b>` on the bar, and not bold text.

## Group Nodes

Nodes can contain other nodes, useful if some styles need to be applied across all of them. It can
also sometimes simplify the construction code if the document is built in pieces even without any
shared formatting. For example:

```go
out := new(pango.Node).Color(orange)
if info.isPlaying() {
	out.AppendTextf("%v/", info.CurrentTime())
}
out.AppendTextf("%v", info.Length).ConcatTextf("- %s", out.Title)
return out // <span color="#f70">1m3s/4m33s</span> - Song Name
```

## Building up

As demonstrated in the example above, there are two ways to add content to an existing node:

* `Append(Node...)`/`AppendText(string...)`/`AppendTextf(format string, interface{}...)`

  Append adds nodes as children of the current node, so they will inherit all styling. e.g.

  ```go
  pango.Text("foo").Color(red).Append(pango.Text("bar").Bold())
  // <span color='#f00'>foo<span weight='bold'>bar</span></span>
  ```

  Here `Color(red)` applies to both the original "foo" text node, and the newly appended "bar" text
  node, but `Bold()` only applies to the "bar" text node.

* `Concat(Node...)`/`ConcatText(string...)`/`ConcatTextf(format string, interface{}...)`

  Concat adds nodes adjacent to the current node, and changes further operations to act on a new
  group node that contains the current node and the concatenated node(s).

  For example, 

  ```go
  pango.Text("foo").Bold().ConcatText("bar").UnderlineError()
  // <span underline='error'><span weight='bold'>foo</span>bar</span>
  ```

  Here `Bold()` only applies to the "foo" text node, but after concatenating the "bar" text,
  `UnderlineError()` applies to the grouping node that contains both the "foo" and "bar" nodes.


## Formatting Methods

Once a pango node is created using Text/Textf, it can then be formatted using the many available
formatting methods. See the [pango.Node godoc](https://godoc.org/github.com/soumya92/barista/pango#Node)
for the complete list.

Most keywords, such as `x-small` (size), `expanded` (stretch), or `medium` (weight), have been added
directly to `*Node`, to allow code like:

```go
pango.Text("foo").Bold().XLarge()
```

However, the `normal` keyword would be ambiguous, so it has been prefixed with the attribute, e.g.
`WeightNormal()`, `StretchNormal()`, etc.

Most methods only change one attribute, but `Color(color.Color)` and `Background(color.Color)`  set
both the colour attribute and the alpha attribute to values obtained from the color.Color.

`Smaller()` and `Larger()` are also special, in that they wrap content rather than set an attribute.
This means that multiple invocations on the same node produce the expected result. i.e.,

```go
pango.Text("tiny").Smaller().Smaller().Smaller()
```

produces tiny text that is three sizes smaller than normal. If Smaller merely set the size='smaller',
this code would produce text that was only one size smaller than normal.
