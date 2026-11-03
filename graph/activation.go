package graph

// Activate runs a node: it calls Process, then reads (takes and clears) the
// node's arguments and writes them to every synapse in insertion order.
//
// This is the equivalent of Python's node.activate(). It is a function rather
// than a method because embedded methods in Go are not virtual: a method on
// BaseNode could not call the Process method of the type that embeds it.
func Activate(node Node) error {
	if node == nil {
		return ErrNilNode
	}
	if err := node.Process(); err != nil {
		return err
	}
	data := node.Read()
	for _, synapse := range node.Synapses() {
		synapse.Write(data)
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
