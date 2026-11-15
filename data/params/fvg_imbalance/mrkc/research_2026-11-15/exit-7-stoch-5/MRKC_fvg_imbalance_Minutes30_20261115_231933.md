# Бэктест MRKC (Minutes30)

- Период: 2023-11-15 23:19 — 2025-11-15 23:19
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.25 |
| MaxWaitBars | 30 |
| StochPeriod | 9 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 1 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 54 |
| Выигрышных / проигрышных | 38 / 16 |
| Win rate | 70.37% |
| Gross profit / loss | 42023.87 / 17283.27 |
| Profit factor | 2.431 |
| Чистый PnL | 24740.60 (24.74%) |
| Стартовый / финальный капитал | 100000.00 / 124740.60 |
| Макс. просадка | 6306.31 (5.98%) |
| Средняя прибыль / убыток | 1105.89 / 1080.20 |
| Expectancy | 458.16 |
| Лучшая / худшая сделка | 3803.38 / -3688.41 |
| Exposure | 2.62% |
| CAGR | 11.67% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-11-22 18:00 | 0.6004 | 0.5820 | 2023-11-24 12:00 | 0.5952 | STOCH_EXIT | 26 | -962.43 | -0.97% | 0.6044 | 0.6188 | 0.0184 | M30 bullish FVG [0.5982, 0.6044], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 2 | 2023-12-08 15:00 | 0.5594 | 0.5406 | 2023-12-08 16:30 | 0.5730 | STOCH_EXIT | 3 | 2293.95 | 2.33% | 0.5632 | 0.5782 | 0.0188 | M30 bullish FVG [0.5564, 0.5632], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 3 | 2023-12-19 16:30 | 0.5786 | 0.5587 | 2023-12-20 16:00 | 0.5810 | STOCH_EXIT | 18 | 318.54 | 0.31% | 0.5816 | 0.5985 | 0.0199 | M30 bullish FVG [0.5750, 0.5816], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 4 | 2024-01-10 13:30 | 0.6146 | 0.5978 | 2024-01-11 15:30 | 0.6062 | STOCH_EXIT | 23 | -1486.72 | -1.47% | 0.6140 | 0.6314 | 0.0168 | M30 bullish FVG [0.6100, 0.6140], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 5 | 2024-01-23 15:00 | 0.6106 | 0.5949 | 2024-01-24 10:00 | 0.6240 | STOCH_EXIT | 9 | 2083.58 | 2.09% | 0.6214 | 0.6263 | 0.0157 | M30 bullish FVG [0.6174, 0.6214], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 6 | 2024-02-22 10:00 | 0.5568 | 0.5423 | 2024-02-22 10:30 | 0.5626 | STOCH_EXIT | 1 | 958.97 | 0.94% | 0.5564 | 0.5713 | 0.0145 | M30 bullish FVG [0.5524, 0.5564], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 7 | 2024-03-11 16:00 | 0.5816 | 0.5696 | 2024-03-12 11:00 | 0.5848 | STOCH_EXIT | 9 | 463.17 | 0.45% | 0.5826 | 0.5936 | 0.0120 | M30 bullish FVG [0.5794, 0.5826], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 8 | 2024-03-13 13:30 | 0.5832 | 0.5713 | 2024-03-14 10:30 | 0.5713 | SL | 13 | -2209.32 | -2.14% | 0.5834 | 0.5951 | 0.0119 | M30 bullish FVG [0.5784, 0.5834], stochastic crossed below 20 | FVG stop 0.5713 |
| 9 | 2024-03-19 18:00 | 0.5628 | 0.5491 | 2024-03-21 09:30 | 0.5650 | STOCH_EXIT | 21 | 294.50 | 0.29% | 0.5700 | 0.5765 | 0.0137 | M30 bullish FVG [0.5664, 0.5700], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 10 | 2024-04-22 17:00 | 0.6160 | 0.6050 | 2024-04-23 09:30 | 0.6198 | STOCH_EXIT | 4 | 525.05 | 0.52% | 0.6130 | 0.6270 | 0.0110 | M30 bullish FVG [0.6098, 0.6130], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 11 | 2024-04-26 11:00 | 0.6048 | 0.5935 | 2024-04-26 18:00 | 0.6036 | STOCH_EXIT | 14 | -304.91 | -0.30% | 0.6074 | 0.6161 | 0.0113 | M30 bullish FVG [0.6028, 0.6074], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 12 | 2024-04-29 12:30 | 0.6254 | 0.6140 | 2024-04-29 18:30 | 0.6296 | STOCH_EXIT | 12 | 578.75 | 0.57% | 0.6304 | 0.6368 | 0.0114 | M30 bullish FVG [0.6262, 0.6304], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 13 | 2024-05-03 17:30 | 0.6350 | 0.6215 | 2024-05-03 18:30 | 0.6396 | STOCH_EXIT | 2 | 637.99 | 0.62% | 0.6360 | 0.6485 | 0.0135 | M30 bullish FVG [0.6300, 0.6360], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 14 | 2024-05-08 14:30 | 0.6438 | 0.6290 | 2024-05-08 16:30 | 0.6586 | TP | 4 | 2256.07 | 2.19% | 0.6484 | 0.6586 | 0.0148 | M30 bullish FVG [0.6406, 0.6484], stochastic crossed below 20 | FVG target 0.6586 |
| 15 | 2024-05-30 16:00 | 0.5664 | 0.5471 | 2024-05-31 10:30 | 0.5471 | SL | 8 | -3688.41 | -3.50% | 0.5700 | 0.5857 | 0.0193 | M30 bullish FVG [0.5642, 0.5700], stochastic crossed below 20 | FVG stop 0.5471 |
| 16 | 2024-07-05 17:30 | 0.5012 | 0.4815 | 2024-07-06 15:30 | 0.5068 | STOCH_EXIT | 24 | 1029.39 | 1.02% | 0.4992 | 0.5209 | 0.0197 | M30 bullish FVG [0.4942, 0.4992], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 17 | 2024-08-08 14:30 | 0.4440 | 0.4289 | 2024-08-08 19:30 | 0.4289 | SL | 10 | -3588.36 | -3.50% | 0.4486 | 0.4591 | 0.0151 | M30 bullish FVG [0.4444, 0.4486], stochastic crossed below 20 | FVG stop 0.4289 |
| 18 | 2024-08-12 19:30 | 0.4320 | 0.4169 | 2024-08-12 19:30 | 0.4354 | STOCH_EXIT | 0 | 679.28 | 0.69% | 0.4294 | 0.4471 | 0.0151 | M30 bullish FVG [0.4240, 0.4294], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 19 | 2024-09-20 15:00 | 0.3878 | 0.3758 | 2024-09-20 16:00 | 0.3910 | STOCH_EXIT | 2 | 722.32 | 0.72% | 0.3860 | 0.3998 | 0.0120 | M30 bullish FVG [0.3822, 0.3860], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 20 | 2024-10-25 15:30 | 0.3854 | 0.3781 | 2024-10-25 18:30 | 0.3890 | STOCH_EXIT | 6 | 835.33 | 0.83% | 0.3880 | 0.3927 | 0.0073 | M30 bullish FVG [0.3860, 0.3880], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 21 | 2024-10-28 15:30 | 0.3896 | 0.3819 | 2024-10-28 22:30 | 0.3944 | STOCH_EXIT | 14 | 1146.08 | 1.13% | 0.3924 | 0.3973 | 0.0077 | M30 bullish FVG [0.3896, 0.3924], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 22 | 2024-11-04 12:00 | 0.4144 | 0.4029 | 2024-11-04 19:30 | 0.4146 | STOCH_EXIT | 14 | -52.98 | -0.05% | 0.4200 | 0.4259 | 0.0115 | M30 bullish FVG [0.4140, 0.4200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 23 | 2024-11-11 16:00 | 0.4542 | 0.4422 | 2024-11-11 18:30 | 0.4588 | STOCH_EXIT | 5 | 932.29 | 0.91% | 0.4516 | 0.4662 | 0.0120 | M30 bullish FVG [0.4446, 0.4516], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 24 | 2024-11-13 19:30 | 0.4712 | 0.4575 | 2024-11-14 11:00 | 0.4718 | STOCH_EXIT | 17 | 28.14 | 0.03% | 0.4630 | 0.4849 | 0.0137 | M30 bullish FVG [0.4542, 0.4630], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 25 | 2024-11-19 12:30 | 0.4790 | 0.4636 | 2024-11-19 19:30 | 0.4806 | STOCH_EXIT | 14 | 240.84 | 0.23% | 0.4922 | 0.4944 | 0.0154 | M30 bullish FVG [0.4878, 0.4922], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 26 | 2024-11-27 19:00 | 0.4240 | 0.4057 | 2024-11-27 19:00 | 0.4274 | STOCH_EXIT | 0 | 725.73 | 0.70% | 0.4232 | 0.4423 | 0.0183 | M30 bullish FVG [0.4166, 0.4232], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 27 | 2024-12-02 13:30 | 0.4360 | 0.4172 | 2024-12-02 16:30 | 0.4460 | STOCH_EXIT | 6 | 2284.60 | 2.19% | 0.4480 | 0.4548 | 0.0188 | M30 bullish FVG [0.4424, 0.4480], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 28 | 2024-12-20 20:30 | 0.4506 | 0.4330 | 2024-12-21 07:30 | 0.4578 | STOCH_EXIT | 8 | 1592.01 | 1.50% | 0.4402 | 0.4682 | 0.0176 | M30 bullish FVG [0.4348, 0.4402], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 29 | 2024-12-23 15:00 | 0.4672 | 0.4488 | 2024-12-23 16:00 | 0.4722 | STOCH_EXIT | 2 | 1046.50 | 0.97% | 0.4746 | 0.4856 | 0.0184 | M30 bullish FVG [0.4700, 0.4746], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 30 | 2024-12-24 10:30 | 0.4726 | 0.4527 | 2024-12-24 14:30 | 0.4728 | STOCH_EXIT | 8 | -62.99 | -0.06% | 0.4792 | 0.4925 | 0.0199 | M30 bullish FVG [0.4746, 0.4792], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 31 | 2025-01-17 23:30 | 0.4952 | 0.4791 | 2025-01-18 09:30 | 0.4996 | STOCH_EXIT | 6 | 858.57 | 0.79% | 0.5002 | 0.5113 | 0.0161 | M30 bullish FVG [0.4944, 0.5002], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 32 | 2025-02-03 18:00 | 0.5150 | 0.5006 | 2025-02-03 19:00 | 0.5294 | TP | 2 | 2947.04 | 2.69% | 0.5096 | 0.5294 | 0.0144 | M30 bullish FVG [0.5020, 0.5096], stochastic crossed below 20 | FVG target 0.5294 |
| 33 | 2025-02-13 14:30 | 0.5402 | 0.5265 | 2025-02-13 17:30 | 0.5448 | STOCH_EXIT | 6 | 848.02 | 0.75% | 0.5512 | 0.5539 | 0.0137 | M30 bullish FVG [0.5420, 0.5512], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 34 | 2025-02-17 15:00 | 0.5600 | 0.5448 | 2025-02-17 18:30 | 0.5628 | STOCH_EXIT | 7 | 454.44 | 0.40% | 0.5604 | 0.5752 | 0.0152 | M30 bullish FVG [0.5524, 0.5604], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 35 | 2025-02-21 09:00 | 0.5890 | 0.5728 | 2025-02-21 10:00 | 0.6052 | TP | 2 | 3021.01 | 2.64% | 0.5744 | 0.6052 | 0.0162 | M30 bullish FVG [0.5616, 0.5744], stochastic crossed below 20 | FVG target 0.6052 |
| 36 | 2025-02-25 20:30 | 0.6688 | 0.6506 | 2025-02-26 14:00 | 0.6506 | SL | 21 | -3302.06 | -2.82% | 0.6500 | 0.6870 | 0.0182 | M30 bullish FVG [0.6330, 0.6500], stochastic crossed below 20 | FVG stop 0.6506 |
| 37 | 2025-02-26 22:00 | 0.6504 | 0.6298 | 2025-02-27 09:00 | 0.6604 | STOCH_EXIT | 8 | 1635.31 | 1.44% | 0.6654 | 0.6710 | 0.0206 | M30 bullish FVG [0.6600, 0.6654], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 38 | 2025-03-07 15:00 | 0.6488 | 0.6244 | 2025-03-07 21:00 | 0.6456 | STOCH_EXIT | 12 | -684.80 | -0.59% | 0.6520 | 0.6732 | 0.0244 | M30 bullish FVG [0.6450, 0.6520], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 39 | 2025-03-24 16:00 | 0.6302 | 0.6089 | 2025-03-24 16:30 | 0.6320 | STOCH_EXIT | 1 | 212.74 | 0.19% | 0.6336 | 0.6515 | 0.0213 | M30 bullish FVG [0.6268, 0.6336], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 40 | 2025-05-05 15:00 | 0.5306 | 0.5120 | 2025-05-05 22:30 | 0.5416 | STOCH_EXIT | 15 | 2270.67 | 1.97% | 0.5300 | 0.5492 | 0.0186 | M30 bullish FVG [0.5252, 0.5300], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 41 | 2025-05-26 19:30 | 0.6020 | 0.5835 | 2025-05-26 21:30 | 0.6024 | STOCH_EXIT | 4 | -39.43 | -0.03% | 0.5972 | 0.6205 | 0.0185 | M30 bullish FVG [0.5918, 0.5972], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 42 | 2025-05-27 22:00 | 0.6272 | 0.6082 | 2025-05-28 16:30 | 0.6328 | STOCH_EXIT | 24 | 929.39 | 0.79% | 0.6100 | 0.6462 | 0.0190 | M30 bullish FVG [0.6048, 0.6100], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 43 | 2025-06-09 13:30 | 0.6610 | 0.6437 | 2025-06-09 20:30 | 0.6614 | STOCH_EXIT | 14 | -46.75 | -0.04% | 0.6650 | 0.6783 | 0.0173 | M30 bullish FVG [0.6580, 0.6650], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 44 | 2025-07-11 08:30 | 0.6206 | 0.5992 | 2025-07-11 19:00 | 0.6184 | STOCH_EXIT | 21 | -535.71 | -0.45% | 0.6220 | 0.6420 | 0.0214 | M30 bullish FVG [0.6136, 0.6220], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 45 | 2025-07-17 16:00 | 0.6484 | 0.6289 | 2025-07-17 21:00 | 0.6476 | STOCH_EXIT | 10 | -262.09 | -0.22% | 0.6576 | 0.6679 | 0.0195 | M30 bullish FVG [0.6484, 0.6576], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 46 | 2025-07-28 18:00 | 0.7070 | 0.6893 | 2025-07-28 21:00 | 0.7106 | STOCH_EXIT | 6 | 479.94 | 0.41% | 0.7088 | 0.7247 | 0.0177 | M30 bullish FVG [0.7010, 0.7088], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 47 | 2025-07-30 20:00 | 0.6888 | 0.6666 | 2025-07-31 09:00 | 0.6902 | STOCH_EXIT | 13 | 121.50 | 0.10% | 0.6940 | 0.7110 | 0.0222 | M30 bullish FVG [0.6856, 0.6940], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 48 | 2025-08-04 17:00 | 0.7232 | 0.7003 | 2025-08-04 19:00 | 0.7302 | STOCH_EXIT | 4 | 1022.55 | 0.87% | 0.7126 | 0.7461 | 0.0229 | M30 bullish FVG [0.7050, 0.7126], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 49 | 2025-08-05 11:00 | 0.7398 | 0.7155 | 2025-08-05 16:00 | 0.7402 | STOCH_EXIT | 10 | -54.74 | -0.05% | 0.7388 | 0.7641 | 0.0243 | M30 bullish FVG [0.7288, 0.7388], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 50 | 2025-09-29 13:30 | 0.7860 | 0.7616 | 2025-09-29 23:30 | 0.7914 | STOCH_EXIT | 20 | 696.31 | 0.59% | 0.7900 | 0.8104 | 0.0244 | M30 bullish FVG [0.7838, 0.7900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 51 | 2025-10-01 15:30 | 0.8102 | 0.7837 | 2025-10-02 06:30 | 0.8110 | STOCH_EXIT | 17 | -1.56 | -0.00% | 0.7972 | 0.8367 | 0.0265 | M30 bullish FVG [0.7896, 0.7972], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 52 | 2025-10-07 22:00 | 0.8292 | 0.8017 | 2025-10-08 07:30 | 0.8320 | STOCH_EXIT | 6 | 283.59 | 0.24% | 0.8354 | 0.8567 | 0.0275 | M30 bullish FVG [0.8270, 0.8354], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 53 | 2025-10-09 23:30 | 0.8552 | 0.8272 | 2025-10-10 07:00 | 0.8832 | TP | 2 | 3803.38 | 3.18% | 0.8882 | 0.8832 | 0.0280 | M30 bullish FVG [0.8776, 0.8882], stochastic crossed below 20 | FVG target 0.8832 |
| 54 | 2025-10-16 10:30 | 0.7774 | 0.7263 | 2025-10-16 16:00 | 0.7830 | STOCH_EXIT | 11 | 766.35 | 0.62% | 0.7780 | 0.8285 | 0.0511 | M30 bullish FVG [0.7642, 0.7780], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 98090.97
- Макс: 124740.60
- Финал: 124740.60

Полная кривая — в `*_equity.csv`.
