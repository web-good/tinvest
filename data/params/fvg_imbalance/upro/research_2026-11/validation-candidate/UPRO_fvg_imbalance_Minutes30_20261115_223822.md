# Бэктест UPRO (Minutes30)

- Период: 2025-11-15 22:38 — 2026-11-15 22:38
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.3 |
| MaxWaitBars | 30 |
| StochPeriod | 10 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 2 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 1 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 16 |
| Выигрышных / проигрышных | 9 / 7 |
| Win rate | 56.25% |
| Gross profit / loss | 4060.15 / 8071.54 |
| Profit factor | 0.503 |
| Чистый PnL | -4011.39 (-4.01%) |
| Стартовый / финальный капитал | 100000.00 / 95988.61 |
| Макс. просадка | 7454.16 (7.37%) |
| Средняя прибыль / убыток | 451.13 / 1153.08 |
| Expectancy | -250.71 |
| Лучшая / худшая сделка | 893.02 / -6435.65 |
| Exposure | 1.67% |
| CAGR | -4.01% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2025-11-21 10:30 | 1.4570 | 1.4098 | 2025-11-21 17:30 | 1.4590 | STOCH_EXIT | 14 | 36.86 | 0.04% | 1.4210 | 1.5513 | 0.0472 | M30 bullish FVG [1.4060, 1.4210], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 2 | 2025-12-26 21:30 | 1.4900 | 1.4516 | 2025-12-26 22:30 | 1.4960 | STOCH_EXIT | 2 | 301.97 | 0.30% | 1.4860 | 1.5667 | 0.0384 | M30 bullish FVG [1.4690, 1.4860], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 3 | 2026-01-23 07:00 | 1.5590 | 1.5221 | 2026-01-23 13:30 | 1.5720 | STOCH_EXIT | 13 | 731.81 | 0.73% | 1.5650 | 1.6328 | 0.0369 | M30 bullish FVG [1.5220, 1.5650], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 4 | 2026-01-28 08:30 | 1.6330 | 1.5905 | 2026-01-28 14:30 | 1.6320 | STOCH_EXIT | 12 | -160.58 | -0.16% | 1.6030 | 1.7181 | 0.0425 | M30 bullish FVG [1.5710, 1.6030], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 5 | 2026-02-17 23:00 | 1.6385 | 1.5941 | 2026-02-19 07:00 | 1.6350 | STOCH_EXIT | 38 | -313.34 | -0.31% | 1.6405 | 1.7273 | 0.0444 | M30 bullish FVG [1.6225, 1.6405], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 6 | 2026-02-19 13:30 | 1.6395 | 1.5927 | 2026-02-19 18:00 | 1.6395 | STOCH_EXIT | 9 | -100.01 | -0.10% | 1.6340 | 1.7330 | 0.0468 | M30 bullish FVG [1.6185, 1.6340], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 7 | 2026-03-25 17:00 | 1.4925 | 1.4572 | 2026-03-26 06:30 | 1.4900 | STOCH_EXIT | 14 | -267.41 | -0.27% | 1.4970 | 1.5631 | 0.0353 | M30 bullish FVG [1.4830, 1.4970], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 8 | 2026-04-10 10:30 | 1.4455 | 1.4112 | 2026-04-10 13:00 | 1.4490 | STOCH_EXIT | 5 | 141.64 | 0.14% | 1.4515 | 1.5140 | 0.0343 | M30 bullish FVG [1.4315, 1.4515], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 9 | 2026-04-21 22:30 | 1.4280 | 1.3960 | 2026-04-22 10:00 | 1.4335 | STOCH_EXIT | 9 | 284.85 | 0.28% | 1.4315 | 1.4921 | 0.0320 | M30 bullish FVG [1.4155, 1.4315], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 10 | 2026-05-07 20:30 | 1.3775 | 1.3495 | 2026-05-08 15:00 | 1.3700 | STOCH_EXIT | 23 | -647.78 | -0.64% | 1.3805 | 1.4334 | 0.0280 | M30 bullish FVG [1.3695, 1.3805], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 11 | 2026-05-13 22:30 | 1.4210 | 1.3909 | 2026-05-14 07:00 | 1.4265 | STOCH_EXIT | 3 | 285.34 | 0.29% | 1.4260 | 1.4812 | 0.0301 | M30 bullish FVG [1.4120, 1.4260], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 12 | 2026-07-27 19:00 | 1.0615 | 1.0046 | 2026-07-28 15:00 | 1.0610 | STOCH_EXIT | 27 | -146.76 | -0.15% | 1.0135 | 1.1753 | 0.0569 | M30 bullish FVG [0.9575, 1.0135], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 13 | 2026-08-26 07:00 | 0.9735 | 0.9113 | 2026-08-26 20:00 | 0.9113 | SL | 26 | -6435.65 | -6.48% | 0.9815 | 1.0978 | 0.0622 | M30 bullish FVG [0.9465, 0.9815], stochastic crossed below 20 | FVG stop 0.9113 |
| 14 | 2026-08-28 19:30 | 0.9945 | 0.9333 | 2026-08-28 22:30 | 1.0050 | STOCH_EXIT | 6 | 893.02 | 0.95% | 1.0025 | 1.1169 | 0.0612 | M30 bullish FVG [0.9765, 1.0025], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 15 | 2026-10-13 17:00 | 1.1275 | 1.0878 | 2026-10-13 20:00 | 1.1380 | STOCH_EXIT | 6 | 777.48 | 0.83% | 1.1200 | 1.2069 | 0.0397 | M30 bullish FVG [1.1040, 1.1200], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 16 | 2026-10-30 22:30 | 1.2110 | 1.1707 | 2026-10-31 08:30 | 1.2200 | STOCH_EXIT | 9 | 607.19 | 0.64% | 1.2065 | 1.2916 | 0.0403 | M30 bullish FVG [1.1910, 1.2065], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 93710.92
- Макс: 101165.08
- Финал: 95988.61

Полная кривая — в `*_equity.csv`.
