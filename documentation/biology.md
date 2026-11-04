# Biology

How the Golang Compute Graph compares with a biological nervous system: what follows it, and what does not. The term
by term table is in [isomorphism.md](isomorphism.md).

The names `Node` and `Synapse` invite this comparison, and the comparison is useful, because it explains the structure
of the library in one picture. It is an analogy of structure, not a simulation: the library models no membranes, ions
or chemistry, and [much of the dynamics of real neurons is absent](#what-does-not-match). The biology here is the
textbook picture, and real neurons have many exceptions to it.

## The node as a neuron

A neuron receives signals on its dendrites, combines them in its cell body, and sends the result along its axon. A
`Node` is the aggregate of the same three parts:

```text
              Drain            Processor              Source
   synapse ─▶ dendrites ─────▶ cell body ───────────▶ axon ─┬─▶ synapse
              Write            Process                Read  └─▶ synapse
```

| Part of a neuron | Part of a `Node` | Method    | Role                       |
|------------------|------------------|-----------|----------------------------|
| Dendrites        | `Drain`          | `Write`   | Take signals in.           |
| Cell body        | `Processor`      | `Process` | Combine them.              |
| Axon             | `Source`         | `Read`    | Give the result out.       |

The analogy holds for the direction of flow too: signals enter a neuron at one end and leave at the other, and a
`Node` keeps the same order: `Write`, then `Process`, then `Read`.

## Summation

A neuron adds up the inputs which arrive over time, which is called temporal summation, and the inputs from many
synapses, which is called spatial summation.

The arguments of a node behave like this. `Write` only appends, so inputs accumulate, from one synapse or from many,
until the node runs. This is why **writing never activates**: as a single input does not necessarily make a neuron
fire, a write only adds to what is pending.

The difference is what decides about firing: see [activation](#activation).

## The synapse

A synapse is a directed junction from one neuron to another, and a `Synapse` is a directed edge to one target. As in
biology, the two are asymmetric: a signal goes from the node which holds the synapse to the target.

A synapse knows only the receiving side of its target, as it touches only the `Drain`. A presynaptic neuron, likewise,
does not reach into the cell body of the next one: it only acts on the membrane at the junction. This is why a target
of a synapse need not be a whole node, which is what the distributed synapse relies on.

## Activation

An action potential is the event by which a neuron fires. `Activate` is its counterpart:

| Step of `Activate`                 | Neuron                                                      |
|------------------------------------|-------------------------------------------------------------|
| `Process`                          | The cell body combines its inputs.                          |
| `Read`, which clears the arguments | The inputs are used up, and the neuron returns to its rest. |
| `Write` to every synapse           | The signal travels down every branch of the axon.           |

The same slice goes to every synapse, as the same action potential travels into every branch of an axon.

What starts the activation differs, and it is the deepest difference between the two. A neuron fires by itself, when
its summed input reaches a threshold. A node fires when something outside activates it: a call of `Activate`, or
`Iterate`, a `SpreadNode` or a `SubgraphActivationNode`. A closer biological picture of `Iterate` is an external
stimulus which drives each neuron of a pathway in turn.

`Activator` is for targets which activate in their own way, as different types of neuron do.

## Circuits

| Structure of a graph                | Biology                                                                           |
|-------------------------------------|-----------------------------------------------------------------------------------|
| A chain, from `Graph.Add`           | A feed-forward pathway, such as a reflex arc.                                     |
| Several synapses on one node        | Divergence: one neuron reaches many.                                              |
| Several synapses to one node        | Convergence: many neurons reach one.                                              |
| `Fork`                              | An axon collateral: a branch of the axon, which leads to a circuit of its own.    |
| `Connect`                           | A tract: the output of one circuit enters another at one point.                   |
| A cycle of synapses                 | A recurrent connection. `visualization` handles cycles.                           |
| `SpreadNode`                        | A burst: one input becomes a series of separate signals, and each is activated.   |
| `Attach` without a synapse          | A neuron which belongs to a population, without a connection to its neighbours.   |

## Development

A neuron differentiates and forms its connections before it carries signals, and `Initialize` is the counterpart of
that stage. All parameters of an object are given in its constructor, which is like what a cell inherits, and the
object begins to work after initialization. The `Initialize` of a `Node` is a no-op by default.

## Observation

| Tool                  | Neuroscience                                                                               |
|-----------------------|--------------------------------------------------------------------------------------------|
| `ProfileNode`         | A recording electrode on one neuron.                                                       |
| `ProfileGraph`        | An electrode array: every added node gets a recording electrode.                           |
| `Delta`               | One measurement.                                                                           |
| `GraphvizNode`        | A connectome: a map of who connects to whom, traced from a starting neuron.                |

Observation disturbs the experiment here, as it can there: `GraphvizNode` consumes the pending deltas of the profile
nodes which it reads.

## Failure

A failed `Write` is a conduction failure: the signal did not reach the next neuron, and `Activate` reports it. A signal
in a real circuit fails locally, though, and the circuit carries on, while `Iterate` stops at the first error.

A `nil` write has no effect, as a junction without a released signal has none.

## What does not match

These are not defects of the library, which is a dataflow framework and not a model of a brain. They mark where the
analogy ends.

| Biology                                                              | Compute Graph                                                                   |
|----------------------------------------------------------------------|---------------------------------------------------------------------------------|
| A neuron fires by itself, at a threshold.                            | A node fires only when activated from outside, on any input, even none.         |
| Firing is all-or-none: a spike carries no content.                   | An activation carries a list of arbitrary values.                               |
| Inputs excite or inhibit.                                            | There is no inhibition: inputs only accumulate.                                 |
| Synapses have strengths, and these change with use, which is learning. | Synapses have no weight and no state, and nothing is learned.                  |
| Membrane potential leaks away, so unused input fades.                | Arguments stay until they are read.                                             |
| A refractory period limits firing, and time matters.                 | There is no time, and a node can run again at once.                             |
| Billions of neurons work in parallel.                                | A graph runs on one goroutine, in insertion order.                              |
| A failure stays local.                                               | `Iterate` stops at the first error.                                             |
| A recording electrode is outside the circuit.                        | `Delta` values travel along regular synapses, mixed with the data.              |
| Neurons form and lose connections throughout life.                   | The graph is built once, and it has no way to remove a synapse.                 |
