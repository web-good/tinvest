# Бэктест RTKM (Minutes30)

- Период: 2023-11-15 22:37 — 2025-11-15 22:37
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.3 |
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
| Всего сделок | 22 |
| Выигрышных / проигрышных | 16 / 6 |
| Win rate | 72.73% |
| Gross profit / loss | 22457.63 / 9487.58 |
| Profit factor | 2.367 |
| Чистый PnL | 12970.05 (12.97%) |
| Стартовый / финальный капитал | 100000.00 / 112970.05 |
| Макс. просадка | 5583.03 (4.76%) |
| Средняя прибыль / убыток | 1403.60 / 1581.26 |
| Expectancy | 589.55 |
| Лучшая / худшая сделка | 5908.64 / -4827.07 |
| Exposure | 1.66% |
| CAGR | 6.28% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-02-20 10:30 | 86.0000 | 84.0237 | 2024-02-20 14:00 | 87.9000 | STOCH_EXIT | 7 | 2103.14 | 2.11% | 86.0000 | 89.9525 | 1.9763 | M30 bullish FVG [85.2000, 86.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 2 | 2024-02-20 21:00 | 86.2800 | 84.3037 | 2024-02-22 09:30 | 86.7700 | STOCH_EXIT | 35 | 476.10 | 0.47% | 87.3500 | 90.2325 | 1.9763 | M30 bullish FVG [86.0500, 87.3500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 3 | 2024-03-21 13:30 | 94.6400 | 92.2428 | 2024-03-22 21:00 | 92.2428 | SL | 44 | -2689.88 | -2.63% | 95.0400 | 99.4344 | 2.3972 | M30 bullish FVG [94.3000, 95.0400], stochastic crossed below 20 | FVG stop 92.2428 |
| 4 | 2024-03-27 13:30 | 93.6800 | 91.0150 | 2024-03-27 18:30 | 95.4000 | STOCH_EXIT | 10 | 1722.99 | 1.73% | 95.2500 | 99.0100 | 2.6650 | M30 bullish FVG [94.4500, 95.2500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 5 | 2024-03-28 11:00 | 94.5300 | 91.8818 | 2024-03-28 12:00 | 95.6800 | STOCH_EXIT | 2 | 1128.74 | 1.12% | 95.2500 | 99.8265 | 2.6482 | M30 bullish FVG [94.4500, 95.2500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 6 | 2024-04-01 13:00 | 102.4200 | 99.4934 | 2024-04-01 14:30 | 104.0400 | STOCH_EXIT | 3 | 1516.77 | 1.48% | 104.0400 | 108.2732 | 2.9266 | M30 bullish FVG [102.4600, 104.0400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 7 | 2024-04-09 18:00 | 97.3600 | 94.5719 | 2024-04-10 09:30 | 98.1000 | STOCH_EXIT | 12 | 687.23 | 0.66% | 98.6100 | 102.9362 | 2.7881 | M30 bullish FVG [97.4300, 98.6100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 8 | 2024-05-07 11:30 | 96.7900 | 94.8288 | 2024-05-07 21:00 | 96.4400 | STOCH_EXIT | 19 | -482.34 | -0.46% | 96.4500 | 100.7123 | 1.9612 | M30 bullish FVG [95.7000, 96.4500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 9 | 2024-05-21 17:00 | 103.3200 | 100.7631 | 2024-05-22 10:30 | 105.6800 | STOCH_EXIT | 16 | 2278.06 | 2.18% | 104.6300 | 108.4338 | 2.5569 | M30 bullish FVG [103.6300, 104.6300], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 10 | 2024-07-08 19:00 | 85.3300 | 82.2205 | 2024-07-09 09:30 | 86.2900 | STOCH_EXIT | 10 | 1092.74 | 1.02% | 85.2800 | 91.5489 | 3.1095 | M30 bullish FVG [84.3200, 85.2800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 11 | 2024-07-17 22:00 | 86.4200 | 82.9986 | 2024-07-18 11:00 | 87.5800 | STOCH_EXIT | 7 | 1330.52 | 1.24% | 86.2400 | 93.2628 | 3.4214 | M30 bullish FVG [84.9800, 86.2400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 12 | 2024-08-14 13:30 | 85.6900 | 82.7529 | 2024-08-16 09:30 | 84.9100 | STOCH_EXIT | 48 | -1098.93 | -1.01% | 85.8900 | 91.5642 | 2.9371 | M30 bullish FVG [85.0000, 85.8900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 13 | 2024-12-19 18:30 | 51.4600 | 49.2691 | 2024-12-20 13:30 | 54.3400 | STOCH_EXIT | 24 | 5908.64 | 5.49% | 52.7300 | 55.8417 | 2.1909 | M30 bullish FVG [51.5000, 52.7300], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 14 | 2025-01-17 23:30 | 67.8000 | 65.1020 | 2025-01-18 16:00 | 68.3200 | STOCH_EXIT | 19 | 759.26 | 0.67% | 68.1000 | 73.1960 | 2.6980 | M30 bullish FVG [67.2700, 68.1000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 15 | 2025-04-10 12:00 | 60.3100 | 56.7544 | 2025-04-11 08:30 | 60.8300 | STOCH_EXIT | 28 | 872.92 | 0.76% | 60.5400 | 67.4212 | 3.5556 | M30 bullish FVG [58.9800, 60.5400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 16 | 2025-06-26 07:30 | 56.1700 | 54.2756 | 2025-06-26 13:00 | 56.2600 | STOCH_EXIT | 11 | 69.26 | 0.06% | 55.3000 | 59.9588 | 1.8944 | M30 bullish FVG [54.7000, 55.3000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 17 | 2025-06-30 22:30 | 58.3100 | 56.4680 | 2025-07-01 09:00 | 58.1900 | STOCH_EXIT | 8 | -352.93 | -0.31% | 58.1700 | 61.9939 | 1.8420 | M30 bullish FVG [56.7200, 58.1700], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 18 | 2025-07-03 21:30 | 58.4700 | 56.7108 | 2025-07-04 12:30 | 58.5100 | STOCH_EXIT | 17 | -36.43 | -0.03% | 58.4900 | 61.9883 | 1.7592 | M30 bullish FVG [57.8000, 58.4900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 19 | 2025-07-17 16:30 | 61.7700 | 59.7075 | 2025-07-18 07:00 | 62.3900 | STOCH_EXIT | 16 | 1037.73 | 0.90% | 63.0600 | 65.8949 | 2.0625 | M30 bullish FVG [62.1400, 63.0600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 20 | 2025-10-07 20:00 | 58.9700 | 56.7298 | 2025-10-07 23:00 | 59.4700 | STOCH_EXIT | 6 | 868.34 | 0.75% | 59.0600 | 63.4504 | 2.2402 | M30 bullish FVG [58.2500, 59.0600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 21 | 2025-10-09 22:30 | 58.2200 | 55.8755 | 2025-10-11 02:00 | 55.8755 | SL | 38 | -4827.07 | -4.12% | 56.6300 | 62.9090 | 2.3445 | M30 bullish FVG [55.5000, 56.6300], stochastic crossed below 20 | FVG stop 55.8755 |
| 22 | 2025-10-17 11:30 | 63.9600 | 61.2115 | 2025-10-17 18:00 | 64.3700 | STOCH_EXIT | 13 | 605.21 | 0.54% | 63.2100 | 69.4570 | 2.7485 | M30 bullish FVG [61.5200, 63.2100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 98639.32
- Макс: 117191.90
- Финал: 112970.05

Полная кривая — в `*_equity.csv`.
