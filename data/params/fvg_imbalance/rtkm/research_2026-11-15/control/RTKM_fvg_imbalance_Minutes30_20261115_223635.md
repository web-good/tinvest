# Бэктест RTKM (Minutes30)

- Период: 2023-11-15 22:36 — 2025-11-15 22:36
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
| Всего сделок | 35 |
| Выигрышных / проигрышных | 26 / 9 |
| Win rate | 74.29% |
| Gross profit / loss | 30438.55 / 18161.03 |
| Profit factor | 1.676 |
| Чистый PnL | 12277.52 (12.28%) |
| Стартовый / финальный капитал | 100000.00 / 112277.52 |
| Макс. просадка | 7533.74 (6.43%) |
| Средняя прибыль / убыток | 1170.71 / 2017.89 |
| Expectancy | 350.79 |
| Лучшая / худшая сделка | 5852.10 / -4731.01 |
| Exposure | 2.37% |
| CAGR | 5.95% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-12-08 13:30 | 71.9000 | 69.7485 | 2023-12-08 17:30 | 72.0800 | STOCH_EXIT | 8 | 150.13 | 0.15% | 71.2200 | 76.2029 | 2.1515 | M30 bullish FVG [70.6700, 71.2200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 2 | 2024-01-08 18:00 | 70.9300 | 69.2009 | 2024-01-08 23:00 | 71.3000 | STOCH_EXIT | 10 | 421.43 | 0.42% | 70.8300 | 74.3881 | 1.7291 | M30 bullish FVG [70.3900, 70.8300], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 3 | 2024-02-07 18:30 | 76.3000 | 74.9425 | 2024-02-08 17:30 | 75.5900 | STOCH_EXIT | 27 | -1029.59 | -1.03% | 76.4500 | 79.0151 | 1.3575 | M30 bullish FVG [76.1000, 76.4500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 4 | 2024-02-20 10:30 | 86.0000 | 84.0237 | 2024-02-20 14:00 | 87.9000 | STOCH_EXIT | 7 | 2085.01 | 2.11% | 86.7600 | 89.9525 | 1.9763 | M30 bullish FVG [86.2700, 86.7600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 5 | 2024-02-20 21:00 | 86.2800 | 84.3037 | 2024-02-22 09:30 | 86.7700 | STOCH_EXIT | 35 | 472.07 | 0.47% | 87.3500 | 90.2325 | 1.9763 | M30 bullish FVG [86.0500, 87.3500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 6 | 2024-03-21 13:30 | 94.6400 | 92.2428 | 2024-03-22 21:00 | 92.2428 | SL | 44 | -2664.97 | -2.63% | 95.0400 | 99.4344 | 2.3972 | M30 bullish FVG [94.3000, 95.0400], stochastic crossed below 20 | FVG stop 92.2428 |
| 7 | 2024-03-26 17:30 | 94.3600 | 91.7708 | 2024-03-27 10:00 | 95.6000 | STOCH_EXIT | 14 | 1202.27 | 1.21% | 94.6700 | 99.5385 | 2.5892 | M30 bullish FVG [93.9300, 94.6700], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 8 | 2024-03-27 13:30 | 93.6800 | 91.0150 | 2024-03-27 18:30 | 95.4000 | STOCH_EXIT | 10 | 1739.24 | 1.73% | 95.2500 | 99.0100 | 2.6650 | M30 bullish FVG [94.4500, 95.2500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 9 | 2024-03-28 11:00 | 94.5300 | 91.8818 | 2024-03-28 12:00 | 95.6800 | STOCH_EXIT | 2 | 1139.29 | 1.12% | 95.2500 | 99.8265 | 2.6482 | M30 bullish FVG [94.4500, 95.2500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 10 | 2024-04-01 13:00 | 102.4200 | 99.4934 | 2024-04-01 14:30 | 104.0400 | STOCH_EXIT | 3 | 1531.94 | 1.48% | 104.0400 | 108.2732 | 2.9266 | M30 bullish FVG [102.4600, 104.0400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 11 | 2024-04-09 18:00 | 97.3600 | 94.5719 | 2024-04-10 09:30 | 98.1000 | STOCH_EXIT | 12 | 687.23 | 0.66% | 98.6100 | 102.9362 | 2.7881 | M30 bullish FVG [97.4300, 98.6100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 12 | 2024-05-07 11:30 | 96.7900 | 94.8288 | 2024-05-07 21:00 | 96.4400 | STOCH_EXIT | 19 | -486.81 | -0.46% | 96.4500 | 100.7123 | 1.9612 | M30 bullish FVG [95.7000, 96.4500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 13 | 2024-05-21 17:00 | 103.3200 | 100.7631 | 2024-05-22 10:30 | 105.6800 | STOCH_EXIT | 16 | 2278.06 | 2.18% | 104.6300 | 108.4338 | 2.5569 | M30 bullish FVG [103.6300, 104.6300], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 14 | 2024-05-24 15:00 | 103.3200 | 100.5864 | 2024-05-25 10:00 | 105.4700 | STOCH_EXIT | 18 | 2127.43 | 1.98% | 106.3600 | 108.7873 | 2.7336 | M30 bullish FVG [105.6500, 106.3600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 15 | 2024-07-08 19:00 | 85.3300 | 82.2205 | 2024-07-09 09:30 | 86.2900 | STOCH_EXIT | 10 | 1118.96 | 1.02% | 85.2800 | 91.5489 | 3.1095 | M30 bullish FVG [84.3200, 85.2800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 16 | 2024-07-17 22:00 | 86.4200 | 82.9986 | 2024-07-18 11:00 | 87.5800 | STOCH_EXIT | 7 | 1373.44 | 1.24% | 86.2400 | 93.2628 | 3.4214 | M30 bullish FVG [84.9800, 86.2400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 17 | 2024-07-26 14:00 | 87.0300 | 83.9613 | 2024-07-27 13:00 | 83.9613 | SL | 26 | -4037.43 | -3.62% | 86.6800 | 93.1675 | 3.0687 | M30 bullish FVG [85.8600, 86.6800], stochastic crossed below 20 | FVG stop 83.9613 |
| 18 | 2024-08-14 13:30 | 85.6900 | 82.7529 | 2024-08-16 09:30 | 84.9100 | STOCH_EXIT | 48 | -1090.28 | -1.01% | 85.8900 | 91.5642 | 2.9371 | M30 bullish FVG [85.0000, 85.8900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 19 | 2024-12-19 18:30 | 51.4600 | 49.2691 | 2024-12-20 13:30 | 54.3400 | STOCH_EXIT | 24 | 5852.10 | 5.49% | 52.7300 | 55.8417 | 2.1909 | M30 bullish FVG [51.5000, 52.7300], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 20 | 2025-01-17 23:30 | 67.8000 | 65.1020 | 2025-01-18 16:00 | 68.3200 | STOCH_EXIT | 19 | 750.22 | 0.67% | 68.1000 | 73.1960 | 2.6980 | M30 bullish FVG [67.2700, 68.1000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 21 | 2025-03-05 07:30 | 71.5700 | 68.1474 | 2025-03-05 11:00 | 72.3900 | STOCH_EXIT | 7 | 1181.87 | 1.04% | 71.2800 | 78.4152 | 3.4226 | M30 bullish FVG [70.4000, 71.2800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 22 | 2025-03-28 23:30 | 66.1800 | 63.4068 | 2025-03-29 05:30 | 66.5700 | STOCH_EXIT | 7 | 559.87 | 0.49% | 66.8000 | 71.7264 | 2.7732 | M30 bullish FVG [65.9700, 66.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 23 | 2025-04-10 12:00 | 60.3100 | 56.7544 | 2025-04-11 08:30 | 60.8300 | STOCH_EXIT | 28 | 877.51 | 0.76% | 60.5400 | 67.4212 | 3.5556 | M30 bullish FVG [58.9800, 60.5400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 24 | 2025-06-26 07:30 | 56.1700 | 54.2756 | 2025-06-26 13:00 | 56.2600 | STOCH_EXIT | 11 | 69.60 | 0.06% | 55.3000 | 59.9588 | 1.8944 | M30 bullish FVG [54.7000, 55.3000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 25 | 2025-06-30 22:30 | 58.3100 | 56.4680 | 2025-07-01 09:00 | 58.1900 | STOCH_EXIT | 8 | -354.72 | -0.31% | 58.1700 | 61.9939 | 1.8420 | M30 bullish FVG [56.7200, 58.1700], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 26 | 2025-07-03 21:30 | 58.4700 | 56.7108 | 2025-07-04 12:30 | 58.5100 | STOCH_EXIT | 17 | -36.61 | -0.03% | 58.4900 | 61.9883 | 1.7592 | M30 bullish FVG [57.8000, 58.4900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 27 | 2025-07-11 08:00 | 59.8400 | 57.9665 | 2025-07-11 09:00 | 57.9665 | SL | 2 | -3729.61 | -3.23% | 60.7900 | 63.5871 | 1.8735 | M30 bullish FVG [60.2700, 60.7900], stochastic crossed below 20 | FVG stop 57.9665 |
| 28 | 2025-07-17 16:30 | 61.7700 | 59.7075 | 2025-07-18 07:00 | 62.3900 | STOCH_EXIT | 16 | 1009.84 | 0.90% | 63.0600 | 65.8949 | 2.0625 | M30 bullish FVG [62.1400, 63.0600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 29 | 2025-08-21 18:30 | 69.4100 | 67.2508 | 2025-08-22 08:00 | 69.7700 | STOCH_EXIT | 14 | 473.37 | 0.42% | 71.3800 | 73.7283 | 2.1592 | M30 bullish FVG [70.8300, 71.3800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 30 | 2025-08-27 09:00 | 73.4300 | 71.1838 | 2025-08-27 10:00 | 73.8400 | STOCH_EXIT | 2 | 518.00 | 0.46% | 73.1100 | 77.9223 | 2.2462 | M30 bullish FVG [72.5000, 73.1100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 31 | 2025-10-07 20:00 | 58.9700 | 56.7298 | 2025-10-07 23:00 | 59.4700 | STOCH_EXIT | 6 | 850.71 | 0.75% | 59.0600 | 63.4504 | 2.2402 | M30 bullish FVG [58.2500, 59.0600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 32 | 2025-10-09 22:30 | 58.2200 | 55.8755 | 2025-10-11 02:00 | 55.8755 | SL | 38 | -4731.01 | -4.12% | 56.6300 | 62.9090 | 2.3445 | M30 bullish FVG [55.5000, 56.6300], stochastic crossed below 20 | FVG stop 55.8755 |
| 33 | 2025-10-17 11:30 | 63.9600 | 61.2115 | 2025-10-17 18:00 | 64.3700 | STOCH_EXIT | 13 | 594.84 | 0.54% | 63.2100 | 69.4570 | 2.7485 | M30 bullish FVG [61.5200, 63.2100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 34 | 2025-10-21 07:30 | 65.5000 | 62.7796 | 2025-10-21 13:30 | 66.3400 | STOCH_EXIT | 12 | 1308.20 | 1.18% | 66.2600 | 70.9408 | 2.7204 | M30 bullish FVG [65.5600, 66.2600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 35 | 2025-10-22 22:30 | 62.9000 | 60.0639 | 2025-10-23 16:00 | 63.0000 | STOCH_EXIT | 22 | 65.95 | 0.06% | 64.2100 | 68.5722 | 2.8361 | M30 bullish FVG [63.4400, 64.2100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 98193.02
- Макс: 117099.27
- Финал: 112277.52

Полная кривая — в `*_equity.csv`.
