package visualization

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vitche/go_compute_graph/graph"
	"github.com/vitche/go_compute_graph/profile"
)

// named is a node with a fixed String() name.
type named struct {
	graph.BaseNode
	name string
}

func (n *named) String() string { return n.name }

func newNamed(name string) *named { return &named{name: name} }

// recorder records the arguments it receives when processed.
type recorder struct {
	graph.BaseNode
	got []any
}

func (r *recorder) Process() error {
	r.got = append(r.got, r.Read()...)
	return nil
}

func link(from, to graph.Node) {
	from.Add(graph.NewSynapse().Add(to))
}

// newRoot creates a Graphviz node and returns it with its quoted DOT name.
func newRoot() (*GraphvizNode, string) {
	v := NewGraphvizNode()
	return v, `"` + fmt.Sprintf("%T@%p", v, v) + `"`
}

// stub replaces lookPath and runCommand for the duration of a test.
func stub(t *testing.T, look func(string) (string, error), run func(string, ...string) error) {
	t.Helper()
	oldLook, oldRun := lookPath, runCommand
	t.Cleanup(func() { lookPath, runCommand = oldLook, oldRun })
	if look != nil {
		lookPath = look
	}
	if run != nil {
		runCommand = run
	}
}

func requireDot(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("dot"); err != nil {
		t.Skip("Graphviz dot executable not installed")
	}
}

func TestNewGraphvizNode(t *testing.T) {
	v := NewGraphvizNode()
	if v.Format != "png" || v.Output != "" {
		t.Fatalf("got Format=%q Output=%q", v.Format, v.Output)
	}
}

