# Бэктест RTKM (Minutes30)

- Период: 2023-11-15 22:37 — 2025-11-15 22:37
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.25 |
| MaxWaitBars | 30 |
| StochPeriod | 21 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 2 |
| HTFTrendEMA | 50 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 21 |
| Выигрышных / проигрышных | 17 / 4 |
| Win rate | 80.95% |
| Gross profit / loss | 23226.77 / 8318.20 |
| Profit factor | 2.792 |
| Чистый PnL | 14908.57 (14.91%) |
| Стартовый / финальный капитал | 100000.00 / 114908.57 |
| Макс. просадка | 4704.45 (4.45%) |
| Средняя прибыль / убыток | 1366.28 / 2079.55 |
| Expectancy | 709.93 |
| Лучшая / худшая сделка | 3368.17 / -3555.69 |
| Exposure | 2.37% |
| CAGR | 7.19% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-02-07 19:00 | 76.0100 | 74.6525 | 2024-02-09 17:30 | 75.4700 | STOCH_EXIT | 55 | -806.62 | -0.81% | 76.4500 | 78.7251 | 1.3575 | M30 bullish FVG [76.1000, 76.4500], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 2 | 2024-02-20 12:00 | 84.9100 | 82.9337 | 2024-02-20 14:00 | 87.9000 | STOCH_EXIT | 4 | 3368.17 | 3.42% | 86.7600 | 88.8625 | 1.9763 | M30 bullish FVG [86.2700, 86.7600], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 3 | 2024-02-20 22:30 | 85.4000 | 83.4237 | 2024-02-23 17:00 | 86.8200 | STOCH_EXIT | 74 | 1600.67 | 1.56% | 87.3500 | 89.3525 | 1.9763 | M30 bullish FVG [86.0500, 87.3500], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 4 | 2024-03-21 15:00 | 94.5100 | 92.1128 | 2024-03-22 22:30 | 92.1128 | SL | 44 | -2739.55 | -2.63% | 95.0400 | 99.3044 | 2.3972 | M30 bullish FVG [94.3000, 95.0400], stochastic crossed below 20 | FVG stop 92.1128 |
| 5 | 2024-03-26 18:30 | 94.0300 | 91.4408 | 2024-03-27 23:30 | 95.1000 | STOCH_EXIT | 39 | 1043.72 | 1.04% | 94.6700 | 99.2085 | 2.5892 | M30 bullish FVG [93.9300, 94.6700], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 6 | 2024-04-01 13:00 | 102.4200 | 99.4934 | 2024-04-01 14:30 | 104.0400 | STOCH_EXIT | 3 | 1501.60 | 1.48% | 104.0400 | 108.2732 | 2.9266 | M30 bullish FVG [102.4600, 104.0400], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 7 | 2024-04-01 19:00 | 102.2900 | 99.3634 | 2024-04-03 10:30 | 102.8000 | STOCH_EXIT | 41 | 411.53 | 0.40% | 104.0400 | 108.1432 | 2.9266 | M30 bullish FVG [102.4600, 104.0400], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 8 | 2024-04-09 18:00 | 97.3600 | 94.5719 | 2024-04-10 13:30 | 97.7400 | STOCH_EXIT | 20 | 302.22 | 0.29% | 98.6100 | 102.9362 | 2.7881 | M30 bullish FVG [97.4300, 98.6100], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 9 | 2024-05-21 17:00 | 103.3200 | 100.7631 | 2024-05-22 10:30 | 105.6800 | STOCH_EXIT | 16 | 2278.06 | 2.18% | 104.6300 | 108.4338 | 2.5569 | M30 bullish FVG [103.6300, 104.6300], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 10 | 2024-05-24 14:30 | 104.2800 | 101.5464 | 2024-05-26 11:30 | 105.5600 | STOCH_EXIT | 50 | 1198.58 | 1.13% | 106.3600 | 109.7473 | 2.7336 | M30 bullish FVG [105.6500, 106.3600], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 11 | 2024-08-14 18:30 | 85.7900 | 82.8529 | 2024-08-16 09:30 | 84.9100 | STOCH_EXIT | 40 | -1216.34 | -1.12% | 85.8900 | 91.6642 | 2.9371 | M30 bullish FVG [85.0000, 85.8900], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 12 | 2025-03-05 07:30 | 71.5700 | 68.1474 | 2025-03-05 11:30 | 73.4800 | STOCH_EXIT | 8 | 2737.84 | 2.57% | 71.2800 | 78.4152 | 3.4226 | M30 bullish FVG [70.4000, 71.2800], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 13 | 2025-07-03 23:00 | 58.2700 | 56.5108 | 2025-07-04 12:30 | 58.5100 | STOCH_EXIT | 14 | 341.43 | 0.31% | 58.4900 | 61.7883 | 1.7592 | M30 bullish FVG [57.8000, 58.4900], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 14 | 2025-07-09 09:30 | 58.9500 | 57.0913 | 2025-07-09 18:30 | 59.3600 | STOCH_EXIT | 18 | 652.57 | 0.59% | 58.2800 | 62.6674 | 1.8587 | M30 bullish FVG [57.7000, 58.2800], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 15 | 2025-07-11 08:00 | 59.8400 | 57.9665 | 2025-07-11 09:00 | 57.9665 | SL | 2 | -3555.69 | -3.23% | 60.7900 | 63.5871 | 1.8735 | M30 bullish FVG [60.2700, 60.7900], stochastic crossed below 20 | FVG stop 57.9665 |
| 16 | 2025-07-17 17:00 | 61.5800 | 59.5175 | 2025-07-18 13:30 | 63.0500 | STOCH_EXIT | 28 | 2435.30 | 2.28% | 63.0600 | 65.7049 | 2.0625 | M30 bullish FVG [62.1400, 63.0600], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 17 | 2025-08-21 18:30 | 69.4100 | 67.2508 | 2025-08-22 23:00 | 69.9600 | STOCH_EXIT | 44 | 754.09 | 0.69% | 71.3800 | 73.7283 | 2.1592 | M30 bullish FVG [70.8300, 71.3800], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 18 | 2025-08-27 09:30 | 73.2000 | 70.9538 | 2025-08-27 10:00 | 73.8400 | STOCH_EXIT | 1 | 849.72 | 0.77% | 73.1100 | 77.6923 | 2.2462 | M30 bullish FVG [72.5000, 73.1100], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 19 | 2025-10-17 17:00 | 63.9800 | 61.2315 | 2025-10-18 10:30 | 65.4000 | STOCH_EXIT | 28 | 2344.69 | 2.12% | 63.2100 | 69.4770 | 2.7485 | M30 bullish FVG [61.5200, 63.2100], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 20 | 2025-10-21 07:30 | 65.5000 | 62.7796 | 2025-10-21 13:30 | 66.3400 | STOCH_EXIT | 12 | 1339.16 | 1.18% | 66.2600 | 70.9408 | 2.7204 | M30 bullish FVG [65.5600, 66.2600], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 21 | 2025-10-22 22:30 | 62.9000 | 60.0639 | 2025-10-23 16:00 | 63.0000 | STOCH_EXIT | 22 | 67.43 | 0.06% | 64.2100 | 68.5722 | 2.8361 | M30 bullish FVG [63.4400, 64.2100], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 98483.01
- Макс: 114929.50
- Финал: 114908.57

Полная кривая — в `*_equity.csv`.
