package trace

import (
	"io"
	"fmt"
)

type Tracer interface {
	Trace(...interface{})
}

func New(w io.Writer) Tracer {
	return &tracer{out: w}
}

// Implementation
type tracer struct {
	out io.Writer
}

func (t *tracer) Trace(a ...interface{}) {
	fmt.Fprint(t.out, a...)
	fmt.Fprintln(t.out)
}

// nillTracer will not log anywhere
type nilTracer struct {
	
}

func (t *nilTracer) Trace(a ...interface{}) { }

func Off() Tracer {
	return &nilTracer{}
}