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
| StopATR | 1.5 |
| TakeProfitATR | 1 |
| HTFTrendEMA | 50 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 22 |
| Выигрышных / проигрышных | 18 / 4 |
| Win rate | 81.82% |
| Gross profit / loss | 25936.40 / 7228.19 |
| Profit factor | 3.588 |
| Чистый PnL | 18708.21 (18.71%) |
| Стартовый / финальный капитал | 100000.00 / 118708.21 |
| Макс. просадка | 4830.03 (4.22%) |
| Средняя прибыль / убыток | 1440.91 / 1807.05 |
| Expectancy | 850.37 |
| Лучшая / худшая сделка | 2811.34 / -3380.06 |
| Exposure | 2.48% |
| CAGR | 8.94% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-02-07 19:00 | 76.0100 | 73.9737 | 2024-02-09 17:30 | 75.4700 | STOCH_EXIT | 55 | -806.62 | -0.81% | 76.4500 | 77.3675 | 1.3575 | M30 bullish FVG [76.1000, 76.4500], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 2 | 2024-02-20 12:00 | 84.9100 | 81.9456 | 2024-02-20 14:00 | 86.8863 | TP | 4 | 2192.83 | 2.23% | 86.7600 | 86.8863 | 1.9763 | M30 bullish FVG [86.2700, 86.7600], stochastic crossed below 20 | FVG target 86.8863 |
| 3 | 2024-02-20 22:30 | 85.4000 | 82.4356 | 2024-02-21 10:00 | 87.3763 | TP | 4 | 2230.06 | 2.21% | 87.3500 | 87.3763 | 1.9763 | M30 bullish FVG [86.0500, 87.3500], stochastic crossed below 20 | FVG target 87.3763 |
| 4 | 2024-02-21 12:30 | 85.6000 | 82.3577 | 2024-02-21 16:30 | 87.7615 | TP | 8 | 2489.83 | 2.42% | 86.8700 | 87.7615 | 2.1615 | M30 bullish FVG [85.8000, 86.8700], stochastic crossed below 20 | FVG target 87.7615 |
| 5 | 2024-03-21 15:00 | 94.5100 | 90.9142 | 2024-03-25 09:30 | 93.0000 | STOCH_EXIT | 83 | -1796.21 | -1.70% | 95.0400 | 96.9072 | 2.3972 | M30 bullish FVG [94.3000, 95.0400], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 6 | 2024-03-26 18:30 | 94.0300 | 90.1461 | 2024-03-27 23:30 | 95.1000 | STOCH_EXIT | 39 | 1072.98 | 1.04% | 94.6700 | 96.6192 | 2.5892 | M30 bullish FVG [93.9300, 94.6700], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 7 | 2024-04-01 13:00 | 102.4200 | 98.0301 | 2024-04-01 14:30 | 104.0400 | STOCH_EXIT | 3 | 1547.11 | 1.48% | 104.0400 | 105.3466 | 2.9266 | M30 bullish FVG [102.4600, 104.0400], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 8 | 2024-04-01 19:00 | 102.2900 | 97.9001 | 2024-04-03 10:30 | 102.8000 | STOCH_EXIT | 41 | 423.75 | 0.40% | 104.0400 | 105.2166 | 2.9266 | M30 bullish FVG [102.4600, 104.0400], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 9 | 2024-04-09 18:00 | 97.3600 | 93.1779 | 2024-04-10 13:30 | 97.7400 | STOCH_EXIT | 20 | 310.70 | 0.29% | 98.6100 | 100.1481 | 2.7881 | M30 bullish FVG [97.4300, 98.6100], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 10 | 2024-05-21 17:00 | 103.3200 | 99.4846 | 2024-05-22 10:00 | 105.8769 | TP | 15 | 2550.42 | 2.37% | 104.6300 | 105.8769 | 2.5569 | M30 bullish FVG [103.6300, 104.6300], stochastic crossed below 20 | FVG target 105.8769 |
| 11 | 2024-05-24 14:30 | 104.2800 | 100.1795 | 2024-05-26 11:30 | 105.5600 | STOCH_EXIT | 50 | 1233.83 | 1.13% | 106.3600 | 107.0136 | 2.7336 | M30 bullish FVG [105.6500, 106.3600], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 12 | 2024-08-14 18:30 | 85.7900 | 81.3843 | 2024-08-16 09:30 | 84.9100 | STOCH_EXIT | 40 | -1245.30 | -1.12% | 85.8900 | 88.7271 | 2.9371 | M30 bullish FVG [85.0000, 85.8900], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 13 | 2025-03-05 07:30 | 71.5700 | 66.4361 | 2025-03-05 11:30 | 73.4800 | STOCH_EXIT | 8 | 2811.34 | 2.57% | 71.2800 | 74.9926 | 3.4226 | M30 bullish FVG [70.4000, 71.2800], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 14 | 2025-07-03 23:00 | 58.2700 | 55.6312 | 2025-07-04 12:30 | 58.5100 | STOCH_EXIT | 14 | 350.51 | 0.31% | 58.4900 | 60.0292 | 1.7592 | M30 bullish FVG [57.8000, 58.4900], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 15 | 2025-07-09 09:30 | 58.9500 | 56.1619 | 2025-07-09 18:30 | 59.3600 | STOCH_EXIT | 18 | 673.62 | 0.59% | 58.2800 | 60.8087 | 1.8587 | M30 bullish FVG [57.7000, 58.2800], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 16 | 2025-07-11 08:00 | 59.8400 | 57.0297 | 2025-07-12 11:30 | 58.1200 | STOCH_EXIT | 51 | -3380.06 | -2.97% | 60.7900 | 61.7135 | 1.8735 | M30 bullish FVG [60.2700, 60.7900], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 17 | 2025-07-17 17:00 | 61.5800 | 58.4863 | 2025-07-18 13:30 | 63.0500 | STOCH_EXIT | 28 | 2519.76 | 2.28% | 63.0600 | 63.6425 | 2.0625 | M30 bullish FVG [62.1400, 63.0600], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 18 | 2025-08-21 18:30 | 69.4100 | 66.1712 | 2025-08-22 23:00 | 69.9600 | STOCH_EXIT | 44 | 778.11 | 0.69% | 71.3800 | 71.5692 | 2.1592 | M30 bullish FVG [70.8300, 71.3800], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 19 | 2025-08-27 09:30 | 73.2000 | 69.8308 | 2025-08-27 10:00 | 73.8400 | STOCH_EXIT | 1 | 878.04 | 0.77% | 73.1100 | 75.4462 | 2.2462 | M30 bullish FVG [72.5000, 73.1100], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 20 | 2025-10-17 17:00 | 63.9800 | 59.8572 | 2025-10-18 10:30 | 65.4000 | STOCH_EXIT | 28 | 2426.00 | 2.12% | 63.2100 | 66.7285 | 2.7485 | M30 bullish FVG [61.5200, 63.2100], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 21 | 2025-10-21 07:30 | 65.5000 | 61.4194 | 2025-10-21 13:30 | 66.3400 | STOCH_EXIT | 12 | 1377.86 | 1.18% | 66.2600 | 68.2204 | 2.7204 | M30 bullish FVG [65.5600, 66.2600], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 22 | 2025-10-22 22:30 | 62.9000 | 58.6459 | 2025-10-23 16:00 | 63.0000 | STOCH_EXIT | 22 | 69.65 | 0.06% | 64.2100 | 65.7361 | 2.8361 | M30 bullish FVG [63.4400, 64.2100], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 98483.01
- Макс: 118729.83
- Финал: 118708.21

Полная кривая — в `*_equity.csv`.
