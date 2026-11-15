# LSRG candidate locked before final test (2026-11-13)

Chosen candidate: {"ATRPeriod":14,"MinGapATR":0.3,"MaxWaitBars":30,"StochPeriod":9,"UseStochExit":1,"StopATR":2,"TakeProfitATR":1,"HTFTrendEMA":0,"RequireEntryAboveFVGLower":1}.

Selection used only the 2023-11-13 to 2025-11-13 training window. Stage 3 rank 1 trained at 2026-11-13 23:02: 21 closed trades, 11 in [2023-11-13, 2024-11-13), 10 in [2024-11-13, 2025-11-13), expectancy 542.20, net PnL 11,386.21 after 0.0005 commission, PF 2.348, max drawdown 6.23%. Training year trade counts are entry-date counts from the single 24-month run, not independent annual replays. No positions were open at report end.

The Stage 2 no-stochastic top had max drawdown 12.70% and failed the precommitted 10% ceiling; the next observed candidate with 10+ trades in each year had materially lower expectancy (407.1 vs 542.2), outside the precommitted 5% near-tie band. Stage 3 local joint refinement kept the same values; neighboring eligible candidates were lower in expectancy. Sensitivity results for StochPeriod and HTFTrendEMA did not meet the 20-trade total floor. MaxWaitBars=15/30/45 sensitivity retained the same candidate at 30. The separate input candidate with RequireEntryAboveFVGLower=0 had lower stage-3 expectancy (522.9) and was not chosen.

Final reserved window is [2025-11-13, 2026-11-13). Candidate identity and selection are fixed; do not revisit selection based on final-window results. Validation requires complete window, >=10 closed trades, positive net PnL, PF > 1, and max DD <=10%. If cached/API history cannot cover the full window, do not call it independent validation and do not change the ticker literal.
