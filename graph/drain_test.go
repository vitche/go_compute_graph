package graph_test

import (
	"errors"
	"testing"

	"github.com/vitche/go_compute_graph/graph"
)

// A Node is the aggregate of its parts.
var (
	_ graph.Source    = graph.NewNode()
	_ graph.Processor = graph.NewNode()
	_ graph.Drain     = graph.NewNode()
)

// bareDrain is a target which is not a Node, such as one leading to another
// process. It records writes and fails with err.
type bareDrain struct {
	writes [][]any
	err    error
}

func (d *bareDrain) Write(data []any) error {
	if data == nil {
		return nil // the contract of Drain
	}
	if d.err != nil {
		return d.err
	}
	d.writes = append(d.writes, data)
	return nil
}

// countingDrain counts every call of Write, including those with nil.
type countingDrain struct {
	nilWrites int
}

func (d *countingDrain) Write(data []any) error {
	if data == nil {
		d.nilWrites++
	}
	return nil
}

// activatorDrain is a bareDrain which activates in its own way.
type activatorDrain struct {
	bareDrain
	activations int
	activateErr error
}

func (d *activatorDrain) Activate() error {
	d.activations++
	return d.activateErr
}

// activatorNode is a whole Node which activates in its own way.
type activatorNode struct {
	writeSpy
	activations int
}

func (n *activatorNode) Activate() error {
	n.activations++
	return nil
}

func TestSynapseToDrainHasNoNode(t *testing.T) {
	drain := &bareDrain{}
	s := graph.NewSynapse().Add(drain)
	if s.Node() != nil {
		t.Fatalf("Node: got %v, want nil", s.Node())
	}
	if s.Drain() != graph.Drain(drain) {
		t.Fatal("Drain did not return the target")
	}
}

func TestSynapseToNodeHasBoth(t *testing.T) {
	node := graph.NewNode()
	s := graph.NewSynapse().Add(node)
	if s.Node() != graph.Node(node) {
		t.Fatal("Node did not return the target")
	}
	if s.Drain() != graph.Drain(node) {
		t.Fatal("Drain did not return the target")
	}
}

func TestSynapseWithoutTargetHasNoDrain(t *testing.T) {
	if got := graph.NewSynapse().Drain(); got != nil {
		t.Fatalf("Drain: got %v, want nil", got)
	}
}

func TestSynapseWriteReturnsTheDrainError(t *testing.T) {
	failure := errors.New("delivery failed")
	err := graph.NewSynapse().Add(&bareDrain{err: failure}).Write([]any{1})
	if !errors.Is(err, failure) {
		t.Fatalf("got %v, want %v", err, failure)
	}
}

func TestActivateNilTarget(t *testing.T) {
	if err := graph.Activate(nil); !errors.Is(err, graph.ErrNilNode) {
		t.Fatalf("nil: got %v", err)
	}
	var node graph.Node
	if err := graph.Activate(node); !errors.Is(err, graph.ErrNilNode) {
		t.Fatalf("nil Node: got %v", err)
	}
}

func TestActivateBareDrainHasNothingToRun(t *testing.T) {
	if err := graph.Activate(&bareDrain{}); err != nil {
		t.Fatal(err)
	}
}

func TestActivateUsesActivator(t *testing.T) {
	failure := errors.New("activation failed")
	drain := &activatorDrain{activateErr: failure}
	if err := graph.Activate(drain); !errors.Is(err, failure) {
		t.Fatalf("got %v, want %v", err, failure)
	}
	if drain.activations != 1 {
		t.Fatalf("activations: got %d, want 1", drain.activations)
	}
}

func TestActivatorReplacesTheDefaultStepsOfANode(t *testing.T) {
	node := &activatorNode{}
	if err := graph.Activate(node); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "activations", node.activations, 1)
	assertEqual(t, "processed", node.processed, 0)
}

