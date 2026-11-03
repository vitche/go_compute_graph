package graph_test

import (
	"errors"
	"testing"

	"github.com/vitche/go_compute_graph/graph"
)

func TestNewSpreadNode(t *testing.T) {
	if graph.NewSpreadNode() == nil {
		t.Fatal("NewSpreadNode returned nil")
	}
}

func TestSpreadNodeDeliversEachItemIndividually(t *testing.T) {
	spread := graph.NewSpreadNode()
	target := newRecorder("t", nil)
	spread.Add(graph.NewSynapse().Add(target))
	spread.Write([]any{"a", "b", "c"})

	if err := graph.Activate(spread); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "target inputs", target.inputs, [][]any{{"a"}, {"b"}, {"c"}})
	if spread.Read() != nil {
		t.Fatal("spread node buffer should be empty after activation")
	}
}

func TestSpreadNodeEmptyInput(t *testing.T) {
	spread := graph.NewSpreadNode()
	target := newRecorder("t", nil)
	spread.Add(graph.NewSynapse().Add(target))
	if err := graph.Activate(spread); err != nil {
		t.Fatal(err)
	}
	if target.calls() != 0 {
		t.Fatal("no items, so the target must not be activated")
	}
}

func TestSpreadNodeSynapseOuterItemInnerOrder(t *testing.T) {
	var log []string
	spread := graph.NewSpreadNode()
	t1, t2 := newRecorder("t1", &log), newRecorder("t2", &log)
	spread.Add(graph.NewSynapse().Add(t1))
	spread.Add(graph.NewSynapse().Add(t2))
	spread.Write([]any{1, 2})

	if err := graph.Activate(spread); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "order", log, []string{"t1", "t1", "t2", "t2"})
	assertEqual(t, "t1 inputs", t1.inputs, [][]any{{1}, {2}})
	assertEqual(t, "t2 inputs", t2.inputs, [][]any{{1}, {2}})
}

func TestSpreadNodeTargetForwardsDownstream(t *testing.T) {
	spread := graph.NewSpreadNode()
	mid := newRecorder("mid", nil)
	sink := graph.NewNode()
	spread.Add(graph.NewSynapse().Add(mid))
	mid.Add(graph.NewSynapse().Add(sink))
	spread.Write([]any{1, 2})

	if err := graph.Activate(spread); err != nil {
		t.Fatal(err)
	}
	// mid is activated per item and forwards each one; sink is not activated,
	// so the items accumulate in its buffer.
	assertEqual(t, "sink buffer", sink.Read(), []any{1, 2})
}

func TestSpreadNodeErrorStopsSpreading(t *testing.T) {
	boom := errors.New("boom")
	var log []string
	spread := graph.NewSpreadNode()
	bad := newRecorder("bad", &log)
	bad.err = boom
	other := newRecorder("other", &log)
	spread.Add(graph.NewSynapse().Add(bad))
	spread.Add(graph.NewSynapse().Add(other))
	spread.Write([]any{1, 2, 3})

	if err := graph.Activate(spread); !errors.Is(err, boom) {
		t.Fatalf("got %v, want %v", err, boom)
	}
	assertEqual(t, "log", log, []string{"bad"})
	assertEqual(t, "bad inputs", bad.inputs, [][]any{{1}})
}

func TestSpreadNodeSynapseWithoutTargetPanics(t *testing.T) {
	// Writing to a target-less synapse is a programming error and panics
	// before Activate's nil check is reached.
	spread := graph.NewSpreadNode()
	spread.Add(graph.NewSynapse()) // no target
	spread.Write([]any{1})
	mustPanic(t, "spread to synapse without target", func() { _ = graph.Activate(spread) })
}

// TestSpreadNodeIterateReactivationQuirk documents the behavior inherited
// from Python: when a spread chain is driven by Iterate, the spread targets
// also appear later in Nodes() and are activated once more with empty input.
func TestSpreadNodeIterateReactivationQuirk(t *testing.T) {
	g := graph.New()
	g.Add(graph.NewSpreadNode())
	target := newRecorder("t", nil)
	g.Add(target)

	if err := g.Write([]any{1, 2}); err != nil {
		t.Fatal(err)
	}
	if err := graph.Iterate(g.Nodes()); err != nil {
		t.Fatal(err)
	}
	if target.calls() != 3 {
		t.Fatalf("target activations: got %d, want 3 (2 items + 1 extra)", target.calls())
	}
	assertEqual(t, "per-item inputs", target.inputs[:2], [][]any{{1}, {2}})
	if len(target.inputs[2]) != 0 {
		t.Fatalf("extra activation input: got %#v, want empty", target.inputs[2])
	}

	// Driving with Activate(g.First()) avoids the extra activation.
	g2 := graph.New()
	g2.Add(graph.NewSpreadNode())
	target2 := newRecorder("t2", nil)
	g2.Add(target2)
	_ = g2.Write([]any{1, 2})
	if err := graph.Activate(g2.First()); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "Activate(First) inputs", target2.inputs, [][]any{{1}, {2}})
}

