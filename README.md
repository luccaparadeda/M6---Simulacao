# queuesim — Tandem Queue Network Simulator

[![Build](https://github.com/luccaparadeda/M6---Simulacao/actions/workflows/build.yml/badge.svg)](https://github.com/luccaparadeda/M6---Simulacao/actions/workflows/build.yml)
[![License](https://img.shields.io/badge/License-All%20Rights%20Reserved-red.svg)](./LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.23%2B-00ADD8)](https://go.dev/)

A discrete-event simulator for **G/G/c/K queue networks**, written in Go. The default scenario is a two-queue tandem network, but the architecture (a `Router` interface + probabilistic routes per queue) supports arbitrary topologies out of the box.

This project was built as part of an Operations Research / Simulation course. It uses a custom Linear Congruential Generator (LCG) so runs are fully deterministic and reproducible across platforms.

---

## Table of contents

- [Features](#features)
- [Quick start](#quick-start)
  - [Download a prebuilt binary](#download-a-prebuilt-binary)
  - [Run from source](#run-from-source)
- [Sample output](#sample-output)
- [How the simulator works](#how-the-simulator-works)
- [Business rules](#business-rules)
- [Project layout](#project-layout)
- [Configuration](#configuration)
- [Extending to arbitrary topologies](#extending-to-arbitrary-topologies)
- [CI / Releases](#ci--releases)
- [Contributing](#contributing)
- [License](#license)

---

## Features

- Discrete-event simulation with a min-heap event scheduler.
- Generic **G/G/c/K** queues: configurable arrival interval, service interval, number of servers, capacity, and routing table.
- Probabilistic routing between queues; residual probability = exit from system.
- Deterministic LCG with a hard **budget of N random numbers** (the canonical stopping condition used in simulation coursework).
- Atomic event processing: random numbers for an event are sampled **before** mutating state, so the run never leaves the system in an inconsistent state when the RNG budget is hit.
- Final report with per-queue state-time distribution, state probabilities, loss count, and total simulated time.
- Single static binary — no runtime dependencies.

## Quick start

### Download a prebuilt binary

Grab the archive for your platform from the [latest release](../../releases/latest) and run it from a terminal. The CI pipeline publishes:

| OS | Arch | Filename |
|---|---|---|
| Linux | amd64 | `queuesim-linux-amd64` |
| Linux | arm64 | `queuesim-linux-arm64` |
| macOS | Intel | `queuesim-darwin-amd64` |
| macOS | Apple Silicon | `queuesim-darwin-arm64` |
| Windows | amd64 | `queuesim-windows-amd64.exe` |

#### macOS

```bash
chmod +x queuesim-darwin-arm64          # or queuesim-darwin-amd64 on Intel
xattr -d com.apple.quarantine queuesim-darwin-arm64  # first-run Gatekeeper bypass
./queuesim-darwin-arm64
```

> If macOS still blocks the binary, open *System Settings → Privacy & Security* and click **Allow Anyway**, then re-run.

#### Linux

```bash
chmod +x queuesim-linux-amd64
./queuesim-linux-amd64
```

#### Windows

In PowerShell or CMD from the folder where you downloaded the file:

```powershell
.\queuesim-windows-amd64.exe
```

> If SmartScreen warns about an unsigned binary, click *More info → Run anyway*.

### Run from source

**Requirements:** [Go 1.23+](https://go.dev/dl/).

```bash
git clone git@github.com:luccaparadeda/M6---Simulacao.git
cd M6---Simulacao
go run .
```

To produce a binary locally:

```bash
go build -trimpath -ldflags="-s -w" -o queuesim .
./queuesim
```

Cross-compiling locally (same matrix the CI uses):

```bash
GOOS=linux   GOARCH=amd64 go build -o dist/queuesim-linux-amd64 .
GOOS=darwin  GOARCH=arm64 go build -o dist/queuesim-darwin-arm64 .
GOOS=windows GOARCH=amd64 go build -o dist/queuesim-windows-amd64.exe .
```

## Sample output

Running the default scenario (`Tandem G/G/2/3 → G/G/1/5`, LCG seed `12345`, 100 000 RNG budget):

```
============================================================
  RELATÓRIO FINAL DA SIMULAÇÃO
============================================================
Tempo global total: 63035.7029
Números aleatórios consumidos: 100000

---- Fila Q1 (G/G/2/3) ----
  Perdas: 61
  Estado   Tempo Acum.     Probabilidade
  0        1255.2553       0.019913
  1        35897.0423      0.569472
  2        23922.1084      0.379501
  3        1961.2969       0.031114
  Tempo total observado: 63035.7029

---- Fila Q2 (G/G/1/5) ----
  Perdas: 335
  ...
```

## How the simulator works

1. **Scheduler** holds a priority queue of events `{Arrival, Departure}` ordered by time.
2. **Clock advancement**: before processing an event, the simulator accumulates `Δt = event.Time − lastClock` into the current state-time bucket of every queue, then advances the clock.
3. **Arrival event**: attempts to admit the customer. If the queue is full, a loss is recorded. Otherwise the customer enters; if a server is free, a departure is scheduled with a sampled service time. If the queue has external arrivals, the next external arrival is also scheduled.
4. **Departure event**: samples the route (always consumes an RNG when routes are configured), then samples the next service time if another customer is waiting, and the destination's service time if it accepts and has an idle server. Only after **all** samples succeed is state mutated — this keeps the stopping condition clean.
5. **Reporting**: after the RNG budget is exhausted, per-queue state probabilities are computed as `stateTime[i] / Σ stateTime`.

### RNG — Linear Congruential Generator

`Xₙ₊₁ = (a · Xₙ + c) mod M` with `a = 1 664 525`, `c = 1 013 904 223`, `M = 2³²`, seed `12345`. This is Numerical Recipes' LCG — good enough for coursework and, critically, deterministic and portable.

## Business rules

These are the conventions the simulator adopts. They match the ones typically assumed in academic simulation exercises; if you share this project with collaborators, align on these first — small differences here produce large differences in the final statistics.

| Rule | Decision |
|---|---|
| Initial queue population | Empty (all queues start at 0). |
| First external arrival | Injected at a fixed time `t₀` (default `1.5`). **Does not consume an RNG** — it is an input, not a sample. |
| Sampling the next external arrival | Consumes **1 RNG**. |
| Sampling a service time | Consumes **1 RNG** each time a server starts serving a customer. |
| Sampling a route | Consumes **1 RNG** whenever the source queue has a routing table, **even if the routing is deterministic** (a single route with probability 1.0). This matches the standard academic convention. |
| Residual routing probability | If the sum of outgoing probabilities is `< 1`, the remainder represents exit from the system. |
| Capacity `K` in G/G/c/K | **Total** customers in the queue (waiting + in service). E.g., `G/G/2/3` holds at most 3 simultaneous customers. |
| Loss | A customer that arrives at (or is routed to) a full queue is dropped and counted in that queue's loss counter. It does not retry. |
| Stopping condition | The simulation ends **exactly when the N-th random number has been used**. Events are atomic: the event that consumes the N-th RNG completes its mutation; no further events are started. |
| Global simulation time | The clock time of the last event processed. |
| Time is accumulated to the *old* state | For every event, `Δt` is credited to the state the queue was in *before* the event's mutation. |

## Project layout

```
.
├── main.go                 # default scenario + entrypoint
├── rng/
│   └── rng.go              # LCG with hard consumption budget
├── queue/
│   └── queue.go            # G/G/c/K queue struct + routing configuration
├── scheduler/
│   └── scheduler.go        # min-heap of timed events
├── sim/
│   └── sim.go              # Simulator + Router interface + ProbabilityRouter
├── .github/workflows/build.yml
├── go.mod
├── LICENSE
└── README.md
```

## Configuration

The scenario lives in `main.go` as a slice of `queue.Config`:

```go
configs := []queue.Config{
    {
        ID:          "Q1",
        Servers:     2,
        Capacity:    3,
        ArrivalMin:  1, ArrivalMax: 4,
        ServiceMin:  3, ServiceMax: 4,
        HasExternal: true,
        Routes:      []queue.Route{{ID: "Q2", Probability: 1.0}},
    },
    {
        ID:         "Q2",
        Servers:    1,
        Capacity:   5,
        ServiceMin: 2, ServiceMax: 3,
    },
}

r := rng.NewLCG(12345, 100_000)
s := sim.New(configs, r)
s.ScheduleFirstArrival("Q1", 1.5)
s.Run()
s.Report()
```

| Field | Meaning |
|---|---|
| `ID` | Queue identifier (any unique string). |
| `Servers` | Number of parallel servers (`c`). |
| `Capacity` | Total capacity (`K`). Use `-1` for infinite. |
| `ArrivalMin/Max` | Uniform interval for **external** inter-arrival times. |
| `ServiceMin/Max` | Uniform interval for service times. |
| `HasExternal` | `true` if the queue receives external arrivals. |
| `Routes` | Slice of `{ID, Probability}`. Residual probability = exit. |

## Extending to arbitrary topologies

The default `ProbabilityRouter` is driven entirely by each queue's `Routes`, so:

- Add more queues to `configs`.
- Set `Routes` on any queue with the desired probabilities.
- Any queue with `HasExternal = true` can be an entry point.

To implement a different routing policy (e.g., *join-the-shortest-queue*), implement the `sim.Router` interface:

```go
type Router interface {
    Next(fromQueueID string, r *rng.LCG) (destID string, ok bool)
}
```

and inject it by setting `simulator.Router = myRouter` after `sim.New`.

## CI / Releases

The `.github/workflows/build.yml` workflow:

- Runs `go vet` on every push and PR.
- Cross-compiles `queuesim` for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64` with `CGO_ENABLED=0` (fully static binaries).
- Uploads each binary as a workflow artifact.
- When a tag `v*` is pushed, attaches the binaries to a GitHub Release and auto-generates release notes.

To cut a release:

```bash
git tag v0.1.0
git push origin v0.1.0
```

## Contributing

Issues and pull requests are welcome. Before opening a PR, please:

1. Run `go vet ./...` and `go build ./...`.
2. Keep the simulator's **business rules** (above) intact, or update this README in the same commit if you are intentionally changing them — they directly affect the numerical results.

## License

**All rights reserved** © 2026 Lucca Paradeda — see [LICENSE](./LICENSE).

This repository is publicly visible for review and portfolio purposes only. No permission is granted to copy, modify, or redistribute the code. Students enrolled in the same course as the author may **not** reuse this work in any form as an academic deliverable.
