# Бэктест ENPG (Minutes30)

- Период: 2023-11-13 23:20 — 2025-11-13 23:20
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.3 |
| MaxWaitBars | 30 |
| StochPeriod | 11 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 2 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 1 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 28 |
| Выигрышных / проигрышных | 19 / 9 |
| Win rate | 67.86% |
| Gross profit / loss | 25575.99 / 6317.93 |
| Profit factor | 4.048 |
| Чистый PnL | 19258.05 (19.26%) |
| Стартовый / финальный капитал | 100000.00 / 119258.05 |
| Макс. просадка | 7154.00 (6.11%) |
| Средняя прибыль / убыток | 1346.10 / 701.99 |
| Expectancy | 687.79 |
| Лучшая / худшая сделка | 3752.69 / -2573.11 |
| Exposure | 1.87% |
| CAGR | 9.19% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-12-13 17:00 | 426.0000 | 412.3863 | 2023-12-14 10:00 | 435.4000 | STOCH_EXIT | 15 | 2098.82 | 2.10% | 427.0000 | 453.2275 | 13.6137 | M30 bullish FVG [422.0000, 427.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 2 | 2023-12-14 16:00 | 433.2000 | 419.4158 | 2023-12-15 10:00 | 431.8000 | STOCH_EXIT | 17 | -430.64 | -0.42% | 431.4000 | 460.7684 | 13.7842 | M30 bullish FVG [425.2000, 431.4000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 3 | 2023-12-18 16:00 | 445.8000 | 431.9728 | 2023-12-18 17:30 | 447.4000 | STOCH_EXIT | 3 | 261.82 | 0.26% | 445.6000 | 473.4544 | 13.8272 | M30 bullish FVG [438.8000, 445.6000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 4 | 2024-01-17 12:00 | 475.2000 | 463.2160 | 2024-01-17 20:00 | 474.4000 | STOCH_EXIT | 16 | -272.81 | -0.27% | 466.2000 | 499.1679 | 11.9840 | M30 bullish FVG [462.0000, 466.2000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 5 | 2024-03-04 23:30 | 449.6000 | 440.0201 | 2024-03-05 16:00 | 448.2000 | STOCH_EXIT | 14 | -416.00 | -0.41% | 446.0000 | 468.7598 | 9.5799 | M30 bullish FVG [442.8000, 446.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 6 | 2024-03-26 10:30 | 432.0000 | 421.3536 | 2024-03-26 15:30 | 431.8000 | STOCH_EXIT | 10 | -147.86 | -0.15% | 430.8000 | 453.2929 | 10.6464 | M30 bullish FVG [427.6000, 430.8000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 7 | 2024-04-04 23:00 | 469.4000 | 458.5738 | 2024-04-05 16:30 | 466.6000 | STOCH_EXIT | 16 | -702.62 | -0.70% | 473.8000 | 491.0524 | 10.8262 | M30 bullish FVG [468.0000, 473.8000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 8 | 2024-04-09 14:30 | 508.2000 | 493.0800 | 2024-04-09 19:00 | 514.8000 | STOCH_EXIT | 9 | 1199.43 | 1.20% | 506.4000 | 538.4400 | 15.1200 | M30 bullish FVG [502.0000, 506.4000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 9 | 2024-05-10 17:30 | 480.7500 | 466.4393 | 2024-05-11 10:00 | 478.5000 | STOCH_EXIT | 13 | -575.95 | -0.57% | 479.7000 | 509.3714 | 14.3107 | M30 bullish FVG [471.1000, 479.7000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 10 | 2024-05-14 23:30 | 482.1500 | 467.8331 | 2024-05-15 10:00 | 492.3500 | STOCH_EXIT | 2 | 2029.96 | 2.01% | 486.5500 | 510.7838 | 14.3169 | M30 bullish FVG [481.5000, 486.5500], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 11 | 2024-05-15 17:00 | 486.0000 | 471.6700 | 2024-05-15 22:30 | 486.7500 | STOCH_EXIT | 11 | 55.62 | 0.05% | 491.2500 | 514.6599 | 14.3300 | M30 bullish FVG [485.2000, 491.2500], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 12 | 2024-06-10 16:00 | 458.6500 | 443.4179 | 2024-06-11 19:00 | 455.2500 | STOCH_EXIT | 35 | -863.96 | -0.84% | 464.9000 | 489.1143 | 15.2321 | M30 bullish FVG [450.9500, 464.9000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 13 | 2024-06-20 21:30 | 442.9000 | 426.6523 | 2024-06-20 23:30 | 447.6000 | STOCH_EXIT | 4 | 978.59 | 0.96% | 441.5500 | 475.3953 | 16.2477 | M30 bullish FVG [427.6500, 441.5500], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 14 | 2024-08-29 15:00 | 334.0000 | 322.5011 | 2024-08-29 18:00 | 339.4000 | STOCH_EXIT | 6 | 1559.50 | 1.52% | 336.8000 | 356.9978 | 11.4989 | M30 bullish FVG [332.2000, 336.8000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 15 | 2024-12-19 18:00 | 290.7000 | 279.2386 | 2024-12-20 07:00 | 293.8000 | STOCH_EXIT | 12 | 1010.79 | 0.97% | 298.2000 | 313.6229 | 11.4614 | M30 bullish FVG [289.4500, 298.2000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 16 | 2024-12-26 13:00 | 322.9000 | 310.7618 | 2024-12-26 23:30 | 325.1500 | STOCH_EXIT | 21 | 629.79 | 0.60% | 318.5500 | 347.1764 | 12.1382 | M30 bullish FVG [310.3500, 318.5500], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 17 | 2025-01-22 19:30 | 374.1500 | 361.2192 | 2025-01-23 23:30 | 374.6000 | STOCH_EXIT | 42 | 21.48 | 0.02% | 378.0000 | 400.0116 | 12.9308 | M30 bullish FVG [373.6000, 378.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 18 | 2025-02-10 23:00 | 409.1500 | 398.1585 | 2025-02-11 10:30 | 424.0000 | STOCH_EXIT | 10 | 3752.69 | 3.53% | 403.2500 | 431.1330 | 10.9915 | M30 bullish FVG [394.0000, 403.2500], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 19 | 2025-02-25 12:30 | 500.4500 | 480.3618 | 2025-02-25 16:00 | 509.0000 | STOCH_EXIT | 7 | 1769.96 | 1.61% | 503.0500 | 540.6263 | 20.0882 | M30 bullish FVG [495.0000, 503.0500], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 20 | 2025-04-10 12:00 | 398.7000 | 365.0241 | 2025-04-10 22:00 | 408.8000 | STOCH_EXIT | 20 | 2714.95 | 2.43% | 391.5000 | 466.0518 | 33.6759 | M30 bullish FVG [380.0000, 391.5000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 21 | 2025-05-06 14:00 | 392.8000 | 369.4592 | 2025-05-06 17:30 | 404.1000 | STOCH_EXIT | 7 | 3172.35 | 2.77% | 395.6000 | 439.4817 | 23.3408 | M30 bullish FVG [386.4500, 395.6000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 22 | 2025-07-01 08:00 | 399.8500 | 384.3153 | 2025-07-01 17:30 | 401.4000 | STOCH_EXIT | 19 | 337.92 | 0.29% | 392.0000 | 430.9193 | 15.5347 | M30 bullish FVG [386.8000, 392.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 23 | 2025-07-15 11:00 | 393.3000 | 377.7459 | 2025-07-16 06:30 | 395.3500 | STOCH_EXIT | 26 | 496.70 | 0.42% | 386.5500 | 424.4082 | 15.5541 | M30 bullish FVG [377.9000, 386.5500], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 24 | 2025-07-23 17:30 | 436.5000 | 419.7763 | 2025-07-24 20:00 | 435.7000 | STOCH_EXIT | 40 | -334.98 | -0.28% | 437.0000 | 469.9474 | 16.7237 | M30 bullish FVG [428.5500, 437.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 25 | 2025-07-29 21:00 | 430.5000 | 414.5000 | 2025-07-30 09:30 | 433.5000 | STOCH_EXIT | 12 | 703.63 | 0.60% | 429.2500 | 462.5001 | 16.0000 | M30 bullish FVG [421.3500, 429.2500], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 26 | 2025-10-07 07:00 | 401.0000 | 385.0765 | 2025-10-07 08:30 | 404.1000 | STOCH_EXIT | 3 | 798.45 | 0.67% | 397.2000 | 432.8471 | 15.9235 | M30 bullish FVG [391.6500, 397.2000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 27 | 2025-10-10 10:30 | 406.2500 | 388.6532 | 2025-10-11 08:30 | 397.9000 | STOCH_EXIT | 38 | -2573.11 | -2.15% | 397.3500 | 441.4436 | 17.5968 | M30 bullish FVG [386.2000, 397.3500], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 28 | 2025-10-17 11:00 | 412.0500 | 393.3785 | 2025-10-17 18:00 | 419.4500 | STOCH_EXIT | 14 | 1983.53 | 1.69% | 406.2000 | 449.3929 | 18.6715 | M30 bullish FVG [392.7500, 406.2000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 98331.25
- Макс: 121066.82
- Финал: 119258.05

Полная кривая — в `*_equity.csv`.
