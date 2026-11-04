# Isomorphism

The matches between the terms of the Golang Compute Graph and biological terms. The explanation is in
[biology.md](biology.md).

## How exact it is

A true isomorphism is a one-to-one mapping which preserves all structure. That holds here only at one level:

| Level                | Match    | Why                                                                                    |
|----------------------|----------|----------------------------------------------------------------------------------------|
| **Topology**         | Exact    | Nodes and synapses form a directed graph, as neurons and synapses do in a connectome. `a` has a synapse to `b` if and only if neuron `a` has a synapse onto neuron `b`. Any connectome can be written as a `Graph`. |
| **Signal direction** | Strong   | Input, processing and output have the same order, and edges have one direction.       |
| **Dynamics**         | Weak     | Thresholds, inhibition, time and parallel firing are absent.                           |
| **Learning**         | None     | Synapses have no strength, and nothing adapts.                                         |

Above the topology, this is an analogy which preserves some structure, and not a strict isomorphism.

Each match below has a strength:

| Strength | Meaning                                                   |
|----------|-----------------------------------------------------------|
| Exact    | The same structure and role.                              |
| Strong   | The same role, with minor differences.                    |
| Partial  | The same idea, with real differences, given in the notes. |
| Weak     | A helpful image only.                                     |

## Structure

| Software              | Biology                     | Strength | Notes                                                       |
|-----------------------|-----------------------------|----------|-------------------------------------------------------------|
| `Node`                | Neuron                      | Strong   | A node has no threshold.                                    |
| `Drain`               | Dendrites                   | Strong   | Where signals enter.                                        |
| `Processor`           | Cell body                   | Strong   | Where signals combine.                                      |
| `Source`              | Axon                        | Strong   | Where the signal leaves.                                    |
| `Synapse`             | Synapse                     | Strong   | Directed. Has no strength.                                  |
| `Synapse.Add`         | Forming a synapse           | Strong   |                                                             |
| `Synapses()`          | Branches of the axon        | Strong   | Each ends in one target.                                    |
| `Graph`               | Neural circuit              | Strong   |                                                             |
| Target of a synapse   | Postsynaptic neuron         | Exact    |                                                             |
| Holder of a synapse   | Presynaptic neuron          | Exact    |                                                             |
| `BaseNode`            | A generic neuron            | Weak     | A software base type.                                       |
| `FuncNode`            | Neuron type by its response | Partial  | Defined by what it computes.                                |

## Signals

| Software                     | Biology                   | Strength | Notes                                                 |
|------------------------------|---------------------------|----------|-------------------------------------------------------|
| Pending arguments            | Summed input              | Partial  | Keeps every value, while a membrane holds one potential, which leaks. |
| `Write`                      | Input at a synapse        | Strong   |                                                       |
| Accumulation by `Write`      | Temporal, spatial summation | Strong | Without inhibition.                                   |
| `Process`                    | Integration               | Partial  | Arbitrary code.                                       |
| `Read`                       | Taking up the summed input | Partial |                                                       |
| A list of values             | A signal                  | Weak     | A spike carries no content.                           |
| `nil` write                  | No signal released        | Strong   |                                                       |
| Same slice to every synapse  | One spike in all branches | Strong   |                                                       |

## Activation

| Software                  | Biology                         | Strength | Notes                                          |
|---------------------------|---------------------------------|----------|------------------------------------------------|
| `Activate`                | Action potential                | Partial  | Triggered from outside, not by a threshold.    |
| `Activator`               | A type with its own firing      | Partial  |                                                |
| `Iterate`                 | Stimulating a pathway in order  | Weak     | Neurons need no external sequencer.            |
| `Activation`              | External stimulation            | Weak     |                                                |
| Clearing by `Read`        | Return to rest                  | Weak     | No refractory period.                          |
| Error from `Activate`     | Conduction failure              | Partial  | Reported, which biology does not do.           |
| `Iterate` stops at error  | —                               | —        | No counterpart: real failures stay local.      |

## Circuits

| Software                  | Biology                 | Strength | Notes                                              |
|---------------------------|-------------------------|----------|----------------------------------------------------|
| Chain                     | Feed-forward pathway    | Strong   |                                                    |
| Several outgoing synapses | Divergence              | Exact    |                                                    |
| Several incoming synapses | Convergence             | Exact    |                                                    |
| Cycle                     | Recurrent connection    | Exact    | In the structure. `Iterate` does not run around it. |
| `Fork`                    | Axon collateral         | Strong   |                                                    |
| `SubgraphActivationNode`  | Entry to a local circuit | Partial |                                                    |
| `Connect`                 | Tract between circuits  | Partial  | One entry point only.                              |
| `SpreadNode`              | Burst                   | Partial  | Each item differs, while spikes are alike.         |
| `Attach` without a link   | Unconnected neuron      | Weak     |                                                    |

## Lifecycle

| Software      | Biology                           | Strength | Notes                                    |
|---------------|-----------------------------------|----------|------------------------------------------|
| Constructor   | What a cell inherits              | Weak     |                                          |
| `Initialize`  | Differentiation                   | Partial  | A no-op by default.                      |

## Observation

| Software                 | Biology             | Strength | Notes                                       |
|--------------------------|---------------------|----------|---------------------------------------------|
| `ProfileNode`            | Recording electrode | Partial  | Attached through a regular synapse.         |
| `ProfileGraph`           | Electrode array     | Partial  |                                             |
| `Delta`                  | A measurement       | Partial  | Travels with the data, which a real one does not. |
| `GraphvizNode`           | Connectome tracing  | Strong   |                                             |
| Edge label in a drawing  | Annotated connection | Weak    |                                             |

## Biology without a software counterpart

| Biology                | State in Compute Graph                |
|------------------------|---------------------------------------|
| Firing threshold       | Absent.                               |
| Inhibition             | Absent.                               |
| Synaptic strength      | Absent.                               |
| Plasticity, learning   | Absent.                               |
| Leak of potential      | Absent.                               |
| Refractory period      | Absent.                               |
| Conduction delay       | Absent in a process.                  |
| Glial cells            | Absent.                               |
| Neuromodulation        | Absent.                               |
