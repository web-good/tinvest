# Бэктест IVAT (Minutes30)

- Период: 2025-11-14 00:02 — 2026-11-14 00:02
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005
- ⚠️ На конце прогона осталась открытая позиция (оценена mark-to-market)

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0 |
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
| Всего сделок | 168 |
| Выигрышных / проигрышных | 98 / 70 |
| Win rate | 58.33% |
| Gross profit / loss | 103642.99 / 96057.04 |
| Profit factor | 1.079 |
| Чистый PnL | 7610.15 (7.61%) |
| Стартовый / финальный капитал | 100000.00 / 107610.15 |
| Макс. просадка | 37274.56 (33.94%) |
| Средняя прибыль / убыток | 1057.58 / 1372.24 |
| Expectancy | 45.15 |
| Лучшая / худшая сделка | 10104.80 / -8399.95 |
| Exposure | 30.10% |
| CAGR | 7.61% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2025-11-17 13:30 | 159.5500 | 153.1352 | 2025-11-17 21:00 | 160.6000 | STOCH_EXIT | 15 | 557.09 | 0.56% | 160.2000 | 172.3796 | 6.4148 | M30 bullish FVG [159.5000, 160.2000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 2 | 2025-11-17 23:30 | 159.4000 | 152.9852 | 2025-11-18 11:00 | 160.5500 | STOCH_EXIT | 10 | 623.72 | 0.62% | 159.9500 | 172.2296 | 6.4148 | M30 bullish FVG [159.7500, 159.9500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 3 | 2025-11-18 20:30 | 159.7500 | 153.5291 | 2025-11-18 22:00 | 160.5500 | STOCH_EXIT | 3 | 405.03 | 0.40% | 160.0500 | 172.1918 | 6.2209 | M30 bullish FVG [160.0000, 160.0500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 4 | 2025-11-20 11:30 | 161.9500 | 156.0126 | 2025-11-20 20:30 | 162.7500 | STOCH_EXIT | 18 | 399.17 | 0.39% | 163.2500 | 173.8248 | 5.9374 | M30 bullish FVG [161.7000, 163.2500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 5 | 2025-11-24 12:00 | 162.2000 | 156.5614 | 2025-11-25 06:30 | 163.0000 | STOCH_EXIT | 24 | 400.29 | 0.39% | 163.6000 | 173.4772 | 5.6386 | M30 bullish FVG [162.6000, 163.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 6 | 2025-11-25 18:00 | 165.8000 | 160.3391 | 2025-11-26 06:30 | 167.7500 | STOCH_EXIT | 12 | 1100.25 | 1.07% | 167.6000 | 176.7217 | 5.4609 | M30 bullish FVG [167.3500, 167.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 7 | 2025-11-26 10:30 | 165.3500 | 159.5721 | 2025-11-26 19:00 | 165.6500 | STOCH_EXIT | 17 | 84.06 | 0.08% | 166.6000 | 176.9059 | 5.7779 | M30 bullish FVG [166.4000, 166.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 8 | 2025-11-27 15:30 | 169.0000 | 163.3848 | 2025-11-27 22:00 | 169.7000 | STOCH_EXIT | 13 | 324.76 | 0.31% | 171.7500 | 180.2305 | 5.6152 | M30 bullish FVG [171.5000, 171.7500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 9 | 2025-11-28 14:00 | 170.8500 | 165.0073 | 2025-11-28 16:30 | 172.6500 | STOCH_EXIT | 5 | 988.35 | 0.95% | 172.2500 | 182.5354 | 5.8427 | M30 bullish FVG [171.2500, 172.2500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 10 | 2025-12-01 07:30 | 170.9000 | 165.0996 | 2025-12-01 19:00 | 171.5500 | STOCH_EXIT | 23 | 293.49 | 0.28% | 170.8500 | 182.5008 | 5.8004 | M30 bullish FVG [170.7500, 170.8500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 11 | 2025-12-01 23:00 | 170.6000 | 164.7996 | 2025-12-01 23:00 | 171.7000 | STOCH_EXIT | 0 | 572.17 | 0.54% | 171.3000 | 182.2008 | 5.8004 | M30 bullish FVG [171.2500, 171.3000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 12 | 2025-12-02 14:30 | 172.4500 | 166.8318 | 2025-12-03 19:30 | 169.7000 | STOCH_EXIT | 45 | -1787.70 | -1.69% | 172.8000 | 183.6864 | 5.6182 | M30 bullish FVG [172.0000, 172.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 13 | 2025-12-04 10:00 | 169.3000 | 163.6925 | 2025-12-05 09:00 | 169.2000 | STOCH_EXIT | 33 | -165.05 | -0.16% | 169.5000 | 180.5151 | 5.6075 | M30 bullish FVG [168.8500, 169.5000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 14 | 2025-12-09 11:00 | 167.5000 | 162.4822 | 2025-12-09 20:00 | 167.5500 | STOCH_EXIT | 18 | -72.75 | -0.07% | 168.4000 | 177.5355 | 5.0178 | M30 bullish FVG [168.3000, 168.4000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 15 | 2025-12-10 10:30 | 166.6000 | 161.7156 | 2025-12-10 10:30 | 168.5000 | STOCH_EXIT | 0 | 1077.58 | 1.04% | 167.5000 | 176.3687 | 4.8844 | M30 bullish FVG [167.4000, 167.5000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 16 | 2025-12-10 17:30 | 166.9500 | 162.0656 | 2025-12-10 21:00 | 169.2000 | STOCH_EXIT | 7 | 1305.37 | 1.25% | 167.5000 | 176.7187 | 4.8844 | M30 bullish FVG [167.3000, 167.5000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 17 | 2025-12-11 18:30 | 167.6500 | 162.8610 | 2025-12-12 06:30 | 168.3000 | STOCH_EXIT | 11 | 304.64 | 0.29% | 168.9500 | 177.2281 | 4.7890 | M30 bullish FVG [168.5500, 168.9500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 18 | 2025-12-12 15:00 | 168.0000 | 163.2852 | 2025-12-13 09:00 | 168.4500 | STOCH_EXIT | 32 | 178.36 | 0.17% | 169.2000 | 177.4297 | 4.7148 | M30 bullish FVG [168.9000, 169.2000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 19 | 2025-12-15 17:00 | 171.3000 | 166.6541 | 2025-12-15 20:00 | 173.6000 | STOCH_EXIT | 6 | 1321.21 | 1.24% | 171.4500 | 180.5918 | 4.6459 | M30 bullish FVG [169.1500, 171.4500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 20 | 2025-12-16 12:00 | 172.8500 | 167.8717 | 2025-12-16 18:00 | 173.2500 | STOCH_EXIT | 12 | 141.39 | 0.13% | 174.1000 | 182.8067 | 4.9783 | M30 bullish FVG [174.0500, 174.1000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 21 | 2025-12-18 17:30 | 174.1500 | 169.2962 | 2025-12-18 23:30 | 175.1500 | STOCH_EXIT | 12 | 511.72 | 0.47% | 175.4000 | 183.8576 | 4.8538 | M30 bullish FVG [172.7000, 175.4000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 22 | 2025-12-19 11:30 | 174.6000 | 169.6893 | 2025-12-20 02:00 | 174.0000 | STOCH_EXIT | 25 | -480.84 | -0.44% | 175.7000 | 184.4213 | 4.9107 | M30 bullish FVG [175.5000, 175.7000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 23 | 2025-12-22 17:30 | 169.6000 | 164.6723 | 2025-12-23 06:30 | 169.7500 | STOCH_EXIT | 13 | -12.51 | -0.01% | 170.3500 | 179.4555 | 4.9277 | M30 bullish FVG [169.9000, 170.3500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 24 | 2025-12-24 11:00 | 170.3500 | 165.4108 | 2025-12-24 21:30 | 170.3500 | STOCH_EXIT | 21 | -108.00 | -0.10% | 170.9000 | 180.2285 | 4.9392 | M30 bullish FVG [170.8500, 170.9000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 25 | 2025-12-26 08:00 | 169.8500 | 165.0761 | 2025-12-27 02:30 | 171.1000 | STOCH_EXIT | 33 | 685.50 | 0.64% | 170.3000 | 179.3978 | 4.7739 | M30 bullish FVG [170.1500, 170.3000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 26 | 2025-12-29 10:30 | 170.7000 | 166.0600 | 2025-12-29 12:00 | 172.7000 | STOCH_EXIT | 3 | 1162.80 | 1.07% | 170.7500 | 179.9801 | 4.6400 | M30 bullish FVG [170.7000, 170.7500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 27 | 2025-12-29 17:00 | 170.8000 | 166.1600 | 2025-12-30 23:00 | 166.1600 | SL | 46 | -3087.06 | -2.81% | 171.7000 | 180.0801 | 4.6400 | M30 bullish FVG [171.5000, 171.7000], stochastic crossed below 20 | FVG stop 166.1600 |
| 28 | 2025-12-31 11:00 | 167.7000 | 163.0109 | 2026-01-02 04:30 | 167.9000 | STOCH_EXIT | 30 | 20.48 | 0.02% | 168.4000 | 177.0782 | 4.6891 | M30 bullish FVG [168.2500, 168.4000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 29 | 2026-01-02 14:30 | 167.3500 | 162.6609 | 2026-01-02 19:30 | 167.9000 | STOCH_EXIT | 10 | 243.57 | 0.23% | 167.8500 | 176.7282 | 4.6891 | M30 bullish FVG [167.7500, 167.8500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 30 | 2026-01-07 04:00 | 168.5500 | 164.1285 | 2026-01-07 04:30 | 169.0000 | STOCH_EXIT | 1 | 178.30 | 0.17% | 168.3000 | 177.3929 | 4.4215 | M30 bullish FVG [168.1500, 168.3000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 31 | 2026-01-07 18:30 | 168.4000 | 163.9785 | 2026-01-10 02:00 | 166.0000 | STOCH_EXIT | 80 | -1632.74 | -1.52% | 168.3500 | 177.2429 | 4.4215 | M30 bullish FVG [168.3000, 168.3500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 32 | 2026-01-12 07:00 | 165.4000 | 161.2560 | 2026-01-12 23:00 | 165.7000 | STOCH_EXIT | 32 | 85.64 | 0.08% | 165.4000 | 173.6881 | 4.1440 | M30 bullish FVG [165.3000, 165.4000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 33 | 2026-01-13 10:30 | 164.7500 | 160.7413 | 2026-01-14 07:00 | 163.6500 | STOCH_EXIT | 28 | -809.09 | -0.77% | 165.7000 | 172.7675 | 4.0087 | M30 bullish FVG [165.3000, 165.7000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 34 | 2026-01-14 18:30 | 162.1000 | 158.0383 | 2026-01-15 06:30 | 162.6000 | STOCH_EXIT | 11 | 218.12 | 0.21% | 162.1500 | 170.2234 | 4.0617 | M30 bullish FVG [161.9500, 162.1500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 35 | 2026-01-15 15:30 | 166.0000 | 162.0070 | 2026-01-16 06:30 | 168.2500 | STOCH_EXIT | 17 | 1316.38 | 1.25% | 168.0000 | 173.9860 | 3.9930 | M30 bullish FVG [166.1500, 168.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 36 | 2026-01-16 20:00 | 166.4000 | 162.0458 | 2026-01-18 02:00 | 166.6000 | STOCH_EXIT | 48 | 21.37 | 0.02% | 167.2500 | 175.1084 | 4.3542 | M30 bullish FVG [166.6000, 167.2500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 37 | 2026-01-19 18:30 | 169.8500 | 165.5961 | 2026-01-19 21:00 | 170.9000 | STOCH_EXIT | 5 | 549.77 | 0.52% | 169.0000 | 178.3578 | 4.2539 | M30 bullish FVG [168.7500, 169.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 38 | 2026-01-20 11:00 | 168.7000 | 164.3571 | 2026-01-21 21:30 | 167.8500 | STOCH_EXIT | 56 | -644.57 | -0.60% | 172.4000 | 177.3858 | 4.3429 | M30 bullish FVG [171.0000, 172.4000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 39 | 2026-01-26 11:00 | 165.5000 | 161.3503 | 2026-01-27 14:00 | 165.0500 | STOCH_EXIT | 41 | -394.39 | -0.37% | 166.3000 | 173.7994 | 4.1497 | M30 bullish FVG [166.1500, 166.3000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 40 | 2026-01-28 17:00 | 166.5000 | 162.5214 | 2026-01-29 07:00 | 162.5214 | SL | 15 | -2630.87 | -2.49% | 167.3500 | 174.4572 | 3.9786 | M30 bullish FVG [167.2500, 167.3500], stochastic crossed below 20 | FVG stop 162.5214 |
| 41 | 2026-01-29 16:30 | 164.0000 | 160.0949 | 2026-01-31 05:30 | 162.7000 | STOCH_EXIT | 57 | -920.45 | -0.89% | 165.4500 | 171.8102 | 3.9051 | M30 bullish FVG [165.3500, 165.4500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 42 | 2026-02-02 08:30 | 163.0500 | 158.7288 | 2026-02-02 15:00 | 163.1500 | STOCH_EXIT | 13 | -39.56 | -0.04% | 164.1000 | 171.6925 | 4.3212 | M30 bullish FVG [164.0500, 164.1000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 43 | 2026-02-03 14:00 | 164.7500 | 160.4338 | 2026-02-03 21:30 | 164.4500 | STOCH_EXIT | 15 | -288.05 | -0.28% | 163.6000 | 173.3823 | 4.3162 | M30 bullish FVG [163.1000, 163.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 44 | 2026-02-04 14:30 | 164.1500 | 159.8671 | 2026-02-05 19:00 | 162.9000 | STOCH_EXIT | 44 | -877.80 | -0.86% | 164.5000 | 172.7157 | 4.2829 | M30 bullish FVG [164.2500, 164.5000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 45 | 2026-02-06 14:30 | 162.6500 | 158.4426 | 2026-02-07 03:00 | 163.6500 | STOCH_EXIT | 20 | 519.68 | 0.51% | 163.2500 | 171.0648 | 4.2074 | M30 bullish FVG [162.9000, 163.2500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 46 | 2026-02-09 15:00 | 161.2500 | 157.1717 | 2026-02-09 23:30 | 161.6000 | STOCH_EXIT | 17 | 118.80 | 0.12% | 162.6000 | 169.4066 | 4.0783 | M30 bullish FVG [162.3000, 162.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 47 | 2026-02-11 12:00 | 161.8500 | 158.1299 | 2026-02-11 23:00 | 162.8000 | STOCH_EXIT | 22 | 494.66 | 0.49% | 161.3500 | 169.2902 | 3.7201 | M30 bullish FVG [161.1000, 161.3500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 48 | 2026-02-12 12:30 | 163.1000 | 159.2921 | 2026-02-12 23:30 | 163.3000 | STOCH_EXIT | 22 | 23.04 | 0.02% | 164.3500 | 170.7159 | 3.8079 | M30 bullish FVG [163.5000, 164.3500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 49 | 2026-02-13 10:00 | 161.9500 | 158.0712 | 2026-02-14 02:00 | 162.5500 | STOCH_EXIT | 28 | 276.22 | 0.27% | 162.5500 | 169.7076 | 3.8788 | M30 bullish FVG [162.4000, 162.5500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 50 | 2026-02-18 09:30 | 161.8500 | 158.3372 | 2026-02-19 06:30 | 161.5000 | STOCH_EXIT | 29 | -323.89 | -0.32% | 162.2000 | 168.8755 | 3.5128 | M30 bullish FVG [162.1500, 162.2000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 51 | 2026-02-24 13:30 | 159.7500 | 156.5259 | 2026-02-25 06:30 | 159.9000 | STOCH_EXIT | 21 | -6.28 | -0.01% | 160.9000 | 166.1982 | 3.2241 | M30 bullish FVG [160.1000, 160.9000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 52 | 2026-02-25 11:00 | 159.1000 | 155.8276 | 2026-02-25 16:00 | 159.5000 | STOCH_EXIT | 10 | 154.53 | 0.15% | 159.6500 | 165.6447 | 3.2724 | M30 bullish FVG [159.5500, 159.6500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 53 | 2026-03-03 08:00 | 150.7000 | 147.3716 | 2026-03-03 23:00 | 151.3500 | STOCH_EXIT | 30 | 338.31 | 0.33% | 152.5000 | 157.3568 | 3.3284 | M30 bullish FVG [152.2500, 152.5000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 54 | 2026-03-05 10:30 | 152.5000 | 149.1345 | 2026-03-06 17:00 | 149.1345 | SL | 47 | -2366.51 | -2.30% | 152.9500 | 159.2311 | 3.3655 | M30 bullish FVG [152.5500, 152.9500], stochastic crossed below 20 | FVG stop 149.1345 |
| 55 | 2026-03-10 13:30 | 147.2500 | 143.7053 | 2026-03-11 08:30 | 146.6500 | STOCH_EXIT | 24 | -508.67 | -0.51% | 148.6500 | 154.3395 | 3.5447 | M30 bullish FVG [148.0000, 148.6500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 56 | 2026-03-12 18:00 | 141.3500 | 137.6364 | 2026-03-13 07:00 | 146.7500 | STOCH_EXIT | 13 | 3705.44 | 3.72% | 142.3000 | 148.7771 | 3.7136 | M30 bullish FVG [142.0000, 142.3000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 57 | 2026-03-16 10:30 | 157.7000 | 152.5847 | 2026-03-16 19:00 | 152.5847 | SL | 17 | -3457.39 | -3.34% | 160.3500 | 167.9305 | 5.1153 | M30 bullish FVG [160.2000, 160.3500], stochastic crossed below 20 | FVG stop 152.5847 |
| 58 | 2026-03-17 11:00 | 154.8000 | 149.1751 | 2026-03-17 22:30 | 156.3500 | STOCH_EXIT | 23 | 900.80 | 0.90% | 155.5500 | 166.0498 | 5.6249 | M30 bullish FVG [154.6500, 155.5500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 59 | 2026-03-18 08:00 | 154.9500 | 149.2733 | 2026-03-19 19:30 | 149.2733 | SL | 58 | -3794.55 | -3.76% | 155.4000 | 166.3034 | 5.6767 | M30 bullish FVG [155.2500, 155.4000], stochastic crossed below 20 | FVG stop 149.2733 |
| 60 | 2026-03-20 17:30 | 154.9000 | 149.0244 | 2026-03-21 09:30 | 154.4000 | STOCH_EXIT | 23 | -410.47 | -0.42% | 156.5000 | 166.6511 | 5.8756 | M30 bullish FVG [155.2000, 156.5000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 61 | 2026-03-24 13:30 | 151.5000 | 145.4402 | 2026-03-24 22:00 | 151.9500 | STOCH_EXIT | 17 | 190.30 | 0.20% | 152.6000 | 163.6196 | 6.0598 | M30 bullish FVG [151.9000, 152.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 62 | 2026-03-25 09:00 | 151.1000 | 145.0838 | 2026-03-25 23:30 | 151.3000 | STOCH_EXIT | 29 | 31.28 | 0.03% | 151.0500 | 163.1325 | 6.0162 | M30 bullish FVG [150.8500, 151.0500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 63 | 2026-03-27 11:00 | 147.3000 | 141.5804 | 2026-03-27 16:30 | 141.5804 | SL | 11 | -3858.55 | -3.98% | 150.5500 | 158.7392 | 5.7196 | M30 bullish FVG [150.2000, 150.5500], stochastic crossed below 20 | FVG stop 141.5804 |
| 64 | 2026-03-31 08:00 | 152.2500 | 146.0152 | 2026-04-01 18:30 | 149.3500 | STOCH_EXIT | 56 | -1864.04 | -2.00% | 151.8000 | 164.7195 | 6.2348 | M30 bullish FVG [151.7000, 151.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 65 | 2026-04-02 15:00 | 149.0500 | 142.9504 | 2026-04-04 06:00 | 145.9000 | STOCH_EXIT | 58 | -2018.05 | -2.21% | 150.7000 | 161.2492 | 6.0996 | M30 bullish FVG [150.4500, 150.7000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 66 | 2026-04-06 10:30 | 144.2500 | 137.9917 | 2026-04-07 08:00 | 146.3000 | STOCH_EXIT | 30 | 1177.12 | 1.32% | 146.2500 | 156.7667 | 6.2583 | M30 bullish FVG [146.1500, 146.2500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 67 | 2026-04-08 17:30 | 144.6500 | 138.8576 | 2026-04-09 22:30 | 144.2500 | STOCH_EXIT | 45 | -339.74 | -0.38% | 145.5000 | 156.2348 | 5.7924 | M30 bullish FVG [144.8000, 145.5000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 68 | 2026-04-10 18:30 | 142.9500 | 137.2277 | 2026-04-11 11:00 | 142.7500 | STOCH_EXIT | 28 | -215.65 | -0.24% | 143.5500 | 154.3946 | 5.7223 | M30 bullish FVG [143.3500, 143.5500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 69 | 2026-04-13 12:00 | 142.5000 | 136.9900 | 2026-04-14 06:30 | 142.6500 | STOCH_EXIT | 24 | 4.68 | 0.01% | 143.6500 | 153.5199 | 5.5100 | M30 bullish FVG [141.8000, 143.6500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 70 | 2026-04-15 10:00 | 142.2500 | 137.1054 | 2026-04-15 17:30 | 142.5000 | STOCH_EXIT | 15 | 67.91 | 0.08% | 142.0000 | 152.5391 | 5.1446 | M30 bullish FVG [141.9500, 142.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 71 | 2026-04-16 14:30 | 143.6000 | 138.5872 | 2026-04-16 22:30 | 143.2000 | STOCH_EXIT | 16 | -340.17 | -0.38% | 143.6000 | 153.6256 | 5.0128 | M30 bullish FVG [143.5500, 143.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 72 | 2026-04-20 09:00 | 142.0500 | 137.3397 | 2026-04-20 23:30 | 142.2000 | STOCH_EXIT | 29 | 4.96 | 0.01% | 142.3500 | 151.4706 | 4.7103 | M30 bullish FVG [142.2000, 142.3500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 73 | 2026-04-21 20:30 | 137.7000 | 133.2119 | 2026-04-22 22:30 | 136.3000 | STOCH_EXIT | 39 | -999.05 | -1.12% | 138.1000 | 146.6762 | 4.4881 | M30 bullish FVG [137.8500, 138.1000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 74 | 2026-04-23 14:30 | 139.2000 | 134.4054 | 2026-04-24 07:00 | 141.5000 | STOCH_EXIT | 20 | 1373.54 | 1.55% | 136.0500 | 148.7892 | 4.7946 | M30 bullish FVG [135.7000, 136.0500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 75 | 2026-04-24 10:00 | 138.7500 | 133.5907 | 2026-04-25 03:30 | 137.7500 | STOCH_EXIT | 30 | -737.59 | -0.82% | 139.9000 | 149.0685 | 5.1593 | M30 bullish FVG [139.3500, 139.9000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 76 | 2026-04-28 08:00 | 134.7500 | 129.5588 | 2026-04-28 12:00 | 129.5588 | SL | 8 | -3524.03 | -3.95% | 135.3000 | 145.1323 | 5.1912 | M30 bullish FVG [135.2500, 135.3000], stochastic crossed below 20 | FVG stop 129.5588 |
| 77 | 2026-04-29 09:00 | 128.1000 | 122.7546 | 2026-04-29 21:00 | 127.2500 | STOCH_EXIT | 24 | -653.09 | -0.76% | 130.5500 | 138.7907 | 5.3454 | M30 bullish FVG [130.4500, 130.5500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 78 | 2026-04-30 08:30 | 124.8500 | 119.4472 | 2026-05-01 08:00 | 122.9000 | STOCH_EXIT | 42 | -1412.31 | -1.66% | 126.6500 | 135.6557 | 5.4028 | M30 bullish FVG [126.5500, 126.6500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 79 | 2026-05-01 17:30 | 123.3000 | 117.6652 | 2026-05-02 07:00 | 123.2500 | STOCH_EXIT | 20 | -117.48 | -0.14% | 123.6000 | 134.5696 | 5.6348 | M30 bullish FVG [123.5000, 123.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 80 | 2026-05-05 08:00 | 121.1000 | 115.6715 | 2026-05-05 16:30 | 121.9000 | STOCH_EXIT | 17 | 467.49 | 0.56% | 122.4500 | 131.9569 | 5.4285 | M30 bullish FVG [121.5000, 122.4500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 81 | 2026-05-06 11:00 | 125.1000 | 119.2379 | 2026-05-06 20:30 | 125.6500 | STOCH_EXIT | 19 | 284.92 | 0.34% | 127.4000 | 136.8243 | 5.8621 | M30 bullish FVG [125.7500, 127.4000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 82 | 2026-05-08 16:30 | 123.2000 | 117.4168 | 2026-05-09 06:00 | 125.5500 | STOCH_EXIT | 18 | 1520.10 | 1.81% | 124.8000 | 134.7663 | 5.7832 | M30 bullish FVG [124.7500, 124.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 83 | 2026-05-12 08:30 | 124.1000 | 118.6015 | 2026-05-12 16:00 | 124.8500 | STOCH_EXIT | 15 | 432.24 | 0.50% | 125.0000 | 135.0970 | 5.4985 | M30 bullish FVG [124.8500, 125.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 84 | 2026-05-13 10:30 | 125.4500 | 120.0085 | 2026-05-13 15:00 | 126.7000 | STOCH_EXIT | 9 | 772.14 | 0.90% | 126.2000 | 136.3329 | 5.4415 | M30 bullish FVG [125.9000, 126.2000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 85 | 2026-05-13 19:00 | 125.9000 | 120.4585 | 2026-05-15 07:30 | 125.0000 | STOCH_EXIT | 45 | -707.56 | -0.81% | 126.1000 | 136.7829 | 5.4415 | M30 bullish FVG [126.0500, 126.1000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 86 | 2026-05-20 15:00 | 124.4500 | 119.9146 | 2026-05-21 17:00 | 122.5500 | STOCH_EXIT | 39 | -1402.29 | -1.63% | 126.9000 | 133.5209 | 4.5354 | M30 bullish FVG [124.0000, 126.9000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 87 | 2026-05-22 12:00 | 123.3000 | 118.6026 | 2026-05-23 03:00 | 122.9500 | STOCH_EXIT | 25 | -325.51 | -0.38% | 123.0500 | 132.6948 | 4.6974 | M30 bullish FVG [122.9000, 123.0500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 88 | 2026-05-25 11:00 | 121.7500 | 117.0060 | 2026-05-25 23:30 | 117.0060 | SL | 25 | -3375.19 | -3.99% | 122.1000 | 131.2380 | 4.7440 | M30 bullish FVG [122.0000, 122.1000], stochastic crossed below 20 | FVG stop 117.0060 |
| 89 | 2026-05-26 14:00 | 112.7000 | 107.7948 | 2026-05-27 13:30 | 107.7948 | SL | 34 | -3611.09 | -4.45% | 113.3500 | 122.5103 | 4.9052 | M30 bullish FVG [112.9000, 113.3500], stochastic crossed below 20 | FVG stop 107.7948 |
| 90 | 2026-05-28 11:30 | 110.4000 | 105.1688 | 2026-05-28 23:30 | 105.1688 | SL | 24 | -3747.99 | -4.83% | 109.0000 | 120.8625 | 5.2312 | M30 bullish FVG [108.1000, 109.0000], stochastic crossed below 20 | FVG stop 105.1688 |
| 91 | 2026-06-01 11:00 | 98.0500 | 92.1948 | 2026-06-01 20:30 | 97.8500 | STOCH_EXIT | 19 | -224.06 | -0.30% | 99.1500 | 109.7605 | 5.8552 | M30 bullish FVG [99.0500, 99.1500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 92 | 2026-06-03 18:00 | 101.0000 | 95.3261 | 2026-06-04 09:30 | 103.5000 | STOCH_EXIT | 17 | 1745.56 | 2.37% | 101.3000 | 112.3478 | 5.6739 | M30 bullish FVG [101.2500, 101.3000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 93 | 2026-06-05 10:30 | 101.4000 | 95.9457 | 2026-06-06 05:00 | 99.7000 | STOCH_EXIT | 28 | -1337.81 | -1.77% | 101.3000 | 112.3086 | 5.4543 | M30 bullish FVG [101.1500, 101.3000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 94 | 2026-06-09 08:00 | 72.6000 | 65.6121 | 2026-06-09 12:30 | 78.0500 | STOCH_EXIT | 9 | 5476.79 | 7.40% | 77.8000 | 86.5758 | 6.9879 | M30 bullish FVG [77.6000, 77.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 95 | 2026-06-09 21:00 | 77.0000 | 70.0121 | 2026-06-10 07:00 | 78.6000 | STOCH_EXIT | 7 | 1570.91 | 1.98% | 77.6000 | 90.9758 | 6.9879 | M30 bullish FVG [77.2000, 77.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 96 | 2026-06-11 17:30 | 80.0000 | 72.6775 | 2026-06-12 07:30 | 79.5000 | STOCH_EXIT | 20 | -587.29 | -0.72% | 80.8500 | 94.6449 | 7.3225 | M30 bullish FVG [79.6500, 80.8500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 97 | 2026-06-12 11:00 | 79.1500 | 72.0041 | 2026-06-13 05:00 | 78.5000 | STOCH_EXIT | 29 | -740.49 | -0.92% | 79.5000 | 93.4417 | 7.1459 | M30 bullish FVG [79.3000, 79.5000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 98 | 2026-06-15 14:00 | 77.3000 | 70.4788 | 2026-06-15 20:30 | 78.5000 | STOCH_EXIT | 13 | 1156.89 | 1.45% | 78.4000 | 90.9423 | 6.8212 | M30 bullish FVG [77.6500, 78.4000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 99 | 2026-06-16 12:30 | 77.3500 | 70.5625 | 2026-06-16 23:30 | 77.1500 | STOCH_EXIT | 22 | -289.73 | -0.36% | 78.2500 | 90.9250 | 6.7875 | M30 bullish FVG [78.2000, 78.2500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 100 | 2026-06-23 08:00 | 55.1000 | 48.3212 | 2026-06-23 10:30 | 58.2000 | STOCH_EXIT | 5 | 4449.38 | 5.52% | 58.3500 | 68.6577 | 6.7788 | M30 bullish FVG [58.1000, 58.3500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 101 | 2026-06-24 09:30 | 59.2500 | 52.5054 | 2026-06-24 23:30 | 57.0000 | STOCH_EXIT | 28 | -3312.16 | -3.89% | 60.2000 | 72.7393 | 6.7446 | M30 bullish FVG [60.0500, 60.2000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 102 | 2026-06-26 11:00 | 55.7000 | 49.0602 | 2026-06-26 14:30 | 57.5500 | STOCH_EXIT | 7 | 2630.88 | 3.22% | 56.8000 | 68.9795 | 6.6398 | M30 bullish FVG [56.7500, 56.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 103 | 2026-07-01 11:00 | 57.8500 | 51.3686 | 2026-07-01 19:30 | 58.8000 | STOCH_EXIT | 17 | 1300.06 | 1.54% | 59.2000 | 70.8127 | 6.4814 | M30 bullish FVG [58.9000, 59.2000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 104 | 2026-07-02 09:30 | 58.2500 | 51.9780 | 2026-07-02 10:30 | 60.3000 | STOCH_EXIT | 2 | 2926.37 | 3.42% | 58.3500 | 70.7940 | 6.2720 | M30 bullish FVG [58.1500, 58.3500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 105 | 2026-07-02 19:30 | 56.0000 | 49.7280 | 2026-07-02 21:00 | 56.8000 | STOCH_EXIT | 3 | 1175.63 | 1.33% | 56.0500 | 68.5440 | 6.2720 | M30 bullish FVG [56.0000, 56.0500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 106 | 2026-07-03 09:00 | 55.7500 | 49.5582 | 2026-07-03 15:00 | 56.3000 | STOCH_EXIT | 12 | 794.81 | 0.89% | 56.3000 | 68.1337 | 6.1918 | M30 bullish FVG [56.2000, 56.3000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 107 | 2026-07-06 14:30 | 56.9000 | 51.0683 | 2026-07-06 23:30 | 55.2000 | STOCH_EXIT | 18 | -2793.88 | -3.08% | 56.4500 | 68.5634 | 5.8317 | M30 bullish FVG [56.2500, 56.4500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 108 | 2026-07-07 08:30 | 54.4000 | 48.5848 | 2026-07-07 17:00 | 53.4500 | STOCH_EXIT | 17 | -1619.33 | -1.84% | 55.1000 | 66.0303 | 5.8152 | M30 bullish FVG [54.9500, 55.1000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 109 | 2026-07-08 10:30 | 53.4500 | 47.6252 | 2026-07-08 18:30 | 55.3000 | STOCH_EXIT | 16 | 2892.75 | 3.36% | 55.8500 | 65.0996 | 5.8248 | M30 bullish FVG [55.3500, 55.8500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 110 | 2026-07-09 08:00 | 54.1500 | 48.3841 | 2026-07-11 21:30 | 52.4500 | STOCH_EXIT | 94 | -2882.43 | -3.24% | 55.1000 | 65.6818 | 5.7659 | M30 bullish FVG [54.8000, 55.1000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 111 | 2026-07-13 17:30 | 52.6500 | 47.3080 | 2026-07-14 22:30 | 52.5000 | STOCH_EXIT | 45 | -331.41 | -0.38% | 52.6000 | 63.3340 | 5.3420 | M30 bullish FVG [52.5000, 52.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 112 | 2026-07-15 08:30 | 51.2500 | 46.2809 | 2026-07-16 07:00 | 46.2809 | SL | 32 | -8399.95 | -9.79% | 52.2500 | 61.1883 | 4.9691 | M30 bullish FVG [51.6000, 52.2500], stochastic crossed below 20 | FVG stop 46.2809 |
| 113 | 2026-07-17 18:00 | 40.2000 | 34.9287 | 2026-07-18 03:00 | 40.7000 | STOCH_EXIT | 13 | 884.63 | 1.14% | 42.0000 | 50.7427 | 5.2713 | M30 bullish FVG [41.7000, 42.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 114 | 2026-07-21 11:00 | 46.5000 | 40.8273 | 2026-07-21 14:00 | 52.5500 | STOCH_EXIT | 6 | 10104.80 | 12.90% | 47.7500 | 57.8455 | 5.6727 | M30 bullish FVG [47.6500, 47.7500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 115 | 2026-07-23 10:30 | 53.9000 | 47.1220 | 2026-07-23 17:00 | 54.1000 | STOCH_EXIT | 13 | 239.44 | 0.27% | 57.6500 | 67.4560 | 6.7780 | M30 bullish FVG [56.9500, 57.6500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 116 | 2026-07-27 19:30 | 69.5000 | 61.8906 | 2026-07-28 22:00 | 69.7500 | STOCH_EXIT | 40 | 229.98 | 0.26% | 71.7500 | 84.7187 | 7.6094 | M30 bullish FVG [71.4000, 71.7500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 117 | 2026-07-30 12:30 | 70.3000 | 62.4125 | 2026-07-31 21:30 | 68.0500 | STOCH_EXIT | 53 | -2931.44 | -3.30% | 71.0000 | 86.0749 | 7.8875 | M30 bullish FVG [69.5500, 71.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 118 | 2026-08-06 07:00 | 75.2500 | 68.0046 | 2026-08-06 22:00 | 74.3000 | STOCH_EXIT | 30 | -1170.29 | -1.36% | 77.1500 | 89.7408 | 7.2454 | M30 bullish FVG [76.3000, 77.1500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 119 | 2026-08-07 12:00 | 74.6500 | 67.5650 | 2026-08-08 17:30 | 74.1000 | STOCH_EXIT | 52 | -708.67 | -0.84% | 76.1500 | 88.8201 | 7.0850 | M30 bullish FVG [74.7000, 76.1500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 120 | 2026-08-10 08:00 | 74.4500 | 67.3639 | 2026-08-10 10:30 | 75.7500 | STOCH_EXIT | 5 | 1382.91 | 1.64% | 74.4500 | 88.6222 | 7.0861 | M30 bullish FVG [74.4000, 74.4500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 121 | 2026-08-11 14:00 | 74.2500 | 67.4129 | 2026-08-11 20:00 | 74.6000 | STOCH_EXIT | 12 | 316.91 | 0.37% | 74.8500 | 87.9242 | 6.8371 | M30 bullish FVG [74.7000, 74.8500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 122 | 2026-08-17 07:30 | 67.3500 | 60.8073 | 2026-08-18 06:30 | 67.6500 | STOCH_EXIT | 33 | 295.97 | 0.35% | 68.7000 | 80.4354 | 6.5427 | M30 bullish FVG [68.3500, 68.7000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 123 | 2026-08-19 11:30 | 69.4000 | 63.1234 | 2026-08-19 18:00 | 71.1500 | STOCH_EXIT | 13 | 2082.86 | 2.42% | 72.1500 | 81.9532 | 6.2766 | M30 bullish FVG [70.6000, 72.1500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 124 | 2026-08-20 08:30 | 69.9500 | 63.7074 | 2026-08-20 10:30 | 71.3000 | STOCH_EXIT | 4 | 1612.01 | 1.83% | 70.5000 | 82.4351 | 6.2426 | M30 bullish FVG [70.4000, 70.5000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 125 | 2026-08-20 15:00 | 69.5000 | 63.2574 | 2026-08-20 23:00 | 69.3000 | STOCH_EXIT | 16 | -347.80 | -0.39% | 70.6000 | 81.9851 | 6.2426 | M30 bullish FVG [70.4000, 70.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 126 | 2026-08-24 07:00 | 68.0000 | 62.2334 | 2026-08-24 07:00 | 68.4500 | STOCH_EXIT | 0 | 501.65 | 0.56% | 68.5000 | 79.5331 | 5.7666 | M30 bullish FVG [68.4500, 68.5000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 127 | 2026-08-25 11:30 | 64.9500 | 59.1561 | 2026-08-25 17:00 | 66.0500 | STOCH_EXIT | 11 | 1431.75 | 1.59% | 66.0000 | 76.5379 | 5.7939 | M30 bullish FVG [65.2000, 66.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 128 | 2026-08-26 15:00 | 65.3500 | 59.7699 | 2026-08-27 11:00 | 63.2000 | STOCH_EXIT | 27 | -3093.34 | -3.39% | 66.0000 | 76.5102 | 5.5801 | M30 bullish FVG [65.8500, 66.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 129 | 2026-08-28 18:30 | 65.6500 | 59.9914 | 2026-08-29 12:00 | 65.7500 | STOCH_EXIT | 25 | 46.10 | 0.05% | 66.8000 | 76.9672 | 5.6586 | M30 bullish FVG [66.7500, 66.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 130 | 2026-08-31 07:00 | 66.9000 | 61.2277 | 2026-08-31 08:30 | 68.3000 | STOCH_EXIT | 3 | 1757.44 | 1.99% | 67.6000 | 78.2445 | 5.6723 | M30 bullish FVG [67.2500, 67.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 131 | 2026-08-31 15:00 | 67.5500 | 61.8777 | 2026-08-31 18:00 | 68.9000 | STOCH_EXIT | 6 | 1708.61 | 1.90% | 68.2500 | 78.8945 | 5.6723 | M30 bullish FVG [68.2000, 68.2500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 132 | 2026-09-01 09:00 | 68.3000 | 62.6293 | 2026-09-01 11:00 | 69.4500 | STOCH_EXIT | 4 | 1451.95 | 1.58% | 68.7500 | 79.6414 | 5.6707 | M30 bullish FVG [68.6500, 68.7500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 133 | 2026-09-02 17:00 | 67.3500 | 61.8737 | 2026-09-03 14:30 | 68.4000 | STOCH_EXIT | 29 | 1358.28 | 1.46% | 67.6000 | 78.3027 | 5.4763 | M30 bullish FVG [67.2500, 67.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 134 | 2026-09-04 10:30 | 68.5500 | 63.5237 | 2026-09-05 02:00 | 69.2000 | STOCH_EXIT | 27 | 801.37 | 0.85% | 69.5000 | 78.6026 | 5.0263 | M30 bullish FVG [69.4000, 69.5000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 135 | 2026-09-08 10:30 | 69.4500 | 64.7911 | 2026-09-08 15:30 | 70.6500 | STOCH_EXIT | 10 | 1551.42 | 1.63% | 70.1500 | 78.7678 | 4.6589 | M30 bullish FVG [70.0500, 70.1500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 136 | 2026-09-09 09:00 | 68.7500 | 64.1203 | 2026-09-09 19:00 | 68.4000 | STOCH_EXIT | 20 | -589.77 | -0.61% | 68.7000 | 78.0094 | 4.6297 | M30 bullish FVG [68.6500, 68.7000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 137 | 2026-09-14 14:30 | 67.6500 | 63.5225 | 2026-09-15 09:30 | 67.9000 | STOCH_EXIT | 25 | 259.31 | 0.27% | 67.9000 | 75.9051 | 4.1275 | M30 bullish FVG [67.7000, 67.9000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 138 | 2026-09-15 12:00 | 67.1500 | 63.1923 | 2026-09-18 07:00 | 64.9500 | STOCH_EXIT | 94 | -3258.58 | -3.37% | 67.4500 | 75.0654 | 3.9577 | M30 bullish FVG [67.4000, 67.4500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 139 | 2026-09-18 18:00 | 65.3500 | 61.8591 | 2026-09-19 10:00 | 65.4000 | STOCH_EXIT | 21 | -21.96 | -0.02% | 65.8000 | 72.3317 | 3.4909 | M30 bullish FVG [65.7000, 65.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 140 | 2026-09-21 11:30 | 65.2500 | 61.7835 | 2026-09-21 18:00 | 65.3500 | STOCH_EXIT | 13 | 49.62 | 0.05% | 65.6000 | 72.1830 | 3.4665 | M30 bullish FVG [65.4000, 65.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 141 | 2026-09-21 22:30 | 65.3500 | 61.8835 | 2026-09-22 07:00 | 65.6500 | STOCH_EXIT | 3 | 334.87 | 0.36% | 65.2500 | 72.2830 | 3.4665 | M30 bullish FVG [65.2000, 65.2500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 142 | 2026-09-22 15:30 | 65.6500 | 62.2954 | 2026-09-22 20:30 | 65.8500 | STOCH_EXIT | 10 | 191.57 | 0.20% | 65.9000 | 72.3593 | 3.3546 | M30 bullish FVG [65.8000, 65.9000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 143 | 2026-09-23 11:00 | 65.8500 | 62.6600 | 2026-09-23 18:00 | 65.9500 | STOCH_EXIT | 14 | 48.59 | 0.05% | 65.7500 | 72.2300 | 3.1900 | M30 bullish FVG [65.6500, 65.7500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 144 | 2026-09-24 12:30 | 65.4500 | 62.3807 | 2026-09-26 03:30 | 63.9500 | STOCH_EXIT | 58 | -2245.34 | -2.39% | 66.3000 | 71.5886 | 3.0693 | M30 bullish FVG [66.1500, 66.3000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 145 | 2026-09-29 10:00 | 62.9500 | 60.1959 | 2026-09-29 16:00 | 63.7000 | STOCH_EXIT | 12 | 999.80 | 1.09% | 64.3500 | 68.4582 | 2.7541 | M30 bullish FVG [64.3000, 64.3500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 146 | 2026-09-29 23:30 | 63.4000 | 60.6459 | 2026-10-01 06:30 | 63.3000 | STOCH_EXIT | 36 | -238.65 | -0.26% | 63.4500 | 68.9082 | 2.7541 | M30 bullish FVG [63.0000, 63.4500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 147 | 2026-10-01 14:30 | 63.8500 | 61.2174 | 2026-10-02 18:00 | 63.3500 | STOCH_EXIT | 42 | -815.53 | -0.88% | 64.5000 | 69.1152 | 2.6326 | M30 bullish FVG [63.3000, 64.5000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 148 | 2026-10-06 14:30 | 65.1000 | 62.5946 | 2026-10-08 06:30 | 65.1000 | STOCH_EXIT | 54 | -91.60 | -0.10% | 63.9000 | 70.1108 | 2.5054 | M30 bullish FVG [63.7500, 63.9000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 149 | 2026-10-09 11:30 | 64.0500 | 61.6100 | 2026-10-09 16:30 | 64.5500 | STOCH_EXIT | 10 | 622.62 | 0.68% | 64.7500 | 68.9300 | 2.4400 | M30 bullish FVG [64.4500, 64.7500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 150 | 2026-10-09 18:30 | 63.9500 | 61.5100 | 2026-10-09 22:00 | 64.6000 | STOCH_EXIT | 7 | 844.03 | 0.92% | 64.2000 | 68.8300 | 2.4400 | M30 bullish FVG [64.1500, 64.2000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 151 | 2026-10-12 07:30 | 64.5500 | 62.1736 | 2026-10-12 14:30 | 65.3500 | STOCH_EXIT | 14 | 1058.47 | 1.14% | 64.7500 | 69.3029 | 2.3764 | M30 bullish FVG [64.3500, 64.7500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 152 | 2026-10-13 17:30 | 67.0500 | 64.7326 | 2026-10-15 07:30 | 66.7000 | STOCH_EXIT | 49 | -584.46 | -0.62% | 68.9000 | 71.6848 | 2.3174 | M30 bullish FVG [68.5000, 68.9000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 153 | 2026-10-15 12:00 | 66.0500 | 63.4955 | 2026-10-15 23:00 | 65.6000 | STOCH_EXIT | 22 | -729.89 | -0.78% | 66.4500 | 71.1591 | 2.5545 | M30 bullish FVG [66.4000, 66.4500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 154 | 2026-10-19 20:00 | 66.1000 | 63.6601 | 2026-10-20 07:30 | 66.6000 | STOCH_EXIT | 10 | 608.41 | 0.66% | 66.1000 | 70.9798 | 2.4399 | M30 bullish FVG [66.0000, 66.1000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 155 | 2026-10-20 13:30 | 66.8000 | 64.3987 | 2026-10-20 20:30 | 67.4000 | STOCH_EXIT | 14 | 744.46 | 0.80% | 67.4000 | 71.6026 | 2.4013 | M30 bullish FVG [67.3000, 67.4000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 156 | 2026-10-21 10:00 | 66.6500 | 64.2059 | 2026-10-21 23:30 | 66.0500 | STOCH_EXIT | 27 | -940.22 | -1.00% | 67.3000 | 71.5382 | 2.4441 | M30 bullish FVG [67.2500, 67.3000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 157 | 2026-10-22 11:00 | 66.1500 | 63.6269 | 2026-10-23 15:00 | 65.5500 | STOCH_EXIT | 43 | -937.52 | -1.01% | 66.6500 | 71.1962 | 2.5231 | M30 bullish FVG [66.0000, 66.6500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 158 | 2026-10-27 10:00 | 65.6000 | 63.1163 | 2026-10-27 14:00 | 66.3000 | STOCH_EXIT | 8 | 890.84 | 0.97% | 66.1000 | 70.5674 | 2.4837 | M30 bullish FVG [66.0500, 66.1000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 159 | 2026-10-27 18:30 | 65.5500 | 63.0663 | 2026-10-29 06:30 | 66.5500 | STOCH_EXIT | 45 | 1326.21 | 1.42% | 65.7500 | 70.5174 | 2.4837 | M30 bullish FVG [65.7000, 65.7500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 160 | 2026-10-29 09:30 | 65.4000 | 62.9363 | 2026-10-31 10:30 | 65.7500 | STOCH_EXIT | 75 | 410.71 | 0.43% | 66.3000 | 70.3275 | 2.4637 | M30 bullish FVG [66.2500, 66.3000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 161 | 2026-11-02 12:00 | 65.8000 | 63.4177 | 2026-11-02 18:30 | 66.3000 | STOCH_EXIT | 13 | 625.32 | 0.66% | 66.8000 | 70.5645 | 2.3823 | M30 bullish FVG [65.8000, 66.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 162 | 2026-11-03 17:00 | 64.9000 | 62.3450 | 2026-11-04 05:00 | 66.4000 | STOCH_EXIT | 14 | 2109.93 | 2.21% | 65.3000 | 70.0099 | 2.5550 | M30 bullish FVG [65.2500, 65.3000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 163 | 2026-11-04 20:00 | 64.8000 | 62.3061 | 2026-11-05 07:00 | 65.7000 | STOCH_EXIT | 6 | 1257.13 | 1.29% | 65.5500 | 69.7878 | 2.4939 | M30 bullish FVG [65.3500, 65.5500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 164 | 2026-11-05 20:00 | 66.3000 | 63.9128 | 2026-11-06 08:30 | 71.0744 | TP | 12 | 7011.46 | 7.09% | 66.5500 | 71.0744 | 2.3872 | M30 bullish FVG [66.2500, 66.5500], stochastic crossed below 20 | FVG target 71.0744 |
| 165 | 2026-11-06 15:30 | 67.5500 | 65.0869 | 2026-11-07 05:00 | 68.7000 | STOCH_EXIT | 19 | 1695.30 | 1.60% | 70.0000 | 72.4762 | 2.4631 | M30 bullish FVG [67.2000, 70.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 166 | 2026-11-10 07:30 | 68.0500 | 65.0053 | 2026-11-11 15:00 | 67.7500 | STOCH_EXIT | 49 | -581.28 | -0.54% | 68.6500 | 74.1394 | 3.0447 | M30 bullish FVG [68.6000, 68.6500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 167 | 2026-11-12 09:00 | 69.6500 | 66.5298 | 2026-11-13 06:30 | 69.6000 | STOCH_EXIT | 30 | -183.62 | -0.17% | 69.5000 | 75.8904 | 3.1202 | M30 bullish FVG [69.4000, 69.5000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 168 | 2026-11-13 12:00 | 68.7000 | 65.5955 | 2026-11-13 17:30 | 69.2500 | STOCH_EXIT | 11 | 747.51 | 0.70% | 69.1500 | 74.9089 | 3.1045 | M30 bullish FVG [69.0000, 69.1500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 72544.89
- Макс: 109819.46
- Финал: 107610.15

Полная кривая — в `*_equity.csv`.
