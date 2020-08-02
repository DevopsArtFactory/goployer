package tool

import (
	"fmt"
	"github.com/fatih/color"
	"io"
	"strings"
)

type Color struct {
	color *color.Color
}

var (
	Red    = Color{color: color.New(color.FgRed)}
	Blue   = Color{color: color.New(color.FgBlue)}
	Green  = Color{color: color.New(color.FgGreen)}
	Yellow = Color{color: color.New(color.FgYellow)}
	Cyan   = Color{color: color.New(color.FgCyan)}
)

// Fprintln outputs the result to out, followed by a newline.
func (c Color) Fprintln(out io.Writer, a ...interface{}) {
	if c.color == nil {
		fmt.Fprintln(out, a...)
		return
	}

	fmt.Fprintln(out, c.color.Sprint(strings.TrimSuffix(fmt.Sprintln(a...), "\n")))
}

// Fprintf outputs the result to out.
func (c Color) Fprintf(out io.Writer, format string, a ...interface{}) {
	if c.color == nil {
		fmt.Fprintf(out, format+"\n", a...)
		return
	}

	fmt.Fprint(out, c.color.Sprintf(format+"\n", a...))
}
