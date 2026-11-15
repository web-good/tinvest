# Бэктест MRKC (Minutes30)

- Период: 2023-11-15 23:18 — 2025-11-15 23:18
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
| TakeProfitATR | 2 |
| HTFTrendEMA | 120 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 41 |
| Выигрышных / проигрышных | 26 / 15 |
| Win rate | 63.41% |
| Gross profit / loss | 26186.09 / 15963.92 |
| Profit factor | 1.640 |
| Чистый PnL | 10222.17 (10.22%) |
| Стартовый / финальный капитал | 100000.00 / 110222.17 |
| Макс. просадка | 7329.79 (6.31%) |
| Средняя прибыль / убыток | 1007.16 / 1064.26 |
| Expectancy | 249.32 |
| Лучшая / худшая сделка | 5743.75 / -3811.82 |
| Exposure | 2.18% |
| CAGR | 4.98% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-11-22 18:00 | 0.6004 | 0.5820 | 2023-11-24 12:00 | 0.5952 | STOCH_EXIT | 26 | -962.43 | -0.97% | 0.6044 | 0.6372 | 0.0184 | M30 bullish FVG [0.5982, 0.6044], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 2 | 2023-12-08 15:00 | 0.5594 | 0.5406 | 2023-12-08 16:30 | 0.5730 | STOCH_EXIT | 3 | 2293.95 | 2.33% | 0.5632 | 0.5970 | 0.0188 | M30 bullish FVG [0.5564, 0.5632], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 3 | 2023-12-19 16:30 | 0.5786 | 0.5587 | 2023-12-20 16:00 | 0.5810 | STOCH_EXIT | 18 | 318.54 | 0.31% | 0.5816 | 0.6183 | 0.0199 | M30 bullish FVG [0.5750, 0.5816], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 4 | 2024-01-10 13:30 | 0.6146 | 0.5978 | 2024-01-11 15:30 | 0.6062 | STOCH_EXIT | 23 | -1486.72 | -1.47% | 0.6140 | 0.6482 | 0.0168 | M30 bullish FVG [0.6100, 0.6140], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 5 | 2024-01-23 15:00 | 0.6106 | 0.5949 | 2024-01-24 10:00 | 0.6240 | STOCH_EXIT | 9 | 2083.58 | 2.09% | 0.6214 | 0.6421 | 0.0157 | M30 bullish FVG [0.6174, 0.6214], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 6 | 2024-03-11 16:00 | 0.5816 | 0.5696 | 2024-03-12 11:00 | 0.5848 | STOCH_EXIT | 9 | 457.94 | 0.45% | 0.5826 | 0.6057 | 0.0120 | M30 bullish FVG [0.5794, 0.5826], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 7 | 2024-03-13 13:30 | 0.5832 | 0.5713 | 2024-03-14 10:30 | 0.5713 | SL | 13 | -2196.84 | -2.14% | 0.5834 | 0.6070 | 0.0119 | M30 bullish FVG [0.5784, 0.5834], stochastic crossed below 20 | FVG stop 0.5713 |
| 8 | 2024-03-19 18:00 | 0.5628 | 0.5491 | 2024-03-21 09:30 | 0.5650 | STOCH_EXIT | 21 | 291.23 | 0.29% | 0.5700 | 0.5901 | 0.0137 | M30 bullish FVG [0.5664, 0.5700], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 9 | 2024-04-22 17:00 | 0.6160 | 0.6050 | 2024-04-23 09:30 | 0.6198 | STOCH_EXIT | 4 | 518.68 | 0.52% | 0.6130 | 0.6379 | 0.0110 | M30 bullish FVG [0.6098, 0.6130], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 10 | 2024-04-26 11:00 | 0.6048 | 0.5935 | 2024-04-26 18:00 | 0.6036 | STOCH_EXIT | 14 | -301.30 | -0.30% | 0.6074 | 0.6273 | 0.0113 | M30 bullish FVG [0.6028, 0.6074], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 11 | 2024-04-29 12:30 | 0.6254 | 0.6140 | 2024-04-29 18:30 | 0.6296 | STOCH_EXIT | 12 | 575.17 | 0.57% | 0.6304 | 0.6482 | 0.0114 | M30 bullish FVG [0.6262, 0.6304], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 12 | 2024-05-03 17:30 | 0.6350 | 0.6215 | 2024-05-03 18:30 | 0.6396 | STOCH_EXIT | 2 | 630.07 | 0.62% | 0.6360 | 0.6619 | 0.0135 | M30 bullish FVG [0.6300, 0.6360], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 13 | 2024-05-08 14:30 | 0.6438 | 0.6290 | 2024-05-08 17:00 | 0.6290 | SL | 5 | -2431.31 | -2.39% | 0.6484 | 0.6733 | 0.0148 | M30 bullish FVG [0.6406, 0.6484], stochastic crossed below 20 | FVG stop 0.6290 |
| 14 | 2024-11-13 19:30 | 0.4712 | 0.4575 | 2024-11-14 11:00 | 0.4718 | STOCH_EXIT | 17 | 27.11 | 0.03% | 0.4630 | 0.4985 | 0.0137 | M30 bullish FVG [0.4542, 0.4630], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 15 | 2024-11-19 12:30 | 0.4790 | 0.4636 | 2024-11-19 19:30 | 0.4806 | STOCH_EXIT | 14 | 233.00 | 0.23% | 0.4922 | 0.5098 | 0.0154 | M30 bullish FVG [0.4878, 0.4922], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 16 | 2024-12-02 13:30 | 0.4360 | 0.4172 | 2024-12-02 16:30 | 0.4460 | STOCH_EXIT | 6 | 2189.01 | 2.19% | 0.4480 | 0.4736 | 0.0188 | M30 bullish FVG [0.4424, 0.4480], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 17 | 2024-12-23 15:00 | 0.4672 | 0.4488 | 2024-12-23 16:00 | 0.4722 | STOCH_EXIT | 2 | 987.61 | 0.97% | 0.4746 | 0.5039 | 0.0184 | M30 bullish FVG [0.4700, 0.4746], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 18 | 2024-12-24 10:30 | 0.4726 | 0.4527 | 2024-12-24 14:30 | 0.4728 | STOCH_EXIT | 8 | -59.45 | -0.06% | 0.4792 | 0.5124 | 0.0199 | M30 bullish FVG [0.4746, 0.4792], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 19 | 2025-01-17 23:30 | 0.4952 | 0.4791 | 2025-01-18 09:30 | 0.4996 | STOCH_EXIT | 6 | 811.74 | 0.79% | 0.5002 | 0.5274 | 0.0161 | M30 bullish FVG [0.4944, 0.5002], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 20 | 2025-02-03 18:00 | 0.5150 | 0.5006 | 2025-02-03 23:30 | 0.5256 | STOCH_EXIT | 11 | 2026.02 | 1.96% | 0.5096 | 0.5437 | 0.0144 | M30 bullish FVG [0.5020, 0.5096], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 21 | 2025-02-13 14:30 | 0.5402 | 0.5265 | 2025-02-13 17:30 | 0.5448 | STOCH_EXIT | 6 | 795.27 | 0.75% | 0.5512 | 0.5677 | 0.0137 | M30 bullish FVG [0.5420, 0.5512], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 22 | 2025-02-17 15:00 | 0.5600 | 0.5448 | 2025-02-17 18:30 | 0.5628 | STOCH_EXIT | 7 | 425.33 | 0.40% | 0.5604 | 0.5904 | 0.0152 | M30 bullish FVG [0.5524, 0.5604], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 23 | 2025-02-21 09:00 | 0.5890 | 0.5728 | 2025-02-21 10:00 | 0.6213 | TP | 2 | 5743.75 | 5.38% | 0.5744 | 0.6213 | 0.0162 | M30 bullish FVG [0.5616, 0.5744], stochastic crossed below 20 | FVG target 0.6213 |
| 24 | 2025-02-25 20:30 | 0.6688 | 0.6506 | 2025-02-26 14:00 | 0.6506 | SL | 21 | -3169.98 | -2.82% | 0.6500 | 0.7052 | 0.0182 | M30 bullish FVG [0.6330, 0.6500], stochastic crossed below 20 | FVG stop 0.6506 |
| 25 | 2025-02-26 22:00 | 0.6504 | 0.6298 | 2025-02-27 09:00 | 0.6604 | STOCH_EXIT | 8 | 1569.89 | 1.44% | 0.6654 | 0.6917 | 0.0206 | M30 bullish FVG [0.6600, 0.6654], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 26 | 2025-03-07 15:00 | 0.6488 | 0.6244 | 2025-03-07 21:00 | 0.6456 | STOCH_EXIT | 12 | -657.87 | -0.59% | 0.6520 | 0.6975 | 0.0244 | M30 bullish FVG [0.6450, 0.6520], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 27 | 2025-03-24 16:00 | 0.6302 | 0.6089 | 2025-03-24 16:30 | 0.6320 | STOCH_EXIT | 1 | 204.56 | 0.19% | 0.6336 | 0.6729 | 0.0213 | M30 bullish FVG [0.6268, 0.6336], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 28 | 2025-05-26 19:30 | 0.6020 | 0.5835 | 2025-05-26 21:30 | 0.6024 | STOCH_EXIT | 4 | -37.20 | -0.03% | 0.5972 | 0.6391 | 0.0185 | M30 bullish FVG [0.5918, 0.5972], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 29 | 2025-05-27 22:00 | 0.6272 | 0.6082 | 2025-05-28 16:30 | 0.6328 | STOCH_EXIT | 24 | 874.72 | 0.79% | 0.6100 | 0.6652 | 0.0190 | M30 bullish FVG [0.6048, 0.6100], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 30 | 2025-06-09 13:30 | 0.6610 | 0.6437 | 2025-06-09 20:30 | 0.6614 | STOCH_EXIT | 14 | -43.88 | -0.04% | 0.6650 | 0.6956 | 0.0173 | M30 bullish FVG [0.6580, 0.6650], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 31 | 2025-07-11 08:30 | 0.6206 | 0.5992 | 2025-07-11 19:00 | 0.6184 | STOCH_EXIT | 21 | -504.69 | -0.45% | 0.6220 | 0.6635 | 0.0214 | M30 bullish FVG [0.6136, 0.6220], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 32 | 2025-07-17 16:00 | 0.6484 | 0.6289 | 2025-07-17 21:00 | 0.6476 | STOCH_EXIT | 10 | -247.61 | -0.22% | 0.6576 | 0.6875 | 0.0195 | M30 bullish FVG [0.6484, 0.6576], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 33 | 2025-07-28 18:00 | 0.7070 | 0.6893 | 2025-07-28 21:00 | 0.7106 | STOCH_EXIT | 6 | 451.03 | 0.41% | 0.7088 | 0.7425 | 0.0177 | M30 bullish FVG [0.7010, 0.7088], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 34 | 2025-07-30 20:00 | 0.6888 | 0.6666 | 2025-07-31 09:00 | 0.6902 | STOCH_EXIT | 13 | 114.39 | 0.10% | 0.6940 | 0.7333 | 0.0222 | M30 bullish FVG [0.6856, 0.6940], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 35 | 2025-08-04 17:00 | 0.7232 | 0.7003 | 2025-08-04 19:00 | 0.7302 | STOCH_EXIT | 4 | 966.09 | 0.87% | 0.7126 | 0.7689 | 0.0229 | M30 bullish FVG [0.7050, 0.7126], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 36 | 2025-08-05 11:00 | 0.7398 | 0.7155 | 2025-08-05 16:00 | 0.7402 | STOCH_EXIT | 10 | -51.34 | -0.05% | 0.7388 | 0.7884 | 0.0243 | M30 bullish FVG [0.7288, 0.7388], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 37 | 2025-09-29 13:30 | 0.7860 | 0.7616 | 2025-09-29 23:30 | 0.7914 | STOCH_EXIT | 20 | 654.80 | 0.59% | 0.7900 | 0.8348 | 0.0244 | M30 bullish FVG [0.7838, 0.7900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 38 | 2025-10-01 15:30 | 0.8102 | 0.7837 | 2025-10-02 06:30 | 0.8110 | STOCH_EXIT | 17 | -1.47 | -0.00% | 0.7972 | 0.8633 | 0.0265 | M30 bullish FVG [0.7896, 0.7972], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 39 | 2025-10-07 22:00 | 0.8292 | 0.8017 | 2025-10-08 07:30 | 0.8320 | STOCH_EXIT | 6 | 267.84 | 0.24% | 0.8354 | 0.8842 | 0.0275 | M30 bullish FVG [0.8270, 0.8354], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 40 | 2025-10-09 23:30 | 0.8552 | 0.8272 | 2025-10-10 11:00 | 0.8272 | SL | 10 | -3811.82 | -3.37% | 0.8882 | 0.9113 | 0.0280 | M30 bullish FVG [0.8776, 0.8882], stochastic crossed below 20 | FVG stop 0.8272 |
| 41 | 2025-10-16 10:30 | 0.7774 | 0.7263 | 2025-10-16 16:00 | 0.7830 | STOCH_EXIT | 11 | 674.77 | 0.62% | 0.7780 | 0.8797 | 0.0511 | M30 bullish FVG [0.7642, 0.7780], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 98090.97
- Макс: 116206.77
- Финал: 110222.17

Полная кривая — в `*_equity.csv`.