func TestDOTChain(t *testing.T) {
	root, v := newRoot()
	a, b := newNamed("a"), newNamed("b")
	link(root, a)
	link(a, b)
	want := "digraph {\n" +
		"\t\"a\"\n" +
		"\t\"b\"\n" +
		"\t\"a\" -> \"b\" [label=\"\"]\n" +
		"\t" + v + " -> \"a\" [label=\"\"]\n" +
		"}\n"
	if got := root.DOT(); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

func TestDOTEmpty(t *testing.T) {
	root, _ := newRoot()
	if got := root.DOT(); got != "digraph {\n}\n" {
		t.Fatalf("got %q", got)
	}
}

func TestDOTFanOut(t *testing.T) {
	root, v := newRoot()
	a, b, c := newNamed("a"), newNamed("b"), newNamed("c")
	link(root, a)
	link(root, b)
	link(a, c)
	link(b, c)
	want := "digraph {\n" +
		"\t\"a\"\n" +
		"\t\"c\"\n" +
		"\t\"a\" -> \"c\" [label=\"\"]\n" +
		"\t" + v + " -> \"a\" [label=\"\"]\n" +
		"\t\"b\"\n" +
		"\t\"c\"\n" +
		"\t\"b\" -> \"c\" [label=\"\"]\n" +
		"\t" + v + " -> \"b\" [label=\"\"]\n" +
		"}\n"
	if got := root.DOT(); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

func TestDOTCycle(t *testing.T) {
	root, v := newRoot()
	a, b := newNamed("a"), newNamed("b")
	link(root, a)
	link(a, b)
	link(b, a)
	link(b, root)
	want := "digraph {\n" +
		"\t\"a\"\n" +
		"\t\"b\"\n" +
		"\t\"a\"\n" +
		"\t\"b\" -> \"a\" [label=\"\"]\n" +
		"\t" + v + "\n" +
		"\t\"b\" -> " + v + " [label=\"\"]\n" +
		"\t\"a\" -> \"b\" [label=\"\"]\n" +
		"\t" + v + " -> \"a\" [label=\"\"]\n" +
		"}\n"
	if got := root.DOT(); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	// The visited set is per traversal: a second call gives the same output.
	if got := root.DOT(); got != want {
		t.Fatalf("second DOT differs:\n%s", got)
	}
}

func TestDOTSkipsNilSynapses(t *testing.T) {
	root, v := newRoot()
	root.Add(nil)
	root.Add(graph.NewSynapse())
	var typedNil *profile.ProfileNode
	root.Add(graph.NewSynapse().Add(typedNil))
	link(root, newNamed("a"))
	want := "digraph {\n\t\"a\"\n\t" + v + " -> \"a\" [label=\"\"]\n}\n"
	if got := root.DOT(); got != want {
		t.Fatalf("got\n%s", got)
	}
}

// namedDrain is a target which is not a Node, with a name.
type namedDrain struct{ name string }

func (d namedDrain) Write([]any) error { return nil }
func (d namedDrain) String() string    { return d.name }

// bareDrain is a target which is not a Node, without a name.
type bareDrain struct{}

func (bareDrain) Write([]any) error { return nil }

func TestDOTDrawsANamedDrainAsALeaf(t *testing.T) {
	v := NewGraphvizNode()
	a := &named{name: "a"}
	a.Add(graph.NewSynapse().Add(namedDrain{name: "remote"}))
	v.Add(graph.NewSynapse().Add(a))

	dot := v.DOT()
	if !strings.Contains(dot, `"a" -> "remote"`) {
		t.Fatalf("no edge to the drain in:\n%s", dot)
	}
}

func TestDOTSkipsAnUnnamedDrain(t *testing.T) {
	v := NewGraphvizNode()
	a := &named{name: "a"}
	a.Add(graph.NewSynapse().Add(bareDrain{}))
	v.Add(graph.NewSynapse().Add(a))

	if dot := v.DOT(); strings.Count(dot, "->") != 1 {
		t.Fatalf("expected only the edge to a in:\n%s", dot)
	}
}

func TestNodeNameFallback(t *testing.T) {
	n := graph.NewNode()
	want := fmt.Sprintf("*graph.BaseNode@%p", n)
	if got := nodeName(n); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	v := NewGraphvizNode()
	link(v, n)
	vName := fmt.Sprintf("*visualization.GraphvizNode@%p", v)
	if !strings.HasPrefix(vName, "*visualization.GraphvizNode@0x") {
		t.Fatalf("unexpected name %q", vName)
	}
	dot := v.DOT()
	if !strings.Contains(dot, "\t\""+vName+"\" -> \""+want+"\" [label=\"\"]\n") {
		t.Fatalf("got\n%s", dot)
	}
}

func TestDOTEscaping(t *testing.T) {
	root, v := newRoot()
	a, b := newNamed(`say "hi"`), newNamed(`C:\dir`)
	link(root, a)
	link(a, b)
	want := "digraph {\n" +
		"\t\"say \\\"hi\\\"\"\n" +
		"\t\"C:\\\\dir\"\n" +
		"\t\"say \\\"hi\\\"\" -> \"C:\\\\dir\" [label=\"\"]\n" +
		"\t" + v + " -> \"say \\\"hi\\\"\" [label=\"\"]\n" +
		"}\n"
	if got := root.DOT(); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

func TestDOTLabels(t *testing.T) {
	tests := []struct {
		name  string
		value any
		label string
	}{
		{"positive", profile.Delta{Amount: 3, Time: 250 * time.Millisecond}, "+3 (N), 0.25 (s)"},
		{"zero", profile.Delta{}, "0 (N), 0 (s)"},
		{"negative", profile.Delta{Amount: -2, Time: 1500 * time.Millisecond}, "-2 (N), 1.5 (s)"},
		{"pointer", &profile.Delta{Amount: 1, Time: time.Second}, "+1 (N), 1 (s)"},
		{"nil pointer", (*profile.Delta)(nil), ""},
		{"not a delta", "text", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, v := newRoot()
			a, b := newNamed("a"), newNamed("b")
			link(root, a)
			link(a, b)
			p := profile.NewProfileNode()
			a.Add(graph.NewSynapse().Add(p))
			p.Arguments([]any{tt.value})
			want := "digraph {\n" +
				"\t\"a\"\n" +
				"\t\"b\"\n" +
				"\t\"a\" -> \"b\" [label=\"" + tt.label + "\"]\n" +
				"\t" + v + " -> \"a\" [label=\"\"]\n" +
				"}\n"
			if got := root.DOT(); got != want {
				t.Fatalf("got\n%s\nwant\n%s", got, want)
			}
			if rest := p.Read(); len(rest) != 0 {
				t.Fatalf("deltas not consumed: %v", rest)
			}
		})
	}
}

func TestDOTProfileWrite(t *testing.T) {
	// Deltas written by the profile node itself, and an empty profile node.
	root, v := newRoot()
	a := newNamed("a")
	link(root, a)
	empty := profile.NewProfileNode()
	p := profile.NewProfileNode()
	root.Add(graph.NewSynapse().Add(p))
	root.Add(graph.NewSynapse().Add(empty))
	p.Write([]any{profile.Delta{Amount: 2, Time: time.Second}, &profile.Delta{Amount: 1}})
	want := "digraph {\n\t\"a\"\n\t" + v + " -> \"a\" [label=\"+3 (N), 1 (s)\"]\n}\n"
	if got := root.DOT(); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	// Consumed: the next traversal has no label.
	want = "digraph {\n\t\"a\"\n\t" + v + " -> \"a\" [label=\"\"]\n}\n"
	if got := root.DOT(); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

func TestDOTProfileNodesNotDrawn(t *testing.T) {
	g := profile.NewProfileGraph()
	root, v := newRoot()
	g.Add(root)
	g.Add(newNamed("a"))
	dot := root.DOT()
	if strings.Contains(dot, "ProfileNode") {
		t.Fatalf("profile node drawn:\n%s", dot)
	}
	want := "digraph {\n\t\"a\"\n\t" + v + " -> \"a\" [label=\"\"]\n}\n"
	if dot != want {
		t.Fatalf("got\n%s", dot)
	}
}

func TestSanitize(t *testing.T) {
	if got := sanitize("*a.B@0x1f/c d-_9"); got != "_a.B_0x1f_c_d-_9" {
		t.Fatalf("got %q", got)
	}
}

func TestProcessRender(t *testing.T) {
	requireDot(t)
	for _, format := range []string{"png", "svg", ""} {
		t.Run("format="+format, func(t *testing.T) {
			root, v := newRoot()
			link(root, newNamed("a"))
			root.Format = format
			root.Output = filepath.Join(t.TempDir(), "out")
			if err := root.Process(); err != nil {
				t.Fatal(err)
			}
			gv, err := os.ReadFile(root.Output + ".gv")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(gv), v+` -> "a"`) {
				t.Fatalf("unexpected DOT source:\n%s", gv)
			}
			ext := format
			if ext == "" {
				ext = "png"
			}
			rendered, err := os.ReadFile(root.Output + "." + ext)
			if err != nil {
				t.Fatal(err)
			}
			switch ext {
			case "png":
				if !strings.HasPrefix(string(rendered), "\x89PNG") {
					t.Fatal("output is not a PNG")
				}
			case "svg":
				if !strings.Contains(string(rendered), "<svg") {
					t.Fatal("output is not an SVG")
				}
			}
		})
	}
}

// No t.Parallel: the test changes the working directory.
func TestProcessDefaultOutput(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatal(err)
		}
	})

	var args []string
	stub(t, func(string) (string, error) { return "/x/dot", nil }, func(name string, a ...string) error {
		args = append([]string{name}, a...)
		return nil
	})
	v := NewGraphvizNode()
	v.Format = ""
	if err := v.Process(); err != nil {
		t.Fatal(err)
	}
	base := sanitize(fmt.Sprintf("%T@%p", v, v))
	want := []string{"/x/dot", "-Tpng", "-o", base + ".png", base + ".gv"}
	if strings.Join(args, " ") != strings.Join(want, " ") {
		t.Fatalf("got %v want %v", args, want)
	}
	if _, err := os.Stat(filepath.Join(dir, base+".gv")); err != nil {
		t.Fatal(err)
	}
}

