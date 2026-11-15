# Бэктест SVCB (Minutes30)

- Период: 2024-10-13 23:06 — 2026-10-13 23:06
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
| StopATR | 1.5 |
| TakeProfitATR | 0 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 26 |
| Выигрышных / проигрышных | 18 / 8 |
| Win rate | 69.23% |
| Gross profit / loss | 24748.53 / 7549.84 |
| Profit factor | 3.278 |
| Чистый PnL | 17198.69 (17.20%) |
| Стартовый / финальный капитал | 100000.00 / 117198.69 |
| Макс. просадка | 4446.33 (3.66%) |
| Средняя прибыль / убыток | 1374.92 / 943.73 |
| Expectancy | 661.49 |
| Лучшая / худшая сделка | 4787.42 / -2326.51 |
| Exposure | 1.10% |
| CAGR | 8.26% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-11-15 10:00 | 12.9400 | 12.2257 | 2024-11-15 12:30 | 13.5750 | STOCH_EXIT | 5 | 4787.42 | 4.80% | 13.1100 | 0.0000 | 0.4762 | M30 bullish FVG [12.9700, 13.1100], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 2 | 2024-11-27 16:30 | 12.4900 | 11.5681 | 2024-11-27 19:00 | 12.9000 | STOCH_EXIT | 5 | 3297.63 | 3.18% | 12.7250 | 0.0000 | 0.6146 | M30 bullish FVG [12.5350, 12.7250], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 3 | 2024-12-16 23:30 | 11.8750 | 11.2024 | 2024-12-17 10:30 | 12.0550 | STOCH_EXIT | 8 | 1512.32 | 1.41% | 12.0150 | 0.0000 | 0.4484 | M30 bullish FVG [11.8750, 12.0150], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 4 | 2024-12-18 09:00 | 12.5350 | 11.7812 | 2024-12-18 12:00 | 12.7950 | STOCH_EXIT | 6 | 2151.81 | 1.97% | 12.4300 | 0.0000 | 0.5025 | M30 bullish FVG [12.2200, 12.4300], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 5 | 2025-01-16 18:00 | 15.4150 | 14.5817 | 2025-01-17 10:30 | 15.4500 | STOCH_EXIT | 19 | 140.89 | 0.13% | 15.5900 | 0.0000 | 0.5555 | M30 bullish FVG [15.4000, 15.5900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 6 | 2025-02-13 14:00 | 16.3600 | 15.6046 | 2025-02-13 16:00 | 16.5150 | STOCH_EXIT | 4 | 942.23 | 0.85% | 16.8900 | 0.0000 | 0.5036 | M30 bullish FVG [16.6000, 16.8900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 7 | 2025-03-28 14:30 | 16.5300 | 15.4555 | 2025-03-28 18:00 | 16.8000 | STOCH_EXIT | 7 | 1722.68 | 1.53% | 17.0300 | 0.0000 | 0.7163 | M30 bullish FVG [16.7500, 17.0300], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 8 | 2025-05-06 15:30 | 14.6550 | 13.6371 | 2025-05-06 18:00 | 14.8050 | STOCH_EXIT | 5 | 1055.11 | 0.92% | 14.6900 | 0.0000 | 0.6786 | M30 bullish FVG [14.5000, 14.6900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 9 | 2025-05-15 11:30 | 15.7300 | 14.8186 | 2025-05-15 16:30 | 15.7050 | STOCH_EXIT | 10 | -297.24 | -0.26% | 15.9100 | 0.0000 | 0.6076 | M30 bullish FVG [15.7400, 15.9100], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 10 | 2025-06-11 08:30 | 15.4750 | 14.4921 | 2025-06-11 09:30 | 15.6800 | STOCH_EXIT | 2 | 1401.73 | 1.22% | 15.5600 | 0.0000 | 0.6553 | M30 bullish FVG [15.3550, 15.5600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 11 | 2025-06-20 07:00 | 16.0100 | 15.1523 | 2025-06-20 09:00 | 16.0700 | STOCH_EXIT | 4 | 316.51 | 0.27% | 15.8600 | 0.0000 | 0.5718 | M30 bullish FVG [15.7150, 15.8600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 12 | 2025-06-30 18:00 | 16.3500 | 15.5958 | 2025-07-01 06:30 | 16.4600 | STOCH_EXIT | 12 | 664.52 | 0.57% | 16.4700 | 0.0000 | 0.5028 | M30 bullish FVG [16.3000, 16.4700], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 13 | 2025-07-16 23:00 | 16.9750 | 16.1610 | 2025-07-17 10:30 | 17.4250 | STOCH_EXIT | 10 | 2986.32 | 2.55% | 16.9000 | 0.0000 | 0.5427 | M30 bullish FVG [16.6850, 16.9000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 14 | 2025-07-17 14:00 | 17.1150 | 16.2670 | 2025-07-18 07:00 | 16.9700 | STOCH_EXIT | 21 | -1134.30 | -0.95% | 17.4150 | 0.0000 | 0.5654 | M30 bullish FVG [17.1450, 17.4150], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 15 | 2025-07-31 21:30 | 16.9600 | 16.1266 | 2025-08-01 06:30 | 17.1200 | STOCH_EXIT | 5 | 1000.72 | 0.84% | 16.9000 | 0.0000 | 0.5556 | M30 bullish FVG [16.6700, 16.9000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 16 | 2025-10-08 12:00 | 12.4600 | 11.7466 | 2025-10-08 20:00 | 12.2300 | STOCH_EXIT | 16 | -2326.51 | -1.94% | 12.6050 | 0.0000 | 0.4756 | M30 bullish FVG [12.4650, 12.6050], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 17 | 2025-10-09 19:00 | 12.2900 | 11.5595 | 2025-10-09 21:30 | 12.4200 | STOCH_EXIT | 5 | 1129.39 | 0.96% | 12.2750 | 0.0000 | 0.4870 | M30 bullish FVG [12.1400, 12.2750], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 18 | 2025-10-17 09:30 | 12.5550 | 11.7983 | 2025-10-17 12:30 | 12.5900 | STOCH_EXIT | 6 | 213.06 | 0.18% | 12.4200 | 0.0000 | 0.5045 | M30 bullish FVG [12.0500, 12.4200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 19 | 2026-01-29 18:30 | 12.7050 | 12.3656 | 2026-01-29 22:30 | 12.8250 | STOCH_EXIT | 8 | 1008.01 | 0.84% | 12.9450 | 0.0000 | 0.2263 | M30 bullish FVG [12.8500, 12.9450], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 20 | 2026-02-04 19:30 | 13.3800 | 12.9739 | 2026-02-05 17:30 | 13.2250 | STOCH_EXIT | 31 | -1514.72 | -1.26% | 13.3200 | 0.0000 | 0.2708 | M30 bullish FVG [13.0900, 13.3200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 21 | 2026-02-06 12:00 | 13.1700 | 12.7181 | 2026-02-07 03:00 | 13.1000 | STOCH_EXIT | 25 | -748.21 | -0.63% | 13.3250 | 0.0000 | 0.3013 | M30 bullish FVG [13.2350, 13.3250], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 22 | 2026-04-01 18:30 | 12.7350 | 12.3530 | 2026-04-01 21:30 | 12.7750 | STOCH_EXIT | 6 | 250.65 | 0.21% | 12.6900 | 0.0000 | 0.2547 | M30 bullish FVG [12.6150, 12.6900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 23 | 2026-04-03 12:30 | 12.8350 | 12.4500 | 2026-04-04 08:00 | 12.7750 | STOCH_EXIT | 33 | -669.81 | -0.57% | 12.8400 | 0.0000 | 0.2567 | M30 bullish FVG [12.7750, 12.8400], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 24 | 2026-05-13 22:30 | 12.3500 | 11.9692 | 2026-05-14 09:00 | 12.3800 | STOCH_EXIT | 8 | 167.53 | 0.14% | 12.4250 | 0.0000 | 0.2539 | M30 bullish FVG [12.3550, 12.4250], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 25 | 2026-05-28 17:00 | 11.9100 | 11.5096 | 2026-05-29 11:30 | 11.8800 | STOCH_EXIT | 24 | -414.76 | -0.35% | 12.0300 | 0.0000 | 0.2669 | M30 bullish FVG [11.9600, 12.0300], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 26 | 2026-07-03 19:00 | 10.7750 | 10.1771 | 2026-07-03 23:30 | 10.7450 | STOCH_EXIT | 9 | -444.28 | -0.38% | 10.8600 | 0.0000 | 0.3986 | M30 bullish FVG [10.7300, 10.8600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 100000.00
- Макс: 121322.08
- Финал: 117198.69

Полная кривая — в `*_equity.csv`.
