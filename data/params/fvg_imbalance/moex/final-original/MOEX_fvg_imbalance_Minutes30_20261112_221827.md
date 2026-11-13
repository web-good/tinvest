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
| StochPeriod | 14 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 2 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 22 |
| Выигрышных / проигрышных | 17 / 5 |
| Win rate | 77.27% |
| Gross profit / loss | 9133.02 / 2421.77 |
| Profit factor | 3.771 |
| Чистый PnL | 6711.25 (6.71%) |
| Стартовый / финальный капитал | 100000.00 / 106711.25 |
| Макс. просадка | 1943.00 (1.87%) |
| Средняя прибыль / убыток | 537.24 / 484.35 |
| Expectancy | 305.06 |
| Лучшая / худшая сделка | 1367.82 / -832.47 |
| Exposure | 2.50% |
| CAGR | 6.71% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2025-11-18 18:30 | 161.8200 | 157.9763 | 2025-11-19 07:00 | 163.0200 | STOCH_EXIT | 12 | 632.92 | 0.64% | 161.5000 | 169.5074 | 3.8437 | M30 bullish FVG [160.3600, 161.5000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 2 | 2025-11-19 23:30 | 168.2700 | 164.3601 | 2025-11-20 12:00 | 169.1500 | STOCH_EXIT | 12 | 419.66 | 0.42% | 168.2500 | 176.0898 | 3.9099 | M30 bullish FVG [167.2000, 168.2500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 3 | 2025-11-21 12:30 | 170.8700 | 166.5031 | 2025-11-21 18:00 | 172.0800 | STOCH_EXIT | 11 | 612.73 | 0.61% | 169.6000 | 179.6038 | 4.3669 | M30 bullish FVG [168.1200, 169.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 4 | 2025-12-10 23:00 | 177.1100 | 172.9375 | 2025-12-11 07:00 | 177.6200 | STOCH_EXIT | 3 | 189.60 | 0.19% | 176.4200 | 185.4550 | 4.1725 | M30 bullish FVG [175.2600, 176.4200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 5 | 2026-01-20 09:30 | 176.8900 | 173.5129 | 2026-01-20 15:30 | 178.0100 | STOCH_EXIT | 12 | 537.25 | 0.53% | 176.4700 | 183.6442 | 3.3771 | M30 bullish FVG [175.4000, 176.4700], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 6 | 2026-02-24 15:30 | 183.6700 | 180.3523 | 2026-02-25 06:30 | 183.3200 | STOCH_EXIT | 17 | -293.42 | -0.29% | 184.6500 | 190.3055 | 3.3177 | M30 bullish FVG [183.7600, 184.6500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 7 | 2026-04-08 14:00 | 170.4100 | 167.8389 | 2026-04-08 20:00 | 170.9000 | STOCH_EXIT | 12 | 188.41 | 0.19% | 171.1200 | 175.5522 | 2.5711 | M30 bullish FVG [170.4300, 171.1200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 8 | 2026-04-15 10:30 | 169.0800 | 166.5074 | 2026-04-15 18:00 | 171.5300 | STOCH_EXIT | 15 | 1367.82 | 1.35% | 168.5600 | 174.2252 | 2.5726 | M30 bullish FVG [167.8700, 168.5600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 9 | 2026-04-16 15:00 | 171.6600 | 169.0126 | 2026-04-17 06:30 | 172.1000 | STOCH_EXIT | 18 | 160.87 | 0.16% | 171.3000 | 176.9548 | 2.6474 | M30 bullish FVG [170.4500, 171.3000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 10 | 2026-04-22 10:00 | 174.1700 | 171.3400 | 2026-04-22 17:00 | 174.0100 | STOCH_EXIT | 14 | -197.11 | -0.19% | 173.7700 | 179.8300 | 2.8300 | M30 bullish FVG [172.9400, 173.7700], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 11 | 2026-04-30 10:00 | 167.6300 | 164.4149 | 2026-05-01 03:30 | 167.9500 | STOCH_EXIT | 31 | 92.85 | 0.09% | 169.0800 | 174.0602 | 3.2151 | M30 bullish FVG [168.2200, 169.0800], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 12 | 2026-05-05 23:30 | 168.8900 | 165.8459 | 2026-05-07 13:30 | 167.8800 | STOCH_EXIT | 50 | -718.81 | -0.70% | 167.3100 | 174.9781 | 3.0441 | M30 bullish FVG [166.4100, 167.3100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 13 | 2026-06-15 16:30 | 177.2200 | 174.2664 | 2026-06-15 20:30 | 178.4400 | STOCH_EXIT | 8 | 604.46 | 0.59% | 177.7500 | 183.1272 | 2.9536 | M30 bullish FVG [176.5300, 177.7500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 14 | 2026-08-03 17:00 | 161.1100 | 155.0644 | 2026-08-03 20:30 | 161.7800 | STOCH_EXIT | 7 | 325.48 | 0.32% | 159.1900 | 173.2012 | 6.0456 | M30 bullish FVG [157.6500, 159.1900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 15 | 2026-08-26 07:30 | 153.0400 | 147.8751 | 2026-08-27 09:00 | 151.9500 | STOCH_EXIT | 38 | -832.47 | -0.81% | 153.5300 | 163.3698 | 5.1649 | M30 bullish FVG [152.0100, 153.5300], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 16 | 2026-09-01 09:00 | 152.6000 | 147.7702 | 2026-09-01 11:00 | 154.7800 | STOCH_EXIT | 4 | 1357.63 | 1.33% | 153.0400 | 162.2596 | 4.8298 | M30 bullish FVG [151.5900, 153.0400], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 17 | 2026-09-24 11:30 | 149.4200 | 145.6841 | 2026-09-24 18:00 | 149.5900 | STOCH_EXIT | 13 | 14.14 | 0.01% | 150.2700 | 156.8918 | 3.7359 | M30 bullish FVG [149.2300, 150.2700], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 18 | 2026-09-30 07:30 | 150.8500 | 147.2231 | 2026-09-30 18:30 | 150.4500 | STOCH_EXIT | 22 | -379.95 | -0.36% | 150.4900 | 158.1038 | 3.6269 | M30 bullish FVG [149.2600, 150.4900], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 19 | 2026-10-13 19:00 | 157.6300 | 154.4345 | 2026-10-14 08:30 | 158.9900 | STOCH_EXIT | 14 | 781.10 | 0.76% | 156.4100 | 164.0209 | 3.1955 | M30 bullish FVG [155.3600, 156.4100], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 20 | 2026-10-27 19:00 | 162.3500 | 159.0921 | 2026-10-28 08:30 | 163.6000 | STOCH_EXIT | 14 | 695.70 | 0.67% | 162.1600 | 168.8659 | 3.2579 | M30 bullish FVG [157.6800, 162.1600], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 21 | 2026-10-29 19:30 | 164.6200 | 161.0296 | 2026-10-29 23:00 | 165.4900 | STOCH_EXIT | 7 | 451.16 | 0.43% | 165.5000 | 171.8007 | 3.5904 | M30 bullish FVG [163.7600, 165.5000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 22 | 2026-11-12 11:00 | 163.6900 | 159.7614 | 2026-11-12 14:30 | 164.9500 | STOCH_EXIT | 7 | 701.24 | 0.67% | 164.9500 | 171.5471 | 3.9286 | M30 bullish FVG [163.5000, 164.9500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 99633.44
- Макс: 106796.03
- Финал: 106711.25

Полная кривая — в `*_equity.csv`.
