/*
copyright 2020 the Goployer authors

licensed under the apache license, version 2.0 (the "license");
you may not use this file except in compliance with the license.
you may obtain a copy of the license at

    http://www.apache.org/licenses/license-2.0

unless required by applicable law or agreed to in writing, software
distributed under the license is distributed on an "as is" basis,
without warranties or conditions of any kind, either express or implied.
see the license for the specific language governing permissions and
limitations under the license.
*/

package tool

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
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
