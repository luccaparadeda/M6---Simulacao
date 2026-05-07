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
- [T1 — Validation against `simulator.jar`](#t1--validation-against-simulatorjar)
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
- Deterministic LCG with a hard **budget of N random numbers per seed** (the canonical stopping condition used in simulation coursework).
- Multi-seed orchestration: runs N independent simulations and aggregates results.
- Atomic event processing: random numbers for an event are sampled **before** mutating state, so the run never leaves the system in an inconsistent state when the RNG budget is hit.
- YAML-driven scenario file (`simulation.yml`) — no recompilation needed to change the network.
- Final report formatted to match the academic `simulator.jar` output for direct comparison.
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

### Command-line flags

| Flag | Default | Description |
|---|---|---|
| `-config` | `simulation.yml` | Path to the YAML scenario file (queues, routes, seeds, RNG budget). |

The full scenario — number of queues, servers, capacity, service/arrival intervals, routing table, list of seeds, RNG budget per seed, and the time of the first external arrival — is defined in `simulation.yml`. See [Configuration](#configuration) for the schema.

```bash
# Run with the default config file (./simulation.yml)
./queuesim-darwin-arm64

# Use a custom path
./queuesim-darwin-arm64 -config=path/to/scenario.yml
```

### Run from source

**Requirements:** [Go 1.23+](https://go.dev/dl/).

```bash
git clone git@github.com:luccaparadeda/M6---Simulacao.git
cd M6---Simulacao
go run . -config=simulation.yml
```

To produce a binary locally:

```bash
go build -trimpath -ldflags="-s -w" -o queuesim .
./queuesim                       # uses ./simulation.yml by default
./queuesim -config=other.yml     # custom scenario
```

Cross-compiling locally (same matrix the CI uses):

```bash
GOOS=linux   GOARCH=amd64 go build -o dist/queuesim-linux-amd64 .
GOOS=darwin  GOARCH=arm64 go build -o dist/queuesim-darwin-arm64 .
GOOS=windows GOARCH=amd64 go build -o dist/queuesim-windows-amd64.exe .
```

## T1 — Validation against `simulator.jar`

The `T1` release is the deliverable for *Trabalho 1*: it bundles the cross-platform binaries together with the `simulation.yml` scenario used for validation. The same parameters can be fed to the academic Java reference simulator (`simulator.jar`) and the two outputs compared side by side.

### Scenario

`simulation.yml` describes a 3-queue network:

| Queue | Type | Service (min) | External arrival (min) | Routes |
|---|---|---|---|---|
| Q1 | G/G/1 (∞) | 1.0…2.0 | 2.0…4.0 | → Q2 (0.8) · → Q3 (0.2) |
| Q2 | G/G/2/5 | 4.0…6.0 | — | → Q1 (0.3) · → Q3 (0.5) · exit (0.2) |
| Q3 | G/G/2/10 | 5.0…15.0 | — | → Q2 (0.7) · exit (0.3) |

Run with seeds `[1, 2, 3, 4, 5]`, RNG budget `100 000` per seed, first arrival at `t = 1.5`.

### How a reviewer runs the validation

1. Download the `T1` release: <https://github.com/luccaparadeda/M6---Simulacao/releases/tag/T1>
2. Pick the binary for your platform and put it in the same directory as the bundled `simulation.yml`.
3. Run it:

   ```bash
   # macOS Apple Silicon — adjust filename for your platform
   chmod +x queuesim-darwin-arm64
   xattr -d com.apple.quarantine queuesim-darwin-arm64
   ./queuesim-darwin-arm64
   ```

   No flags are needed — the binary picks up `./simulation.yml` automatically.

4. (Optional) Re-run the academic Java simulator with the equivalent `model.yml` and compare:

   ```bash
   docker compose run --rm simulator
   ```

   Both reports use the same banner, per-queue G/G/c/K block, and footer, so a line-by-line diff is meaningful.

### Expected agreement

The two simulators use different LCG constants, so individual sample paths will differ. Under identical parameters they nevertheless converge to the same steady-state distribution — in the validation run, probabilities agreed to within **≤0.5 percentage points**, losses within **~0.6%**, and the simulation average time within **~0.06%**.

## Sample output

Running with the bundled `simulation.yml` produces output in the academic report format:

```
Simulation: #1
...simulating with random numbers (seed '1')...
Simulation: #2
...simulating with random numbers (seed '2')...
...
=========================================================
=================    END OF SIMULATION   ================
=========================================================

=========================================================
======================    REPORT   ======================
=========================================================
*********************************************************
Queue:   Q1 (G/G/1)
Arrival: 2.0 ... 4.0
Service: 1.0 ... 2.0
*********************************************************
   State               Time               Probability
      0           66335.6520                32.24%
      1          108714.1723                52.84%
      2           28267.1322                13.74%
      3            2350.9593                 1.14%
      4              66.8835                 0.03%
      5               1.2698                 0.00%

Number of losses: 0

...

=========================================================
Simulation average time: 41147.2138
=========================================================
```

## How the simulator works

1. **Scheduler** holds a priority queue of events `{Arrival, Departure}` ordered by time.
2. **Clock advancement**: before processing an event, the simulator accumulates `Δt = event.Time − lastClock` into the current state-time bucket of every queue, then advances the clock.
3. **Arrival event**: attempts to admit the customer. If the queue is full, a loss is recorded. Otherwise the customer enters; if a server is free, a departure is scheduled with a sampled service time. If the queue has external arrivals, the next external arrival is also scheduled.
4. **Departure event**: samples the route (always consumes an RNG when routes are configured), then samples the next service time if another customer is waiting, and the destination's service time if it accepts and has an idle server. Only after **all** samples succeed is state mutated — this keeps the stopping condition clean.
5. **Reporting**: after the RNG budget is exhausted, per-queue state probabilities are computed as `stateTime[i] / Σ stateTime`.

### RNG — Linear Congruential Generator

`Xₙ₊₁ = (a · Xₙ + c) mod M` with `a = 1 664 525`, `c = 1 013 904 223`, `M = 2³²`. This is Numerical Recipes' LCG — good enough for coursework and, critically, deterministic and portable. Each seed listed under `simulation.seeds` initializes a fresh generator for that run.

## Business rules

These are the conventions the simulator adopts. They match the ones typically assumed in academic simulation exercises; if you share this project with collaborators, align on these first — small differences here produce large differences in the final statistics.

| Rule | Decision |
|---|---|
| Initial queue population | Empty (all queues start at 0). |
| First external arrival | Injected at the time configured in `simulation.firstArrival.time`. **Does not consume an RNG** — it is an input, not a sample. |
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
├── main.go                 # CLI entrypoint: loads YAML, runs N seeds, prints report
├── simulation.yml          # default scenario fed to the simulator
├── config/
│   └── config.go           # YAML loader + validator + translator into queue.Config
├── rng/
│   └── rng.go              # LCG with hard consumption budget
├── queue/
│   └── queue.go            # G/G/c/K queue struct + routing configuration
├── scheduler/
│   └── scheduler.go        # min-heap of timed events
├── logger/
│   └── logger.go           # CSV/JSON event logger (library; not exposed via CLI in this build)
├── sim/
│   └── sim.go              # Simulator + Router interface + ProbabilityRouter
├── .github/workflows/build.yml
├── go.mod
├── LICENSE
└── README.md
```

## Configuration

The scenario is fully described by `simulation.yml`:

```yaml
simulation:
  seeds: [1, 2, 3, 4, 5]   # one independent run per seed
  rngPerSeed: 100000       # RNG budget per run
  firstArrival:
    queue: Q1
    time: 1.5

queues:
  - id: Q1
    servers: 1             # G/G/1 with infinite capacity (omit `capacity`)
    service: { min: 1.0, max: 2.0 }
    arrival: { min: 2.0, max: 4.0 }   # only on entry queues
    routes:
      - { to: Q2, probability: 0.8 }
      - { to: Q3, probability: 0.2 }

  - id: Q2
    servers: 2
    capacity: 5
    service: { min: 4.0, max: 6.0 }
    routes:
      - { to: Q1, probability: 0.3 }
      - { to: Q3, probability: 0.5 }   # residual 0.2 → exit
```

| Field | Meaning |
|---|---|
| `simulation.seeds` | List of LCG seeds. The simulator runs one full pass per seed and the report aggregates state-times across them. |
| `simulation.rngPerSeed` | RNG budget for each individual run. |
| `simulation.firstArrival.queue` / `.time` | Which queue receives the first external arrival, and at what clock time. |
| `queues[].id` | Queue identifier (any unique string). |
| `queues[].servers` | Number of parallel servers (`c`). |
| `queues[].capacity` | Total capacity (`K`). **Omit** for infinite. |
| `queues[].service.{min,max}` | Uniform interval for service times. |
| `queues[].arrival.{min,max}` | Uniform interval for **external** inter-arrival times. Omit if the queue has no external arrivals. |
| `queues[].routes` | List of `{to, probability}`. Residual probability (`1 − Σ`) = exit from the system. |

## Extending to arbitrary topologies

Edit `simulation.yml`:

- Add more queue blocks under `queues:`.
- Set `routes` on any queue with the desired probabilities.
- Any queue with an `arrival` block can be an entry point — name it under `firstArrival.queue`.

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
- On every push to `main`, updates a rolling **"Latest (main)"** pre-release with fresh binaries.
- When a tag matching `v*` **or** `T*` is pushed, creates a versioned GitHub Release. The release bundle includes every binary plus the current `simulation.yml`, so a reviewer can download a single zip and run.

To cut a versioned release:

```bash
git tag v0.1.0
git push origin v0.1.0
```

To cut a course-deliverable release (e.g. T1):

```bash
git tag T1
git push origin T1
```

## Contributing

Issues and pull requests are welcome. Before opening a PR, please:

1. Run `go vet ./...` and `go build ./...`.
2. Keep the simulator's **business rules** (above) intact, or update this README in the same commit if you are intentionally changing them — they directly affect the numerical results.

## License

**All rights reserved** © 2026 Lucca Paradeda — see [LICENSE](./LICENSE).

This repository is publicly visible for review and portfolio purposes only. No permission is granted to copy, modify, or redistribute the code. Students enrolled in the same course as the author may **not** reuse this work in any form as an academic deliverable.
