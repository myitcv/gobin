<!-- __JSON: go list -json .
## `{{ filepathBase .Out.ImportPath}}`

{{.Out.Doc}}

```
go get -u {{.Out.ImportPath}}
```
-->
## `mdreplace`

mdreplace is a tool to help you keep your markdown README/documentation current.

```
go get -u myitcv.io/cmd/mdreplace
```
<!-- END -->

_(will soon be available as a [`vgo` module](https://github.com/golang/go/issues/24301))_

A common problem with non `.go` documentation files is that their contents can easily become stale. For example with a
program it's common to include a "Help" section in the corresponding `README.md` which typically involves a discussion
of the program's flags. The contents of the `README.md` can, however, easily fall out of step with respect to the
_actual_ flags were we to run our program with `-h` (or equivalent) today.

`mdreplace` helps alleviate these problems by allowing you to insert special comment blocks in your markdown files that
are replaced with command output.

For example, were we to include the following special comment block:

<!-- __TEMPLATE: cat _examples/hello_world_today
{{.Out -}}
-->
    <!-- __TEMPLATE: echo -n "hello world" today
    {{.Out -}}
    -->
    <!-- END -->
<!-- END -->

results in:

---
<!-- __TEMPLATE: sh -c "cat _examples/hello_world_today | sed -e 's/^    //' | mdreplace -strip"
{{.Out}}
-->
hello world today
<!-- END -->
---

_To see this in action, look at the [source of the
`README.md`](https://raw.githubusercontent.com/myitcv/x/master/cmd/mdreplace/README.md) you are currently reading._


<!-- __TEMPLATE: sh -c "${DOLLAR}(go list -f '{{.ImportPath}}' | xargs basename) -h"
### Usage

```
{{.Out -}}
```
-->
### Usage

```
Usage:

  mdreplace file1 file2 ...
  mdreplace

When called with no file arguments, mdreplace works with stdin
Flags:
  -debug
    	whether to print debug information of not
  -long
    	run LONG blocks
  -online
    	run ONLINE blocks
  -strip
    	whether to strip special comments from the file
  -w	whether to write back to input files (cannot be used when reading from
    	stdin)

```
<!-- END -->

### Code fences

Code fences can appear within templates. Hence the following special template within a markdown file:

<!-- __TEMPLATE: cat _examples/code_fence
{{.Out -}}
-->
    <!-- __TEMPLATE: echo -n "hello world"
    ```go
    package main

    import "fmt"

    func main() {
            fmt.Println("{{.Out}}")
    }
    ```
    -->
    <!-- END -->
<!-- END -->

results in:

---
<!-- __TEMPLATE: sh -c "cat _examples/code_fence | sed -e 's/^    //' | mdreplace -strip"
{{.Out}}
-->
```go
package main

import "fmt"

func main() {
        fmt.Println("hello world")
}
```

<!-- END -->
---


The only place that special comment blocks are _not_ interpreted by `mdreplace` is within code blocks. Hence how we are
able to render the example special code blocks in this README.

_Note it is not possible to nest code fences._

### JSON blocks

The `__JSON` block is used where the output from the command is valid JSON. This JSON is then unmarshalled and passed as
an argument to the template block. For example:

<!-- __TEMPLATE: cat _examples/json_block
{{.Out -}}
-->
    <!-- __JSON: go list -json encoding/json
    Package `{{.Out.ImportPath}}` has name `{{.Out.Name}}` and the following doc string:

    ```
    {{.Out.Doc}}
    ```
    -->
    <!-- END -->
<!-- END -->

results in :

---
<!-- __TEMPLATE: sh -c "cat _examples/json_block | sed -e 's/^    //' | mdreplace -strip"
{{.Out}}
-->
Package `encoding/json` has name `json` and the following doc string:

```
Package json implements encoding and decoding of JSON as defined in RFC 7159.
```

<!-- END -->
---

### Variable expansion

Variable expansion also works; use the special `$DOLLAR` variable to expand to the literal `$` sign:

<!-- __TEMPLATE: cat _examples/variable_expansion
{{.Out -}}
-->
    <!-- __TEMPLATE: sh -c "BANANA=fruit; echo -n \"${DOLLAR}BANANA\""
    {{.Out}}
    -->
    <!-- END -->
<!-- END -->

results in :

---
<!-- __TEMPLATE: sh -c "cat _examples/variable_expansion | sed -e 's/^    //' | mdreplace -strip"
{{.Out}}
-->
fruit

<!-- END -->
---

### Template functions

Both the `__TEMPLATE` and `__JSON` blocks support the following template functions:

* `lines(string) []string` - split a string into lines
* `lineEllipsis(s string, n int) string)` - output at most `n` lines from `s`, adding ellipsis if required
* `trimLinePrefixWhitespace(s string, m string) string` - remove lines from `s`, upto and including the line
  matching `m`, as well as any blank lines that follow `m`
* ... more to follow

_TODO: move these to be an internal package that can then be automatically documented._


### Implementation

This rather basic program is an implementation of the techniques proposed by [Rob Pike](https://twitter.com/rob_pike) in
his brilliant presentation [Lexical Scanning in Go](https://youtu.be/HxaD_trXwRE)
([slides](https://talks.golang.org/2011/lex.slide#1)).

