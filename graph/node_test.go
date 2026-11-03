package graph_test

import (
	"testing"

	"github.com/vitche/go_compute_graph/graph"
)

// Compile-time checks that all node types implement graph.Node.
var (
	_ graph.Node = (*graph.BaseNode)(nil)
	_ graph.Node = (*graph.SpreadNode)(nil)
	_ graph.Node = (*graph.SubgraphActivationNode)(nil)
	_ graph.Node = (*graph.FuncNode)(nil)
)

func TestBaseNodeZeroValueUsable(t *testing.T) {
	var n graph.BaseNode
	if got := n.Read(); got != nil {
		t.Fatalf("Read on zero value: got %#v, want nil", got)
	}
	if got := n.Synapses(); len(got) != 0 {
		t.Fatalf("Synapses on zero value: got %d, want 0", len(got))
	}
	n.Write([]any{1})
	assertEqual(t, "Read after Write", n.Read(), []any{1})
	if err := graph.Activate(&n); err != nil {
		t.Fatalf("Activate zero value: %v", err)
	}
}

func TestNewNode(t *testing.T) {
	n := graph.NewNode()
	if n == nil {
		t.Fatal("NewNode returned nil")
	}
	if n.Read() != nil || len(n.Synapses()) != 0 {
		t.Fatal("NewNode should return an empty node")
	}
}

func TestBaseNodeReadReturnsPendingAndClears(t *testing.T) {
	n := graph.NewNode()
	n.Arguments([]any{"a", "b"})
	assertEqual(t, "first Read", n.Read(), []any{"a", "b"})
	if got := n.Read(); got != nil {
		t.Fatalf("second Read: got %#v, want nil", got)
	}
}

func TestBaseNodeWrite(t *testing.T) {
	tests := []struct {
		name   string
		writes [][]any
		want   []any
	}{
		{"single", [][]any{{1, 2}}, []any{1, 2}},
		{"appends in order", [][]any{{1}, {2, 3}, {4}}, []any{1, 2, 3, 4}},
		{"nil ignored", [][]any{nil}, nil},
		{"nil between", [][]any{{1}, nil, {2}}, []any{1, 2}},
		{"empty non-nil", [][]any{{}}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := graph.NewNode()
			for _, w := range tt.writes {
				n.Write(w)
			}
			got := n.Read()
			if len(tt.want) == 0 {
				if len(got) != 0 {
					t.Fatalf("Read: got %#v, want empty", got)
				}
				return
			}
			assertEqual(t, "Read", got, tt.want)
		})
	}
}

func TestBaseNodeWriteDoesNotMutateArgumentsBackingArray(t *testing.T) {
	backing := make([]any, 1, 10)
	backing[0] = "a"
	n := graph.NewNode()
	n.Arguments(backing)
	n.Write([]any{"b"})

	if spare := backing[:2]; spare[1] != nil {
		t.Fatalf("Write wrote into the caller's spare capacity: %#v", spare)
	}
	assertEqual(t, "caller slice", backing, []any{"a"})
	assertEqual(t, "Read", n.Read(), []any{"a", "b"})
}

func TestBaseNodeWriteDoesNotAliasInput(t *testing.T) {
	in := []any{1, 2}
	n := graph.NewNode()
	n.Write(in)
	in[0] = 99
	assertEqual(t, "Read", n.Read(), []any{1, 2})
}

func TestBaseNodeArgumentsReplaces(t *testing.T) {
	n := graph.NewNode()
	n.Write([]any{1, 2})
	n.Arguments([]any{3})
	assertEqual(t, "after replace", n.Read(), []any{3})

	n.Write([]any{1})
	n.Arguments(nil)
	if got := n.Read(); got != nil {
		t.Fatalf("Arguments(nil): got %#v, want nil", got)
	}
}

func TestBaseNodeInitializeAndProcessAreNoOps(t *testing.T) {
	n := graph.NewNode()
	n.Write([]any{1})
	n.Initialize()
	if err := n.Process(); err != nil {
		t.Fatalf("Process: %v", err)
	}
	assertEqual(t, "arguments untouched", n.Read(), []any{1})
}

func TestBaseNodeAddKeepsSynapseOrder(t *testing.T) {
	n := graph.NewNode()
	s1, s2, s3 := graph.NewSynapse(), graph.NewSynapse(), graph.NewSynapse()
	n.Add(s1)
	n.Add(s2)
	n.Add(s3)
	got := n.Synapses()
	if len(got) != 3 || got[0] != s1 || got[1] != s2 || got[2] != s3 {
		t.Fatalf("Synapses: got %v, want [s1 s2 s3]", got)
	}
}
