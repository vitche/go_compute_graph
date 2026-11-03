package graph_test

import (
	"fmt"

	"github.com/vitche/go_compute_graph/graph"
)

// A linear chain driven by Iterate, as in the README quick start.
func Example() {
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
		fmt.Println(args)
		return nil, nil
	}))

	if err := g.Write([]any{1, 2, 3}); err != nil {
		fmt.Println(err)
		return
	}
	if err := graph.Iterate(g.Nodes()); err != nil {
		fmt.Println(err)
	}
	// Output: [12]
}

// A SpreadNode activates its target once per input item. Driving the chain
// with Activate(g.First()) avoids the extra empty activation that Iterate
// would cause.
func ExampleSpreadNode() {
	g := graph.New()
	g.Add(graph.NewSpreadNode())
	g.Add(graph.NewFuncNode(func(args []any) ([]any, error) {
		fmt.Println("item:", args)
		return nil, nil
	}))

	_ = g.Write([]any{"a", "b", "c"})
	if err := graph.Activate(g.First()); err != nil {
		fmt.Println(err)
	}
	// Output:
	// item: [a]
	// item: [b]
	// item: [c]
}

// Fork branches a sub-graph off the current node; the main chain continues
// from the node the fork branched off.
func ExampleGraph_Fork() {
	printer := func(label string) graph.Node {
		return graph.NewFuncNode(func(args []any) ([]any, error) {
			fmt.Println(label, args)
			return nil, nil
		})
	}
	g := graph.New()
	g.Add(graph.NewNode())
	if _, err := g.Fork(printer("branch:")); err != nil {
		fmt.Println(err)
		return
	}
	g.Add(printer("main:"))

	_ = g.Write([]any{42})
	if err := graph.Iterate(g.Nodes()); err != nil {
		fmt.Println(err)
	}
	// Output:
	// branch: [42]
	// main: [42]
}
