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
| StochPeriod | 10 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 1 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 30 |
| Выигрышных / проигрышных | 19 / 11 |
| Win rate | 63.33% |
| Gross profit / loss | 31284.43 / 6395.51 |
| Profit factor | 4.892 |
| Чистый PnL | 24888.92 (24.89%) |
| Стартовый / финальный капитал | 100000.00 / 124888.92 |
| Макс. просадка | 5660.20 (4.43%) |
| Средняя прибыль / убыток | 1646.55 / 581.41 |
| Expectancy | 829.63 |
| Лучшая / худшая сделка | 5167.33 / -3001.66 |
| Exposure | 1.71% |
| CAGR | 11.74% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-12-13 17:00 | 426.0000 | 412.3863 | 2023-12-14 09:30 | 433.0000 | STOCH_EXIT | 14 | 1537.50 | 1.54% | 427.0000 | 439.6137 | 13.6137 | M30 bullish FVG [422.0000, 427.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 2 | 2023-12-14 15:30 | 433.8000 | 420.0158 | 2023-12-15 10:00 | 431.8000 | STOCH_EXIT | 18 | -566.84 | -0.56% | 431.4000 | 447.5842 | 13.7842 | M30 bullish FVG [425.2000, 431.4000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 3 | 2024-01-17 12:00 | 475.2000 | 463.2160 | 2024-01-17 20:00 | 474.4000 | STOCH_EXIT | 16 | -270.26 | -0.27% | 466.2000 | 487.1840 | 11.9840 | M30 bullish FVG [462.0000, 466.2000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 4 | 2024-03-04 23:30 | 449.6000 | 440.0201 | 2024-03-05 16:00 | 448.2000 | STOCH_EXIT | 14 | -412.30 | -0.41% | 446.0000 | 459.1799 | 9.5799 | M30 bullish FVG [442.8000, 446.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 5 | 2024-03-26 10:30 | 432.0000 | 421.3536 | 2024-03-26 15:30 | 431.8000 | STOCH_EXIT | 10 | -146.60 | -0.15% | 430.8000 | 442.6464 | 10.6464 | M30 bullish FVG [427.6000, 430.8000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 6 | 2024-03-29 11:00 | 427.8000 | 418.0174 | 2024-03-29 16:00 | 428.0000 | STOCH_EXIT | 10 | -53.10 | -0.05% | 431.4000 | 437.5826 | 9.7826 | M30 bullish FVG [428.0000, 431.4000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 7 | 2024-04-04 22:30 | 470.0000 | 459.1738 | 2024-04-05 16:30 | 466.6000 | STOCH_EXIT | 17 | -820.08 | -0.82% | 473.8000 | 480.8262 | 10.8262 | M30 bullish FVG [468.0000, 473.8000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 8 | 2024-04-09 14:30 | 508.2000 | 493.0800 | 2024-04-09 19:00 | 514.8000 | STOCH_EXIT | 9 | 1187.26 | 1.20% | 506.4000 | 523.3200 | 15.1200 | M30 bullish FVG [502.0000, 506.4000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 9 | 2024-05-10 17:00 | 480.9000 | 466.5893 | 2024-05-11 10:00 | 478.5000 | STOCH_EXIT | 14 | -598.98 | -0.60% | 479.7000 | 495.2107 | 14.3107 | M30 bullish FVG [471.1000, 479.7000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 10 | 2024-05-14 23:30 | 482.1500 | 467.8331 | 2024-05-15 10:00 | 492.3500 | STOCH_EXIT | 2 | 2010.54 | 2.01% | 486.5500 | 496.4669 | 14.3169 | M30 bullish FVG [481.5000, 486.5500], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 11 | 2024-05-15 17:00 | 486.0000 | 471.6700 | 2024-05-15 22:00 | 486.4500 | STOCH_EXIT | 10 | -7.57 | -0.01% | 491.2500 | 500.3300 | 14.3300 | M30 bullish FVG [485.2000, 491.2500], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 12 | 2024-06-10 15:30 | 456.5000 | 441.2679 | 2024-06-11 19:00 | 455.2500 | STOCH_EXIT | 36 | -380.41 | -0.37% | 464.9000 | 471.7321 | 15.2321 | M30 bullish FVG [450.9500, 464.9000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 13 | 2024-06-20 21:30 | 442.9000 | 426.6523 | 2024-06-20 23:30 | 447.6000 | STOCH_EXIT | 4 | 974.34 | 0.96% | 441.5500 | 459.1477 | 16.2477 | M30 bullish FVG [427.6500, 441.5500], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 14 | 2024-08-29 14:30 | 333.7500 | 322.2511 | 2024-08-29 18:00 | 339.4000 | STOCH_EXIT | 7 | 1625.91 | 1.59% | 336.8000 | 345.2489 | 11.4989 | M30 bullish FVG [332.2000, 336.8000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 15 | 2024-12-19 18:00 | 290.7000 | 279.2386 | 2024-12-20 07:00 | 293.8000 | STOCH_EXIT | 12 | 1002.37 | 0.97% | 298.2000 | 302.1614 | 11.4614 | M30 bullish FVG [289.4500, 298.2000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 16 | 2024-12-26 12:30 | 322.7000 | 310.5618 | 2024-12-26 23:30 | 325.1500 | STOCH_EXIT | 22 | 690.97 | 0.66% | 318.5500 | 334.8382 | 12.1382 | M30 bullish FVG [310.3500, 318.5500], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 17 | 2025-01-22 19:30 | 374.1500 | 361.2192 | 2025-01-23 07:00 | 387.0808 | TP | 9 | 3539.15 | 3.35% | 378.0000 | 387.0808 | 12.9308 | M30 bullish FVG [373.6000, 378.0000], stochastic crossed below 20 | FVG target 387.0808 |
| 18 | 2025-02-10 23:00 | 409.1500 | 398.1585 | 2025-02-11 07:00 | 420.1415 | TP | 3 | 2824.02 | 2.58% | 403.2500 | 420.1415 | 10.9915 | M30 bullish FVG [394.0000, 403.2500], stochastic crossed below 20 | FVG target 420.1415 |
| 19 | 2025-02-13 10:30 | 431.9500 | 416.4465 | 2025-02-14 10:00 | 438.2000 | STOCH_EXIT | 27 | 1506.07 | 1.35% | 448.8000 | 447.4535 | 15.5035 | M30 bullish FVG [438.9500, 448.8000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 20 | 2025-02-25 12:30 | 500.4500 | 480.3618 | 2025-02-25 15:00 | 507.3000 | STOCH_EXIT | 5 | 1434.22 | 1.27% | 503.0500 | 520.5382 | 20.0882 | M30 bullish FVG [495.0000, 503.0500], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 21 | 2025-04-10 12:00 | 398.7000 | 365.0241 | 2025-04-10 15:00 | 417.0500 | STOCH_EXIT | 6 | 5167.33 | 4.50% | 391.5000 | 432.3759 | 33.6759 | M30 bullish FVG [380.0000, 391.5000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 22 | 2025-05-06 14:00 | 392.8000 | 369.4592 | 2025-05-06 15:30 | 399.1500 | STOCH_EXIT | 3 | 1815.98 | 1.52% | 395.6000 | 416.1408 | 23.3408 | M30 bullish FVG [386.4500, 395.6000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 23 | 2025-07-01 08:00 | 399.8500 | 384.3153 | 2025-07-01 17:30 | 401.4000 | STOCH_EXIT | 19 | 350.56 | 0.29% | 392.0000 | 415.3847 | 15.5347 | M30 bullish FVG [386.8000, 392.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 24 | 2025-07-04 19:30 | 405.1000 | 390.3545 | 2025-07-05 07:30 | 408.0000 | STOCH_EXIT | 17 | 753.02 | 0.62% | 403.0500 | 419.8455 | 14.7455 | M30 bullish FVG [398.4000, 403.0500], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 25 | 2025-07-15 11:00 | 393.3000 | 377.7459 | 2025-07-16 06:30 | 395.3500 | STOCH_EXIT | 26 | 516.57 | 0.42% | 386.5500 | 408.8541 | 15.5541 | M30 bullish FVG [377.9000, 386.5500], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 26 | 2025-07-23 17:00 | 436.6500 | 419.9263 | 2025-07-24 09:00 | 436.6000 | STOCH_EXIT | 19 | -137.71 | -0.11% | 437.0000 | 453.3737 | 16.7237 | M30 bullish FVG [428.5500, 437.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 27 | 2025-07-29 21:00 | 430.5000 | 414.5000 | 2025-07-30 09:30 | 433.5000 | STOCH_EXIT | 12 | 734.45 | 0.60% | 429.2500 | 446.5000 | 16.0000 | M30 bullish FVG [421.3500, 429.2500], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 28 | 2025-10-06 15:30 | 395.3500 | 379.6593 | 2025-10-06 16:30 | 402.7500 | STOCH_EXIT | 2 | 2198.30 | 1.77% | 397.2000 | 411.0407 | 15.6907 | M30 bullish FVG [391.6500, 397.2000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 29 | 2025-10-10 10:30 | 406.2500 | 388.6532 | 2025-10-11 08:00 | 397.0000 | STOCH_EXIT | 37 | -3001.66 | -2.37% | 397.3500 | 423.8468 | 17.5968 | M30 bullish FVG [386.2000, 397.3500], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 30 | 2025-10-17 11:00 | 412.0500 | 393.3785 | 2025-10-17 12:30 | 417.2000 | STOCH_EXIT | 3 | 1415.88 | 1.15% | 406.2000 | 430.7215 | 18.6715 | M30 bullish FVG [392.7500, 406.2000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 97229.76
- Макс: 127764.37
- Финал: 124888.92

Полная кривая — в `*_equity.csv`.
