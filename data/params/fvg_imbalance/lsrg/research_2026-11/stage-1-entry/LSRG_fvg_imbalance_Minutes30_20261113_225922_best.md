# Бэктест LSRG (Minutes30)

- Период: 2023-11-13 22:59 — 2025-11-13 22:59
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.3 |
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
| Всего сделок | 24 |
| Выигрышных / проигрышных | 18 / 6 |
| Win rate | 75.00% |
| Gross profit / loss | 20067.93 / 13262.35 |
| Profit factor | 1.513 |
| Чистый PnL | 6805.59 (6.81%) |
| Стартовый / финальный капитал | 100000.00 / 106805.59 |
| Макс. просадка | 8514.33 (8.19%) |
| Средняя прибыль / убыток | 1114.89 / 2210.39 |
| Expectancy | 283.57 |
| Лучшая / худшая сделка | 3570.01 / -4085.73 |
| Exposure | 2.11% |
| CAGR | 3.34% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-01-09 11:30 | 675.2000 | 652.9419 | 2024-01-09 16:00 | 679.0000 | STOCH_EXIT | 9 | 462.19 | 0.46% | 665.8000 | 719.7162 | 22.2581 | M30 bullish FVG [659.2000, 665.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 2 | 2024-01-17 18:00 | 715.0000 | 692.3623 | 2024-01-18 12:30 | 718.8000 | STOCH_EXIT | 8 | 431.63 | 0.43% | 712.0000 | 760.2754 | 22.6377 | M30 bullish FVG [703.2000, 712.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 3 | 2024-01-23 10:30 | 745.6000 | 723.1777 | 2024-01-24 17:30 | 747.4000 | STOCH_EXIT | 33 | 142.22 | 0.14% | 739.4000 | 790.4446 | 22.4223 | M30 bullish FVG [729.8000, 739.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 4 | 2024-02-05 15:00 | 757.0000 | 737.9687 | 2024-02-06 18:00 | 760.2000 | STOCH_EXIT | 25 | 324.71 | 0.32% | 750.4000 | 795.0626 | 19.0313 | M30 bullish FVG [735.8000, 750.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 5 | 2024-02-08 12:30 | 767.2000 | 747.6430 | 2024-02-10 10:30 | 742.2000 | SL | 32 | -3399.62 | -3.36% | 767.0000 | 806.3140 | 19.5570 | M30 bullish FVG [759.6000, 767.0000], stochastic crossed below 20 | FVG stop 747.6430 |
| 6 | 2024-02-14 17:00 | 811.0000 | 789.8722 | 2024-02-15 11:00 | 826.0000 | STOCH_EXIT | 7 | 1701.78 | 1.75% | 809.8000 | 853.2556 | 21.1278 | M30 bullish FVG [789.4000, 809.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 7 | 2024-02-15 16:00 | 816.4000 | 793.6099 | 2024-02-16 18:00 | 817.4000 | STOCH_EXIT | 23 | 22.34 | 0.02% | 824.2000 | 861.9802 | 22.7901 | M30 bullish FVG [813.0000, 824.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 8 | 2024-02-27 11:30 | 868.0000 | 838.1807 | 2024-02-27 12:00 | 900.2000 | STOCH_EXIT | 1 | 3570.01 | 3.61% | 862.8000 | 927.6386 | 29.8193 | M30 bullish FVG [854.6000, 862.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 9 | 2024-02-28 16:00 | 1015.4000 | 980.4250 | 2024-02-29 17:00 | 980.4250 | SL | 21 | -3633.27 | -3.54% | 1023.8000 | 1085.3501 | 34.9750 | M30 bullish FVG [974.0000, 1023.8000], stochastic crossed below 20 | FVG stop 980.4250 |
| 10 | 2024-05-28 16:30 | 790.6000 | 758.3286 | 2024-05-29 11:00 | 802.0000 | STOCH_EXIT | 8 | 1325.46 | 1.34% | 797.6000 | 855.1428 | 32.2714 | M30 bullish FVG [782.6000, 797.6000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 11 | 2024-06-06 13:30 | 806.2000 | 770.9987 | 2024-06-07 10:00 | 812.4000 | STOCH_EXIT | 12 | 673.84 | 0.67% | 830.2000 | 876.6025 | 35.2013 | M30 bullish FVG [818.8000, 830.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 12 | 2024-07-25 11:00 | 763.8000 | 733.5962 | 2024-07-26 15:00 | 733.5962 | SL | 27 | -4085.73 | -4.05% | 750.6000 | 824.2075 | 30.2038 | M30 bullish FVG [741.8000, 750.6000], stochastic crossed below 20 | FVG stop 733.5962 |
| 13 | 2024-09-03 15:00 | 595.2000 | 565.7685 | 2024-09-03 20:00 | 598.8000 | STOCH_EXIT | 10 | 489.49 | 0.50% | 614.2000 | 654.0630 | 29.4315 | M30 bullish FVG [602.0000, 614.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 14 | 2024-11-06 16:00 | 730.2000 | 709.6481 | 2024-11-07 17:30 | 730.0000 | STOCH_EXIT | 32 | -124.63 | -0.13% | 744.8000 | 771.3038 | 20.5519 | M30 bullish FVG [736.0000, 744.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 15 | 2024-11-15 09:00 | 783.4000 | 759.1614 | 2024-11-15 18:00 | 774.4000 | STOCH_EXIT | 18 | -1212.58 | -1.25% | 785.8000 | 831.8771 | 24.2386 | M30 bullish FVG [778.0000, 785.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 16 | 2024-11-28 10:30 | 705.0000 | 674.3256 | 2024-11-28 17:30 | 725.6000 | STOCH_EXIT | 14 | 2724.20 | 2.82% | 686.6000 | 766.3489 | 30.6744 | M30 bullish FVG [670.0000, 686.6000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 17 | 2024-12-30 20:00 | 832.6000 | 798.3465 | 2024-12-30 22:30 | 844.6000 | STOCH_EXIT | 5 | 1328.21 | 1.34% | 820.4000 | 901.1070 | 34.2535 | M30 bullish FVG [810.0000, 820.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 18 | 2025-06-18 17:30 | 742.8000 | 717.6960 | 2025-06-18 23:30 | 748.0000 | STOCH_EXIT | 12 | 601.37 | 0.60% | 746.6000 | 793.0081 | 25.1040 | M30 bullish FVG [739.0000, 746.6000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 19 | 2025-07-01 12:00 | 771.0000 | 747.8126 | 2025-07-03 08:00 | 772.4000 | STOCH_EXIT | 62 | 82.31 | 0.08% | 772.4000 | 817.3747 | 23.1874 | M30 bullish FVG [765.6000, 772.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 20 | 2025-07-15 11:00 | 745.2000 | 723.2876 | 2025-07-16 07:30 | 757.4000 | STOCH_EXIT | 27 | 1557.02 | 1.54% | 732.8000 | 789.0248 | 21.9124 | M30 bullish FVG [722.4000, 732.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 21 | 2025-07-16 21:00 | 765.4000 | 743.7099 | 2025-07-17 06:30 | 770.0000 | STOCH_EXIT | 6 | 513.53 | 0.50% | 760.2000 | 808.7802 | 21.6901 | M30 bullish FVG [750.0000, 760.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 22 | 2025-07-28 17:30 | 761.4000 | 740.6773 | 2025-07-29 09:30 | 773.0000 | STOCH_EXIT | 19 | 1462.43 | 1.42% | 762.4000 | 802.8455 | 20.7227 | M30 bullish FVG [752.4000, 762.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 23 | 2025-08-07 17:00 | 776.8000 | 756.1549 | 2025-08-08 08:00 | 771.6000 | STOCH_EXIT | 16 | -806.52 | -0.77% | 780.0000 | 818.0902 | 20.6451 | M30 bullish FVG [771.8000, 780.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 24 | 2025-08-11 19:30 | 806.4000 | 786.3407 | 2025-08-12 07:30 | 827.8000 | STOCH_EXIT | 11 | 2655.19 | 2.55% | 809.0000 | 846.5186 | 20.0593 | M30 bullish FVG [794.4000, 809.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 95396.65
- Макс: 106805.59
- Финал: 106805.59

Полная кривая — в `*_equity.csv`.
