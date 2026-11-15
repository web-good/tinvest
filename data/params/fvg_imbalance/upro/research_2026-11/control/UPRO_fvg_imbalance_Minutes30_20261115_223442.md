# Бэктест UPRO (Minutes30)

- Период: 2023-11-15 22:34 — 2025-11-15 22:34
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
| Всего сделок | 30 |
| Выигрышных / проигрышных | 23 / 7 |
| Win rate | 76.67% |
| Gross profit / loss | 20925.45 / 6899.67 |
| Profit factor | 3.033 |
| Чистый PnL | 14025.78 (14.03%) |
| Стартовый / финальный капитал | 100000.00 / 114025.78 |
| Макс. просадка | 4783.96 (4.57%) |
| Средняя прибыль / убыток | 909.80 / 985.67 |
| Expectancy | 467.53 |
| Лучшая / худшая сделка | 3137.86 / -3775.21 |
| Exposure | 2.25% |
| CAGR | 6.77% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-12-06 11:00 | 1.9470 | 1.8749 | 2023-12-07 11:30 | 1.8749 | SL | 30 | -3775.21 | -3.80% | 1.9390 | 2.0912 | 0.0721 | M30 bullish FVG [1.9060, 1.9390], stochastic crossed below 20 | FVG stop 1.8749 |
| 2 | 2023-12-20 18:00 | 2.0790 | 1.9954 | 2023-12-20 20:00 | 2.0820 | STOCH_EXIT | 4 | 42.30 | 0.04% | 2.0790 | 2.2463 | 0.0836 | M30 bullish FVG [2.0560, 2.0790], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 3 | 2024-01-24 17:30 | 2.1170 | 2.0669 | 2024-01-25 23:00 | 2.1160 | STOCH_EXIT | 40 | -140.24 | -0.15% | 2.1340 | 2.2173 | 0.0501 | M30 bullish FVG [2.1100, 2.1340], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 4 | 2024-01-30 20:30 | 2.1730 | 2.1257 | 2024-01-30 22:00 | 2.1880 | STOCH_EXIT | 3 | 564.06 | 0.59% | 2.1760 | 2.2677 | 0.0473 | M30 bullish FVG [2.1640, 2.1760], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 5 | 2024-02-12 22:30 | 2.2540 | 2.1896 | 2024-02-13 09:30 | 2.2630 | STOCH_EXIT | 3 | 283.14 | 0.30% | 2.2270 | 2.3829 | 0.0644 | M30 bullish FVG [2.1990, 2.2270], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 6 | 2024-02-22 17:30 | 2.1260 | 2.0523 | 2024-02-22 22:30 | 2.1480 | STOCH_EXIT | 10 | 893.84 | 0.93% | 2.1200 | 2.2734 | 0.0737 | M30 bullish FVG [2.0910, 2.1200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 7 | 2024-03-13 17:00 | 2.1110 | 2.0567 | 2024-03-14 20:30 | 2.0930 | STOCH_EXIT | 36 | -924.69 | -0.95% | 2.1240 | 2.2197 | 0.0543 | M30 bullish FVG [2.1090, 2.1240], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 8 | 2024-04-17 10:30 | 2.2160 | 2.1669 | 2024-04-17 17:30 | 2.2280 | STOCH_EXIT | 14 | 420.45 | 0.44% | 2.1550 | 2.3143 | 0.0491 | M30 bullish FVG [2.0900, 2.1550], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 9 | 2024-04-22 17:30 | 2.2140 | 2.1648 | 2024-04-22 22:30 | 2.2330 | STOCH_EXIT | 10 | 721.39 | 0.76% | 2.2380 | 2.3125 | 0.0492 | M30 bullish FVG [2.1870, 2.2380], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 10 | 2024-05-07 16:30 | 2.2050 | 2.1575 | 2024-05-07 22:00 | 2.2410 | STOCH_EXIT | 11 | 1486.19 | 1.53% | 2.2110 | 2.3000 | 0.0475 | M30 bullish FVG [2.1930, 2.2110], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 11 | 2024-05-28 17:30 | 1.9860 | 1.9298 | 2024-05-29 11:00 | 1.9900 | STOCH_EXIT | 16 | 100.60 | 0.10% | 1.9880 | 2.0983 | 0.0562 | M30 bullish FVG [1.9610, 1.9880], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 12 | 2024-06-13 22:30 | 1.9740 | 1.9097 | 2024-06-14 16:30 | 2.0230 | STOCH_EXIT | 17 | 2350.08 | 2.38% | 1.9370 | 2.1026 | 0.0643 | M30 bullish FVG [1.9100, 1.9370], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 13 | 2024-06-21 12:30 | 1.9350 | 1.8683 | 2024-06-22 22:00 | 1.9380 | STOCH_EXIT | 47 | 55.30 | 0.05% | 1.9360 | 2.0685 | 0.0667 | M30 bullish FVG [1.9090, 1.9360], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 14 | 2024-07-17 18:30 | 1.7610 | 1.7042 | 2024-07-18 14:00 | 1.7650 | STOCH_EXIT | 20 | 127.51 | 0.13% | 1.7810 | 1.8745 | 0.0568 | M30 bullish FVG [1.7660, 1.7810], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 15 | 2024-07-22 17:00 | 1.8410 | 1.7849 | 2024-07-23 10:30 | 1.8560 | STOCH_EXIT | 16 | 723.33 | 0.71% | 1.8520 | 1.9531 | 0.0561 | M30 bullish FVG [1.8370, 1.8520], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 16 | 2024-08-13 11:00 | 1.8860 | 1.8318 | 2024-08-13 12:30 | 1.9060 | STOCH_EXIT | 3 | 977.62 | 0.96% | 1.8770 | 1.9944 | 0.0542 | M30 bullish FVG [1.8390, 1.8770], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 17 | 2024-10-04 15:30 | 1.6980 | 1.6500 | 2024-10-04 19:30 | 1.7020 | STOCH_EXIT | 8 | 140.30 | 0.14% | 1.7010 | 1.7940 | 0.0480 | M30 bullish FVG [1.6850, 1.7010], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 18 | 2024-10-10 13:30 | 1.7120 | 1.6676 | 2024-10-11 21:30 | 1.6960 | STOCH_EXIT | 49 | -1062.24 | -1.03% | 1.7080 | 1.8008 | 0.0444 | M30 bullish FVG [1.6910, 1.7080], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 19 | 2024-11-21 11:00 | 1.7390 | 1.6752 | 2024-11-21 22:00 | 1.7390 | STOCH_EXIT | 22 | -102.60 | -0.10% | 1.7550 | 1.8666 | 0.0638 | M30 bullish FVG [1.7350, 1.7550], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 20 | 2025-01-16 16:30 | 2.0010 | 1.9385 | 2025-01-17 07:30 | 2.0040 | STOCH_EXIT | 16 | 50.87 | 0.05% | 2.0300 | 2.1260 | 0.0625 | M30 bullish FVG [2.0070, 2.0300], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 21 | 2025-01-20 11:30 | 2.0540 | 1.9862 | 2025-01-21 09:30 | 2.0420 | STOCH_EXIT | 30 | -702.40 | -0.68% | 2.0930 | 2.1896 | 0.0678 | M30 bullish FVG [2.0590, 2.0930], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 22 | 2025-02-06 11:30 | 2.1540 | 2.0799 | 2025-02-06 17:30 | 2.1610 | STOCH_EXIT | 12 | 227.60 | 0.22% | 2.1640 | 2.3021 | 0.0741 | M30 bullish FVG [2.1450, 2.1640], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 23 | 2025-02-14 16:30 | 2.5130 | 2.4271 | 2025-02-14 19:00 | 2.5940 | STOCH_EXIT | 5 | 3137.86 | 3.12% | 2.5850 | 2.6847 | 0.0859 | M30 bullish FVG [2.4770, 2.5850], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 24 | 2025-03-06 07:00 | 2.6000 | 2.4690 | 2025-03-06 08:30 | 2.6430 | STOCH_EXIT | 3 | 1615.14 | 1.55% | 2.6370 | 2.8620 | 0.1310 | M30 bullish FVG [2.5930, 2.6370], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 25 | 2025-04-10 11:00 | 1.6570 | 1.5446 | 2025-04-11 08:00 | 1.6760 | STOCH_EXIT | 28 | 1109.34 | 1.05% | 1.6570 | 1.8818 | 0.1124 | M30 bullish FVG [1.6300, 1.6570], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 26 | 2025-07-09 18:00 | 1.4480 | 1.4118 | 2025-07-10 09:00 | 1.4630 | STOCH_EXIT | 17 | 1002.29 | 0.93% | 1.4640 | 1.5205 | 0.0362 | M30 bullish FVG [1.4540, 1.4640], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 27 | 2025-07-29 21:00 | 1.4800 | 1.4405 | 2025-07-30 08:30 | 1.4970 | STOCH_EXIT | 10 | 1132.34 | 1.05% | 1.4710 | 1.5590 | 0.0395 | M30 bullish FVG [1.4600, 1.4710], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 28 | 2025-08-07 21:00 | 1.6350 | 1.5890 | 2025-08-08 14:30 | 1.6790 | STOCH_EXIT | 22 | 2836.98 | 2.59% | 1.5860 | 1.7270 | 0.0460 | M30 bullish FVG [1.5670, 1.5860], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 29 | 2025-10-31 14:00 | 1.4040 | 1.3484 | 2025-11-01 08:00 | 1.4030 | STOCH_EXIT | 22 | -192.28 | -0.17% | 1.4210 | 1.5152 | 0.0556 | M30 bullish FVG [1.3870, 1.4210], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 30 | 2025-11-07 21:30 | 1.4070 | 1.3562 | 2025-11-08 02:00 | 1.4200 | STOCH_EXIT | 5 | 926.92 | 0.82% | 1.4090 | 1.5086 | 0.0508 | M30 bullish FVG [1.3920, 1.4090], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 95605.54
- Макс: 114025.78
- Финал: 114025.78

Полная кривая — в `*_equity.csv`.
