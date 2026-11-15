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
| StopATR | 1.2 |
| TakeProfitATR | 0 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 22 |
| Выигрышных / проигрышных | 16 / 6 |
| Win rate | 72.73% |
| Gross profit / loss | 22588.77 / 6719.28 |
| Profit factor | 3.362 |
| Чистый PnL | 15869.49 (15.87%) |
| Стартовый / финальный капитал | 100000.00 / 115869.49 |
| Макс. просадка | 4367.70 (4.19%) |
| Средняя прибыль / убыток | 1411.80 / 1119.88 |
| Expectancy | 721.34 |
| Лучшая / худшая сделка | 5965.18 / -2742.29 |
| Exposure | 1.83% |
| CAGR | 7.63% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-02-20 10:30 | 86.0000 | 83.6285 | 2024-02-20 14:00 | 87.9000 | STOCH_EXIT | 7 | 2103.14 | 2.11% | 86.0000 | 0.0000 | 1.9763 | M30 bullish FVG [85.2000, 86.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 2 | 2024-02-20 21:00 | 86.2800 | 83.9085 | 2024-02-22 09:30 | 86.7700 | STOCH_EXIT | 35 | 476.10 | 0.47% | 87.3500 | 0.0000 | 1.9763 | M30 bullish FVG [86.0500, 87.3500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 3 | 2024-03-21 13:30 | 94.6400 | 91.7634 | 2024-03-24 16:30 | 92.8900 | STOCH_EXIT | 81 | -1991.27 | -1.95% | 95.0400 | 0.0000 | 2.3972 | M30 bullish FVG [94.3000, 95.0400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 4 | 2024-03-27 13:30 | 93.6800 | 90.4820 | 2024-03-27 18:30 | 95.4000 | STOCH_EXIT | 10 | 1739.24 | 1.73% | 95.2500 | 0.0000 | 2.6650 | M30 bullish FVG [94.4500, 95.2500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 5 | 2024-03-28 11:00 | 94.5300 | 91.3521 | 2024-03-28 12:00 | 95.6800 | STOCH_EXIT | 2 | 1139.29 | 1.12% | 95.2500 | 0.0000 | 2.6482 | M30 bullish FVG [94.4500, 95.2500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 6 | 2024-04-01 13:00 | 102.4200 | 98.9081 | 2024-04-01 14:30 | 104.0400 | STOCH_EXIT | 3 | 1516.77 | 1.48% | 104.0400 | 0.0000 | 2.9266 | M30 bullish FVG [102.4600, 104.0400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 7 | 2024-04-09 18:00 | 97.3600 | 94.0143 | 2024-04-10 09:30 | 98.1000 | STOCH_EXIT | 12 | 687.23 | 0.66% | 98.6100 | 0.0000 | 2.7881 | M30 bullish FVG [97.4300, 98.6100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 8 | 2024-05-07 11:30 | 96.7900 | 94.4366 | 2024-05-07 21:00 | 96.4400 | STOCH_EXIT | 19 | -486.81 | -0.46% | 96.4500 | 0.0000 | 1.9612 | M30 bullish FVG [95.7000, 96.4500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 9 | 2024-05-21 17:00 | 103.3200 | 100.2517 | 2024-05-22 10:30 | 105.6800 | STOCH_EXIT | 16 | 2278.06 | 2.18% | 104.6300 | 0.0000 | 2.5569 | M30 bullish FVG [103.6300, 104.6300], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 10 | 2024-07-08 19:00 | 85.3300 | 81.5986 | 2024-07-09 09:30 | 86.2900 | STOCH_EXIT | 10 | 1092.74 | 1.02% | 85.2800 | 0.0000 | 3.1095 | M30 bullish FVG [84.3200, 85.2800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 11 | 2024-07-17 22:00 | 86.4200 | 82.3143 | 2024-07-18 11:00 | 87.5800 | STOCH_EXIT | 7 | 1341.25 | 1.24% | 86.2400 | 0.0000 | 3.4214 | M30 bullish FVG [84.9800, 86.2400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 12 | 2024-08-14 13:30 | 85.6900 | 82.1655 | 2024-08-16 09:30 | 84.9100 | STOCH_EXIT | 48 | -1107.58 | -1.01% | 85.8900 | 0.0000 | 2.9371 | M30 bullish FVG [85.0000, 85.8900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 13 | 2024-12-19 18:30 | 51.4600 | 48.8310 | 2024-12-20 13:30 | 54.3400 | STOCH_EXIT | 24 | 5965.18 | 5.49% | 52.7300 | 0.0000 | 2.1909 | M30 bullish FVG [51.5000, 52.7300], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 14 | 2025-01-17 23:30 | 67.8000 | 64.5624 | 2025-01-18 16:00 | 68.3200 | STOCH_EXIT | 19 | 763.78 | 0.67% | 68.1000 | 0.0000 | 2.6980 | M30 bullish FVG [67.2700, 68.1000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 15 | 2025-04-10 12:00 | 60.3100 | 56.0433 | 2025-04-11 08:30 | 60.8300 | STOCH_EXIT | 28 | 877.51 | 0.76% | 60.5400 | 0.0000 | 3.5556 | M30 bullish FVG [58.9800, 60.5400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 16 | 2025-06-26 07:30 | 56.1700 | 53.8967 | 2025-06-26 13:00 | 56.2600 | STOCH_EXIT | 11 | 69.93 | 0.06% | 55.3000 | 0.0000 | 1.8944 | M30 bullish FVG [54.7000, 55.3000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 17 | 2025-06-30 22:30 | 58.3100 | 56.0996 | 2025-07-01 09:00 | 58.1900 | STOCH_EXIT | 8 | -354.72 | -0.31% | 58.1700 | 0.0000 | 1.8420 | M30 bullish FVG [56.7200, 58.1700], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 18 | 2025-07-03 21:30 | 58.4700 | 56.3590 | 2025-07-04 12:30 | 58.5100 | STOCH_EXIT | 17 | -36.61 | -0.03% | 58.4900 | 0.0000 | 1.7592 | M30 bullish FVG [57.8000, 58.4900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 19 | 2025-07-17 16:30 | 61.7700 | 59.2951 | 2025-07-18 07:00 | 62.3900 | STOCH_EXIT | 16 | 1043.31 | 0.90% | 63.0600 | 0.0000 | 2.0625 | M30 bullish FVG [62.1400, 63.0600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 20 | 2025-10-07 20:00 | 58.9700 | 56.2817 | 2025-10-07 23:00 | 59.4700 | STOCH_EXIT | 6 | 872.74 | 0.75% | 59.0600 | 0.0000 | 2.2402 | M30 bullish FVG [58.2500, 59.0600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 21 | 2025-10-09 22:30 | 58.2200 | 55.4066 | 2025-10-11 04:00 | 56.9200 | STOCH_EXIT | 42 | -2742.29 | -2.33% | 56.6300 | 0.0000 | 2.3445 | M30 bullish FVG [55.5000, 56.6300], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 22 | 2025-10-17 11:30 | 63.9600 | 60.6618 | 2025-10-17 18:00 | 64.3700 | STOCH_EXIT | 13 | 622.50 | 0.54% | 63.2100 | 0.0000 | 2.7485 | M30 bullish FVG [61.5200, 63.2100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 98639.32
- Макс: 117989.28
- Финал: 115869.49

Полная кривая — в `*_equity.csv`.
