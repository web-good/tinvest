# Бэктест RTKM (Minutes30)

- Период: 2025-11-15 22:39 — 2026-11-15 22:39
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
| Всего сделок | 14 |
| Выигрышных / проигрышных | 9 / 5 |
| Win rate | 64.29% |
| Gross profit / loss | 10156.21 / 8563.06 |
| Profit factor | 1.186 |
| Чистый PnL | 1593.15 (1.59%) |
| Стартовый / финальный капитал | 100000.00 / 101593.15 |
| Макс. просадка | 6383.23 (5.95%) |
| Средняя прибыль / убыток | 1128.47 / 1712.61 |
| Expectancy | 113.80 |
| Лучшая / худшая сделка | 4394.35 / -4802.60 |
| Exposure | 1.99% |
| CAGR | 1.59% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2025-12-12 22:00 | 60.9700 | 59.2701 | 2025-12-13 06:00 | 61.3700 | STOCH_EXIT | 12 | 552.29 | 0.56% | 62.0100 | 64.3699 | 1.6999 | M30 bullish FVG [61.0600, 62.0100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 2 | 2026-01-14 21:00 | 63.6000 | 62.1288 | 2026-01-15 07:00 | 64.0100 | STOCH_EXIT | 7 | 546.99 | 0.54% | 63.1500 | 66.5423 | 1.4712 | M30 bullish FVG [62.7300, 63.1500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 3 | 2026-01-22 11:00 | 64.2000 | 62.7608 | 2026-01-22 15:30 | 64.6800 | STOCH_EXIT | 9 | 652.43 | 0.65% | 63.7900 | 67.0784 | 1.4392 | M30 bullish FVG [63.3300, 63.7900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 4 | 2026-02-26 10:00 | 65.2700 | 63.8479 | 2026-02-27 10:00 | 65.0500 | STOCH_EXIT | 35 | -442.00 | -0.44% | 65.5100 | 68.1143 | 1.4221 | M30 bullish FVG [64.9300, 65.5100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 5 | 2026-06-01 12:00 | 50.4800 | 49.4645 | 2026-06-01 22:30 | 50.0800 | STOCH_EXIT | 21 | -900.56 | -0.89% | 50.8800 | 52.5110 | 1.0155 | M30 bullish FVG [50.6200, 50.8800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 6 | 2026-06-03 13:00 | 51.6000 | 50.4840 | 2026-06-03 15:00 | 51.8400 | STOCH_EXIT | 4 | 365.26 | 0.36% | 51.1600 | 53.8321 | 1.1160 | M30 bullish FVG [50.8700, 51.1600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 7 | 2026-06-10 11:00 | 49.5700 | 48.3455 | 2026-06-10 17:30 | 50.1200 | STOCH_EXIT | 13 | 1015.31 | 1.01% | 49.9100 | 52.0191 | 1.2245 | M30 bullish FVG [49.2600, 49.9100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 8 | 2026-06-11 11:00 | 49.7000 | 48.4708 | 2026-06-11 17:00 | 49.8900 | STOCH_EXIT | 12 | 286.02 | 0.28% | 49.9800 | 52.1584 | 1.2292 | M30 bullish FVG [49.4900, 49.9800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 9 | 2026-07-09 07:30 | 41.8400 | 40.0857 | 2026-07-10 09:00 | 41.4400 | STOCH_EXIT | 38 | -1073.19 | -1.06% | 41.1500 | 45.3486 | 1.7543 | M30 bullish FVG [40.6200, 41.1500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 10 | 2026-07-21 13:00 | 38.2900 | 36.3395 | 2026-07-21 16:00 | 40.0000 | STOCH_EXIT | 6 | 4394.35 | 4.36% | 39.1300 | 42.1910 | 1.9505 | M30 bullish FVG [38.3500, 39.1300], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 11 | 2026-07-22 07:30 | 39.7100 | 37.7017 | 2026-07-22 12:30 | 40.4300 | STOCH_EXIT | 10 | 1801.81 | 1.71% | 39.4200 | 43.7267 | 2.0083 | M30 bullish FVG [38.8900, 39.4200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 12 | 2026-08-26 07:30 | 40.7100 | 38.9237 | 2026-08-27 09:00 | 38.9237 | SL | 38 | -4802.60 | -4.48% | 41.0000 | 44.2825 | 1.7863 | M30 bullish FVG [40.1700, 41.0000], stochastic crossed below 20 | FVG stop 38.9237 |
| 13 | 2026-11-06 11:30 | 44.3200 | 42.9075 | 2026-11-06 19:30 | 44.6000 | STOCH_EXIT | 16 | 541.74 | 0.53% | 44.4600 | 47.1451 | 1.4125 | M30 bullish FVG [44.0000, 44.4600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 14 | 2026-11-09 23:30 | 47.1300 | 45.7041 | 2026-11-10 23:30 | 46.5600 | STOCH_EXIT | 35 | -1344.72 | -1.31% | 46.1300 | 49.9818 | 1.4259 | M30 bullish FVG [45.5100, 46.1300], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 99167.91
- Макс: 107198.73
- Финал: 101593.15

Полная кривая — в `*_equity.csv`.
