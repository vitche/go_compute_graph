// Package graph implements the structure and activation primitives of the
// Compute Graph: nodes, synapses, graphs, spreading, forking, and activation.
//
// It is a Go port of the compute.graph.structure and compute.graph.activation
// modules of https://github.com/vitche/python-compute-graph. The concepts and
// execution semantics are preserved; see the project README for the list of
// Go-specific adaptations.
package graph

import "errors"

// ErrEmptyGraph is returned by operations that need a graph with at least one node.
var ErrEmptyGraph = errors.New("graph: graph has no nodes")

// ErrNilNode is returned by Activate when it is given a nil node.
var ErrNilNode = errors.New("graph: nil node")

// Node is a processing unit of the compute graph. It holds a list of pending
// arguments (its input buffer) and an ordered list of outgoing synapses.
//
// Custom nodes embed BaseNode and override Process.
type Node interface {
	// Add appends an outgoing synapse.
	Add(synapse *Synapse)
	// Arguments replaces the pending arguments.
	Arguments(value []any)
	// Read returns the pending arguments and clears them.
	Read() []any
	// Write appends data to the pending arguments; nil is ignored.
	Write(data []any)
	// Initialize is a lifecycle hook. It is a no-op by default and is never
	// called by the framework (kept for compatibility with the Python API).
	Initialize()
	// Process holds the node logic. It typically consumes its input with Read
	// and stores its output with Arguments. The default is a no-op, which
	// passes the input through unchanged.
	Process() error
	// Synapses returns the outgoing synapses in insertion order.
	Synapses() []*Synapse
}

// BaseNode is the default Node implementation. Embed it to create custom
// node types. The zero value is ready to use.
type BaseNode struct {
	arguments []any
	synapses  []*Synapse
}

// NewNode creates a plain pass-through node.
func NewNode() *BaseNode {
	return &BaseNode{}
}

// Add appends an outgoing synapse.
func (n *BaseNode) Add(synapse *Synapse) {
	n.synapses = append(n.synapses, synapse)
}

// Arguments replaces the pending arguments.
func (n *BaseNode) Arguments(value []any) {
	n.arguments = value
}

// Read returns the pending arguments and clears them.
func (n *BaseNode) Read() []any {
	result := n.arguments
	n.arguments = nil
	return result
}

// Write appends data to the pending arguments. A nil data slice is ignored.
//
// The full slice expression forces append to allocate instead of writing
// into the spare capacity of a slice that was passed to Arguments, so the
// caller's backing array is never modified.
func (n *BaseNode) Write(data []any) {
	if data == nil {
		return
	}
	n.arguments = append(n.arguments[:len(n.arguments):len(n.arguments)], data...)
}

// Initialize is a no-op lifecycle hook.
func (n *BaseNode) Initialize() {}

// Process is a no-op: the node's input becomes its output.
func (n *BaseNode) Process() error { return nil }

// Synapses returns the outgoing synapses in insertion order.
func (n *BaseNode) Synapses() []*Synapse {
	return n.synapses
}
