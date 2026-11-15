# Бэктест UPRO (Minutes30)

- Период: 2023-11-15 22:37 — 2025-11-15 22:37
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
| TakeProfitATR | 1.5 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 1 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 26 |
| Выигрышных / проигрышных | 20 / 6 |
| Win rate | 76.92% |
| Gross profit / loss | 19065.88 / 2430.22 |
| Profit factor | 7.845 |
| Чистый PnL | 16635.66 (16.64%) |
| Стартовый / финальный капитал | 100000.00 / 116635.66 |
| Макс. просадка | 3709.09 (3.48%) |
| Средняя прибыль / убыток | 953.29 / 405.04 |
| Expectancy | 639.83 |
| Лучшая / худшая сделка | 2397.08 / -884.73 |
| Exposure | 1.31% |
| CAGR | 7.99% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-12-06 11:00 | 1.9470 | 1.8749 | 2023-12-06 15:30 | 1.9400 | STOCH_EXIT | 9 | -456.12 | -0.46% | 1.9390 | 2.0552 | 0.0721 | M30 bullish FVG [1.9060, 1.9390], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 2 | 2024-01-09 17:00 | 2.1140 | 2.0504 | 2024-01-09 21:00 | 2.1290 | STOCH_EXIT | 8 | 605.29 | 0.61% | 2.1070 | 2.2095 | 0.0636 | M30 bullish FVG [2.0840, 2.1070], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 3 | 2024-01-24 17:30 | 2.1170 | 2.0669 | 2024-01-25 22:00 | 2.1160 | STOCH_EXIT | 38 | -146.48 | -0.15% | 2.1340 | 2.1922 | 0.0501 | M30 bullish FVG [2.1100, 2.1340], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 4 | 2024-02-22 16:00 | 2.1370 | 2.0633 | 2024-02-22 21:30 | 2.1470 | STOCH_EXIT | 11 | 361.47 | 0.37% | 2.1200 | 2.2476 | 0.0737 | M30 bullish FVG [2.0910, 2.1200], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 5 | 2024-04-17 10:30 | 2.2160 | 2.1669 | 2024-04-17 15:30 | 2.2220 | STOCH_EXIT | 10 | 170.14 | 0.17% | 2.1550 | 2.2897 | 0.0491 | M30 bullish FVG [2.0900, 2.1550], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 6 | 2024-04-22 16:00 | 2.2280 | 2.1788 | 2024-04-22 22:30 | 2.2330 | STOCH_EXIT | 13 | 124.63 | 0.12% | 2.2380 | 2.3019 | 0.0492 | M30 bullish FVG [2.1870, 2.2380], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 7 | 2024-05-07 15:30 | 2.2200 | 2.1725 | 2024-05-07 22:00 | 2.2410 | STOCH_EXIT | 13 | 844.63 | 0.85% | 2.2110 | 2.2912 | 0.0475 | M30 bullish FVG [2.1930, 2.2110], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 8 | 2024-05-28 16:30 | 1.9970 | 1.9408 | 2024-05-29 11:00 | 1.9900 | STOCH_EXIT | 18 | -449.67 | -0.45% | 1.9880 | 2.0812 | 0.0562 | M30 bullish FVG [1.9610, 1.9880], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 9 | 2024-06-13 22:30 | 1.9740 | 1.9097 | 2024-06-14 16:30 | 2.0230 | STOCH_EXIT | 17 | 2397.08 | 2.38% | 1.9370 | 2.0705 | 0.0643 | M30 bullish FVG [1.9100, 1.9370], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 10 | 2024-06-14 23:00 | 2.0120 | 1.9429 | 2024-06-15 12:00 | 2.0230 | STOCH_EXIT | 6 | 458.11 | 0.45% | 2.0090 | 2.1157 | 0.0691 | M30 bullish FVG [1.9880, 2.0090], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 11 | 2024-06-20 22:30 | 1.9400 | 1.8738 | 2024-06-21 16:00 | 1.9410 | STOCH_EXIT | 16 | -49.85 | -0.05% | 1.9360 | 2.0393 | 0.0662 | M30 bullish FVG [1.9090, 1.9360], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 12 | 2024-08-13 11:30 | 1.8750 | 1.8208 | 2024-08-13 12:30 | 1.9060 | STOCH_EXIT | 2 | 1601.02 | 1.55% | 1.8770 | 1.9563 | 0.0542 | M30 bullish FVG [1.8390, 1.8770], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 13 | 2024-10-04 10:30 | 1.6950 | 1.6470 | 2024-10-04 19:30 | 1.7020 | STOCH_EXIT | 18 | 328.69 | 0.31% | 1.7010 | 1.7670 | 0.0480 | M30 bullish FVG [1.6850, 1.7010], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 14 | 2024-10-10 12:30 | 1.7160 | 1.6716 | 2024-10-10 22:30 | 1.7190 | STOCH_EXIT | 20 | 78.23 | 0.07% | 1.7080 | 1.7826 | 0.0444 | M30 bullish FVG [1.6910, 1.7080], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 15 | 2024-11-21 10:30 | 1.7520 | 1.6882 | 2024-11-21 22:00 | 1.7390 | STOCH_EXIT | 23 | -884.73 | -0.84% | 1.7550 | 1.8477 | 0.0638 | M30 bullish FVG [1.7350, 1.7550], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 16 | 2025-01-13 07:30 | 1.8810 | 1.8253 | 2025-01-13 07:30 | 1.8900 | STOCH_EXIT | 0 | 391.30 | 0.38% | 1.8710 | 1.9646 | 0.0557 | M30 bullish FVG [1.8500, 1.8710], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 17 | 2025-02-10 12:30 | 2.2320 | 2.1607 | 2025-02-10 14:30 | 2.2610 | STOCH_EXIT | 4 | 1257.41 | 1.20% | 2.2360 | 2.3389 | 0.0713 | M30 bullish FVG [2.2030, 2.2360], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 18 | 2025-02-14 14:00 | 2.5450 | 2.4591 | 2025-02-14 19:00 | 2.5940 | STOCH_EXIT | 10 | 1903.65 | 1.82% | 2.5850 | 2.6738 | 0.0859 | M30 bullish FVG [2.4770, 2.5850], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 19 | 2025-02-14 23:30 | 2.5440 | 2.4581 | 2025-02-15 07:00 | 2.5610 | STOCH_EXIT | 10 | 606.79 | 0.57% | 2.5720 | 2.6728 | 0.0859 | M30 bullish FVG [2.5400, 2.5720], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 20 | 2025-03-05 23:30 | 2.5970 | 2.4651 | 2025-03-06 08:00 | 2.6380 | STOCH_EXIT | 4 | 1612.07 | 1.48% | 2.6370 | 2.7949 | 0.1319 | M30 bullish FVG [2.5930, 2.6370], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 21 | 2025-04-25 20:00 | 1.8850 | 1.7993 | 2025-04-25 22:30 | 1.9010 | STOCH_EXIT | 5 | 818.21 | 0.75% | 1.9020 | 2.0136 | 0.0857 | M30 bullish FVG [1.8750, 1.9020], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 22 | 2025-07-15 11:00 | 1.4200 | 1.3790 | 2025-07-15 11:00 | 1.4420 | STOCH_EXIT | 0 | 1604.38 | 1.45% | 1.3950 | 1.4815 | 0.0410 | M30 bullish FVG [1.3700, 1.3950], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 23 | 2025-08-07 19:00 | 1.6570 | 1.6110 | 2025-08-07 23:30 | 1.6880 | STOCH_EXIT | 9 | 1994.27 | 1.77% | 1.5860 | 1.7260 | 0.0460 | M30 bullish FVG [1.5670, 1.5860], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 24 | 2025-10-17 10:00 | 1.4800 | 1.4283 | 2025-10-17 14:00 | 1.4950 | STOCH_EXIT | 8 | 1040.46 | 0.91% | 1.4770 | 1.5576 | 0.0517 | M30 bullish FVG [1.4520, 1.4770], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 25 | 2025-10-31 10:00 | 1.4090 | 1.3534 | 2025-10-31 19:30 | 1.4050 | STOCH_EXIT | 19 | -443.37 | -0.38% | 1.4210 | 1.4924 | 0.0556 | M30 bullish FVG [1.3870, 1.4210], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 26 | 2025-11-07 15:30 | 1.4080 | 1.3572 | 2025-11-08 02:00 | 1.4200 | STOCH_EXIT | 17 | 868.05 | 0.75% | 1.4090 | 1.4842 | 0.0508 | M30 bullish FVG [1.3920, 1.4090], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 98318.35
- Макс: 116635.66
- Финал: 116635.66

Полная кривая — в `*_equity.csv`.
