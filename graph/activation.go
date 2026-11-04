package graph

// Activate runs a target. An Activator activates in its own way. A Node runs
// the default steps: it calls Process, then reads (takes and clears) the node's
// arguments and writes them to every synapse in insertion order, stopping at
// the first error. Any other Drain has nothing to run.
//
// This is the equivalent of Python's node.activate(). It is a function rather
// than a method because embedded methods in Go are not virtual: a method on
// BaseNode could not call the Process method of the type that embeds it.
func Activate(target Drain) error {
	if target == nil {
		return ErrNilNode
	}
	if activator, ok := target.(Activator); ok {
		return activator.Activate()
	}
	node, ok := target.(Node)
	if !ok {
		return nil
	}
	if err := node.Process(); err != nil {
		return err
	}
	data := node.Read()
	for _, synapse := range node.Synapses() {
		if err := synapse.Write(data); err != nil {
			return err
		}
	}
	return nil
}

// Activation activates node lists. Equivalent to Python's Activation class.
type Activation struct{}

// Iterate activates the nodes in slice order and stops at the first error.
func (Activation) Iterate(nodes []Node) error {
	for _, node := range nodes {
		if err := Activate(node); err != nil {
			return err
		}
	}
	return nil
}

// Iterate is a shorthand for Activation{}.Iterate(nodes).
func Iterate(nodes []Node) error {
	return Activation{}.Iterate(nodes)
}
