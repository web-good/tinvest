# Бэктест ENPG (Minutes30)

- Период: 2023-11-13 23:18 — 2025-11-13 23:18
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
| Всего сделок | 37 |
| Выигрышных / проигрышных | 19 / 18 |
| Win rate | 51.35% |
| Gross profit / loss | 15497.40 / 15691.18 |
| Profit factor | 0.988 |
| Чистый PnL | -193.79 (-0.19%) |
| Стартовый / финальный капитал | 100000.00 / 99806.21 |
| Макс. просадка | 9131.97 (8.93%) |
| Средняя прибыль / убыток | 815.65 / 871.73 |
| Expectancy | -5.24 |
| Лучшая / худшая сделка | 2526.36 / -3483.08 |
| Exposure | 3.23% |
| CAGR | -0.10% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-12-13 17:30 | 425.4000 | 411.7863 | 2023-12-14 10:00 | 435.4000 | STOCH_EXIT | 14 | 2239.29 | 2.25% | 427.0000 | 452.6275 | 13.6137 | M30 bullish FVG [422.0000, 427.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 2 | 2023-12-14 16:30 | 433.0000 | 419.2158 | 2023-12-15 10:00 | 431.8000 | STOCH_EXIT | 16 | -385.25 | -0.38% | 431.4000 | 460.5684 | 13.7842 | M30 bullish FVG [425.2000, 431.4000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 3 | 2023-12-27 23:00 | 449.2000 | 435.4664 | 2023-12-29 14:00 | 443.6000 | STOCH_EXIT | 40 | -1366.49 | -1.35% | 452.0000 | 476.6672 | 13.7336 | M30 bullish FVG [448.2000, 452.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 4 | 2024-01-17 12:00 | 475.2000 | 463.2160 | 2024-01-17 20:00 | 474.4000 | STOCH_EXIT | 16 | -268.98 | -0.27% | 466.2000 | 499.1679 | 11.9840 | M30 bullish FVG [462.0000, 466.2000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 5 | 2024-03-01 23:30 | 437.8000 | 428.1601 | 2024-03-02 16:00 | 438.8000 | STOCH_EXIT | 13 | 128.07 | 0.13% | 438.6000 | 457.0798 | 9.6399 | M30 bullish FVG [436.0000, 438.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 6 | 2024-03-19 14:00 | 442.2000 | 433.5342 | 2024-03-19 21:30 | 440.0000 | STOCH_EXIT | 15 | -596.89 | -0.60% | 444.2000 | 459.5316 | 8.6658 | M30 bullish FVG [442.0000, 444.2000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 7 | 2024-03-26 10:30 | 432.0000 | 421.3536 | 2024-03-26 17:00 | 432.0000 | STOCH_EXIT | 13 | -99.36 | -0.10% | 430.8000 | 453.2929 | 10.6464 | M30 bullish FVG [427.6000, 430.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 8 | 2024-03-29 11:00 | 427.8000 | 418.0174 | 2024-03-29 17:30 | 428.0000 | STOCH_EXIT | 13 | -52.87 | -0.05% | 431.4000 | 447.3652 | 9.7826 | M30 bullish FVG [428.0000, 431.4000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 9 | 2024-04-04 22:30 | 470.0000 | 459.1738 | 2024-04-05 17:00 | 469.4000 | STOCH_EXIT | 18 | -225.71 | -0.23% | 473.8000 | 491.6524 | 10.8262 | M30 bullish FVG [468.0000, 473.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 10 | 2024-04-09 14:30 | 508.2000 | 493.0800 | 2024-04-09 20:30 | 515.4000 | STOCH_EXIT | 12 | 1304.20 | 1.32% | 506.4000 | 538.4400 | 15.1200 | M30 bullish FVG [502.0000, 506.4000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 11 | 2024-04-10 22:00 | 529.2000 | 511.1886 | 2024-04-11 23:00 | 518.8000 | STOCH_EXIT | 31 | -2075.56 | -2.06% | 532.0000 | 565.2229 | 18.0114 | M30 bullish FVG [526.8000, 532.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 12 | 2024-05-10 18:00 | 479.7000 | 465.3893 | 2024-05-11 13:00 | 478.9000 | STOCH_EXIT | 18 | -262.26 | -0.27% | 479.7000 | 508.3214 | 14.3107 | M30 bullish FVG [471.1000, 479.7000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 13 | 2024-05-15 17:00 | 486.0000 | 471.6700 | 2024-05-16 15:30 | 485.3000 | STOCH_EXIT | 26 | -239.50 | -0.24% | 491.2500 | 514.6599 | 14.3300 | M30 bullish FVG [485.2000, 491.2500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 14 | 2024-05-22 10:00 | 463.9500 | 449.6900 | 2024-05-22 11:30 | 465.9000 | STOCH_EXIT | 3 | 313.35 | 0.32% | 466.4500 | 492.4699 | 14.2600 | M30 bullish FVG [462.6000, 466.4500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 15 | 2024-06-10 18:30 | 456.2500 | 441.0179 | 2024-06-11 19:00 | 455.2500 | STOCH_EXIT | 30 | -312.99 | -0.32% | 464.9000 | 486.7143 | 15.2321 | M30 bullish FVG [450.9500, 464.9000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 16 | 2024-06-21 11:30 | 443.1500 | 425.7915 | 2024-06-22 12:30 | 441.7500 | STOCH_EXIT | 30 | -407.18 | -0.42% | 441.5500 | 477.8671 | 17.3585 | M30 bullish FVG [427.6500, 441.5500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 17 | 2024-08-30 11:00 | 337.6500 | 325.9296 | 2024-08-30 19:00 | 325.9296 | SL | 16 | -3483.08 | -3.57% | 336.8000 | 361.0908 | 11.7204 | M30 bullish FVG [332.2000, 336.8000], stochastic crossed below 20 | FVG stop 325.9296 |
| 18 | 2024-10-09 11:30 | 329.6500 | 318.7111 | 2024-10-09 22:00 | 328.7500 | STOCH_EXIT | 21 | -350.32 | -0.37% | 332.0500 | 351.5279 | 10.9389 | M30 bullish FVG [328.4000, 332.0500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 19 | 2024-10-23 20:30 | 330.5000 | 320.5360 | 2024-10-24 11:00 | 332.4000 | STOCH_EXIT | 15 | 443.90 | 0.47% | 333.0500 | 350.4280 | 9.9640 | M30 bullish FVG [330.3500, 333.0500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 20 | 2024-12-19 18:00 | 290.7000 | 279.2386 | 2024-12-20 07:00 | 293.8000 | STOCH_EXIT | 12 | 909.71 | 0.97% | 298.2000 | 313.6229 | 11.4614 | M30 bullish FVG [289.4500, 298.2000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 21 | 2024-12-26 14:30 | 323.1500 | 311.0118 | 2024-12-26 23:30 | 325.1500 | STOCH_EXIT | 18 | 492.70 | 0.52% | 318.5500 | 347.4264 | 12.1382 | M30 bullish FVG [310.3500, 318.5500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 22 | 2024-12-30 10:00 | 338.1500 | 326.7484 | 2024-12-30 12:30 | 343.1500 | STOCH_EXIT | 5 | 1313.94 | 1.38% | 337.5000 | 360.9532 | 11.4016 | M30 bullish FVG [334.4000, 337.5000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 23 | 2025-01-22 20:00 | 373.1500 | 360.2192 | 2025-01-23 23:30 | 374.6000 | STOCH_EXIT | 41 | 278.72 | 0.29% | 378.0000 | 399.0116 | 12.9308 | M30 bullish FVG [373.6000, 378.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 24 | 2025-02-06 12:00 | 378.2000 | 366.4356 | 2025-02-06 21:00 | 379.4000 | STOCH_EXIT | 18 | 211.05 | 0.22% | 371.5000 | 401.7288 | 11.7644 | M30 bullish FVG [368.4000, 371.5000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 25 | 2025-02-13 11:30 | 429.5000 | 413.9965 | 2025-02-14 10:00 | 438.2000 | STOCH_EXIT | 25 | 1868.15 | 1.92% | 448.8000 | 460.5070 | 15.5035 | M30 bullish FVG [438.9500, 448.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 26 | 2025-02-25 13:00 | 501.2000 | 481.1118 | 2025-02-26 06:30 | 502.6000 | STOCH_EXIT | 22 | 177.82 | 0.18% | 503.0500 | 541.3763 | 20.0882 | M30 bullish FVG [495.0000, 503.0500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 27 | 2025-04-10 12:00 | 398.7000 | 365.0241 | 2025-04-10 23:00 | 409.2500 | STOCH_EXIT | 22 | 2526.36 | 2.54% | 411.0000 | 466.0518 | 33.6759 | M30 bullish FVG [400.9500, 411.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 28 | 2025-06-04 16:30 | 393.3000 | 373.6121 | 2025-06-04 23:00 | 395.4000 | STOCH_EXIT | 13 | 441.76 | 0.43% | 391.3000 | 432.6757 | 19.6879 | M30 bullish FVG [386.5500, 391.3000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 29 | 2025-07-01 08:00 | 399.8500 | 384.3153 | 2025-07-01 17:30 | 401.4000 | STOCH_EXIT | 19 | 294.24 | 0.29% | 392.0000 | 430.9193 | 15.5347 | M30 bullish FVG [386.8000, 392.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 30 | 2025-07-04 23:30 | 404.5500 | 389.8045 | 2025-07-05 07:30 | 408.0000 | STOCH_EXIT | 9 | 773.11 | 0.75% | 403.0500 | 434.0410 | 14.7455 | M30 bullish FVG [398.4000, 403.0500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 31 | 2025-07-15 15:30 | 393.2000 | 377.6459 | 2025-07-16 06:30 | 395.3500 | STOCH_EXIT | 17 | 461.76 | 0.45% | 393.9000 | 424.3082 | 15.5541 | M30 bullish FVG [389.7500, 393.9000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 32 | 2025-07-23 18:00 | 436.1000 | 419.3763 | 2025-07-24 20:00 | 435.7000 | STOCH_EXIT | 39 | -198.94 | -0.19% | 437.0000 | 469.5474 | 16.7237 | M30 bullish FVG [428.5500, 437.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 33 | 2025-07-29 21:00 | 430.5000 | 414.5000 | 2025-07-31 08:00 | 423.1500 | STOCH_EXIT | 44 | -1874.21 | -1.81% | 429.2500 | 462.5001 | 16.0000 | M30 bullish FVG [421.3500, 429.2500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 34 | 2025-09-24 22:00 | 402.2000 | 385.9231 | 2025-09-25 11:30 | 404.1500 | STOCH_EXIT | 13 | 391.35 | 0.38% | 400.9000 | 434.7538 | 16.2769 | M30 bullish FVG [396.7500, 400.9000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 35 | 2025-09-29 18:30 | 404.2500 | 388.3707 | 2025-09-30 13:00 | 399.4500 | STOCH_EXIT | 23 | -1316.07 | -1.29% | 406.7500 | 436.0086 | 15.8793 | M30 bullish FVG [402.5500, 406.7500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 36 | 2025-10-07 07:00 | 401.0000 | 385.0765 | 2025-10-07 10:30 | 405.1000 | STOCH_EXIT | 7 | 927.93 | 0.92% | 397.2000 | 432.8471 | 15.9235 | M30 bullish FVG [391.6500, 397.2000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 37 | 2025-10-10 10:30 | 406.2500 | 388.6532 | 2025-10-11 17:00 | 397.9500 | STOCH_EXIT | 55 | -2175.52 | -2.14% | 397.3500 | 441.4436 | 17.5968 | M30 bullish FVG [386.2000, 397.3500], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 93107.32
- Макс: 104608.22
- Финал: 99806.21

Полная кривая — в `*_equity.csv`.
