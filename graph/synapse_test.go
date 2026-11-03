package graph_test

import (
	"testing"

	"github.com/vitche/go_compute_graph/graph"
)

func TestNewSynapseHasNoTarget(t *testing.T) {
	if got := graph.NewSynapse().Node(); got != nil {
		t.Fatalf("Node: got %v, want nil", got)
	}
}

func TestSynapseAddOverwritesAndReturnsItself(t *testing.T) {
	a, b := graph.NewNode(), graph.NewNode()
	s := graph.NewSynapse()
	if got := s.Add(a); got != s {
		t.Fatal("Add did not return the synapse itself")
	}
	if s.Node() != a {
		t.Fatal("Node did not return the first target")
	}
	s.Add(b)
	if s.Node() != b {
		t.Fatal("Add did not overwrite the target")
	}
}

func TestSynapseWriteForwardsToTargetWrite(t *testing.T) {
	spy := &writeSpy{}
	s := graph.NewSynapse().Add(spy)
	data := []any{1, 2}
	s.Write(data)
	s.Write(nil)

	if len(spy.writes) != 2 {
		t.Fatalf("target Write calls: got %d, want 2", len(spy.writes))
	}
	if &spy.writes[0][0] != &data[0] {
		t.Fatal("synapse did not pass the same slice to the target")
	}
	if spy.writes[1] != nil {
		t.Fatalf("second write: got %#v, want nil", spy.writes[1])
	}
	if spy.processed != 0 {
		t.Fatal("Synapse.Write must not activate the target")
	}
	assertEqual(t, "target arguments", spy.Read(), []any{1, 2})
}

func TestSynapseWriteWithoutTargetPanics(t *testing.T) {
	mustPanic(t, "Write without target", func() {
		graph.NewSynapse().Write([]any{1})
	})
}
