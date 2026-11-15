# Бэктест SVCB (Minutes30)

- Период: 2024-10-13 23:03 — 2026-10-13 23:03
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.25 |
| MaxWaitBars | 30 |
| StochPeriod | 10 |
| UseStochExit | 1 |
| StopATR | 1.5 |
| TakeProfitATR | 0 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 35 |
| Выигрышных / проигрышных | 23 / 12 |
| Win rate | 65.71% |
| Gross profit / loss | 27773.75 / 15268.77 |
| Profit factor | 1.819 |
| Чистый PnL | 12504.98 (12.50%) |
| Стартовый / финальный капитал | 100000.00 / 112504.98 |
| Макс. просадка | 7080.58 (5.95%) |
| Средняя прибыль / убыток | 1207.55 / 1272.40 |
| Expectancy | 357.29 |
| Лучшая / худшая сделка | 4600.89 / -3362.63 |
| Exposure | 1.75% |
| CAGR | 6.07% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-10-25 12:00 | 12.8400 | 12.3039 | 2024-10-25 21:00 | 12.5300 | STOCH_EXIT | 18 | -2484.67 | -2.51% | 12.8950 | 0.0000 | 0.3574 | M30 bullish FVG [12.7900, 12.8950], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 2 | 2024-10-30 12:00 | 12.6800 | 12.0391 | 2024-10-30 17:30 | 12.5900 | STOCH_EXIT | 11 | -780.03 | -0.81% | 12.4900 | 0.0000 | 0.4273 | M30 bullish FVG [12.3000, 12.4900], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 3 | 2024-11-15 10:00 | 12.9400 | 12.2257 | 2024-11-15 12:30 | 13.5750 | STOCH_EXIT | 5 | 4600.89 | 4.80% | 13.1100 | 0.0000 | 0.4762 | M30 bullish FVG [12.9700, 13.1100], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 4 | 2024-11-18 22:00 | 14.0600 | 13.3169 | 2024-11-19 12:30 | 14.2100 | STOCH_EXIT | 15 | 978.23 | 0.97% | 13.7600 | 0.0000 | 0.4954 | M30 bullish FVG [13.4400, 13.7600], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 5 | 2024-11-27 16:30 | 12.4900 | 11.5681 | 2024-11-27 19:00 | 12.9000 | STOCH_EXIT | 5 | 3218.17 | 3.18% | 12.7250 | 0.0000 | 0.6146 | M30 bullish FVG [12.5350, 12.7250], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 6 | 2024-12-16 23:30 | 11.8750 | 11.2024 | 2024-12-17 10:30 | 12.0550 | STOCH_EXIT | 8 | 1478.71 | 1.41% | 12.0150 | 0.0000 | 0.4484 | M30 bullish FVG [11.8750, 12.0150], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 7 | 2024-12-18 09:30 | 12.5350 | 11.7812 | 2024-12-18 12:00 | 12.7950 | STOCH_EXIT | 5 | 2102.35 | 1.97% | 12.4300 | 0.0000 | 0.5025 | M30 bullish FVG [12.2200, 12.4300], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 8 | 2025-01-16 18:00 | 15.4150 | 14.5817 | 2025-01-17 10:30 | 15.4500 | STOCH_EXIT | 19 | 136.97 | 0.13% | 15.5900 | 0.0000 | 0.5555 | M30 bullish FVG [15.4000, 15.5900], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 9 | 2025-02-13 14:00 | 16.3600 | 15.6046 | 2025-02-13 16:00 | 16.5150 | STOCH_EXIT | 4 | 914.51 | 0.85% | 16.8900 | 0.0000 | 0.5036 | M30 bullish FVG [16.6000, 16.8900], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 10 | 2025-03-05 21:30 | 17.6850 | 16.6489 | 2025-03-06 07:00 | 17.8400 | STOCH_EXIT | 6 | 850.87 | 0.78% | 17.6800 | 0.0000 | 0.6907 | M30 bullish FVG [17.4800, 17.6800], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 11 | 2025-03-28 14:30 | 16.5300 | 15.4555 | 2025-03-28 18:00 | 16.8000 | STOCH_EXIT | 7 | 1697.34 | 1.53% | 17.0300 | 0.0000 | 0.7163 | M30 bullish FVG [16.7500, 17.0300], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 12 | 2025-05-06 15:30 | 14.6550 | 13.6371 | 2025-05-06 18:00 | 14.8050 | STOCH_EXIT | 5 | 1028.05 | 0.92% | 14.6900 | 0.0000 | 0.6786 | M30 bullish FVG [14.5000, 14.6900], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 13 | 2025-05-15 11:30 | 15.7300 | 14.8186 | 2025-05-15 16:30 | 15.7050 | STOCH_EXIT | 10 | -293.17 | -0.26% | 15.9100 | 0.0000 | 0.6076 | M30 bullish FVG [15.7400, 15.9100], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 14 | 2025-06-11 08:30 | 15.4750 | 14.4921 | 2025-06-11 09:30 | 15.6800 | STOCH_EXIT | 2 | 1382.78 | 1.22% | 15.5600 | 0.0000 | 0.6553 | M30 bullish FVG [15.3550, 15.5600], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 15 | 2025-06-20 07:00 | 16.0100 | 15.1523 | 2025-06-20 09:00 | 16.0700 | STOCH_EXIT | 4 | 312.12 | 0.27% | 15.8600 | 0.0000 | 0.5718 | M30 bullish FVG [15.7150, 15.8600], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 16 | 2025-07-10 13:00 | 15.7250 | 14.9896 | 2025-07-10 15:30 | 15.8400 | STOCH_EXIT | 5 | 724.29 | 0.63% | 15.6650 | 0.0000 | 0.4903 | M30 bullish FVG [15.4350, 15.6650], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 17 | 2025-07-16 23:30 | 16.9450 | 16.1310 | 2025-07-17 10:30 | 17.4250 | STOCH_EXIT | 9 | 3147.14 | 2.73% | 16.9000 | 0.0000 | 0.5427 | M30 bullish FVG [16.6850, 16.9000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 18 | 2025-07-17 14:00 | 17.1150 | 16.2670 | 2025-07-18 07:00 | 16.9700 | STOCH_EXIT | 21 | -1118.09 | -0.95% | 17.4150 | 0.0000 | 0.5654 | M30 bullish FVG [17.1450, 17.4150], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 19 | 2025-07-31 21:30 | 16.9600 | 16.1266 | 2025-08-01 06:30 | 17.1200 | STOCH_EXIT | 5 | 986.42 | 0.84% | 16.9000 | 0.0000 | 0.5556 | M30 bullish FVG [16.6700, 16.9000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 20 | 2025-10-08 12:00 | 12.4600 | 11.7466 | 2025-10-08 20:00 | 12.2300 | STOCH_EXIT | 16 | -2302.28 | -1.94% | 12.6050 | 0.0000 | 0.4756 | M30 bullish FVG [12.4650, 12.6050], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 21 | 2025-10-10 08:30 | 12.3350 | 11.5790 | 2025-10-11 08:00 | 12.1350 | STOCH_EXIT | 43 | -1995.01 | -1.72% | 12.2750 | 0.0000 | 0.5040 | M30 bullish FVG [12.1400, 12.2750], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 22 | 2025-10-17 09:30 | 12.5550 | 11.7983 | 2025-10-17 13:00 | 12.6100 | STOCH_EXIT | 7 | 386.00 | 0.34% | 12.4200 | 0.0000 | 0.5045 | M30 bullish FVG [12.0500, 12.4200], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 23 | 2026-01-29 18:30 | 12.7050 | 12.3656 | 2026-01-29 23:00 | 12.8800 | STOCH_EXIT | 9 | 1459.87 | 1.28% | 12.9450 | 0.0000 | 0.2263 | M30 bullish FVG [12.8500, 12.9450], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 24 | 2026-02-04 21:00 | 13.3500 | 12.9439 | 2026-02-05 17:30 | 13.2250 | STOCH_EXIT | 28 | -1203.10 | -1.04% | 13.3200 | 0.0000 | 0.2708 | M30 bullish FVG [13.0900, 13.3200], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 25 | 2026-02-06 12:00 | 13.1700 | 12.7181 | 2026-02-07 03:00 | 13.1000 | STOCH_EXIT | 25 | -723.27 | -0.63% | 13.3250 | 0.0000 | 0.3013 | M30 bullish FVG [13.2350, 13.3250], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 26 | 2026-04-01 20:30 | 12.7550 | 12.3730 | 2026-04-01 21:30 | 12.7750 | STOCH_EXIT | 2 | 64.39 | 0.06% | 12.6900 | 0.0000 | 0.2547 | M30 bullish FVG [12.6150, 12.6900], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 27 | 2026-04-03 12:30 | 12.8350 | 12.4500 | 2026-04-04 08:00 | 12.7750 | STOCH_EXIT | 33 | -647.96 | -0.57% | 12.8400 | 0.0000 | 0.2567 | M30 bullish FVG [12.7750, 12.8400], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 28 | 2026-05-13 07:30 | 12.4250 | 12.0442 | 2026-05-13 15:30 | 12.4450 | STOCH_EXIT | 16 | 68.84 | 0.06% | 12.3500 | 0.0000 | 0.2539 | M30 bullish FVG [12.2750, 12.3500], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 29 | 2026-05-13 22:30 | 12.3500 | 11.9692 | 2026-05-14 10:00 | 12.3750 | STOCH_EXIT | 10 | 116.26 | 0.10% | 12.4250 | 0.0000 | 0.2539 | M30 bullish FVG [12.3550, 12.4250], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 30 | 2026-05-28 17:00 | 11.9100 | 11.5096 | 2026-05-29 12:00 | 11.8850 | STOCH_EXIT | 25 | -350.53 | -0.31% | 12.0300 | 0.0000 | 0.2669 | M30 bullish FVG [11.9600, 12.0300], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 31 | 2026-06-11 15:00 | 11.2850 | 10.8546 | 2026-06-12 02:00 | 11.3300 | STOCH_EXIT | 18 | 336.93 | 0.30% | 11.2900 | 0.0000 | 0.2870 | M30 bullish FVG [11.2100, 11.2900], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 32 | 2026-06-23 23:30 | 10.8200 | 10.2889 | 2026-06-24 07:30 | 10.9750 | STOCH_EXIT | 3 | 1513.08 | 1.33% | 10.6000 | 0.0000 | 0.3541 | M30 bullish FVG [10.4650, 10.6000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 33 | 2026-07-03 19:30 | 10.7450 | 10.1471 | 2026-07-04 02:00 | 10.7550 | STOCH_EXIT | 9 | -8.02 | -0.01% | 10.8600 | 0.0000 | 0.3986 | M30 bullish FVG [10.7300, 10.8600], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 34 | 2026-07-08 07:00 | 10.4800 | 9.8435 | 2026-07-08 07:30 | 10.5150 | STOCH_EXIT | 1 | 269.53 | 0.23% | 10.3850 | 0.0000 | 0.4243 | M30 bullish FVG [10.2500, 10.3850], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 35 | 2026-10-07 11:00 | 10.1100 | 9.6743 | 2026-10-08 22:00 | 9.8250 | STOCH_EXIT | 57 | -3362.63 | -2.92% | 10.0100 | 0.0000 | 0.2905 | M30 bullish FVG [9.8000, 10.0100], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 95871.14
- Макс: 119014.57
- Финал: 112504.98

Полная кривая — в `*_equity.csv`.
