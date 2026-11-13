# Бэктест SPBE (Minutes30)

- Период: 2023-11-12 22:59 — 2025-11-12 22:59
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
| Всего сделок | 27 |
| Выигрышных / проигрышных | 18 / 9 |
| Win rate | 66.67% |
| Gross profit / loss | 43334.63 / 24611.60 |
| Profit factor | 1.761 |
| Чистый PnL | 18723.02 (18.72%) |
| Стартовый / финальный капитал | 100000.00 / 118723.02 |
| Макс. просадка | 13127.46 (11.66%) |
| Средняя прибыль / убыток | 2407.48 / 2734.62 |
| Expectancy | 693.45 |
| Лучшая / худшая сделка | 7208.33 / -8690.60 |
| Exposure | 3.32% |
| CAGR | 8.95% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-11-16 12:00 | 107.8000 | 96.7414 | 2023-11-17 10:00 | 108.3000 | STOCH_EXIT | 24 | 363.34 | 0.36% | 111.1000 | 129.9171 | 11.0586 | M30 bullish FVG [105.7000, 111.1000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 2 | 2024-01-11 12:30 | 97.7000 | 90.5790 | 2024-01-11 19:30 | 101.7000 | STOCH_EXIT | 14 | 4001.71 | 3.99% | 102.7000 | 111.9421 | 7.1210 | M30 bullish FVG [96.7000, 102.7000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 3 | 2024-02-08 11:00 | 93.9000 | 87.3412 | 2024-02-12 20:30 | 92.6000 | STOCH_EXIT | 75 | -1546.51 | -1.48% | 95.5000 | 107.0175 | 6.5588 | M30 bullish FVG [93.8000, 95.5000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 4 | 2024-03-22 10:30 | 88.4000 | 84.1528 | 2024-03-25 18:30 | 88.0000 | STOCH_EXIT | 44 | -567.29 | -0.55% | 88.8000 | 96.8944 | 4.2472 | M30 bullish FVG [86.9000, 88.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 5 | 2024-04-09 18:30 | 92.5000 | 89.1989 | 2024-04-10 14:00 | 92.9000 | STOCH_EXIT | 19 | 339.26 | 0.33% | 93.5000 | 99.1022 | 3.3011 | M30 bullish FVG [92.5000, 93.5000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 6 | 2024-04-12 19:00 | 99.5000 | 95.9493 | 2024-04-15 10:00 | 106.6014 | TP | 10 | 7208.33 | 7.03% | 102.0000 | 106.6014 | 3.5507 | M30 bullish FVG [100.8000, 102.0000], stochastic crossed below 20 | FVG target 106.6014 |
| 7 | 2024-04-16 11:30 | 115.6000 | 110.4670 | 2024-04-17 10:00 | 110.4670 | SL | 25 | -4978.50 | -4.54% | 118.3000 | 125.8660 | 5.1330 | M30 bullish FVG [115.5000, 118.3000], stochastic crossed below 20 | FVG stop 110.4670 |
| 8 | 2024-05-27 18:30 | 115.6000 | 109.5183 | 2024-05-28 10:00 | 119.7000 | STOCH_EXIT | 11 | 3608.01 | 3.44% | 119.3000 | 127.7634 | 6.0817 | M30 bullish FVG [116.3000, 119.3000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 9 | 2024-07-29 15:00 | 135.8000 | 125.0399 | 2024-07-29 18:30 | 125.0399 | SL | 7 | -8690.60 | -8.02% | 134.5000 | 157.3201 | 10.7601 | M30 bullish FVG [130.4000, 134.5000], stochastic crossed below 20 | FVG stop 125.0399 |
| 10 | 2024-07-30 23:00 | 133.0000 | 121.4228 | 2024-07-31 11:30 | 135.3000 | STOCH_EXIT | 5 | 1622.22 | 1.63% | 131.5000 | 156.1544 | 11.5772 | M30 bullish FVG [128.7000, 131.5000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 11 | 2024-09-23 18:30 | 121.0000 | 114.3358 | 2024-09-24 07:00 | 121.8000 | STOCH_EXIT | 11 | 567.99 | 0.56% | 121.2000 | 134.3285 | 6.6642 | M30 bullish FVG [119.5000, 121.2000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 12 | 2024-09-26 18:00 | 129.4000 | 122.6673 | 2024-09-27 08:00 | 130.6000 | STOCH_EXIT | 14 | 842.09 | 0.83% | 127.1000 | 142.8653 | 6.7327 | M30 bullish FVG [125.3000, 127.1000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 13 | 2024-10-02 17:30 | 129.8000 | 122.8546 | 2024-10-03 08:30 | 130.4000 | STOCH_EXIT | 16 | 371.69 | 0.36% | 131.6000 | 143.6908 | 6.9454 | M30 bullish FVG [129.6000, 131.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 14 | 2024-10-04 10:30 | 138.2000 | 130.6705 | 2024-10-07 21:30 | 137.9000 | STOCH_EXIT | 56 | -326.35 | -0.32% | 138.4000 | 153.2589 | 7.5295 | M30 bullish FVG [136.3000, 138.4000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 15 | 2024-10-17 18:00 | 127.6000 | 119.9999 | 2024-10-21 08:00 | 124.9000 | STOCH_EXIT | 48 | -2275.13 | -2.21% | 129.7000 | 142.8003 | 7.6001 | M30 bullish FVG [127.7000, 129.7000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 16 | 2024-11-07 16:00 | 130.7000 | 123.0270 | 2024-11-07 19:00 | 133.2000 | STOCH_EXIT | 6 | 1818.66 | 1.81% | 129.6000 | 146.0461 | 7.6730 | M30 bullish FVG [127.5000, 129.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 17 | 2025-01-29 10:00 | 144.2000 | 137.5993 | 2025-01-29 10:00 | 146.4000 | STOCH_EXIT | 0 | 1456.78 | 1.42% | 141.9000 | 157.4015 | 6.6007 | M30 bullish FVG [140.0000, 141.9000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 18 | 2025-02-18 22:00 | 255.7000 | 241.4173 | 2025-02-19 19:30 | 265.5000 | STOCH_EXIT | 29 | 3863.46 | 3.73% | 269.0000 | 284.2654 | 14.2827 | M30 bullish FVG [264.5000, 269.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 19 | 2025-03-04 15:30 | 301.0000 | 273.9597 | 2025-03-04 18:30 | 314.9000 | STOCH_EXIT | 6 | 4852.36 | 4.51% | 291.6000 | 355.0807 | 27.0403 | M30 bullish FVG [281.8000, 291.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 20 | 2025-04-23 09:00 | 274.1000 | 249.3945 | 2025-04-23 16:30 | 278.3000 | STOCH_EXIT | 15 | 1608.76 | 1.43% | 285.2000 | 323.5110 | 24.7055 | M30 bullish FVG [274.0000, 285.2000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 21 | 2025-05-29 10:00 | 248.1000 | 230.3884 | 2025-05-30 09:00 | 246.2000 | STOCH_EXIT | 32 | -985.54 | -0.87% | 248.4000 | 283.5231 | 17.7116 | M30 bullish FVG [243.5000, 248.4000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 22 | 2025-06-25 12:30 | 217.5000 | 204.4501 | 2025-06-25 22:30 | 218.6000 | STOCH_EXIT | 20 | 457.73 | 0.41% | 209.4000 | 243.5999 | 13.0499 | M30 bullish FVG [205.3000, 209.4000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 23 | 2025-07-03 10:30 | 219.9000 | 208.5149 | 2025-07-03 16:00 | 222.9000 | STOCH_EXIT | 11 | 1433.76 | 1.26% | 219.5000 | 242.6702 | 11.3851 | M30 bullish FVG [216.3000, 219.5000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 24 | 2025-07-15 15:00 | 202.3000 | 189.5522 | 2025-07-17 09:30 | 196.0000 | STOCH_EXIT | 57 | -3691.52 | -3.21% | 203.8000 | 227.7956 | 12.7478 | M30 bullish FVG [198.7000, 203.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 25 | 2025-08-07 19:00 | 245.3000 | 234.5504 | 2025-08-08 14:30 | 259.1000 | STOCH_EXIT | 25 | 6137.15 | 5.52% | 253.7000 | 266.7993 | 10.7496 | M30 bullish FVG [249.5000, 253.7000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 26 | 2025-08-21 18:00 | 261.8000 | 245.7527 | 2025-08-22 10:00 | 258.6000 | STOCH_EXIT | 18 | -1550.17 | -1.32% | 268.1000 | 293.8945 | 16.0473 | M30 bullish FVG [263.2000, 268.1000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 27 | 2025-10-07 19:30 | 219.6000 | 208.0022 | 2025-10-08 08:00 | 225.1000 | STOCH_EXIT | 11 | 2781.32 | 2.40% | 215.9000 | 242.7955 | 11.5978 | M30 bullish FVG [206.3000, 215.9000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 97539.83
- Макс: 118723.02
- Финал: 118723.02

Полная кривая — в `*_equity.csv`.
