# Бэктест SPBE (Minutes30)

- Период: 2023-11-12 23:03 — 2025-11-12 23:03
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.1 |
| MaxWaitBars | 30 |
| StochPeriod | 21 |
| UseStochExit | 1 |
| StopATR | 1.7 |
| TakeProfitATR | 0 |
| HTFTrendEMA | 100 |
| RequireEntryAboveFVGLower | 1 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 24 |
| Выигрышных / проигрышных | 16 / 8 |
| Win rate | 66.67% |
| Gross profit / loss | 36337.24 / 27553.36 |
| Profit factor | 1.319 |
| Чистый PnL | 8783.88 (8.78%) |
| Стартовый / финальный капитал | 100000.00 / 108783.88 |
| Макс. просадка | 12143.51 (10.39%) |
| Средняя прибыль / убыток | 2271.08 / 3444.17 |
| Expectancy | 365.99 |
| Лучшая / худшая сделка | 5009.91 / -8094.33 |
| Exposure | 3.87% |
| CAGR | 4.29% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-04-16 15:30 | 115.5000 | 106.7739 | 2024-04-17 21:30 | 110.4000 | STOCH_EXIT | 40 | -4509.20 | -4.51% | 118.3000 | 0.0000 | 5.1330 | M30 bullish FVG [115.5000, 118.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 2 | 2024-05-15 12:30 | 110.0000 | 101.1667 | 2024-05-16 10:30 | 112.6000 | STOCH_EXIT | 24 | 2157.70 | 2.26% | 110.9000 | 0.0000 | 5.1960 | M30 bullish FVG [110.0000, 110.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 3 | 2024-06-18 18:00 | 124.0000 | 109.8968 | 2024-06-19 11:30 | 129.0000 | STOCH_EXIT | 15 | 3835.44 | 3.93% | 125.6000 | 0.0000 | 8.2960 | M30 bullish FVG [123.6000, 125.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 4 | 2024-06-26 14:00 | 143.5000 | 126.0915 | 2024-06-27 15:30 | 141.4000 | STOCH_EXIT | 31 | -1583.17 | -1.56% | 144.2000 | 0.0000 | 10.2403 | M30 bullish FVG [143.3000, 144.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 5 | 2024-07-29 17:00 | 131.2000 | 112.9079 | 2024-07-30 15:30 | 137.5000 | STOCH_EXIT | 25 | 4692.06 | 4.70% | 134.5000 | 0.0000 | 10.7601 | M30 bullish FVG [130.4000, 134.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 6 | 2024-08-01 15:30 | 143.0000 | 121.4374 | 2024-08-02 11:30 | 150.0000 | STOCH_EXIT | 20 | 5009.91 | 4.79% | 136.6000 | 0.0000 | 12.6839 | M30 bullish FVG [134.1000, 136.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 7 | 2024-08-19 20:30 | 125.6000 | 109.2924 | 2024-08-20 09:30 | 127.1000 | STOCH_EXIT | 12 | 1197.82 | 1.09% | 125.9000 | 0.0000 | 9.5927 | M30 bullish FVG [124.8000, 125.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 8 | 2024-09-23 19:00 | 119.8000 | 108.4708 | 2024-09-24 07:00 | 121.8000 | STOCH_EXIT | 10 | 1736.38 | 1.57% | 121.2000 | 0.0000 | 6.6642 | M30 bullish FVG [119.5000, 121.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 9 | 2024-09-26 21:30 | 129.6000 | 118.1545 | 2024-09-27 10:00 | 132.1000 | STOCH_EXIT | 11 | 2054.05 | 1.83% | 127.1000 | 0.0000 | 6.7327 | M30 bullish FVG [125.3000, 127.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 10 | 2024-10-04 13:00 | 136.9000 | 124.0999 | 2024-10-07 21:30 | 137.9000 | STOCH_EXIT | 51 | 721.13 | 0.63% | 138.4000 | 0.0000 | 7.5295 | M30 bullish FVG [136.3000, 138.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 11 | 2024-10-11 10:30 | 140.0000 | 126.4793 | 2024-10-15 15:00 | 130.3000 | STOCH_EXIT | 77 | -8094.33 | -7.02% | 141.4000 | 0.0000 | 7.9533 | M30 bullish FVG [138.6000, 141.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 12 | 2024-11-07 16:00 | 130.7000 | 117.6558 | 2024-11-07 23:00 | 133.5000 | STOCH_EXIT | 14 | 2185.01 | 2.04% | 129.6000 | 0.0000 | 7.6730 | M30 bullish FVG [127.5000, 129.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 13 | 2024-11-13 22:00 | 127.3000 | 114.9531 | 2024-11-15 07:00 | 129.5000 | STOCH_EXIT | 38 | 1777.43 | 1.63% | 128.5000 | 0.0000 | 7.2629 | M30 bullish FVG [127.2000, 128.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 14 | 2025-01-13 21:30 | 129.6000 | 120.0240 | 2025-01-14 11:30 | 132.5000 | STOCH_EXIT | 14 | 2372.99 | 2.14% | 129.5000 | 0.0000 | 5.6329 | M30 bullish FVG [128.2000, 129.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 15 | 2025-02-06 14:30 | 151.0000 | 138.1587 | 2025-02-07 09:30 | 152.3000 | STOCH_EXIT | 24 | 862.41 | 0.76% | 149.5000 | 0.0000 | 7.5537 | M30 bullish FVG [148.6000, 149.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 16 | 2025-03-20 10:30 | 313.2000 | 273.7361 | 2025-03-23 10:30 | 310.6000 | STOCH_EXIT | 90 | -1062.84 | -0.93% | 316.0000 | 0.0000 | 23.2140 | M30 bullish FVG [313.0000, 316.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 17 | 2025-04-01 21:00 | 253.6000 | 211.8274 | 2025-04-02 15:30 | 247.5000 | STOCH_EXIT | 23 | -2832.35 | -2.50% | 253.0000 | 0.0000 | 24.5721 | M30 bullish FVG [250.0000, 253.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 18 | 2025-04-03 16:30 | 252.8000 | 209.4943 | 2025-04-03 23:30 | 260.5000 | STOCH_EXIT | 14 | 3245.30 | 2.94% | 251.3000 | 0.0000 | 25.4739 | M30 bullish FVG [248.7000, 251.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 19 | 2025-05-05 09:30 | 244.7000 | 207.2660 | 2025-05-06 10:00 | 239.8000 | STOCH_EXIT | 35 | -2386.00 | -2.10% | 243.6000 | 0.0000 | 22.0200 | M30 bullish FVG [241.1000, 243.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 20 | 2025-05-13 11:30 | 267.7000 | 233.2517 | 2025-05-13 18:30 | 272.4000 | STOCH_EXIT | 14 | 1838.43 | 1.65% | 267.1000 | 0.0000 | 20.2637 | M30 bullish FVG [265.1000, 267.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 21 | 2025-08-15 13:00 | 290.1000 | 264.0621 | 2025-08-15 18:30 | 292.4000 | STOCH_EXIT | 11 | 783.41 | 0.69% | 284.7000 | 0.0000 | 15.3164 | M30 bullish FVG [282.6000, 284.7000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 22 | 2025-08-19 15:30 | 284.7000 | 256.7942 | 2025-08-21 14:00 | 271.4000 | STOCH_EXIT | 65 | -5431.22 | -4.77% | 282.0000 | 0.0000 | 16.4152 | M30 bullish FVG [280.0000, 282.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 23 | 2025-08-28 08:30 | 266.4000 | 239.2932 | 2025-08-29 15:30 | 262.6000 | STOCH_EXIT | 48 | -1654.25 | -1.52% | 257.0000 | 0.0000 | 15.9452 | M30 bullish FVG [255.0000, 257.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 24 | 2025-09-18 13:00 | 243.2000 | 223.0364 | 2025-09-18 15:30 | 247.7000 | STOCH_EXIT | 5 | 1867.75 | 1.75% | 241.6000 | 0.0000 | 11.8610 | M30 bullish FVG [240.4000, 241.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 93116.55
- Макс: 116874.58
- Финал: 108783.88

Полная кривая — в `*_equity.csv`.
