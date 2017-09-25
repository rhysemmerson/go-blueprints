package trace 

import (
	"testing"
	"bytes"
)

func TestNew(t *testing.T) {
	var buf bytes.Buffer
	tracer := New(&buf)
	if tracer == nil {
		t.Error("Return from New should not be nil")
		return
	}

	tracer.Trace("Hello trace package.")
	if buf.String() != "Hello trace package.\n" {
		t.Errorf("Unexpected string written '%s'.", buf.String())
	}
}

func TestOff(t *testing.T) {
	var buf bytes.Buffer
	var silentTracer Tracer = Off()
	silentTracer.Trace("something")
	if buf.String() != "" {
		t.Errorf("Unexpected string written '%s'. Should not have written anything.", buf.String())
	}
}