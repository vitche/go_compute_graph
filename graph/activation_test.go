package graph_test

import (
	"errors"
	"testing"

	"github.com/vitche/go_compute_graph/graph"
)

func TestActivateNilNode(t *testing.T) {
	if err := graph.Activate(nil); !errors.Is(err, graph.ErrNilNode) {
		t.Fatalf("got %v, want ErrNilNode", err)
	}
}

func TestActivateProcessesThenForwardsToAllSynapsesInOrder(t *testing.T) {
	var log []string
	src := newRecorder("src", &log)
	spy1, spy2 := &writeSpy{}, &writeSpy{}
	// Track write order across both targets.
	order := &orderSpy{log: &log}
	src.Add(graph.NewSynapse().Add(spy1))
	src.Add(graph.NewSynapse().Add(order))
	src.Add(graph.NewSynapse().Add(spy2))
	order.name = "order"

	src.Write([]any{"x", "y"})
	if err := graph.Activate(src); err != nil {
		t.Fatal(err)
	}

	assertEqual(t, "log", log, []string{"src", "order"})
	assertEqual(t, "src input", src.inputs, [][]any{{"x", "y"}})
	if len(spy1.writes) != 1 || len(spy2.writes) != 1 {
		t.Fatalf("each target should get exactly one write, got %d and %d", len(spy1.writes), len(spy2.writes))
	}
	if &spy1.writes[0][0] != &spy2.writes[0][0] {
		t.Fatal("all synapses should receive the same data slice")
	}
	assertEqual(t, "target 1", spy1.Read(), []any{"x", "y"})
	assertEqual(t, "target 2", spy2.Read(), []any{"x", "y"})
	if spy1.processed != 0 || spy2.processed != 0 {
		t.Fatal("writing must never activate a target")
	}
	if got := src.Read(); got != nil {
		t.Fatalf("buffer after activation: got %#v, want nil", got)
	}
}

// orderSpy logs its name when written to.
type orderSpy struct {
	graph.BaseNode
	name string
	log  *[]string
}

func (o *orderSpy) Write(data []any) {
	*o.log = append(*o.log, o.name)
	o.BaseNode.Write(data)
}

func TestActivateSynapseOrder(t *testing.T) {
	var log []string
	src := graph.NewNode()
	for _, name := range []string{"a", "b", "c"} {
		src.Add(graph.NewSynapse().Add(&orderSpy{name: name, log: &log}))
	}
	src.Write([]any{1})
	if err := graph.Activate(src); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "write order", log, []string{"a", "b", "c"})
}

func TestActivateBaseNodePassesThrough(t *testing.T) {
	a, b := graph.NewNode(), graph.NewNode()
	a.Add(graph.NewSynapse().Add(b))
	a.Write([]any{1, "two"})
	if err := graph.Activate(a); err != nil {
		t.Fatal(err)
	}
	if a.Read() != nil {
		t.Fatal("source buffer should be cleared")
	}
	assertEqual(t, "target", b.Read(), []any{1, "two"})
}

func TestActivateWithEmptyBufferForwardsNothing(t *testing.T) {
	a, b := graph.NewNode(), graph.NewNode()
	a.Add(graph.NewSynapse().Add(b))
	if err := graph.Activate(a); err != nil {
		t.Fatal(err)
	}
	if got := b.Read(); got != nil {
		t.Fatalf("target: got %#v, want nil", got)
	}
}

func TestActivateProcessErrorStopsForwarding(t *testing.T) {
	boom := errors.New("boom")
	src := newRecorder("src", nil)
	src.err = boom
	spy := &writeSpy{}
	src.Add(graph.NewSynapse().Add(spy))
	src.Write([]any{1})

	if err := graph.Activate(src); !errors.Is(err, boom) {
		t.Fatalf("got %v, want %v", err, boom)
	}
	if len(spy.writes) != 0 {
		t.Fatal("nothing should be forwarded after a Process error")
	}
}

func TestIterate(t *testing.T) {
	boom := errors.New("boom")
	iterators := map[string]func([]graph.Node) error{
		"Iterate":              graph.Iterate,
		"Activation{}.Iterate": graph.Activation{}.Iterate,
	}
	tests := []struct {
		name    string
		failAt  int // index of the failing node, -1 for none
		count   int
		wantLog []string
	}{
		{"empty", -1, 0, nil},
		{"all succeed", -1, 3, []string{"n0", "n1", "n2"}},
		{"first fails", 0, 3, []string{"n0"}},
		{"middle fails", 1, 3, []string{"n0", "n1"}},
		{"last fails", 2, 3, []string{"n0", "n1", "n2"}},
	}
	for iname, iterate := range iterators {
		for _, tt := range tests {
			t.Run(iname+"/"+tt.name, func(t *testing.T) {
				var log []string
				nodes := make([]graph.Node, tt.count)
				for i := range nodes {
					r := newRecorder("n"+string(rune('0'+i)), &log)
					if i == tt.failAt {
						r.err = boom
					}
					nodes[i] = r
				}
				err := iterate(nodes)
				if tt.failAt >= 0 {
					if !errors.Is(err, boom) {
						t.Fatalf("got %v, want %v", err, boom)
					}
				} else if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				assertEqual(t, "activation log", log, tt.wantLog)
			})
		}
	}
}

func TestIterateNilSlice(t *testing.T) {
	if err := graph.Iterate(nil); err != nil {
		t.Fatal(err)
	}
	if err := (graph.Activation{}).Iterate(nil); err != nil {
		t.Fatal(err)
	}
}

func TestIterateNilNodeElement(t *testing.T) {
	ran := newRecorder("after", nil)
	err := graph.Iterate([]graph.Node{nil, ran})
	if !errors.Is(err, graph.ErrNilNode) {
		t.Fatalf("got %v, want ErrNilNode", err)
	}
	if ran.calls() != 0 {
		t.Fatal("nodes after the failing one must not be activated")
	}
}

func TestIteratePropagatesDataAlongChain(t *testing.T) {
	a, b, c := newRecorder("a", nil), newRecorder("b", nil), newRecorder("c", nil)
	a.Add(graph.NewSynapse().Add(b))
	b.Add(graph.NewSynapse().Add(c))
	a.Write([]any{7})
	if err := graph.Iterate([]graph.Node{a, b, c}); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "c input", c.inputs, [][]any{{7}})
}
