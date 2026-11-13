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
| StochPeriod | 9 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 0 |
| HTFTrendEMA | 50 |
| RequireEntryAboveFVGLower | 1 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 67 |
| Выигрышных / проигрышных | 43 / 24 |
| Win rate | 64.18% |
| Gross profit / loss | 22748.81 / 18118.16 |
| Profit factor | 1.256 |
| Чистый PnL | 4630.65 (4.63%) |
| Стартовый / финальный капитал | 100000.00 / 104630.65 |
| Макс. просадка | 6387.78 (5.77%) |
| Средняя прибыль / убыток | 529.04 / 754.92 |
| Expectancy | 69.11 |
| Лучшая / худшая сделка | 1626.51 / -3185.81 |
| Exposure | 2.80% |
| CAGR | 2.29% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-11-14 21:30 | 198.5500 | 193.5003 | 2023-11-15 14:00 | 199.1100 | STOCH_EXIT | 14 | 180.59 | 0.18% | 199.3500 | 0.0000 | 5.0497 | M30 bullish FVG [198.5000, 199.3500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 2 | 2023-11-21 23:00 | 205.1500 | 200.3749 | 2023-11-22 09:30 | 206.1900 | STOCH_EXIT | 2 | 400.48 | 0.41% | 205.1800 | 0.0000 | 4.7751 | M30 bullish FVG [204.5300, 205.1800], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 3 | 2023-11-23 20:00 | 204.9800 | 200.4131 | 2023-11-23 21:30 | 205.5700 | STOCH_EXIT | 3 | 188.52 | 0.19% | 205.4200 | 0.0000 | 4.5669 | M30 bullish FVG [204.8700, 205.4200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 4 | 2023-11-29 19:00 | 202.0000 | 197.3131 | 2023-11-30 15:00 | 200.0300 | STOCH_EXIT | 21 | -1063.80 | -1.07% | 202.3400 | 0.0000 | 4.6869 | M30 bullish FVG [201.7900, 202.3400], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 5 | 2023-12-01 18:30 | 199.1900 | 194.7164 | 2023-12-02 10:00 | 199.9600 | STOCH_EXIT | 11 | 285.21 | 0.29% | 199.3900 | 0.0000 | 4.4736 | M30 bullish FVG [198.7600, 199.3900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 6 | 2023-12-04 19:30 | 200.5000 | 196.1466 | 2023-12-04 21:30 | 201.4100 | STOCH_EXIT | 4 | 347.43 | 0.35% | 200.7600 | 0.0000 | 4.3534 | M30 bullish FVG [200.2500, 200.7600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 7 | 2023-12-08 15:30 | 198.4000 | 193.6034 | 2023-12-08 21:30 | 198.4400 | STOCH_EXIT | 12 | -79.21 | -0.08% | 198.0700 | 0.0000 | 4.7966 | M30 bullish FVG [197.3300, 198.0700], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 8 | 2023-12-19 13:30 | 200.9100 | 195.4420 | 2023-12-19 20:00 | 200.4400 | STOCH_EXIT | 13 | -328.63 | -0.33% | 201.2300 | 0.0000 | 5.4680 | M30 bullish FVG [200.6600, 201.2300], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 9 | 2024-01-04 10:30 | 195.0000 | 189.9379 | 2024-01-04 18:30 | 194.2700 | STOCH_EXIT | 16 | -471.56 | -0.47% | 194.7400 | 0.0000 | 5.0621 | M30 bullish FVG [193.9800, 194.7400], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 10 | 2024-01-23 12:30 | 198.7900 | 194.9756 | 2024-01-24 09:30 | 198.0900 | STOCH_EXIT | 23 | -449.22 | -0.45% | 198.9000 | 0.0000 | 3.8144 | M30 bullish FVG [198.4700, 198.9000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 11 | 2024-01-30 11:30 | 204.2800 | 199.9859 | 2024-01-30 16:00 | 204.1400 | STOCH_EXIT | 9 | -165.22 | -0.17% | 205.1500 | 0.0000 | 4.2941 | M30 bullish FVG [204.0000, 205.1500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 12 | 2024-02-29 18:00 | 199.3200 | 194.6564 | 2024-03-01 09:30 | 200.0700 | STOCH_EXIT | 12 | 269.65 | 0.28% | 200.0000 | 0.0000 | 4.6636 | M30 bullish FVG [199.1900, 200.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 13 | 2024-03-05 18:30 | 200.8300 | 196.5589 | 2024-03-05 19:00 | 202.7700 | STOCH_EXIT | 1 | 851.72 | 0.87% | 200.6500 | 0.0000 | 4.2711 | M30 bullish FVG [200.1900, 200.6500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 14 | 2024-03-06 10:30 | 205.1400 | 200.5954 | 2024-03-06 15:00 | 206.3000 | STOCH_EXIT | 9 | 458.05 | 0.46% | 202.6600 | 0.0000 | 4.5446 | M30 bullish FVG [200.8500, 202.6600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 15 | 2024-03-15 23:00 | 209.8700 | 206.0000 | 2024-03-16 14:30 | 211.2000 | STOCH_EXIT | 11 | 526.15 | 0.53% | 209.2000 | 0.0000 | 3.8700 | M30 bullish FVG [208.5000, 209.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 16 | 2024-03-19 10:30 | 208.4100 | 204.5028 | 2024-03-19 18:30 | 209.6000 | STOCH_EXIT | 16 | 470.88 | 0.47% | 208.9700 | 0.0000 | 3.9072 | M30 bullish FVG [208.3400, 208.9700], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 17 | 2024-03-20 15:00 | 209.3600 | 205.5533 | 2024-03-20 17:30 | 210.6600 | STOCH_EXIT | 5 | 523.20 | 0.52% | 209.5600 | 0.0000 | 3.8067 | M30 bullish FVG [208.6000, 209.5600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 18 | 2024-03-21 13:00 | 212.9800 | 209.0531 | 2024-03-22 10:30 | 213.5700 | STOCH_EXIT | 24 | 177.06 | 0.18% | 212.3200 | 0.0000 | 3.9269 | M30 bullish FVG [211.8000, 212.3200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 19 | 2024-03-28 13:00 | 219.7300 | 215.6655 | 2024-03-28 18:30 | 219.0000 | STOCH_EXIT | 11 | -436.71 | -0.43% | 219.0900 | 0.0000 | 4.0645 | M30 bullish FVG [218.5800, 219.0900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 20 | 2024-03-28 23:00 | 218.2400 | 214.1755 | 2024-03-29 11:30 | 218.8300 | STOCH_EXIT | 6 | 170.87 | 0.17% | 218.3300 | 0.0000 | 4.0645 | M30 bullish FVG [217.8800, 218.3300], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 21 | 2024-04-03 15:30 | 228.8200 | 224.5711 | 2024-04-03 20:00 | 229.0800 | STOCH_EXIT | 9 | 13.66 | 0.01% | 226.5500 | 0.0000 | 4.2489 | M30 bullish FVG [226.0700, 226.5500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 22 | 2024-04-19 10:00 | 228.2700 | 224.4153 | 2024-04-19 18:30 | 227.9000 | STOCH_EXIT | 17 | -263.16 | -0.26% | 228.1200 | 0.0000 | 3.8547 | M30 bullish FVG [227.6700, 228.1200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 23 | 2024-04-26 23:30 | 230.2700 | 226.7790 | 2024-04-27 10:00 | 232.3900 | STOCH_EXIT | 2 | 831.01 | 0.82% | 230.3600 | 0.0000 | 3.4910 | M30 bullish FVG [229.9000, 230.3600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 24 | 2024-05-03 16:30 | 234.5000 | 230.5939 | 2024-05-03 17:30 | 235.0300 | STOCH_EXIT | 2 | 126.95 | 0.13% | 234.9700 | 0.0000 | 3.9061 | M30 bullish FVG [234.3800, 234.9700], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 25 | 2024-05-10 22:30 | 232.7300 | 228.6869 | 2024-05-10 23:30 | 233.1800 | STOCH_EXIT | 2 | 95.50 | 0.09% | 232.7100 | 0.0000 | 4.0431 | M30 bullish FVG [232.0200, 232.7100], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 26 | 2024-05-14 22:30 | 232.8000 | 228.9224 | 2024-05-15 10:00 | 234.5100 | STOCH_EXIT | 4 | 649.59 | 0.63% | 232.7700 | 0.0000 | 3.8776 | M30 bullish FVG [232.0800, 232.7700], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 27 | 2024-05-15 14:00 | 233.0200 | 229.2673 | 2024-05-15 18:00 | 232.6000 | STOCH_EXIT | 8 | -287.24 | -0.28% | 233.7600 | 0.0000 | 3.7527 | M30 bullish FVG [233.0200, 233.7600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 28 | 2024-05-16 23:30 | 236.6300 | 232.8518 | 2024-05-17 10:00 | 237.7300 | STOCH_EXIT | 2 | 371.01 | 0.36% | 237.0000 | 0.0000 | 3.7782 | M30 bullish FVG [236.0000, 237.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 29 | 2024-05-30 14:00 | 244.8400 | 240.3371 | 2024-05-31 14:30 | 244.5400 | STOCH_EXIT | 30 | -228.77 | -0.22% | 243.2100 | 0.0000 | 4.5029 | M30 bullish FVG [242.7000, 243.2100], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 30 | 2024-06-03 13:30 | 240.3100 | 235.0161 | 2024-06-03 14:00 | 235.0161 | SL | 1 | -2323.24 | -2.30% | 242.1300 | 0.0000 | 5.2939 | M30 bullish FVG [240.0000, 242.1300], stochastic crossed below 20 | FVG stop 235.0161 |
| 31 | 2024-06-05 12:30 | 247.8700 | 241.7490 | 2024-06-05 17:30 | 249.8300 | STOCH_EXIT | 10 | 684.46 | 0.69% | 248.1600 | 0.0000 | 6.1210 | M30 bullish FVG [247.2000, 248.1600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 32 | 2024-06-14 10:00 | 235.4100 | 227.0457 | 2024-06-14 15:30 | 239.4300 | STOCH_EXIT | 11 | 1626.51 | 1.61% | 234.0000 | 0.0000 | 8.3643 | M30 bullish FVG [226.0400, 234.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 33 | 2024-06-14 21:30 | 239.3100 | 230.9457 | 2024-06-14 23:00 | 240.2300 | STOCH_EXIT | 3 | 292.50 | 0.28% | 239.4300 | 0.0000 | 8.3643 | M30 bullish FVG [238.5000, 239.4300], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 34 | 2024-07-03 11:30 | 237.8000 | 230.5481 | 2024-07-03 13:00 | 239.6400 | STOCH_EXIT | 3 | 688.55 | 0.67% | 235.8900 | 0.0000 | 7.2519 | M30 bullish FVG [235.0000, 235.8900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 35 | 2024-07-05 21:30 | 234.7300 | 227.7165 | 2024-07-05 23:30 | 235.7000 | STOCH_EXIT | 4 | 323.31 | 0.31% | 233.6900 | 0.0000 | 7.0135 | M30 bullish FVG [232.0400, 233.6900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 36 | 2024-07-25 22:30 | 231.9200 | 225.7625 | 2024-07-26 12:00 | 235.4800 | STOCH_EXIT | 8 | 1496.84 | 1.43% | 233.1600 | 0.0000 | 6.1575 | M30 bullish FVG [231.8000, 233.1600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 37 | 2024-09-24 10:30 | 216.3100 | 210.1682 | 2024-09-24 17:30 | 216.3900 | STOCH_EXIT | 14 | -65.45 | -0.06% | 217.0000 | 0.0000 | 6.1418 | M30 bullish FVG [215.4400, 217.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 38 | 2024-09-25 17:30 | 216.9400 | 210.8319 | 2024-09-25 22:30 | 216.9400 | STOCH_EXIT | 10 | -104.13 | -0.10% | 216.5000 | 0.0000 | 6.1081 | M30 bullish FVG [215.3000, 216.5000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 39 | 2024-11-14 15:00 | 207.1800 | 200.6135 | 2024-11-14 23:30 | 207.0000 | STOCH_EXIT | 17 | -197.42 | -0.19% | 207.6400 | 0.0000 | 6.5665 | M30 bullish FVG [206.7500, 207.6400], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 40 | 2024-11-18 15:30 | 205.1900 | 198.7367 | 2024-11-18 20:00 | 206.8700 | STOCH_EXIT | 9 | 751.72 | 0.72% | 206.4600 | 0.0000 | 6.4533 | M30 bullish FVG [203.4900, 206.4600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 41 | 2024-12-26 12:00 | 195.3000 | 187.7212 | 2024-12-26 18:30 | 195.8300 | STOCH_EXIT | 13 | 180.59 | 0.17% | 193.9500 | 0.0000 | 7.5788 | M30 bullish FVG [193.0400, 193.9500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 42 | 2024-12-30 16:30 | 198.0100 | 190.7384 | 2024-12-31 10:30 | 200.4800 | STOCH_EXIT | 22 | 1203.50 | 1.15% | 198.6800 | 0.0000 | 7.2716 | M30 bullish FVG [197.2000, 198.6800], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 43 | 2025-01-15 12:00 | 205.7200 | 198.6047 | 2025-01-15 19:00 | 208.0000 | STOCH_EXIT | 14 | 1078.03 | 1.01% | 203.5500 | 0.0000 | 7.1153 | M30 bullish FVG [202.6600, 203.5500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 44 | 2025-01-24 17:00 | 209.3800 | 203.2996 | 2025-01-24 18:00 | 210.9800 | STOCH_EXIT | 2 | 708.81 | 0.66% | 208.1300 | 0.0000 | 6.0804 | M30 bullish FVG [207.3800, 208.1300], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 45 | 2025-01-29 11:00 | 207.6000 | 201.6950 | 2025-01-29 11:00 | 208.9300 | STOCH_EXIT | 0 | 583.30 | 0.54% | 206.8800 | 0.0000 | 5.9050 | M30 bullish FVG [205.6400, 206.8800], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 46 | 2025-02-03 15:30 | 203.4200 | 197.9636 | 2025-02-03 16:00 | 204.0500 | STOCH_EXIT | 1 | 230.18 | 0.21% | 204.0300 | 0.0000 | 5.4564 | M30 bullish FVG [203.4000, 204.0300], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 47 | 2025-02-06 09:30 | 205.1800 | 199.3992 | 2025-02-06 17:30 | 204.7000 | STOCH_EXIT | 16 | -363.02 | -0.33% | 201.1500 | 0.0000 | 5.7808 | M30 bullish FVG [200.1600, 201.1500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 48 | 2025-02-10 22:30 | 208.8200 | 203.4856 | 2025-02-11 09:30 | 208.1800 | STOCH_EXIT | 9 | -441.22 | -0.41% | 208.1100 | 0.0000 | 5.3344 | M30 bullish FVG [207.1500, 208.1100], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 49 | 2025-02-12 13:00 | 215.6400 | 210.0340 | 2025-02-12 19:00 | 216.1900 | STOCH_EXIT | 12 | 167.04 | 0.15% | 212.5200 | 0.0000 | 5.6060 | M30 bullish FVG [210.7800, 212.5200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 50 | 2025-02-13 19:00 | 234.0000 | 227.8351 | 2025-02-14 07:30 | 236.4700 | STOCH_EXIT | 5 | 1027.99 | 0.95% | 234.0900 | 0.0000 | 6.1649 | M30 bullish FVG [230.1600, 234.0900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 51 | 2025-02-14 12:00 | 236.9000 | 230.2312 | 2025-02-14 14:00 | 230.2312 | SL | 4 | -3175.09 | -2.91% | 238.2500 | 0.0000 | 6.6688 | M30 bullish FVG [236.8000, 238.2500], stochastic crossed below 20 | FVG stop 230.2312 |
| 52 | 2025-02-17 14:30 | 233.3600 | 225.5797 | 2025-02-17 15:30 | 234.6500 | STOCH_EXIT | 2 | 485.76 | 0.45% | 234.0200 | 0.0000 | 7.7803 | M30 bullish FVG [232.1400, 234.0200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 53 | 2025-02-18 18:00 | 237.7600 | 229.9933 | 2025-02-18 23:30 | 237.9800 | STOCH_EXIT | 11 | -8.04 | -0.01% | 237.5000 | 0.0000 | 7.7667 | M30 bullish FVG [236.5600, 237.5000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 54 | 2025-02-27 18:00 | 220.6900 | 213.1109 | 2025-02-28 08:30 | 218.0600 | STOCH_EXIT | 16 | -1367.70 | -1.29% | 221.3000 | 0.0000 | 7.5791 | M30 bullish FVG [220.3900, 221.3000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 55 | 2025-03-04 11:00 | 223.8500 | 216.2055 | 2025-03-05 18:30 | 222.5500 | STOCH_EXIT | 50 | -715.90 | -0.68% | 218.6500 | 0.0000 | 7.6445 | M30 bullish FVG [216.5900, 218.6500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 56 | 2025-03-19 21:00 | 216.0000 | 208.7311 | 2025-03-20 06:30 | 217.1700 | STOCH_EXIT | 6 | 467.17 | 0.44% | 216.6000 | 0.0000 | 7.2689 | M30 bullish FVG [215.8600, 216.6000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 57 | 2025-04-23 09:00 | 202.4800 | 195.4284 | 2025-04-23 13:00 | 203.8000 | STOCH_EXIT | 8 | 580.77 | 0.55% | 203.2000 | 0.0000 | 7.0516 | M30 bullish FVG [202.0700, 203.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 58 | 2025-04-24 16:30 | 209.1500 | 202.1349 | 2025-04-24 19:30 | 210.0200 | STOCH_EXIT | 6 | 336.81 | 0.32% | 209.4800 | 0.0000 | 7.0151 | M30 bullish FVG [208.2800, 209.4800], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 59 | 2025-06-25 15:30 | 197.0500 | 192.6769 | 2025-06-25 17:30 | 198.7900 | STOCH_EXIT | 4 | 832.72 | 0.78% | 195.1500 | 0.0000 | 4.3731 | M30 bullish FVG [194.4900, 195.1500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 60 | 2025-07-03 14:30 | 200.8200 | 196.8758 | 2025-07-04 09:00 | 198.9900 | STOCH_EXIT | 24 | -1075.85 | -1.01% | 201.1000 | 0.0000 | 3.9442 | M30 bullish FVG [200.3400, 201.1000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 61 | 2025-08-15 10:30 | 186.5100 | 182.2521 | 2025-08-15 11:30 | 187.7000 | STOCH_EXIT | 2 | 571.65 | 0.54% | 186.3400 | 0.0000 | 4.2579 | M30 bullish FVG [185.7600, 186.3400], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 62 | 2025-08-15 15:30 | 186.6400 | 182.3821 | 2025-08-15 18:00 | 187.5300 | STOCH_EXIT | 5 | 400.66 | 0.38% | 187.5200 | 0.0000 | 4.2579 | M30 bullish FVG [186.6200, 187.5200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 63 | 2025-08-18 18:30 | 185.0600 | 180.9591 | 2025-08-18 21:30 | 187.0000 | STOCH_EXIT | 6 | 1017.30 | 0.95% | 186.0400 | 0.0000 | 4.1009 | M30 bullish FVG [185.0500, 186.0400], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 64 | 2025-08-19 16:00 | 186.8200 | 182.6291 | 2025-08-20 07:30 | 186.8900 | STOCH_EXIT | 18 | -67.78 | -0.06% | 186.3300 | 0.0000 | 4.1909 | M30 bullish FVG [185.5100, 186.3300], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 65 | 2025-08-25 22:30 | 182.5700 | 178.5762 | 2025-08-26 07:00 | 182.8800 | STOCH_EXIT | 4 | 75.09 | 0.07% | 182.7500 | 0.0000 | 3.9938 | M30 bullish FVG [182.0300, 182.7500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 66 | 2025-10-21 07:00 | 173.5200 | 168.8920 | 2025-10-21 13:30 | 171.6700 | STOCH_EXIT | 13 | -1254.01 | -1.17% | 171.7100 | 0.0000 | 4.6280 | M30 bullish FVG [171.1400, 171.7100], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 67 | 2025-10-22 18:30 | 170.4600 | 165.5712 | 2025-10-22 23:00 | 165.5712 | SL | 9 | -3185.81 | -2.97% | 170.4700 | 0.0000 | 4.8888 | M30 bullish FVG [169.9700, 170.4700], stochastic crossed below 20 | FVG stop 165.5712 |

## Движение капитала

- Старт: 100000.00
- Мин: 98360.77
- Макс: 110712.89
- Финал: 104630.65

Полная кривая — в `*_equity.csv`.
