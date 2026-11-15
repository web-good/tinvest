# Бэктест RTKM (Minutes30)

- Период: 2023-11-15 22:38 — 2025-11-15 22:38
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.1 |
| MaxWaitBars | 30 |
| StochPeriod | 14 |
| UseStochExit | 1 |
| StopATR | 1.5 |
| TakeProfitATR | 0 |
| HTFTrendEMA | 50 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 81 |
| Выигрышных / проигрышных | 57 / 24 |
| Win rate | 70.37% |
| Gross profit / loss | 60273.35 / 26402.20 |
| Profit factor | 2.283 |
| Чистый PnL | 33871.15 (33.87%) |
| Стартовый / финальный капитал | 100000.00 / 133871.15 |
| Макс. просадка | 6784.34 (6.30%) |
| Средняя прибыль / убыток | 1057.43 / 1100.09 |
| Expectancy | 418.16 |
| Лучшая / худшая сделка | 3252.42 / -4466.84 |
| Exposure | 6.27% |
| CAGR | 15.68% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-11-21 11:30 | 82.3200 | 79.8432 | 2023-11-21 19:30 | 82.2700 | STOCH_EXIT | 16 | -160.08 | -0.16% | 82.2500 | 0.0000 | 1.6512 | M30 bullish FVG [81.9800, 82.2500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 2 | 2023-11-28 11:00 | 79.9900 | 77.6453 | 2023-11-28 17:30 | 80.8500 | STOCH_EXIT | 13 | 966.68 | 0.97% | 80.7100 | 0.0000 | 1.5632 | M30 bullish FVG [80.5400, 80.7100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 3 | 2023-11-29 23:00 | 81.3200 | 78.9606 | 2023-11-30 23:30 | 81.5800 | STOCH_EXIT | 30 | 219.62 | 0.22% | 81.7900 | 0.0000 | 1.5729 | M30 bullish FVG [81.4500, 81.7900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 4 | 2024-01-16 11:30 | 73.6200 | 71.2787 | 2024-01-16 12:30 | 74.5900 | STOCH_EXIT | 2 | 1227.38 | 1.22% | 74.0800 | 0.0000 | 1.5609 | M30 bullish FVG [73.8500, 74.0800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 5 | 2024-01-18 19:00 | 76.8600 | 74.4710 | 2024-01-19 22:00 | 75.4500 | STOCH_EXIT | 35 | -1961.72 | -1.93% | 76.5800 | 0.0000 | 1.5927 | M30 bullish FVG [76.4000, 76.5800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 6 | 2024-01-23 11:30 | 76.8300 | 74.4348 | 2024-01-24 20:30 | 75.7500 | STOCH_EXIT | 47 | -1503.18 | -1.50% | 76.8100 | 0.0000 | 1.5968 | M30 bullish FVG [76.4900, 76.8100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 7 | 2024-01-29 17:30 | 77.0600 | 74.7425 | 2024-01-30 17:00 | 76.4400 | STOCH_EXIT | 28 | -891.84 | -0.90% | 77.5400 | 0.0000 | 1.5450 | M30 bullish FVG [77.3600, 77.5400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 8 | 2024-02-07 18:30 | 76.3000 | 74.2637 | 2024-02-08 17:30 | 75.5900 | STOCH_EXIT | 27 | -1006.01 | -1.03% | 76.4500 | 0.0000 | 1.3575 | M30 bullish FVG [76.3100, 76.4500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 9 | 2024-02-16 17:30 | 83.6700 | 80.9699 | 2024-02-16 23:00 | 84.1800 | STOCH_EXIT | 11 | 489.99 | 0.51% | 84.3400 | 0.0000 | 1.8001 | M30 bullish FVG [83.9000, 84.3400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 10 | 2024-02-20 10:30 | 86.0000 | 83.0356 | 2024-02-20 14:00 | 87.9000 | STOCH_EXIT | 7 | 2048.75 | 2.11% | 86.7600 | 0.0000 | 1.9763 | M30 bullish FVG [86.2700, 86.7600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 11 | 2024-02-20 21:00 | 86.2800 | 83.3156 | 2024-02-22 09:30 | 86.7700 | STOCH_EXIT | 35 | 464.00 | 0.47% | 88.4500 | 0.0000 | 1.9763 | M30 bullish FVG [88.0000, 88.4500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 12 | 2024-03-11 18:30 | 92.4300 | 88.9007 | 2024-03-12 23:30 | 91.9800 | STOCH_EXIT | 39 | -585.58 | -0.59% | 92.5600 | 0.0000 | 2.3529 | M30 bullish FVG [92.3200, 92.5600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 13 | 2024-03-21 13:30 | 94.6400 | 91.0442 | 2024-03-24 16:30 | 92.8900 | STOCH_EXIT | 81 | -1917.52 | -1.95% | 95.0400 | 0.0000 | 2.3972 | M30 bullish FVG [94.3000, 95.0400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 14 | 2024-03-26 17:30 | 94.3600 | 90.4761 | 2024-03-27 10:00 | 95.6000 | STOCH_EXIT | 14 | 1179.37 | 1.21% | 95.3000 | 0.0000 | 2.5892 | M30 bullish FVG [94.8700, 95.3000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 15 | 2024-03-27 13:30 | 93.6800 | 89.6825 | 2024-03-27 18:30 | 95.4000 | STOCH_EXIT | 10 | 1706.73 | 1.73% | 95.2500 | 0.0000 | 2.6650 | M30 bullish FVG [94.4500, 95.2500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 16 | 2024-03-28 11:00 | 94.5300 | 90.5577 | 2024-03-28 12:00 | 95.6800 | STOCH_EXIT | 2 | 1118.19 | 1.12% | 95.2500 | 0.0000 | 2.6482 | M30 bullish FVG [94.4500, 95.2500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 17 | 2024-04-01 13:00 | 102.4200 | 98.0301 | 2024-04-01 14:30 | 104.0400 | STOCH_EXIT | 3 | 1486.43 | 1.48% | 104.0400 | 0.0000 | 2.9266 | M30 bullish FVG [102.4600, 104.0400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 18 | 2024-04-04 23:00 | 99.4400 | 95.0035 | 2024-04-06 11:30 | 98.2500 | STOCH_EXIT | 34 | -1327.51 | -1.30% | 99.4600 | 0.0000 | 2.9577 | M30 bullish FVG [98.8000, 99.4600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 19 | 2024-04-09 18:00 | 97.3600 | 93.1779 | 2024-04-10 09:30 | 98.1000 | STOCH_EXIT | 12 | 667.96 | 0.66% | 98.6100 | 0.0000 | 2.7881 | M30 bullish FVG [97.4300, 98.6100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 20 | 2024-04-10 23:00 | 97.1900 | 92.9573 | 2024-04-11 17:00 | 96.9500 | STOCH_EXIT | 17 | -353.92 | -0.35% | 97.0400 | 0.0000 | 2.8218 | M30 bullish FVG [96.7000, 97.0400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 21 | 2024-04-16 15:00 | 99.1600 | 95.2930 | 2024-04-16 22:00 | 99.3400 | STOCH_EXIT | 14 | 82.37 | 0.08% | 99.8100 | 0.0000 | 2.5780 | M30 bullish FVG [99.4800, 99.8100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 22 | 2024-04-17 11:30 | 98.6500 | 94.8385 | 2024-04-18 17:00 | 98.3200 | STOCH_EXIT | 40 | -441.34 | -0.43% | 99.8100 | 0.0000 | 2.5410 | M30 bullish FVG [99.4800, 99.8100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 23 | 2024-04-26 12:30 | 97.3400 | 93.7286 | 2024-04-27 19:00 | 97.3000 | STOCH_EXIT | 42 | -142.81 | -0.14% | 98.0900 | 0.0000 | 2.4076 | M30 bullish FVG [97.7800, 98.0900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 24 | 2024-04-30 11:30 | 96.6300 | 93.2648 | 2024-04-30 20:30 | 96.8400 | STOCH_EXIT | 18 | 117.80 | 0.12% | 97.2100 | 0.0000 | 2.2435 | M30 bullish FVG [96.9300, 97.2100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 25 | 2024-05-07 11:30 | 96.7900 | 93.8483 | 2024-05-07 21:00 | 96.4400 | STOCH_EXIT | 19 | -464.48 | -0.46% | 96.5000 | 0.0000 | 1.9612 | M30 bullish FVG [96.1200, 96.5000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 26 | 2024-05-15 15:30 | 95.7900 | 93.1548 | 2024-05-16 13:00 | 98.5600 | STOCH_EXIT | 24 | 2806.47 | 2.79% | 96.0300 | 0.0000 | 1.7568 | M30 bullish FVG [95.8100, 96.0300], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 27 | 2024-05-20 19:30 | 105.0000 | 101.3542 | 2024-05-22 10:30 | 105.6800 | STOCH_EXIT | 40 | 563.17 | 0.55% | 105.7000 | 0.0000 | 2.4305 | M30 bullish FVG [105.2200, 105.7000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 28 | 2024-05-23 12:30 | 105.1000 | 101.0995 | 2024-05-23 20:00 | 105.5400 | STOCH_EXIT | 15 | 331.33 | 0.32% | 106.7400 | 0.0000 | 2.6670 | M30 bullish FVG [106.3600, 106.7400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 29 | 2024-05-24 15:00 | 103.3200 | 99.2195 | 2024-05-25 10:00 | 105.4700 | STOCH_EXIT | 18 | 2066.06 | 1.98% | 106.3600 | 0.0000 | 2.7336 | M30 bullish FVG [105.6500, 106.3600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 30 | 2024-08-14 13:30 | 85.6900 | 81.2843 | 2024-08-16 09:30 | 84.9100 | STOCH_EXIT | 48 | -1072.97 | -1.01% | 85.8900 | 0.0000 | 2.9371 | M30 bullish FVG [85.0000, 85.8900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 31 | 2024-12-31 17:00 | 66.2000 | 62.3731 | 2025-01-02 07:00 | 66.5500 | STOCH_EXIT | 14 | 450.96 | 0.43% | 66.2800 | 0.0000 | 2.5512 | M30 bullish FVG [65.6500, 66.2800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 32 | 2025-01-09 07:30 | 65.7900 | 62.0168 | 2025-01-10 08:30 | 63.0800 | STOCH_EXIT | 36 | -4466.84 | -4.21% | 66.1000 | 0.0000 | 2.5155 | M30 bullish FVG [65.4800, 66.1000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 33 | 2025-01-15 11:30 | 63.1300 | 58.9753 | 2025-01-15 19:00 | 63.9000 | STOCH_EXIT | 15 | 1137.44 | 1.12% | 63.8500 | 0.0000 | 2.7698 | M30 bullish FVG [63.4900, 63.8500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 34 | 2025-01-17 08:30 | 65.7700 | 61.7230 | 2025-01-17 14:00 | 66.1600 | STOCH_EXIT | 11 | 505.49 | 0.49% | 65.8100 | 0.0000 | 2.6980 | M30 bullish FVG [65.4900, 65.8100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 35 | 2025-01-17 23:30 | 67.8000 | 63.7530 | 2025-01-18 16:00 | 68.3200 | STOCH_EXIT | 19 | 686.95 | 0.67% | 68.1000 | 0.0000 | 2.6980 | M30 bullish FVG [67.2700, 68.1000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 36 | 2025-01-22 11:00 | 66.6200 | 62.3851 | 2025-01-22 12:30 | 67.9500 | STOCH_EXIT | 3 | 1969.84 | 1.89% | 65.2800 | 0.0000 | 2.8233 | M30 bullish FVG [64.9600, 65.2800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 37 | 2025-01-22 19:30 | 67.1300 | 62.8951 | 2025-01-23 10:30 | 67.4500 | STOCH_EXIT | 16 | 396.75 | 0.38% | 67.7000 | 0.0000 | 2.8233 | M30 bullish FVG [67.1900, 67.7000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 38 | 2025-01-29 20:30 | 65.9000 | 62.0966 | 2025-01-30 08:30 | 66.3100 | STOCH_EXIT | 11 | 553.67 | 0.52% | 66.0100 | 0.0000 | 2.5356 | M30 bullish FVG [65.7000, 66.0100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 39 | 2025-01-30 12:30 | 65.8000 | 62.0990 | 2025-01-30 19:00 | 66.1300 | STOCH_EXIT | 13 | 427.74 | 0.40% | 66.0600 | 0.0000 | 2.4673 | M30 bullish FVG [65.8000, 66.0600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 40 | 2025-02-07 19:30 | 64.7100 | 61.4448 | 2025-02-08 02:00 | 65.0600 | STOCH_EXIT | 9 | 470.44 | 0.44% | 65.2200 | 0.0000 | 2.1768 | M30 bullish FVG [64.8600, 65.2200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 41 | 2025-02-10 23:00 | 65.7200 | 62.5509 | 2025-02-11 12:00 | 66.1300 | STOCH_EXIT | 13 | 564.28 | 0.52% | 66.2700 | 0.0000 | 2.1127 | M30 bullish FVG [66.0500, 66.2700], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 42 | 2025-02-12 10:30 | 69.1000 | 65.6674 | 2025-02-12 20:00 | 69.9000 | STOCH_EXIT | 19 | 1139.58 | 1.06% | 68.7500 | 0.0000 | 2.2884 | M30 bullish FVG [68.4000, 68.7500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 43 | 2025-02-13 10:30 | 70.5800 | 66.6168 | 2025-02-13 17:30 | 72.7500 | STOCH_EXIT | 14 | 3252.42 | 2.97% | 74.6000 | 0.0000 | 2.6421 | M30 bullish FVG [74.2000, 74.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 44 | 2025-02-14 16:30 | 71.3800 | 67.2146 | 2025-02-14 23:00 | 72.4000 | STOCH_EXIT | 13 | 1488.53 | 1.33% | 74.4500 | 0.0000 | 2.7770 | M30 bullish FVG [73.9800, 74.4500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 45 | 2025-02-21 13:30 | 78.5400 | 73.8841 | 2025-02-21 18:30 | 79.7000 | STOCH_EXIT | 10 | 1567.28 | 1.38% | 79.7200 | 0.0000 | 3.1040 | M30 bullish FVG [79.2400, 79.7200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 46 | 2025-02-25 12:00 | 79.5200 | 74.9891 | 2025-02-25 22:30 | 79.8300 | STOCH_EXIT | 21 | 333.97 | 0.29% | 80.0200 | 0.0000 | 3.0206 | M30 bullish FVG [79.7000, 80.0200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 47 | 2025-02-28 10:30 | 69.8700 | 64.9437 | 2025-02-28 14:00 | 70.9300 | STOCH_EXIT | 7 | 1642.74 | 1.42% | 70.4500 | 0.0000 | 3.2842 | M30 bullish FVG [69.7700, 70.4500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 48 | 2025-03-05 07:30 | 71.5700 | 66.4361 | 2025-03-05 11:00 | 72.3900 | STOCH_EXIT | 7 | 1226.75 | 1.04% | 71.9000 | 0.0000 | 3.4226 | M30 bullish FVG [71.5500, 71.9000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 49 | 2025-03-05 22:00 | 73.0000 | 67.8661 | 2025-03-06 10:30 | 73.3500 | STOCH_EXIT | 12 | 451.22 | 0.38% | 73.2200 | 0.0000 | 3.4226 | M30 bullish FVG [72.6000, 73.2200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 50 | 2025-03-17 10:30 | 70.7800 | 66.1267 | 2025-03-18 06:30 | 70.8100 | STOCH_EXIT | 27 | -68.54 | -0.06% | 71.4300 | 0.0000 | 3.1022 | M30 bullish FVG [70.9800, 71.4300], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 51 | 2025-03-20 07:00 | 70.2000 | 65.8772 | 2025-03-20 09:00 | 70.8900 | STOCH_EXIT | 4 | 1053.07 | 0.88% | 70.0500 | 0.0000 | 2.8819 | M30 bullish FVG [69.6900, 70.0500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 52 | 2025-03-21 15:00 | 70.1700 | 65.8796 | 2025-03-21 21:30 | 70.3500 | STOCH_EXIT | 13 | 187.66 | 0.16% | 70.2500 | 0.0000 | 2.8603 | M30 bullish FVG [69.8000, 70.2500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 53 | 2025-03-24 21:30 | 71.3000 | 67.1874 | 2025-03-25 07:30 | 72.2800 | STOCH_EXIT | 7 | 1534.87 | 1.27% | 70.8000 | 0.0000 | 2.7417 | M30 bullish FVG [70.3900, 70.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 54 | 2025-03-25 13:00 | 70.9200 | 66.8076 | 2025-03-25 19:00 | 72.5700 | STOCH_EXIT | 12 | 2714.60 | 2.22% | 71.9200 | 0.0000 | 2.7416 | M30 bullish FVG [71.4600, 71.9200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 55 | 2025-03-26 10:30 | 72.0900 | 67.9885 | 2025-03-26 22:30 | 71.3200 | STOCH_EXIT | 24 | -1456.15 | -1.17% | 72.4600 | 0.0000 | 2.7343 | M30 bullish FVG [71.9500, 72.4600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 56 | 2025-06-04 14:30 | 59.9000 | 56.6424 | 2025-06-05 09:30 | 59.2900 | STOCH_EXIT | 25 | -1379.37 | -1.12% | 59.2000 | 0.0000 | 2.1717 | M30 bullish FVG [58.8900, 59.2000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 57 | 2025-06-05 20:00 | 59.4400 | 56.1290 | 2025-06-06 07:30 | 59.9000 | STOCH_EXIT | 10 | 820.68 | 0.67% | 60.0000 | 0.0000 | 2.2073 | M30 bullish FVG [59.5400, 60.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 58 | 2025-07-03 07:00 | 57.7400 | 55.1012 | 2025-07-03 11:00 | 58.7000 | STOCH_EXIT | 8 | 1911.77 | 1.56% | 58.0200 | 0.0000 | 1.7592 | M30 bullish FVG [57.8000, 58.0200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 59 | 2025-07-03 21:30 | 58.4700 | 55.8312 | 2025-07-04 12:30 | 58.5100 | STOCH_EXIT | 17 | -39.38 | -0.03% | 58.4900 | 0.0000 | 1.7592 | M30 bullish FVG [57.8000, 58.4900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 60 | 2025-07-04 21:30 | 58.1800 | 55.5154 | 2025-07-05 16:30 | 58.5900 | STOCH_EXIT | 34 | 752.46 | 0.60% | 58.4200 | 0.0000 | 1.7764 | M30 bullish FVG [58.0800, 58.4200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 61 | 2025-07-11 08:00 | 59.8400 | 57.0297 | 2025-07-12 03:00 | 58.1300 | STOCH_EXIT | 34 | -3697.18 | -2.95% | 60.7900 | 0.0000 | 1.8735 | M30 bullish FVG [60.2700, 60.7900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 62 | 2025-07-16 22:00 | 61.3500 | 58.3610 | 2025-07-17 09:00 | 62.2800 | STOCH_EXIT | 9 | 1719.01 | 1.41% | 60.9100 | 0.0000 | 1.9926 | M30 bullish FVG [60.5300, 60.9100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 63 | 2025-07-17 16:30 | 61.7700 | 58.6763 | 2025-07-18 07:00 | 62.3900 | STOCH_EXIT | 16 | 1110.26 | 0.90% | 63.0600 | 0.0000 | 2.0625 | M30 bullish FVG [62.1400, 63.0600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 64 | 2025-07-23 17:30 | 65.2800 | 62.1771 | 2025-07-23 18:30 | 65.9500 | STOCH_EXIT | 2 | 1148.33 | 0.93% | 65.4800 | 0.0000 | 2.0686 | M30 bullish FVG [65.1800, 65.4800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 65 | 2025-07-24 16:00 | 65.5500 | 62.4973 | 2025-07-25 07:00 | 65.8700 | STOCH_EXIT | 17 | 485.69 | 0.39% | 65.8100 | 0.0000 | 2.0352 | M30 bullish FVG [65.6000, 65.8100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 66 | 2025-08-01 20:00 | 62.2000 | 59.3088 | 2025-08-01 23:30 | 62.8500 | STOCH_EXIT | 7 | 1186.70 | 0.94% | 62.7500 | 0.0000 | 1.9275 | M30 bullish FVG [62.4300, 62.7500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 67 | 2025-08-07 10:00 | 68.1600 | 65.0171 | 2025-08-07 14:00 | 68.7200 | STOCH_EXIT | 8 | 914.30 | 0.72% | 67.8600 | 0.0000 | 2.0953 | M30 bullish FVG [67.5900, 67.8600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 68 | 2025-08-13 07:30 | 70.3900 | 67.3093 | 2025-08-13 17:30 | 70.0800 | STOCH_EXIT | 20 | -692.03 | -0.54% | 72.2400 | 0.0000 | 2.0538 | M30 bullish FVG [71.9800, 72.2400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 69 | 2025-08-15 10:30 | 69.1400 | 65.8850 | 2025-08-15 11:30 | 70.3600 | STOCH_EXIT | 2 | 2116.46 | 1.66% | 68.3700 | 0.0000 | 2.1700 | M30 bullish FVG [68.1100, 68.3700], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 70 | 2025-08-18 18:30 | 69.6100 | 66.2853 | 2025-08-18 21:30 | 70.5000 | STOCH_EXIT | 6 | 1525.10 | 1.18% | 69.9600 | 0.0000 | 2.2164 | M30 bullish FVG [69.6100, 69.9600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 71 | 2025-08-21 18:30 | 69.4100 | 66.1712 | 2025-08-22 08:00 | 69.7700 | STOCH_EXIT | 14 | 548.87 | 0.42% | 71.3800 | 0.0000 | 2.1592 | M30 bullish FVG [70.8300, 71.3800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 72 | 2025-08-27 09:00 | 73.4300 | 70.0608 | 2025-08-27 10:00 | 73.8400 | STOCH_EXIT | 2 | 602.09 | 0.46% | 73.1100 | 0.0000 | 2.2462 | M30 bullish FVG [72.5000, 73.1100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 73 | 2025-08-29 13:30 | 72.2700 | 68.9483 | 2025-08-29 21:00 | 72.1300 | STOCH_EXIT | 15 | -388.33 | -0.29% | 72.9200 | 0.0000 | 2.2144 | M30 bullish FVG [72.6200, 72.9200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 74 | 2025-09-03 11:00 | 69.9100 | 66.5994 | 2025-09-03 19:30 | 69.9800 | STOCH_EXIT | 17 | 0.10 | 0.00% | 70.5900 | 0.0000 | 2.2070 | M30 bullish FVG [70.3100, 70.5900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 75 | 2025-09-04 14:30 | 70.1400 | 66.8720 | 2025-09-04 22:00 | 69.8100 | STOCH_EXIT | 15 | -751.95 | -0.57% | 70.6400 | 0.0000 | 2.1787 | M30 bullish FVG [70.2900, 70.6400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 76 | 2025-09-09 09:30 | 69.8100 | 66.7025 | 2025-09-09 23:00 | 69.8900 | STOCH_EXIT | 27 | 19.08 | 0.01% | 69.8500 | 0.0000 | 2.0717 | M30 bullish FVG [69.6000, 69.8500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 77 | 2025-10-17 11:30 | 63.9600 | 59.8372 | 2025-10-17 18:00 | 64.3700 | STOCH_EXIT | 13 | 708.96 | 0.54% | 64.8500 | 0.0000 | 2.7485 | M30 bullish FVG [64.5000, 64.8500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 78 | 2025-10-21 07:30 | 65.5000 | 61.4194 | 2025-10-21 13:30 | 66.3400 | STOCH_EXIT | 12 | 1555.90 | 1.18% | 66.2600 | 0.0000 | 2.7204 | M30 bullish FVG [65.5600, 66.2600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 79 | 2025-10-21 21:00 | 63.2100 | 59.1294 | 2025-10-22 07:30 | 64.1500 | STOCH_EXIT | 8 | 1849.04 | 1.39% | 64.0100 | 0.0000 | 2.7204 | M30 bullish FVG [63.4400, 64.0100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 80 | 2025-10-22 13:30 | 63.5500 | 59.2959 | 2025-10-23 16:00 | 63.0000 | STOCH_EXIT | 40 | -1306.28 | -0.96% | 64.0100 | 0.0000 | 2.8361 | M30 bullish FVG [63.4400, 64.0100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 81 | 2025-11-07 18:30 | 62.2300 | 58.3197 | 2025-11-08 06:00 | 62.1400 | STOCH_EXIT | 19 | -327.20 | -0.24% | 63.1300 | 0.0000 | 2.6068 | M30 bullish FVG [62.6900, 63.1300], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 96055.34
- Макс: 137311.34
- Финал: 133871.15

Полная кривая — в `*_equity.csv`.
