// Package profile adds profiling to the Compute Graph: a ProfileNode sink that
// accumulates Delta records, and a graph constructor that attaches such a sink
// to every added node.
//
// It is a Go port of the compute.graph.profile module of
// https://github.com/vitche/python-compute-graph (ProfileGraph and
// ProfileNode). The Python {"amount": n, "time": s} dictionaries are
// represented by the typed Delta struct.
package profile

import (
	"time"

	"github.com/vitche/go_compute_graph/graph"
)

// Delta is a profiling record. Nodes report profiling data by including Delta
// (or *Delta) values in their output. Equivalent to Python's
// {"amount": ..., "time": ...} dictionary.
type Delta struct {
	Amount int           // change in item count
	Time   time.Duration // elapsed time
}

// ProfileNode accumulates the Delta records written to it. Its pending
// arguments are always a single snapshot of the accumulated totals:
// []any{Delta{...}}. The zero value is ready to use.
type ProfileNode struct {
	graph.BaseNode
	Delta Delta // accumulated totals
}

// NewProfileNode creates a profile node with zero totals.
func NewProfileNode() *ProfileNode {
	return &ProfileNode{}
}

// Write adds every Delta and non-nil *Delta item of data to the accumulated
// totals and ignores all other items. It then replaces the pending arguments
// with []any{p.Delta}, even when data is nil.
func (p *ProfileNode) Write(data []any) error {
	for _, item := range data {
		switch delta := item.(type) {
		case Delta:
			p.add(delta)
		case *Delta:
			if delta != nil {
				p.add(*delta)
			}
		}
	}
	p.Arguments([]any{p.Delta})
	return nil
}

func (p *ProfileNode) add(delta Delta) {
	p.Delta.Amount += delta.Amount
	p.Delta.Time += delta.Time
}

// NewProfileGraph creates a graph that links a new ProfileNode from every node
// passed to Add, which is what Python's ProfileGraph.add does. Nodes added with
// Attach are not profiled. The caller's options are applied first, then the
// profile hook. Because the hook is one of the graph's options, sub-graphs
// created by Fork are profiled as well.
func NewProfileGraph(options ...graph.Option) *graph.Graph {
	all := make([]graph.Option, 0, len(options)+1)
	all = append(all, options...)
	all = append(all, graph.WithAddHook(addProfileNode))
	return graph.New(all...)
}

func addProfileNode(_ *graph.Graph, node graph.Node) {
	node.Add(graph.NewSynapse().Add(NewProfileNode()))
}
