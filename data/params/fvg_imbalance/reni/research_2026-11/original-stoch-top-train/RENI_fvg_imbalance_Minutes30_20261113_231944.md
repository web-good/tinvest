# Бэктест RENI (Minutes30)

- Период: 2023-11-13 23:19 — 2025-11-13 23:19
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
| StopATR | 1.7 |
| TakeProfitATR | 1 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 41 |
| Выигрышных / проигрышных | 33 / 8 |
| Win rate | 80.49% |
| Gross profit / loss | 37260.48 / 8749.24 |
| Profit factor | 4.259 |
| Чистый PnL | 28511.24 (28.51%) |
| Стартовый / финальный капитал | 100000.00 / 128511.24 |
| Макс. просадка | 7012.27 (6.01%) |
| Средняя прибыль / убыток | 1129.11 / 1093.65 |
| Expectancy | 695.40 |
| Лучшая / худшая сделка | 3221.08 / -3779.77 |
| Exposure | 3.38% |
| CAGR | 13.34% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-12-06 12:00 | 87.3200 | 81.2494 | 2023-12-06 18:00 | 88.5000 | STOCH_EXIT | 12 | 1244.98 | 1.25% | 89.0200 | 90.8910 | 3.5710 | M30 bullish FVG [88.0000, 89.0200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 2 | 2024-01-16 15:30 | 96.5000 | 91.5007 | 2024-01-17 11:00 | 96.8000 | STOCH_EXIT | 10 | 211.48 | 0.21% | 97.7000 | 99.4408 | 2.9408 | M30 bullish FVG [96.2600, 97.7000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 3 | 2024-01-26 18:30 | 99.2200 | 94.4183 | 2024-01-29 15:30 | 99.8800 | STOCH_EXIT | 13 | 571.66 | 0.56% | 99.7600 | 102.0446 | 2.8246 | M30 bullish FVG [98.6800, 99.7600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 4 | 2024-02-20 12:00 | 101.3800 | 96.6963 | 2024-02-20 17:00 | 102.1600 | STOCH_EXIT | 10 | 678.23 | 0.67% | 101.9000 | 104.1351 | 2.7551 | M30 bullish FVG [100.5000, 101.9000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 5 | 2024-04-04 10:30 | 102.7000 | 98.6272 | 2024-04-06 10:30 | 105.0958 | TP | 36 | 2268.97 | 2.23% | 104.6200 | 105.0958 | 2.3958 | M30 bullish FVG [103.9800, 104.6200], stochastic crossed below 20 | FVG target 105.0958 |
| 6 | 2024-04-19 18:30 | 107.0800 | 103.1602 | 2024-04-20 16:30 | 107.9600 | STOCH_EXIT | 14 | 749.31 | 0.72% | 106.8000 | 109.3858 | 2.3058 | M30 bullish FVG [106.0000, 106.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 7 | 2024-06-05 15:30 | 102.4400 | 97.0226 | 2024-06-05 17:00 | 103.9600 | STOCH_EXIT | 3 | 1459.30 | 1.38% | 100.8800 | 105.6267 | 3.1867 | M30 bullish FVG [99.1200, 100.8800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 8 | 2024-06-20 14:30 | 99.0800 | 92.9448 | 2024-06-20 18:00 | 100.6400 | STOCH_EXIT | 7 | 1576.95 | 1.47% | 100.9600 | 102.6890 | 3.6090 | M30 bullish FVG [100.0000, 100.9600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 9 | 2024-08-07 22:00 | 93.7600 | 88.2970 | 2024-08-09 21:00 | 94.0000 | STOCH_EXIT | 56 | 168.04 | 0.16% | 93.5600 | 96.9735 | 3.2135 | M30 bullish FVG [92.0000, 93.5600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 10 | 2024-08-15 23:00 | 94.9800 | 89.3404 | 2024-08-16 17:00 | 95.7600 | STOCH_EXIT | 17 | 780.48 | 0.72% | 94.7800 | 98.2974 | 3.3174 | M30 bullish FVG [93.8800, 94.7800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 11 | 2024-08-16 23:00 | 93.5000 | 87.9378 | 2024-08-17 13:00 | 93.3600 | STOCH_EXIT | 12 | -273.11 | -0.25% | 95.3800 | 96.7719 | 3.2719 | M30 bullish FVG [94.3000, 95.3800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 12 | 2024-08-19 20:00 | 93.9400 | 88.4229 | 2024-08-20 10:00 | 95.5000 | STOCH_EXIT | 9 | 1699.72 | 1.56% | 95.2200 | 97.1853 | 3.2453 | M30 bullish FVG [94.3000, 95.2200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 13 | 2024-08-23 20:00 | 92.4000 | 87.0065 | 2024-08-24 09:30 | 94.4200 | STOCH_EXIT | 13 | 2311.91 | 2.08% | 92.7800 | 95.5726 | 3.1726 | M30 bullish FVG [91.7400, 92.7800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 14 | 2024-08-29 13:00 | 95.1000 | 89.4460 | 2024-08-29 22:30 | 95.8000 | STOCH_EXIT | 19 | 719.41 | 0.64% | 96.6400 | 98.4259 | 3.3259 | M30 bullish FVG [95.7400, 96.6400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 15 | 2024-09-27 08:00 | 99.8800 | 94.2764 | 2024-09-27 17:00 | 100.3000 | STOCH_EXIT | 18 | 364.70 | 0.32% | 97.3400 | 103.1762 | 3.2962 | M30 bullish FVG [96.3000, 97.3400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 16 | 2024-11-13 20:00 | 103.2600 | 98.3689 | 2024-11-14 07:00 | 104.2000 | STOCH_EXIT | 8 | 919.90 | 0.81% | 103.1600 | 106.1371 | 2.8771 | M30 bullish FVG [102.0200, 103.1600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 17 | 2024-11-28 10:00 | 97.6000 | 92.8053 | 2024-11-28 10:00 | 98.6000 | STOCH_EXIT | 0 | 1064.24 | 0.92% | 95.7400 | 100.4204 | 2.8204 | M30 bullish FVG [94.5000, 95.7400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 18 | 2024-11-29 23:00 | 98.7800 | 93.8664 | 2024-11-30 12:00 | 98.9600 | STOCH_EXIT | 11 | 94.92 | 0.08% | 98.1000 | 101.6704 | 2.8904 | M30 bullish FVG [95.7200, 98.1000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 19 | 2024-12-03 19:00 | 100.0000 | 94.8808 | 2024-12-04 11:00 | 96.8400 | STOCH_EXIT | 18 | -3779.77 | -3.26% | 97.2000 | 103.0113 | 3.0113 | M30 bullish FVG [96.3600, 97.2000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 20 | 2024-12-18 12:30 | 94.6400 | 89.9912 | 2024-12-18 18:30 | 96.0400 | STOCH_EXIT | 12 | 1552.55 | 1.38% | 96.5400 | 97.3746 | 2.7346 | M30 bullish FVG [95.1400, 96.5400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 21 | 2024-12-19 18:00 | 95.0800 | 90.3528 | 2024-12-20 13:30 | 97.8607 | TP | 25 | 3221.08 | 2.82% | 97.1200 | 97.8607 | 2.7807 | M30 bullish FVG [96.3000, 97.1200], stochastic crossed below 20 | FVG target 97.8607 |
| 22 | 2024-12-24 18:30 | 100.3800 | 94.3563 | 2024-12-24 23:30 | 100.9200 | STOCH_EXIT | 10 | 514.04 | 0.44% | 101.0200 | 103.9233 | 3.5433 | M30 bullish FVG [100.0800, 101.0200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 23 | 2025-01-06 21:30 | 109.3000 | 103.1170 | 2025-01-07 16:00 | 109.2800 | STOCH_EXIT | 22 | -139.63 | -0.12% | 110.5800 | 112.9370 | 3.6370 | M30 bullish FVG [109.6600, 110.5800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 24 | 2025-01-10 21:30 | 112.0000 | 105.6983 | 2025-01-12 14:00 | 112.4800 | STOCH_EXIT | 52 | 386.15 | 0.33% | 109.0400 | 115.7069 | 3.7069 | M30 bullish FVG [107.3000, 109.0400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 25 | 2025-01-23 10:30 | 115.6800 | 109.2842 | 2025-01-24 07:00 | 117.6600 | STOCH_EXIT | 27 | 1900.60 | 1.61% | 117.7000 | 119.4423 | 3.7623 | M30 bullish FVG [115.9000, 117.7000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 26 | 2025-01-27 17:00 | 118.2400 | 111.6428 | 2025-01-28 08:00 | 117.4400 | STOCH_EXIT | 16 | -927.02 | -0.78% | 120.2200 | 122.1207 | 3.8807 | M30 bullish FVG [119.2000, 120.2200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 27 | 2025-02-14 10:30 | 126.0400 | 118.9734 | 2025-02-14 11:00 | 128.4600 | STOCH_EXIT | 1 | 2155.19 | 1.82% | 127.0800 | 130.1968 | 4.1568 | M30 bullish FVG [125.0000, 127.0800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 28 | 2025-02-26 14:00 | 127.4800 | 120.1739 | 2025-02-27 13:30 | 128.2400 | STOCH_EXIT | 33 | 600.53 | 0.50% | 130.5800 | 131.7777 | 4.2977 | M30 bullish FVG [128.5000, 130.5800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 29 | 2025-02-27 21:30 | 128.6200 | 120.3227 | 2025-02-28 08:30 | 131.0200 | STOCH_EXIT | 8 | 2133.97 | 1.76% | 127.2400 | 133.5008 | 4.8808 | M30 bullish FVG [125.8800, 127.2400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 30 | 2025-03-04 12:30 | 135.6000 | 126.3558 | 2025-03-05 16:00 | 134.6600 | STOCH_EXIT | 41 | -978.37 | -0.79% | 136.8200 | 141.0377 | 5.4377 | M30 bullish FVG [135.0000, 136.8200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 31 | 2025-04-01 14:30 | 126.6200 | 117.9180 | 2025-04-02 13:00 | 124.1000 | STOCH_EXIT | 31 | -2566.00 | -2.09% | 128.3200 | 131.7388 | 5.1188 | M30 bullish FVG [125.4600, 128.3200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 32 | 2025-04-10 15:00 | 117.8800 | 107.5249 | 2025-04-10 17:00 | 119.4600 | STOCH_EXIT | 4 | 1490.56 | 1.24% | 118.0200 | 123.9712 | 6.0912 | M30 bullish FVG [116.4000, 118.0200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 33 | 2025-04-23 09:30 | 120.5800 | 111.8636 | 2025-04-23 17:30 | 121.6000 | STOCH_EXIT | 16 | 907.90 | 0.75% | 123.1400 | 125.7073 | 5.1273 | M30 bullish FVG [121.6000, 123.1400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 34 | 2025-05-22 22:00 | 111.6000 | 105.2769 | 2025-05-23 10:30 | 113.1000 | STOCH_EXIT | 11 | 1526.42 | 1.24% | 112.1400 | 115.3195 | 3.7195 | M30 bullish FVG [110.2600, 112.1400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 35 | 2025-06-04 07:30 | 118.4400 | 111.7890 | 2025-06-05 07:30 | 118.5000 | STOCH_EXIT | 35 | -61.39 | -0.05% | 116.0400 | 122.3523 | 3.9123 | M30 bullish FVG [114.9800, 116.0400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 36 | 2025-06-12 04:30 | 119.3200 | 112.6012 | 2025-06-12 09:30 | 119.6600 | STOCH_EXIT | 10 | 229.33 | 0.18% | 118.5600 | 123.2722 | 3.9522 | M30 bullish FVG [116.7000, 118.5600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 37 | 2025-07-10 22:30 | 120.9000 | 114.6336 | 2025-07-11 10:00 | 121.9600 | STOCH_EXIT | 10 | 966.73 | 0.78% | 121.2800 | 124.5861 | 3.6861 | M30 bullish FVG [120.3000, 121.2800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 38 | 2025-07-15 10:30 | 123.6600 | 116.4818 | 2025-07-15 17:00 | 123.7600 | STOCH_EXIT | 13 | -23.95 | -0.02% | 128.0200 | 127.8825 | 4.2225 | M30 bullish FVG [126.8000, 128.0200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 39 | 2025-07-17 17:00 | 124.5200 | 117.0048 | 2025-07-18 07:00 | 125.1800 | STOCH_EXIT | 15 | 535.15 | 0.43% | 126.1400 | 128.9407 | 4.4207 | M30 bullish FVG [124.7000, 126.1400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 40 | 2025-08-12 10:30 | 130.8600 | 124.5307 | 2025-08-12 17:00 | 131.2600 | STOCH_EXIT | 13 | 258.18 | 0.21% | 132.6000 | 134.5831 | 3.7231 | M30 bullish FVG [131.3800, 132.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 41 | 2025-10-17 12:00 | 100.8200 | 93.8757 | 2025-10-17 17:00 | 102.5200 | STOCH_EXIT | 10 | 1997.91 | 1.58% | 99.6800 | 104.9049 | 4.0849 | M30 bullish FVG [98.0400, 99.6800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 99859.03
- Макс: 128511.24
- Финал: 128511.24

Полная кривая — в `*_equity.csv`.
