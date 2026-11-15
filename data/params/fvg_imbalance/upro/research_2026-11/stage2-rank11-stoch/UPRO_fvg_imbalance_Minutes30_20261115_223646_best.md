# Бэктест UPRO (Minutes30)

- Период: 2023-11-15 22:36 — 2025-11-15 22:36
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.1 |
| MaxWaitBars | 30 |
| StochPeriod | 11 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 0 |
| HTFTrendEMA | 200 |
| RequireEntryAboveFVGLower | 1 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 39 |
| Выигрышных / проигрышных | 30 / 9 |
| Win rate | 76.92% |
| Gross profit / loss | 23590.50 / 3321.05 |
| Profit factor | 7.103 |
| Чистый PnL | 20269.45 (20.27%) |
| Стартовый / финальный капитал | 100000.00 / 120269.45 |
| Макс. просадка | 3035.78 (2.77%) |
| Средняя прибыль / убыток | 786.35 / 369.01 |
| Expectancy | 519.73 |
| Лучшая / худшая сделка | 3577.86 / -1230.43 |
| Exposure | 2.15% |
| CAGR | 9.65% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-11-21 10:30 | 2.2560 | 2.1939 | 2023-11-21 19:30 | 2.2460 | STOCH_EXIT | 18 | -539.04 | -0.54% | 2.2480 | 0.0000 | 0.0621 | M30 bullish FVG [2.2390, 2.2480], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 2 | 2023-12-01 17:00 | 2.0180 | 1.9525 | 2023-12-02 10:30 | 2.0120 | STOCH_EXIT | 15 | -392.73 | -0.40% | 2.0260 | 0.0000 | 0.0655 | M30 bullish FVG [2.0150, 2.0260], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 3 | 2023-12-19 14:00 | 2.0370 | 1.9550 | 2023-12-19 19:00 | 2.0500 | STOCH_EXIT | 10 | 525.91 | 0.54% | 2.0310 | 0.0000 | 0.0820 | M30 bullish FVG [2.0150, 2.0310], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 4 | 2023-12-20 11:00 | 2.0710 | 1.9874 | 2023-12-20 20:00 | 2.0820 | STOCH_EXIT | 18 | 428.33 | 0.43% | 2.0790 | 0.0000 | 0.0836 | M30 bullish FVG [2.0560, 2.0790], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 5 | 2023-12-21 22:30 | 2.0250 | 1.9424 | 2023-12-23 15:30 | 2.0350 | STOCH_EXIT | 43 | 390.53 | 0.39% | 2.0370 | 0.0000 | 0.0826 | M30 bullish FVG [2.0250, 2.0370], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 6 | 2024-01-08 18:00 | 2.0660 | 2.0005 | 2024-01-08 23:30 | 2.0910 | STOCH_EXIT | 11 | 1100.23 | 1.11% | 2.0790 | 0.0000 | 0.0655 | M30 bullish FVG [2.0650, 2.0790], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 7 | 2024-01-24 16:00 | 2.1180 | 2.0679 | 2024-01-25 22:00 | 2.1160 | STOCH_EXIT | 41 | -193.50 | -0.19% | 2.1340 | 0.0000 | 0.0501 | M30 bullish FVG [2.1100, 2.1340], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 8 | 2024-01-30 18:30 | 2.1740 | 2.1267 | 2024-01-30 22:00 | 2.1880 | STOCH_EXIT | 7 | 543.67 | 0.54% | 2.1760 | 0.0000 | 0.0473 | M30 bullish FVG [2.1640, 2.1760], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 9 | 2024-02-01 11:00 | 2.1620 | 2.1153 | 2024-02-02 10:30 | 2.1690 | STOCH_EXIT | 28 | 227.22 | 0.22% | 2.1650 | 0.0000 | 0.0467 | M30 bullish FVG [2.1600, 2.1650], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 10 | 2024-02-14 13:30 | 2.2870 | 2.2219 | 2024-02-14 17:30 | 2.3010 | STOCH_EXIT | 8 | 515.06 | 0.51% | 2.2770 | 0.0000 | 0.0651 | M30 bullish FVG [2.2630, 2.2770], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 11 | 2024-02-22 17:00 | 2.1320 | 2.0583 | 2024-02-22 21:30 | 2.1470 | STOCH_EXIT | 9 | 617.30 | 0.60% | 2.1350 | 0.0000 | 0.0737 | M30 bullish FVG [2.1250, 2.1350], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 12 | 2024-03-01 22:00 | 2.1890 | 2.1203 | 2024-03-01 22:30 | 2.1970 | STOCH_EXIT | 1 | 272.93 | 0.27% | 2.1920 | 0.0000 | 0.0687 | M30 bullish FVG [2.1800, 2.1920], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 13 | 2024-03-13 15:30 | 2.1110 | 2.0567 | 2024-03-13 23:30 | 2.1190 | STOCH_EXIT | 16 | 288.36 | 0.28% | 2.1240 | 0.0000 | 0.0543 | M30 bullish FVG [2.1090, 2.1240], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 14 | 2024-04-02 10:30 | 2.0820 | 2.0351 | 2024-04-02 21:00 | 2.0840 | STOCH_EXIT | 21 | -4.07 | -0.00% | 2.0820 | 0.0000 | 0.0469 | M30 bullish FVG [2.0770, 2.0820], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 15 | 2024-04-10 10:30 | 2.0640 | 2.0199 | 2024-04-10 12:30 | 2.0740 | STOCH_EXIT | 4 | 396.55 | 0.38% | 2.0660 | 0.0000 | 0.0441 | M30 bullish FVG [2.0610, 2.0660], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 16 | 2024-04-10 22:30 | 2.0750 | 2.0309 | 2024-04-11 09:30 | 2.0850 | STOCH_EXIT | 3 | 396.00 | 0.38% | 2.0800 | 0.0000 | 0.0441 | M30 bullish FVG [2.0740, 2.0800], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 17 | 2024-04-17 10:30 | 2.2160 | 2.1669 | 2024-04-17 15:30 | 2.2220 | STOCH_EXIT | 10 | 177.71 | 0.17% | 2.1760 | 0.0000 | 0.0491 | M30 bullish FVG [2.1660, 2.1760], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 18 | 2024-04-22 16:00 | 2.2280 | 2.1788 | 2024-04-22 22:30 | 2.2330 | STOCH_EXIT | 13 | 127.40 | 0.12% | 2.2380 | 0.0000 | 0.0492 | M30 bullish FVG [2.1870, 2.2380], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 19 | 2024-05-07 16:00 | 2.2060 | 2.1585 | 2024-05-07 22:00 | 2.2410 | STOCH_EXIT | 12 | 1540.50 | 1.49% | 2.2110 | 0.0000 | 0.0475 | M30 bullish FVG [2.1930, 2.2110], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 20 | 2024-05-08 11:00 | 2.2160 | 2.1625 | 2024-05-08 20:30 | 2.2170 | STOCH_EXIT | 19 | -57.18 | -0.05% | 2.2170 | 0.0000 | 0.0535 | M30 bullish FVG [2.2090, 2.2170], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 21 | 2024-05-15 14:30 | 2.1730 | 2.1229 | 2024-05-16 10:00 | 2.1630 | STOCH_EXIT | 20 | -584.06 | -0.56% | 2.1790 | 0.0000 | 0.0501 | M30 bullish FVG [2.1730, 2.1790], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 22 | 2024-05-16 12:30 | 2.1520 | 2.1022 | 2024-05-16 20:00 | 2.1550 | STOCH_EXIT | 15 | 41.48 | 0.04% | 2.1630 | 0.0000 | 0.0498 | M30 bullish FVG [2.1510, 2.1630], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 23 | 2024-11-18 16:00 | 1.8460 | 1.7882 | 2024-11-18 17:30 | 1.8640 | STOCH_EXIT | 3 | 920.27 | 0.87% | 1.8600 | 0.0000 | 0.0578 | M30 bullish FVG [1.8460, 1.8600], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 24 | 2024-12-31 13:30 | 1.8490 | 1.7946 | 2024-12-31 20:30 | 1.8480 | STOCH_EXIT | 14 | -162.36 | -0.15% | 1.8590 | 0.0000 | 0.0544 | M30 bullish FVG [1.8480, 1.8590], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 25 | 2025-01-13 07:30 | 1.8810 | 1.8253 | 2025-01-13 07:30 | 1.8900 | STOCH_EXIT | 0 | 398.41 | 0.38% | 1.8710 | 0.0000 | 0.0557 | M30 bullish FVG [1.8500, 1.8710], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 26 | 2025-01-14 20:30 | 1.9360 | 1.8785 | 2025-01-15 08:00 | 1.9550 | STOCH_EXIT | 9 | 938.00 | 0.88% | 1.9480 | 0.0000 | 0.0575 | M30 bullish FVG [1.9360, 1.9480], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 27 | 2025-01-16 16:30 | 2.0010 | 1.9385 | 2025-01-17 07:30 | 2.0040 | STOCH_EXIT | 16 | 52.87 | 0.05% | 1.9940 | 0.0000 | 0.0625 | M30 bullish FVG [1.9830, 1.9940], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 28 | 2025-01-17 21:30 | 2.0340 | 1.9692 | 2025-01-18 07:00 | 2.0570 | STOCH_EXIT | 5 | 1110.59 | 1.03% | 2.0010 | 0.0000 | 0.0648 | M30 bullish FVG [1.9940, 2.0010], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 29 | 2025-01-29 09:00 | 2.0590 | 1.9929 | 2025-01-29 11:00 | 2.0650 | STOCH_EXIT | 4 | 204.78 | 0.19% | 2.0580 | 0.0000 | 0.0661 | M30 bullish FVG [2.0410, 2.0580], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 30 | 2025-02-04 11:00 | 2.1370 | 2.0662 | 2025-02-05 08:00 | 2.1150 | STOCH_EXIT | 29 | -1230.43 | -1.13% | 2.1360 | 0.0000 | 0.0708 | M30 bullish FVG [2.1290, 2.1360], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 31 | 2025-02-06 11:30 | 2.1540 | 2.0799 | 2025-02-06 17:00 | 2.1530 | STOCH_EXIT | 11 | -157.67 | -0.15% | 2.1640 | 0.0000 | 0.0741 | M30 bullish FVG [2.1450, 2.1640], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 32 | 2025-02-10 18:30 | 2.2420 | 2.1707 | 2025-02-10 21:30 | 2.2630 | STOCH_EXIT | 6 | 899.88 | 0.84% | 2.2360 | 0.0000 | 0.0713 | M30 bullish FVG [2.2030, 2.2360], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 33 | 2025-02-13 10:30 | 2.4450 | 2.3598 | 2025-02-13 19:00 | 2.4610 | STOCH_EXIT | 17 | 596.07 | 0.55% | 2.4500 | 0.0000 | 0.0852 | M30 bullish FVG [2.4420, 2.4500], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 34 | 2025-02-14 16:30 | 2.5130 | 2.4271 | 2025-02-14 19:00 | 2.5940 | STOCH_EXIT | 5 | 3373.20 | 3.12% | 2.5850 | 0.0000 | 0.0859 | M30 bullish FVG [2.4770, 2.5850], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 35 | 2025-02-28 21:30 | 2.4510 | 2.3287 | 2025-03-01 04:30 | 2.5330 | STOCH_EXIT | 10 | 3577.86 | 3.24% | 2.4300 | 0.0000 | 0.1223 | M30 bullish FVG [2.4110, 2.4300], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 36 | 2025-03-05 23:30 | 2.5970 | 2.4651 | 2025-03-06 08:00 | 2.6380 | STOCH_EXIT | 4 | 1688.83 | 1.48% | 2.6370 | 0.0000 | 0.1319 | M30 bullish FVG [2.5930, 2.6370], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 37 | 2025-08-15 10:30 | 1.7870 | 1.7168 | 2025-08-15 13:00 | 1.8000 | STOCH_EXIT | 5 | 739.63 | 0.63% | 1.7920 | 0.0000 | 0.0702 | M30 bullish FVG [1.7800, 1.7920], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 38 | 2025-08-18 18:00 | 1.7680 | 1.6987 | 2025-08-18 21:30 | 1.7910 | STOCH_EXIT | 7 | 1421.77 | 1.20% | 1.7520 | 0.0000 | 0.0693 | M30 bullish FVG [1.7440, 1.7520], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 39 | 2025-08-19 13:00 | 1.7990 | 1.7270 | 2025-08-19 22:00 | 1.8020 | STOCH_EXIT | 18 | 79.17 | 0.07% | 1.7710 | 0.0000 | 0.0720 | M30 bullish FVG [1.7600, 1.7710], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 98059.33
- Макс: 120856.92
- Финал: 120269.45

Полная кривая — в `*_equity.csv`.
