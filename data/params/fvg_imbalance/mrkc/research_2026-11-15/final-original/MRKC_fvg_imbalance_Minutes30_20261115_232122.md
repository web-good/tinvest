# Бэктест MRKC (Minutes30)

- Период: 2025-11-15 23:21 — 2026-11-15 23:21
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
| Всего сделок | 20 |
| Выигрышных / проигрышных | 13 / 7 |
| Win rate | 65.00% |
| Gross profit / loss | 15546.05 / 16813.99 |
| Profit factor | 0.925 |
| Чистый PnL | -1267.94 (-1.27%) |
| Стартовый / финальный капитал | 100000.00 / 98732.06 |
| Макс. просадка | 10121.38 (9.50%) |
| Средняя прибыль / убыток | 1195.85 / 2402.00 |
| Expectancy | -63.40 |
| Лучшая / худшая сделка | 3450.46 / -4683.49 |
| Exposure | 2.00% |
| CAGR | -1.27% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2025-11-21 15:00 | 0.6636 | 0.6378 | 2025-11-21 19:30 | 0.6676 | STOCH_EXIT | 9 | 500.16 | 0.50% | 0.6678 | 0.7151 | 0.0258 | M30 bullish FVG [0.6572, 0.6678], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 2 | 2025-11-27 17:30 | 0.6944 | 0.6690 | 2025-11-27 21:30 | 0.7036 | STOCH_EXIT | 8 | 1224.14 | 1.22% | 0.7030 | 0.7451 | 0.0254 | M30 bullish FVG [0.6950, 0.7030], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 3 | 2025-12-26 18:30 | 0.8182 | 0.7964 | 2025-12-26 20:00 | 0.8216 | STOCH_EXIT | 3 | 319.93 | 0.32% | 0.8080 | 0.8619 | 0.0218 | M30 bullish FVG [0.8000, 0.8080], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 4 | 2026-01-06 16:00 | 0.9300 | 0.9036 | 2026-01-06 22:30 | 0.9300 | STOCH_EXIT | 13 | -101.37 | -0.10% | 0.9292 | 0.9827 | 0.0264 | M30 bullish FVG [0.9138, 0.9292], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 5 | 2026-01-15 15:00 | 0.8842 | 0.8547 | 2026-01-15 18:30 | 0.8928 | STOCH_EXIT | 7 | 886.82 | 0.87% | 0.8992 | 0.9431 | 0.0295 | M30 bullish FVG [0.8904, 0.8992], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 6 | 2026-01-28 12:00 | 0.9080 | 0.8831 | 2026-01-28 16:30 | 0.9154 | STOCH_EXIT | 9 | 733.18 | 0.71% | 0.9124 | 0.9578 | 0.0249 | M30 bullish FVG [0.8988, 0.9124], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 7 | 2026-02-13 19:30 | 0.8450 | 0.8173 | 2026-02-14 05:30 | 0.8508 | STOCH_EXIT | 13 | 604.16 | 0.59% | 0.8568 | 0.9004 | 0.0277 | M30 bullish FVG [0.8450, 0.8568], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 8 | 2026-02-25 21:30 | 0.8326 | 0.8098 | 2026-02-25 22:00 | 0.8370 | STOCH_EXIT | 1 | 445.65 | 0.43% | 0.8386 | 0.8783 | 0.0228 | M30 bullish FVG [0.8300, 0.8386], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 9 | 2026-04-06 14:30 | 0.9256 | 0.9021 | 2026-04-07 09:30 | 0.9440 | STOCH_EXIT | 24 | 1956.10 | 1.89% | 0.9398 | 0.9726 | 0.0235 | M30 bullish FVG [0.9310, 0.9398], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 10 | 2026-05-08 14:30 | 0.9218 | 0.8887 | 2026-05-08 16:00 | 0.8887 | SL | 3 | -3905.74 | -3.68% | 0.9340 | 0.9879 | 0.0331 | M30 bullish FVG [0.8978, 0.9340], stochastic crossed below 20 | FVG stop 0.8887 |
| 11 | 2026-05-14 11:30 | 0.9500 | 0.9109 | 2026-05-15 18:00 | 0.9109 | SL | 47 | -4320.87 | -4.21% | 0.9488 | 1.0282 | 0.0391 | M30 bullish FVG [0.9348, 0.9488], stochastic crossed below 20 | FVG stop 0.9109 |
| 12 | 2026-07-09 08:00 | 0.5214 | 0.4870 | 2026-07-09 15:30 | 0.5184 | STOCH_EXIT | 15 | -661.74 | -0.67% | 0.5236 | 0.5903 | 0.0344 | M30 bullish FVG [0.5150, 0.5236], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 13 | 2026-07-28 14:30 | 0.6364 | 0.5993 | 2026-07-28 23:30 | 0.6596 | STOCH_EXIT | 18 | 3450.46 | 3.54% | 0.6502 | 0.7107 | 0.0371 | M30 bullish FVG [0.6298, 0.6502], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 14 | 2026-08-12 17:30 | 0.6964 | 0.6648 | 2026-08-13 16:00 | 0.6648 | SL | 31 | -4683.49 | -4.64% | 0.6966 | 0.7596 | 0.0316 | M30 bullish FVG [0.6842, 0.6966], stochastic crossed below 20 | FVG stop 0.6648 |
| 15 | 2026-09-15 22:30 | 0.6450 | 0.6244 | 2026-09-16 07:00 | 0.6624 | STOCH_EXIT | 4 | 2495.20 | 2.60% | 0.6476 | 0.6862 | 0.0206 | M30 bullish FVG [0.6346, 0.6476], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 16 | 2026-09-21 14:30 | 0.6802 | 0.6576 | 2026-09-21 16:00 | 0.6874 | STOCH_EXIT | 3 | 944.85 | 0.96% | 0.6814 | 0.7253 | 0.0226 | M30 bullish FVG [0.6724, 0.6814], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 17 | 2026-09-25 09:00 | 0.7082 | 0.6857 | 2026-09-25 13:30 | 0.7152 | STOCH_EXIT | 9 | 880.36 | 0.89% | 0.7036 | 0.7531 | 0.0225 | M30 bullish FVG [0.6976, 0.7036], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 18 | 2026-10-09 09:30 | 0.6852 | 0.6646 | 2026-10-09 18:00 | 0.6844 | STOCH_EXIT | 17 | -216.78 | -0.22% | 0.6798 | 0.7263 | 0.0206 | M30 bullish FVG [0.6720, 0.6798], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 19 | 2026-10-19 18:00 | 0.7296 | 0.7101 | 2026-10-19 21:30 | 0.7384 | STOCH_EXIT | 7 | 1105.04 | 1.10% | 0.7380 | 0.7687 | 0.0195 | M30 bullish FVG [0.7326, 0.7380], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 20 | 2026-10-21 17:00 | 0.8106 | 0.7880 | 2026-10-22 08:30 | 0.7880 | SL | 18 | -2924.00 | -2.88% | 0.8312 | 0.8558 | 0.0226 | M30 bullish FVG [0.8134, 0.8312], stochastic crossed below 20 | FVG stop 0.7880 |

## Движение капитала

- Старт: 100000.00
- Мин: 96447.39
- Макс: 106568.78
- Финал: 98732.06

Полная кривая — в `*_equity.csv`.
