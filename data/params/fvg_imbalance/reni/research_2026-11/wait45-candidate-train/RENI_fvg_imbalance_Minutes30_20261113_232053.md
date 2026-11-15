# Бэктест RENI (Minutes30)

- Период: 2023-11-13 23:20 — 2025-11-13 23:20
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.25 |
| MaxWaitBars | 45 |
| StochPeriod | 14 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 1 |
| HTFTrendEMA | 50 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 33 |
| Выигрышных / проигрышных | 27 / 6 |
| Win rate | 81.82% |
| Gross profit / loss | 26319.26 / 2325.20 |
| Profit factor | 11.319 |
| Чистый PnL | 23994.06 (23.99%) |
| Стартовый / финальный капитал | 100000.00 / 123994.06 |
| Макс. просадка | 4912.80 (4.11%) |
| Средняя прибыль / убыток | 974.79 / 387.53 |
| Expectancy | 727.09 |
| Лучшая / худшая сделка | 2246.05 / -946.11 |
| Exposure | 2.71% |
| CAGR | 11.34% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-01-16 15:30 | 96.5000 | 93.5592 | 2024-01-17 11:00 | 96.8000 | STOCH_EXIT | 10 | 209.45 | 0.21% | 97.7000 | 99.4408 | 2.9408 | M30 bullish FVG [96.2600, 97.7000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 2 | 2024-01-26 18:30 | 99.2200 | 96.3954 | 2024-01-29 15:30 | 99.8800 | STOCH_EXIT | 13 | 560.45 | 0.56% | 99.7600 | 102.0446 | 2.8246 | M30 bullish FVG [98.6800, 99.7600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 3 | 2024-02-20 12:00 | 101.3800 | 98.6249 | 2024-02-20 17:00 | 102.1600 | STOCH_EXIT | 10 | 671.45 | 0.67% | 101.9000 | 104.1351 | 2.7551 | M30 bullish FVG [100.5000, 101.9000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 4 | 2024-04-04 10:30 | 102.7000 | 100.3042 | 2024-04-06 10:30 | 105.0958 | TP | 36 | 2246.05 | 2.23% | 104.6200 | 105.0958 | 2.3958 | M30 bullish FVG [103.9800, 104.6200], stochastic crossed below 20 | FVG target 105.0958 |
| 5 | 2024-04-19 18:30 | 107.0800 | 104.7742 | 2024-04-20 16:30 | 107.9600 | STOCH_EXIT | 14 | 741.58 | 0.72% | 106.8000 | 109.3858 | 2.3058 | M30 bullish FVG [106.0000, 106.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 6 | 2024-08-15 23:00 | 94.9800 | 91.6626 | 2024-08-16 17:00 | 95.7600 | STOCH_EXIT | 17 | 746.25 | 0.72% | 94.7800 | 98.2974 | 3.3174 | M30 bullish FVG [93.8800, 94.7800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 7 | 2024-08-16 23:00 | 93.5000 | 90.2281 | 2024-08-17 13:00 | 93.3600 | STOCH_EXIT | 12 | -261.44 | -0.25% | 95.3800 | 96.7719 | 3.2719 | M30 bullish FVG [94.3000, 95.3800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 8 | 2024-08-19 20:00 | 93.9400 | 90.6947 | 2024-08-20 10:00 | 95.5000 | STOCH_EXIT | 9 | 1626.46 | 1.56% | 95.2200 | 97.1853 | 3.2453 | M30 bullish FVG [94.3000, 95.2200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 9 | 2024-08-23 20:00 | 92.4000 | 89.2274 | 2024-08-24 09:30 | 94.4200 | STOCH_EXIT | 13 | 2215.58 | 2.08% | 92.7800 | 95.5726 | 3.1726 | M30 bullish FVG [91.7400, 92.7800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 10 | 2024-08-29 13:00 | 95.1000 | 91.7741 | 2024-08-29 22:30 | 95.8000 | STOCH_EXIT | 19 | 689.19 | 0.64% | 96.6400 | 98.4259 | 3.3259 | M30 bullish FVG [95.7400, 96.6400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 11 | 2024-09-27 08:00 | 99.8800 | 96.5838 | 2024-09-27 17:00 | 100.3000 | STOCH_EXIT | 18 | 348.70 | 0.32% | 97.3400 | 103.1762 | 3.2962 | M30 bullish FVG [96.3000, 97.3400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 12 | 2024-11-12 10:30 | 100.5200 | 97.6908 | 2024-11-13 09:00 | 101.3200 | STOCH_EXIT | 31 | 762.00 | 0.70% | 100.1800 | 103.3492 | 2.8292 | M30 bullish FVG [99.4800, 100.1800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 13 | 2024-11-13 20:00 | 103.2600 | 100.3829 | 2024-11-14 07:00 | 104.2000 | STOCH_EXIT | 8 | 894.81 | 0.81% | 103.1600 | 106.1371 | 2.8771 | M30 bullish FVG [102.0200, 103.1600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 14 | 2024-11-28 10:00 | 97.6000 | 94.7796 | 2024-11-28 10:00 | 98.6000 | STOCH_EXIT | 0 | 1028.17 | 0.92% | 95.7400 | 100.4204 | 2.8204 | M30 bullish FVG [94.5000, 95.7400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 15 | 2024-12-18 12:30 | 94.6400 | 91.9054 | 2024-12-18 18:30 | 96.0400 | STOCH_EXIT | 12 | 1539.50 | 1.38% | 96.5400 | 97.3746 | 2.7346 | M30 bullish FVG [95.1400, 96.5400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 16 | 2024-12-24 18:30 | 100.3800 | 96.8367 | 2024-12-24 23:30 | 100.9200 | STOCH_EXIT | 10 | 496.47 | 0.44% | 101.0200 | 103.9233 | 3.5433 | M30 bullish FVG [100.0800, 101.0200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 17 | 2025-01-06 21:30 | 109.3000 | 105.6630 | 2025-01-07 16:00 | 109.2800 | STOCH_EXIT | 22 | -134.46 | -0.12% | 110.5800 | 112.9370 | 3.6370 | M30 bullish FVG [109.6600, 110.5800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 18 | 2025-01-10 21:30 | 112.0000 | 108.2931 | 2025-01-12 14:00 | 112.4800 | STOCH_EXIT | 52 | 375.12 | 0.33% | 109.0400 | 115.7069 | 3.7069 | M30 bullish FVG [107.3000, 109.0400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 19 | 2025-01-23 10:30 | 115.6800 | 111.9177 | 2025-01-24 07:00 | 117.6600 | STOCH_EXIT | 27 | 1844.70 | 1.61% | 117.7000 | 119.4423 | 3.7623 | M30 bullish FVG [115.9000, 117.7000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 20 | 2025-01-27 17:00 | 118.2400 | 114.3593 | 2025-01-28 08:00 | 117.4400 | STOCH_EXIT | 16 | -899.48 | -0.78% | 120.2200 | 122.1207 | 3.8807 | M30 bullish FVG [119.2000, 120.2200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 21 | 2025-02-14 10:30 | 126.0400 | 121.8832 | 2025-02-14 11:00 | 128.4600 | STOCH_EXIT | 1 | 2086.40 | 1.82% | 127.0800 | 130.1968 | 4.1568 | M30 bullish FVG [125.0000, 127.0800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 22 | 2025-02-26 14:00 | 127.4800 | 123.1823 | 2025-02-27 13:30 | 128.2400 | STOCH_EXIT | 33 | 581.57 | 0.50% | 130.5800 | 131.7777 | 4.2977 | M30 bullish FVG [128.5000, 130.5800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 23 | 2025-02-27 21:30 | 128.6200 | 123.7392 | 2025-02-28 08:30 | 131.0200 | STOCH_EXIT | 8 | 2065.86 | 1.76% | 127.2400 | 133.5008 | 4.8808 | M30 bullish FVG [125.8800, 127.2400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 24 | 2025-03-04 12:30 | 135.6000 | 130.1623 | 2025-03-05 16:00 | 134.6600 | STOCH_EXIT | 41 | -946.11 | -0.79% | 136.8200 | 141.0377 | 5.4377 | M30 bullish FVG [135.0000, 136.8200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 25 | 2025-03-21 11:00 | 136.4800 | 131.5702 | 2025-03-21 12:30 | 138.6000 | STOCH_EXIT | 3 | 1724.74 | 1.45% | 138.5800 | 141.3898 | 4.9098 | M30 bullish FVG [136.6800, 138.5800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 26 | 2025-04-23 09:30 | 120.5800 | 115.4527 | 2025-04-23 17:30 | 121.6000 | STOCH_EXIT | 16 | 898.91 | 0.75% | 123.1400 | 125.7073 | 5.1273 | M30 bullish FVG [121.6000, 123.1400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 27 | 2025-06-04 07:30 | 118.4400 | 114.5277 | 2025-06-05 07:30 | 118.5000 | STOCH_EXIT | 35 | -60.22 | -0.05% | 116.0400 | 122.3523 | 3.9123 | M30 bullish FVG [114.9800, 116.0400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 28 | 2025-06-12 04:30 | 119.3200 | 115.3678 | 2025-06-12 09:30 | 119.6600 | STOCH_EXIT | 10 | 224.92 | 0.18% | 118.5600 | 123.2722 | 3.9522 | M30 bullish FVG [116.7000, 118.5600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 29 | 2025-07-03 19:30 | 123.8600 | 120.4552 | 2025-07-04 10:00 | 124.0000 | STOCH_EXIT | 16 | 15.75 | 0.01% | 122.4800 | 127.2648 | 3.4048 | M30 bullish FVG [121.5800, 122.4800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 30 | 2025-07-10 22:30 | 120.9000 | 117.2139 | 2025-07-11 10:00 | 121.9600 | STOCH_EXIT | 10 | 947.96 | 0.78% | 121.2800 | 124.5861 | 3.6861 | M30 bullish FVG [120.3000, 121.2800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 31 | 2025-07-15 10:30 | 123.6600 | 119.4375 | 2025-07-15 17:00 | 123.7600 | STOCH_EXIT | 13 | -23.47 | -0.02% | 128.0200 | 127.8825 | 4.2225 | M30 bullish FVG [126.8000, 128.0200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 32 | 2025-07-17 17:00 | 124.5200 | 120.0993 | 2025-07-18 07:00 | 125.1800 | STOCH_EXIT | 15 | 524.45 | 0.43% | 126.1400 | 128.9407 | 4.4207 | M30 bullish FVG [124.7000, 126.1400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 33 | 2025-08-12 10:30 | 130.8600 | 127.1369 | 2025-08-12 17:00 | 131.2600 | STOCH_EXIT | 13 | 252.80 | 0.21% | 132.6000 | 134.5831 | 3.7231 | M30 bullish FVG [131.3800, 132.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 99059.84
- Макс: 124093.36
- Финал: 123994.06

Полная кривая — в `*_equity.csv`.
