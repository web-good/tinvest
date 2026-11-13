# Бэктест SPBE (Minutes30)

- Период: 2023-11-12 23:00 — 2025-11-12 23:00
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.3 |
| MaxWaitBars | 30 |
| StochPeriod | 10 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 2 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 26 |
| Выигрышных / проигрышных | 18 / 8 |
| Win rate | 69.23% |
| Gross profit / loss | 54076.89 / 22529.14 |
| Profit factor | 2.400 |
| Чистый PnL | 31547.76 (31.55%) |
| Стартовый / финальный капитал | 100000.00 / 131547.76 |
| Макс. просадка | 15468.99 (14.08%) |
| Средняя прибыль / убыток | 3004.27 / 2816.14 |
| Expectancy | 1213.38 |
| Лучшая / худшая сделка | 8576.55 / -8222.31 |
| Exposure | 2.05% |
| CAGR | 14.67% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-11-16 11:30 | 109.9000 | 98.8414 | 2023-11-17 10:00 | 108.3000 | STOCH_EXIT | 25 | -1553.57 | -1.55% | 111.1000 | 132.0171 | 11.0586 | M30 bullish FVG [105.7000, 111.1000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 2 | 2024-01-11 14:00 | 96.7000 | 89.5790 | 2024-01-11 17:30 | 101.5000 | STOCH_EXIT | 7 | 4780.82 | 4.86% | 102.7000 | 110.9421 | 7.1210 | M30 bullish FVG [96.7000, 102.7000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 3 | 2024-01-16 11:00 | 98.5000 | 90.9169 | 2024-01-16 17:30 | 100.9000 | STOCH_EXIT | 13 | 2408.41 | 2.33% | 100.0000 | 113.6662 | 7.5831 | M30 bullish FVG [97.0000, 100.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 4 | 2024-02-27 17:30 | 88.3000 | 83.2753 | 2024-02-27 20:30 | 90.3000 | STOCH_EXIT | 6 | 2283.29 | 2.16% | 87.1000 | 98.3494 | 5.0247 | M30 bullish FVG [84.0000, 87.1000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 5 | 2024-03-21 16:00 | 88.9000 | 84.9338 | 2024-03-25 18:30 | 88.0000 | STOCH_EXIT | 61 | -1198.99 | -1.11% | 88.8000 | 96.8325 | 3.9662 | M30 bullish FVG [86.9000, 88.8000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 6 | 2024-04-09 16:00 | 93.3000 | 89.9989 | 2024-04-10 14:00 | 92.9000 | STOCH_EXIT | 24 | -563.61 | -0.53% | 93.5000 | 99.9022 | 3.3011 | M30 bullish FVG [92.5000, 93.5000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 7 | 2024-04-12 18:30 | 99.7000 | 96.1493 | 2024-04-12 23:30 | 100.7000 | STOCH_EXIT | 10 | 957.39 | 0.90% | 102.0000 | 106.8014 | 3.5507 | M30 bullish FVG [100.8000, 102.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 8 | 2024-04-16 11:30 | 115.6000 | 110.4670 | 2024-04-17 10:00 | 110.4670 | SL | 25 | -4857.84 | -4.54% | 118.3000 | 125.8660 | 5.1330 | M30 bullish FVG [115.5000, 118.3000], stochastic crossed below 20 | FVG stop 110.4670 |
| 9 | 2024-05-27 17:30 | 116.6000 | 110.5183 | 2024-05-27 22:30 | 117.1000 | STOCH_EXIT | 10 | 335.64 | 0.33% | 119.3000 | 128.7634 | 6.0817 | M30 bullish FVG [116.3000, 119.3000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 10 | 2024-07-29 15:00 | 135.8000 | 125.0399 | 2024-07-29 18:30 | 125.0399 | SL | 7 | -8222.31 | -8.02% | 134.5000 | 157.3201 | 10.7601 | M30 bullish FVG [130.4000, 134.5000], stochastic crossed below 20 | FVG stop 125.0399 |
| 11 | 2024-09-26 08:00 | 125.2000 | 118.4673 | 2024-09-26 08:00 | 127.4000 | STOCH_EXIT | 0 | 1561.50 | 1.66% | 125.0000 | 138.6653 | 6.7327 | M30 bullish FVG [123.0000, 125.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 12 | 2024-10-04 10:30 | 138.2000 | 130.6705 | 2024-10-07 07:30 | 138.9000 | STOCH_EXIT | 28 | 389.08 | 0.41% | 138.4000 | 153.2589 | 7.5295 | M30 bullish FVG [136.3000, 138.4000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 13 | 2024-10-10 20:00 | 141.2000 | 133.2041 | 2024-10-10 23:30 | 141.3000 | STOCH_EXIT | 7 | -28.09 | -0.03% | 141.4000 | 157.1918 | 7.9959 | M30 bullish FVG [138.6000, 141.4000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 14 | 2024-11-07 16:00 | 130.7000 | 123.0270 | 2024-11-07 18:00 | 132.5000 | STOCH_EXIT | 4 | 1227.94 | 1.28% | 127.1000 | 146.0461 | 7.6730 | M30 bullish FVG [124.6000, 127.1000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 15 | 2024-12-20 21:00 | 117.8000 | 112.4960 | 2024-12-23 07:00 | 122.1000 | STOCH_EXIT | 6 | 3456.90 | 3.55% | 115.5000 | 128.4081 | 5.3040 | M30 bullish FVG [112.5000, 115.5000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 16 | 2025-02-13 09:30 | 168.6000 | 160.7328 | 2025-02-13 16:00 | 179.4000 | STOCH_EXIT | 13 | 6354.35 | 6.30% | 162.2000 | 184.3344 | 7.8672 | M30 bullish FVG [156.6000, 162.2000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 17 | 2025-02-14 17:00 | 206.0000 | 197.2662 | 2025-02-14 19:00 | 217.8000 | STOCH_EXIT | 4 | 6025.81 | 5.62% | 206.4000 | 223.4676 | 8.7338 | M30 bullish FVG [192.7000, 206.4000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 18 | 2025-02-18 10:00 | 269.3000 | 255.0173 | 2025-02-18 10:00 | 290.0000 | STOCH_EXIT | 0 | 8576.55 | 7.58% | 244.0000 | 297.8654 | 14.2827 | M30 bullish FVG [239.0000, 244.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 19 | 2025-02-18 22:00 | 255.7000 | 241.4173 | 2025-02-19 19:30 | 265.5000 | STOCH_EXIT | 29 | 4540.75 | 3.73% | 269.0000 | 284.2654 | 14.2827 | M30 bullish FVG [264.5000, 269.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 20 | 2025-03-04 15:30 | 301.0000 | 273.9597 | 2025-03-04 18:00 | 307.6000 | STOCH_EXIT | 5 | 2637.90 | 2.09% | 291.6000 | 355.0807 | 27.0403 | M30 bullish FVG [281.8000, 291.6000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 21 | 2025-04-23 09:00 | 274.1000 | 249.3945 | 2025-04-23 13:30 | 278.3000 | STOCH_EXIT | 9 | 1844.19 | 1.43% | 285.2000 | 323.5110 | 24.7055 | M30 bullish FVG [274.0000, 285.2000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 22 | 2025-06-25 12:30 | 217.5000 | 204.4501 | 2025-06-25 19:30 | 218.6000 | STOCH_EXIT | 14 | 530.05 | 0.41% | 209.4000 | 243.5999 | 13.0499 | M30 bullish FVG [205.3000, 209.4000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 23 | 2025-07-15 15:00 | 202.3000 | 189.5522 | 2025-07-16 13:30 | 197.1000 | STOCH_EXIT | 31 | -3504.41 | -2.67% | 203.8000 | 227.7956 | 12.7478 | M30 bullish FVG [198.7000, 203.8000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 24 | 2025-08-07 19:00 | 245.3000 | 234.5504 | 2025-08-07 23:00 | 253.6000 | STOCH_EXIT | 8 | 4194.34 | 3.28% | 253.7000 | 266.7993 | 10.7496 | M30 bullish FVG [249.5000, 253.7000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 25 | 2025-08-21 18:00 | 261.8000 | 245.7527 | 2025-08-22 09:00 | 256.9000 | STOCH_EXIT | 16 | -2600.31 | -1.97% | 268.1000 | 293.8945 | 16.0473 | M30 bullish FVG [263.2000, 268.1000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 26 | 2025-10-07 17:30 | 221.5000 | 209.9022 | 2025-10-08 08:00 | 225.1000 | STOCH_EXIT | 15 | 1971.99 | 1.52% | 215.9000 | 244.6955 | 11.5978 | M30 bullish FVG [206.3000, 215.9000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 94369.22
- Макс: 132394.00
- Финал: 131547.76

Полная кривая — в `*_equity.csv`.
