# Бэктест MOEX (Minutes30)

- Период: 2023-11-12 22:16 — 2025-11-12 22:16
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
| HTFTrendEMA | 100 |
| RequireEntryAboveFVGLower | 1 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 43 |
| Выигрышных / проигрышных | 29 / 14 |
| Win rate | 67.44% |
| Gross profit / loss | 17463.33 / 12615.85 |
| Profit factor | 1.384 |
| Чистый PnL | 4847.48 (4.85%) |
| Стартовый / финальный капитал | 100000.00 / 104847.48 |
| Макс. просадка | 4099.91 (3.83%) |
| Средняя прибыль / убыток | 602.18 / 901.13 |
| Expectancy | 112.73 |
| Лучшая / худшая сделка | 1827.96 / -3104.70 |
| Exposure | 3.11% |
| CAGR | 2.39% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-11-22 19:00 | 206.2000 | 201.4410 | 2023-11-23 09:30 | 207.3200 | STOCH_EXIT | 10 | 438.36 | 0.44% | 205.1800 | 215.7180 | 4.7590 | M30 bullish FVG [204.5300, 205.1800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 2 | 2023-11-24 15:00 | 205.1800 | 200.7121 | 2023-11-24 20:30 | 205.7900 | STOCH_EXIT | 11 | 194.17 | 0.20% | 205.4200 | 214.1157 | 4.4679 | M30 bullish FVG [204.8700, 205.4200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 3 | 2023-11-29 11:30 | 202.0700 | 197.3831 | 2023-12-01 11:30 | 200.1600 | STOCH_EXIT | 58 | -1034.45 | -1.04% | 202.3400 | 211.4438 | 4.6869 | M30 bullish FVG [201.7900, 202.3400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 4 | 2023-12-01 18:30 | 199.1900 | 194.7164 | 2023-12-02 10:00 | 199.9600 | STOCH_EXIT | 11 | 279.51 | 0.29% | 199.3900 | 208.1372 | 4.4736 | M30 bullish FVG [198.7600, 199.3900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 5 | 2023-12-08 18:00 | 197.3700 | 192.5734 | 2023-12-09 17:30 | 198.1700 | STOCH_EXIT | 27 | 301.12 | 0.30% | 198.0700 | 206.9632 | 4.7966 | M30 bullish FVG [197.3300, 198.0700], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 6 | 2023-12-14 18:30 | 189.5000 | 184.3646 | 2023-12-14 23:30 | 192.4000 | STOCH_EXIT | 10 | 1408.71 | 1.43% | 190.1800 | 199.7708 | 5.1354 | M30 bullish FVG [189.1400, 190.1800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 7 | 2023-12-19 13:30 | 200.9100 | 195.4420 | 2023-12-19 20:30 | 200.4500 | STOCH_EXIT | 14 | -330.34 | -0.33% | 201.2300 | 211.8460 | 5.4680 | M30 bullish FVG [200.6600, 201.2300], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 8 | 2023-12-28 22:00 | 189.4000 | 184.2384 | 2023-12-29 15:30 | 190.5000 | STOCH_EXIT | 16 | 482.33 | 0.48% | 189.6600 | 199.7231 | 5.1616 | M30 bullish FVG [188.5400, 189.6600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 9 | 2024-01-04 10:30 | 195.0000 | 189.9379 | 2024-01-07 17:00 | 193.1300 | STOCH_EXIT | 88 | -1073.31 | -1.06% | 194.7400 | 205.1242 | 5.0621 | M30 bullish FVG [193.9800, 194.7400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 10 | 2024-01-15 18:30 | 193.5300 | 189.5941 | 2024-01-15 23:00 | 194.0700 | STOCH_EXIT | 9 | 176.56 | 0.18% | 193.9100 | 201.4018 | 3.9359 | M30 bullish FVG [193.4000, 193.9100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 11 | 2024-01-19 10:30 | 193.2400 | 189.6304 | 2024-01-19 21:30 | 192.9600 | STOCH_EXIT | 22 | -246.01 | -0.24% | 194.2700 | 200.4592 | 3.6096 | M30 bullish FVG [193.1800, 194.2700], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 12 | 2024-01-23 12:30 | 198.7900 | 194.9756 | 2024-01-24 18:30 | 197.7400 | STOCH_EXIT | 41 | -624.13 | -0.63% | 198.9000 | 206.4187 | 3.8144 | M30 bullish FVG [198.4700, 198.9000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 13 | 2024-01-29 10:00 | 198.6100 | 194.8848 | 2024-01-29 10:00 | 199.7500 | STOCH_EXIT | 0 | 470.41 | 0.47% | 198.5100 | 206.0604 | 3.7252 | M30 bullish FVG [197.9700, 198.5100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 14 | 2024-01-30 14:00 | 203.8000 | 199.5059 | 2024-01-31 17:30 | 203.8000 | STOCH_EXIT | 36 | -99.86 | -0.10% | 203.6100 | 212.3882 | 4.2941 | M30 bullish FVG [201.4700, 203.6100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 15 | 2024-02-19 21:30 | 197.2800 | 193.5435 | 2024-02-20 19:00 | 193.5435 | SL | 24 | -1965.96 | -1.99% | 197.0000 | 204.7530 | 3.7365 | M30 bullish FVG [196.5000, 197.0000], stochastic crossed below 20 | FVG stop 193.5435 |
| 16 | 2024-03-01 11:30 | 199.2100 | 194.6388 | 2024-03-01 18:30 | 199.8000 | STOCH_EXIT | 14 | 191.34 | 0.20% | 200.0000 | 208.3523 | 4.5712 | M30 bullish FVG [199.1900, 200.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 17 | 2024-03-06 19:00 | 205.2100 | 200.6654 | 2024-03-07 10:00 | 206.7200 | STOCH_EXIT | 11 | 625.94 | 0.64% | 202.6600 | 214.2991 | 4.5446 | M30 bullish FVG [200.8500, 202.6600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 18 | 2024-03-19 10:30 | 208.4100 | 204.5028 | 2024-03-19 18:30 | 209.6000 | STOCH_EXIT | 16 | 461.07 | 0.47% | 208.9700 | 216.2245 | 3.9072 | M30 bullish FVG [208.3400, 208.9700], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 19 | 2024-03-20 15:30 | 208.8300 | 205.0233 | 2024-03-20 17:30 | 210.6600 | STOCH_EXIT | 4 | 761.52 | 0.78% | 209.5600 | 216.4434 | 3.8067 | M30 bullish FVG [208.6000, 209.5600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 20 | 2024-03-21 15:30 | 212.3900 | 208.4631 | 2024-03-22 10:30 | 213.5700 | STOCH_EXIT | 19 | 454.50 | 0.46% | 212.3200 | 220.2439 | 3.9269 | M30 bullish FVG [211.8000, 212.3200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 21 | 2024-04-03 16:30 | 228.5100 | 224.2611 | 2024-04-03 22:30 | 229.0600 | STOCH_EXIT | 12 | 141.33 | 0.14% | 226.5500 | 237.0078 | 4.2489 | M30 bullish FVG [226.0700, 226.5500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 22 | 2024-04-19 10:30 | 228.0000 | 224.1453 | 2024-04-19 18:30 | 227.9000 | STOCH_EXIT | 16 | -144.30 | -0.14% | 228.1200 | 235.7094 | 3.8547 | M30 bullish FVG [227.6700, 228.1200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 23 | 2024-05-30 14:00 | 244.8400 | 240.3371 | 2024-05-31 15:00 | 244.7400 | STOCH_EXIT | 31 | -141.36 | -0.14% | 243.2100 | 253.8458 | 4.5029 | M30 bullish FVG [242.7000, 243.2100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 24 | 2024-06-03 13:30 | 240.3100 | 235.0161 | 2024-06-03 14:00 | 235.0161 | SL | 1 | -2267.92 | -2.30% | 242.1300 | 250.8977 | 5.2939 | M30 bullish FVG [240.0000, 242.1300], stochastic crossed below 20 | FVG stop 235.0161 |
| 25 | 2024-06-05 14:30 | 247.9200 | 241.7990 | 2024-06-05 17:30 | 249.8300 | STOCH_EXIT | 6 | 647.84 | 0.67% | 248.1600 | 260.1621 | 6.1210 | M30 bullish FVG [247.2000, 248.1600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 26 | 2024-06-14 10:00 | 235.4100 | 227.0457 | 2024-06-14 16:00 | 240.0000 | STOCH_EXIT | 12 | 1827.96 | 1.85% | 234.0000 | 252.1385 | 8.3643 | M30 bullish FVG [226.0400, 234.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 27 | 2024-07-03 11:30 | 237.8000 | 230.5481 | 2024-07-03 13:00 | 239.6400 | STOCH_EXIT | 3 | 672.54 | 0.67% | 235.8900 | 252.3038 | 7.2519 | M30 bullish FVG [235.0000, 235.8900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 28 | 2024-07-17 22:30 | 227.5200 | 220.4140 | 2024-07-18 17:00 | 228.2500 | STOCH_EXIT | 18 | 220.93 | 0.22% | 227.1900 | 241.7320 | 7.1060 | M30 bullish FVG [226.3800, 227.1900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 29 | 2024-07-26 10:00 | 232.1600 | 226.0930 | 2024-07-26 12:00 | 235.4800 | STOCH_EXIT | 4 | 1327.06 | 1.33% | 233.1600 | 244.2939 | 6.0670 | M30 bullish FVG [231.8000, 233.1600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 30 | 2024-08-13 12:00 | 227.0000 | 221.2650 | 2024-08-13 22:30 | 226.6700 | STOCH_EXIT | 21 | -250.58 | -0.25% | 225.4200 | 238.4700 | 5.7350 | M30 bullish FVG [224.8000, 225.4200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 31 | 2025-01-15 12:00 | 205.7200 | 198.6047 | 2025-01-15 19:00 | 208.0000 | STOCH_EXIT | 14 | 1015.84 | 1.01% | 203.5500 | 219.9507 | 7.1153 | M30 bullish FVG [202.6600, 203.5500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 32 | 2025-01-29 11:00 | 207.6000 | 201.6950 | 2025-01-29 11:00 | 208.9300 | STOCH_EXIT | 0 | 560.87 | 0.54% | 206.8800 | 219.4099 | 5.9050 | M30 bullish FVG [205.6400, 206.8800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 33 | 2025-02-06 10:00 | 204.9400 | 199.1592 | 2025-02-07 07:30 | 204.5900 | STOCH_EXIT | 30 | -277.38 | -0.27% | 201.1500 | 216.5016 | 5.7808 | M30 bullish FVG [200.1600, 201.1500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 34 | 2025-02-10 23:30 | 206.0800 | 200.7456 | 2025-02-11 12:00 | 208.3600 | STOCH_EXIT | 12 | 1036.39 | 1.01% | 206.5000 | 216.7487 | 5.3344 | M30 bullish FVG [205.3000, 206.5000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 35 | 2025-02-12 15:00 | 214.9000 | 209.2940 | 2025-02-12 19:30 | 217.2200 | STOCH_EXIT | 9 | 1009.89 | 0.98% | 212.5200 | 226.1120 | 5.6060 | M30 bullish FVG [210.7800, 212.5200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 36 | 2025-02-14 12:30 | 233.8600 | 227.1912 | 2025-02-14 16:00 | 227.1912 | SL | 7 | -3104.70 | -2.95% | 234.0900 | 247.1976 | 6.6688 | M30 bullish FVG [230.1600, 234.0900], stochastic crossed below 20 | FVG stop 227.1912 |
| 37 | 2025-02-17 14:30 | 233.3600 | 225.5797 | 2025-02-17 17:00 | 235.3600 | STOCH_EXIT | 5 | 776.88 | 0.76% | 234.0200 | 248.9206 | 7.7803 | M30 bullish FVG [232.1400, 234.0200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 38 | 2025-02-18 20:30 | 236.7700 | 229.0033 | 2025-02-19 07:00 | 238.5100 | STOCH_EXIT | 8 | 646.01 | 0.63% | 237.5000 | 252.3034 | 7.7667 | M30 bullish FVG [236.5600, 237.5000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 39 | 2025-03-04 14:00 | 222.2700 | 214.6255 | 2025-03-05 18:30 | 222.5500 | STOCH_EXIT | 44 | 27.07 | 0.03% | 218.6500 | 237.5590 | 7.6445 | M30 bullish FVG [216.5900, 218.6500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 40 | 2025-03-20 13:30 | 215.9000 | 208.7689 | 2025-03-20 17:30 | 216.1600 | STOCH_EXIT | 8 | 21.11 | 0.02% | 216.6000 | 230.1622 | 7.1311 | M30 bullish FVG [215.8600, 216.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 41 | 2025-04-23 09:00 | 202.4800 | 195.4284 | 2025-04-23 17:00 | 203.9200 | STOCH_EXIT | 16 | 630.77 | 0.61% | 203.2000 | 216.5833 | 7.0516 | M30 bullish FVG [202.0700, 203.2000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 42 | 2025-04-24 19:00 | 208.5900 | 201.5749 | 2025-04-24 20:00 | 210.1100 | STOCH_EXIT | 2 | 655.33 | 0.63% | 209.4800 | 222.6202 | 7.0151 | M30 bullish FVG [208.2800, 209.4800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 43 | 2025-07-03 14:30 | 200.8200 | 196.8758 | 2025-07-04 09:00 | 198.9900 | STOCH_EXIT | 24 | -1055.55 | -1.01% | 201.1000 | 208.7084 | 3.9442 | M30 bullish FVG [200.3400, 201.1000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 97997.46
- Макс: 107088.94
- Финал: 104847.48

Полная кривая — в `*_equity.csv`.
