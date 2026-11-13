# Бэктест MOEX (Minutes30)

- Период: 2025-11-12 22:18 — 2026-11-12 22:18
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
| StopATR | 1.2 |
| TakeProfitATR | 1 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 24 |
| Выигрышных / проигрышных | 16 / 8 |
| Win rate | 66.67% |
| Gross profit / loss | 8776.87 / 3638.63 |
| Profit factor | 2.412 |
| Чистый PnL | 5138.24 (5.14%) |
| Стартовый / финальный капитал | 100000.00 / 105138.24 |
| Макс. просадка | 2623.35 (2.54%) |
| Средняя прибыль / убыток | 548.55 / 454.83 |
| Expectancy | 214.09 |
| Лучшая / худшая сделка | 1484.40 / -953.06 |
| Exposure | 2.08% |
| CAGR | 5.14% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2025-11-18 18:00 | 162.0600 | 157.4475 | 2025-11-19 07:00 | 163.0200 | STOCH_EXIT | 13 | 486.45 | 0.49% | 161.5000 | 165.9037 | 3.8437 | M30 bullish FVG [160.3600, 161.5000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 2 | 2025-11-20 10:30 | 167.3300 | 162.2036 | 2025-11-20 12:00 | 169.1500 | STOCH_EXIT | 3 | 991.06 | 0.99% | 168.2500 | 171.6020 | 4.2720 | M30 bullish FVG [167.2000, 168.2500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 3 | 2025-11-21 09:30 | 170.6900 | 165.4497 | 2025-11-21 10:30 | 172.9700 | STOCH_EXIT | 2 | 1243.82 | 1.23% | 169.6000 | 175.0569 | 4.3669 | M30 bullish FVG [168.1200, 169.6000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 4 | 2025-12-10 19:30 | 177.0900 | 172.0830 | 2025-12-11 07:00 | 177.6200 | STOCH_EXIT | 10 | 201.01 | 0.20% | 176.4200 | 181.2625 | 4.1725 | M30 bullish FVG [175.2600, 176.4200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 5 | 2026-01-20 09:30 | 176.8900 | 172.8375 | 2026-01-20 13:30 | 177.6400 | STOCH_EXIT | 8 | 332.19 | 0.32% | 176.4700 | 180.2671 | 3.3771 | M30 bullish FVG [175.4000, 176.4700], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 6 | 2026-02-03 15:30 | 186.5400 | 181.9602 | 2026-02-03 19:30 | 186.7100 | STOCH_EXIT | 8 | -9.14 | -0.01% | 188.1900 | 190.3565 | 3.8165 | M30 bullish FVG [182.9500, 188.1900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 7 | 2026-02-24 14:00 | 183.7200 | 179.7387 | 2026-02-24 21:00 | 183.0700 | STOCH_EXIT | 14 | -466.70 | -0.45% | 184.6500 | 187.0377 | 3.3177 | M30 bullish FVG [183.7600, 184.6500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 8 | 2026-04-08 12:00 | 170.5000 | 167.4147 | 2026-04-08 18:00 | 170.8100 | STOCH_EXIT | 12 | 83.61 | 0.08% | 171.1200 | 173.0711 | 2.5711 | M30 bullish FVG [170.4300, 171.1200], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 9 | 2026-04-15 10:30 | 169.0800 | 165.9929 | 2026-04-15 14:00 | 169.6100 | STOCH_EXIT | 7 | 216.39 | 0.21% | 168.5600 | 171.6526 | 2.5726 | M30 bullish FVG [167.8700, 168.5600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 10 | 2026-04-16 07:00 | 171.3400 | 168.1631 | 2026-04-16 07:00 | 171.6600 | STOCH_EXIT | 0 | 89.10 | 0.09% | 171.3000 | 173.9874 | 2.6474 | M30 bullish FVG [170.4500, 171.3000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 11 | 2026-04-22 10:00 | 174.1700 | 170.7740 | 2026-04-22 16:30 | 173.5300 | STOCH_EXIT | 13 | -480.17 | -0.47% | 173.7700 | 177.0000 | 2.8300 | M30 bullish FVG [172.9400, 173.7700], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 12 | 2026-04-30 08:00 | 168.7200 | 164.8619 | 2026-05-01 02:00 | 167.9000 | STOCH_EXIT | 32 | -592.99 | -0.59% | 169.0800 | 171.9351 | 3.2151 | M30 bullish FVG [168.2200, 169.0800], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 13 | 2026-05-05 21:30 | 169.1500 | 165.4971 | 2026-05-06 18:30 | 167.7300 | STOCH_EXIT | 29 | -953.06 | -0.94% | 167.3100 | 172.1941 | 3.0441 | M30 bullish FVG [166.4100, 167.3100], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 14 | 2026-06-01 18:00 | 176.9700 | 173.3247 | 2026-06-01 19:30 | 178.8300 | STOCH_EXIT | 3 | 958.80 | 0.95% | 177.7600 | 180.0078 | 3.0378 | M30 bullish FVG [176.8700, 177.7600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 15 | 2026-06-15 14:00 | 177.6400 | 174.0957 | 2026-06-15 20:30 | 178.4400 | STOCH_EXIT | 13 | 354.52 | 0.35% | 177.7500 | 180.5936 | 2.9536 | M30 bullish FVG [176.5300, 177.7500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 16 | 2026-08-03 14:30 | 160.9500 | 153.6953 | 2026-08-03 20:30 | 161.7800 | STOCH_EXIT | 12 | 421.24 | 0.42% | 159.1900 | 166.9956 | 6.0456 | M30 bullish FVG [157.6500, 159.1900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 17 | 2026-08-26 07:30 | 153.0400 | 146.8421 | 2026-08-26 11:30 | 153.1600 | STOCH_EXIT | 8 | -22.18 | -0.02% | 153.5300 | 158.2049 | 5.1649 | M30 bullish FVG [152.0100, 153.5300], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 18 | 2026-09-01 07:30 | 152.7700 | 146.9742 | 2026-09-01 11:00 | 154.7800 | STOCH_EXIT | 7 | 1243.67 | 1.21% | 153.0400 | 157.5998 | 4.8298 | M30 bullish FVG [151.5900, 153.0400], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 19 | 2026-09-24 10:30 | 149.8700 | 145.3869 | 2026-09-24 17:00 | 149.4600 | STOCH_EXIT | 13 | -386.17 | -0.37% | 150.2700 | 153.6059 | 3.7359 | M30 bullish FVG [149.2300, 150.2700], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 20 | 2026-09-29 21:30 | 151.3700 | 147.2137 | 2026-09-30 18:30 | 150.4500 | STOCH_EXIT | 29 | -728.22 | -0.71% | 150.4900 | 154.8336 | 3.4636 | M30 bullish FVG [149.2600, 150.4900], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 21 | 2026-10-13 16:30 | 158.1200 | 154.2854 | 2026-10-14 08:30 | 158.9900 | STOCH_EXIT | 19 | 462.44 | 0.45% | 156.4100 | 161.3155 | 3.1955 | M30 bullish FVG [155.3600, 156.4100], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 22 | 2026-10-27 18:30 | 162.5500 | 158.6405 | 2026-10-28 09:00 | 165.0700 | STOCH_EXIT | 16 | 1484.40 | 1.45% | 162.1600 | 165.8079 | 3.2579 | M30 bullish FVG [157.6800, 162.1600], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 23 | 2026-10-29 17:00 | 164.9900 | 160.6816 | 2026-10-29 22:00 | 165.1700 | STOCH_EXIT | 10 | 9.40 | 0.01% | 165.5000 | 168.5804 | 3.5904 | M30 bullish FVG [163.7600, 165.5000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 24 | 2026-11-11 21:00 | 164.2300 | 159.7965 | 2026-11-11 22:30 | 164.7100 | STOCH_EXIT | 3 | 198.78 | 0.19% | 164.9500 | 167.9246 | 3.6946 | M30 bullish FVG [163.5000, 164.9500], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 99486.97
- Макс: 105265.72
- Финал: 105138.24

Полная кривая — в `*_equity.csv`.
