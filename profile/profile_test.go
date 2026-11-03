package profile_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/vitche/go_compute_graph/graph"
	"github.com/vitche/go_compute_graph/profile"
)

var _ graph.Node = (*profile.ProfileNode)(nil)

func assertSnapshot(t *testing.T, p *profile.ProfileNode, want profile.Delta) {
	t.Helper()
	if p.Delta != want {
		t.Fatalf("Delta = %+v, want %+v", p.Delta, want)
	}
	got := p.Read()
	if !reflect.DeepEqual(got, []any{want}) {
		t.Fatalf("Read() = %#v, want %#v", got, []any{want})
	}
	if again := p.Read(); again != nil {
		t.Fatalf("second Read() = %#v, want nil", again)
	}
}

func TestNewProfileNode(t *testing.T) {
	p := profile.NewProfileNode()
	if p == nil {
		t.Fatal("NewProfileNode returned nil")
	}
	if p.Delta != (profile.Delta{}) {
		t.Fatalf("initial Delta = %+v, want zero", p.Delta)
	}
	if args := p.Read(); args != nil {
		t.Fatalf("initial Read() = %#v, want nil", args)
	}
	if len(p.Synapses()) != 0 {
		t.Fatalf("initial synapses = %d, want 0", len(p.Synapses()))
	}
}

func TestProfileNodeAccumulates(t *testing.T) {
	p := profile.NewProfileNode()

	p.Write([]any{profile.Delta{Amount: 2, Time: time.Second}})
	assertSnapshot(t, p, profile.Delta{Amount: 2, Time: time.Second})

	p.Write([]any{
		&profile.Delta{Amount: -1, Time: 2 * time.Millisecond},
		profile.Delta{Amount: 5, Time: 3 * time.Millisecond},
	})
	assertSnapshot(t, p, profile.Delta{Amount: 6, Time: time.Second + 5*time.Millisecond})
}

func TestProfileNodeIgnoresOtherItems(t *testing.T) {
	p := profile.NewProfileNode()
	p.Write([]any{
		nil,
		(*profile.Delta)(nil),
		1,
		"text",
		map[string]any{"amount": 10, "time": 10},
		struct{ Amount int }{Amount: 7},
		profile.Delta{Amount: 3, Time: time.Minute},
	})
	assertSnapshot(t, p, profile.Delta{Amount: 3, Time: time.Minute})
}

func TestProfileNodeNilAndEmptyData(t *testing.T) {
	p := profile.NewProfileNode()
	p.Write(nil)
	assertSnapshot(t, p, profile.Delta{})

	p.Write([]any{profile.Delta{Amount: 4}})
	p.Read()
	p.Write(nil)
	assertSnapshot(t, p, profile.Delta{Amount: 4})

	p.Write([]any{})
	assertSnapshot(t, p, profile.Delta{Amount: 4})
}

func TestProfileNodeArgumentsReplaced(t *testing.T) {
	// Repeated writes without a Read do not accumulate snapshots: the pending
	// arguments are always exactly one snapshot of the current totals.
	p := profile.NewProfileNode()
	p.Write([]any{profile.Delta{Amount: 1}})
	p.Write([]any{profile.Delta{Amount: 1}})
	assertSnapshot(t, p, profile.Delta{Amount: 2})
}

func TestProfileNodeSnapshotIsCopy(t *testing.T) {
	p := profile.NewProfileNode()
	p.Write([]any{profile.Delta{Amount: 1}})
	args := p.Read()
	p.Write([]any{profile.Delta{Amount: 1}})
	if args[0] != (profile.Delta{Amount: 1}) {
		t.Fatalf("earlier snapshot changed to %+v", args[0])
	}
}

func TestProfileNodeZeroValue(t *testing.T) {
	var p profile.ProfileNode
	p.Write([]any{profile.Delta{Amount: 1, Time: time.Nanosecond}})
	assertSnapshot(t, &p, profile.Delta{Amount: 1, Time: time.Nanosecond})
}

