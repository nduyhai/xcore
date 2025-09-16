package xerr

import "fmt"

type StackFrame interface {
	Function() string
	File() string
	Line() int
	Format(sep string) string
}
type stackFrame struct {
	name string
	file string
	line int
}

func newStackFrame(name string, file string, line int) *stackFrame {
	return &stackFrame{name: name, file: file, line: line}
}

func (f stackFrame) Function() string {
	return f.name
}

func (f stackFrame) File() string {
	return f.file
}

func (f stackFrame) Line() int {
	return f.line
}

func (f stackFrame) Format(sep string) string {
	return fmt.Sprintf("%v%v%v%v%v", f.name, sep, f.file, sep, f.line)
}
