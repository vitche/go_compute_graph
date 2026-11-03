package graph_test

import (
	"errors"
	"testing"

	"github.com/vitche/go_compute_graph/graph"
)

// targets returns the target nodes of n's synapses in order.
func targets(n graph.Node) []graph.Node {
	var out []graph.Node
	for _, s := range n.Synapses() {
		out = append(out, s.Node())
	}
	return out
}

func assertNodes(t *testing.T, what string, got []graph.Node, want ...graph.Node) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: got %d nodes, want %d", what, len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s: node %d differs: got %v, want %v", what, i, got[i], want[i])
		}
	}
}

func TestNewIsEmpty(t *testing.T) {
	g := graph.New()
	if g.First() != nil || g.Previous() != nil {
		t.Fatal("new graph should have no First/Previous")
	}
	if n := g.Nodes(); n == nil || len(n) != 0 {
		t.Fatalf("Nodes: got %#v, want empty non-nil slice", n)
	}
}

func TestAddLinksChain(t *testing.T) {
	g := graph.New()
	a, b, c := graph.NewNode(), graph.NewNode(), graph.NewNode()

	if got := g.Add(a); got != a {
		t.Fatal("Add should return its argument")
	}
	if g.First() != a || g.Previous() != a {
		t.Fatal("after first Add, First and Previous should be a")
	}
	if len(a.Synapses()) != 0 {
		t.Fatal("first node must not get a synapse")
	}

	g.Add(b)
	g.Add(c)
	if g.First() != a {
		t.Fatal("First must stay the first added node")
	}
	if g.Previous() != c {
		t.Fatal("Previous must be the last added node")
	}
	assertNodes(t, "a targets", targets(a), b)
	assertNodes(t, "b targets", targets(b), c)
	assertNodes(t, "c targets", targets(c))
	assertNodes(t, "Nodes", g.Nodes(), a, b, c)
}

func TestAddAndAttachNilPanic(t *testing.T) {
	g := graph.New()
	mustPanic(t, "Add(nil)", func() { g.Add(nil) })
	mustPanic(t, "Attach(nil)", func() { g.Attach(nil) })
	if len(g.Nodes()) != 0 {
		t.Fatal("nil nodes must not be appended")
	}
}

func TestAttachDoesNotLink(t *testing.T) {
	g := graph.New()
	a, x, b := graph.NewNode(), graph.NewNode(), graph.NewNode()
	g.Add(a)
	if got := g.Attach(x); got != x {
		t.Fatal("Attach should return its argument")
	}
	if g.Previous() != a {
		t.Fatal("Attach must not change Previous")
	}
	if len(a.Synapses()) != 0 {
		t.Fatal("Attach must not link the node")
	}
	g.Add(b)
	assertNodes(t, "a targets", targets(a), b)
	assertNodes(t, "x targets", targets(x))
	assertNodes(t, "Nodes", g.Nodes(), a, x, b)
}

func TestAttachOnEmptyGraph(t *testing.T) {
	g := graph.New()
	x := graph.NewNode()
	g.Attach(x)
	if g.First() != nil || g.Previous() != nil {
		t.Fatal("Attach must not set First or Previous")
	}
	if err := g.Write([]any{1}); !errors.Is(err, graph.ErrEmptyGraph) {
		t.Fatalf("Write: got %v, want ErrEmptyGraph", err)
	}
	assertNodes(t, "Nodes", g.Nodes(), x)
}

func TestNodesReturnsCopy(t *testing.T) {
	g := graph.New()
	a, b := graph.NewNode(), graph.NewNode()
	g.Add(a)
	g.Add(b)

	nodes := g.Nodes()
	nodes[0] = graph.NewNode()
	nodes = append(nodes, graph.NewNode())
	_ = nodes
	assertNodes(t, "Nodes after mutation", g.Nodes(), a, b)
}

func TestWrite(t *testing.T) {
	g := graph.New()
	if err := g.Write([]any{1}); !errors.Is(err, graph.ErrEmptyGraph) {
		t.Fatalf("empty graph: got %v, want ErrEmptyGraph", err)
	}
	a, b := graph.NewNode(), graph.NewNode()
	g.Add(a)
	g.Add(b)
	if err := g.Write([]any{1}); err != nil {
		t.Fatal(err)
	}
	if err := g.Write([]any{2}); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, "first node", a.Read(), []any{1, 2})
	if b.Read() != nil {
		t.Fatal("Write must only reach the first node")
	}
}

