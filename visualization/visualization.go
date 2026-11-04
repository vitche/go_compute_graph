// Package visualization renders compute graphs with Graphviz.
//
// It is a Go port of the compute.graph.visualization module of
// https://github.com/vitche/python-compute-graph. GraphvizNode traverses the
// graph depth-first from itself, writes the DOT source natively and renders
// it with the Graphviz `dot` executable. Profile nodes (see package profile)
// are not drawn: their pending deltas label the edges of the node they
// profile.
package visualization

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/vitche/go_compute_graph/graph"
	"github.com/vitche/go_compute_graph/profile"
)

// lookPath finds the Graphviz executable. Replaced in tests.
var lookPath = exec.LookPath

// runCommand runs an executable and reports its combined output on failure.
// Replaced in tests.
var runCommand = func(name string, args ...string) error {
	output, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// GraphvizNode draws the graph reachable from itself when processed.
type GraphvizNode struct {
	graph.BaseNode
	// Output is the output file base name; it may include a directory.
	// When empty, the sanitized node name is used.
	Output string
	// Format is the output format passed to `dot -T`; "" means "png".
	Format string
}

// NewGraphvizNode creates a Graphviz node that renders PNG files named after
// the node.
func NewGraphvizNode() *GraphvizNode {
	return &GraphvizNode{Format: "png"}
}

// DOT traverses the graph from v and returns its DOT source. Like the Python
// original, it reads (and therefore consumes) the pending deltas of the
// profile nodes it encounters.
//
// The traversal starts at v itself, so the root is named after the
// *GraphvizNode (embedded methods are not virtual: a type that embeds
// GraphvizNode does not change the root's name or identity).
func (v *GraphvizNode) DOT() string {
	var b strings.Builder
	b.WriteString("digraph {\n")
	traverse(&b, v, map[graph.Node]bool{})
	b.WriteString("}\n")
	return b.String()
}

// Process writes the DOT source to <Output>.gv and renders it to
// <Output>.<Format> with `dot`. It does not read its own arguments, so the
// node's input passes through to its synapses.
func (v *GraphvizNode) Process() error {
	format := v.Format
	if format == "" {
		format = "png"
	}
	output := v.Output
	if output == "" {
		output = sanitize(nodeName(v))
	}
	source := output + ".gv"
	if err := os.WriteFile(source, []byte(v.DOT()), 0o644); err != nil {
		return fmt.Errorf("visualization: write DOT source: %w", err)
	}
	dot, err := lookPath("dot")
	if err != nil {
		return fmt.Errorf("visualization: Graphviz dot executable not found: %w", err)
	}
	if err := runCommand(dot, "-T"+format, "-o", output+"."+format, source); err != nil {
		return fmt.Errorf("visualization: render %s: %w", source, err)
	}
	return nil
}

// traverse mirrors Python's GraphvizNode.__traverse.
func traverse(b *strings.Builder, node graph.Node, opened map[graph.Node]bool) {
	if opened[node] {
		return
	}
	opened[node] = true
	synapses := node.Synapses()

	var delta *profile.Delta
	for _, synapse := range synapses {
		if synapse == nil {
			continue
		}
		if profileNode, ok := synapse.Node().(*profile.ProfileNode); ok && profileNode != nil {
			if deltas := profileNode.Read(); len(deltas) > 0 {
				delta = asDelta(deltas[0])
			}
		}
	}

	label := ""
	if delta != nil {
		amount := strconv.Itoa(delta.Amount)
		if delta.Amount > 0 {
			amount = "+" + amount
		}
		label = amount + " (N), " + strconv.FormatFloat(delta.Time.Seconds(), 'f', -1, 64) + " (s)"
	}

	for _, synapse := range synapses {
		if synapse == nil {
			continue
		}
		neighbor := synapse.Node()
		if neighbor == nil {
			// A target which is not a whole node is a leaf: there is nothing
			// to traverse, and it is drawn when it can name itself.
			if stringer, ok := synapse.Drain().(fmt.Stringer); ok {
				name := quote(stringer.String())
				fmt.Fprintf(b, "\t%s\n", name)
				fmt.Fprintf(b, "\t%s -> %s [label=%s]\n", quote(nodeName(node)), name, quote(label))
			}
			continue
		}
		if _, ok := neighbor.(*profile.ProfileNode); ok {
			continue
		}
		fmt.Fprintf(b, "\t%s\n", quote(nodeName(neighbor)))
		traverse(b, neighbor, opened)
		fmt.Fprintf(b, "\t%s -> %s [label=%s]\n", quote(nodeName(node)), quote(nodeName(neighbor)), quote(label))
	}
}

// asDelta converts a pending profile value to a delta, or returns nil.
func asDelta(value any) *profile.Delta {
	switch d := value.(type) {
	case profile.Delta:
		return &d
	case *profile.Delta:
		return d
	}
	return nil
}

// nodeName returns the node's String() or, failing that, "%T@%p".
func nodeName(node graph.Node) string {
	if stringer, ok := node.(fmt.Stringer); ok {
		return stringer.String()
	}
	return fmt.Sprintf("%T@%p", node, node)
}

// quote returns s as a double-quoted DOT ID.
func quote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

// sanitize replaces every character outside [A-Za-z0-9._-] with '_'.
func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			return r
		}
		return '_'
	}, s)
}
