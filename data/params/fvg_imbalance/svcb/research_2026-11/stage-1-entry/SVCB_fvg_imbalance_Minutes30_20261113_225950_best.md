# Бэктест SVCB (Minutes30)

- Период: 2024-10-13 22:59 — 2026-10-13 22:59
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
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 21 |
| Выигрышных / проигрышных | 16 / 5 |
| Win rate | 76.19% |
| Gross profit / loss | 19979.45 / 7773.47 |
| Profit factor | 2.570 |
| Чистый PnL | 12205.98 (12.21%) |
| Стартовый / финальный капитал | 100000.00 / 112205.98 |
| Макс. просадка | 4052.00 (3.49%) |
| Средняя прибыль / убыток | 1248.72 / 1554.69 |
| Expectancy | 581.24 |
| Лучшая / худшая сделка | 3178.44 / -3424.88 |
| Exposure | 0.82% |
| CAGR | 5.93% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-10-30 12:00 | 12.6800 | 12.2527 | 2024-10-30 17:30 | 12.5900 | STOCH_EXIT | 11 | -800.55 | -0.81% | 12.4900 | 13.5346 | 0.4273 | M30 bullish FVG [12.3000, 12.4900], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 2 | 2024-11-18 22:00 | 14.0600 | 13.5646 | 2024-11-19 12:30 | 14.2100 | STOCH_EXIT | 15 | 951.06 | 0.97% | 13.7600 | 15.0508 | 0.4954 | M30 bullish FVG [13.4400, 13.7600], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 3 | 2024-11-27 16:30 | 12.4900 | 11.8754 | 2024-11-27 19:00 | 12.9000 | STOCH_EXIT | 5 | 3178.44 | 3.18% | 12.7250 | 13.7193 | 0.6146 | M30 bullish FVG [12.5350, 12.7250], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 4 | 2024-12-16 23:30 | 11.8750 | 11.4266 | 2024-12-17 10:30 | 12.0550 | STOCH_EXIT | 8 | 1445.10 | 1.41% | 12.0150 | 12.7719 | 0.4484 | M30 bullish FVG [11.8750, 12.0150], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 5 | 2024-12-18 09:30 | 12.5350 | 12.0325 | 2024-12-18 12:00 | 12.7950 | STOCH_EXIT | 5 | 2052.88 | 1.97% | 12.4300 | 13.5400 | 0.5025 | M30 bullish FVG [12.2200, 12.4300], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 6 | 2025-01-16 18:00 | 15.4150 | 14.8595 | 2025-01-17 10:30 | 15.4500 | STOCH_EXIT | 19 | 135.02 | 0.13% | 15.5900 | 16.5261 | 0.5555 | M30 bullish FVG [15.4000, 15.5900], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 7 | 2025-02-13 14:00 | 16.3600 | 15.8564 | 2025-02-13 16:00 | 16.5150 | STOCH_EXIT | 4 | 900.66 | 0.85% | 16.8900 | 17.3672 | 0.5036 | M30 bullish FVG [16.6000, 16.8900], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 8 | 2025-03-28 14:30 | 16.5300 | 15.8137 | 2025-03-28 18:00 | 16.8000 | STOCH_EXIT | 7 | 1646.68 | 1.53% | 17.0300 | 17.9627 | 0.7163 | M30 bullish FVG [16.7500, 17.0300], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 9 | 2025-06-11 08:30 | 15.4750 | 14.8197 | 2025-06-11 09:30 | 15.6800 | STOCH_EXIT | 2 | 1325.96 | 1.22% | 15.5600 | 16.7855 | 0.6553 | M30 bullish FVG [15.3550, 15.5600], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 10 | 2025-07-10 13:00 | 15.7250 | 15.2347 | 2025-07-10 15:30 | 15.8400 | STOCH_EXIT | 5 | 694.52 | 0.63% | 15.6650 | 16.7055 | 0.4903 | M30 bullish FVG [15.4350, 15.6650], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 11 | 2025-07-16 23:30 | 16.9450 | 16.4023 | 2025-07-17 10:30 | 17.4250 | STOCH_EXIT | 9 | 3008.30 | 2.73% | 16.9000 | 18.0304 | 0.5427 | M30 bullish FVG [16.6850, 16.9000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 12 | 2025-07-17 14:00 | 17.1150 | 16.5496 | 2025-07-18 07:00 | 16.9700 | STOCH_EXIT | 21 | -1069.48 | -0.95% | 17.4150 | 18.2457 | 0.5654 | M30 bullish FVG [17.1450, 17.4150], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 13 | 2025-07-31 21:30 | 16.9600 | 16.4044 | 2025-08-01 06:30 | 17.1200 | STOCH_EXIT | 5 | 943.54 | 0.84% | 16.9000 | 18.0712 | 0.5556 | M30 bullish FVG [16.6700, 16.9000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 14 | 2025-10-17 09:30 | 12.5550 | 12.0505 | 2025-10-17 13:00 | 12.6100 | STOCH_EXIT | 7 | 386.00 | 0.34% | 12.4200 | 13.5639 | 0.5045 | M30 bullish FVG [12.0500, 12.4200], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 15 | 2026-01-29 18:30 | 12.7050 | 12.4787 | 2026-01-29 23:00 | 12.8800 | STOCH_EXIT | 9 | 1459.87 | 1.28% | 12.9450 | 13.1575 | 0.2263 | M30 bullish FVG [12.8500, 12.9450], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 16 | 2026-02-04 21:00 | 13.3500 | 13.0792 | 2026-02-05 16:30 | 13.0792 | SL | 26 | -2470.53 | -2.13% | 13.3200 | 13.8915 | 0.2708 | M30 bullish FVG [13.0900, 13.3200], stochastic crossed below 20 | FVG stop 13.0792 |
| 17 | 2026-05-13 07:30 | 12.4250 | 12.1711 | 2026-05-13 15:30 | 12.4450 | STOCH_EXIT | 16 | 68.84 | 0.06% | 12.3500 | 12.9327 | 0.2539 | M30 bullish FVG [12.2750, 12.3500], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 18 | 2026-06-23 23:30 | 10.8200 | 10.4659 | 2026-06-24 07:30 | 10.9750 | STOCH_EXIT | 3 | 1513.08 | 1.33% | 10.6000 | 11.5281 | 0.3541 | M30 bullish FVG [10.4650, 10.6000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 19 | 2026-07-03 19:30 | 10.7450 | 10.3464 | 2026-07-04 02:00 | 10.7550 | STOCH_EXIT | 9 | -8.02 | -0.01% | 10.8600 | 11.5423 | 0.3986 | M30 bullish FVG [10.7300, 10.8600], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 20 | 2026-07-08 07:00 | 10.4800 | 10.0557 | 2026-07-08 07:30 | 10.5150 | STOCH_EXIT | 1 | 269.53 | 0.23% | 10.3850 | 11.3287 | 0.4243 | M30 bullish FVG [10.2500, 10.3850], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 21 | 2026-10-07 11:00 | 10.1100 | 9.8195 | 2026-10-08 09:00 | 9.8195 | SL | 31 | -3424.88 | -2.97% | 10.0100 | 10.6909 | 0.2905 | M30 bullish FVG [9.8000, 10.0100], stochastic crossed below 20 | FVG stop 9.8195 |

## Движение капитала

- Старт: 100000.00
- Мин: 98312.55
- Макс: 116257.97
- Финал: 112205.98

Полная кривая — в `*_equity.csv`.
