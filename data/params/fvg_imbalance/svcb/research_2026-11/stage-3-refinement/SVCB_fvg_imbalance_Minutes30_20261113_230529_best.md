# Бэктест SVCB (Minutes30)

- Период: 2024-10-13 23:05 — 2026-10-13 23:05
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.3 |
| MaxWaitBars | 30 |
| StochPeriod | 9 |
| UseStochExit | 1 |
| StopATR | 1.5 |
| TakeProfitATR | 0 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 22 |
| Выигрышных / проигрышных | 17 / 5 |
| Win rate | 77.27% |
| Gross profit / loss | 19311.38 / 7132.40 |
| Profit factor | 2.708 |
| Чистый PnL | 12178.97 (12.18%) |
| Стартовый / финальный капитал | 100000.00 / 112178.97 |
| Макс. просадка | 5163.16 (4.42%) |
| Средняя прибыль / убыток | 1135.96 / 1426.48 |
| Expectancy | 553.59 |
| Лучшая / худшая сделка | 3178.44 / -3362.63 |
| Exposure | 0.99% |
| CAGR | 5.91% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-10-30 12:00 | 12.6800 | 12.0391 | 2024-10-30 17:30 | 12.5900 | STOCH_EXIT | 11 | -800.55 | -0.81% | 12.4900 | 0.0000 | 0.4273 | M30 bullish FVG [12.3000, 12.4900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 2 | 2024-11-18 22:00 | 14.0600 | 13.3169 | 2024-11-19 12:30 | 14.2100 | STOCH_EXIT | 15 | 951.06 | 0.97% | 13.7600 | 0.0000 | 0.4954 | M30 bullish FVG [13.4400, 13.7600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 3 | 2024-11-27 16:30 | 12.4900 | 11.5681 | 2024-11-27 19:00 | 12.9000 | STOCH_EXIT | 5 | 3178.44 | 3.18% | 12.7250 | 0.0000 | 0.6146 | M30 bullish FVG [12.5350, 12.7250], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 4 | 2024-12-16 23:30 | 11.8750 | 11.2024 | 2024-12-17 10:30 | 12.0550 | STOCH_EXIT | 8 | 1445.10 | 1.41% | 12.0150 | 0.0000 | 0.4484 | M30 bullish FVG [11.8750, 12.0150], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 5 | 2024-12-18 09:00 | 12.5350 | 11.7812 | 2024-12-18 12:00 | 12.7950 | STOCH_EXIT | 6 | 2052.88 | 1.97% | 12.4300 | 0.0000 | 0.5025 | M30 bullish FVG [12.2200, 12.4300], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 6 | 2025-01-16 18:00 | 15.4150 | 14.5817 | 2025-01-17 10:30 | 15.4500 | STOCH_EXIT | 19 | 135.02 | 0.13% | 15.5900 | 0.0000 | 0.5555 | M30 bullish FVG [15.4000, 15.5900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 7 | 2025-02-13 14:00 | 16.3600 | 15.6046 | 2025-02-13 16:00 | 16.5150 | STOCH_EXIT | 4 | 900.66 | 0.85% | 16.8900 | 0.0000 | 0.5036 | M30 bullish FVG [16.6000, 16.8900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 8 | 2025-03-28 14:30 | 16.5300 | 15.4555 | 2025-03-28 18:00 | 16.8000 | STOCH_EXIT | 7 | 1646.68 | 1.53% | 17.0300 | 0.0000 | 0.7163 | M30 bullish FVG [16.7500, 17.0300], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 9 | 2025-06-11 08:30 | 15.4750 | 14.4921 | 2025-06-11 09:30 | 15.6800 | STOCH_EXIT | 2 | 1325.96 | 1.22% | 15.5600 | 0.0000 | 0.6553 | M30 bullish FVG [15.3550, 15.5600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 10 | 2025-06-30 18:00 | 16.3500 | 15.5958 | 2025-07-01 06:30 | 16.4600 | STOCH_EXIT | 12 | 627.09 | 0.57% | 16.4700 | 0.0000 | 0.5028 | M30 bullish FVG [16.3000, 16.4700], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 11 | 2025-07-10 13:00 | 15.7250 | 14.9896 | 2025-07-10 15:30 | 15.8400 | STOCH_EXIT | 5 | 694.52 | 0.63% | 15.6650 | 0.0000 | 0.4903 | M30 bullish FVG [15.4350, 15.6650], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 12 | 2025-07-16 23:00 | 16.9750 | 16.1610 | 2025-07-17 10:30 | 17.4250 | STOCH_EXIT | 10 | 2856.48 | 2.55% | 16.9000 | 0.0000 | 0.5427 | M30 bullish FVG [16.6850, 16.9000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 13 | 2025-07-17 14:00 | 17.1150 | 16.2670 | 2025-07-18 07:00 | 16.9700 | STOCH_EXIT | 21 | -1085.68 | -0.95% | 17.4150 | 0.0000 | 0.5654 | M30 bullish FVG [17.1450, 17.4150], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 14 | 2025-07-31 21:30 | 16.9600 | 16.1266 | 2025-08-01 06:30 | 17.1200 | STOCH_EXIT | 5 | 957.83 | 0.84% | 16.9000 | 0.0000 | 0.5556 | M30 bullish FVG [16.6700, 16.9000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 15 | 2025-10-17 09:30 | 12.5550 | 11.7983 | 2025-10-17 12:30 | 12.5900 | STOCH_EXIT | 6 | 204.09 | 0.18% | 12.4200 | 0.0000 | 0.5045 | M30 bullish FVG [12.0500, 12.4200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 16 | 2026-01-29 18:30 | 12.7050 | 12.3656 | 2026-01-29 22:30 | 12.8250 | STOCH_EXIT | 8 | 965.12 | 0.84% | 12.9450 | 0.0000 | 0.2263 | M30 bullish FVG [12.8500, 12.9450], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 17 | 2026-02-04 19:30 | 13.3800 | 12.9739 | 2026-02-05 17:30 | 13.2250 | STOCH_EXIT | 31 | -1447.40 | -1.26% | 13.3200 | 0.0000 | 0.2708 | M30 bullish FVG [13.0900, 13.3200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 18 | 2026-05-13 07:30 | 12.4250 | 12.0442 | 2026-05-13 16:00 | 12.4500 | STOCH_EXIT | 17 | 115.57 | 0.10% | 12.3500 | 0.0000 | 0.2539 | M30 bullish FVG [12.2750, 12.3500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 19 | 2026-06-23 22:30 | 10.8700 | 10.3389 | 2026-06-24 07:30 | 10.9750 | STOCH_EXIT | 5 | 987.81 | 0.87% | 10.6000 | 0.0000 | 0.3541 | M30 bullish FVG [10.4650, 10.6000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 20 | 2026-07-03 19:00 | 10.7750 | 10.1771 | 2026-07-03 23:30 | 10.7450 | STOCH_EXIT | 9 | -436.13 | -0.38% | 10.8600 | 0.0000 | 0.3986 | M30 bullish FVG [10.7300, 10.8600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 21 | 2026-07-08 07:00 | 10.4800 | 9.8435 | 2026-07-08 07:00 | 10.5150 | STOCH_EXIT | 0 | 267.08 | 0.23% | 10.3850 | 0.0000 | 0.4243 | M30 bullish FVG [10.2500, 10.3850], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 22 | 2026-10-07 11:00 | 10.1100 | 9.6743 | 2026-10-08 22:00 | 9.8250 | STOCH_EXIT | 57 | -3362.63 | -2.92% | 10.0100 | 0.0000 | 0.2905 | M30 bullish FVG [9.8000, 10.0100], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 98312.55
- Макс: 116771.14
- Финал: 112178.97

Полная кривая — в `*_equity.csv`.