func TestConnectEmpty(t *testing.T) {
	tests := []struct {
		name string
		sub  *graph.Graph
	}{
		{"nil", nil},
		{"empty", graph.New()},
		{"attach only", func() *graph.Graph { s := graph.New(); s.Attach(graph.NewNode()); return s }()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := graph.New()
			a := graph.NewNode()
			g.Add(a)
			if err := g.Connect(tt.sub); !errors.Is(err, graph.ErrEmptyGraph) {
				t.Fatalf("got %v, want ErrEmptyGraph", err)
			}
			assertNodes(t, "Nodes", g.Nodes(), a)
			if g.Previous() != a || len(a.Synapses()) != 0 {
				t.Fatal("failed Connect must not change the graph")
			}
		})
	}
}

func TestConnectLinksOnlySubgraphFirst(t *testing.T) {
	g := graph.New()
	a := graph.NewNode()
	g.Add(a)

	sub := graph.New()
	s1, s2 := graph.NewNode(), graph.NewNode()
	sub.Add(s1)
	sub.Add(s2)

	if err := g.Connect(sub); err != nil {
		t.Fatal(err)
	}
	assertNodes(t, "Nodes", g.Nodes(), a, s1)
	assertNodes(t, "a targets", targets(a), s1)
	if g.Previous() != s1 {
		t.Fatal("sub.First() should become Previous")
	}

	// Subsequent Adds chain from sub.First(), next to its existing synapse.
	b := graph.NewNode()
	g.Add(b)
	assertNodes(t, "s1 targets", targets(s1), s2, b)
	assertNodes(t, "Nodes", g.Nodes(), a, s1, b)
}

func TestConnectOnEmptyGraphSetsFirst(t *testing.T) {
	g := graph.New()
	sub := graph.New()
	s1 := graph.NewNode()
	sub.Add(s1)
	if err := g.Connect(sub); err != nil {
		t.Fatal(err)
	}
	if g.First() != s1 || g.Previous() != s1 {
		t.Fatal("Connect on an empty graph should make sub.First() First and Previous")
	}
}

type hookCall struct {
	hook  string
	graph *graph.Graph
	node  graph.Node
}

func TestWithAddHook(t *testing.T) {
	var calls []hookCall
	hook := func(name string) func(*graph.Graph, graph.Node) {
		return func(g *graph.Graph, n graph.Node) {
			// The node is already part of the graph when the hook runs.
			if g.Previous() != n {
				t.Errorf("hook %s: Previous is not the added node", name)
			}
			calls = append(calls, hookCall{name, g, n})
		}
	}
	g := graph.New(graph.WithAddHook(hook("h1")), graph.WithAddHook(hook("h2")))
	a, x, b := graph.NewNode(), graph.NewNode(), graph.NewNode()
	g.Add(a)
	g.Attach(x)
	g.Add(b)

	want := []hookCall{{"h1", g, a}, {"h2", g, a}, {"h1", g, b}, {"h2", g, b}}
	if len(calls) != len(want) {
		t.Fatalf("hook calls: got %d, want %d", len(calls), len(want))
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("hook call %d: got %+v, want %+v", i, calls[i], want[i])
		}
	}
}

func TestWithAddHookCalledOnConnect(t *testing.T) {
	var added []graph.Node
	g := graph.New(graph.WithAddHook(func(_ *graph.Graph, n graph.Node) { added = append(added, n) }))
	sub := graph.New()
	s1 := graph.NewNode()
	sub.Add(s1)
	if err := g.Connect(sub); err != nil {
		t.Fatal(err)
	}
	assertNodes(t, "hooked", added, s1)
}

func TestForkEmptyGraph(t *testing.T) {
	g := graph.New()
	sub, err := g.Fork(graph.NewNode())
	if !errors.Is(err, graph.ErrEmptyGraph) {
		t.Fatalf("got %v, want ErrEmptyGraph", err)
	}
	if sub != nil {
		t.Fatal("subgraph should be nil on error")
	}
	if len(g.Nodes()) != 0 {
		t.Fatal("failed Fork must not change the graph")
	}
}

