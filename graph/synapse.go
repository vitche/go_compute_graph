package graph

// Synapse is a directed edge pointing to exactly one target. The target is
// usually a whole Node, but only its Drain part is required, so a Drain which
// leads elsewhere, such as to another process, can stand in its place. Writing
// to a synapse passes the data to the target's Drain; it never activates the
// target.
type Synapse struct {
	drain Drain
}

// NewSynapse creates a synapse without a target.
func NewSynapse() *Synapse {
	return &Synapse{}
}

// Add sets the target, overwriting any previous one, and returns the synapse
// for chaining: a.Add(graph.NewSynapse().Add(b)).
func (s *Synapse) Add(drain Drain) *Synapse {
	s.drain = drain
	return s
}

// Node returns the target if it is a whole Node, and nil otherwise.
func (s *Synapse) Node() Node {
	node, _ := s.drain.(Node)
	return node
}

// Drain returns the target.
func (s *Synapse) Drain() Drain {
	return s.drain
}

// Write passes data to the target and returns its error.
// It panics if no target has been set, which is a programming error.
func (s *Synapse) Write(data []any) error {
	if s.drain == nil {
		panic("graph: write to a synapse without a target node")
	}
	return s.drain.Write(data)
}