func TestProfileNodeActivation(t *testing.T) {
	p := profile.NewProfileNode()
	sink := graph.NewNode()
	p.Add(graph.NewSynapse().Add(sink))
	p.Write([]any{profile.Delta{Amount: 2}})
	if err := graph.Activate(p); err != nil {
		t.Fatal(err)
	}
	if got := sink.Read(); !reflect.DeepEqual(got, []any{profile.Delta{Amount: 2}}) {
		t.Fatalf("sink received %#v", got)
	}
	if got := p.Read(); got != nil {
		t.Fatalf("profile node arguments after activation = %#v, want nil", got)
	}
}

// profileNodes returns the targets of node's synapses that are profile nodes.
func profileNodes(node graph.Node) []*profile.ProfileNode {
	var result []*profile.ProfileNode
	for _, s := range node.Synapses() {
		if p, ok := s.Node().(*profile.ProfileNode); ok {
			result = append(result, p)
		}
	}
	return result
}

func TestNewProfileGraphProfilesAddedNodes(t *testing.T) {
	g := profile.NewProfileGraph()
	a, b, c := graph.NewNode(), graph.NewNode(), graph.NewNode()
	g.Add(a)
	g.Add(b)
	g.Add(c)

	seen := map[*profile.ProfileNode]bool{}
	for i, n := range []graph.Node{a, b, c} {
		ps := profileNodes(n)
		if len(ps) != 1 {
			t.Fatalf("node %d has %d profile nodes, want 1", i, len(ps))
		}
		if seen[ps[0]] {
			t.Fatalf("node %d shares a profile node", i)
		}
		seen[ps[0]] = true
	}

	// a's profile synapse comes first (added by the hook when a was added),
	// then the chain synapse to b (added when b was added), as in Python.
	if s := a.Synapses(); len(s) != 2 || s[1].Node() != b {
		t.Fatalf("a synapses unexpected: %d", len(s))
	}
	if s := b.Synapses(); len(s) != 2 || s[1].Node() != c {
		t.Fatalf("b synapses unexpected: %d", len(s))
	}
	if s := c.Synapses(); len(s) != 1 {
		t.Fatalf("c has %d synapses, want 1", len(s))
	}
	if n := len(g.Nodes()); n != 3 {
		t.Fatalf("graph has %d nodes, want 3 (profile nodes are not graph nodes)", n)
	}
}

func TestNewProfileGraphAttachNotProfiled(t *testing.T) {
	g := profile.NewProfileGraph()
	n := graph.NewNode()
	g.Attach(n)
	if len(n.Synapses()) != 0 {
		t.Fatalf("attached node has %d synapses, want 0", len(n.Synapses()))
	}
}

func TestNewProfileGraphCallerOptions(t *testing.T) {
	var order []string
	custom := graph.WithAddHook(func(_ *graph.Graph, node graph.Node) {
		order = append(order, fmt.Sprintf("custom:%d", len(node.Synapses())))
	})
	options := make([]graph.Option, 1, 4) // spare capacity must not be written
	options[0] = custom
	backing := options[:cap(options)]

	g := profile.NewProfileGraph(options...)
	n := graph.NewNode()
	g.Add(n)

	// Caller's hook ran first: the profile synapse did not exist yet.
	if !reflect.DeepEqual(order, []string{"custom:0"}) {
		t.Fatalf("custom hook calls = %v", order)
	}
	if len(profileNodes(n)) != 1 {
		t.Fatal("node not profiled")
	}
	for i := 1; i < len(backing); i++ {
		if backing[i] != nil {
			t.Fatalf("caller's options backing array modified at %d", i)
		}
	}
}

