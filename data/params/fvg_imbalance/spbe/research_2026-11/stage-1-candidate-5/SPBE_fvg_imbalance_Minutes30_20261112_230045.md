# Бэктест SPBE (Minutes30)

- Период: 2023-11-12 23:00 — 2025-11-12 23:00
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.25 |
| MaxWaitBars | 30 |
| StochPeriod | 11 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 2 |
| HTFTrendEMA | 50 |
| RequireEntryAboveFVGLower | 1 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 20 |
| Выигрышных / проигрышных | 16 / 4 |
| Win rate | 80.00% |
| Gross profit / loss | 37899.38 / 16399.50 |
| Profit factor | 2.311 |
| Чистый PnL | 21499.88 (21.50%) |
| Стартовый / финальный капитал | 100000.00 / 121499.88 |
| Макс. просадка | 14726.01 (13.75%) |
| Средняя прибыль / убыток | 2368.71 / 4099.88 |
| Expectancy | 1074.99 |
| Лучшая / худшая сделка | 6216.21 / -8048.06 |
| Exposure | 1.41% |
| CAGR | 10.21% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-01-11 14:00 | 96.7000 | 89.5790 | 2024-01-11 17:30 | 101.5000 | STOCH_EXIT | 7 | 4856.03 | 4.86% | 102.7000 | 110.9421 | 7.1210 | M30 bullish FVG [96.7000, 102.7000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 2 | 2024-04-09 16:30 | 93.2000 | 89.8989 | 2024-04-10 14:00 | 92.9000 | STOCH_EXIT | 23 | -441.79 | -0.42% | 93.5000 | 99.8022 | 3.3011 | M30 bullish FVG [92.5000, 93.5000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 3 | 2024-04-16 11:30 | 115.6000 | 110.4670 | 2024-04-17 10:00 | 110.4670 | SL | 25 | -4731.94 | -4.54% | 118.3000 | 125.8660 | 5.1330 | M30 bullish FVG [115.5000, 118.3000], stochastic crossed below 20 | FVG stop 110.4670 |
| 4 | 2024-05-27 17:30 | 116.6000 | 110.5183 | 2024-05-27 23:30 | 117.6000 | STOCH_EXIT | 12 | 754.00 | 0.76% | 119.3000 | 128.7634 | 6.0817 | M30 bullish FVG [116.3000, 119.3000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 5 | 2024-07-29 15:00 | 135.8000 | 125.0399 | 2024-07-29 18:30 | 125.0399 | SL | 7 | -8048.06 | -8.02% | 134.5000 | 157.3201 | 10.7601 | M30 bullish FVG [130.4000, 134.5000], stochastic crossed below 20 | FVG stop 125.0399 |
| 6 | 2024-07-30 21:00 | 132.6000 | 121.0228 | 2024-07-31 11:30 | 135.3000 | STOCH_EXIT | 9 | 1785.97 | 1.93% | 131.5000 | 155.7544 | 11.5772 | M30 bullish FVG [128.7000, 131.5000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 7 | 2024-09-23 16:30 | 121.0000 | 114.3358 | 2024-09-24 09:00 | 122.8000 | STOCH_EXIT | 19 | 1303.88 | 1.39% | 121.2000 | 134.3285 | 6.6642 | M30 bullish FVG [119.5000, 121.2000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 8 | 2024-09-26 08:00 | 125.2000 | 118.4673 | 2024-09-26 08:00 | 127.4000 | STOCH_EXIT | 0 | 1580.16 | 1.66% | 125.0000 | 138.6653 | 6.7327 | M30 bullish FVG [123.0000, 125.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 9 | 2024-09-26 17:00 | 128.9000 | 122.1673 | 2024-09-27 08:00 | 130.6000 | STOCH_EXIT | 16 | 1180.83 | 1.22% | 127.1000 | 142.3653 | 6.7327 | M30 bullish FVG [125.3000, 127.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 10 | 2024-10-02 17:30 | 129.8000 | 122.8546 | 2024-10-02 22:30 | 130.3000 | STOCH_EXIT | 10 | 279.68 | 0.28% | 131.6000 | 143.6908 | 6.9454 | M30 bullish FVG [129.6000, 131.6000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 11 | 2024-10-04 10:30 | 138.2000 | 130.6705 | 2024-10-07 07:30 | 138.9000 | STOCH_EXIT | 28 | 399.75 | 0.41% | 138.4000 | 153.2589 | 7.5295 | M30 bullish FVG [136.3000, 138.4000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 12 | 2024-10-10 20:00 | 141.2000 | 133.2041 | 2024-10-11 07:00 | 142.0000 | STOCH_EXIT | 8 | 460.88 | 0.47% | 141.4000 | 157.1918 | 7.9959 | M30 bullish FVG [138.6000, 141.4000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 13 | 2024-10-17 17:00 | 128.9000 | 121.2999 | 2024-10-21 08:00 | 124.9000 | STOCH_EXIT | 50 | -3177.71 | -3.20% | 129.7000 | 144.1003 | 7.6001 | M30 bullish FVG [127.7000, 129.7000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 14 | 2024-11-07 16:00 | 130.7000 | 123.0270 | 2024-11-07 18:00 | 132.5000 | STOCH_EXIT | 4 | 1226.27 | 1.28% | 129.6000 | 146.0461 | 7.6730 | M30 bullish FVG [127.5000, 129.6000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 15 | 2025-01-29 10:00 | 144.2000 | 137.5993 | 2025-01-29 10:00 | 146.4000 | STOCH_EXIT | 0 | 1386.92 | 1.42% | 141.9000 | 157.4015 | 6.6007 | M30 bullish FVG [140.0000, 141.9000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 16 | 2025-02-13 10:00 | 168.6000 | 160.7328 | 2025-02-13 16:00 | 179.4000 | STOCH_EXIT | 12 | 6216.21 | 6.30% | 162.2000 | 184.3344 | 7.8672 | M30 bullish FVG [156.6000, 162.2000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 17 | 2025-02-14 17:00 | 206.0000 | 197.2662 | 2025-02-14 19:00 | 217.8000 | STOCH_EXIT | 4 | 5898.34 | 5.62% | 206.4000 | 223.4676 | 8.7338 | M30 bullish FVG [192.7000, 206.4000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 18 | 2025-03-04 15:30 | 301.0000 | 273.9597 | 2025-03-04 18:30 | 314.9000 | STOCH_EXIT | 6 | 5001.87 | 4.51% | 291.6000 | 355.0807 | 27.0403 | M30 bullish FVG [281.8000, 291.6000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 19 | 2025-04-23 09:00 | 274.1000 | 249.3945 | 2025-04-23 14:00 | 278.0000 | STOCH_EXIT | 10 | 1529.31 | 1.32% | 285.2000 | 323.5110 | 24.7055 | M30 bullish FVG [274.0000, 285.2000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 20 | 2025-08-07 19:00 | 245.3000 | 234.5504 | 2025-08-07 23:30 | 254.0000 | STOCH_EXIT | 9 | 4039.27 | 3.44% | 246.6000 | 266.7993 | 10.7496 | M30 bullish FVG [236.9000, 246.6000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 92342.09
- Макс: 121499.88
- Финал: 121499.88

Полная кривая — в `*_equity.csv`.
