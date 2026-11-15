# Бэктест SVCB (Minutes30)

- Период: 2024-10-13 23:01 — 2026-10-13 23:01
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.25 |
| MaxWaitBars | 30 |
| StochPeriod | 9 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 2 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 36 |
| Выигрышных / проигрышных | 25 / 11 |
| Win rate | 69.44% |
| Gross profit / loss | 28407.20 / 15526.14 |
| Profit factor | 1.830 |
| Чистый PnL | 12881.06 (12.88%) |
| Стартовый / финальный капитал | 100000.00 / 112881.06 |
| Макс. просадка | 6781.28 (5.67%) |
| Средняя прибыль / убыток | 1136.29 / 1411.47 |
| Expectancy | 357.81 |
| Лучшая / худшая сделка | 4600.89 / -3454.92 |
| Exposure | 1.51% |
| CAGR | 6.25% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-10-25 12:00 | 12.8400 | 12.4826 | 2024-10-25 16:30 | 12.4826 | SL | 9 | -2849.23 | -2.88% | 12.8950 | 13.5547 | 0.3574 | M30 bullish FVG [12.7900, 12.8950], stochastic crossed below 20 | FVG stop 12.4826 |
| 2 | 2024-10-30 12:00 | 12.6800 | 12.2527 | 2024-10-30 17:30 | 12.5900 | STOCH_EXIT | 11 | -780.03 | -0.81% | 12.4900 | 13.5346 | 0.4273 | M30 bullish FVG [12.3000, 12.4900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 3 | 2024-11-15 10:00 | 12.9400 | 12.4638 | 2024-11-15 12:30 | 13.5750 | STOCH_EXIT | 5 | 4600.89 | 4.80% | 13.1100 | 13.8924 | 0.4762 | M30 bullish FVG [12.9700, 13.1100], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 4 | 2024-11-18 22:00 | 14.0600 | 13.5646 | 2024-11-19 12:30 | 14.2100 | STOCH_EXIT | 15 | 964.64 | 0.97% | 13.7600 | 15.0508 | 0.4954 | M30 bullish FVG [13.4400, 13.7600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 5 | 2024-11-27 16:30 | 12.4900 | 11.8754 | 2024-11-27 19:00 | 12.9000 | STOCH_EXIT | 5 | 3218.17 | 3.18% | 12.7250 | 13.7193 | 0.6146 | M30 bullish FVG [12.5350, 12.7250], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 6 | 2024-12-16 23:30 | 11.8750 | 11.4266 | 2024-12-17 10:30 | 12.0550 | STOCH_EXIT | 8 | 1478.71 | 1.41% | 12.0150 | 12.7719 | 0.4484 | M30 bullish FVG [11.8750, 12.0150], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 7 | 2024-12-18 09:00 | 12.5350 | 12.0325 | 2024-12-18 12:00 | 12.7950 | STOCH_EXIT | 6 | 2102.35 | 1.97% | 12.4300 | 13.5400 | 0.5025 | M30 bullish FVG [12.2200, 12.4300], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 8 | 2025-01-16 18:00 | 15.4150 | 14.8595 | 2025-01-17 10:30 | 15.4500 | STOCH_EXIT | 19 | 136.97 | 0.13% | 15.5900 | 16.5261 | 0.5555 | M30 bullish FVG [15.4000, 15.5900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 9 | 2025-02-13 14:00 | 16.3600 | 15.8564 | 2025-02-13 16:00 | 16.5150 | STOCH_EXIT | 4 | 914.51 | 0.85% | 16.8900 | 17.3672 | 0.5036 | M30 bullish FVG [16.6000, 16.8900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 10 | 2025-03-05 21:30 | 17.6850 | 16.9943 | 2025-03-06 07:00 | 17.8400 | STOCH_EXIT | 6 | 850.87 | 0.78% | 17.6800 | 19.0665 | 0.6907 | M30 bullish FVG [17.4800, 17.6800], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 11 | 2025-03-28 14:30 | 16.5300 | 15.8137 | 2025-03-28 18:00 | 16.8000 | STOCH_EXIT | 7 | 1672.01 | 1.53% | 17.0300 | 17.9627 | 0.7163 | M30 bullish FVG [16.7500, 17.0300], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 12 | 2025-05-06 15:30 | 14.6550 | 13.9764 | 2025-05-06 18:00 | 14.8050 | STOCH_EXIT | 5 | 1028.05 | 0.92% | 14.6900 | 16.0122 | 0.6786 | M30 bullish FVG [14.5000, 14.6900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 13 | 2025-05-15 11:30 | 15.7300 | 15.1224 | 2025-05-15 16:30 | 15.7050 | STOCH_EXIT | 10 | -293.17 | -0.26% | 15.9100 | 16.9451 | 0.6076 | M30 bullish FVG [15.7400, 15.9100], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 14 | 2025-06-11 08:30 | 15.4750 | 14.8197 | 2025-06-11 09:30 | 15.6800 | STOCH_EXIT | 2 | 1382.78 | 1.22% | 15.5600 | 16.7855 | 0.6553 | M30 bullish FVG [15.3550, 15.5600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 15 | 2025-06-20 07:00 | 16.0100 | 15.4382 | 2025-06-20 09:00 | 16.0700 | STOCH_EXIT | 4 | 312.12 | 0.27% | 15.8600 | 17.1536 | 0.5718 | M30 bullish FVG [15.7150, 15.8600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 16 | 2025-06-30 18:00 | 16.3500 | 15.8472 | 2025-07-01 06:30 | 16.4600 | STOCH_EXIT | 12 | 655.16 | 0.57% | 16.4700 | 17.3556 | 0.5028 | M30 bullish FVG [16.3000, 16.4700], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 17 | 2025-07-10 13:00 | 15.7250 | 15.2347 | 2025-07-10 15:30 | 15.8400 | STOCH_EXIT | 5 | 724.29 | 0.63% | 15.6650 | 16.7055 | 0.4903 | M30 bullish FVG [15.4350, 15.6650], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 18 | 2025-07-16 23:00 | 16.9750 | 16.4323 | 2025-07-17 10:30 | 17.4250 | STOCH_EXIT | 10 | 2943.04 | 2.55% | 16.9000 | 18.0604 | 0.5427 | M30 bullish FVG [16.6850, 16.9000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 19 | 2025-07-17 14:00 | 17.1150 | 16.5496 | 2025-07-18 07:00 | 16.9700 | STOCH_EXIT | 21 | -1118.09 | -0.95% | 17.4150 | 18.2457 | 0.5654 | M30 bullish FVG [17.1450, 17.4150], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 20 | 2025-07-31 21:30 | 16.9600 | 16.4044 | 2025-08-01 06:30 | 17.1200 | STOCH_EXIT | 5 | 986.42 | 0.84% | 16.9000 | 18.0712 | 0.5556 | M30 bullish FVG [16.6700, 16.9000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 21 | 2025-10-08 12:00 | 12.4600 | 11.9844 | 2025-10-08 20:00 | 12.2300 | STOCH_EXIT | 16 | -2302.28 | -1.94% | 12.6050 | 13.4113 | 0.4756 | M30 bullish FVG [12.4650, 12.6050], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 22 | 2025-10-09 19:00 | 12.2900 | 11.8030 | 2025-10-09 21:30 | 12.4200 | STOCH_EXIT | 5 | 1105.86 | 0.96% | 12.2750 | 13.2640 | 0.4870 | M30 bullish FVG [12.1400, 12.2750], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 23 | 2025-10-17 09:30 | 12.5550 | 12.0505 | 2025-10-17 12:30 | 12.5900 | STOCH_EXIT | 6 | 208.58 | 0.18% | 12.4200 | 13.5639 | 0.5045 | M30 bullish FVG [12.0500, 12.4200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 24 | 2026-01-29 18:30 | 12.7050 | 12.4787 | 2026-01-29 22:30 | 12.8250 | STOCH_EXIT | 8 | 986.56 | 0.84% | 12.9450 | 13.1575 | 0.2263 | M30 bullish FVG [12.8500, 12.9450], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 25 | 2026-02-04 19:30 | 13.3800 | 13.1092 | 2026-02-05 13:30 | 13.1092 | SL | 23 | -2499.20 | -2.12% | 13.3200 | 13.9215 | 0.2708 | M30 bullish FVG [13.0900, 13.3200], stochastic crossed below 20 | FVG stop 13.1092 |
| 26 | 2026-02-06 12:00 | 13.1700 | 12.8687 | 2026-02-07 03:00 | 13.1000 | STOCH_EXIT | 25 | -731.59 | -0.63% | 13.3250 | 13.7725 | 0.3013 | M30 bullish FVG [13.2350, 13.3250], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 27 | 2026-04-01 18:30 | 12.7350 | 12.4803 | 2026-04-01 21:30 | 12.7750 | STOCH_EXIT | 6 | 245.21 | 0.21% | 12.6900 | 13.2444 | 0.2547 | M30 bullish FVG [12.6150, 12.6900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 28 | 2026-04-03 12:30 | 12.8350 | 12.5783 | 2026-04-04 08:00 | 12.7750 | STOCH_EXIT | 33 | -655.25 | -0.57% | 12.8400 | 13.3483 | 0.2567 | M30 bullish FVG [12.7750, 12.8400], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 29 | 2026-05-13 07:30 | 12.4250 | 12.1711 | 2026-05-13 16:00 | 12.4500 | STOCH_EXIT | 17 | 115.57 | 0.10% | 12.3500 | 12.9327 | 0.2539 | M30 bullish FVG [12.2750, 12.3500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 30 | 2026-05-13 22:30 | 12.3500 | 12.0961 | 2026-05-14 09:00 | 12.3800 | STOCH_EXIT | 8 | 164.01 | 0.14% | 12.4250 | 12.8577 | 0.2539 | M30 bullish FVG [12.3550, 12.4250], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 31 | 2026-05-28 17:00 | 11.9100 | 11.6431 | 2026-05-29 11:30 | 11.8800 | STOCH_EXIT | 24 | -402.19 | -0.35% | 12.0300 | 12.4439 | 0.2669 | M30 bullish FVG [11.9600, 12.0300], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 32 | 2026-06-11 15:00 | 11.2850 | 10.9980 | 2026-06-12 02:00 | 11.3300 | STOCH_EXIT | 18 | 343.66 | 0.30% | 11.2900 | 11.8589 | 0.2870 | M30 bullish FVG [11.2100, 11.2900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 33 | 2026-06-23 22:30 | 10.8700 | 10.5159 | 2026-06-24 07:30 | 10.9750 | STOCH_EXIT | 5 | 997.22 | 0.87% | 10.6000 | 11.5781 | 0.3541 | M30 bullish FVG [10.4650, 10.6000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 34 | 2026-07-03 19:00 | 10.7750 | 10.3764 | 2026-07-03 23:30 | 10.7450 | STOCH_EXIT | 9 | -440.21 | -0.38% | 10.8600 | 11.5723 | 0.3986 | M30 bullish FVG [10.7300, 10.8600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 35 | 2026-07-08 07:00 | 10.4800 | 10.0557 | 2026-07-08 07:00 | 10.5150 | STOCH_EXIT | 0 | 269.53 | 0.23% | 10.3850 | 11.3287 | 0.4243 | M30 bullish FVG [10.2500, 10.3850], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 36 | 2026-10-07 11:00 | 10.1100 | 9.8195 | 2026-10-08 09:00 | 9.8195 | SL | 31 | -3454.92 | -2.97% | 10.0100 | 10.6909 | 0.2905 | M30 bullish FVG [9.8000, 10.0100], stochastic crossed below 20 | FVG stop 9.8195 |

## Движение капитала

- Старт: 100000.00
- Мин: 95506.59
- Макс: 119662.34
- Финал: 112881.06

Полная кривая — в `*_equity.csv`.
