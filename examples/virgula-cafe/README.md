# Vírgula Café — Open Network with Returns

Three-queue open Jackson-style network with mutual feedback between two of the stations and return arcs back to the entry queue. The example is built so that none of the three queues is a trivial passthrough: each one has non-trivial load, and losses appear in two of the three.

## Network diagram

```
   external arrivals
   U(1, 3) min
   ─────────────────►  ┌────────────────────────┐
                       │  F1  —  Atendimento    │
                       │  G/G/2/10              │
                       │  service U(1, 3) min   │
                       └────┬───────────────┬───┘
                            │               │
                       0.70 │               │ 0.30
                            ▼               ▼
              ┌──────────────────┐   ┌────────────────────┐
              │  F2 — Baristas   │   │ F3 — Padaria /     │
              │  G/G/2/6         │   │ Expedição          │
              │  service U(4, 8) │   │ G/G/1/5            │
              │                  │   │ service U(2, 5)    │
              └──┬────┬────┬─────┘   └───┬───┬────┬───────┘
                 │    │    │             │   │    │
            0.10 │    │    │ 0.05   0.08 │   │    │ 0.02
                 │    │    └───► back to F1 ◄────┘
                 │    │                  │
                 │    └─► exit (residual 0.85)
                 │                       └─► exit (residual 0.90)
                 │
                 └────► F3  (combo: drink + pastry)
                                ▲
                                │  0.08
                                └────── F2

   F2 → F3 : 0.10   F3 → F2 : 0.08   (mutual feedback)
   F2 → F1 : 0.05   F3 → F1 : 0.02   (return to counter)
   F2 exit : 0.85   F3 exit : 0.90   (residual)
```

The non-tandem structure comes from three real feedback arcs:

- **F2 ↔ F3** — a barista may pass an order to the bakery for a paired pastry; the bakery may pass an order back to a barista for a paired drink.
- **F2 → F1** and **F3 → F1** — wrong-item returns: a small fraction of customers go back to the counter to fix the order.

## Per-queue specification

| Queue | Role | Kendall | Servers (c) | Capacity (K) | Service time | External arrival | Outgoing routes |
|---|---|---|---|---|---|---|---|
| **F1** | Atendimento (counter) | G/G/2/10 | 2 | 10 | U(1, 3) min | U(1, 3) min | 0.70 → F2 · 0.30 → F3 |
| **F2** | Baristas | G/G/2/6 | 2 | 6 | U(4, 8) min | — | 0.10 → F3 · 0.05 → F1 · 0.85 exit |
| **F3** | Padaria / Expedição | G/G/1/5 | 1 | 5 | U(2, 5) min | — | 0.08 → F2 · 0.02 → F1 · 0.90 exit |

Loss policy: an entity arriving at a full queue is dropped and counted in that queue's loss counter (no retry).

### Why these parameters

External arrival rate `λ = 1/2 min⁻¹` (mean inter-arrival 2 min). Solving the Jackson traffic equations

```
x₁ = λ + 0.05·x₂ + 0.02·x₃
x₂ = 0.70·x₁ + 0.08·x₃
x₃ = 0.30·x₁ + 0.10·x₂
```

gives `x₁ ≈ 0.522`, `x₂ ≈ 0.381`, `x₃ ≈ 0.195` arrivals per minute, hence

| Queue | Mean service | c | Offered ρ |
|---|---|---|---|
| F1 | 2.0 min | 2 | **0.52** |
| F2 | 6.0 min | 2 | **1.14** (saturates — losses dominate) |
| F3 | 3.5 min | 1 | **0.68** |

F2 is intentionally pushed past 1 so it back-pressures and produces the bulk of system losses; F1 and F3 stay below 1 so they remain instructive but not over-loaded.

## Running

```bash
go run . -config=examples/virgula-cafe/scenario.yml
```

or with a prebuilt binary:

```bash
./queuesim -config=examples/virgula-cafe/scenario.yml
```

## Reading the report

For each queue the engine prints:

- the **Kendall label** (`G/G/c/K`),
- the configured service and (where applicable) arrival intervals,
- the **state-time distribution** aggregated across all seeds,
- **`Number of losses`** — customers dropped at that queue due to full capacity.

Derived metrics, all computable from the printed distribution:

| Metric | Formula |
|---|---|
| Mean population **L** in queue *i* | `Σ k · P(state = k)` |
| Utilization **ρ** of a queue with c servers | `(Σ min(k, c) · P(state = k)) / c` |
| System throughput **X** | exits per minute (instrument F2 + F3 departures that route to "exit") |
| Mean response time **W** | Little's Law: `W = L_total / X` |

## Expected result (5 seeds × 100 000 RNGs)

| Queue | Measured ρ | Mass at K | Losses |
|---|---|---|---|
| F1 | ≈ 0.53 | ≈ 0 | 0 |
| F2 | ≈ 0.96 | ≈ 21 % | ≈ 9 800 |
| F3 | ≈ 0.66 | ≈ 0.6 % | ≈ 125 |

F2 sits at full capacity (state = 6) roughly a fifth of the time, which is where almost all of the system's losses come from. F3 saturates only briefly — its losses are an order of magnitude smaller. F1 always has spare seats.

## Improvement proposal

The bottleneck is F2 — it sits at offered ρ ≈ 1.14 and accounts for ~98 % of total system losses. The proposed fix is to **hire a third barista**, raising F2's server count from 2 to 3. Every other parameter (capacities, service times, routing probabilities, external arrival interval) is held constant, so the impact of the staffing change can be measured cleanly.

`scenario-improved.yml` encodes this: only `F2.servers` changes from `2` to `3`. Running it:

```bash
go run . -config=examples/virgula-cafe/scenario-improved.yml
```

### Result

| Metric | Baseline | Improved | Δ |
|---|---|---|---|
| Total system losses | 9 928 | 373 | **−96.2 %** |
| F2 utilization (ρ) | 0.988 | 0.760 | −0.23 |
| System throughput X (cust/min) | 0.4488 | 0.4980 | **+11.0 %** |
| System response time W (min) | 14.72 | 9.50 | **−35.4 %** |

See `metrics/comparison.csv` for the full per-queue breakdown and `metrics/README.md` for the formulas used.

## Files

- `scenario.yml` — baseline scenario.
- `scenario-improved.yml` — proposed improvement (F2 with 3 baristas instead of 2).
- `metrics/state-probabilities.csv` — raw state-time distributions for both runs.
- `metrics/performance-indices.csv` — computed L, ρ, X, W per queue and system-level.
- `metrics/comparison.csv` — side-by-side baseline vs. improved with deltas.
- `metrics/README.md` — formulas and methodology for the indices.
- `README.md` — this file.