func TestNewSubgraphActivationNode(t *testing.T) {
	nonEmpty := graph.New()
	first := graph.NewNode()
	nonEmpty.Add(first)
	nonEmpty.Add(graph.NewNode())

	attachOnly := graph.New()
	attachOnly.Attach(graph.NewNode())

	tests := []struct {
		name        string
		sub         *graph.Graph
		wantTargets []graph.Node
	}{
		{"nil", nil, nil},
		{"empty", graph.New(), nil},
		{"attach only", attachOnly, nil},
		{"non-empty", nonEmpty, []graph.Node{first}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := graph.NewSubgraphActivationNode(tt.sub)
			if n.Subgraph != tt.sub {
				t.Fatal("Subgraph field not set")
			}
			assertNodes(t, "targets", targets(n), tt.wantTargets...)
		})
	}
}

func TestSubgraphActivationNodeEmptyErrors(t *testing.T) {
	tests := []struct {
		name string
		node *graph.SubgraphActivationNode
	}{
		{"nil subgraph", graph.NewSubgraphActivationNode(nil)},
		{"zero value", &graph.SubgraphActivationNode{}},
		{"empty subgraph", graph.NewSubgraphActivationNode(graph.New())},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.node.Write([]any{1})
			if err := tt.node.Process(); !errors.Is(err, graph.ErrEmptyGraph) {
				t.Fatalf("Process: got %v, want ErrEmptyGraph", err)
			}
			if tt.node.Read() != nil {
				t.Fatal("input should have been consumed")
			}
		})
	}
}

func TestSubgraphActivationNodeRunsSubgraph(t *testing.T) {
	var log []string
	sub := graph.New()
	s1, s2 := newRecorder("s1", &log), newRecorder("s2", &log)
	sub.Add(s1)
	sub.Add(s2)
	n := graph.NewSubgraphActivationNode(sub)
	n.Write([]any{"x"})

	if err := graph.Activate(n); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "log", log, []string{"s1", "s2"})
	assertEqual(t, "s1 inputs", s1.inputs, [][]any{{"x"}})
	assertEqual(t, "s2 inputs", s2.inputs, [][]any{{"x"}})
	// The activation node forwards its (empty) output to sub.First() via its
	// synapse; since that is nil it is ignored and s1 has nothing pending.
	if s1.Read() != nil {
		t.Fatal("s1 should have no pending input after activation")
	}
}

func TestSubgraphActivationNodeErrorPropagates(t *testing.T) {
	boom := errors.New("boom")
	sub := graph.New()
	bad := newRecorder("bad", nil)
	bad.err = boom
	after := newRecorder("after", nil)
	sub.Add(bad)
	sub.Add(after)
	n := graph.NewSubgraphActivationNode(sub)
	n.Write([]any{1})

	if err := graph.Activate(n); !errors.Is(err, boom) {
		t.Fatalf("got %v, want %v", err, boom)
	}
	if after.calls() != 0 {
		t.Fatal("nodes after the failing one must not run")
	}
}

func TestFuncNode(t *testing.T) {
	boom := errors.New("boom")
	double := func(args []any) ([]any, error) {
		out := make([]any, len(args))
		for i, v := range args {
			out[i] = v.(int) * 2
		}
		return out, nil
	}
	tests := []struct {
		name    string
		fn      func([]any) ([]any, error)
		input   []any
		want    []any
		wantErr error
	}{
		{"transforms", double, []any{1, 2, 3}, []any{2, 4, 6}, nil},
		{"nil fn passes through", nil, []any{1, "a"}, []any{1, "a"}, nil},
		{"nil output", func([]any) ([]any, error) { return nil, nil }, []any{1}, nil, nil},
		{"error", func([]any) ([]any, error) { return []any{"ignored"}, boom }, []any{1}, nil, boom},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := graph.NewFuncNode(tt.fn)
			sink := graph.NewNode()
			n.Add(graph.NewSynapse().Add(sink))
			n.Write(tt.input)

			err := graph.Activate(n)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Activate: got %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if got := n.Read(); got != nil {
					t.Fatalf("output must not be set on error, got %#v", got)
				}
				if sink.Read() != nil {
					t.Fatal("nothing must be forwarded on error")
				}
				return
			}
			assertEqual(t, "forwarded", sink.Read(), tt.want)
		})
	}
}

func TestFuncNodeReceivesClearedInput(t *testing.T) {
	var n *graph.FuncNode
	var seenPending []any
	var seenArgs []any
	n = graph.NewFuncNode(func(args []any) ([]any, error) {
		seenArgs = args
		seenPending = n.Read() // the buffer was already taken by Read
		return args, nil
	})
	n.Write([]any{1, 2})
	if err := n.Process(); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "args", seenArgs, []any{1, 2})
	if seenPending != nil {
		t.Fatalf("buffer during fn: got %#v, want nil", seenPending)
	}
	assertEqual(t, "output", n.Read(), []any{1, 2})
}
