# Бэктест UPRO (Minutes30)

- Период: 2023-11-15 22:35 — 2025-11-15 22:35
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.3 |
| MaxWaitBars | 30 |
| StochPeriod | 11 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 2 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 1 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 21 |
| Выигрышных / проигрышных | 16 / 5 |
| Win rate | 76.19% |
| Gross profit / loss | 17656.70 / 3956.69 |
| Profit factor | 4.462 |
| Чистый PnL | 13700.01 (13.70%) |
| Стартовый / финальный капитал | 100000.00 / 113700.01 |
| Макс. просадка | 5060.51 (4.75%) |
| Средняя прибыль / убыток | 1103.54 / 791.34 |
| Expectancy | 652.38 |
| Лучшая / худшая сделка | 3439.68 / -1324.07 |
| Exposure | 1.32% |
| CAGR | 6.62% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-12-06 11:00 | 1.9470 | 1.8749 | 2023-12-07 09:30 | 1.9250 | STOCH_EXIT | 26 | -1220.74 | -1.23% | 1.9390 | 2.0912 | 0.0721 | M30 bullish FVG [1.9060, 1.9390], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 2 | 2024-01-24 16:00 | 2.1180 | 2.0679 | 2024-01-25 22:00 | 2.1160 | STOCH_EXIT | 41 | -189.38 | -0.19% | 2.1340 | 2.2183 | 0.0501 | M30 bullish FVG [2.1100, 2.1340], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 3 | 2024-02-22 17:00 | 2.1320 | 2.0583 | 2024-02-22 21:30 | 2.1470 | STOCH_EXIT | 9 | 591.58 | 0.60% | 2.1200 | 2.2794 | 0.0737 | M30 bullish FVG [2.0910, 2.1200], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 4 | 2024-04-17 10:30 | 2.2160 | 2.1669 | 2024-04-17 15:30 | 2.2220 | STOCH_EXIT | 10 | 166.36 | 0.17% | 2.1550 | 2.3143 | 0.0491 | M30 bullish FVG [2.0900, 2.1550], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 5 | 2024-04-22 16:00 | 2.2280 | 2.1788 | 2024-04-22 22:30 | 2.2330 | STOCH_EXIT | 13 | 121.86 | 0.12% | 2.2380 | 2.3265 | 0.0492 | M30 bullish FVG [2.1870, 2.2380], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 6 | 2024-05-07 16:00 | 2.2060 | 2.1585 | 2024-05-07 22:00 | 2.2410 | STOCH_EXIT | 12 | 1474.94 | 1.49% | 2.2110 | 2.3010 | 0.0475 | M30 bullish FVG [2.1930, 2.2110], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 7 | 2024-05-28 17:00 | 1.9870 | 1.9308 | 2024-05-29 11:00 | 1.9900 | STOCH_EXIT | 17 | 50.58 | 0.05% | 1.9880 | 2.0993 | 0.0562 | M30 bullish FVG [1.9610, 1.9880], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 8 | 2024-06-13 22:30 | 1.9740 | 1.9097 | 2024-06-14 16:30 | 2.0230 | STOCH_EXIT | 17 | 2397.08 | 2.38% | 1.9370 | 2.1026 | 0.0643 | M30 bullish FVG [1.9100, 1.9370], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 9 | 2024-06-14 23:00 | 2.0120 | 1.9429 | 2024-06-15 12:00 | 2.0230 | STOCH_EXIT | 6 | 458.11 | 0.45% | 2.0090 | 2.1503 | 0.0691 | M30 bullish FVG [1.9880, 2.0090], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 10 | 2024-06-20 23:00 | 1.9380 | 1.8718 | 2024-06-21 16:00 | 1.9410 | STOCH_EXIT | 15 | 56.21 | 0.05% | 1.9360 | 2.0703 | 0.0662 | M30 bullish FVG [1.9090, 1.9360], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 11 | 2024-08-13 11:30 | 1.8750 | 1.8208 | 2024-08-13 12:30 | 1.9060 | STOCH_EXIT | 2 | 1601.02 | 1.55% | 1.8770 | 1.9834 | 0.0542 | M30 bullish FVG [1.8390, 1.8770], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 12 | 2024-10-04 10:30 | 1.6950 | 1.6470 | 2024-10-04 19:30 | 1.7020 | STOCH_EXIT | 18 | 328.69 | 0.31% | 1.7010 | 1.7910 | 0.0480 | M30 bullish FVG [1.6850, 1.7010], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 13 | 2024-10-10 12:30 | 1.7160 | 1.6716 | 2024-10-11 20:30 | 1.6960 | STOCH_EXIT | 49 | -1324.07 | -1.26% | 1.7080 | 1.8048 | 0.0444 | M30 bullish FVG [1.6910, 1.7080], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 14 | 2024-11-21 10:30 | 1.7520 | 1.6882 | 2024-11-21 22:00 | 1.7390 | STOCH_EXIT | 23 | -869.98 | -0.84% | 1.7550 | 1.8796 | 0.0638 | M30 bullish FVG [1.7350, 1.7550], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 15 | 2025-01-13 07:30 | 1.8810 | 1.8253 | 2025-01-13 07:30 | 1.8900 | STOCH_EXIT | 0 | 391.30 | 0.38% | 1.8710 | 1.9924 | 0.0557 | M30 bullish FVG [1.8500, 1.8710], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 16 | 2025-02-10 18:30 | 2.2420 | 2.1707 | 2025-02-10 21:30 | 2.2630 | STOCH_EXIT | 6 | 862.39 | 0.84% | 2.2360 | 2.3846 | 0.0713 | M30 bullish FVG [2.2030, 2.2360], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 17 | 2025-02-14 16:30 | 2.5130 | 2.4271 | 2025-02-14 19:00 | 2.5940 | STOCH_EXIT | 5 | 3216.31 | 3.12% | 2.5850 | 2.6847 | 0.0859 | M30 bullish FVG [2.4770, 2.5850], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 18 | 2025-03-05 23:30 | 2.5970 | 2.4651 | 2025-03-06 08:00 | 2.6380 | STOCH_EXIT | 4 | 1573.68 | 1.48% | 2.6370 | 2.8608 | 0.1319 | M30 bullish FVG [2.5930, 2.6370], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 19 | 2025-08-07 21:00 | 1.6350 | 1.5890 | 2025-08-07 23:30 | 1.6880 | STOCH_EXIT | 5 | 3439.68 | 3.14% | 1.5860 | 1.7270 | 0.0460 | M30 bullish FVG [1.5670, 1.5860], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 20 | 2025-10-31 10:30 | 1.4080 | 1.3524 | 2025-10-31 19:30 | 1.4050 | STOCH_EXIT | 18 | -352.52 | -0.31% | 1.4210 | 1.5192 | 0.0556 | M30 bullish FVG [1.3870, 1.4210], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 21 | 2025-11-07 16:30 | 1.4070 | 1.3562 | 2025-11-08 02:00 | 1.4200 | STOCH_EXIT | 15 | 926.92 | 0.82% | 1.4090 | 1.5086 | 0.0508 | M30 bullish FVG [1.3920, 1.4090], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 97812.71
- Макс: 113700.01
- Финал: 113700.01

Полная кривая — в `*_equity.csv`.
