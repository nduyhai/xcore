package xerr

import "runtime"

type StackTrace interface {
	Format() []string
}

type stackTrace struct {
	frames []StackFrame
	sep    string
}

func NewStackTrace() StackTrace {
	var pcs [32]uintptr
	n := runtime.Callers(0, pcs[:])
	st := make([]StackFrame, 0, n)
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		st = append(st, newStackFrame(frame.Function, frame.File, frame.Line))
		if !more {
			break
		}
	}
	return &stackTrace{
		frames: st,
		sep:    ":",
	}
}

func (s stackTrace) Format() []string {
	var str []string
	for _, f := range s.frames {
		str = append([]string{f.Format(s.sep)}, str...)
	}
	return str
}