func TestForkStructure(t *testing.T) {
	g := graph.New()
	a := graph.NewNode()
	g.Add(a)

	branch := graph.NewNode()
	sub, err := g.Fork(branch)
	if err != nil {
		t.Fatal(err)
	}
	if sub == nil || sub == g {
		t.Fatal("Fork should return a new subgraph")
	}
	if sub.First() != branch || sub.Previous() != branch {
		t.Fatal("forked node should be First and Previous of the subgraph")
	}
	assertNodes(t, "sub Nodes", sub.Nodes(), branch)

	if g.Previous() != a {
		t.Fatal("Fork must not change Previous")
	}
	nodes := g.Nodes()
	if len(nodes) != 2 || nodes[0] != a {
		t.Fatalf("parent Nodes: got %v", nodes)
	}
	act, ok := nodes[1].(*graph.SubgraphActivationNode)
	if !ok {
		t.Fatalf("attached node: got %T, want *graph.SubgraphActivationNode", nodes[1])
	}
	if act.Subgraph != sub {
		t.Fatal("activation node Subgraph should be the returned subgraph")
	}
	assertNodes(t, "a targets", targets(a), act)
	assertNodes(t, "activation node targets", targets(act), branch)

	// The main chain continues from a.
	b := graph.NewNode()
	g.Add(b)
	assertNodes(t, "a targets after Add", targets(a), act, b)
	assertNodes(t, "parent Nodes after Add", g.Nodes(), a, act, b)
}

func TestForkInheritsOptions(t *testing.T) {
	var calls []hookCall
	g := graph.New(graph.WithAddHook(func(g *graph.Graph, n graph.Node) {
		calls = append(calls, hookCall{"h", g, n})
	}))
	a := graph.NewNode()
	g.Add(a)

	branch, next := graph.NewNode(), graph.NewNode()
	sub, err := g.Fork(branch)
	if err != nil {
		t.Fatal(err)
	}
	sub.Add(next)

	want := []hookCall{{"h", g, a}, {"h", sub, branch}, {"h", sub, next}}
	if len(calls) != len(want) {
		t.Fatalf("hook calls: got %+v, want %+v", calls, want)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("hook call %d: got %+v, want %+v", i, calls[i], want[i])
		}
	}
}

func TestForkEndToEnd(t *testing.T) {
	var log []string
	g := graph.New()
	src := newRecorder("src", &log)
	g.Add(src)

	branch1 := newRecorder("branch1", &log)
	sub, err := g.Fork(branch1)
	if err != nil {
		t.Fatal(err)
	}
	branch2 := newRecorder("branch2", &log)
	sub.Add(branch2)

	main := newRecorder("main", &log)
	g.Add(main)

	if err := g.Write([]any{1, 2}); err != nil {
		t.Fatal(err)
	}
	if err := graph.Iterate(g.Nodes()); err != nil {
		t.Fatal(err)
	}

	assertEqual(t, "activation order", log, []string{"src", "branch1", "branch2", "main"})
	assertEqual(t, "branch1 input", branch1.inputs, [][]any{{1, 2}})
	assertEqual(t, "branch2 input", branch2.inputs, [][]any{{1, 2}})
	assertEqual(t, "main input", main.inputs, [][]any{{1, 2}})
}

func TestForkBranchErrorPropagates(t *testing.T) {
	boom := errors.New("boom")
	g := graph.New()
	g.Add(graph.NewNode())
	bad := newRecorder("bad", nil)
	bad.err = boom
	if _, err := g.Fork(bad); err != nil {
		t.Fatal(err)
	}
	main := newRecorder("main", nil)
	g.Add(main)

	if err := g.Write([]any{1}); err != nil {
		t.Fatal(err)
	}
	if err := graph.Iterate(g.Nodes()); !errors.Is(err, boom) {
		t.Fatalf("got %v, want %v", err, boom)
	}
	if main.calls() != 0 {
		t.Fatal("main chain must not run after the branch failed")
	}
}