func TestProcessDotNotFound(t *testing.T) {
	notFound := errors.New("not found")
	stub(t, func(string) (string, error) { return "", notFound }, func(string, ...string) error {
		t.Fatal("runCommand called")
		return nil
	})
	v := NewGraphvizNode()
	v.Output = filepath.Join(t.TempDir(), "out")
	if err := v.Process(); !errors.Is(err, notFound) || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("got %v", err)
	}
}

func TestProcessRenderFailure(t *testing.T) {
	failed := errors.New("boom")
	stub(t, func(string) (string, error) { return "dot", nil }, func(string, ...string) error { return failed })
	v := NewGraphvizNode()
	v.Output = filepath.Join(t.TempDir(), "out")
	if err := v.Process(); !errors.Is(err, failed) || !strings.Contains(err.Error(), "render") {
		t.Fatalf("got %v", err)
	}
}

func TestProcessRealRenderFailure(t *testing.T) {
	requireDot(t)
	v := NewGraphvizNode()
	v.Format = "no-such-format"
	v.Output = filepath.Join(t.TempDir(), "out")
	err := v.Process()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("got %v", err)
	}
}

func TestRunCommand(t *testing.T) {
	if err := runCommand("true"); err != nil {
		t.Fatalf("true: %v", err)
	}
	if err := runCommand("sh", "-c", "echo oops; exit 1"); err == nil || !strings.Contains(err.Error(), "oops") {
		t.Fatalf("got %v", err)
	}
}

func TestProcessWriteFailure(t *testing.T) {
	stub(t, func(string) (string, error) {
		t.Fatal("lookPath called")
		return "", nil
	}, nil)
	v := NewGraphvizNode()
	v.Output = filepath.Join(t.TempDir(), "missing", "out")
	if err := v.Process(); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("got %v", err)
	}
}

func TestProcessPassesInputThrough(t *testing.T) {
	stub(t, func(string) (string, error) { return "dot", nil }, func(string, ...string) error { return nil })
	v := NewGraphvizNode()
	v.Output = filepath.Join(t.TempDir(), "out")
	down := &recorder{}
	link(v, down)
	v.Write([]any{1, "two"})
	if err := graph.Activate(v); err != nil {
		t.Fatal(err)
	}
	if err := graph.Activate(down); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(down.got) != "[1 two]" {
		t.Fatalf("got %v", down.got)
	}
}

// Port of the Python examples/visualization.py.
func TestExample(t *testing.T) {
	requireDot(t)
	g := profile.NewProfileGraph()
	v := NewGraphvizNode()
	v.Output = filepath.Join(t.TempDir(), "example")
	g.Add(v)
	for i := 0; i < 7; i++ {
		g.Add(graph.NewNode())
	}
	if err := graph.Activate(g.First()); err != nil {
		t.Fatal(err)
	}
	gv, err := os.ReadFile(v.Output + ".gv")
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(gv), " -> "); n != 7 {
		t.Fatalf("got %d edges:\n%s", n, gv)
	}
	if strings.Contains(string(gv), "ProfileNode") {
		t.Fatalf("profile node drawn:\n%s", gv)
	}
	info, err := os.Stat(v.Output + ".png")
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Fatal("empty PNG")
	}
}
