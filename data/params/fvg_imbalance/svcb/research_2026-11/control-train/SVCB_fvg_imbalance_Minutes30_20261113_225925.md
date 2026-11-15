# Бэктест SVCB (Minutes30)

- Период: 2024-10-13 22:59 — 2026-10-13 22:59
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
| Всего сделок | 29 |
| Выигрышных / проигрышных | 14 / 15 |
| Win rate | 48.28% |
| Gross profit / loss | 18722.29 / 19750.84 |
| Profit factor | 0.948 |
| Чистый PnL | -1028.55 (-1.03%) |
| Стартовый / финальный капитал | 100000.00 / 98971.45 |
| Макс. просадка | 10139.83 (9.34%) |
| Средняя прибыль / убыток | 1337.31 / 1316.72 |
| Expectancy | -35.47 |
| Лучшая / худшая сделка | 4600.89 / -3558.30 |
| Exposure | 2.09% |
| CAGR | -0.52% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-10-25 12:00 | 12.8400 | 12.4826 | 2024-10-25 16:30 | 12.4826 | SL | 9 | -2849.23 | -2.88% | 12.8950 | 13.5547 | 0.3574 | M30 bullish FVG [12.7900, 12.8950], stochastic crossed below 20 | FVG stop 12.4826 |
| 2 | 2024-10-30 12:30 | 12.6400 | 12.2127 | 2024-10-30 19:00 | 12.6100 | STOCH_EXIT | 13 | -323.95 | -0.34% | 12.4900 | 13.4946 | 0.4273 | M30 bullish FVG [12.3000, 12.4900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 3 | 2024-11-15 10:00 | 12.9400 | 12.4638 | 2024-11-15 12:30 | 13.5750 | STOCH_EXIT | 5 | 4600.89 | 4.80% | 13.1100 | 13.8924 | 0.4762 | M30 bullish FVG [12.9700, 13.1100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 4 | 2024-11-28 10:30 | 12.8800 | 12.2500 | 2024-11-28 18:30 | 12.8900 | STOCH_EXIT | 16 | -22.50 | -0.02% | 12.7250 | 14.1400 | 0.6300 | M30 bullish FVG [12.5350, 12.7250], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 5 | 2024-12-16 23:30 | 11.8750 | 11.4266 | 2024-12-17 10:30 | 12.0550 | STOCH_EXIT | 8 | 1428.30 | 1.41% | 12.0150 | 12.7719 | 0.4484 | M30 bullish FVG [11.8750, 12.0150], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 6 | 2025-01-16 18:30 | 15.3000 | 14.7445 | 2025-01-17 10:30 | 15.4500 | STOCH_EXIT | 18 | 901.99 | 0.88% | 15.5900 | 16.4111 | 0.5555 | M30 bullish FVG [15.4000, 15.5900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 7 | 2025-02-13 14:00 | 16.3600 | 15.8564 | 2025-02-13 17:00 | 16.5950 | STOCH_EXIT | 6 | 1376.69 | 1.34% | 16.8900 | 17.3672 | 0.5036 | M30 bullish FVG [16.6000, 16.8900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 8 | 2025-03-05 22:30 | 17.5250 | 16.8343 | 2025-03-06 07:00 | 17.8400 | STOCH_EXIT | 4 | 1754.17 | 1.70% | 17.6800 | 18.9065 | 0.6907 | M30 bullish FVG [17.4800, 17.6800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 9 | 2025-03-28 14:30 | 16.5300 | 15.8137 | 2025-03-29 04:00 | 16.7500 | STOCH_EXIT | 22 | 1301.50 | 1.23% | 17.0300 | 17.9627 | 0.7163 | M30 bullish FVG [16.7500, 17.0300], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 10 | 2025-05-15 11:30 | 15.7300 | 15.1224 | 2025-05-16 10:00 | 15.5400 | STOCH_EXIT | 32 | -1398.32 | -1.31% | 15.9100 | 16.9451 | 0.6076 | M30 bullish FVG [15.7400, 15.9100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 11 | 2025-06-20 07:00 | 16.0100 | 15.4382 | 2025-06-20 23:00 | 15.8050 | STOCH_EXIT | 32 | -1457.99 | -1.38% | 15.8600 | 17.1536 | 0.5718 | M30 bullish FVG [15.7150, 15.8600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 12 | 2025-07-10 13:00 | 15.7250 | 15.2347 | 2025-07-10 16:00 | 15.9050 | STOCH_EXIT | 6 | 1083.62 | 1.04% | 15.6650 | 16.7055 | 0.4903 | M30 bullish FVG [15.4350, 15.6650], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 13 | 2025-07-17 14:00 | 17.1150 | 16.5496 | 2025-07-18 07:00 | 16.9700 | STOCH_EXIT | 21 | -1004.66 | -0.95% | 17.4150 | 18.2457 | 0.5654 | M30 bullish FVG [17.1450, 17.4150], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 14 | 2025-08-01 10:30 | 16.9950 | 16.4284 | 2025-08-02 02:00 | 16.4284 | SL | 27 | -3558.30 | -3.43% | 16.9000 | 18.1282 | 0.5666 | M30 bullish FVG [16.6700, 16.9000], stochastic crossed below 20 | FVG stop 16.4284 |
| 15 | 2025-10-08 12:00 | 12.4600 | 11.9844 | 2025-10-09 06:30 | 12.2800 | STOCH_EXIT | 24 | -1558.20 | -1.54% | 12.6050 | 13.4113 | 0.4756 | M30 bullish FVG [12.4650, 12.6050], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 16 | 2025-10-10 10:30 | 12.3150 | 11.8110 | 2025-10-11 09:00 | 12.1200 | STOCH_EXIT | 41 | -1678.46 | -1.68% | 12.2750 | 13.3230 | 0.5040 | M30 bullish FVG [12.1400, 12.2750], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 17 | 2025-10-17 10:00 | 12.4700 | 11.9655 | 2025-10-17 14:00 | 12.7450 | STOCH_EXIT | 8 | 2072.90 | 2.10% | 12.4200 | 13.4789 | 0.5045 | M30 bullish FVG [12.0500, 12.4200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 18 | 2026-01-29 18:30 | 12.7050 | 12.4787 | 2026-01-30 07:00 | 13.0550 | STOCH_EXIT | 12 | 2663.25 | 2.65% | 12.9450 | 13.1575 | 0.2263 | M30 bullish FVG [12.8500, 12.9450], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 19 | 2026-01-30 11:30 | 12.7600 | 12.5213 | 2026-01-31 06:30 | 12.6700 | STOCH_EXIT | 33 | -821.72 | -0.80% | 12.9450 | 13.2373 | 0.2387 | M30 bullish FVG [12.8500, 12.9450], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 20 | 2026-02-04 22:30 | 13.2750 | 13.0042 | 2026-02-05 20:30 | 13.2250 | STOCH_EXIT | 31 | -487.02 | -0.48% | 13.3200 | 13.8165 | 0.2708 | M30 bullish FVG [13.0900, 13.3200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 21 | 2026-02-06 12:00 | 13.1700 | 12.8687 | 2026-02-07 03:00 | 13.1000 | STOCH_EXIT | 25 | -640.14 | -0.63% | 13.3250 | 13.7725 | 0.3013 | M30 bullish FVG [13.2350, 13.3250], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 22 | 2026-04-02 08:30 | 12.7550 | 12.4924 | 2026-04-02 16:00 | 12.7850 | STOCH_EXIT | 15 | 136.12 | 0.14% | 12.6900 | 13.2801 | 0.2626 | M30 bullish FVG [12.6150, 12.6900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 23 | 2026-04-03 14:30 | 12.8200 | 12.5633 | 2026-04-04 08:00 | 12.7750 | STOCH_EXIT | 29 | -456.60 | -0.45% | 12.8400 | 13.3333 | 0.2567 | M30 bullish FVG [12.7750, 12.8400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 24 | 2026-05-13 10:30 | 12.3650 | 12.1111 | 2026-05-13 15:30 | 12.4450 | STOCH_EXIT | 10 | 547.52 | 0.55% | 12.3500 | 12.8727 | 0.2539 | M30 bullish FVG [12.2750, 12.3500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 25 | 2026-05-13 22:30 | 12.3500 | 12.0961 | 2026-05-14 13:00 | 12.4250 | STOCH_EXIT | 16 | 513.42 | 0.51% | 12.4250 | 12.8577 | 0.2539 | M30 bullish FVG [12.3550, 12.4250], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 26 | 2026-05-28 17:30 | 11.8850 | 11.6181 | 2026-05-29 13:30 | 11.9100 | STOCH_EXIT | 27 | 111.37 | 0.11% | 12.0300 | 12.4189 | 0.2669 | M30 bullish FVG [11.9600, 12.0300], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 27 | 2026-07-03 20:00 | 10.7150 | 10.3164 | 2026-07-04 04:30 | 10.7500 | STOCH_EXIT | 12 | 230.54 | 0.23% | 10.8600 | 11.5123 | 0.3986 | M30 bullish FVG [10.7300, 10.8600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 28 | 2026-07-08 07:00 | 10.4800 | 10.0557 | 2026-07-08 18:30 | 10.4400 | STOCH_EXIT | 23 | -489.46 | -0.48% | 10.3850 | 11.3287 | 0.4243 | M30 bullish FVG [10.2500, 10.3850], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 29 | 2026-10-07 11:00 | 10.1100 | 9.8195 | 2026-10-08 09:00 | 9.8195 | SL | 31 | -3004.28 | -2.97% | 10.0100 | 10.6909 | 0.2905 | M30 bullish FVG [9.8000, 10.0100], stochastic crossed below 20 | FVG stop 9.8195 |

## Движение капитала

- Старт: 100000.00
- Мин: 95810.74
- Макс: 108541.47
- Финал: 98971.45

Полная кривая — в `*_equity.csv`.
