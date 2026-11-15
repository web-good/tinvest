# Бэктест MRKC (Minutes30)

- Период: 2023-11-15 23:18 — 2025-11-15 23:18
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.1 |
| MaxWaitBars | 30 |
| StochPeriod | 11 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 2 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 1 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 96 |
| Выигрышных / проигрышных | 65 / 31 |
| Win rate | 67.71% |
| Gross profit / loss | 64365.77 / 40168.48 |
| Profit factor | 1.602 |
| Чистый PnL | 24197.29 (24.20%) |
| Стартовый / финальный капитал | 100000.00 / 124197.29 |
| Макс. просадка | 12054.44 (10.25%) |
| Средняя прибыль / убыток | 990.24 / 1295.76 |
| Expectancy | 252.06 |
| Лучшая / худшая сделка | 13660.25 / -4761.08 |
| Exposure | 5.97% |
| CAGR | 11.43% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-11-20 18:00 | 0.5972 | 0.5778 | 2023-11-21 18:30 | 0.5958 | STOCH_EXIT | 20 | -333.42 | -0.33% | 0.5980 | 0.6360 | 0.0194 | M30 bullish FVG [0.5948, 0.5980], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 2 | 2023-11-22 17:30 | 0.6012 | 0.5828 | 2023-11-24 16:00 | 0.5952 | STOCH_EXIT | 35 | -1088.70 | -1.10% | 0.6044 | 0.6380 | 0.0184 | M30 bullish FVG [0.5982, 0.6044], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 3 | 2023-12-08 15:30 | 0.5580 | 0.5392 | 2023-12-08 16:30 | 0.5730 | STOCH_EXIT | 2 | 2540.47 | 2.59% | 0.5632 | 0.5956 | 0.0188 | M30 bullish FVG [0.5564, 0.5632], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 4 | 2023-12-19 17:30 | 0.5776 | 0.5577 | 2023-12-20 16:00 | 0.5810 | STOCH_EXIT | 16 | 490.80 | 0.49% | 0.5816 | 0.6173 | 0.0199 | M30 bullish FVG [0.5750, 0.5816], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 5 | 2024-01-08 17:30 | 0.5924 | 0.5773 | 2024-01-09 09:30 | 0.6018 | STOCH_EXIT | 3 | 1505.30 | 1.49% | 0.5912 | 0.6227 | 0.0151 | M30 bullish FVG [0.5892, 0.5912], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 6 | 2024-01-10 14:00 | 0.6160 | 0.5992 | 2024-01-11 16:00 | 0.6064 | STOCH_EXIT | 23 | -1705.27 | -1.66% | 0.6140 | 0.6496 | 0.0168 | M30 bullish FVG [0.6100, 0.6140], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 7 | 2024-01-18 17:00 | 0.6140 | 0.5982 | 2024-01-19 12:30 | 0.6128 | STOCH_EXIT | 10 | -299.21 | -0.30% | 0.6142 | 0.6456 | 0.0158 | M30 bullish FVG [0.6116, 0.6142], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 8 | 2024-02-01 16:30 | 0.6224 | 0.6049 | 2024-02-02 15:00 | 0.6186 | STOCH_EXIT | 16 | -716.12 | -0.71% | 0.6230 | 0.6574 | 0.0175 | M30 bullish FVG [0.6198, 0.6230], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 9 | 2024-02-27 16:00 | 0.5718 | 0.5570 | 2024-02-28 18:30 | 0.5668 | STOCH_EXIT | 24 | -974.63 | -0.97% | 0.5740 | 0.6015 | 0.0148 | M30 bullish FVG [0.5710, 0.5740], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 10 | 2024-03-07 10:30 | 0.5774 | 0.5650 | 2024-03-11 10:00 | 0.5834 | STOCH_EXIT | 18 | 932.17 | 0.94% | 0.5776 | 0.6023 | 0.0124 | M30 bullish FVG [0.5762, 0.5776], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 11 | 2024-03-11 16:00 | 0.5816 | 0.5696 | 2024-03-12 11:00 | 0.5848 | STOCH_EXIT | 9 | 450.09 | 0.45% | 0.5826 | 0.6057 | 0.0120 | M30 bullish FVG [0.5794, 0.5826], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 12 | 2024-03-13 14:30 | 0.5826 | 0.5707 | 2024-03-14 10:30 | 0.5707 | SL | 11 | -2146.81 | -2.14% | 0.5834 | 0.6064 | 0.0119 | M30 bullish FVG [0.5784, 0.5834], stochastic crossed below 20 | FVG stop 0.5707 |
| 13 | 2024-03-19 10:30 | 0.5682 | 0.5545 | 2024-03-19 13:00 | 0.5710 | STOCH_EXIT | 5 | 385.86 | 0.39% | 0.5700 | 0.5955 | 0.0137 | M30 bullish FVG [0.5664, 0.5700], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 14 | 2024-03-21 16:00 | 0.5674 | 0.5543 | 2024-03-21 17:30 | 0.5712 | STOCH_EXIT | 3 | 562.14 | 0.57% | 0.5676 | 0.5937 | 0.0131 | M30 bullish FVG [0.5650, 0.5676], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 15 | 2024-03-27 17:00 | 0.5750 | 0.5630 | 2024-03-28 09:30 | 0.5790 | STOCH_EXIT | 4 | 592.18 | 0.60% | 0.5726 | 0.5990 | 0.0120 | M30 bullish FVG [0.5704, 0.5726], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 16 | 2024-03-29 12:30 | 0.5806 | 0.5691 | 2024-03-29 18:00 | 0.5812 | STOCH_EXIT | 11 | 3.29 | 0.00% | 0.5810 | 0.6035 | 0.0115 | M30 bullish FVG [0.5790, 0.5810], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 17 | 2024-04-08 12:30 | 0.5902 | 0.5804 | 2024-04-08 14:30 | 0.5946 | STOCH_EXIT | 4 | 643.48 | 0.64% | 0.5892 | 0.6097 | 0.0098 | M30 bullish FVG [0.5878, 0.5892], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 18 | 2024-04-11 12:30 | 0.5986 | 0.5882 | 2024-04-11 16:00 | 0.6010 | STOCH_EXIT | 7 | 302.43 | 0.30% | 0.5998 | 0.6193 | 0.0104 | M30 bullish FVG [0.5982, 0.5998], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 19 | 2024-04-17 16:00 | 0.6140 | 0.6036 | 2024-04-17 18:00 | 0.6156 | STOCH_EXIT | 4 | 161.57 | 0.16% | 0.6052 | 0.6348 | 0.0104 | M30 bullish FVG [0.6030, 0.6052], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 20 | 2024-04-22 17:30 | 0.6148 | 0.6038 | 2024-04-23 09:30 | 0.6198 | STOCH_EXIT | 3 | 718.76 | 0.71% | 0.6146 | 0.6367 | 0.0110 | M30 bullish FVG [0.6124, 0.6146], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 21 | 2024-04-26 11:00 | 0.6048 | 0.5935 | 2024-04-26 18:00 | 0.6036 | STOCH_EXIT | 14 | -303.11 | -0.30% | 0.6074 | 0.6273 | 0.0113 | M30 bullish FVG [0.6028, 0.6074], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 22 | 2024-04-29 15:00 | 0.6268 | 0.6154 | 2024-04-29 18:30 | 0.6296 | STOCH_EXIT | 7 | 351.83 | 0.35% | 0.6304 | 0.6496 | 0.0114 | M30 bullish FVG [0.6262, 0.6304], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 23 | 2024-05-03 17:30 | 0.6350 | 0.6215 | 2024-05-04 10:30 | 0.6390 | STOCH_EXIT | 4 | 538.08 | 0.53% | 0.6360 | 0.6619 | 0.0135 | M30 bullish FVG [0.6300, 0.6360], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 24 | 2024-05-08 17:00 | 0.6430 | 0.6282 | 2024-05-08 17:00 | 0.6282 | SL | 0 | -2446.57 | -2.39% | 0.6484 | 0.6725 | 0.0148 | M30 bullish FVG [0.6406, 0.6484], stochastic crossed below 20 | FVG stop 0.6282 |
| 25 | 2024-05-30 16:00 | 0.5664 | 0.5471 | 2024-05-31 10:30 | 0.5471 | SL | 8 | -3490.11 | -3.50% | 0.5700 | 0.6049 | 0.0193 | M30 bullish FVG [0.5642, 0.5700], stochastic crossed below 20 | FVG stop 0.5471 |
| 26 | 2024-06-05 15:00 | 0.5702 | 0.5504 | 2024-06-06 09:30 | 0.5730 | STOCH_EXIT | 8 | 376.60 | 0.39% | 0.5728 | 0.6098 | 0.0198 | M30 bullish FVG [0.5698, 0.5728], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 27 | 2024-06-21 12:00 | 0.5790 | 0.5598 | 2024-06-21 18:00 | 0.5776 | STOCH_EXIT | 12 | -330.38 | -0.34% | 0.5762 | 0.6174 | 0.0192 | M30 bullish FVG [0.5728, 0.5762], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 28 | 2024-07-05 23:00 | 0.5016 | 0.4819 | 2024-07-06 18:00 | 0.5066 | STOCH_EXIT | 18 | 863.21 | 0.90% | 0.4992 | 0.5410 | 0.0197 | M30 bullish FVG [0.4942, 0.4992], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 29 | 2024-07-12 11:30 | 0.4510 | 0.4282 | 2024-07-13 16:30 | 0.4530 | STOCH_EXIT | 38 | 334.37 | 0.34% | 0.4368 | 0.4966 | 0.0228 | M30 bullish FVG [0.4340, 0.4368], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 30 | 2024-07-17 15:30 | 0.4360 | 0.4124 | 2024-07-18 12:00 | 0.4408 | STOCH_EXIT | 22 | 977.00 | 1.00% | 0.4332 | 0.4832 | 0.0236 | M30 bullish FVG [0.4290, 0.4332], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 31 | 2024-07-19 19:00 | 0.4626 | 0.4408 | 2024-07-20 18:00 | 0.4558 | STOCH_EXIT | 25 | -1546.21 | -1.57% | 0.4484 | 0.5063 | 0.0218 | M30 bullish FVG [0.4442, 0.4484], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 32 | 2024-08-12 20:30 | 0.4322 | 0.4171 | 2024-08-13 16:00 | 0.4364 | STOCH_EXIT | 20 | 847.28 | 0.87% | 0.4294 | 0.4624 | 0.0151 | M30 bullish FVG [0.4240, 0.4294], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 33 | 2024-09-20 16:00 | 0.3860 | 0.3740 | 2024-09-20 17:00 | 0.3906 | STOCH_EXIT | 2 | 1069.77 | 1.09% | 0.3860 | 0.4099 | 0.0120 | M30 bullish FVG [0.3822, 0.3860], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 34 | 2024-10-02 16:30 | 0.3858 | 0.3751 | 2024-10-02 19:00 | 0.3751 | SL | 5 | -2834.88 | -2.86% | 0.3846 | 0.4071 | 0.0107 | M30 bullish FVG [0.3834, 0.3846], stochastic crossed below 20 | FVG stop 0.3751 |
| 35 | 2024-10-25 16:00 | 0.3832 | 0.3759 | 2024-10-25 21:00 | 0.3916 | STOCH_EXIT | 10 | 2011.16 | 2.09% | 0.3840 | 0.3977 | 0.0073 | M30 bullish FVG [0.3792, 0.3840], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 36 | 2024-11-04 12:00 | 0.4144 | 0.4029 | 2024-11-04 20:30 | 0.4148 | STOCH_EXIT | 16 | -3.46 | -0.00% | 0.4200 | 0.4374 | 0.0115 | M30 bullish FVG [0.4140, 0.4200], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 37 | 2024-11-06 16:00 | 0.4264 | 0.4153 | 2024-11-07 09:00 | 0.4288 | STOCH_EXIT | 19 | 453.65 | 0.46% | 0.4220 | 0.4486 | 0.0111 | M30 bullish FVG [0.4194, 0.4220], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 38 | 2024-11-07 19:30 | 0.4282 | 0.4164 | 2024-11-08 07:30 | 0.4308 | STOCH_EXIT | 10 | 499.22 | 0.51% | 0.4270 | 0.4519 | 0.0118 | M30 bullish FVG [0.4256, 0.4270], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 39 | 2024-11-11 20:00 | 0.4550 | 0.4430 | 2024-11-11 21:00 | 0.4602 | STOCH_EXIT | 2 | 1033.84 | 1.04% | 0.4516 | 0.4790 | 0.0120 | M30 bullish FVG [0.4446, 0.4516], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 40 | 2024-11-12 23:00 | 0.4562 | 0.4427 | 2024-11-13 10:00 | 0.4634 | STOCH_EXIT | 8 | 1482.84 | 1.48% | 0.4588 | 0.4831 | 0.0135 | M30 bullish FVG [0.4558, 0.4588], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 41 | 2024-11-13 19:30 | 0.4712 | 0.4575 | 2024-11-14 11:00 | 0.4718 | STOCH_EXIT | 17 | 27.76 | 0.03% | 0.4724 | 0.4985 | 0.0137 | M30 bullish FVG [0.4710, 0.4724], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 42 | 2024-12-03 09:30 | 0.4494 | 0.4305 | 2024-12-03 20:00 | 0.4398 | STOCH_EXIT | 21 | -2270.08 | -2.23% | 0.4432 | 0.4871 | 0.0189 | M30 bullish FVG [0.4408, 0.4432], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 43 | 2024-12-06 09:30 | 0.4262 | 0.4074 | 2024-12-06 17:30 | 0.4396 | STOCH_EXIT | 16 | 3021.33 | 3.04% | 0.4270 | 0.4639 | 0.0188 | M30 bullish FVG [0.4250, 0.4270], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 44 | 2024-12-20 21:30 | 0.4508 | 0.4332 | 2024-12-21 07:30 | 0.4578 | STOCH_EXIT | 6 | 1485.87 | 1.45% | 0.4402 | 0.4860 | 0.0176 | M30 bullish FVG [0.4348, 0.4402], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 45 | 2024-12-24 10:30 | 0.4726 | 0.4527 | 2024-12-24 15:30 | 0.4738 | STOCH_EXIT | 10 | 159.90 | 0.15% | 0.4718 | 0.5124 | 0.0199 | M30 bullish FVG [0.4660, 0.4718], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 46 | 2025-01-08 22:30 | 0.4838 | 0.4668 | 2025-01-10 08:00 | 0.4812 | STOCH_EXIT | 38 | -662.74 | -0.64% | 0.4832 | 0.5179 | 0.0170 | M30 bullish FVG [0.4792, 0.4832], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 47 | 2025-01-13 23:00 | 0.4904 | 0.4727 | 2025-01-14 07:30 | 0.4918 | STOCH_EXIT | 3 | 191.78 | 0.19% | 0.4884 | 0.5258 | 0.0177 | M30 bullish FVG [0.4864, 0.4884], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 48 | 2025-01-17 23:30 | 0.4952 | 0.4791 | 2025-01-18 09:30 | 0.4996 | STOCH_EXIT | 6 | 815.64 | 0.79% | 0.5002 | 0.5274 | 0.0161 | M30 bullish FVG [0.4944, 0.5002], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 49 | 2025-01-31 16:30 | 0.4980 | 0.4831 | 2025-02-01 06:00 | 0.5010 | STOCH_EXIT | 19 | 525.11 | 0.50% | 0.4990 | 0.5278 | 0.0149 | M30 bullish FVG [0.4962, 0.4990], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 50 | 2025-02-03 20:30 | 0.5188 | 0.5044 | 2025-02-04 10:00 | 0.5288 | STOCH_EXIT | 13 | 1914.19 | 1.83% | 0.5180 | 0.5475 | 0.0144 | M30 bullish FVG [0.5150, 0.5180], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 51 | 2025-02-12 15:00 | 0.5322 | 0.5180 | 2025-02-12 20:00 | 0.5350 | STOCH_EXIT | 10 | 455.55 | 0.43% | 0.5320 | 0.5605 | 0.0142 | M30 bullish FVG [0.5298, 0.5320], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 52 | 2025-02-13 13:30 | 0.5424 | 0.5287 | 2025-02-13 18:30 | 0.5482 | STOCH_EXIT | 10 | 1040.43 | 0.97% | 0.5512 | 0.5699 | 0.0137 | M30 bullish FVG [0.5420, 0.5512], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 53 | 2025-02-17 16:00 | 0.5616 | 0.5464 | 2025-02-17 18:30 | 0.5628 | STOCH_EXIT | 5 | 123.10 | 0.11% | 0.5604 | 0.5920 | 0.0152 | M30 bullish FVG [0.5524, 0.5604], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 54 | 2025-03-05 07:30 | 0.6426 | 0.6185 | 2025-03-05 11:30 | 0.6480 | STOCH_EXIT | 8 | 803.54 | 0.74% | 0.6406 | 0.6907 | 0.0241 | M30 bullish FVG [0.6352, 0.6406], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 55 | 2025-03-05 20:00 | 0.6564 | 0.6323 | 2025-03-06 09:30 | 0.6564 | STOCH_EXIT | 13 | -108.96 | -0.10% | 0.6516 | 0.7045 | 0.0241 | M30 bullish FVG [0.6480, 0.6516], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 56 | 2025-03-07 16:00 | 0.6484 | 0.6240 | 2025-03-07 21:30 | 0.6454 | STOCH_EXIT | 11 | -612.68 | -0.56% | 0.6520 | 0.6971 | 0.0244 | M30 bullish FVG [0.6450, 0.6520], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 57 | 2025-03-11 23:00 | 0.6450 | 0.6217 | 2025-03-12 21:30 | 0.6478 | STOCH_EXIT | 30 | 361.80 | 0.33% | 0.6450 | 0.6916 | 0.0233 | M30 bullish FVG [0.6410, 0.6450], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 58 | 2025-03-17 17:00 | 0.6616 | 0.6386 | 2025-03-17 19:00 | 0.6648 | STOCH_EXIT | 4 | 416.04 | 0.38% | 0.6632 | 0.7075 | 0.0230 | M30 bullish FVG [0.6596, 0.6632], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 59 | 2025-03-24 16:00 | 0.6302 | 0.6089 | 2025-03-24 18:00 | 0.6358 | STOCH_EXIT | 4 | 859.29 | 0.79% | 0.6336 | 0.6729 | 0.0213 | M30 bullish FVG [0.6268, 0.6336], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 60 | 2025-04-03 10:30 | 0.5626 | 0.5410 | 2025-04-03 23:30 | 0.5498 | STOCH_EXIT | 26 | -2617.82 | -2.37% | 0.5612 | 0.6057 | 0.0216 | M30 bullish FVG [0.5586, 0.5612], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 61 | 2025-04-08 20:00 | 0.5352 | 0.5120 | 2025-04-09 07:00 | 0.5120 | SL | 8 | -4761.08 | -4.42% | 0.5336 | 0.5815 | 0.0232 | M30 bullish FVG [0.5306, 0.5336], stochastic crossed below 20 | FVG stop 0.5120 |
| 62 | 2025-04-10 11:30 | 0.5436 | 0.5206 | 2025-04-10 15:30 | 0.5538 | STOCH_EXIT | 8 | 1824.10 | 1.77% | 0.5378 | 0.5897 | 0.0230 | M30 bullish FVG [0.5350, 0.5378], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 63 | 2025-04-11 23:30 | 0.5534 | 0.5286 | 2025-04-12 11:30 | 0.5662 | STOCH_EXIT | 15 | 2313.40 | 2.21% | 0.5540 | 0.6030 | 0.0248 | M30 bullish FVG [0.5488, 0.5540], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 64 | 2025-04-14 09:30 | 0.5642 | 0.5405 | 2025-04-15 09:30 | 0.5530 | STOCH_EXIT | 34 | -2222.38 | -2.08% | 0.5664 | 0.6116 | 0.0237 | M30 bullish FVG [0.5638, 0.5664], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 65 | 2025-04-17 20:30 | 0.5728 | 0.5514 | 2025-04-17 20:30 | 0.5774 | STOCH_EXIT | 0 | 736.56 | 0.70% | 0.5678 | 0.6155 | 0.0214 | M30 bullish FVG [0.5644, 0.5678], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 66 | 2025-04-21 18:30 | 0.5780 | 0.5577 | 2025-04-21 22:00 | 0.5840 | STOCH_EXIT | 7 | 986.26 | 0.94% | 0.5788 | 0.6185 | 0.0203 | M30 bullish FVG [0.5766, 0.5788], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 67 | 2025-05-05 16:00 | 0.5306 | 0.5120 | 2025-05-05 22:30 | 0.5416 | STOCH_EXIT | 13 | 2103.24 | 1.97% | 0.5300 | 0.5677 | 0.0186 | M30 bullish FVG [0.5252, 0.5300], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 68 | 2025-05-20 09:30 | 0.5964 | 0.5781 | 2025-05-20 12:30 | 0.5984 | STOCH_EXIT | 6 | 255.27 | 0.24% | 0.5966 | 0.6329 | 0.0183 | M30 bullish FVG [0.5946, 0.5966], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 69 | 2025-05-21 10:30 | 0.6088 | 0.5903 | 2025-05-22 07:00 | 0.6076 | STOCH_EXIT | 27 | -323.67 | -0.30% | 0.6116 | 0.6457 | 0.0185 | M30 bullish FVG [0.6082, 0.6116], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 70 | 2025-05-27 07:30 | 0.6000 | 0.5810 | 2025-05-27 09:00 | 0.6036 | STOCH_EXIT | 3 | 542.67 | 0.50% | 0.6012 | 0.6380 | 0.0190 | M30 bullish FVG [0.5988, 0.6012], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 71 | 2025-05-27 23:00 | 0.6278 | 0.6088 | 2025-05-28 16:30 | 0.6328 | STOCH_EXIT | 22 | 760.33 | 0.70% | 0.6240 | 0.6658 | 0.0190 | M30 bullish FVG [0.6206, 0.6240], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 72 | 2025-06-09 13:30 | 0.6610 | 0.6437 | 2025-06-09 20:30 | 0.6614 | STOCH_EXIT | 14 | -43.36 | -0.04% | 0.6650 | 0.6956 | 0.0173 | M30 bullish FVG [0.6580, 0.6650], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 73 | 2025-06-13 17:30 | 0.6826 | 0.6646 | 2025-06-14 18:30 | 0.6890 | STOCH_EXIT | 43 | 919.99 | 0.84% | 0.6832 | 0.7185 | 0.0180 | M30 bullish FVG [0.6796, 0.6832], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 74 | 2025-06-20 08:30 | 0.6752 | 0.6563 | 2025-06-20 17:00 | 0.6790 | STOCH_EXIT | 17 | 512.16 | 0.46% | 0.6770 | 0.7130 | 0.0189 | M30 bullish FVG [0.6750, 0.6770], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 75 | 2025-06-26 07:00 | 0.5852 | 0.5586 | 2025-06-26 07:30 | 0.5948 | STOCH_EXIT | 1 | 1711.90 | 1.54% | 0.5790 | 0.6384 | 0.0266 | M30 bullish FVG [0.5752, 0.5790], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 76 | 2025-07-03 10:00 | 0.6204 | 0.5962 | 2025-07-03 15:00 | 0.6200 | STOCH_EXIT | 10 | -185.68 | -0.16% | 0.6174 | 0.6689 | 0.0242 | M30 bullish FVG [0.6120, 0.6174], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 77 | 2025-07-11 09:30 | 0.6214 | 0.6000 | 2025-07-11 19:00 | 0.6184 | STOCH_EXIT | 19 | -655.20 | -0.58% | 0.6220 | 0.6643 | 0.0214 | M30 bullish FVG [0.6136, 0.6220], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 78 | 2025-07-17 17:00 | 0.6466 | 0.6271 | 2025-07-17 23:30 | 0.6490 | STOCH_EXIT | 13 | 303.13 | 0.27% | 0.6422 | 0.6857 | 0.0195 | M30 bullish FVG [0.6300, 0.6422], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 79 | 2025-07-21 17:00 | 0.6736 | 0.6521 | 2025-07-22 07:00 | 0.6732 | STOCH_EXIT | 14 | -179.26 | -0.16% | 0.6756 | 0.7165 | 0.0215 | M30 bullish FVG [0.6732, 0.6756], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 80 | 2025-07-28 19:00 | 0.7066 | 0.6889 | 2025-07-28 22:00 | 0.7106 | STOCH_EXIT | 6 | 523.33 | 0.47% | 0.7088 | 0.7421 | 0.0177 | M30 bullish FVG [0.7010, 0.7088], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 81 | 2025-07-30 20:00 | 0.6888 | 0.6666 | 2025-07-31 09:00 | 0.6902 | STOCH_EXIT | 13 | 115.81 | 0.10% | 0.6940 | 0.7333 | 0.0222 | M30 bullish FVG [0.6856, 0.6940], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 82 | 2025-07-31 16:00 | 0.6890 | 0.6665 | 2025-07-31 17:30 | 0.6920 | STOCH_EXIT | 3 | 378.76 | 0.34% | 0.6896 | 0.7340 | 0.0225 | M30 bullish FVG [0.6866, 0.6896], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 83 | 2025-08-05 11:00 | 0.7398 | 0.7155 | 2025-08-05 16:30 | 0.7456 | STOCH_EXIT | 11 | 773.77 | 0.68% | 0.7388 | 0.7884 | 0.0243 | M30 bullish FVG [0.7288, 0.7388], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 84 | 2025-08-06 12:30 | 0.7602 | 0.7354 | 2025-08-06 20:30 | 0.7572 | STOCH_EXIT | 16 | -563.80 | -0.49% | 0.7498 | 0.8099 | 0.0248 | M30 bullish FVG [0.7464, 0.7498], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 85 | 2025-09-01 18:30 | 0.7356 | 0.7143 | 2025-09-01 18:30 | 0.7384 | STOCH_EXIT | 0 | 317.70 | 0.28% | 0.7330 | 0.7782 | 0.0213 | M30 bullish FVG [0.7298, 0.7330], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 86 | 2025-09-23 11:30 | 0.7180 | 0.6981 | 2025-09-23 12:00 | 0.7228 | STOCH_EXIT | 1 | 644.58 | 0.57% | 0.7058 | 0.7578 | 0.0199 | M30 bullish FVG [0.7020, 0.7058], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 87 | 2025-09-25 17:30 | 0.7684 | 0.7441 | 2025-09-25 20:00 | 0.7724 | STOCH_EXIT | 5 | 481.21 | 0.42% | 0.7712 | 0.8170 | 0.0243 | M30 bullish FVG [0.7670, 0.7712], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 88 | 2025-09-29 13:30 | 0.7860 | 0.7616 | 2025-09-29 23:30 | 0.7914 | STOCH_EXIT | 20 | 673.25 | 0.59% | 0.7900 | 0.8348 | 0.0244 | M30 bullish FVG [0.7838, 0.7900], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 89 | 2025-10-01 15:30 | 0.8102 | 0.7837 | 2025-10-02 12:30 | 0.8016 | STOCH_EXIT | 29 | -1335.64 | -1.16% | 0.8022 | 0.8633 | 0.0265 | M30 bullish FVG [0.7976, 0.8022], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 90 | 2025-10-07 22:00 | 0.8292 | 0.8017 | 2025-10-08 07:30 | 0.8320 | STOCH_EXIT | 6 | 269.81 | 0.24% | 0.8354 | 0.8842 | 0.0275 | M30 bullish FVG [0.8270, 0.8354], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 91 | 2025-10-09 23:30 | 0.8552 | 0.8272 | 2025-10-10 11:00 | 0.8272 | SL | 10 | -3869.57 | -3.37% | 0.8612 | 0.9113 | 0.0280 | M30 bullish FVG [0.8424, 0.8612], stochastic crossed below 20 | FVG stop 0.8272 |
| 92 | 2025-10-14 09:30 | 0.7118 | 0.6674 | 2025-10-15 13:00 | 0.8007 | TP | 41 | 13660.25 | 12.38% | 0.7052 | 0.8007 | 0.0444 | M30 bullish FVG [0.7008, 0.7052], stochastic crossed below 20 | FVG target 0.8007 |
| 93 | 2025-10-16 10:30 | 0.7774 | 0.7263 | 2025-10-16 16:00 | 0.7830 | STOCH_EXIT | 11 | 771.17 | 0.62% | 0.7780 | 0.8797 | 0.0511 | M30 bullish FVG [0.7642, 0.7780], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 94 | 2025-10-22 14:00 | 0.7686 | 0.7222 | 2025-10-22 19:00 | 0.7722 | STOCH_EXIT | 10 | 458.40 | 0.37% | 0.7692 | 0.8614 | 0.0464 | M30 bullish FVG [0.7632, 0.7692], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 95 | 2025-10-28 14:00 | 0.7182 | 0.6751 | 2025-10-28 20:00 | 0.7146 | STOCH_EXIT | 12 | -751.05 | -0.60% | 0.7168 | 0.8044 | 0.0431 | M30 bullish FVG [0.7074, 0.7168], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 96 | 2025-11-10 13:00 | 0.6770 | 0.6435 | 2025-11-10 21:00 | 0.6734 | STOCH_EXIT | 16 | -786.64 | -0.63% | 0.6786 | 0.7439 | 0.0335 | M30 bullish FVG [0.6738, 0.6786], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 96288.33
- Макс: 126577.30
- Финал: 124197.29

Полная кривая — в `*_equity.csv`.
