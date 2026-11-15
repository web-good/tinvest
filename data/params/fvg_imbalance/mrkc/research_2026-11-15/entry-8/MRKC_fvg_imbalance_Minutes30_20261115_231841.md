# Бэктест MRKC (Minutes30)

- Период: 2023-11-15 23:18 — 2025-11-15 23:18
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0 |
| MaxWaitBars | 30 |
| StochPeriod | 11 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 2 |
| HTFTrendEMA | 120 |
| RequireEntryAboveFVGLower | 1 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 133 |
| Выигрышных / проигрышных | 96 / 37 |
| Win rate | 72.18% |
| Gross profit / loss | 78577.42 / 44592.85 |
| Profit factor | 1.762 |
| Чистый PnL | 33984.57 (33.98%) |
| Стартовый / финальный капитал | 100000.00 / 133984.57 |
| Макс. просадка | 13373.19 (10.56%) |
| Средняя прибыль / убыток | 818.51 / 1205.21 |
| Expectancy | 255.52 |
| Лучшая / худшая сделка | 14717.82 / -4129.47 |
| Exposure | 7.41% |
| CAGR | 15.73% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-11-20 11:30 | 0.5898 | 0.5704 | 2023-11-20 11:30 | 0.5942 | STOCH_EXIT | 0 | 643.55 | 0.65% | 0.5886 | 0.6286 | 0.0194 | M30 bullish FVG [0.5874, 0.5886], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 2 | 2023-11-20 18:00 | 0.5972 | 0.5778 | 2023-11-21 18:30 | 0.5958 | STOCH_EXIT | 20 | -335.41 | -0.33% | 0.5980 | 0.6360 | 0.0194 | M30 bullish FVG [0.5948, 0.5980], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 3 | 2023-11-22 17:30 | 0.6012 | 0.5828 | 2023-11-24 16:00 | 0.5952 | STOCH_EXIT | 35 | -1095.30 | -1.10% | 0.6044 | 0.6380 | 0.0184 | M30 bullish FVG [0.5982, 0.6044], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 4 | 2023-11-24 17:30 | 0.5918 | 0.5742 | 2023-11-24 18:00 | 0.5954 | STOCH_EXIT | 1 | 502.07 | 0.51% | 0.5912 | 0.6270 | 0.0176 | M30 bullish FVG [0.5910, 0.5912], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 5 | 2023-12-08 15:30 | 0.5580 | 0.5392 | 2023-12-08 16:30 | 0.5730 | STOCH_EXIT | 2 | 2569.34 | 2.59% | 0.5632 | 0.5956 | 0.0188 | M30 bullish FVG [0.5564, 0.5632], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 6 | 2023-12-13 16:00 | 0.5400 | 0.5203 | 2023-12-14 09:30 | 0.5498 | STOCH_EXIT | 6 | 1749.21 | 1.71% | 0.5414 | 0.5794 | 0.0197 | M30 bullish FVG [0.5396, 0.5414], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 7 | 2023-12-19 17:30 | 0.5776 | 0.5577 | 2023-12-20 16:00 | 0.5810 | STOCH_EXIT | 16 | 507.73 | 0.49% | 0.5816 | 0.6173 | 0.0199 | M30 bullish FVG [0.5750, 0.5816], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 8 | 2023-12-27 16:30 | 0.5638 | 0.5454 | 2023-12-28 09:30 | 0.5674 | STOCH_EXIT | 5 | 561.36 | 0.54% | 0.5640 | 0.6007 | 0.0184 | M30 bullish FVG [0.5636, 0.5640], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 9 | 2024-01-03 18:00 | 0.5696 | 0.5531 | 2024-01-04 09:30 | 0.5706 | STOCH_EXIT | 2 | 79.10 | 0.08% | 0.5700 | 0.6026 | 0.0165 | M30 bullish FVG [0.5688, 0.5700], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 10 | 2024-01-08 17:30 | 0.5924 | 0.5773 | 2024-01-09 09:30 | 0.6018 | STOCH_EXIT | 3 | 1558.11 | 1.49% | 0.5912 | 0.6227 | 0.0151 | M30 bullish FVG [0.5892, 0.5912], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 11 | 2024-01-10 14:00 | 0.6160 | 0.5992 | 2024-01-11 16:00 | 0.6064 | STOCH_EXIT | 23 | -1766.54 | -1.66% | 0.6140 | 0.6496 | 0.0168 | M30 bullish FVG [0.6100, 0.6140], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 12 | 2024-01-17 16:30 | 0.6140 | 0.5980 | 2024-01-18 11:00 | 0.6178 | STOCH_EXIT | 8 | 541.30 | 0.52% | 0.6132 | 0.6461 | 0.0160 | M30 bullish FVG [0.6120, 0.6132], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 13 | 2024-01-18 17:00 | 0.6140 | 0.5982 | 2024-01-19 12:30 | 0.6128 | STOCH_EXIT | 10 | -310.09 | -0.30% | 0.6142 | 0.6456 | 0.0158 | M30 bullish FVG [0.6116, 0.6142], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 14 | 2024-01-29 18:30 | 0.6242 | 0.6058 | 2024-01-30 09:30 | 0.6290 | STOCH_EXIT | 1 | 701.13 | 0.67% | 0.6220 | 0.6611 | 0.0184 | M30 bullish FVG [0.6206, 0.6220], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 15 | 2024-02-01 16:30 | 0.6224 | 0.6049 | 2024-02-02 15:00 | 0.6186 | STOCH_EXIT | 16 | -751.48 | -0.71% | 0.6230 | 0.6574 | 0.0175 | M30 bullish FVG [0.6198, 0.6230], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 16 | 2024-02-06 10:00 | 0.6128 | 0.5969 | 2024-02-06 10:00 | 0.6144 | STOCH_EXIT | 0 | 168.67 | 0.16% | 0.6132 | 0.6446 | 0.0159 | M30 bullish FVG [0.6126, 0.6132], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 17 | 2024-02-07 15:00 | 0.6126 | 0.5971 | 2024-02-08 17:00 | 0.5971 | SL | 23 | -2746.05 | -2.62% | 0.6122 | 0.6435 | 0.0155 | M30 bullish FVG [0.6120, 0.6122], stochastic crossed below 20 | FVG stop 0.5971 |
| 18 | 2024-02-14 13:30 | 0.5956 | 0.5813 | 2024-02-15 10:00 | 0.5813 | SL | 12 | -2561.67 | -2.50% | 0.5910 | 0.6242 | 0.0143 | M30 bullish FVG [0.5904, 0.5910], stochastic crossed below 20 | FVG stop 0.5813 |
| 19 | 2024-02-27 16:00 | 0.5718 | 0.5570 | 2024-02-28 18:30 | 0.5668 | STOCH_EXIT | 24 | -969.06 | -0.97% | 0.5740 | 0.6015 | 0.0148 | M30 bullish FVG [0.5710, 0.5740], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 20 | 2024-03-07 10:30 | 0.5774 | 0.5650 | 2024-03-11 10:00 | 0.5834 | STOCH_EXIT | 18 | 926.75 | 0.94% | 0.5776 | 0.6023 | 0.0124 | M30 bullish FVG [0.5762, 0.5776], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 21 | 2024-03-11 16:00 | 0.5816 | 0.5696 | 2024-03-12 11:00 | 0.5848 | STOCH_EXIT | 9 | 447.47 | 0.45% | 0.5826 | 0.6057 | 0.0120 | M30 bullish FVG [0.5794, 0.5826], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 22 | 2024-03-13 14:30 | 0.5826 | 0.5707 | 2024-03-14 10:30 | 0.5707 | SL | 11 | -2146.81 | -2.14% | 0.5822 | 0.6064 | 0.0119 | M30 bullish FVG [0.5818, 0.5822], stochastic crossed below 20 | FVG stop 0.5707 |
| 23 | 2024-03-14 17:30 | 0.5706 | 0.5592 | 2024-03-15 10:30 | 0.5748 | STOCH_EXIT | 5 | 623.90 | 0.64% | 0.5710 | 0.5934 | 0.0114 | M30 bullish FVG [0.5704, 0.5710], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 24 | 2024-03-19 10:30 | 0.5682 | 0.5545 | 2024-03-19 13:00 | 0.5710 | STOCH_EXIT | 5 | 385.86 | 0.39% | 0.5700 | 0.5955 | 0.0137 | M30 bullish FVG [0.5664, 0.5700], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 25 | 2024-03-27 17:00 | 0.5750 | 0.5630 | 2024-03-28 09:30 | 0.5790 | STOCH_EXIT | 4 | 588.76 | 0.60% | 0.5746 | 0.5990 | 0.0120 | M30 bullish FVG [0.5740, 0.5746], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 26 | 2024-03-29 12:30 | 0.5806 | 0.5691 | 2024-03-29 18:00 | 0.5812 | STOCH_EXIT | 11 | 3.27 | 0.00% | 0.5810 | 0.6035 | 0.0115 | M30 bullish FVG [0.5790, 0.5810], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 27 | 2024-04-03 10:30 | 0.5846 | 0.5742 | 2024-04-03 16:00 | 0.5858 | STOCH_EXIT | 11 | 104.52 | 0.11% | 0.5836 | 0.6054 | 0.0104 | M30 bullish FVG [0.5834, 0.5836], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 28 | 2024-04-08 12:30 | 0.5902 | 0.5804 | 2024-04-08 14:30 | 0.5946 | STOCH_EXIT | 4 | 643.48 | 0.64% | 0.5892 | 0.6097 | 0.0098 | M30 bullish FVG [0.5878, 0.5892], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 29 | 2024-04-11 12:30 | 0.5986 | 0.5882 | 2024-04-11 16:00 | 0.6010 | STOCH_EXIT | 7 | 302.43 | 0.30% | 0.5998 | 0.6193 | 0.0104 | M30 bullish FVG [0.5982, 0.5998], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 30 | 2024-04-17 16:00 | 0.6140 | 0.6036 | 2024-04-17 18:00 | 0.6156 | STOCH_EXIT | 4 | 161.57 | 0.16% | 0.6106 | 0.6348 | 0.0104 | M30 bullish FVG [0.6098, 0.6106], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 31 | 2024-04-22 17:30 | 0.6148 | 0.6038 | 2024-04-23 09:30 | 0.6198 | STOCH_EXIT | 3 | 718.76 | 0.71% | 0.6146 | 0.6367 | 0.0110 | M30 bullish FVG [0.6124, 0.6146], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 32 | 2024-04-26 11:00 | 0.6048 | 0.5935 | 2024-04-26 18:00 | 0.6036 | STOCH_EXIT | 14 | -303.11 | -0.30% | 0.6074 | 0.6273 | 0.0113 | M30 bullish FVG [0.6028, 0.6074], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 33 | 2024-04-29 15:00 | 0.6268 | 0.6154 | 2024-04-29 18:30 | 0.6296 | STOCH_EXIT | 7 | 349.66 | 0.35% | 0.6304 | 0.6496 | 0.0114 | M30 bullish FVG [0.6262, 0.6304], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 34 | 2024-05-03 17:30 | 0.6350 | 0.6215 | 2024-05-04 10:30 | 0.6390 | STOCH_EXIT | 4 | 538.08 | 0.53% | 0.6360 | 0.6619 | 0.0135 | M30 bullish FVG [0.6300, 0.6360], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 35 | 2024-05-08 17:00 | 0.6430 | 0.6282 | 2024-05-08 17:00 | 0.6282 | SL | 0 | -2446.57 | -2.39% | 0.6484 | 0.6725 | 0.0148 | M30 bullish FVG [0.6406, 0.6484], stochastic crossed below 20 | FVG stop 0.6282 |
| 36 | 2024-05-13 13:30 | 0.6440 | 0.6268 | 2024-05-14 09:30 | 0.6466 | STOCH_EXIT | 11 | 302.98 | 0.30% | 0.6420 | 0.6784 | 0.0172 | M30 bullish FVG [0.6418, 0.6420], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 37 | 2024-05-23 12:00 | 0.6018 | 0.5831 | 2024-05-24 14:00 | 0.5831 | SL | 23 | -3198.66 | -3.20% | 0.5996 | 0.6392 | 0.0187 | M30 bullish FVG [0.5992, 0.5996], stochastic crossed below 20 | FVG stop 0.5831 |
| 38 | 2024-11-12 23:00 | 0.4562 | 0.4427 | 2024-11-13 10:00 | 0.4634 | STOCH_EXIT | 8 | 1428.92 | 1.48% | 0.4588 | 0.4831 | 0.0135 | M30 bullish FVG [0.4558, 0.4588], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 39 | 2024-11-13 19:30 | 0.4712 | 0.4575 | 2024-11-14 11:00 | 0.4718 | STOCH_EXIT | 17 | 26.73 | 0.03% | 0.4724 | 0.4985 | 0.0137 | M30 bullish FVG [0.4710, 0.4724], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 40 | 2024-11-25 10:30 | 0.4540 | 0.4365 | 2024-11-25 12:00 | 0.4564 | STOCH_EXIT | 3 | 420.08 | 0.43% | 0.4542 | 0.4889 | 0.0175 | M30 bullish FVG [0.4534, 0.4542], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 41 | 2024-11-25 17:00 | 0.4518 | 0.4343 | 2024-11-25 19:00 | 0.4518 | STOCH_EXIT | 4 | -98.49 | -0.10% | 0.4520 | 0.4867 | 0.0175 | M30 bullish FVG [0.4506, 0.4520], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 42 | 2024-12-03 09:30 | 0.4494 | 0.4305 | 2024-12-03 20:00 | 0.4398 | STOCH_EXIT | 21 | -2199.77 | -2.23% | 0.4496 | 0.4871 | 0.0189 | M30 bullish FVG [0.4480, 0.4496], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 43 | 2024-12-24 10:30 | 0.4726 | 0.4527 | 2024-12-24 15:30 | 0.4738 | STOCH_EXIT | 10 | 148.27 | 0.15% | 0.4718 | 0.5124 | 0.0199 | M30 bullish FVG [0.4660, 0.4718], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 44 | 2024-12-30 16:30 | 0.4792 | 0.4609 | 2024-12-31 12:00 | 0.4820 | STOCH_EXIT | 25 | 466.20 | 0.48% | 0.4752 | 0.5158 | 0.0183 | M30 bullish FVG [0.4746, 0.4752], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 45 | 2025-01-07 08:00 | 0.4688 | 0.4518 | 2025-01-07 12:30 | 0.4694 | STOCH_EXIT | 9 | 27.10 | 0.03% | 0.4674 | 0.5029 | 0.0170 | M30 bullish FVG [0.4670, 0.4674], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 46 | 2025-01-08 22:30 | 0.4838 | 0.4668 | 2025-01-10 08:00 | 0.4812 | STOCH_EXIT | 38 | -616.50 | -0.64% | 0.4832 | 0.5179 | 0.0170 | M30 bullish FVG [0.4792, 0.4832], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 47 | 2025-01-10 13:00 | 0.4784 | 0.4602 | 2025-01-10 14:00 | 0.4834 | STOCH_EXIT | 2 | 908.34 | 0.94% | 0.4754 | 0.5148 | 0.0182 | M30 bullish FVG [0.4744, 0.4754], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 48 | 2025-01-10 20:30 | 0.4838 | 0.4656 | 2025-01-11 08:00 | 0.4894 | STOCH_EXIT | 9 | 1027.79 | 1.06% | 0.4810 | 0.5202 | 0.0182 | M30 bullish FVG [0.4808, 0.4810], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 49 | 2025-01-13 23:00 | 0.4904 | 0.4727 | 2025-01-14 07:30 | 0.4918 | STOCH_EXIT | 3 | 181.78 | 0.19% | 0.4894 | 0.5258 | 0.0177 | M30 bullish FVG [0.4892, 0.4894], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 50 | 2025-01-16 16:30 | 0.4932 | 0.4772 | 2025-01-17 07:00 | 0.4982 | STOCH_EXIT | 15 | 900.86 | 0.91% | 0.4870 | 0.5251 | 0.0160 | M30 bullish FVG [0.4868, 0.4870], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 51 | 2025-01-17 23:30 | 0.4952 | 0.4791 | 2025-01-18 09:30 | 0.4996 | STOCH_EXIT | 6 | 784.42 | 0.79% | 0.5002 | 0.5274 | 0.0161 | M30 bullish FVG [0.4944, 0.5002], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 52 | 2025-01-21 23:00 | 0.4854 | 0.4670 | 2025-01-22 08:00 | 0.4928 | STOCH_EXIT | 3 | 1423.65 | 1.42% | 0.4848 | 0.5222 | 0.0184 | M30 bullish FVG [0.4834, 0.4848], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 53 | 2025-01-24 17:30 | 0.4940 | 0.4772 | 2025-01-25 02:30 | 0.5022 | STOCH_EXIT | 14 | 1586.59 | 1.56% | 0.4918 | 0.5276 | 0.0168 | M30 bullish FVG [0.4904, 0.4918], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 54 | 2025-01-29 09:00 | 0.4946 | 0.4784 | 2025-01-29 14:00 | 0.4976 | STOCH_EXIT | 10 | 523.32 | 0.51% | 0.4916 | 0.5269 | 0.0162 | M30 bullish FVG [0.4900, 0.4916], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 55 | 2025-01-31 16:30 | 0.4980 | 0.4831 | 2025-02-01 06:00 | 0.5010 | STOCH_EXIT | 19 | 520.10 | 0.50% | 0.4990 | 0.5278 | 0.0149 | M30 bullish FVG [0.4962, 0.4990], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 56 | 2025-02-03 08:00 | 0.4940 | 0.4796 | 2025-02-03 09:30 | 0.4992 | STOCH_EXIT | 3 | 992.42 | 0.95% | 0.4928 | 0.5227 | 0.0144 | M30 bullish FVG [0.4924, 0.4928], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 57 | 2025-02-03 20:30 | 0.5188 | 0.5044 | 2025-02-04 10:00 | 0.5288 | STOCH_EXIT | 13 | 1923.67 | 1.83% | 0.5176 | 0.5475 | 0.0144 | M30 bullish FVG [0.5162, 0.5176], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 58 | 2025-02-04 23:30 | 0.5238 | 0.5084 | 2025-02-06 17:00 | 0.5322 | STOCH_EXIT | 55 | 1605.89 | 1.50% | 0.5222 | 0.5546 | 0.0154 | M30 bullish FVG [0.5216, 0.5222], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 59 | 2025-02-12 15:00 | 0.5322 | 0.5180 | 2025-02-12 20:00 | 0.5350 | STOCH_EXIT | 10 | 462.35 | 0.43% | 0.5320 | 0.5605 | 0.0142 | M30 bullish FVG [0.5298, 0.5320], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 60 | 2025-02-13 13:30 | 0.5424 | 0.5287 | 2025-02-13 18:30 | 0.5482 | STOCH_EXIT | 10 | 1056.19 | 0.97% | 0.5512 | 0.5699 | 0.0137 | M30 bullish FVG [0.5420, 0.5512], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 61 | 2025-02-17 16:00 | 0.5616 | 0.5464 | 2025-02-17 18:30 | 0.5628 | STOCH_EXIT | 5 | 125.01 | 0.11% | 0.5604 | 0.5920 | 0.0152 | M30 bullish FVG [0.5524, 0.5604], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 62 | 2025-03-05 07:30 | 0.6426 | 0.6185 | 2025-03-05 11:30 | 0.6480 | STOCH_EXIT | 8 | 817.81 | 0.74% | 0.6406 | 0.6907 | 0.0241 | M30 bullish FVG [0.6352, 0.6406], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 63 | 2025-03-05 20:00 | 0.6564 | 0.6323 | 2025-03-06 09:30 | 0.6564 | STOCH_EXIT | 13 | -110.93 | -0.10% | 0.6516 | 0.7045 | 0.0241 | M30 bullish FVG [0.6480, 0.6516], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 64 | 2025-03-07 08:00 | 0.6456 | 0.6212 | 2025-03-07 08:30 | 0.6488 | STOCH_EXIT | 1 | 439.08 | 0.40% | 0.6422 | 0.6943 | 0.0244 | M30 bullish FVG [0.6412, 0.6422], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 65 | 2025-03-07 16:00 | 0.6484 | 0.6240 | 2025-03-07 21:30 | 0.6454 | STOCH_EXIT | 11 | -627.27 | -0.56% | 0.6520 | 0.6971 | 0.0244 | M30 bullish FVG [0.6450, 0.6520], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 66 | 2025-03-11 23:00 | 0.6450 | 0.6217 | 2025-03-12 21:30 | 0.6478 | STOCH_EXIT | 30 | 370.42 | 0.33% | 0.6450 | 0.6916 | 0.0233 | M30 bullish FVG [0.6410, 0.6450], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 67 | 2025-03-17 17:00 | 0.6616 | 0.6386 | 2025-03-17 19:00 | 0.6648 | STOCH_EXIT | 4 | 426.18 | 0.38% | 0.6632 | 0.7075 | 0.0230 | M30 bullish FVG [0.6596, 0.6632], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 68 | 2025-03-24 16:00 | 0.6302 | 0.6089 | 2025-03-24 18:00 | 0.6358 | STOCH_EXIT | 4 | 879.16 | 0.79% | 0.6336 | 0.6729 | 0.0213 | M30 bullish FVG [0.6268, 0.6336], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 69 | 2025-03-28 23:30 | 0.6088 | 0.5892 | 2025-03-29 03:00 | 0.6120 | STOCH_EXIT | 3 | 479.08 | 0.43% | 0.6040 | 0.6481 | 0.0196 | M30 bullish FVG [0.6036, 0.6040], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 70 | 2025-04-03 10:30 | 0.5626 | 0.5410 | 2025-04-03 23:30 | 0.5498 | STOCH_EXIT | 26 | -2684.60 | -2.37% | 0.5612 | 0.6057 | 0.0216 | M30 bullish FVG [0.5586, 0.5612], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 71 | 2025-04-11 23:30 | 0.5534 | 0.5286 | 2025-04-12 11:30 | 0.5662 | STOCH_EXIT | 15 | 2435.80 | 2.21% | 0.5540 | 0.6030 | 0.0248 | M30 bullish FVG [0.5488, 0.5540], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 72 | 2025-04-14 09:30 | 0.5642 | 0.5405 | 2025-04-15 09:30 | 0.5530 | STOCH_EXIT | 34 | -2351.72 | -2.08% | 0.5664 | 0.6116 | 0.0237 | M30 bullish FVG [0.5638, 0.5664], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 73 | 2025-04-17 20:30 | 0.5728 | 0.5514 | 2025-04-17 20:30 | 0.5774 | STOCH_EXIT | 0 | 776.81 | 0.70% | 0.5678 | 0.6155 | 0.0214 | M30 bullish FVG [0.5644, 0.5678], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 74 | 2025-04-21 18:30 | 0.5780 | 0.5577 | 2025-04-21 22:00 | 0.5840 | STOCH_EXIT | 7 | 1040.45 | 0.94% | 0.5788 | 0.6185 | 0.0203 | M30 bullish FVG [0.5766, 0.5788], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 75 | 2025-04-23 08:00 | 0.5854 | 0.5661 | 2025-04-23 15:30 | 0.5782 | STOCH_EXIT | 15 | -1494.11 | -1.33% | 0.5772 | 0.6239 | 0.0193 | M30 bullish FVG [0.5762, 0.5772], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 76 | 2025-05-08 07:30 | 0.5442 | 0.5253 | 2025-05-08 07:30 | 0.5500 | STOCH_EXIT | 0 | 1066.34 | 0.96% | 0.5444 | 0.5820 | 0.0189 | M30 bullish FVG [0.5438, 0.5444], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 77 | 2025-05-20 09:30 | 0.5964 | 0.5781 | 2025-05-20 12:30 | 0.5984 | STOCH_EXIT | 6 | 262.29 | 0.24% | 0.5966 | 0.6329 | 0.0183 | M30 bullish FVG [0.5946, 0.5966], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 78 | 2025-05-21 10:30 | 0.6088 | 0.5903 | 2025-05-22 07:00 | 0.6076 | STOCH_EXIT | 27 | -332.71 | -0.30% | 0.6116 | 0.6457 | 0.0185 | M30 bullish FVG [0.6082, 0.6116], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 79 | 2025-05-27 07:30 | 0.6000 | 0.5810 | 2025-05-27 09:00 | 0.6036 | STOCH_EXIT | 3 | 557.67 | 0.50% | 0.6012 | 0.6380 | 0.0190 | M30 bullish FVG [0.5988, 0.6012], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 80 | 2025-05-27 23:00 | 0.6278 | 0.6088 | 2025-05-28 16:30 | 0.6328 | STOCH_EXIT | 22 | 782.18 | 0.70% | 0.6270 | 0.6658 | 0.0190 | M30 bullish FVG [0.6262, 0.6270], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 81 | 2025-05-29 11:00 | 0.6360 | 0.6161 | 2025-05-29 12:00 | 0.6384 | STOCH_EXIT | 2 | 313.78 | 0.28% | 0.6354 | 0.6758 | 0.0199 | M30 bullish FVG [0.6348, 0.6354], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 82 | 2025-06-02 13:30 | 0.6300 | 0.6115 | 2025-06-02 17:00 | 0.6348 | STOCH_EXIT | 7 | 750.17 | 0.66% | 0.6280 | 0.6669 | 0.0185 | M30 bullish FVG [0.6276, 0.6280], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 83 | 2025-06-05 12:00 | 0.6420 | 0.6246 | 2025-06-05 16:00 | 0.6436 | STOCH_EXIT | 8 | 170.38 | 0.15% | 0.6412 | 0.6767 | 0.0174 | M30 bullish FVG [0.6398, 0.6412], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 84 | 2025-06-06 17:30 | 0.6490 | 0.6321 | 2025-06-06 22:30 | 0.6526 | STOCH_EXIT | 10 | 519.06 | 0.45% | 0.6504 | 0.6827 | 0.0169 | M30 bullish FVG [0.6490, 0.6504], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 85 | 2025-06-09 13:30 | 0.6610 | 0.6437 | 2025-06-09 20:30 | 0.6614 | STOCH_EXIT | 14 | -45.45 | -0.04% | 0.6650 | 0.6956 | 0.0173 | M30 bullish FVG [0.6580, 0.6650], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 86 | 2025-06-12 09:30 | 0.6812 | 0.6632 | 2025-06-12 14:30 | 0.6800 | STOCH_EXIT | 10 | -315.94 | -0.28% | 0.6658 | 0.7171 | 0.0180 | M30 bullish FVG [0.6646, 0.6658], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 87 | 2025-06-13 17:30 | 0.6826 | 0.6646 | 2025-06-14 18:30 | 0.6890 | STOCH_EXIT | 43 | 959.99 | 0.84% | 0.6832 | 0.7185 | 0.0180 | M30 bullish FVG [0.6796, 0.6832], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 88 | 2025-06-17 22:30 | 0.6810 | 0.6617 | 2025-06-17 23:00 | 0.6834 | STOCH_EXIT | 1 | 290.31 | 0.25% | 0.6796 | 0.7196 | 0.0193 | M30 bullish FVG [0.6794, 0.6796], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 89 | 2025-06-20 08:30 | 0.6752 | 0.6563 | 2025-06-20 17:00 | 0.6790 | STOCH_EXIT | 17 | 534.02 | 0.46% | 0.6770 | 0.7130 | 0.0189 | M30 bullish FVG [0.6750, 0.6770], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 90 | 2025-06-26 07:00 | 0.5852 | 0.5586 | 2025-06-26 07:30 | 0.5948 | STOCH_EXIT | 1 | 1792.99 | 1.54% | 0.5790 | 0.6384 | 0.0266 | M30 bullish FVG [0.5752, 0.5790], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 91 | 2025-06-27 23:30 | 0.5922 | 0.5664 | 2025-06-28 03:30 | 0.5966 | STOCH_EXIT | 2 | 757.31 | 0.64% | 0.5888 | 0.6438 | 0.0258 | M30 bullish FVG [0.5864, 0.5888], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 92 | 2025-06-30 22:30 | 0.5966 | 0.5711 | 2025-06-30 23:30 | 0.5994 | STOCH_EXIT | 2 | 438.20 | 0.37% | 0.5988 | 0.6475 | 0.0255 | M30 bullish FVG [0.5966, 0.5988], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 93 | 2025-07-01 20:00 | 0.6144 | 0.5899 | 2025-07-02 06:30 | 0.6184 | STOCH_EXIT | 8 | 656.42 | 0.55% | 0.6120 | 0.6633 | 0.0245 | M30 bullish FVG [0.6114, 0.6120], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 94 | 2025-07-03 10:00 | 0.6204 | 0.5962 | 2025-07-03 15:00 | 0.6200 | STOCH_EXIT | 10 | -196.90 | -0.16% | 0.6202 | 0.6689 | 0.0242 | M30 bullish FVG [0.6196, 0.6202], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 95 | 2025-07-03 18:00 | 0.6164 | 0.5922 | 2025-07-04 16:00 | 0.6124 | STOCH_EXIT | 31 | -895.19 | -0.75% | 0.6172 | 0.6649 | 0.0242 | M30 bullish FVG [0.6162, 0.6172], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 96 | 2025-07-07 22:00 | 0.6138 | 0.5909 | 2025-07-08 12:30 | 0.6176 | STOCH_EXIT | 16 | 614.57 | 0.52% | 0.6144 | 0.6597 | 0.0229 | M30 bullish FVG [0.6130, 0.6144], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 97 | 2025-07-08 16:00 | 0.6128 | 0.5908 | 2025-07-08 17:00 | 0.6174 | STOCH_EXIT | 2 | 777.06 | 0.65% | 0.6126 | 0.6568 | 0.0220 | M30 bullish FVG [0.6120, 0.6126], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 98 | 2025-07-11 09:30 | 0.6214 | 0.6000 | 2025-07-11 19:00 | 0.6184 | STOCH_EXIT | 19 | -698.64 | -0.58% | 0.6220 | 0.6643 | 0.0214 | M30 bullish FVG [0.6136, 0.6220], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 99 | 2025-07-15 23:00 | 0.6204 | 0.5994 | 2025-07-16 08:00 | 0.6226 | STOCH_EXIT | 5 | 304.65 | 0.25% | 0.6206 | 0.6624 | 0.0210 | M30 bullish FVG [0.6200, 0.6206], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 100 | 2025-07-16 11:00 | 0.6204 | 0.6001 | 2025-07-16 15:30 | 0.6224 | STOCH_EXIT | 9 | 266.07 | 0.22% | 0.6210 | 0.6611 | 0.0203 | M30 bullish FVG [0.6204, 0.6210], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 101 | 2025-07-17 17:00 | 0.6466 | 0.6271 | 2025-07-17 23:30 | 0.6490 | STOCH_EXIT | 13 | 325.91 | 0.27% | 0.6422 | 0.6857 | 0.0195 | M30 bullish FVG [0.6300, 0.6422], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 102 | 2025-07-21 17:00 | 0.6736 | 0.6521 | 2025-07-22 07:00 | 0.6732 | STOCH_EXIT | 14 | -192.14 | -0.16% | 0.6756 | 0.7165 | 0.0215 | M30 bullish FVG [0.6732, 0.6756], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 103 | 2025-07-28 19:00 | 0.7066 | 0.6889 | 2025-07-28 22:00 | 0.7106 | STOCH_EXIT | 6 | 559.54 | 0.47% | 0.7088 | 0.7421 | 0.0177 | M30 bullish FVG [0.7010, 0.7088], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 104 | 2025-07-30 20:00 | 0.6888 | 0.6666 | 2025-07-31 09:00 | 0.6902 | STOCH_EXIT | 13 | 124.34 | 0.10% | 0.6940 | 0.7333 | 0.0222 | M30 bullish FVG [0.6856, 0.6940], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 105 | 2025-07-31 16:00 | 0.6890 | 0.6665 | 2025-07-31 17:30 | 0.6920 | STOCH_EXIT | 3 | 404.16 | 0.34% | 0.6896 | 0.7340 | 0.0225 | M30 bullish FVG [0.6866, 0.6896], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 106 | 2025-08-01 20:00 | 0.7036 | 0.6815 | 2025-08-02 04:00 | 0.7126 | STOCH_EXIT | 10 | 1426.21 | 1.18% | 0.7014 | 0.7477 | 0.0221 | M30 bullish FVG [0.7004, 0.7014], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 107 | 2025-08-05 11:00 | 0.7398 | 0.7155 | 2025-08-05 16:30 | 0.7456 | STOCH_EXIT | 11 | 839.51 | 0.68% | 0.7388 | 0.7884 | 0.0243 | M30 bullish FVG [0.7288, 0.7388], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 108 | 2025-08-06 12:30 | 0.7602 | 0.7354 | 2025-08-06 20:30 | 0.7572 | STOCH_EXIT | 16 | -608.91 | -0.49% | 0.7526 | 0.8099 | 0.0248 | M30 bullish FVG [0.7520, 0.7526], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 109 | 2025-08-14 18:30 | 0.7214 | 0.6977 | 2025-08-14 22:30 | 0.7250 | STOCH_EXIT | 8 | 489.06 | 0.40% | 0.7200 | 0.7688 | 0.0237 | M30 bullish FVG [0.7192, 0.7200], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 110 | 2025-08-19 17:00 | 0.7326 | 0.7095 | 2025-08-19 19:30 | 0.7334 | STOCH_EXIT | 5 | 11.26 | 0.01% | 0.7324 | 0.7788 | 0.0231 | M30 bullish FVG [0.7320, 0.7324], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 111 | 2025-08-22 18:00 | 0.7178 | 0.6958 | 2025-08-22 18:00 | 0.7230 | STOCH_EXIT | 0 | 770.49 | 0.62% | 0.7138 | 0.7619 | 0.0220 | M30 bullish FVG [0.7122, 0.7138], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 112 | 2025-08-26 09:00 | 0.7402 | 0.7181 | 2025-08-26 13:30 | 0.7452 | STOCH_EXIT | 9 | 715.23 | 0.57% | 0.7354 | 0.7845 | 0.0221 | M30 bullish FVG [0.7352, 0.7354], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 113 | 2025-08-26 17:00 | 0.7440 | 0.7219 | 2025-08-26 22:30 | 0.7386 | STOCH_EXIT | 11 | -1031.74 | -0.83% | 0.7430 | 0.7883 | 0.0221 | M30 bullish FVG [0.7428, 0.7430], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 114 | 2025-09-01 18:30 | 0.7356 | 0.7143 | 2025-09-01 18:30 | 0.7384 | STOCH_EXIT | 0 | 346.58 | 0.28% | 0.7330 | 0.7782 | 0.0213 | M30 bullish FVG [0.7298, 0.7330], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 115 | 2025-09-03 11:00 | 0.7270 | 0.7063 | 2025-09-03 21:00 | 0.7234 | STOCH_EXIT | 20 | -739.61 | -0.59% | 0.7264 | 0.7685 | 0.0207 | M30 bullish FVG [0.7262, 0.7264], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 116 | 2025-09-04 17:30 | 0.7276 | 0.7071 | 2025-09-05 11:00 | 0.7312 | STOCH_EXIT | 21 | 488.00 | 0.39% | 0.7248 | 0.7685 | 0.0205 | M30 bullish FVG [0.7238, 0.7248], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 117 | 2025-09-08 16:00 | 0.7490 | 0.7290 | 2025-09-09 08:30 | 0.7496 | STOCH_EXIT | 19 | -24.63 | -0.02% | 0.7494 | 0.7891 | 0.0200 | M30 bullish FVG [0.7480, 0.7494], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 118 | 2025-09-10 11:00 | 0.7500 | 0.7310 | 2025-09-10 18:00 | 0.7540 | STOCH_EXIT | 14 | 535.92 | 0.43% | 0.7486 | 0.7881 | 0.0190 | M30 bullish FVG [0.7476, 0.7486], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 119 | 2025-09-16 07:30 | 0.7152 | 0.6953 | 2025-09-16 08:00 | 0.7206 | STOCH_EXIT | 1 | 814.69 | 0.65% | 0.7136 | 0.7551 | 0.0199 | M30 bullish FVG [0.7130, 0.7136], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 120 | 2025-09-17 13:00 | 0.7168 | 0.6972 | 2025-09-17 13:30 | 0.7218 | STOCH_EXIT | 1 | 749.12 | 0.60% | 0.7140 | 0.7561 | 0.0196 | M30 bullish FVG [0.7138, 0.7140], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 121 | 2025-09-18 14:30 | 0.7180 | 0.6983 | 2025-09-19 16:00 | 0.7160 | STOCH_EXIT | 37 | -475.47 | -0.38% | 0.7172 | 0.7574 | 0.0197 | M30 bullish FVG [0.7162, 0.7172], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 122 | 2025-09-23 11:30 | 0.7180 | 0.6981 | 2025-09-23 12:00 | 0.7228 | STOCH_EXIT | 1 | 713.93 | 0.57% | 0.7108 | 0.7578 | 0.0199 | M30 bullish FVG [0.7096, 0.7108], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 123 | 2025-09-23 19:30 | 0.7264 | 0.7065 | 2025-09-23 23:00 | 0.7065 | SL | 7 | -3584.04 | -2.83% | 0.7224 | 0.7662 | 0.0199 | M30 bullish FVG [0.7206, 0.7224], stochastic crossed below 20 | FVG stop 0.7065 |
| 124 | 2025-09-25 17:30 | 0.7684 | 0.7441 | 2025-09-25 20:00 | 0.7724 | STOCH_EXIT | 5 | 516.74 | 0.42% | 0.7712 | 0.8170 | 0.0243 | M30 bullish FVG [0.7670, 0.7712], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 125 | 2025-09-29 13:30 | 0.7860 | 0.7616 | 2025-09-29 23:30 | 0.7914 | STOCH_EXIT | 20 | 723.97 | 0.59% | 0.7900 | 0.8348 | 0.0244 | M30 bullish FVG [0.7838, 0.7900], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 126 | 2025-10-01 15:30 | 0.8102 | 0.7837 | 2025-10-02 12:30 | 0.8016 | STOCH_EXIT | 29 | -1439.10 | -1.16% | 0.8064 | 0.8633 | 0.0265 | M30 bullish FVG [0.8058, 0.8064], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 127 | 2025-10-07 22:00 | 0.8292 | 0.8017 | 2025-10-08 07:30 | 0.8320 | STOCH_EXIT | 6 | 291.47 | 0.24% | 0.8354 | 0.8842 | 0.0275 | M30 bullish FVG [0.8270, 0.8354], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 128 | 2025-10-09 23:30 | 0.8552 | 0.8272 | 2025-10-10 11:00 | 0.8272 | SL | 10 | -4129.47 | -3.37% | 0.8612 | 0.9113 | 0.0280 | M30 bullish FVG [0.8424, 0.8612], stochastic crossed below 20 | FVG stop 0.8272 |
| 129 | 2025-10-14 09:30 | 0.7118 | 0.6674 | 2025-10-15 13:00 | 0.8007 | TP | 41 | 14717.82 | 12.38% | 0.7090 | 0.8007 | 0.0444 | M30 bullish FVG [0.7080, 0.7090], stochastic crossed below 20 | FVG target 0.8007 |
| 130 | 2025-10-16 10:30 | 0.7774 | 0.7263 | 2025-10-16 16:00 | 0.7830 | STOCH_EXIT | 11 | 824.19 | 0.62% | 0.7780 | 0.8797 | 0.0511 | M30 bullish FVG [0.7642, 0.7780], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 131 | 2025-10-20 11:00 | 0.7820 | 0.7330 | 2025-10-20 22:00 | 0.7824 | STOCH_EXIT | 22 | -65.36 | -0.05% | 0.7814 | 0.8800 | 0.0490 | M30 bullish FVG [0.7800, 0.7814], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 132 | 2025-10-22 14:00 | 0.7686 | 0.7222 | 2025-10-22 19:00 | 0.7722 | STOCH_EXIT | 10 | 492.35 | 0.37% | 0.7698 | 0.8614 | 0.0464 | M30 bullish FVG [0.7684, 0.7698], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 133 | 2025-10-27 13:30 | 0.7114 | 0.6677 | 2025-10-27 21:30 | 0.7068 | STOCH_EXIT | 16 | -1003.42 | -0.75% | 0.7126 | 0.7987 | 0.0437 | M30 bullish FVG [0.7112, 0.7126], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 95218.98
- Макс: 135055.17
- Финал: 133984.57

Полная кривая — в `*_equity.csv`.