func TestActivateStopsAtTheFirstFailedSynapse(t *testing.T) {
	failure := errors.New("delivery failed")
	first, last := &writeSpy{}, &writeSpy{}
	node := graph.NewNode()
	node.Add(graph.NewSynapse().Add(first))
	node.Add(graph.NewSynapse().Add(&bareDrain{err: failure}))
	node.Add(graph.NewSynapse().Add(last))
	node.Write([]any{1})

	if err := graph.Activate(node); !errors.Is(err, failure) {
		t.Fatalf("got %v, want %v", err, failure)
	}
	assertEqual(t, "writes before the failure", len(first.writes), 1)
	assertEqual(t, "writes after the failure", len(last.writes), 0)
}

func TestChainThroughDrain(t *testing.T) {
	drain := &bareDrain{}
	g := graph.New()
	g.Add(graph.NewNode())
	g.Previous().Add(graph.NewSynapse().Add(drain))
	if err := g.Write([]any{1, 2}); err != nil {
		t.Fatal(err)
	}
	if err := graph.Iterate(g.Nodes()); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "writes", drain.writes, [][]any{{1, 2}})
}

func TestIterateReturnsADrainError(t *testing.T) {
	failure := errors.New("delivery failed")
	g := graph.New()
	g.Add(graph.NewNode())
	g.Previous().Add(graph.NewSynapse().Add(&bareDrain{err: failure}))
	g.Write([]any{1})
	if err := graph.Iterate(g.Nodes()); !errors.Is(err, failure) {
		t.Fatalf("got %v, want %v", err, failure)
	}
}

func TestForkThroughDrain(t *testing.T) {
	failure := errors.New("delivery failed")
	g := graph.New()
	g.Add(graph.NewNode())
	branch, err := g.Fork(graph.NewNode())
	if err != nil {
		t.Fatal(err)
	}
	branch.Previous().Add(graph.NewSynapse().Add(&bareDrain{err: failure}))
	g.Write([]any{1})
	if err := graph.Iterate(g.Nodes()); !errors.Is(err, failure) {
		t.Fatalf("got %v, want %v", err, failure)
	}
}

func TestSpreadThroughDrain(t *testing.T) {
	drain := &bareDrain{}
	spread := graph.NewSpreadNode()
	spread.Add(graph.NewSynapse().Add(drain))
	spread.Write([]any{"a", "b"})
	if err := graph.Activate(spread); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "writes", drain.writes, [][]any{{"a"}, {"b"}})
}

// After spreading, Activate forwards the empty output of the spread node, so a
// Drain has to ignore nil as its contract says, or it would send an extra message.
func TestSpreadForwardsNilToItsDrain(t *testing.T) {
	drain := &countingDrain{}
	spread := graph.NewSpreadNode()
	spread.Add(graph.NewSynapse().Add(drain))
	spread.Write([]any{"a"})
	if err := graph.Activate(spread); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "nil writes", drain.nilWrites, 1)
}

func TestSpreadActivatesAnActivatorPerItem(t *testing.T) {
	drain := &activatorDrain{}
	spread := graph.NewSpreadNode()
	spread.Add(graph.NewSynapse().Add(drain))
	spread.Write([]any{"a", "b"})
	if err := graph.Activate(spread); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "activations", drain.activations, 2)
}

func TestSpreadStopsAtAWriteError(t *testing.T) {
	failure := errors.New("delivery failed")
	drain := &activatorDrain{bareDrain: bareDrain{err: failure}}
	spread := graph.NewSpreadNode()
	spread.Add(graph.NewSynapse().Add(drain))
	spread.Write([]any{"a", "b"})
	if err := graph.Activate(spread); !errors.Is(err, failure) {
		t.Fatalf("got %v, want %v", err, failure)
	}
	assertEqual(t, "activations", drain.activations, 0)
}

func TestGraphWriteReturnsTheFirstNodeError(t *testing.T) {
	failure := errors.New("refused")
	g := graph.New()
	g.Add(&failingWriteNode{err: failure})
	if err := g.Write([]any{1}); !errors.Is(err, failure) {
		t.Fatalf("got %v, want %v", err, failure)
	}
}

// failingWriteNode is a whole Node whose Write fails.
type failingWriteNode struct {
	graph.BaseNode
	err error
}

func (n *failingWriteNode) Write([]any) error { return n.err }
