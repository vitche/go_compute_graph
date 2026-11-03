package graph

// Option configures a Graph. Sub-graphs created by Fork receive the same
// options as their parent, so a fork of a configured graph is configured the
// same way (the equivalent of Python's self.__class__()).
type Option func(*Graph)

// WithAddHook registers a function that is called after every Add.
func WithAddHook(hook func(g *Graph, node Node)) Option {
	return func(g *Graph) {
		g.addHooks = append(g.addHooks, hook)
	}
}

// Graph is an insertion-ordered list of nodes plus a linear chain: every node
// passed to Add is linked from the previously added node.
type Graph struct {
	nodes    []Node
	first    Node
	previous Node
	options  []Option
	addHooks []func(g *Graph, node Node)
}

// New creates an empty graph.
func New(options ...Option) *Graph {
	g := &Graph{options: options}
	for _, option := range options {
		option(g)
	}
	return g
}

// Add appends node to the graph, links it from the previously added node,
// makes it the new Previous node, runs the add hooks, and returns it.
// It panics if node is nil.
func (g *Graph) Add(node Node) Node {
	if node == nil {
		panic("graph: Add of a nil node")
	}
	if g.first == nil {
		g.first = node
	}
	if g.previous != nil {
		g.previous.Add(NewSynapse().Add(node))
	}
	g.previous = node
	g.nodes = append(g.nodes, node)
	for _, hook := range g.addHooks {
		hook(g, node)
	}
	return node
}

// Attach appends node to the graph without linking it and returns it.
// It panics if node is nil.
func (g *Graph) Attach(node Node) Node {
	if node == nil {
		panic("graph: Attach of a nil node")
	}
	g.nodes = append(g.nodes, node)
	return node
}

// Connect adds the first node of subgraph to this graph. Only that node is
// linked and appended; subsequent Add calls chain from it.
func (g *Graph) Connect(subgraph *Graph) error {
	if subgraph == nil || subgraph.first == nil {
		return ErrEmptyGraph
	}
	g.Add(subgraph.first)
	return nil
}

// First returns the first added node, or nil.
func (g *Graph) First() Node {
	return g.first
}

// Previous returns the last added node, or nil.
func (g *Graph) Previous() Node {
	return g.previous
}

// Fork creates a sub-graph (with the same options as g) starting at node,
// wraps it in a SubgraphActivationNode linked from Previous, attaches that
// activation node to g, and returns the sub-graph. Previous is unchanged, so
// the main chain continues from the node the fork branched off.
func (g *Graph) Fork(node Node) (*Graph, error) {
	if g.previous == nil {
		return nil, ErrEmptyGraph
	}
	subgraph := New(g.options...)
	subgraph.Add(node)
	activationNode := NewSubgraphActivationNode(subgraph)
	g.previous.Add(NewSynapse().Add(activationNode))
	g.Attach(activationNode)
	return subgraph, nil
}

// Write appends data to the arguments of the first node.
func (g *Graph) Write(data []any) error {
	if g.first == nil {
		return ErrEmptyGraph
	}
	g.first.Write(data)
	return nil
}

// Nodes returns a copy of the node list in insertion order.
func (g *Graph) Nodes() []Node {
	result := make([]Node, len(g.nodes))
	copy(result, g.nodes)
	return result
}
