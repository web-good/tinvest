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
| StopATR | 1.7 |
| TakeProfitATR | 1 |
| HTFTrendEMA | 100 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 49 |
| Выигрышных / проигрышных | 36 / 13 |
| Win rate | 73.47% |
| Gross profit / loss | 122486.55 / 64282.85 |
| Profit factor | 1.905 |
| Чистый PnL | 58203.70 (58.20%) |
| Стартовый / финальный капитал | 100000.00 / 158203.70 |
| Макс. просадка | 15608.80 (12.62%) |
| Средняя прибыль / убыток | 3402.40 / 4944.83 |
| Expectancy | 1187.83 |
| Лучшая / худшая сделка | 11562.92 / -11771.77 |
| Exposure | 6.68% |
| CAGR | 25.74% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-04-16 15:30 | 115.5000 | 106.7739 | 2024-04-17 21:30 | 110.4000 | STOCH_EXIT | 40 | -4509.20 | -4.51% | 118.3000 | 120.6330 | 5.1330 | M30 bullish FVG [115.5000, 118.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 2 | 2024-05-15 12:30 | 110.0000 | 101.1667 | 2024-05-16 10:30 | 112.6000 | STOCH_EXIT | 24 | 2157.70 | 2.26% | 110.9000 | 115.1960 | 5.1960 | M30 bullish FVG [110.0000, 110.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 3 | 2024-05-21 11:30 | 120.6000 | 110.8790 | 2024-05-22 10:00 | 125.2000 | STOCH_EXIT | 25 | 3621.97 | 3.71% | 123.1000 | 126.3182 | 5.7182 | M30 bullish FVG [121.8000, 123.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 4 | 2024-05-24 11:00 | 117.4000 | 107.0505 | 2024-05-27 10:30 | 123.4880 | TP | 27 | 5144.00 | 5.08% | 118.0000 | 123.4880 | 6.0880 | M30 bullish FVG [117.2000, 118.0000], stochastic crossed below 20 | FVG target 123.4880 |
| 5 | 2024-05-27 21:30 | 115.7000 | 105.3611 | 2024-05-28 10:30 | 121.5000 | STOCH_EXIT | 6 | 5221.21 | 4.91% | 119.3000 | 121.7817 | 6.0817 | M30 bullish FVG [116.3000, 119.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 6 | 2024-06-11 17:30 | 118.9000 | 106.4628 | 2024-06-13 10:00 | 106.4628 | SL | 13 | -11771.77 | -10.55% | 120.5000 | 126.2160 | 7.3160 | M30 bullish FVG [119.4000, 120.5000], stochastic crossed below 20 | FVG stop 106.4628 |
| 7 | 2024-06-14 16:00 | 120.4000 | 106.1018 | 2024-06-17 18:00 | 123.6000 | STOCH_EXIT | 32 | 2551.66 | 2.56% | 122.4000 | 128.8107 | 8.4107 | M30 bullish FVG [121.0000, 122.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 8 | 2024-06-18 18:00 | 124.0000 | 109.8968 | 2024-06-19 11:30 | 129.0000 | STOCH_EXIT | 15 | 4020.64 | 3.93% | 125.6000 | 132.2960 | 8.2960 | M30 bullish FVG [123.6000, 125.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 9 | 2024-06-26 14:00 | 143.5000 | 126.0915 | 2024-06-27 15:30 | 141.4000 | STOCH_EXIT | 31 | -1661.66 | -1.56% | 144.2000 | 153.7403 | 10.2403 | M30 bullish FVG [143.3000, 144.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 10 | 2024-07-17 18:00 | 114.6000 | 97.8657 | 2024-07-18 18:00 | 116.0000 | STOCH_EXIT | 28 | 1172.93 | 1.12% | 117.3000 | 124.4437 | 9.8437 | M30 bullish FVG [115.0000, 117.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 11 | 2024-07-19 23:00 | 115.8000 | 99.4723 | 2024-07-22 10:30 | 123.1000 | STOCH_EXIT | 3 | 6563.02 | 6.20% | 117.0000 | 125.4045 | 9.6045 | M30 bullish FVG [115.7000, 117.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 12 | 2024-07-29 17:00 | 131.2000 | 112.9079 | 2024-07-30 15:30 | 137.5000 | STOCH_EXIT | 25 | 5283.96 | 4.70% | 141.6000 | 141.9601 | 10.7601 | M30 bullish FVG [140.5000, 141.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 13 | 2024-08-01 15:30 | 143.0000 | 121.4374 | 2024-08-02 11:30 | 150.0000 | STOCH_EXIT | 20 | 5640.43 | 4.79% | 146.3000 | 155.6839 | 12.6839 | M30 bullish FVG [144.0000, 146.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 14 | 2024-08-02 21:00 | 141.4000 | 119.7747 | 2024-08-06 22:00 | 128.5000 | STOCH_EXIT | 58 | -11366.48 | -9.21% | 143.0000 | 154.1208 | 12.7208 | M30 bullish FVG [141.7000, 143.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 15 | 2024-08-19 20:30 | 125.6000 | 109.2924 | 2024-08-20 09:30 | 127.1000 | STOCH_EXIT | 12 | 1223.92 | 1.09% | 125.9000 | 135.1927 | 9.5927 | M30 bullish FVG [124.8000, 125.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 16 | 2024-09-23 19:00 | 119.8000 | 108.4708 | 2024-09-24 07:00 | 121.8000 | STOCH_EXIT | 10 | 1775.84 | 1.57% | 121.2000 | 126.4642 | 6.6642 | M30 bullish FVG [119.5000, 121.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 17 | 2024-09-25 11:00 | 120.5000 | 109.4556 | 2024-09-25 14:00 | 126.9967 | TP | 6 | 6079.81 | 5.29% | 121.7000 | 126.9967 | 6.4967 | M30 bullish FVG [120.8000, 121.7000], stochastic crossed below 20 | FVG target 126.9967 |
| 18 | 2024-09-26 21:30 | 129.6000 | 118.1545 | 2024-09-27 10:00 | 132.1000 | STOCH_EXIT | 11 | 2212.79 | 1.83% | 127.1000 | 136.3327 | 6.7327 | M30 bullish FVG [125.3000, 127.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 19 | 2024-10-02 18:00 | 129.0000 | 117.1928 | 2024-10-03 09:00 | 130.5000 | STOCH_EXIT | 16 | 1308.59 | 1.06% | 131.6000 | 135.9454 | 6.9454 | M30 bullish FVG [129.6000, 131.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 20 | 2024-10-04 13:00 | 136.9000 | 124.0999 | 2024-10-07 21:30 | 137.9000 | STOCH_EXIT | 51 | 784.97 | 0.63% | 138.4000 | 144.4295 | 7.5295 | M30 bullish FVG [136.3000, 138.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 21 | 2024-10-10 12:00 | 138.3000 | 124.7070 | 2024-10-10 15:30 | 144.0000 | STOCH_EXIT | 7 | 5036.32 | 4.02% | 139.6000 | 146.2959 | 7.9959 | M30 bullish FVG [138.8000, 139.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 22 | 2024-10-11 10:30 | 140.0000 | 126.4793 | 2024-10-15 15:00 | 130.3000 | STOCH_EXIT | 77 | -9156.52 | -7.02% | 141.4000 | 147.9533 | 7.9533 | M30 bullish FVG [138.6000, 141.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 23 | 2024-10-15 21:30 | 128.9000 | 115.3701 | 2024-10-16 09:00 | 130.0000 | STOCH_EXIT | 9 | 912.32 | 0.75% | 129.7000 | 136.8587 | 7.9587 | M30 bullish FVG [128.7000, 129.7000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 24 | 2024-10-17 18:00 | 127.6000 | 114.6798 | 2024-10-21 09:30 | 125.2000 | STOCH_EXIT | 51 | -2417.76 | -1.98% | 129.7000 | 135.2001 | 7.6001 | M30 bullish FVG [127.7000, 129.7000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 25 | 2024-11-07 16:00 | 130.7000 | 117.6558 | 2024-11-07 23:00 | 133.5000 | STOCH_EXIT | 14 | 2443.80 | 2.04% | 129.6000 | 138.3730 | 7.6730 | M30 bullish FVG [127.5000, 129.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 26 | 2024-11-12 09:00 | 128.9000 | 116.1656 | 2024-11-13 12:00 | 127.2000 | STOCH_EXIT | 40 | -1732.99 | -1.42% | 130.8000 | 136.3908 | 7.4908 | M30 bullish FVG [129.8000, 130.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 27 | 2024-11-13 22:00 | 127.3000 | 114.9531 | 2024-11-15 07:00 | 129.5000 | STOCH_EXIT | 38 | 1959.73 | 1.63% | 128.5000 | 134.5629 | 7.2629 | M30 bullish FVG [127.2000, 128.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 28 | 2025-01-13 21:30 | 129.6000 | 120.0240 | 2025-01-14 11:30 | 132.5000 | STOCH_EXIT | 14 | 2613.89 | 2.14% | 129.5000 | 135.2329 | 5.6329 | M30 bullish FVG [128.2000, 129.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 29 | 2025-01-27 23:00 | 138.4000 | 127.6383 | 2025-01-28 14:00 | 144.7304 | TP | 16 | 5588.54 | 4.47% | 142.8000 | 144.7304 | 6.3304 | M30 bullish FVG [141.5000, 142.8000], stochastic crossed below 20 | FVG target 144.7304 |
| 30 | 2025-02-04 14:30 | 148.9000 | 136.8660 | 2025-02-05 14:00 | 146.0000 | STOCH_EXIT | 33 | -2672.61 | -2.05% | 149.3000 | 155.9788 | 7.0788 | M30 bullish FVG [148.5000, 149.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 31 | 2025-02-06 14:30 | 151.0000 | 138.1587 | 2025-02-07 09:30 | 152.3000 | STOCH_EXIT | 24 | 972.65 | 0.76% | 153.4000 | 158.5537 | 7.5537 | M30 bullish FVG [152.2000, 153.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 32 | 2025-02-10 23:00 | 151.0000 | 138.4593 | 2025-02-11 16:00 | 154.4000 | STOCH_EXIT | 20 | 2769.95 | 2.15% | 154.9000 | 158.3769 | 7.3769 | M30 bullish FVG [154.1000, 154.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 33 | 2025-02-18 22:00 | 255.7000 | 231.4194 | 2025-02-19 07:00 | 269.9827 | TP | 4 | 7220.24 | 5.48% | 269.0000 | 269.9827 | 14.2827 | M30 bullish FVG [264.5000, 269.0000], stochastic crossed below 20 | FVG target 269.9827 |
| 34 | 2025-02-27 13:00 | 283.2000 | 242.5820 | 2025-02-27 16:00 | 307.0929 | TP | 6 | 11562.92 | 8.33% | 292.2000 | 307.0929 | 23.8929 | M30 bullish FVG [286.7000, 292.2000], stochastic crossed below 20 | FVG target 307.0929 |
| 35 | 2025-02-27 23:30 | 284.8000 | 244.1820 | 2025-02-28 14:00 | 292.1000 | STOCH_EXIT | 15 | 3702.10 | 2.46% | 291.1000 | 308.6929 | 23.8929 | M30 bullish FVG [286.0000, 291.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 36 | 2025-03-18 21:00 | 307.3000 | 269.2064 | 2025-03-19 14:00 | 312.4000 | STOCH_EXIT | 20 | 2399.87 | 1.56% | 329.0000 | 329.7080 | 22.4080 | M30 bullish FVG [325.0000, 329.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 37 | 2025-03-20 10:30 | 313.2000 | 273.7361 | 2025-03-23 10:30 | 310.6000 | STOCH_EXIT | 90 | -1453.04 | -0.93% | 316.0000 | 336.4140 | 23.2140 | M30 bullish FVG [313.0000, 316.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 38 | 2025-04-01 21:00 | 253.6000 | 211.8274 | 2025-04-02 15:30 | 247.5000 | STOCH_EXIT | 23 | -3880.19 | -2.50% | 253.0000 | 278.1721 | 24.5721 | M30 bullish FVG [250.0000, 253.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 39 | 2025-04-03 16:30 | 252.8000 | 209.4943 | 2025-04-03 23:30 | 260.5000 | STOCH_EXIT | 14 | 4451.12 | 2.94% | 251.3000 | 278.2739 | 25.4739 | M30 bullish FVG [248.7000, 251.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 40 | 2025-04-23 10:30 | 271.1000 | 229.1007 | 2025-04-23 18:00 | 283.5000 | STOCH_EXIT | 15 | 6958.43 | 4.47% | 285.2000 | 295.8055 | 24.7055 | M30 bullish FVG [274.0000, 285.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 41 | 2025-05-05 09:30 | 244.7000 | 207.2660 | 2025-05-06 10:00 | 239.8000 | STOCH_EXIT | 35 | -3414.45 | -2.10% | 243.6000 | 266.7200 | 22.0200 | M30 bullish FVG [241.1000, 243.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 42 | 2025-05-13 11:30 | 267.7000 | 233.2517 | 2025-05-13 18:30 | 272.4000 | STOCH_EXIT | 14 | 2631.39 | 1.65% | 267.1000 | 287.9637 | 20.2637 | M30 bullish FVG [265.1000, 267.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 43 | 2025-05-23 10:00 | 251.7000 | 222.4342 | 2025-05-23 12:00 | 254.7000 | STOCH_EXIT | 4 | 1766.19 | 1.09% | 256.0000 | 268.9152 | 17.2152 | M30 bullish FVG [253.7000, 256.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 44 | 2025-08-15 13:00 | 290.1000 | 264.0621 | 2025-08-15 18:30 | 292.4000 | STOCH_EXIT | 11 | 1132.93 | 0.69% | 284.7000 | 305.4164 | 15.3164 | M30 bullish FVG [282.6000, 284.7000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 45 | 2025-08-19 15:30 | 284.7000 | 256.7942 | 2025-08-21 14:00 | 271.4000 | STOCH_EXIT | 65 | -7848.11 | -4.77% | 282.0000 | 301.1152 | 16.4152 | M30 bullish FVG [280.0000, 282.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 46 | 2025-08-21 18:00 | 261.8000 | 234.5196 | 2025-08-22 16:30 | 262.9000 | STOCH_EXIT | 31 | 501.75 | 0.32% | 268.1000 | 277.8473 | 16.0473 | M30 bullish FVG [263.2000, 268.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 47 | 2025-08-28 08:30 | 266.4000 | 239.2932 | 2025-08-29 15:30 | 262.6000 | STOCH_EXIT | 48 | -2398.05 | -1.52% | 257.0000 | 282.3452 | 15.9452 | M30 bullish FVG [255.0000, 257.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 48 | 2025-09-05 18:30 | 260.4000 | 236.3457 | 2025-09-06 20:30 | 261.3000 | STOCH_EXIT | 32 | 380.29 | 0.25% | 262.6000 | 274.5496 | 14.1496 | M30 bullish FVG [260.9000, 262.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 49 | 2025-09-18 13:00 | 243.2000 | 223.0364 | 2025-09-18 15:30 | 247.7000 | STOCH_EXIT | 5 | 2718.66 | 1.75% | 241.6000 | 255.0610 | 11.8610 | M30 bullish FVG [240.4000, 241.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 93116.55
- Макс: 164849.16
- Финал: 158203.70

Полная кривая — в `*_equity.csv`.
