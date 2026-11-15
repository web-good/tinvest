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
| StochPeriod | 9 |
| UseStochExit | 1 |
| StopATR | 0.5 |
| TakeProfitATR | 0 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 1 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 25 |
| Выигрышных / проигрышных | 19 / 6 |
| Win rate | 76.00% |
| Gross profit / loss | 14877.44 / 7693.56 |
| Profit factor | 1.934 |
| Чистый PnL | 7183.88 (7.18%) |
| Стартовый / финальный капитал | 100000.00 / 107183.88 |
| Макс. просадка | 4215.81 (4.20%) |
| Средняя прибыль / убыток | 783.02 / 1282.26 |
| Expectancy | 287.36 |
| Лучшая / худшая сделка | 2640.08 / -1741.89 |
| Exposure | 0.68% |
| CAGR | 3.53% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-10-25 12:00 | 12.8400 | 12.6613 | 2024-10-25 13:30 | 12.6613 | SL | 3 | -1474.05 | -1.49% | 12.8950 | 0.0000 | 0.3574 | M30 bullish FVG [12.7900, 12.8950], stochastic crossed below 20 | FVG stop 12.6613 |
| 2 | 2024-10-30 12:00 | 12.6800 | 12.4664 | 2024-10-30 15:00 | 12.4664 | SL | 6 | -1741.89 | -1.78% | 12.4900 | 0.0000 | 0.4273 | M30 bullish FVG [12.3000, 12.4900], stochastic crossed below 20 | FVG stop 12.4664 |
| 3 | 2024-11-18 22:00 | 14.0600 | 13.8123 | 2024-11-19 12:30 | 14.2100 | STOCH_EXIT | 15 | 923.88 | 0.97% | 13.7600 | 0.0000 | 0.4954 | M30 bullish FVG [13.4400, 13.7600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 4 | 2024-12-18 09:00 | 12.5350 | 12.2837 | 2024-12-18 12:00 | 12.7950 | STOCH_EXIT | 6 | 1904.48 | 1.97% | 12.4300 | 0.0000 | 0.5025 | M30 bullish FVG [12.2200, 12.4300], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 5 | 2025-01-16 18:00 | 15.4150 | 15.1372 | 2025-01-17 10:30 | 15.4500 | STOCH_EXIT | 19 | 125.23 | 0.13% | 15.5900 | 0.0000 | 0.5555 | M30 bullish FVG [15.4000, 15.5900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 6 | 2025-02-13 14:00 | 16.3600 | 16.1082 | 2025-02-13 16:00 | 16.5150 | STOCH_EXIT | 4 | 831.38 | 0.85% | 15.8400 | 0.0000 | 0.5036 | M30 bullish FVG [15.7000, 15.8400], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 7 | 2025-03-05 21:30 | 17.6850 | 17.3396 | 2025-03-06 07:00 | 17.8400 | STOCH_EXIT | 6 | 768.53 | 0.78% | 17.6800 | 0.0000 | 0.6907 | M30 bullish FVG [17.4800, 17.6800], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 8 | 2025-05-06 15:30 | 14.6550 | 14.3157 | 2025-05-06 18:00 | 14.8050 | STOCH_EXIT | 5 | 933.36 | 0.92% | 14.6900 | 0.0000 | 0.6786 | M30 bullish FVG [14.5000, 14.6900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 9 | 2025-06-11 08:30 | 15.4750 | 15.1474 | 2025-06-11 09:30 | 15.6800 | STOCH_EXIT | 2 | 1250.19 | 1.22% | 15.5600 | 0.0000 | 0.6553 | M30 bullish FVG [15.3550, 15.5600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 10 | 2025-06-20 07:00 | 16.0100 | 15.7241 | 2025-06-20 09:00 | 16.0700 | STOCH_EXIT | 4 | 281.34 | 0.27% | 15.8600 | 0.0000 | 0.5718 | M30 bullish FVG [15.7150, 15.8600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 11 | 2025-06-30 18:00 | 16.3500 | 16.0986 | 2025-07-01 06:30 | 16.4600 | STOCH_EXIT | 12 | 589.65 | 0.57% | 16.4700 | 0.0000 | 0.5028 | M30 bullish FVG [16.3000, 16.4700], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 12 | 2025-07-10 13:00 | 15.7250 | 15.4799 | 2025-07-10 15:30 | 15.8400 | STOCH_EXIT | 5 | 654.84 | 0.63% | 15.6650 | 0.0000 | 0.4903 | M30 bullish FVG [15.4350, 15.6650], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 13 | 2025-07-16 23:00 | 16.9750 | 16.7037 | 2025-07-17 10:30 | 17.4250 | STOCH_EXIT | 10 | 2640.08 | 2.55% | 16.9000 | 0.0000 | 0.5427 | M30 bullish FVG [16.6850, 16.9000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 14 | 2025-07-31 21:30 | 16.9600 | 16.6822 | 2025-08-01 06:30 | 17.1200 | STOCH_EXIT | 5 | 900.65 | 0.84% | 16.9000 | 0.0000 | 0.5556 | M30 bullish FVG [16.6700, 16.9000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 15 | 2025-10-09 19:00 | 12.2900 | 12.0465 | 2025-10-09 21:30 | 12.4200 | STOCH_EXIT | 5 | 1035.28 | 0.96% | 12.2750 | 0.0000 | 0.4870 | M30 bullish FVG [12.1400, 12.2750], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 16 | 2025-10-17 09:30 | 12.5550 | 12.3028 | 2025-10-17 12:30 | 12.5900 | STOCH_EXIT | 6 | 195.12 | 0.18% | 12.4200 | 0.0000 | 0.5045 | M30 bullish FVG [12.0500, 12.4200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 17 | 2026-02-04 19:30 | 13.3800 | 13.2446 | 2026-02-04 22:30 | 13.2446 | SL | 6 | -1219.26 | -1.11% | 13.3200 | 0.0000 | 0.2708 | M30 bullish FVG [13.0900, 13.3200], stochastic crossed below 20 | FVG stop 13.2446 |
| 18 | 2026-04-01 18:30 | 12.7350 | 12.6077 | 2026-04-01 21:30 | 12.7750 | STOCH_EXIT | 6 | 231.58 | 0.21% | 12.6900 | 0.0000 | 0.2547 | M30 bullish FVG [12.6150, 12.6900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 19 | 2026-04-03 12:30 | 12.8350 | 12.7067 | 2026-04-03 14:30 | 12.7067 | SL | 4 | -1185.31 | -1.10% | 12.8400 | 0.0000 | 0.2567 | M30 bullish FVG [12.7750, 12.8400], stochastic crossed below 20 | FVG stop 12.7067 |
| 20 | 2026-05-13 07:30 | 12.4250 | 12.2981 | 2026-05-13 16:00 | 12.4500 | STOCH_EXIT | 17 | 108.04 | 0.10% | 12.3500 | 0.0000 | 0.2539 | M30 bullish FVG [12.2750, 12.3500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 21 | 2026-06-11 15:00 | 11.2850 | 11.1415 | 2026-06-12 02:00 | 11.3300 | STOCH_EXIT | 18 | 320.08 | 0.30% | 11.2900 | 0.0000 | 0.2870 | M30 bullish FVG [11.2100, 11.2900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 22 | 2026-06-23 22:30 | 10.8700 | 10.6930 | 2026-06-24 07:30 | 10.9750 | STOCH_EXIT | 5 | 931.37 | 0.87% | 10.6000 | 0.0000 | 0.3541 | M30 bullish FVG [10.4650, 10.6000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 23 | 2026-07-03 19:00 | 10.7750 | 10.5757 | 2026-07-03 23:30 | 10.7450 | STOCH_EXIT | 9 | -411.68 | -0.38% | 10.8600 | 0.0000 | 0.3986 | M30 bullish FVG [10.7300, 10.8600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 24 | 2026-07-08 07:00 | 10.4800 | 10.2678 | 2026-07-08 07:00 | 10.5150 | STOCH_EXIT | 0 | 252.38 | 0.23% | 10.3850 | 0.0000 | 0.4243 | M30 bullish FVG [10.2500, 10.3850], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 25 | 2026-10-07 11:00 | 10.1100 | 9.9648 | 2026-10-07 13:00 | 9.9648 | SL | 4 | -1661.38 | -1.54% | 10.0100 | 0.0000 | 0.2905 | M30 bullish FVG [9.8000, 10.0100], stochastic crossed below 20 | FVG stop 9.9648 |

## Движение капитала

- Старт: 100000.00
- Мин: 96158.26
- Макс: 110501.20
- Финал: 107183.88

Полная кривая — в `*_equity.csv`.
