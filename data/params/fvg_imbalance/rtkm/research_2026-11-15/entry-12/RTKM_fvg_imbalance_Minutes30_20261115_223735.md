# Бэктест RTKM (Minutes30)

- Период: 2023-11-15 22:37 — 2025-11-15 22:37
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
| StopATR | 1 |
| TakeProfitATR | 2 |
| HTFTrendEMA | 50 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 81 |
| Выигрышных / проигрышных | 57 / 24 |
| Win rate | 70.37% |
| Gross profit / loss | 59860.50 / 27160.72 |
| Profit factor | 2.204 |
| Чистый PnL | 32699.78 (32.70%) |
| Стартовый / финальный капитал | 100000.00 / 132699.78 |
| Макс. просадка | 6412.69 (6.27%) |
| Средняя прибыль / убыток | 1050.18 / 1131.70 |
| Expectancy | 403.70 |
| Лучшая / худшая сделка | 3231.44 / -4102.18 |
| Exposure | 5.81% |
| CAGR | 15.17% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-11-21 11:30 | 82.3200 | 80.6688 | 2023-11-21 19:30 | 82.2700 | STOCH_EXIT | 16 | -160.08 | -0.16% | 82.2500 | 85.6225 | 1.6512 | M30 bullish FVG [81.9800, 82.2500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 2 | 2023-11-28 11:00 | 79.9900 | 78.4268 | 2023-11-28 17:30 | 80.8500 | STOCH_EXIT | 13 | 966.68 | 0.97% | 80.7100 | 83.1163 | 1.5632 | M30 bullish FVG [80.5400, 80.7100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 3 | 2023-11-29 23:00 | 81.3200 | 79.7471 | 2023-11-30 23:30 | 81.5800 | STOCH_EXIT | 30 | 219.62 | 0.22% | 81.7900 | 84.4659 | 1.5729 | M30 bullish FVG [81.4500, 81.7900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 4 | 2024-01-16 11:30 | 73.6200 | 72.0591 | 2024-01-16 12:30 | 74.5900 | STOCH_EXIT | 2 | 1227.38 | 1.22% | 74.0800 | 76.7418 | 1.5609 | M30 bullish FVG [73.8500, 74.0800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 5 | 2024-01-18 19:00 | 76.8600 | 75.2673 | 2024-01-19 15:00 | 75.2673 | SL | 21 | -2202.72 | -2.17% | 76.5800 | 80.0453 | 1.5927 | M30 bullish FVG [76.4000, 76.5800], stochastic crossed below 20 | FVG stop 75.2673 |
| 6 | 2024-01-23 11:30 | 76.8300 | 75.2332 | 2024-01-24 20:30 | 75.7500 | STOCH_EXIT | 47 | -1503.18 | -1.50% | 76.8100 | 80.0236 | 1.5968 | M30 bullish FVG [76.4900, 76.8100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 7 | 2024-01-29 17:30 | 77.0600 | 75.5150 | 2024-01-30 17:00 | 76.4400 | STOCH_EXIT | 28 | -884.87 | -0.90% | 77.5400 | 80.1500 | 1.5450 | M30 bullish FVG [77.3600, 77.5400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 8 | 2024-02-07 18:30 | 76.3000 | 74.9425 | 2024-02-08 17:30 | 75.5900 | STOCH_EXIT | 27 | -998.15 | -1.03% | 76.4500 | 79.0151 | 1.3575 | M30 bullish FVG [76.3100, 76.4500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 9 | 2024-02-16 17:30 | 83.6700 | 81.8699 | 2024-02-16 23:00 | 84.1800 | STOCH_EXIT | 11 | 489.99 | 0.51% | 84.3400 | 87.2701 | 1.8001 | M30 bullish FVG [83.9000, 84.3400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 10 | 2024-02-20 10:30 | 86.0000 | 84.0237 | 2024-02-20 14:00 | 87.9000 | STOCH_EXIT | 7 | 2030.62 | 2.11% | 86.7600 | 89.9525 | 1.9763 | M30 bullish FVG [86.2700, 86.7600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 11 | 2024-02-20 21:00 | 86.2800 | 84.3037 | 2024-02-22 09:30 | 86.7700 | STOCH_EXIT | 35 | 459.96 | 0.47% | 88.4500 | 90.2325 | 1.9763 | M30 bullish FVG [88.0000, 88.4500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 12 | 2024-03-11 18:30 | 92.4300 | 90.0771 | 2024-03-12 23:30 | 91.9800 | STOCH_EXIT | 39 | -580.16 | -0.59% | 92.5600 | 97.1358 | 2.3529 | M30 bullish FVG [92.3200, 92.5600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 13 | 2024-03-21 13:30 | 94.6400 | 92.2428 | 2024-03-22 21:00 | 92.2428 | SL | 44 | -2590.25 | -2.63% | 95.0400 | 99.4344 | 2.3972 | M30 bullish FVG [94.3000, 95.0400], stochastic crossed below 20 | FVG stop 92.2428 |
| 14 | 2024-03-26 17:30 | 94.3600 | 91.7708 | 2024-03-27 10:00 | 95.6000 | STOCH_EXIT | 14 | 1167.92 | 1.21% | 95.3000 | 99.5385 | 2.5892 | M30 bullish FVG [94.8700, 95.3000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 15 | 2024-03-27 13:30 | 93.6800 | 91.0150 | 2024-03-27 18:30 | 95.4000 | STOCH_EXIT | 10 | 1690.48 | 1.73% | 95.2500 | 99.0100 | 2.6650 | M30 bullish FVG [94.4500, 95.2500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 16 | 2024-03-28 11:00 | 94.5300 | 91.8818 | 2024-03-28 12:00 | 95.6800 | STOCH_EXIT | 2 | 1107.64 | 1.12% | 95.2500 | 99.8265 | 2.6482 | M30 bullish FVG [94.4500, 95.2500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 17 | 2024-04-01 13:00 | 102.4200 | 99.4934 | 2024-04-01 14:30 | 104.0400 | STOCH_EXIT | 3 | 1486.43 | 1.48% | 104.0400 | 108.2732 | 2.9266 | M30 bullish FVG [102.4600, 104.0400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 18 | 2024-04-04 23:00 | 99.4400 | 96.4823 | 2024-04-06 11:30 | 98.2500 | STOCH_EXIT | 34 | -1314.62 | -1.30% | 99.4600 | 105.3554 | 2.9577 | M30 bullish FVG [98.8000, 99.4600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 19 | 2024-04-09 18:00 | 97.3600 | 94.5719 | 2024-04-10 09:30 | 98.1000 | STOCH_EXIT | 12 | 661.54 | 0.66% | 98.6100 | 102.9362 | 2.7881 | M30 bullish FVG [97.4300, 98.6100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 20 | 2024-04-10 23:00 | 97.1900 | 94.3682 | 2024-04-11 17:00 | 96.9500 | STOCH_EXIT | 17 | -350.55 | -0.35% | 97.0400 | 102.8336 | 2.8218 | M30 bullish FVG [96.7000, 97.0400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 21 | 2024-04-16 15:00 | 99.1600 | 96.5820 | 2024-04-16 22:00 | 99.3400 | STOCH_EXIT | 14 | 81.56 | 0.08% | 99.8100 | 104.3160 | 2.5780 | M30 bullish FVG [99.4800, 99.8100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 22 | 2024-04-17 11:30 | 98.6500 | 96.1090 | 2024-04-18 17:00 | 98.3200 | STOCH_EXIT | 40 | -437.05 | -0.43% | 99.8100 | 103.7320 | 2.5410 | M30 bullish FVG [99.4800, 99.8100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 23 | 2024-04-26 12:30 | 97.3400 | 94.9324 | 2024-04-27 19:00 | 97.3000 | STOCH_EXIT | 42 | -141.44 | -0.14% | 98.0900 | 102.1552 | 2.4076 | M30 bullish FVG [97.7800, 98.0900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 24 | 2024-04-30 11:30 | 96.6300 | 94.3865 | 2024-04-30 20:30 | 96.8400 | STOCH_EXIT | 18 | 116.66 | 0.12% | 97.2100 | 101.1169 | 2.2435 | M30 bullish FVG [96.9300, 97.2100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 25 | 2024-05-07 11:30 | 96.7900 | 94.8288 | 2024-05-07 21:00 | 96.4400 | STOCH_EXIT | 19 | -460.01 | -0.46% | 96.5000 | 100.7123 | 1.9612 | M30 bullish FVG [96.1200, 96.5000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 26 | 2024-05-15 15:30 | 95.7900 | 94.0332 | 2024-05-16 13:00 | 98.5600 | STOCH_EXIT | 24 | 2779.74 | 2.79% | 96.0300 | 99.3036 | 1.7568 | M30 bullish FVG [95.8100, 96.0300], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 27 | 2024-05-20 19:30 | 105.0000 | 102.5695 | 2024-05-22 10:30 | 105.6800 | STOCH_EXIT | 40 | 557.42 | 0.55% | 105.7000 | 109.8611 | 2.4305 | M30 bullish FVG [105.2200, 105.7000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 28 | 2024-05-23 12:30 | 105.1000 | 102.4330 | 2024-05-23 20:00 | 105.5400 | STOCH_EXIT | 15 | 327.99 | 0.32% | 106.7400 | 110.4340 | 2.6670 | M30 bullish FVG [106.3600, 106.7400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 29 | 2024-05-24 15:00 | 103.3200 | 100.5864 | 2024-05-25 10:00 | 105.4700 | STOCH_EXIT | 18 | 2045.61 | 1.98% | 106.3600 | 108.7873 | 2.7336 | M30 bullish FVG [105.6500, 106.3600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 30 | 2024-08-14 13:30 | 85.6900 | 82.7529 | 2024-08-16 09:30 | 84.9100 | STOCH_EXIT | 48 | -1064.32 | -1.01% | 85.8900 | 91.5642 | 2.9371 | M30 bullish FVG [85.0000, 85.8900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 31 | 2024-12-31 17:00 | 66.2000 | 63.6488 | 2025-01-02 07:00 | 66.5500 | STOCH_EXIT | 14 | 448.13 | 0.43% | 66.2800 | 71.3025 | 2.5512 | M30 bullish FVG [65.6500, 66.2800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 32 | 2025-01-09 07:30 | 65.7900 | 63.2745 | 2025-01-09 12:00 | 63.2745 | SL | 9 | -4102.18 | -3.92% | 66.1000 | 70.8209 | 2.5155 | M30 bullish FVG [65.4800, 66.1000], stochastic crossed below 20 | FVG stop 63.2745 |
| 33 | 2025-01-15 11:30 | 63.1300 | 60.3602 | 2025-01-15 19:00 | 63.9000 | STOCH_EXIT | 15 | 1130.38 | 1.12% | 63.8500 | 68.6696 | 2.7698 | M30 bullish FVG [63.4900, 63.8500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 34 | 2025-01-17 08:30 | 65.7700 | 63.0720 | 2025-01-17 14:00 | 66.1600 | STOCH_EXIT | 11 | 502.25 | 0.49% | 65.8100 | 71.1660 | 2.6980 | M30 bullish FVG [65.4900, 65.8100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 35 | 2025-01-17 23:30 | 67.8000 | 65.1020 | 2025-01-18 16:00 | 68.3200 | STOCH_EXIT | 19 | 682.43 | 0.67% | 68.1000 | 73.1960 | 2.6980 | M30 bullish FVG [67.2700, 68.1000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 36 | 2025-01-22 11:00 | 66.6200 | 63.7967 | 2025-01-22 12:30 | 67.9500 | STOCH_EXIT | 3 | 1957.21 | 1.89% | 65.2800 | 72.2666 | 2.8233 | M30 bullish FVG [64.9600, 65.2800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 37 | 2025-01-22 19:30 | 67.1300 | 64.3067 | 2025-01-23 10:30 | 67.4500 | STOCH_EXIT | 16 | 394.23 | 0.38% | 67.7000 | 72.7766 | 2.8233 | M30 bullish FVG [67.1900, 67.7000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 38 | 2025-01-29 20:30 | 65.9000 | 63.3644 | 2025-01-30 08:30 | 66.3100 | STOCH_EXIT | 11 | 550.23 | 0.52% | 66.0100 | 70.9712 | 2.5356 | M30 bullish FVG [65.7000, 66.0100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 39 | 2025-01-30 12:30 | 65.8000 | 63.3327 | 2025-01-30 19:00 | 66.1300 | STOCH_EXIT | 13 | 425.10 | 0.40% | 66.0600 | 70.7347 | 2.4673 | M30 bullish FVG [65.8000, 66.0600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 40 | 2025-02-07 19:30 | 64.7100 | 62.5332 | 2025-02-08 02:00 | 65.0600 | STOCH_EXIT | 9 | 467.59 | 0.44% | 65.2200 | 69.0636 | 2.1768 | M30 bullish FVG [64.8600, 65.2200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 41 | 2025-02-10 23:00 | 65.7200 | 63.6073 | 2025-02-11 12:00 | 66.1300 | STOCH_EXIT | 13 | 560.84 | 0.52% | 66.2700 | 69.9455 | 2.1127 | M30 bullish FVG [66.0500, 66.2700], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 42 | 2025-02-12 10:30 | 69.1000 | 66.8116 | 2025-02-12 20:00 | 69.9000 | STOCH_EXIT | 19 | 1132.28 | 1.06% | 68.7500 | 73.6769 | 2.2884 | M30 bullish FVG [68.4000, 68.7500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 43 | 2025-02-13 10:30 | 70.5800 | 67.9379 | 2025-02-13 17:30 | 72.7500 | STOCH_EXIT | 14 | 3231.44 | 2.97% | 74.6000 | 75.8642 | 2.6421 | M30 bullish FVG [74.2000, 74.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 44 | 2025-02-14 16:30 | 71.3800 | 68.6030 | 2025-02-14 23:00 | 72.4000 | STOCH_EXIT | 13 | 1479.05 | 1.33% | 74.4500 | 76.9339 | 2.7770 | M30 bullish FVG [73.9800, 74.4500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 45 | 2025-02-21 13:30 | 78.5400 | 75.4360 | 2025-02-21 18:30 | 79.7000 | STOCH_EXIT | 10 | 1556.47 | 1.38% | 79.7200 | 84.7479 | 3.1040 | M30 bullish FVG [79.2400, 79.7200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 46 | 2025-02-25 12:00 | 79.5200 | 76.4994 | 2025-02-25 22:30 | 79.8300 | STOCH_EXIT | 21 | 331.67 | 0.29% | 80.0200 | 85.5612 | 3.0206 | M30 bullish FVG [79.7000, 80.0200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 47 | 2025-02-28 10:30 | 69.8700 | 66.5858 | 2025-02-28 14:00 | 70.9300 | STOCH_EXIT | 7 | 1632.84 | 1.42% | 70.4500 | 76.4384 | 3.2842 | M30 bullish FVG [69.7700, 70.4500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 48 | 2025-03-05 07:30 | 71.5700 | 68.1474 | 2025-03-05 11:00 | 72.3900 | STOCH_EXIT | 7 | 1219.27 | 1.04% | 71.9000 | 78.4152 | 3.4226 | M30 bullish FVG [71.5500, 71.9000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 49 | 2025-03-05 22:00 | 73.0000 | 69.5774 | 2025-03-06 10:30 | 73.3500 | STOCH_EXIT | 12 | 448.46 | 0.38% | 73.2200 | 79.8452 | 3.4226 | M30 bullish FVG [72.6000, 73.2200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 50 | 2025-03-17 10:30 | 70.7800 | 67.6778 | 2025-03-18 06:30 | 70.8100 | STOCH_EXIT | 27 | -68.13 | -0.06% | 71.4300 | 76.9844 | 3.1022 | M30 bullish FVG [70.9800, 71.4300], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 51 | 2025-03-20 07:00 | 70.2000 | 67.3181 | 2025-03-20 09:00 | 70.8900 | STOCH_EXIT | 4 | 1046.88 | 0.88% | 70.0500 | 75.9637 | 2.8819 | M30 bullish FVG [69.6900, 70.0500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 52 | 2025-03-21 15:00 | 70.1700 | 67.3097 | 2025-03-21 21:30 | 70.3500 | STOCH_EXIT | 13 | 186.56 | 0.16% | 70.2500 | 75.8906 | 2.8603 | M30 bullish FVG [69.8000, 70.2500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 53 | 2025-03-24 21:30 | 71.3000 | 68.5583 | 2025-03-25 07:30 | 72.2800 | STOCH_EXIT | 7 | 1525.79 | 1.27% | 70.8000 | 76.7834 | 2.7417 | M30 bullish FVG [70.3900, 70.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 54 | 2025-03-25 13:00 | 70.9200 | 68.1784 | 2025-03-25 19:00 | 72.5700 | STOCH_EXIT | 12 | 2698.82 | 2.22% | 71.9200 | 76.4032 | 2.7416 | M30 bullish FVG [71.4600, 71.9200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 55 | 2025-03-26 10:30 | 72.0900 | 69.3557 | 2025-03-26 22:30 | 71.3200 | STOCH_EXIT | 24 | -1447.73 | -1.17% | 72.4600 | 77.5587 | 2.7343 | M30 bullish FVG [71.9500, 72.4600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 56 | 2025-06-04 14:30 | 59.9000 | 57.7283 | 2025-06-05 09:30 | 59.2900 | STOCH_EXIT | 25 | -1365.97 | -1.12% | 59.2000 | 64.2435 | 2.1717 | M30 bullish FVG [58.8900, 59.2000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 57 | 2025-06-05 20:00 | 59.4400 | 57.2327 | 2025-06-06 07:30 | 59.9000 | STOCH_EXIT | 10 | 816.67 | 0.67% | 60.0000 | 63.8547 | 2.2073 | M30 bullish FVG [59.5400, 60.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 58 | 2025-07-03 07:00 | 57.7400 | 55.9808 | 2025-07-03 11:00 | 58.7000 | STOCH_EXIT | 8 | 1902.76 | 1.56% | 58.0200 | 61.2583 | 1.7592 | M30 bullish FVG [57.8000, 58.0200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 59 | 2025-07-03 21:30 | 58.4700 | 56.7108 | 2025-07-04 12:30 | 58.5100 | STOCH_EXIT | 17 | -39.20 | -0.03% | 58.4900 | 61.9883 | 1.7592 | M30 bullish FVG [57.8000, 58.4900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 60 | 2025-07-04 21:30 | 58.1800 | 56.4036 | 2025-07-05 16:30 | 58.5900 | STOCH_EXIT | 34 | 748.94 | 0.60% | 58.4200 | 61.7327 | 1.7764 | M30 bullish FVG [58.0800, 58.4200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 61 | 2025-07-11 08:00 | 59.8400 | 57.9665 | 2025-07-11 09:00 | 57.9665 | SL | 2 | -4019.48 | -3.23% | 60.7900 | 63.5871 | 1.8735 | M30 bullish FVG [60.2700, 60.7900], stochastic crossed below 20 | FVG stop 57.9665 |
| 62 | 2025-07-16 22:00 | 61.3500 | 59.3574 | 2025-07-17 09:00 | 62.2800 | STOCH_EXIT | 9 | 1701.64 | 1.41% | 60.9100 | 65.3353 | 1.9926 | M30 bullish FVG [60.5300, 60.9100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 63 | 2025-07-17 16:30 | 61.7700 | 59.7075 | 2025-07-18 07:00 | 62.3900 | STOCH_EXIT | 16 | 1104.68 | 0.90% | 63.0600 | 65.8949 | 2.0625 | M30 bullish FVG [62.1400, 63.0600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 64 | 2025-07-23 17:30 | 65.2800 | 63.2114 | 2025-07-23 18:30 | 65.9500 | STOCH_EXIT | 2 | 1142.29 | 0.93% | 65.4800 | 69.4173 | 2.0686 | M30 bullish FVG [65.1800, 65.4800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 65 | 2025-07-24 16:00 | 65.5500 | 63.5148 | 2025-07-25 07:00 | 65.8700 | STOCH_EXIT | 17 | 483.15 | 0.39% | 65.8100 | 69.6203 | 2.0352 | M30 bullish FVG [65.6000, 65.8100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 66 | 2025-08-01 20:00 | 62.2000 | 60.2725 | 2025-08-01 23:30 | 62.8500 | STOCH_EXIT | 7 | 1180.82 | 0.94% | 62.7500 | 66.0549 | 1.9275 | M30 bullish FVG [62.4300, 62.7500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 67 | 2025-08-07 10:00 | 68.1600 | 66.0647 | 2025-08-07 14:00 | 68.7200 | STOCH_EXIT | 8 | 909.39 | 0.72% | 67.8600 | 72.3505 | 2.0953 | M30 bullish FVG [67.5900, 67.8600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 68 | 2025-08-13 07:30 | 70.3900 | 68.3362 | 2025-08-13 17:30 | 70.0800 | STOCH_EXIT | 20 | -684.42 | -0.54% | 72.2400 | 74.4976 | 2.0538 | M30 bullish FVG [71.9800, 72.2400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 69 | 2025-08-15 10:30 | 69.1400 | 66.9700 | 2025-08-15 11:30 | 70.3600 | STOCH_EXIT | 2 | 2104.96 | 1.66% | 68.3700 | 73.4800 | 2.1700 | M30 bullish FVG [68.1100, 68.3700], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 70 | 2025-08-18 18:30 | 69.6100 | 67.3936 | 2025-08-18 21:30 | 70.5000 | STOCH_EXIT | 6 | 1508.70 | 1.18% | 69.9600 | 74.0429 | 2.2164 | M30 bullish FVG [69.6100, 69.9600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 71 | 2025-08-21 18:30 | 69.4100 | 67.2508 | 2025-08-22 08:00 | 69.7700 | STOCH_EXIT | 14 | 543.07 | 0.42% | 71.3800 | 73.7283 | 2.1592 | M30 bullish FVG [70.8300, 71.3800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 72 | 2025-08-27 09:00 | 73.4300 | 71.1838 | 2025-08-27 10:00 | 73.8400 | STOCH_EXIT | 2 | 595.37 | 0.46% | 73.1100 | 77.9223 | 2.2462 | M30 bullish FVG [72.5000, 73.1100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 73 | 2025-08-29 13:30 | 72.2700 | 70.0556 | 2025-08-29 21:00 | 72.1300 | STOCH_EXIT | 15 | -384.08 | -0.29% | 72.9200 | 76.6989 | 2.2144 | M30 bullish FVG [72.6200, 72.9200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 74 | 2025-09-03 11:00 | 69.9100 | 67.7030 | 2025-09-03 19:30 | 69.9800 | STOCH_EXIT | 17 | 0.10 | 0.00% | 70.5900 | 74.3241 | 2.2070 | M30 bullish FVG [70.3100, 70.5900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 75 | 2025-09-04 14:30 | 70.1400 | 67.9613 | 2025-09-04 22:00 | 69.8100 | STOCH_EXIT | 15 | -743.95 | -0.57% | 70.6400 | 74.4974 | 2.1787 | M30 bullish FVG [70.2900, 70.6400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 76 | 2025-09-09 09:30 | 69.8100 | 67.7383 | 2025-09-09 23:00 | 69.8900 | STOCH_EXIT | 27 | 18.88 | 0.01% | 69.8500 | 73.9534 | 2.0717 | M30 bullish FVG [69.6000, 69.8500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 77 | 2025-10-17 11:30 | 63.9600 | 61.2115 | 2025-10-17 18:00 | 64.3700 | STOCH_EXIT | 13 | 702.05 | 0.54% | 64.8500 | 69.4570 | 2.7485 | M30 bullish FVG [64.5000, 64.8500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 78 | 2025-10-21 07:30 | 65.5000 | 62.7796 | 2025-10-21 13:30 | 66.3400 | STOCH_EXIT | 12 | 1540.42 | 1.18% | 66.2600 | 70.9408 | 2.7204 | M30 bullish FVG [65.5600, 66.2600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 79 | 2025-10-21 21:00 | 63.2100 | 60.4896 | 2025-10-22 07:30 | 64.1500 | STOCH_EXIT | 8 | 1831.51 | 1.39% | 64.0100 | 68.6508 | 2.7204 | M30 bullish FVG [63.4400, 64.0100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 80 | 2025-10-22 13:30 | 63.5500 | 60.7139 | 2025-10-23 16:00 | 63.0000 | STOCH_EXIT | 40 | -1294.01 | -0.96% | 64.0100 | 69.2222 | 2.8361 | M30 bullish FVG [63.4400, 64.0100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 81 | 2025-11-07 18:30 | 62.2300 | 59.6232 | 2025-11-08 06:00 | 62.1400 | STOCH_EXIT | 19 | -324.15 | -0.24% | 63.1300 | 67.4437 | 2.6068 | M30 bullish FVG [62.6900, 63.1300], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 95840.90
- Макс: 136107.70
- Финал: 132699.78

Полная кривая — в `*_equity.csv`.