func TestNewProfileGraphFork(t *testing.T) {
	g := profile.NewProfileGraph()
	root := graph.NewNode()
	g.Add(root)
	forked := graph.NewNode()
	sub, err := g.Fork(forked)
	if err != nil {
		t.Fatal(err)
	}
	if len(profileNodes(forked)) != 1 {
		t.Fatal("forked node not profiled")
	}
	next := graph.NewNode()
	sub.Add(next)
	if len(profileNodes(next)) != 1 {
		t.Fatal("node added to sub-graph not profiled")
	}
	// The fork's activation node is attached, so it is not profiled.
	nodes := g.Nodes()
	if act := nodes[len(nodes)-1]; len(act.Synapses()) != 1 {
		t.Fatalf("activation node has %d synapses, want 1", len(act.Synapses()))
	}
	// The main chain node keeps one profile node.
	if len(profileNodes(root)) != 1 {
		t.Fatal("root profile count changed")
	}
}

// worker emits its input, a marker item, and a profiling record.
type worker struct {
	graph.BaseNode
	delta profile.Delta
}

func (w *worker) Process() error {
	in := w.Read()
	out := append([]any{}, in...)
	out = append(out, "work", w.delta)
	w.Arguments(out)
	return nil
}

func TestProfileGraphEndToEnd(t *testing.T) {
	g := profile.NewProfileGraph()
	w := &worker{delta: profile.Delta{Amount: 1, Time: 10 * time.Millisecond}}
	next := graph.NewNode()
	g.Add(w)
	g.Add(next)

	for run := 1; run <= 3; run++ {
		if err := g.Write([]any{"in"}); err != nil {
			t.Fatal(err)
		}
		if err := graph.Iterate(g.Nodes()); err != nil {
			t.Fatal(err)
		}
		p := profileNodes(w)[0]
		want := profile.Delta{Amount: run, Time: time.Duration(run) * 10 * time.Millisecond}
		if p.Delta != want {
			t.Fatalf("run %d: worker profile = %+v, want %+v", run, p.Delta, want)
		}
		if got := p.Read(); !reflect.DeepEqual(got, []any{want}) {
			t.Fatalf("run %d: worker profile args = %#v", run, got)
		}

		// Python quirk: the Delta also travels down the normal chain, so
		// next's profile node sees it as well.
		np := profileNodes(next)[0]
		if np.Delta != want {
			t.Fatalf("run %d: next profile = %+v, want %+v", run, np.Delta, want)
		}
	}
}

func TestProfileGraphDeltaReachesNextNode(t *testing.T) {
	g := profile.NewProfileGraph()
	d := profile.Delta{Amount: 2, Time: time.Second}
	w := &worker{delta: d}
	var received []any
	next := graph.NewFuncNode(func(args []any) ([]any, error) {
		received = args
		return nil, nil
	})
	g.Add(w)
	g.Add(next)
	if err := g.Write([]any{"in"}); err != nil {
		t.Fatal(err)
	}
	if err := graph.Iterate(g.Nodes()); err != nil {
		t.Fatal(err)
	}
	if want := []any{"in", "work", d}; !reflect.DeepEqual(received, want) {
		t.Fatalf("next received %#v, want %#v", received, want)
	}
	// next emitted nothing, but its profile node still holds a snapshot.
	if got := profileNodes(next)[0].Read(); !reflect.DeepEqual(got, []any{profile.Delta{}}) {
		t.Fatalf("next profile args = %#v", got)
	}
}

func ExampleNewProfileGraph() {
	g := profile.NewProfileGraph()
	w := &worker{delta: profile.Delta{Amount: 1, Time: 250 * time.Millisecond}}
	g.Add(w)

	for i := 0; i < 2; i++ {
		_ = g.Write([]any{i})
		_ = graph.Iterate(g.Nodes())
	}

	for _, s := range w.Synapses() {
		if p, ok := s.Node().(*profile.ProfileNode); ok {
			fmt.Println(p.Delta.Amount, p.Delta.Time)
			fmt.Println(p.Read()...)
		}
	}
	// Output:
	// 2 500ms
	// {2 500ms}
}

func ExampleProfileNode() {
	p := profile.NewProfileNode()
	p.Write([]any{"ignored", profile.Delta{Amount: 3, Time: time.Second}})
	p.Write([]any{&profile.Delta{Amount: -1, Time: time.Second}})
	fmt.Printf("%+v\n", p.Read()[0])
	// Output:
	// {Amount:2 Time:2s}
}
