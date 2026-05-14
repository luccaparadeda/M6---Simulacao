# Vírgula Café — Performance Indices

This folder contains the spreadsheet artefacts required by the assignment: the raw state-time distributions emitted by the simulator and the derived performance indices (L, ρ, X, W) for both the baseline and the improved scenario.

## Files

| File | Contents |
|---|---|
| `state-probabilities.csv` | Per-state time accumulated and probability for every (scenario, queue, state) triple. Direct copy of what the simulator prints. |
| `performance-indices.csv` | Computed L, ρ, X, W per queue and for the system as a whole, for both scenarios. |
| `comparison.csv` | Side-by-side baseline vs. improved with absolute and percent delta — ready to paste into a comparison slide. |

## How each index is computed

Notation: `P(k)` is the probability the queue is at population `k` (from `state-probabilities.csv`). `c` is the number of servers, `E[S]` is the mean service time, `T` is the total simulated time across all seeds.

| Index | Formula | Notes |
|---|---|---|
| **L** (mean population) | `Σ k · P(k)` | Pure function of the state distribution. |
| **ρ** (utilization) | `(Σ min(k, c) · P(k)) / c` | Fraction of busy servers averaged over time. Equivalent to `1 − P(0)` only when `c = 1`. |
| **X** (throughput, completions/min) | `ρ · c / E[S]` | Each busy server completes `1/E[S]` customers per unit time. |
| **W** (mean response time per visit, min) | `L / X` | Little's Law applied at the queue level. |

### System-level metrics

- **L_total** = `Σ L_i` across all queues.
- **X_system** = customers that permanently exit the network per minute, estimated as
  ```
  X_system ≈ (external_arrivals − total_losses) / T
  ```
  where `external_arrivals ≈ T / E[interarrival]` for the entry queue (`E[interarrival] = 2` min for both scenarios). This is the *system throughput*, not a per-queue rate — it counts each customer once, regardless of how many queues they passed through (a single customer may visit F2 or F3 multiple times via the feedback arcs).
- **W_system** = `L_total / X_system` — mean time a customer spends in the network from arrival to permanent exit.

### Why X is approximate

The simulator does not directly count "exits from the system" because the original engine has no exit counter; it only tracks per-queue state-time and losses. The estimate above is exact in steady state under two assumptions: (a) total external arrivals over `T` equals `T / E[interarrival]` (true on average for uniform inter-arrivals; the small RNG-budget cutoff effect is negligible for `rngPerSeed = 100 000`), and (b) every customer either exits the system or is lost at some queue — no leftovers in queue at end-of-run. Sanity check across both scenarios: total external attempts ≈ admitted + lost within 0.1 %.

A cross-check is available: the per-queue `X_F2 = ρ · c / E[S]` should equal `(offered arrivals to F2) − (F2 losses / T)`. For the baseline, `X_F2 = 0.329`, F2-loss rate = `9803 / 194001 = 0.0505`, sum = `0.380` — matches the analytical Jackson flow `x_2 = 0.381` to three decimals. Same agreement holds for F1 and F3.

## Headline result

| | Baseline (F2: c=2) | Improved (F2: c=3) | Δ |
|---|---|---|---|
| Total system losses | **9 928** | **373** | **−96.2 %** |
| System throughput (cust/min) | 0.4488 | 0.4980 | **+11.0 %** |
| System response time (min) | 14.72 | 9.50 | **−35.4 %** |
| F2 utilization | 0.988 | 0.760 | F2 stops being saturated |
| F2 response time per visit (min) | 13.50 | 6.69 | **−50.5 %** |

The single-parameter change (`F2.servers: 2 → 3`) removes the bottleneck: F2 drops from ρ ≈ 1.0 to ρ ≈ 0.76, total system losses fall by 96 %, and per-customer response time is cut by more than a third.
