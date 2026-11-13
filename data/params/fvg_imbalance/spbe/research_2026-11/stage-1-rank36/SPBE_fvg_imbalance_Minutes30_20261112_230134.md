# Бэктест SPBE (Minutes30)

- Период: 2023-11-12 23:01 — 2025-11-12 23:01
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.1 |
| MaxWaitBars | 30 |
| StochPeriod | 21 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 2 |
| HTFTrendEMA | 100 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 48 |
| Выигрышных / проигрышных | 34 / 14 |
| Win rate | 70.83% |
| Gross profit / loss | 95389.92 / 63643.73 |
| Profit factor | 1.499 |
| Чистый PnL | 31746.19 (31.75%) |
| Стартовый / финальный капитал | 100000.00 / 131746.19 |
| Макс. просадка | 12834.86 (9.24%) |
| Средняя прибыль / убыток | 2805.59 / 4545.98 |
| Expectancy | 661.38 |
| Лучшая / худшая сделка | 6132.19 / -10477.49 |
| Exposure | 6.22% |
| CAGR | 14.76% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-04-16 15:30 | 115.5000 | 110.3670 | 2024-04-17 10:00 | 110.3670 | SL | 17 | -4537.75 | -4.54% | 118.3000 | 125.7660 | 5.1330 | M30 bullish FVG [115.5000, 118.3000], stochastic crossed below 20 | FVG stop 110.3670 |
| 2 | 2024-05-15 12:30 | 110.0000 | 104.8040 | 2024-05-16 10:30 | 112.6000 | STOCH_EXIT | 24 | 2157.70 | 2.26% | 110.9000 | 120.3921 | 5.1960 | M30 bullish FVG [110.0000, 110.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 3 | 2024-05-21 11:30 | 120.6000 | 114.8818 | 2024-05-22 10:00 | 125.2000 | STOCH_EXIT | 25 | 3621.97 | 3.71% | 123.1000 | 132.0365 | 5.7182 | M30 bullish FVG [121.8000, 123.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 4 | 2024-05-24 11:00 | 117.4000 | 111.3120 | 2024-05-27 10:00 | 111.3120 | SL | 26 | -5340.20 | -5.28% | 118.0000 | 129.5759 | 6.0880 | M30 bullish FVG [117.2000, 118.0000], stochastic crossed below 20 | FVG stop 111.3120 |
| 5 | 2024-05-27 21:30 | 115.7000 | 109.6183 | 2024-05-28 10:30 | 121.5000 | STOCH_EXIT | 6 | 4704.20 | 4.91% | 119.3000 | 127.8634 | 6.0817 | M30 bullish FVG [116.3000, 119.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 6 | 2024-06-11 17:30 | 118.9000 | 111.5840 | 2024-06-13 10:00 | 110.4000 | SL | 13 | -7279.38 | -7.24% | 120.5000 | 133.5320 | 7.3160 | M30 bullish FVG [119.4000, 120.5000], stochastic crossed below 20 | FVG stop 111.5840 |
| 7 | 2024-06-14 16:00 | 120.4000 | 111.9893 | 2024-06-17 18:00 | 123.6000 | STOCH_EXIT | 32 | 2382.37 | 2.56% | 122.4000 | 137.2215 | 8.4107 | M30 bullish FVG [121.0000, 122.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 8 | 2024-06-18 18:00 | 124.0000 | 115.7040 | 2024-06-19 11:30 | 129.0000 | STOCH_EXIT | 15 | 3757.47 | 3.93% | 125.6000 | 140.5920 | 8.2960 | M30 bullish FVG [123.6000, 125.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 9 | 2024-06-26 14:00 | 143.5000 | 133.2597 | 2024-06-27 15:30 | 141.4000 | STOCH_EXIT | 31 | -1551.78 | -1.56% | 144.2000 | 163.9806 | 10.2403 | M30 bullish FVG [143.3000, 144.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 10 | 2024-07-17 18:00 | 114.6000 | 104.7563 | 2024-07-18 18:00 | 116.0000 | STOCH_EXIT | 28 | 1095.85 | 1.12% | 117.3000 | 134.2874 | 9.8437 | M30 bullish FVG [115.0000, 117.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 11 | 2024-07-19 23:00 | 115.8000 | 106.1955 | 2024-07-22 10:30 | 123.1000 | STOCH_EXIT | 3 | 6132.19 | 6.20% | 117.0000 | 135.0090 | 9.6045 | M30 bullish FVG [115.7000, 117.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 12 | 2024-07-29 17:00 | 131.2000 | 120.4399 | 2024-07-30 15:30 | 137.5000 | STOCH_EXIT | 25 | 4932.52 | 4.70% | 141.6000 | 152.7201 | 10.7601 | M30 bullish FVG [140.5000, 141.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 13 | 2024-08-01 15:30 | 143.0000 | 130.3161 | 2024-08-02 11:30 | 150.0000 | STOCH_EXIT | 20 | 5270.34 | 4.79% | 146.3000 | 168.3678 | 12.6839 | M30 bullish FVG [144.0000, 146.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 14 | 2024-08-02 21:00 | 141.4000 | 128.6792 | 2024-08-05 10:00 | 128.6792 | SL | 6 | -10477.49 | -9.09% | 143.0000 | 166.8415 | 12.7208 | M30 bullish FVG [141.7000, 143.0000], stochastic crossed below 20 | FVG stop 128.6792 |
| 15 | 2024-08-19 20:30 | 125.6000 | 116.0073 | 2024-08-20 09:30 | 127.1000 | STOCH_EXIT | 12 | 1145.62 | 1.09% | 125.9000 | 144.7854 | 9.5927 | M30 bullish FVG [124.8000, 125.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 16 | 2024-09-23 19:00 | 119.8000 | 113.1358 | 2024-09-24 07:00 | 121.8000 | STOCH_EXIT | 10 | 1661.21 | 1.57% | 121.2000 | 133.1285 | 6.6642 | M30 bullish FVG [119.5000, 121.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 17 | 2024-09-25 11:00 | 120.5000 | 114.0033 | 2024-09-25 14:00 | 127.0000 | STOCH_EXIT | 6 | 5693.99 | 5.29% | 121.7000 | 133.4934 | 6.4967 | M30 bullish FVG [120.8000, 121.7000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 18 | 2024-09-26 21:30 | 129.6000 | 122.8673 | 2024-09-27 10:00 | 132.1000 | STOCH_EXIT | 11 | 2070.64 | 1.83% | 127.1000 | 143.0653 | 6.7327 | M30 bullish FVG [125.3000, 127.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 19 | 2024-10-02 18:00 | 129.0000 | 122.0546 | 2024-10-03 09:00 | 130.5000 | STOCH_EXIT | 16 | 1225.00 | 1.06% | 131.6000 | 142.8908 | 6.9454 | M30 bullish FVG [129.6000, 131.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 20 | 2024-10-04 13:00 | 136.9000 | 129.3705 | 2024-10-07 21:30 | 137.9000 | STOCH_EXIT | 51 | 734.07 | 0.63% | 138.4000 | 151.9589 | 7.5295 | M30 bullish FVG [136.3000, 138.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 21 | 2024-10-10 12:00 | 138.3000 | 130.3041 | 2024-10-10 15:30 | 144.0000 | STOCH_EXIT | 7 | 4713.90 | 4.02% | 139.6000 | 154.2918 | 7.9959 | M30 bullish FVG [138.8000, 139.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 22 | 2024-10-11 10:30 | 140.0000 | 132.0467 | 2024-10-14 10:00 | 132.0467 | SL | 33 | -7045.82 | -5.78% | 141.4000 | 155.9067 | 7.9533 | M30 bullish FVG [138.6000, 141.4000], stochastic crossed below 20 | FVG stop 132.0467 |
| 23 | 2024-10-15 21:30 | 128.9000 | 120.9413 | 2024-10-16 09:00 | 130.0000 | STOCH_EXIT | 9 | 865.73 | 0.75% | 129.7000 | 144.8175 | 7.9587 | M30 bullish FVG [128.7000, 129.7000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 24 | 2024-10-17 18:00 | 127.6000 | 119.9999 | 2024-10-21 09:30 | 125.2000 | STOCH_EXIT | 51 | -2293.97 | -1.98% | 129.7000 | 142.8003 | 7.6001 | M30 bullish FVG [127.7000, 129.7000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 25 | 2024-11-07 16:00 | 130.7000 | 123.0270 | 2024-11-07 23:00 | 133.5000 | STOCH_EXIT | 14 | 2318.41 | 2.04% | 129.6000 | 146.0461 | 7.6730 | M30 bullish FVG [127.5000, 129.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 26 | 2024-11-12 09:00 | 128.9000 | 121.4092 | 2024-11-13 12:00 | 127.2000 | STOCH_EXIT | 40 | -1643.42 | -1.42% | 130.8000 | 143.8816 | 7.4908 | M30 bullish FVG [129.8000, 130.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 27 | 2024-11-13 22:00 | 127.3000 | 120.0371 | 2024-11-15 07:00 | 129.5000 | STOCH_EXIT | 38 | 1858.23 | 1.63% | 128.5000 | 141.8258 | 7.2629 | M30 bullish FVG [127.2000, 128.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 28 | 2025-01-13 21:30 | 129.6000 | 123.9671 | 2025-01-14 11:30 | 132.5000 | STOCH_EXIT | 14 | 2478.21 | 2.14% | 129.5000 | 140.8659 | 5.6329 | M30 bullish FVG [128.2000, 129.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 29 | 2025-01-27 23:00 | 138.4000 | 132.0696 | 2025-01-28 18:00 | 143.9000 | STOCH_EXIT | 24 | 4587.18 | 3.87% | 142.8000 | 151.0609 | 6.3304 | M30 bullish FVG [141.5000, 142.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 30 | 2025-02-04 14:30 | 148.9000 | 141.8212 | 2025-02-05 12:00 | 141.8212 | SL | 29 | -5974.40 | -4.85% | 149.3000 | 163.0576 | 7.0788 | M30 bullish FVG [148.5000, 149.3000], stochastic crossed below 20 | FVG stop 141.8212 |
| 31 | 2025-02-06 14:30 | 151.0000 | 143.4463 | 2025-02-07 09:30 | 152.3000 | STOCH_EXIT | 24 | 891.12 | 0.76% | 153.4000 | 166.1074 | 7.5537 | M30 bullish FVG [152.2000, 153.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 32 | 2025-02-10 23:00 | 151.0000 | 143.6231 | 2025-02-11 16:00 | 154.4000 | STOCH_EXIT | 20 | 2539.39 | 2.15% | 154.9000 | 165.7538 | 7.3769 | M30 bullish FVG [154.1000, 154.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 33 | 2025-02-18 22:00 | 255.7000 | 241.4173 | 2025-02-19 21:30 | 267.5000 | STOCH_EXIT | 33 | 5434.59 | 4.51% | 269.0000 | 284.2654 | 14.2827 | M30 bullish FVG [264.5000, 269.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 34 | 2025-02-27 13:00 | 283.2000 | 259.3071 | 2025-02-28 14:00 | 292.1000 | STOCH_EXIT | 36 | 3832.50 | 3.04% | 292.2000 | 330.9859 | 23.8929 | M30 bullish FVG [286.7000, 292.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 35 | 2025-03-18 21:00 | 307.3000 | 284.8920 | 2025-03-19 14:00 | 312.4000 | STOCH_EXIT | 20 | 2021.44 | 1.56% | 329.0000 | 352.1160 | 22.4080 | M30 bullish FVG [325.0000, 329.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 36 | 2025-03-20 10:30 | 313.2000 | 289.9860 | 2025-03-23 10:30 | 310.6000 | STOCH_EXIT | 90 | -1225.91 | -0.93% | 316.0000 | 359.6281 | 23.2140 | M30 bullish FVG [313.0000, 316.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 37 | 2025-04-01 21:00 | 253.6000 | 229.0279 | 2025-04-02 15:30 | 247.5000 | STOCH_EXIT | 23 | -3270.53 | -2.50% | 253.0000 | 302.7443 | 24.5721 | M30 bullish FVG [250.0000, 253.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 38 | 2025-04-03 16:30 | 252.8000 | 227.3261 | 2025-04-03 23:30 | 260.5000 | STOCH_EXIT | 14 | 3751.45 | 2.94% | 251.3000 | 303.7479 | 25.4739 | M30 bullish FVG [248.7000, 251.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 39 | 2025-04-23 10:30 | 271.1000 | 246.3945 | 2025-04-23 18:00 | 283.5000 | STOCH_EXIT | 15 | 5855.26 | 4.47% | 285.2000 | 320.5110 | 24.7055 | M30 bullish FVG [274.0000, 285.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 40 | 2025-05-05 09:30 | 244.7000 | 222.6800 | 2025-05-06 10:00 | 239.8000 | STOCH_EXIT | 35 | -2874.52 | -2.10% | 243.6000 | 288.7400 | 22.0200 | M30 bullish FVG [241.1000, 243.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 41 | 2025-05-13 11:30 | 267.7000 | 247.4363 | 2025-05-13 18:30 | 272.4000 | STOCH_EXIT | 14 | 2219.40 | 1.65% | 267.1000 | 308.2275 | 20.2637 | M30 bullish FVG [265.1000, 267.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 42 | 2025-05-23 10:00 | 251.7000 | 234.4848 | 2025-05-23 12:00 | 254.7000 | STOCH_EXIT | 4 | 1486.02 | 1.09% | 256.0000 | 286.1303 | 17.2152 | M30 bullish FVG [253.7000, 256.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 43 | 2025-08-15 13:00 | 290.1000 | 274.7836 | 2025-08-15 18:30 | 292.4000 | STOCH_EXIT | 11 | 954.16 | 0.69% | 284.7000 | 320.7328 | 15.3164 | M30 bullish FVG [282.6000, 284.7000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 44 | 2025-08-19 15:30 | 284.7000 | 268.2848 | 2025-08-21 07:00 | 268.2848 | SL | 51 | -8128.84 | -5.86% | 282.0000 | 317.5303 | 16.4152 | M30 bullish FVG [280.0000, 282.0000], stochastic crossed below 20 | FVG stop 268.2848 |
| 45 | 2025-08-21 18:00 | 261.8000 | 245.7527 | 2025-08-22 16:30 | 262.9000 | STOCH_EXIT | 31 | 417.99 | 0.32% | 268.1000 | 293.8945 | 16.0473 | M30 bullish FVG [263.2000, 268.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 46 | 2025-08-28 08:30 | 266.4000 | 250.4548 | 2025-08-29 15:30 | 262.6000 | STOCH_EXIT | 48 | -1999.73 | -1.52% | 257.0000 | 298.2904 | 15.9452 | M30 bullish FVG [255.0000, 257.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 47 | 2025-09-05 18:30 | 260.4000 | 246.2504 | 2025-09-06 20:30 | 261.3000 | STOCH_EXIT | 32 | 316.38 | 0.25% | 262.6000 | 288.6992 | 14.1496 | M30 bullish FVG [260.9000, 262.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 48 | 2025-09-18 13:00 | 243.2000 | 231.3390 | 2025-09-18 15:30 | 247.7000 | STOCH_EXIT | 5 | 2263.42 | 1.75% | 241.6000 | 266.9219 | 11.8610 | M30 bullish FVG [240.4000, 241.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 92970.36
- Макс: 138876.98
- Финал: 131746.19

Полная кривая — в `*_equity.csv`.
