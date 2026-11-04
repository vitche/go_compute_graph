# Golang Compute Graph

A Go implementation of the [Python Compute Graph](https://github.com/vitche/python-compute-graph) (`compute_graph`):
a minimalist dataflow framework in which **nodes** exchange lists of values through **synapses**, are assembled into
**graphs**, can be **forked** into sub-graphs, **spread** over individual items, **profiled**, and **visualized** with
Graphviz.

The concepts, names, and execution semantics of the Python library are preserved one-to-one. Where Python relies on
features Go does not have (class inheritance with virtual methods, `self.__class__()`, exceptions, duck typing), the
smallest idiomatic Go adaptation is used. Every such adaptation is listed in
[Differences from the Python implementation](#differences-from-the-python-implementation).

## Installation

```bash
go get github.com/vitche/go_compute_graph
```

Requires Go 1.21+. The core packages have no third-party dependencies. Rendering images with the `visualization`
package requires the Graphviz `dot` executable on `PATH`.

## Package layout

| Python module                   | Go package                                           | Contents                                                                       |
|---------------------------------|------------------------------------------------------|--------------------------------------------------------------------------------|
| `compute.graph.structure`       | `github.com/vitche/go_compute_graph/graph`           | `Graph`, `Node`, `Source`, `Processor`, `Drain`, `BaseNode`, `Synapse`, `SpreadNode`, `SubgraphActivationNode` |
| `compute.graph.activation`      | `github.com/vitche/go_compute_graph/graph`           | `Activation`, `Activate`                                                       |
| `compute.graph.profile`         | `github.com/vitche/go_compute_graph/profile`         | `NewProfileGraph`, `ProfileNode`, `Delta`                                      |
| `compute.graph.visualization`   | `github.com/vitche/go_compute_graph/visualization`   | `GraphvizNode`                                                                 |

`structure` and `activation` share one Go package, because they reference each other: `SubgraphActivationNode` uses
`Activation`. Go does not allow import cycles.

## Core concepts

| Concept        | Meaning                                                                                                                      |
|----------------|------------------------------------------------------------------------------------------------------------------------------|
| **Node**       | A processing unit. It holds a list of pending **arguments** (its input buffer) and an ordered list of outgoing **synapses**. |
| **Source, Processor, Drain** | The parts of a node: `Read` gives data out, `Process` does the work, `Write` takes data in.                    |
| **Synapse**    | A directed edge pointing to exactly one target. It only uses the target's `Drain` part, which may lead elsewhere than a local node. |
| **Graph**      | An insertion-ordered list of nodes plus a linear chain: each `Add`ed node is linked from the previously added one.           |
| **Activation** | Runs a node: `Process`, then `Read` (take and clear) the arguments, then write them to every synapse in order.               |
| **Fork**       | Starts a sub-graph branching from the current node. The sub-graph runs through a `SubgraphActivationNode`.                   |
| **Spread**     | A `SpreadNode` sends each input item separately to each target and activates the target once per item.                       |
| **Profile**    | A `ProfileNode` sink attached to every node accumulates `Delta{Amount, Time}` records found in that node's output.           |

Data is always a list: `[]any` in Go, `list` in Python.

## Quick start

### A linear chain

```go
package main

import (
	"fmt"
	"log"

	"github.com/vitche/go_compute_graph/graph"
)

func main() {
	g := graph.New()

	// Double every value.
	g.Add(graph.NewFuncNode(func(args []any) ([]any, error) {
		out := make([]any, len(args))
		for i, v := range args {
			out[i] = v.(int) * 2
		}
		return out, nil
	}))

	// Sum all values.
	g.Add(graph.NewFuncNode(func(args []any) ([]any, error) {
		sum := 0
		for _, v := range args {
			sum += v.(int)
		}
		return []any{sum}, nil
	}))

	// Print the result.
	g.Add(graph.NewFuncNode(func(args []any) ([]any, error) {
		fmt.Println(args) // [12]
		return nil, nil
	}))

	if err := g.Write([]any{1, 2, 3}); err != nil {
		log.Fatal(err)
	}

	// Equivalent to Python's Activation().iterate(graph.nodes).
	if err := graph.Iterate(g.Nodes()); err != nil {
		log.Fatal(err)
	}
}
```

### A custom node type

This is the Go equivalent of subclassing `Node` and overriding `process()`. Embed `graph.BaseNode` and implement
`Process`:

```go
type Upper struct {
	graph.BaseNode
}

func (u *Upper) Process() error {
	in := u.Read()
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = strings.ToUpper(v.(string))
	}
	u.Arguments(out) // becomes this node's output
	return nil
}

g := graph.New()
g.Add(&Upper{}) // the zero value is ready to use
```

### Spreading over items

```go
g := graph.New()
g.Add(graph.NewSpreadNode())
g.Add(graph.NewFuncNode(func(args []any) ([]any, error) {
	fmt.Println(args) // called three times: [a], [b], [c]
	return args, nil
}))

g.Write([]any{"a", "b", "c"})
graph.Activate(g.First())
```

### Forking a sub-graph

```go
g := graph.New()
g.Add(source)
g.Add(transform)

branch, err := g.Fork(sinkA) // branch starts at sinkA, fed by transform
if err != nil {
	log.Fatal(err)
}
branch.Add(sinkB) // sinkA -> sinkB inside the branch

g.Add(next) // the main chain continues from transform, not from the branch

g.Write(input)
graph.Iterate(g.Nodes()) // source, transform, <branch activation>, next
```

### Profiling and visualization

This is a port of `examples/visualization.py`:

```go
package main

import (
	"log"

	"github.com/vitche/go_compute_graph/graph"
	"github.com/vitche/go_compute_graph/profile"
	"github.com/vitche/go_compute_graph/visualization"
)

func main() {
	g := profile.NewProfileGraph()
	g.Add(visualization.NewGraphvizNode())
	for i := 0; i < 7; i++ {
		g.Add(graph.NewNode())
	}

	// Renders the graph (DOT source + PNG) when the Graphviz node is activated.
	if err := graph.Activate(g.First()); err != nil {
		log.Fatal(err)
	}
}
```

Nodes report profiling data by including `profile.Delta` values in their output:

```go
func (w *Worker) Process() error {
	start := time.Now()
	in := w.Read()
	out := w.compute(in)
	w.Arguments(append(out, profile.Delta{
		Amount: len(out) - len(in),
		Time:   time.Since(start),
	}))
	return nil
}
```

## Execution model

The semantics match the Python implementation exactly. They are worth knowing before you build a graph:

1. **Writing never activates.** `Synapse.Write` and `Node.Write` only pass data to the target. A node runs only when
   something activates it.
2. **Exactly three things activate nodes:**
    - an explicit `graph.Activate(node)` call;
    - `Activation{}.Iterate(nodes)` / `graph.Iterate(nodes)`, which activates nodes in slice order. Because
      `Graph.Nodes()` is in insertion order, each node's output reaches the next node before that node runs;
    - a `SpreadNode` (activates its targets once per item) or a `SubgraphActivationNode` (iterates its sub-graph).
3. **Activation steps** for `graph.Activate(n)`:
    1. `n.Process()`. The node reads its arguments with `Read()` and stores its output with `Arguments(...)`.
       `BaseNode.Process` is a no-op, so a plain node passes its input through unchanged.
    2. `data := n.Read()` takes the output and clears the buffer.
    3. `s.Write(data)` runs for every synapse, in the order the synapses were added. The same slice goes to every
       target. The first error stops the activation, and the synapses after it get no data.
4. **Ordering** is insertion order everywhere: graph nodes, synapses, and spread items. A `SpreadNode` loops over
   synapses first, then over items.
5. **No caching, no parallelism.** Every activation recomputes. A graph runs on the calling goroutine. Nodes may use
   goroutines inside `Process`, but the graph structures are not safe for concurrent use.
6. **Edge cases inherited from Python:**
    - When a `SpreadNode` is driven by `Iterate`, its targets also appear later in `Nodes()`. They are activated once
      more, with the empty list the spread node forwards after it finishes. Drive spread chains with
      `graph.Activate(g.First())`, or make targets tolerate empty input.
    - `profile.Delta` records travel along normal synapses too. Downstream nodes receive them mixed in with regular
      data and should skip values they do not recognize.
    - `Graph.Connect(sub)` links and appends only the sub-graph's first node, and subsequent `Add` calls chain from
      that node.

## API reference

### Package `graph`

#### `Node`

A node is the aggregate of three parts:

```go
type Source interface {
	Read() []any // return the pending arguments and clear them
}

type Processor interface {
	Process() error // node logic; no-op by default
}

type Drain interface {
	Write(data []any) error // append data to the pending arguments; nil is ignored
}

type Node interface {
	Source
	Processor
	Drain
	Add(synapse *Synapse)  // append an outgoing synapse
	Arguments(value []any) // replace the pending arguments
	Initialize()           // lifecycle hook; no-op by default, never called by the framework
	Synapses() []*Synapse  // outgoing synapses, in insertion order
}
```

A synapse needs only the `Drain` of its target, so a type which leads elsewhere, such as to another process, can be a
target without being a whole node.

#### `BaseNode`

```go
type BaseNode struct { /* arguments, synapses */ }

func NewNode() *BaseNode
```

The default `Node` implementation, equivalent to Python's `Node`. Embed it to create custom node types. The zero value
is ready to use.

#### `FuncNode` (Go extension)

```go
type FuncNode struct { BaseNode /* fn */ }

func NewFuncNode(fn func(args []any) ([]any, error)) *FuncNode
```

Adapts a function to a node. `Process` reads the arguments, calls `fn`, and stores the result as the node's output.

#### `Activate`, `Activation`, `Iterate`

```go
func Activate(target Drain) error            // Python: node.activate()

type Activation struct{}
func (Activation) Iterate(nodes []Node) error // Python: Activation().iterate(nodes)

func Iterate(nodes []Node) error             // shorthand for Activation{}.Iterate(nodes)

type Activator interface {                   // activates in its own way
	Activate() error
}
```

`Activate` is a function, not a method. See [Differences](#differences-from-the-python-implementation) for why.
`Activate(nil)` returns `ErrNilNode`. `Iterate` stops at the first error and returns it.

`Activate` uses the `Activate` method of an `Activator` instead of its default steps. A `Node` runs the default steps,
and any other `Drain` has nothing to run.

#### `Synapse`

```go
type Synapse struct { /* target */ }

func NewSynapse() *Synapse
func (s *Synapse) Add(drain Drain) *Synapse // set the target (overwrites); returns s for chaining
func (s *Synapse) Node() Node               // the target, or nil if it is not a whole Node
func (s *Synapse) Drain() Drain             // the target
func (s *Synapse) Write(data []any) error   // target.Write(data); panics if no target is set
```

Typical use, same as Python: `a.Add(graph.NewSynapse().Add(b))`.

#### `SpreadNode`

```go
type SpreadNode struct { BaseNode }

func NewSpreadNode() *SpreadNode
```

`Process` reads its input and, for each synapse and each item, writes `[]any{item}` to the target and activates the
target immediately. Errors from writing and from target activation are returned.

#### `SubgraphActivationNode`

```go
type SubgraphActivationNode struct {
	BaseNode
	Subgraph *Graph
}

func NewSubgraphActivationNode(subgraph *Graph) *SubgraphActivationNode
```

The constructor adds a synapse to `subgraph.First()`. `Process` writes its input into the sub-graph's first node and
then runs `Iterate(subgraph.Nodes())`. `Graph.Fork` creates these nodes. You rarely need to construct one yourself.

#### `Graph`

```go
type Graph struct { /* nodes, first, previous, options */ }

type Option func(*Graph)

func New(options ...Option) *Graph
func WithAddHook(hook func(g *Graph, node Node)) Option // called after every Add

func (g *Graph) Add(node Node) Node              // append + link from the previous node; becomes Previous()
func (g *Graph) Attach(node Node) Node           // append to Nodes() without linking
func (g *Graph) Connect(subgraph *Graph) error   // Add(subgraph.First()); ErrEmptyGraph if subgraph is empty
func (g *Graph) First() Node                     // first added node, or nil
func (g *Graph) Previous() Node                  // last added node, or nil
func (g *Graph) Fork(node Node) (*Graph, error)  // see below; ErrEmptyGraph if g is empty
func (g *Graph) Write(data []any) error          // First().Write(data); ErrEmptyGraph if g is empty
func (g *Graph) Nodes() []Node                   // copy of all nodes in insertion order (Python: graph.nodes)

var ErrEmptyGraph = errors.New("graph: graph has no nodes")
var ErrNilNode    = errors.New("graph: nil node")
```

`Add` and `Attach` panic when given a nil node, because that is a programming error.

`Fork(node)` works as in Python:

1. It creates a sub-graph **with the same options as `g`**. This replaces `self.__class__()`, so forks of a profile
   graph are profile graphs too.
2. It adds `node` to the sub-graph.
3. It wraps the sub-graph in a `SubgraphActivationNode`.
4. It links that node from `g.Previous()` and `Attach`es it to `g`.
5. It returns the sub-graph.

`g.Previous()` does not change, so the main chain continues from the node the fork branched off.

### Package `profile`

```go
type Delta struct {
	Amount int           // Python: delta["amount"] — change in item count
	Time   time.Duration // Python: delta["time"]   — elapsed time (seconds in Python)
}

type ProfileNode struct {
	graph.BaseNode
	Delta Delta // accumulated totals
}

func NewProfileNode() *ProfileNode
func (p *ProfileNode) Write(data []any) error // adds every Delta / *Delta item to p.Delta,
                                              // then sets Arguments([]any{p.Delta})

func NewProfileGraph(options ...graph.Option) *graph.Graph
```

`NewProfileGraph` returns a regular `*graph.Graph` with an add hook. After every `Add`, the hook attaches a new
`ProfileNode` sink to the added node, which is what Python's `ProfileGraph.add` does. Nodes added with `Attach` are not
profiled.

### Package `visualization`

```go
type GraphvizNode struct {
	graph.BaseNode
	Output string // output file base name; defaults to the node's name
	Format string // output format passed to `dot -T`; defaults to "png"
}

func NewGraphvizNode() *GraphvizNode
func (v *GraphvizNode) Process() error // traverse from v, write <Output>.gv, render <Output>.<Format>
func (v *GraphvizNode) DOT() string    // Go extension: the DOT source, without rendering
```

The traversal matches Python's:

- It is depth-first from the Graphviz node and visits each node once.
- `ProfileNode` neighbours are not drawn. Their pending delta is read (and therefore consumed) and becomes the label of
  the node's outgoing edges, in the format `"+3 (N), 0.25 (s)"`. The `+` appears only for positive amounts.
- A target which is not a whole node is a leaf, drawn if it implements `fmt.Stringer` and skipped otherwise.
- Node names come from `String()` if the node implements `fmt.Stringer`, and from `"%T@%p"` otherwise. This takes the
  place of Python's default object `repr`.

`Process` returns an error if the `dot` executable cannot be found or rendering fails.

## Differences from the Python implementation

| Python                                 | Go                                                          | Reason                                                                                                   |
|----------------------------------------|-------------------------------------------------------------|----------------------------------------------------------------------------------------------------------|
| `class X(Node)` overriding `process()` | `type X struct{ graph.BaseNode }` + `Process() error`       | Go uses embedding instead of inheritance                                                                 |
| `node.activate()`                      | `graph.Activate(node)`                                      | Embedded methods are not virtual: a `BaseNode.Activate` method could not call the outer type's `Process` |
| `Node.add(synapse)` returns `self`     | `Add(synapse)` returns nothing                              | With embedding, the method could only return the inner `*BaseNode`, not the outer node                   |
| `process()` returns `None`             | `Process() error`; `Activate`/`Iterate` return `error`      | Idiomatic error propagation instead of exceptions                                                        |
| `write()` returns `None`               | `Write(data) error`, on `Node` and `Synapse`                | A write may fail when the target is remote                                                               |
| a synapse's target is a `Node`         | a synapse's target is a `Drain`, a part of `Node`           | A target can lead elsewhere than a local node                                                            |
| `AttributeError` on an empty graph     | `ErrEmptyGraph`                                             | Same cause, reported as an error value                                                                   |
| `self.__class__()` in `fork`           | sub-graph inherits `g`'s `Option`s                          | Go has no dynamic class construction                                                                     |
| `ProfileGraph` subclass                | `profile.NewProfileGraph()` = `graph.New(WithAddHook(...))` | Composition instead of overriding `add`                                                                  |
| `{"amount": n, "time": s}` dict        | `profile.Delta{Amount, Time}`                               | Typed struct; `Time` is a `time.Duration`                                                                |
| `graph.nodes` attribute                | `g.Nodes()`                                                 | Encapsulated field                                                                                       |
| `graphviz` PyPI dependency             | DOT written natively, `dot` binary for rendering            | No third-party Go dependency                                                                             |
| shared class-level `__opened` set      | visited set per `Process` call                              | Avoids global state                                                                                      |
| —                                      | `FuncNode`, `graph.Iterate`, `GraphvizNode.DOT()`           | Go conveniences; they add no new concepts                                                                |

## Testing

```bash
go test -race ./...
```

## Related projects

Compute Graph is part of the same dataflow toolset as the PySyun Pipeline, which is available in both languages:

| Project                  | Python                                                                                                                                                 | Golang                                                             |
|--------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------|
| Compute Graph            | [python-compute-graph](https://github.com/vitche/python-compute-graph)                                                                                 | go_compute_graph (this project)                                    |
| PySyun Pipeline & Chain  | [pysyun-timeline](https://github.com/pysyun/pysyun-timeline) (graph pipelines), [pysyun_chain](https://github.com/pysyun/pysyun_chain) (linear chains) | [go_pysyun_pipeline](https://github.com/pysyun/go_pysyun_pipeline) |

How the two differ:

- **Compute Graph** passes lists through explicit synapses, and you choose when each node runs (`Activate`, `Iterate`,
  spread and fork nodes).
- **PySyun Pipeline** composes `Process(data) any` stages:
    - a `PipelineNode` activates its downstream nodes recursively;
    - `Chainable` and `ChainableGroup` provide linear and parallel chains.

Any `Process(data any) any` processor from PySyun Pipeline can be wrapped in a `graph.FuncNode`:

```go
node := graph.NewFuncNode(func(args []any) ([]any, error) {
	return []any{processor.Process(args)}, nil
})
```

## License

LGPL-2.1, the same license as [go_pysyun_pipeline](https://github.com/pysyun/go_pysyun_pipeline) and
[pysyun_chain](https://github.com/pysyun/pysyun_chain).

## Credits

Based on [python-compute-graph](https://github.com/vitche/python-compute-graph) by the Vitche Research Team.
Companion of the [PySyun](https://github.com/pysyun) pipeline ecosystem.
