# Бэктест MRKC (Minutes30)

- Период: 2023-11-15 23:20 — 2025-11-15 23:20
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.25 |
| MaxWaitBars | 15 |
| StochPeriod | 9 |
| UseStochExit | 1 |
| StopATR | 1.7 |
| TakeProfitATR | 1 |
| HTFTrendEMA | 100 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 29 |
| Выигрышных / проигрышных | 20 / 9 |
| Win rate | 68.97% |
| Gross profit / loss | 25015.14 / 4759.90 |
| Profit factor | 5.255 |
| Чистый PnL | 20255.24 (20.26%) |
| Стартовый / финальный капитал | 100000.00 / 120255.24 |
| Макс. просадка | 4747.20 (4.04%) |
| Средняя прибыль / убыток | 1250.76 / 528.88 |
| Expectancy | 698.46 |
| Лучшая / худшая сделка | 3694.71 / -1486.72 |
| Exposure | 1.39% |
| CAGR | 9.65% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-11-22 18:00 | 0.6004 | 0.5692 | 2023-11-24 12:00 | 0.5952 | STOCH_EXIT | 26 | -962.43 | -0.97% | 0.6044 | 0.6188 | 0.0184 | M30 bullish FVG [0.5982, 0.6044], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 2 | 2023-12-08 15:00 | 0.5594 | 0.5275 | 2023-12-08 16:30 | 0.5730 | STOCH_EXIT | 3 | 2293.95 | 2.33% | 0.5632 | 0.5782 | 0.0188 | M30 bullish FVG [0.5564, 0.5632], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 3 | 2023-12-19 16:30 | 0.5786 | 0.5448 | 2023-12-20 16:00 | 0.5810 | STOCH_EXIT | 18 | 318.54 | 0.31% | 0.5816 | 0.5985 | 0.0199 | M30 bullish FVG [0.5750, 0.5816], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 4 | 2024-01-10 13:30 | 0.6146 | 0.5861 | 2024-01-11 15:30 | 0.6062 | STOCH_EXIT | 23 | -1486.72 | -1.47% | 0.6140 | 0.6314 | 0.0168 | M30 bullish FVG [0.6100, 0.6140], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 5 | 2024-01-23 15:00 | 0.6106 | 0.5838 | 2024-01-24 10:00 | 0.6240 | STOCH_EXIT | 9 | 2083.58 | 2.09% | 0.6214 | 0.6263 | 0.0157 | M30 bullish FVG [0.6174, 0.6214], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 6 | 2024-03-11 16:00 | 0.5816 | 0.5612 | 2024-03-12 11:00 | 0.5848 | STOCH_EXIT | 9 | 457.94 | 0.45% | 0.5826 | 0.5936 | 0.0120 | M30 bullish FVG [0.5794, 0.5826], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 7 | 2024-04-29 12:30 | 0.6254 | 0.6060 | 2024-04-29 18:30 | 0.6296 | STOCH_EXIT | 12 | 585.89 | 0.57% | 0.6304 | 0.6368 | 0.0114 | M30 bullish FVG [0.6262, 0.6304], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 8 | 2024-05-03 17:30 | 0.6350 | 0.6121 | 2024-05-03 18:30 | 0.6396 | STOCH_EXIT | 2 | 641.96 | 0.62% | 0.6360 | 0.6485 | 0.0135 | M30 bullish FVG [0.6300, 0.6360], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 9 | 2024-05-08 14:30 | 0.6438 | 0.6187 | 2024-05-08 16:30 | 0.6586 | TP | 4 | 2270.17 | 2.19% | 0.6484 | 0.6586 | 0.0148 | M30 bullish FVG [0.6406, 0.6484], stochastic crossed below 20 | FVG target 0.6586 |
| 10 | 2024-11-11 16:00 | 0.4542 | 0.4338 | 2024-11-11 18:30 | 0.4588 | STOCH_EXIT | 5 | 965.44 | 0.91% | 0.4516 | 0.4662 | 0.0120 | M30 bullish FVG [0.4446, 0.4516], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 11 | 2024-12-02 13:30 | 0.4360 | 0.4040 | 2024-12-02 16:30 | 0.4460 | STOCH_EXIT | 6 | 2341.96 | 2.19% | 0.4480 | 0.4548 | 0.0188 | M30 bullish FVG [0.4424, 0.4480], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 12 | 2024-12-23 15:00 | 0.4672 | 0.4360 | 2024-12-23 16:00 | 0.4722 | STOCH_EXIT | 2 | 1060.09 | 0.97% | 0.4746 | 0.4856 | 0.0184 | M30 bullish FVG [0.4700, 0.4746], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 13 | 2025-01-17 23:30 | 0.4952 | 0.4678 | 2025-01-18 09:30 | 0.4996 | STOCH_EXIT | 6 | 870.28 | 0.79% | 0.5002 | 0.5113 | 0.0161 | M30 bullish FVG [0.4944, 0.5002], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 14 | 2025-02-03 18:00 | 0.5150 | 0.4906 | 2025-02-03 19:00 | 0.5294 | TP | 2 | 2988.55 | 2.69% | 0.5096 | 0.5294 | 0.0144 | M30 bullish FVG [0.5020, 0.5096], stochastic crossed below 20 | FVG target 0.5294 |
| 15 | 2025-02-13 14:30 | 0.5402 | 0.5169 | 2025-02-13 17:30 | 0.5448 | STOCH_EXIT | 6 | 856.13 | 0.75% | 0.5512 | 0.5539 | 0.0137 | M30 bullish FVG [0.5420, 0.5512], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 16 | 2025-02-17 15:00 | 0.5600 | 0.5341 | 2025-02-17 18:30 | 0.5628 | STOCH_EXIT | 7 | 458.91 | 0.40% | 0.5604 | 0.5752 | 0.0152 | M30 bullish FVG [0.5524, 0.5604], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 17 | 2025-02-25 20:30 | 0.6688 | 0.6378 | 2025-02-26 16:30 | 0.6654 | STOCH_EXIT | 26 | -699.54 | -0.61% | 0.6500 | 0.6870 | 0.0182 | M30 bullish FVG [0.6330, 0.6500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 18 | 2025-02-26 22:00 | 0.6504 | 0.6153 | 2025-02-27 09:00 | 0.6604 | STOCH_EXIT | 8 | 1644.65 | 1.44% | 0.6654 | 0.6710 | 0.0206 | M30 bullish FVG [0.6600, 0.6654], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 19 | 2025-03-07 15:00 | 0.6488 | 0.6074 | 2025-03-07 21:00 | 0.6456 | STOCH_EXIT | 12 | -688.65 | -0.59% | 0.6520 | 0.6732 | 0.0244 | M30 bullish FVG [0.6450, 0.6520], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 20 | 2025-03-24 16:00 | 0.6302 | 0.5939 | 2025-03-24 16:30 | 0.6320 | STOCH_EXIT | 1 | 213.91 | 0.19% | 0.6336 | 0.6515 | 0.0213 | M30 bullish FVG [0.6268, 0.6336], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 21 | 2025-05-26 19:30 | 0.6020 | 0.5705 | 2025-05-26 21:30 | 0.6024 | STOCH_EXIT | 4 | -38.82 | -0.03% | 0.5972 | 0.6205 | 0.0185 | M30 bullish FVG [0.5918, 0.5972], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 22 | 2025-06-09 13:30 | 0.6610 | 0.6316 | 2025-06-09 20:30 | 0.6614 | STOCH_EXIT | 14 | -45.71 | -0.04% | 0.6650 | 0.6783 | 0.0173 | M30 bullish FVG [0.6580, 0.6650], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 23 | 2025-07-11 08:30 | 0.6206 | 0.5841 | 2025-07-11 19:00 | 0.6184 | STOCH_EXIT | 21 | -527.25 | -0.45% | 0.6220 | 0.6420 | 0.0214 | M30 bullish FVG [0.6136, 0.6220], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 24 | 2025-07-17 16:00 | 0.6484 | 0.6152 | 2025-07-17 21:00 | 0.6476 | STOCH_EXIT | 10 | -257.74 | -0.22% | 0.6576 | 0.6679 | 0.0195 | M30 bullish FVG [0.6484, 0.6576], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 25 | 2025-07-28 18:00 | 0.7070 | 0.6768 | 2025-07-28 21:00 | 0.7106 | STOCH_EXIT | 6 | 471.27 | 0.41% | 0.7088 | 0.7247 | 0.0177 | M30 bullish FVG [0.7010, 0.7088], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 26 | 2025-07-30 20:00 | 0.6888 | 0.6510 | 2025-07-31 09:00 | 0.6902 | STOCH_EXIT | 13 | 119.36 | 0.10% | 0.6940 | 0.7110 | 0.0222 | M30 bullish FVG [0.6856, 0.6940], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 27 | 2025-08-05 11:00 | 0.7398 | 0.6985 | 2025-08-05 16:00 | 0.7402 | STOCH_EXIT | 10 | -53.04 | -0.05% | 0.7388 | 0.7641 | 0.0243 | M30 bullish FVG [0.7288, 0.7388], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 28 | 2025-09-29 13:30 | 0.7860 | 0.7445 | 2025-09-29 23:30 | 0.7914 | STOCH_EXIT | 20 | 677.86 | 0.59% | 0.7900 | 0.8104 | 0.0244 | M30 bullish FVG [0.7838, 0.7900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 29 | 2025-10-09 23:30 | 0.8552 | 0.8075 | 2025-10-10 07:00 | 0.8832 | TP | 2 | 3694.71 | 3.18% | 0.8882 | 0.8832 | 0.0280 | M30 bullish FVG [0.8776, 0.8882], stochastic crossed below 20 | FVG target 0.8832 |

## Движение капитала

- Старт: 100000.00
- Мин: 98090.97
- Макс: 120255.24
- Финал: 120255.24

Полная кривая — в `*_equity.csv`.
