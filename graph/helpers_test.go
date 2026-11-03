package graph_test

import (
	"reflect"
	"testing"

	"github.com/vitche/go_compute_graph/graph"
)

// recorder is a pass-through node that records every Process call together
// with the input it saw. It can be configured to fail.
type recorder struct {
	graph.BaseNode
	name   string
	log    *[]string // shared activation log, optional
	inputs [][]any
	err    error
}

func newRecorder(name string, log *[]string) *recorder {
	return &recorder{name: name, log: log}
}

func (r *recorder) Process() error {
	if r.log != nil {
		*r.log = append(*r.log, r.name)
	}
	in := r.Read()
	r.inputs = append(r.inputs, in)
	if r.err != nil {
		return r.err
	}
	r.Arguments(in)
	return nil
}

func (r *recorder) calls() int { return len(r.inputs) }

// writeSpy records every slice passed to Write (without copying) and never
// processes anything itself.
type writeSpy struct {
	graph.BaseNode
	writes    [][]any
	processed int
}

func (w *writeSpy) Write(data []any) {
	w.writes = append(w.writes, data)
	w.BaseNode.Write(data)
}

func (w *writeSpy) Process() error {
	w.processed++
	return nil
}

func assertEqual(t *testing.T, what string, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: got %#v, want %#v", what, got, want)
	}
}

func mustPanic(t *testing.T, what string, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if recover() == nil {
			t.Fatalf("%s: expected a panic", what)
		}
	}()
	f()
}
