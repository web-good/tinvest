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
| StochPeriod | 21 |
| UseStochExit | 1 |
| StopATR | 1.7 |
| TakeProfitATR | 0 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 20 |
| Выигрышных / проигрышных | 16 / 4 |
| Win rate | 80.00% |
| Gross profit / loss | 26950.50 / 6154.34 |
| Profit factor | 4.379 |
| Чистый PnL | 20796.15 (20.80%) |
| Стартовый / финальный капитал | 100000.00 / 120796.15 |
| Макс. просадка | 7001.60 (6.34%) |
| Средняя прибыль / убыток | 1684.41 / 1538.59 |
| Expectancy | 1039.81 |
| Лучшая / худшая сделка | 5823.83 / -1780.17 |
| Exposure | 2.35% |
| CAGR | 9.89% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-02-20 12:00 | 84.9100 | 81.5503 | 2024-02-20 14:00 | 87.9000 | STOCH_EXIT | 4 | 3397.21 | 3.42% | 86.0000 | 0.0000 | 1.9763 | M30 bullish FVG [85.2000, 86.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 2 | 2024-02-20 22:30 | 85.4000 | 82.0403 | 2024-02-23 17:00 | 86.8200 | STOCH_EXIT | 74 | 1614.01 | 1.56% | 87.3500 | 0.0000 | 1.9763 | M30 bullish FVG [86.0500, 87.3500], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 3 | 2024-03-21 15:00 | 94.5100 | 90.4348 | 2024-03-25 09:30 | 93.0000 | STOCH_EXIT | 83 | -1780.17 | -1.70% | 95.0400 | 0.0000 | 2.3972 | M30 bullish FVG [94.3000, 95.0400], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 4 | 2024-03-27 13:30 | 93.6800 | 89.1495 | 2024-03-27 23:30 | 95.1000 | STOCH_EXIT | 20 | 1458.17 | 1.41% | 95.2500 | 0.0000 | 2.6650 | M30 bullish FVG [94.4500, 95.2500], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 5 | 2024-04-01 13:00 | 102.4200 | 97.4448 | 2024-04-01 14:30 | 104.0400 | STOCH_EXIT | 3 | 1547.11 | 1.48% | 104.0400 | 0.0000 | 2.9266 | M30 bullish FVG [102.4600, 104.0400], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 6 | 2024-04-01 19:00 | 102.2900 | 97.3148 | 2024-04-03 10:30 | 102.8000 | STOCH_EXIT | 41 | 419.68 | 0.40% | 104.0400 | 0.0000 | 2.9266 | M30 bullish FVG [102.4600, 104.0400], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 7 | 2024-04-09 18:00 | 97.3600 | 92.6202 | 2024-04-10 13:30 | 97.7400 | STOCH_EXIT | 20 | 307.87 | 0.29% | 98.6100 | 0.0000 | 2.7881 | M30 bullish FVG [97.4300, 98.6100], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 8 | 2024-05-21 17:00 | 103.3200 | 98.9732 | 2024-05-22 10:30 | 105.6800 | STOCH_EXIT | 16 | 2323.17 | 2.18% | 104.6300 | 0.0000 | 2.5569 | M30 bullish FVG [103.6300, 104.6300], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 9 | 2024-07-09 11:30 | 84.7300 | 79.5046 | 2024-07-11 11:00 | 83.5700 | STOCH_EXIT | 57 | -1592.51 | -1.47% | 85.2800 | 0.0000 | 3.0738 | M30 bullish FVG [84.3200, 85.2800], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 10 | 2024-08-14 18:30 | 85.7900 | 80.7969 | 2024-08-16 09:30 | 84.9100 | STOCH_EXIT | 40 | -1206.69 | -1.12% | 85.8900 | 0.0000 | 2.9371 | M30 bullish FVG [85.0000, 85.8900], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 11 | 2024-12-19 18:30 | 51.4600 | 47.7355 | 2024-12-20 13:30 | 54.3400 | STOCH_EXIT | 24 | 5823.83 | 5.49% | 52.7300 | 0.0000 | 2.1909 | M30 bullish FVG [51.5000, 52.7300], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 12 | 2025-04-10 15:00 | 60.5500 | 54.5055 | 2025-04-11 08:30 | 60.8300 | STOCH_EXIT | 22 | 405.72 | 0.36% | 60.5400 | 0.0000 | 3.5556 | M30 bullish FVG [58.9800, 60.5400], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 13 | 2025-06-26 08:00 | 55.9000 | 52.6795 | 2025-06-26 13:00 | 56.2600 | STOCH_EXIT | 10 | 610.88 | 0.54% | 55.3000 | 0.0000 | 1.8944 | M30 bullish FVG [54.7000, 55.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 14 | 2025-07-03 23:00 | 58.2700 | 55.2794 | 2025-07-04 12:30 | 58.5100 | STOCH_EXIT | 14 | 352.32 | 0.31% | 58.4900 | 0.0000 | 1.7592 | M30 bullish FVG [57.8000, 58.4900], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 15 | 2025-07-09 09:30 | 58.9500 | 55.7902 | 2025-07-09 18:30 | 59.3600 | STOCH_EXIT | 18 | 673.62 | 0.59% | 58.2800 | 0.0000 | 1.8587 | M30 bullish FVG [57.7000, 58.2800], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 16 | 2025-07-17 17:00 | 61.5800 | 58.0738 | 2025-07-18 13:30 | 63.0500 | STOCH_EXIT | 28 | 2604.22 | 2.28% | 63.0600 | 0.0000 | 2.0625 | M30 bullish FVG [62.1400, 63.0600], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 17 | 2025-10-07 10:00 | 58.0100 | 54.2016 | 2025-10-07 10:30 | 59.0600 | STOCH_EXIT | 1 | 1992.84 | 1.71% | 57.6000 | 0.0000 | 2.2402 | M30 bullish FVG [56.6900, 57.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 18 | 2025-10-07 21:30 | 59.0000 | 55.1916 | 2025-10-08 07:30 | 59.5200 | STOCH_EXIT | 7 | 926.09 | 0.78% | 59.0600 | 0.0000 | 2.2402 | M30 bullish FVG [58.2500, 59.0600], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 19 | 2025-10-10 07:30 | 57.5500 | 53.1824 | 2025-10-11 09:30 | 56.8500 | STOCH_EXIT | 48 | -1574.98 | -1.32% | 56.6300 | 0.0000 | 2.5692 | M30 bullish FVG [55.5000, 56.6300], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 20 | 2025-10-17 17:00 | 63.9800 | 59.3075 | 2025-10-18 10:30 | 65.4000 | STOCH_EXIT | 28 | 2493.77 | 2.12% | 63.2100 | 0.0000 | 2.7485 | M30 bullish FVG [61.5200, 63.2100], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 100000.00
- Макс: 120815.91
- Финал: 120796.15

Полная кривая — в `*_equity.csv`.
