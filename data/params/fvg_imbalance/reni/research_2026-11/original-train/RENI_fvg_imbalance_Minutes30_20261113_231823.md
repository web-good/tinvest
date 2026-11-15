# Бэктест RENI (Minutes30)

- Период: 2023-11-13 23:18 — 2025-11-13 23:18
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.25 |
| MaxWaitBars | 30 |
| StochPeriod | 14 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 2 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 41 |
| Выигрышных / проигрышных | 31 / 10 |
| Win rate | 75.61% |
| Gross profit / loss | 29797.68 / 15702.71 |
| Profit factor | 1.898 |
| Чистый PnL | 14094.97 (14.09%) |
| Стартовый / финальный капитал | 100000.00 / 114094.97 |
| Макс. просадка | 7218.46 (6.35%) |
| Средняя прибыль / убыток | 961.22 / 1570.27 |
| Expectancy | 343.78 |
| Лучшая / худшая сделка | 2254.11 / -4952.62 |
| Exposure | 3.35% |
| CAGR | 6.81% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-12-06 12:00 | 87.3200 | 83.7490 | 2023-12-06 18:00 | 88.5000 | STOCH_EXIT | 12 | 1244.98 | 1.25% | 89.0200 | 94.4619 | 3.5710 | M30 bullish FVG [88.0000, 89.0200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 2 | 2024-01-16 15:30 | 96.5000 | 93.5592 | 2024-01-17 11:00 | 96.8000 | STOCH_EXIT | 10 | 211.48 | 0.21% | 97.7000 | 102.3816 | 2.9408 | M30 bullish FVG [96.2600, 97.7000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 3 | 2024-01-26 18:30 | 99.2200 | 96.3954 | 2024-01-29 15:30 | 99.8800 | STOCH_EXIT | 13 | 571.66 | 0.56% | 99.7600 | 104.8691 | 2.8246 | M30 bullish FVG [98.6800, 99.7600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 4 | 2024-02-20 12:00 | 101.3800 | 98.6249 | 2024-02-20 17:00 | 102.1600 | STOCH_EXIT | 10 | 678.23 | 0.67% | 101.9000 | 106.8902 | 2.7551 | M30 bullish FVG [100.5000, 101.9000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 5 | 2024-04-04 10:30 | 102.7000 | 100.3042 | 2024-04-07 20:30 | 102.3800 | STOCH_EXIT | 74 | -418.31 | -0.41% | 104.6200 | 107.4916 | 2.3958 | M30 bullish FVG [103.9800, 104.6200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 6 | 2024-04-19 18:30 | 107.0800 | 104.7742 | 2024-04-20 16:30 | 107.9600 | STOCH_EXIT | 14 | 733.86 | 0.72% | 106.8000 | 111.6915 | 2.3058 | M30 bullish FVG [106.0000, 106.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 7 | 2024-06-05 15:30 | 102.4400 | 99.2533 | 2024-06-05 17:00 | 103.9600 | STOCH_EXIT | 3 | 1416.80 | 1.38% | 100.8800 | 108.8135 | 3.1867 | M30 bullish FVG [99.1200, 100.8800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 8 | 2024-06-20 14:30 | 99.0800 | 95.4710 | 2024-06-20 18:00 | 100.6400 | STOCH_EXIT | 7 | 1533.15 | 1.47% | 100.9600 | 106.2979 | 3.6090 | M30 bullish FVG [100.0000, 100.9600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 9 | 2024-08-07 22:00 | 93.7600 | 90.5465 | 2024-08-09 21:00 | 94.0000 | STOCH_EXIT | 56 | 163.65 | 0.16% | 93.5600 | 100.1871 | 3.2135 | M30 bullish FVG [92.0000, 93.5600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 10 | 2024-08-15 23:00 | 94.9800 | 91.6626 | 2024-08-16 17:00 | 95.7600 | STOCH_EXIT | 17 | 759.94 | 0.72% | 94.7800 | 101.6148 | 3.3174 | M30 bullish FVG [93.8800, 94.7800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 11 | 2024-08-16 23:00 | 93.5000 | 90.2281 | 2024-08-17 13:00 | 93.3600 | STOCH_EXIT | 12 | -266.11 | -0.25% | 95.3800 | 100.0438 | 3.2719 | M30 bullish FVG [94.3000, 95.3800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 12 | 2024-08-19 20:00 | 93.9400 | 90.6947 | 2024-08-20 10:00 | 95.5000 | STOCH_EXIT | 9 | 1655.77 | 1.56% | 95.2200 | 100.4307 | 3.2453 | M30 bullish FVG [94.3000, 95.2200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 13 | 2024-08-23 20:00 | 92.4000 | 89.2274 | 2024-08-24 09:30 | 94.4200 | STOCH_EXIT | 13 | 2254.11 | 2.08% | 92.7800 | 98.7453 | 3.1726 | M30 bullish FVG [91.7400, 92.7800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 14 | 2024-08-29 13:00 | 95.1000 | 91.7741 | 2024-08-29 22:30 | 95.8000 | STOCH_EXIT | 19 | 701.28 | 0.64% | 96.6400 | 101.7517 | 3.3259 | M30 bullish FVG [95.7400, 96.6400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 15 | 2024-09-27 08:00 | 99.8800 | 96.5838 | 2024-09-27 17:00 | 100.3000 | STOCH_EXIT | 18 | 355.10 | 0.32% | 97.3400 | 106.4725 | 3.2962 | M30 bullish FVG [96.3000, 97.3400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 16 | 2024-11-13 20:00 | 103.2600 | 100.3829 | 2024-11-14 07:00 | 104.2000 | STOCH_EXIT | 8 | 903.17 | 0.81% | 103.1600 | 109.0143 | 2.8771 | M30 bullish FVG [102.0200, 103.1600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 17 | 2024-11-28 10:00 | 97.6000 | 94.7796 | 2024-11-28 10:00 | 98.6000 | STOCH_EXIT | 0 | 1037.18 | 0.92% | 95.7400 | 103.2408 | 2.8204 | M30 bullish FVG [94.5000, 95.7400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 18 | 2024-11-29 23:00 | 98.7800 | 95.8896 | 2024-11-30 12:00 | 98.9600 | STOCH_EXIT | 11 | 92.49 | 0.08% | 98.1000 | 104.5607 | 2.8904 | M30 bullish FVG [95.7200, 98.1000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 19 | 2024-12-03 19:00 | 100.0000 | 96.9887 | 2024-12-03 19:00 | 96.9887 | SL | 0 | -3514.04 | -3.11% | 97.2000 | 106.0226 | 3.0113 | M30 bullish FVG [96.3600, 97.2000], stochastic crossed below 20 | FVG stop 96.9887 |
| 20 | 2024-12-18 12:30 | 94.6400 | 91.9054 | 2024-12-18 18:30 | 96.0400 | STOCH_EXIT | 12 | 1513.41 | 1.38% | 96.5400 | 100.1092 | 2.7346 | M30 bullish FVG [95.1400, 96.5400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 21 | 2024-12-19 18:00 | 95.0800 | 92.2993 | 2024-12-20 07:00 | 90.9400 | SL | 12 | -4952.62 | -4.45% | 97.1200 | 100.6414 | 2.7807 | M30 bullish FVG [96.3000, 97.1200], stochastic crossed below 20 | FVG stop 92.2993 |
| 22 | 2024-12-24 18:30 | 100.3800 | 96.8367 | 2024-12-24 23:30 | 100.9200 | STOCH_EXIT | 10 | 465.71 | 0.44% | 101.0200 | 107.4667 | 3.5433 | M30 bullish FVG [100.0800, 101.0200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 23 | 2025-01-06 21:30 | 109.3000 | 105.6630 | 2025-01-07 16:00 | 109.2800 | STOCH_EXIT | 22 | -125.41 | -0.12% | 110.5800 | 116.5741 | 3.6370 | M30 bullish FVG [109.6600, 110.5800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 24 | 2025-01-10 21:30 | 112.0000 | 108.2931 | 2025-01-12 14:00 | 112.4800 | STOCH_EXIT | 52 | 349.37 | 0.33% | 109.0400 | 119.4137 | 3.7069 | M30 bullish FVG [107.3000, 109.0400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 25 | 2025-01-23 10:30 | 115.6800 | 111.9177 | 2025-01-24 07:00 | 117.6600 | STOCH_EXIT | 27 | 1714.26 | 1.61% | 117.7000 | 123.2045 | 3.7623 | M30 bullish FVG [115.9000, 117.7000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 26 | 2025-01-27 17:00 | 118.2400 | 114.3593 | 2025-01-28 08:00 | 117.4400 | STOCH_EXIT | 16 | -844.41 | -0.78% | 120.2200 | 126.0014 | 3.8807 | M30 bullish FVG [119.2000, 120.2200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 27 | 2025-02-14 10:30 | 126.0400 | 121.8832 | 2025-02-14 11:00 | 128.4600 | STOCH_EXIT | 1 | 1948.84 | 1.82% | 127.0800 | 134.3536 | 4.1568 | M30 bullish FVG [125.0000, 127.0800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 28 | 2025-02-26 14:00 | 127.4800 | 123.1823 | 2025-02-27 13:30 | 128.2400 | STOCH_EXIT | 33 | 543.64 | 0.50% | 130.5800 | 136.0755 | 4.2977 | M30 bullish FVG [128.5000, 130.5800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 29 | 2025-02-27 21:30 | 128.6200 | 123.7392 | 2025-02-28 08:30 | 131.0200 | STOCH_EXIT | 8 | 1952.35 | 1.76% | 127.2400 | 138.3815 | 4.8808 | M30 bullish FVG [125.8800, 127.2400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 30 | 2025-03-04 12:30 | 135.6000 | 130.1623 | 2025-03-05 16:00 | 134.6600 | STOCH_EXIT | 41 | -892.36 | -0.79% | 136.8200 | 146.4755 | 5.4377 | M30 bullish FVG [135.0000, 136.8200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 31 | 2025-04-01 14:30 | 126.6200 | 121.5012 | 2025-04-02 07:00 | 121.5012 | SL | 19 | -4613.72 | -4.14% | 128.3200 | 136.8576 | 5.1188 | M30 bullish FVG [125.4600, 128.3200], stochastic crossed below 20 | FVG stop 121.5012 |
| 32 | 2025-04-10 15:00 | 117.8800 | 111.7888 | 2025-04-10 17:00 | 119.4600 | STOCH_EXIT | 4 | 1315.20 | 1.24% | 118.0200 | 130.0625 | 6.0912 | M30 bullish FVG [116.4000, 118.0200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 33 | 2025-04-23 09:30 | 120.5800 | 115.4527 | 2025-04-23 17:30 | 121.6000 | STOCH_EXIT | 16 | 800.03 | 0.75% | 123.1400 | 130.8346 | 5.1273 | M30 bullish FVG [121.6000, 123.1400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 34 | 2025-05-22 22:00 | 111.6000 | 107.8805 | 2025-05-23 10:30 | 113.1000 | STOCH_EXIT | 11 | 1346.02 | 1.24% | 112.1400 | 119.0390 | 3.7195 | M30 bullish FVG [110.2600, 112.1400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 35 | 2025-06-04 07:30 | 118.4400 | 114.5277 | 2025-06-05 07:30 | 118.5000 | STOCH_EXIT | 35 | -54.38 | -0.05% | 116.0400 | 126.2647 | 3.9123 | M30 bullish FVG [114.9800, 116.0400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 36 | 2025-06-12 04:30 | 119.3200 | 115.3678 | 2025-06-12 09:30 | 119.6600 | STOCH_EXIT | 10 | 202.87 | 0.18% | 118.5600 | 127.2244 | 3.9522 | M30 bullish FVG [116.7000, 118.5600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 37 | 2025-07-10 22:30 | 120.9000 | 117.2139 | 2025-07-11 10:00 | 121.9600 | STOCH_EXIT | 10 | 854.10 | 0.78% | 121.2800 | 128.2722 | 3.6861 | M30 bullish FVG [120.3000, 121.2800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 38 | 2025-07-15 10:30 | 123.6600 | 119.4375 | 2025-07-15 17:00 | 123.7600 | STOCH_EXIT | 13 | -21.34 | -0.02% | 128.0200 | 132.1049 | 4.2225 | M30 bullish FVG [126.8000, 128.0200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 39 | 2025-07-17 17:00 | 124.5200 | 120.0993 | 2025-07-18 07:00 | 125.1800 | STOCH_EXIT | 15 | 476.28 | 0.43% | 126.1400 | 133.3614 | 4.4207 | M30 bullish FVG [124.7000, 126.1400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 40 | 2025-08-12 10:30 | 130.8600 | 127.1369 | 2025-08-12 17:00 | 131.2600 | STOCH_EXIT | 13 | 228.60 | 0.21% | 132.6000 | 138.3062 | 3.7231 | M30 bullish FVG [131.3800, 132.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 41 | 2025-10-17 12:00 | 100.8200 | 96.7351 | 2025-10-17 17:00 | 102.5200 | STOCH_EXIT | 10 | 1774.15 | 1.58% | 99.6800 | 108.9897 | 4.0849 | M30 bullish FVG [98.0400, 99.6800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 99859.03
- Макс: 114094.97
- Финал: 114094.97

Полная кривая — в `*_equity.csv`.
