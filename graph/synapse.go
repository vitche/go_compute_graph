package graph

// Synapse is a directed edge pointing to exactly one target node. Writing to
// a synapse appends data to the target's arguments; it never activates the
// target.
type Synapse struct {
	node Node
}

// NewSynapse creates a synapse without a target.
func NewSynapse() *Synapse {
	return &Synapse{}
}

// Add sets the target node, overwriting any previous one, and returns the
// synapse for chaining: a.Add(graph.NewSynapse().Add(b)).
func (s *Synapse) Add(node Node) *Synapse {
	s.node = node
	return s
}

// Node returns the target node.
func (s *Synapse) Node() Node {
	return s.node
}

// Write appends data to the target's arguments.
// It panics if no target has been set, which is a programming error.
func (s *Synapse) Write(data []any) {
	if s.node == nil {
		panic("graph: write to a synapse without a target node")
	}
	s.node.Write(data)
}
