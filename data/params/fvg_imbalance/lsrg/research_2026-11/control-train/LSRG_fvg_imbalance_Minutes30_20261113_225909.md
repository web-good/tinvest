# Бэктест LSRG (Minutes30)

- Период: 2023-11-13 22:59 — 2025-11-13 22:59
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
| Всего сделок | 34 |
| Выигрышных / проигрышных | 22 / 12 |
| Win rate | 64.71% |
| Gross profit / loss | 18248.35 / 25901.63 |
| Profit factor | 0.705 |
| Чистый PnL | -7653.28 (-7.65%) |
| Стартовый / финальный капитал | 100000.00 / 92346.72 |
| Макс. просадка | 9913.66 (9.91%) |
| Средняя прибыль / убыток | 829.47 / 2158.47 |
| Expectancy | -225.10 |
| Лучшая / худшая сделка | 2585.01 / -4507.05 |
| Exposure | 3.00% |
| CAGR | -3.90% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-12-04 18:00 | 643.8000 | 622.2108 | 2023-12-06 11:00 | 622.2108 | SL | 24 | -3444.45 | -3.45% | 647.0000 | 686.9784 | 21.5892 | M30 bullish FVG [641.6000, 647.0000], stochastic crossed below 20 | FVG stop 622.2108 |
| 2 | 2024-01-09 11:30 | 675.2000 | 652.9419 | 2024-01-09 18:30 | 680.0000 | STOCH_EXIT | 14 | 585.38 | 0.61% | 665.8000 | 719.7162 | 22.2581 | M30 bullish FVG [659.2000, 665.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 3 | 2024-01-17 18:00 | 715.0000 | 692.3623 | 2024-01-18 14:00 | 722.8000 | STOCH_EXIT | 11 | 955.95 | 0.99% | 712.0000 | 760.2754 | 22.6377 | M30 bullish FVG [703.2000, 712.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 4 | 2024-01-23 10:30 | 745.6000 | 723.1777 | 2024-01-24 17:30 | 747.4000 | STOCH_EXIT | 33 | 138.01 | 0.14% | 739.4000 | 790.4446 | 22.4223 | M30 bullish FVG [729.8000, 739.4000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 5 | 2024-02-08 12:30 | 767.2000 | 747.6430 | 2024-02-10 10:30 | 742.2000 | SL | 32 | -3270.85 | -3.36% | 767.0000 | 806.3140 | 19.5570 | M30 bullish FVG [759.6000, 767.0000], stochastic crossed below 20 | FVG stop 747.6430 |
| 6 | 2024-02-15 10:00 | 808.8000 | 786.0099 | 2024-02-15 11:00 | 826.0000 | STOCH_EXIT | 2 | 1916.76 | 2.02% | 809.8000 | 854.3802 | 22.7901 | M30 bullish FVG [789.4000, 809.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 7 | 2024-02-15 18:00 | 814.0000 | 791.2099 | 2024-02-17 10:00 | 776.6000 | SL | 21 | -4507.05 | -4.69% | 824.2000 | 859.5802 | 22.7901 | M30 bullish FVG [813.0000, 824.2000], stochastic crossed below 20 | FVG stop 791.2099 |
| 8 | 2024-02-29 11:00 | 984.2000 | 945.2517 | 2024-03-01 12:30 | 992.6000 | STOCH_EXIT | 22 | 689.28 | 0.75% | 1023.8000 | 1062.0965 | 38.9483 | M30 bullish FVG [974.0000, 1023.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 9 | 2024-03-04 17:00 | 992.6000 | 952.8477 | 2024-03-05 14:00 | 1004.0000 | STOCH_EXIT | 13 | 967.36 | 1.05% | 996.2000 | 1072.1046 | 39.7523 | M30 bullish FVG [985.0000, 996.2000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 10 | 2024-04-02 14:30 | 997.8000 | 963.3275 | 2024-04-03 11:00 | 1005.8000 | STOCH_EXIT | 12 | 657.83 | 0.70% | 989.8000 | 1066.7451 | 34.4725 | M30 bullish FVG [979.4000, 989.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 11 | 2024-05-28 17:30 | 790.2000 | 757.9286 | 2024-05-30 11:00 | 800.0000 | STOCH_EXIT | 25 | 1071.58 | 1.14% | 797.6000 | 854.7428 | 32.2714 | M30 bullish FVG [782.6000, 797.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 12 | 2024-06-06 13:30 | 806.2000 | 770.9987 | 2024-06-07 10:00 | 812.4000 | STOCH_EXIT | 12 | 636.10 | 0.67% | 830.2000 | 876.6025 | 35.2013 | M30 bullish FVG [818.8000, 830.2000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 13 | 2024-07-17 14:00 | 706.2000 | 674.8775 | 2024-07-18 16:30 | 708.0000 | STOCH_EXIT | 24 | 148.63 | 0.15% | 715.2000 | 768.8451 | 31.3225 | M30 bullish FVG [706.6000, 715.2000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 14 | 2024-07-25 11:30 | 761.8000 | 731.5962 | 2024-07-26 15:00 | 731.5962 | SL | 26 | -3899.76 | -4.06% | 750.6000 | 822.2075 | 30.2038 | M30 bullish FVG [741.8000, 750.6000], stochastic crossed below 20 | FVG stop 731.5962 |
| 15 | 2024-08-12 12:00 | 688.6000 | 663.1624 | 2024-08-12 17:30 | 692.6000 | STOCH_EXIT | 11 | 443.46 | 0.48% | 703.8000 | 739.4752 | 25.4376 | M30 bullish FVG [696.2000, 703.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 16 | 2024-08-26 11:30 | 681.4000 | 656.7336 | 2024-08-26 21:30 | 669.8000 | STOCH_EXIT | 20 | -1669.48 | -1.80% | 685.8000 | 730.7327 | 24.6664 | M30 bullish FVG [678.6000, 685.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 17 | 2024-09-03 15:00 | 595.2000 | 565.7685 | 2024-09-03 23:00 | 598.2000 | STOCH_EXIT | 16 | 367.70 | 0.40% | 614.2000 | 654.0630 | 29.4315 | M30 bullish FVG [602.0000, 614.2000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 18 | 2024-09-04 10:00 | 591.4000 | 561.2565 | 2024-09-04 13:30 | 605.6000 | STOCH_EXIT | 7 | 2108.23 | 2.30% | 614.2000 | 651.6871 | 30.1435 | M30 bullish FVG [602.0000, 614.2000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 19 | 2024-09-24 10:30 | 700.0000 | 673.9913 | 2024-09-25 07:00 | 689.0000 | STOCH_EXIT | 27 | -1567.06 | -1.67% | 706.0000 | 752.0174 | 26.0087 | M30 bullish FVG [699.0000, 706.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 20 | 2024-09-25 19:00 | 690.8000 | 664.0776 | 2024-09-26 08:00 | 691.4000 | STOCH_EXIT | 11 | -12.12 | -0.01% | 691.6000 | 744.2448 | 26.7224 | M30 bullish FVG [684.2000, 691.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 21 | 2024-11-07 08:30 | 733.2000 | 711.5447 | 2024-11-07 18:00 | 735.0000 | STOCH_EXIT | 14 | 133.24 | 0.15% | 744.8000 | 776.5107 | 21.6553 | M30 bullish FVG [736.0000, 744.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 22 | 2024-11-15 11:30 | 783.4000 | 759.1614 | 2024-11-15 22:00 | 783.2000 | STOCH_EXIT | 21 | -115.05 | -0.13% | 785.8000 | 831.8771 | 24.2386 | M30 bullish FVG [778.0000, 785.8000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 23 | 2024-11-28 10:30 | 705.0000 | 674.3256 | 2024-11-28 17:30 | 725.6000 | STOCH_EXIT | 14 | 2585.01 | 2.82% | 686.6000 | 766.3489 | 30.6744 | M30 bullish FVG [670.0000, 686.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 24 | 2024-12-23 15:00 | 791.2000 | 757.6387 | 2024-12-24 10:00 | 757.6387 | SL | 24 | -4085.95 | -4.34% | 825.4000 | 858.3226 | 33.5613 | M30 bullish FVG [816.6000, 825.4000], stochastic crossed below 20 | FVG stop 757.6387 |
| 25 | 2025-06-18 17:30 | 742.8000 | 717.6960 | 2025-06-19 10:00 | 752.6000 | STOCH_EXIT | 20 | 1104.38 | 1.22% | 746.6000 | 793.0081 | 25.1040 | M30 bullish FVG [739.0000, 746.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 26 | 2025-07-01 12:00 | 771.0000 | 747.8126 | 2025-07-02 15:30 | 772.0000 | STOCH_EXIT | 42 | 27.19 | 0.03% | 772.4000 | 817.3747 | 23.1874 | M30 bullish FVG [765.6000, 772.4000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 27 | 2025-07-09 17:30 | 728.0000 | 707.8149 | 2025-07-09 23:00 | 733.0000 | STOCH_EXIT | 11 | 537.96 | 0.59% | 732.6000 | 768.3702 | 20.1851 | M30 bullish FVG [727.0000, 732.6000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 28 | 2025-07-15 15:30 | 747.4000 | 725.4876 | 2025-07-16 07:30 | 757.4000 | STOCH_EXIT | 18 | 1137.45 | 1.24% | 759.0000 | 791.2248 | 21.9124 | M30 bullish FVG [752.6000, 759.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 29 | 2025-07-17 08:30 | 763.0000 | 740.9021 | 2025-07-18 06:30 | 758.8000 | STOCH_EXIT | 31 | -605.23 | -0.65% | 760.2000 | 807.1959 | 22.0979 | M30 bullish FVG [750.0000, 760.2000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 30 | 2025-07-28 21:00 | 758.4000 | 737.6773 | 2025-07-29 09:30 | 773.0000 | STOCH_EXIT | 12 | 1687.78 | 1.82% | 762.4000 | 799.8455 | 20.7227 | M30 bullish FVG [752.4000, 762.4000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 31 | 2025-08-07 19:30 | 776.8000 | 756.1549 | 2025-08-08 10:30 | 776.8000 | STOCH_EXIT | 16 | -93.99 | -0.10% | 780.0000 | 818.0902 | 20.6451 | M30 bullish FVG [771.8000, 780.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 32 | 2025-08-14 23:00 | 807.0000 | 783.4908 | 2025-08-15 07:30 | 810.0000 | STOCH_EXIT | 3 | 256.41 | 0.27% | 810.0000 | 854.0184 | 23.5092 | M30 bullish FVG [804.0000, 810.0000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 33 | 2025-08-19 16:00 | 807.4000 | 783.4981 | 2025-08-19 22:30 | 809.0000 | STOCH_EXIT | 13 | 92.64 | 0.10% | 809.2000 | 855.2038 | 23.9019 | M30 bullish FVG [803.0000, 809.2000], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 34 | 2025-10-01 13:30 | 734.0000 | 714.3315 | 2025-10-02 07:30 | 714.3315 | SL | 22 | -2630.65 | -2.78% | 730.0000 | 773.3370 | 19.6685 | M30 bullish FVG [724.6000, 730.0000], stochastic crossed below 20 | FVG stop 714.3315 |

## Движение капитала

- Старт: 100000.00
- Мин: 90086.34
- Макс: 100000.00
- Финал: 92346.72

Полная кривая — в `*_equity.csv`.
