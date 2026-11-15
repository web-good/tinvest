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
| TakeProfitATR | 2 |
| HTFTrendEMA | 50 |
| RequireEntryAboveFVGLower | 1 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 42 |
| Выигрышных / проигрышных | 33 / 9 |
| Win rate | 78.57% |
| Gross profit / loss | 29048.84 / 4461.57 |
| Profit factor | 6.511 |
| Чистый PnL | 24587.26 (24.59%) |
| Стартовый / финальный капитал | 100000.00 / 124587.26 |
| Макс. просадка | 3441.56 (3.19%) |
| Средняя прибыль / убыток | 880.27 / 495.73 |
| Expectancy | 585.41 |
| Лучшая / худшая сделка | 3696.37 / -1230.43 |
| Exposure | 2.13% |
| CAGR | 11.60% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-01-24 16:00 | 2.1180 | 2.0679 | 2024-01-25 22:00 | 2.1160 | STOCH_EXIT | 41 | -193.50 | -0.19% | 2.1340 | 2.2183 | 0.0501 | M30 bullish FVG [2.1100, 2.1340], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 2 | 2024-01-30 18:30 | 2.1740 | 2.1267 | 2024-01-30 22:00 | 2.1880 | STOCH_EXIT | 7 | 531.86 | 0.54% | 2.1760 | 2.2687 | 0.0473 | M30 bullish FVG [2.1640, 2.1760], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 3 | 2024-02-01 11:00 | 2.1620 | 2.1153 | 2024-02-02 10:30 | 2.1690 | STOCH_EXIT | 28 | 222.39 | 0.22% | 2.1650 | 2.2555 | 0.0467 | M30 bullish FVG [2.1600, 2.1650], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 4 | 2024-02-14 13:30 | 2.2870 | 2.2219 | 2024-02-14 17:30 | 2.3010 | STOCH_EXIT | 8 | 503.36 | 0.51% | 2.2770 | 2.4171 | 0.0651 | M30 bullish FVG [2.2630, 2.2770], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 5 | 2024-03-01 22:00 | 2.1890 | 2.1203 | 2024-03-01 22:30 | 2.1970 | STOCH_EXIT | 1 | 267.12 | 0.27% | 2.1920 | 2.3264 | 0.0687 | M30 bullish FVG [2.1800, 2.1920], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 6 | 2024-04-17 10:30 | 2.2160 | 2.1669 | 2024-04-17 15:30 | 2.2220 | STOCH_EXIT | 10 | 170.14 | 0.17% | 2.1760 | 2.3143 | 0.0491 | M30 bullish FVG [2.1660, 2.1760], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 7 | 2024-04-22 16:00 | 2.2280 | 2.1788 | 2024-04-22 22:30 | 2.2330 | STOCH_EXIT | 13 | 124.63 | 0.12% | 2.2380 | 2.3265 | 0.0492 | M30 bullish FVG [2.1870, 2.2380], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 8 | 2024-05-07 16:00 | 2.2060 | 2.1585 | 2024-05-07 22:00 | 2.2410 | STOCH_EXIT | 12 | 1507.72 | 1.49% | 2.2110 | 2.3010 | 0.0475 | M30 bullish FVG [2.1930, 2.2110], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 9 | 2024-05-08 11:00 | 2.2160 | 2.1625 | 2024-05-08 20:30 | 2.2170 | STOCH_EXIT | 19 | -55.96 | -0.05% | 2.2170 | 2.3229 | 0.0535 | M30 bullish FVG [2.2090, 2.2170], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 10 | 2024-05-15 14:30 | 2.1730 | 2.1229 | 2024-05-16 10:00 | 2.1630 | STOCH_EXIT | 20 | -571.90 | -0.56% | 2.1790 | 2.2731 | 0.0501 | M30 bullish FVG [2.1730, 2.1790], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 11 | 2024-05-16 12:30 | 2.1520 | 2.1022 | 2024-05-16 20:00 | 2.1550 | STOCH_EXIT | 15 | 39.79 | 0.04% | 2.1630 | 2.2517 | 0.0498 | M30 bullish FVG [2.1510, 2.1630], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 12 | 2024-08-13 11:30 | 1.8750 | 1.8208 | 2024-08-13 12:30 | 1.9060 | STOCH_EXIT | 2 | 1571.91 | 1.55% | 1.8770 | 1.9834 | 0.0542 | M30 bullish FVG [1.8390, 1.8770], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 13 | 2024-08-13 18:00 | 1.8950 | 1.8408 | 2024-08-13 23:30 | 1.8970 | STOCH_EXIT | 11 | 5.62 | 0.01% | 1.8980 | 2.0034 | 0.0542 | M30 bullish FVG [1.8910, 1.8980], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 14 | 2024-11-07 09:30 | 1.6800 | 1.6322 | 2024-11-07 16:30 | 1.7020 | STOCH_EXIT | 10 | 1238.85 | 1.21% | 1.6890 | 1.7755 | 0.0478 | M30 bullish FVG [1.6790, 1.6890], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 15 | 2024-11-08 16:30 | 1.7660 | 1.7139 | 2024-11-08 19:00 | 1.7840 | STOCH_EXIT | 5 | 957.28 | 0.92% | 1.7350 | 1.8703 | 0.0521 | M30 bullish FVG [1.7270, 1.7350], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 16 | 2024-11-13 19:30 | 1.8400 | 1.7806 | 2024-11-14 10:30 | 1.8500 | STOCH_EXIT | 16 | 464.84 | 0.44% | 1.8310 | 1.9589 | 0.0594 | M30 bullish FVG [1.8250, 1.8310], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 17 | 2024-11-18 16:00 | 1.8460 | 1.7882 | 2024-11-18 17:30 | 1.8640 | STOCH_EXIT | 3 | 920.27 | 0.87% | 1.8600 | 1.9616 | 0.0578 | M30 bullish FVG [1.8460, 1.8600], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 18 | 2024-11-21 10:30 | 1.7520 | 1.6882 | 2024-11-21 22:00 | 1.7390 | STOCH_EXIT | 23 | -899.48 | -0.84% | 1.7550 | 1.8796 | 0.0638 | M30 bullish FVG [1.7350, 1.7550], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 19 | 2024-12-23 10:00 | 1.7760 | 1.7211 | 2024-12-24 11:30 | 1.7660 | STOCH_EXIT | 37 | -706.26 | -0.66% | 1.7770 | 1.8859 | 0.0549 | M30 bullish FVG [1.7700, 1.7770], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 20 | 2024-12-25 18:30 | 1.7460 | 1.6936 | 2024-12-26 08:30 | 1.7570 | STOCH_EXIT | 14 | 554.91 | 0.53% | 1.7430 | 1.8509 | 0.0524 | M30 bullish FVG [1.7340, 1.7430], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 21 | 2024-12-31 13:30 | 1.8490 | 1.7946 | 2024-12-31 20:30 | 1.8480 | STOCH_EXIT | 14 | -162.36 | -0.15% | 1.8590 | 1.9578 | 0.0544 | M30 bullish FVG [1.8480, 1.8590], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 22 | 2025-01-10 22:00 | 1.8470 | 1.7933 | 2025-01-10 23:30 | 1.8550 | STOCH_EXIT | 3 | 350.49 | 0.33% | 1.8100 | 1.9544 | 0.0537 | M30 bullish FVG [1.8030, 1.8100], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 23 | 2025-01-13 07:30 | 1.8810 | 1.8253 | 2025-01-13 07:30 | 1.8900 | STOCH_EXIT | 0 | 398.41 | 0.38% | 1.8710 | 1.9924 | 0.0557 | M30 bullish FVG [1.8500, 1.8710], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 24 | 2025-01-14 20:30 | 1.9360 | 1.8785 | 2025-01-15 08:00 | 1.9550 | STOCH_EXIT | 9 | 938.00 | 0.88% | 1.9480 | 2.0510 | 0.0575 | M30 bullish FVG [1.9360, 1.9480], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 25 | 2025-01-16 16:30 | 2.0010 | 1.9385 | 2025-01-17 07:30 | 2.0040 | STOCH_EXIT | 16 | 53.87 | 0.05% | 1.9940 | 2.1260 | 0.0625 | M30 bullish FVG [1.9830, 1.9940], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 26 | 2025-01-17 21:30 | 2.0340 | 1.9692 | 2025-01-18 07:00 | 2.0570 | STOCH_EXIT | 5 | 1110.59 | 1.03% | 2.0010 | 2.1637 | 0.0648 | M30 bullish FVG [1.9940, 2.0010], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 27 | 2025-01-29 09:00 | 2.0590 | 1.9929 | 2025-01-29 11:00 | 2.0650 | STOCH_EXIT | 4 | 208.71 | 0.19% | 2.0580 | 2.1911 | 0.0661 | M30 bullish FVG [2.0410, 2.0580], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 28 | 2025-02-04 11:00 | 2.1370 | 2.0662 | 2025-02-05 08:00 | 2.1150 | STOCH_EXIT | 29 | -1230.43 | -1.13% | 2.1360 | 2.2785 | 0.0708 | M30 bullish FVG [2.1290, 2.1360], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 29 | 2025-02-06 11:30 | 2.1540 | 2.0799 | 2025-02-06 17:00 | 2.1530 | STOCH_EXIT | 11 | -157.67 | -0.15% | 2.1640 | 2.3021 | 0.0741 | M30 bullish FVG [2.1450, 2.1640], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 30 | 2025-02-10 18:30 | 2.2420 | 2.1707 | 2025-02-10 21:30 | 2.2630 | STOCH_EXIT | 6 | 899.88 | 0.84% | 2.2360 | 2.3846 | 0.0713 | M30 bullish FVG [2.2030, 2.2360], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 31 | 2025-02-13 10:30 | 2.4450 | 2.3598 | 2025-02-13 19:00 | 2.4610 | STOCH_EXIT | 17 | 596.07 | 0.55% | 2.4500 | 2.6155 | 0.0852 | M30 bullish FVG [2.4420, 2.4500], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 32 | 2025-02-14 16:30 | 2.5130 | 2.4271 | 2025-02-14 19:00 | 2.5940 | STOCH_EXIT | 5 | 3373.20 | 3.12% | 2.5850 | 2.6847 | 0.0859 | M30 bullish FVG [2.4770, 2.5850], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 33 | 2025-02-28 21:30 | 2.4510 | 2.3287 | 2025-03-01 04:30 | 2.5330 | STOCH_EXIT | 10 | 3657.37 | 3.24% | 2.4300 | 2.6957 | 0.1223 | M30 bullish FVG [2.4110, 2.4300], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 34 | 2025-03-05 23:30 | 2.5970 | 2.4651 | 2025-03-06 08:00 | 2.6380 | STOCH_EXIT | 4 | 1688.83 | 1.48% | 2.6370 | 2.8608 | 0.1319 | M30 bullish FVG [2.5930, 2.6370], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 35 | 2025-04-28 20:00 | 1.8680 | 1.7816 | 2025-04-29 07:30 | 1.8710 | STOCH_EXIT | 9 | 71.22 | 0.06% | 1.8730 | 2.0407 | 0.0864 | M30 bullish FVG [1.8640, 1.8730], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 36 | 2025-08-07 21:00 | 1.6350 | 1.5890 | 2025-08-07 23:30 | 1.6880 | STOCH_EXIT | 5 | 3696.37 | 3.14% | 1.5860 | 1.7270 | 0.0460 | M30 bullish FVG [1.5670, 1.5860], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 37 | 2025-08-15 10:30 | 1.7870 | 1.7168 | 2025-08-15 13:00 | 1.8000 | STOCH_EXIT | 5 | 762.04 | 0.63% | 1.7920 | 1.9274 | 0.0702 | M30 bullish FVG [1.7800, 1.7920], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 38 | 2025-08-18 18:00 | 1.7680 | 1.6987 | 2025-08-18 21:30 | 1.7910 | STOCH_EXIT | 7 | 1464.21 | 1.20% | 1.7520 | 1.9065 | 0.0693 | M30 bullish FVG [1.7440, 1.7520], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 39 | 2025-08-19 13:00 | 1.7990 | 1.7270 | 2025-08-19 22:00 | 1.8020 | STOCH_EXIT | 18 | 82.77 | 0.07% | 1.7710 | 1.9429 | 0.0720 | M30 bullish FVG [1.7600, 1.7710], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 40 | 2025-08-28 20:30 | 1.7250 | 1.6596 | 2025-08-29 08:00 | 1.7200 | STOCH_EXIT | 10 | -484.02 | -0.39% | 1.7300 | 1.8557 | 0.0654 | M30 bullish FVG [1.7180, 1.7300], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 41 | 2025-09-03 10:30 | 1.6630 | 1.6025 | 2025-09-03 15:30 | 1.6670 | STOCH_EXIT | 10 | 172.79 | 0.14% | 1.6720 | 1.7841 | 0.0605 | M30 bullish FVG [1.6630, 1.6720], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 42 | 2025-10-20 20:00 | 1.5230 | 1.4702 | 2025-10-20 23:30 | 1.5300 | STOCH_EXIT | 7 | 443.35 | 0.36% | 1.5310 | 1.6287 | 0.0528 | M30 bullish FVG [1.5190, 1.5310], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 99460.63
- Макс: 125069.31
- Финал: 124587.26

Полная кривая — в `*_equity.csv`.
