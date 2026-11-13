# Stage 1 review

Training window in reports: 2023-11-12 13:27/13:29 through 2025-11-12 13:27/13:29 MSK. The original control exactly meets the year split: 22 / 10 closed entries. Four candidates passed the explicit entry-date split, quality gates (positive Net PnL, PF > 1, Max DD <= 10%), and the 20 / 10 / 10 trade gates:

| Rank | Candidate file | 2023-11-12..2024-11-12 | 2024-11-12..2025-11-12 | Net PnL | PF | Max DD |
|---:|---|---:|---:|---:|---:|---:|
| 1 | `stage-1-candidate-1.json` | 22 | 10 | 15191.99 | 2.285 | 4.85% |
| 7 | `stage-1-candidate-2.json` | 25 | 13 | 10608.10 | 2.317 | 4.68% |
| 9 | `stage-1-candidate-3.json` | 40 | 17 | 14963.91 | 2.075 | 5.54% |
| 14 | `stage-1-candidate-4.json` | 20 | 11 | 6926.62 | 1.851 | 5.29% |

The two close stochastic-period neighbors at ranks 7 and 8 were both profitable and passed the year counts (rank 7: 25 / 13; rank 8: 22 / 12); the more filtered EMA200 / entry-floor regime also passed in multiple nearby grid points. All 350 combinations were retained in the report. The shortlist spans the original entry, nearby stochastic periods, and filtered regimes; this does not establish a global optimum.
