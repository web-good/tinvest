# Бэктест MRKC (Minutes30)

- Период: 2025-11-15 23:21 — 2026-11-15 23:21
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.25 |
| MaxWaitBars | 15 |
| StochPeriod | 9 |
| UseStochExit | 1 |
| StopATR | 1.7 |
| TakeProfitATR | 1 |
| HTFTrendEMA | 100 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 10 |
| Выигрышных / проигрышных | 5 / 5 |
| Win rate | 50.00% |
| Gross profit / loss | 3393.09 / 5586.45 |
| Profit factor | 0.607 |
| Чистый PnL | -2193.36 (-2.19%) |
| Стартовый / финальный капитал | 100000.00 / 97806.64 |
| Макс. просадка | 5900.98 (5.90%) |
| Средняя прибыль / убыток | 678.62 / 1117.29 |
| Expectancy | -219.34 |
| Лучшая / худшая сделка | 1064.71 / -3272.45 |
| Exposure | 0.86% |
| CAGR | -2.19% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2026-01-06 15:00 | 0.9326 | 0.8878 | 2026-01-06 21:30 | 0.9260 | STOCH_EXIT | 13 | -805.64 | -0.81% | 0.9292 | 0.9590 | 0.0264 | M30 bullish FVG [0.9138, 0.9292], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 2 | 2026-01-15 11:00 | 0.8918 | 0.8417 | 2026-01-15 18:00 | 0.8900 | STOCH_EXIT | 14 | -298.69 | -0.30% | 0.8992 | 0.9213 | 0.0295 | M30 bullish FVG [0.8904, 0.8992], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 3 | 2026-01-28 10:30 | 0.9100 | 0.8677 | 2026-01-28 16:30 | 0.9154 | STOCH_EXIT | 12 | 484.63 | 0.49% | 0.9124 | 0.9349 | 0.0249 | M30 bullish FVG [0.8988, 0.9124], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 4 | 2026-02-13 18:30 | 0.8600 | 0.8129 | 2026-02-13 23:30 | 0.8524 | STOCH_EXIT | 10 | -972.46 | -0.98% | 0.8568 | 0.8877 | 0.0277 | M30 bullish FVG [0.8450, 0.8568], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 5 | 2026-04-06 12:30 | 0.9268 | 0.8869 | 2026-04-06 16:00 | 0.9336 | STOCH_EXIT | 7 | 622.20 | 0.63% | 0.9398 | 0.9503 | 0.0235 | M30 bullish FVG [0.9310, 0.9398], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 6 | 2026-05-08 13:00 | 0.9152 | 0.8590 | 2026-05-08 21:00 | 0.8858 | STOCH_EXIT | 16 | -3272.45 | -3.31% | 0.9340 | 0.9483 | 0.0331 | M30 bullish FVG [0.8978, 0.9340], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 7 | 2026-09-21 12:30 | 0.6788 | 0.6405 | 2026-09-21 15:30 | 0.6854 | STOCH_EXIT | 6 | 828.51 | 0.87% | 0.6814 | 0.7014 | 0.0226 | M30 bullish FVG [0.6724, 0.6814], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 8 | 2026-09-25 09:00 | 0.7082 | 0.6700 | 2026-09-25 12:00 | 0.7118 | STOCH_EXIT | 6 | 393.04 | 0.41% | 0.7036 | 0.7307 | 0.0225 | M30 bullish FVG [0.6976, 0.7036], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 9 | 2026-10-08 23:30 | 0.6828 | 0.6484 | 2026-10-09 16:00 | 0.6818 | STOCH_EXIT | 20 | -237.20 | -0.25% | 0.6798 | 0.7030 | 0.0202 | M30 bullish FVG [0.6720, 0.6798], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 10 | 2026-10-19 18:00 | 0.7296 | 0.6964 | 2026-10-19 21:30 | 0.7384 | STOCH_EXIT | 7 | 1064.71 | 1.10% | 0.7380 | 0.7491 | 0.0195 | M30 bullish FVG [0.7326, 0.7380], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 94099.02
- Макс: 100000.00
- Финал: 97806.64

Полная кривая — в `*_equity.csv`.
