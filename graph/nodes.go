package graph

// SpreadNode sends every input item separately to every target and activates
// the target once per item. Synapses are the outer loop, items the inner one.
type SpreadNode struct {
	BaseNode
}

// NewSpreadNode creates a spread node.
func NewSpreadNode() *SpreadNode {
	return &SpreadNode{}
}

// Process spreads the input over the targets. The first write or activation
// error is returned and stops the spreading.
func (n *SpreadNode) Process() error {
	data := n.Read()
	for _, synapse := range n.Synapses() {
		for _, item := range data {
			if err := synapse.Write([]any{item}); err != nil {
				return err
			}
			if err := Activate(synapse.Drain()); err != nil {
				return err
			}
		}
	}
	return nil
}

// SubgraphActivationNode runs a sub-graph: it writes its input into the
// sub-graph's first node and then iterates over all sub-graph nodes.
// Graph.Fork creates these nodes.
type SubgraphActivationNode struct {
	BaseNode
	Subgraph *Graph
}

// NewSubgraphActivationNode creates an activation node for subgraph and adds
// a synapse to the sub-graph's first node (when there is one).
func NewSubgraphActivationNode(subgraph *Graph) *SubgraphActivationNode {
	n := &SubgraphActivationNode{Subgraph: subgraph}
	if subgraph != nil && subgraph.First() != nil {
		n.Add(NewSynapse().Add(subgraph.First()))
	}
	return n
}

// Process feeds the input into the sub-graph and activates its nodes in
// order. It returns ErrEmptyGraph if the sub-graph is nil or empty.
func (n *SubgraphActivationNode) Process() error {
	data := n.Read()
	if n.Subgraph == nil {
		return ErrEmptyGraph
	}
	if err := n.Subgraph.Write(data); err != nil {
		return err
	}
	return Iterate(n.Subgraph.Nodes())
}

// FuncNode adapts a function to a node: Process reads the arguments, calls
// the function, and stores the result as the node's output. A FuncNode with
// a nil function passes its input through. This is a Go extension.
type FuncNode struct {
	BaseNode
	fn func(args []any) ([]any, error)
}

// NewFuncNode creates a node from fn.
func NewFuncNode(fn func(args []any) ([]any, error)) *FuncNode {
	return &FuncNode{fn: fn}
}

// Process calls the function with the pending arguments.
func (n *FuncNode) Process() error {
	if n.fn == nil {
		return nil
	}
	output, err := n.fn(n.Read())
	if err != nil {
		return err
	}
	n.Arguments(output)
	return nil
}
