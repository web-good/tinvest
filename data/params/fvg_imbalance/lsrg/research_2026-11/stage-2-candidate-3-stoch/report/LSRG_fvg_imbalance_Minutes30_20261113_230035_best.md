# Бэктест LSRG (Minutes30)

- Период: 2023-11-13 23:00 — 2025-11-13 23:00
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.1 |
| MaxWaitBars | 30 |
| StochPeriod | 9 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 2 |
| HTFTrendEMA | 120 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 57 |
| Выигрышных / проигрышных | 36 / 21 |
| Win rate | 63.16% |
| Gross profit / loss | 31335.20 / 21108.48 |
| Profit factor | 1.484 |
| Чистый PnL | 10226.72 (10.23%) |
| Стартовый / финальный капитал | 100000.00 / 110226.72 |
| Макс. просадка | 7321.76 (7.06%) |
| Средняя прибыль / убыток | 870.42 / 1005.17 |
| Expectancy | 179.42 |
| Лучшая / худшая сделка | 3570.01 / -3633.27 |
| Exposure | 3.59% |
| CAGR | 4.98% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-01-09 11:30 | 675.2000 | 652.9419 | 2024-01-09 16:00 | 679.0000 | STOCH_EXIT | 9 | 462.19 | 0.46% | 672.6000 | 719.7162 | 22.2581 | M30 bullish FVG [667.4000, 672.6000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 2 | 2024-01-10 15:00 | 695.8000 | 673.3032 | 2024-01-12 10:00 | 692.0000 | STOCH_EXIT | 28 | -647.12 | -0.65% | 679.0000 | 740.7936 | 22.4968 | M30 bullish FVG [675.6000, 679.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 3 | 2024-01-17 18:00 | 715.0000 | 692.3623 | 2024-01-18 12:30 | 718.8000 | STOCH_EXIT | 8 | 428.55 | 0.43% | 712.0000 | 760.2754 | 22.6377 | M30 bullish FVG [703.2000, 712.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 4 | 2024-01-18 16:30 | 718.8000 | 696.1793 | 2024-01-19 09:30 | 722.6000 | STOCH_EXIT | 5 | 428.02 | 0.43% | 717.0000 | 764.0415 | 22.6207 | M30 bullish FVG [714.4000, 717.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 5 | 2024-01-23 10:30 | 745.6000 | 723.1777 | 2024-01-24 17:30 | 747.4000 | STOCH_EXIT | 33 | 141.17 | 0.14% | 739.4000 | 790.4446 | 22.4223 | M30 bullish FVG [729.8000, 739.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 6 | 2024-02-05 15:00 | 757.0000 | 737.9687 | 2024-02-06 18:00 | 760.2000 | STOCH_EXIT | 25 | 324.71 | 0.32% | 761.4000 | 795.0626 | 19.0313 | M30 bullish FVG [758.6000, 761.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 7 | 2024-02-08 12:30 | 767.2000 | 747.6430 | 2024-02-10 10:30 | 742.2000 | SL | 32 | -3373.87 | -3.36% | 770.8000 | 806.3140 | 19.5570 | M30 bullish FVG [768.8000, 770.8000], stochastic crossed below 20 | FVG stop 747.6430 |
| 8 | 2024-02-14 17:00 | 811.0000 | 789.8722 | 2024-02-15 11:00 | 826.0000 | STOCH_EXIT | 7 | 1701.78 | 1.75% | 820.8000 | 853.2556 | 21.1278 | M30 bullish FVG [817.6000, 820.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 9 | 2024-02-15 16:00 | 816.4000 | 793.6099 | 2024-02-16 18:00 | 817.4000 | STOCH_EXIT | 23 | 22.16 | 0.02% | 824.2000 | 861.9802 | 22.7901 | M30 bullish FVG [813.0000, 824.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 10 | 2024-02-27 11:30 | 868.0000 | 838.1807 | 2024-02-27 12:00 | 900.2000 | STOCH_EXIT | 1 | 3570.01 | 3.61% | 862.8000 | 927.6386 | 29.8193 | M30 bullish FVG [854.6000, 862.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 11 | 2024-02-28 16:00 | 1015.4000 | 980.4250 | 2024-02-29 17:00 | 980.4250 | SL | 21 | -3633.27 | -3.54% | 1023.8000 | 1085.3501 | 34.9750 | M30 bullish FVG [974.0000, 1023.8000], stochastic crossed below 20 | FVG stop 980.4250 |
| 12 | 2024-03-04 16:30 | 993.0000 | 953.2477 | 2024-03-05 14:00 | 1004.0000 | STOCH_EXIT | 14 | 1000.15 | 1.01% | 996.2000 | 1072.5046 | 39.7523 | M30 bullish FVG [985.0000, 996.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 13 | 2024-03-06 10:30 | 997.0000 | 956.9799 | 2024-03-06 11:00 | 1021.0000 | STOCH_EXIT | 1 | 2299.10 | 2.30% | 996.0000 | 1077.0402 | 40.0201 | M30 bullish FVG [987.6000, 996.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 14 | 2024-03-11 17:00 | 991.6000 | 953.2663 | 2024-03-12 17:00 | 987.0000 | STOCH_EXIT | 19 | -575.70 | -0.56% | 991.2000 | 1068.2673 | 38.3337 | M30 bullish FVG [985.0000, 991.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 15 | 2024-03-14 16:00 | 998.2000 | 963.7849 | 2024-03-15 09:30 | 1005.2000 | STOCH_EXIT | 6 | 611.83 | 0.60% | 1003.0000 | 1067.0302 | 34.4151 | M30 bullish FVG [994.8000, 1003.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 16 | 2024-04-02 14:00 | 999.4000 | 964.9275 | 2024-04-03 09:30 | 1008.0000 | STOCH_EXIT | 10 | 774.82 | 0.76% | 989.8000 | 1068.3451 | 34.4725 | M30 bullish FVG [979.4000, 989.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 17 | 2024-04-11 12:30 | 1143.2000 | 1109.2818 | 2024-04-11 18:30 | 1130.0000 | STOCH_EXIT | 12 | -1290.29 | -1.25% | 1135.0000 | 1211.0363 | 33.9182 | M30 bullish FVG [1130.2000, 1135.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 18 | 2024-05-07 15:00 | 947.0000 | 905.8300 | 2024-05-08 18:30 | 943.0000 | STOCH_EXIT | 26 | -529.12 | -0.52% | 947.0000 | 1029.3399 | 41.1700 | M30 bullish FVG [942.0000, 947.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 19 | 2024-05-15 13:00 | 943.2000 | 910.6302 | 2024-05-16 12:30 | 937.0000 | STOCH_EXIT | 18 | -763.99 | -0.76% | 948.6000 | 1008.3397 | 32.5698 | M30 bullish FVG [943.0000, 948.6000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 20 | 2024-11-08 12:00 | 735.0000 | 713.7629 | 2024-11-08 19:00 | 732.2000 | STOCH_EXIT | 14 | -484.10 | -0.48% | 732.2000 | 777.4742 | 21.2371 | M30 bullish FVG [730.0000, 732.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 21 | 2024-11-12 08:30 | 746.0000 | 724.7843 | 2024-11-12 11:00 | 748.0000 | STOCH_EXIT | 5 | 167.90 | 0.17% | 737.6000 | 788.4313 | 21.2157 | M30 bullish FVG [735.0000, 737.6000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 22 | 2024-11-12 22:30 | 778.4000 | 757.1843 | 2024-11-14 09:30 | 766.0000 | STOCH_EXIT | 42 | -1699.21 | -1.69% | 784.0000 | 820.8313 | 21.2157 | M30 bullish FVG [781.0000, 784.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 23 | 2024-11-15 09:00 | 783.4000 | 759.1614 | 2024-11-15 18:00 | 774.4000 | STOCH_EXIT | 18 | -1232.14 | -1.25% | 785.8000 | 831.8771 | 24.2386 | M30 bullish FVG [778.0000, 785.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 24 | 2024-11-18 22:00 | 798.2000 | 772.9071 | 2024-11-19 08:30 | 803.8000 | STOCH_EXIT | 7 | 585.48 | 0.60% | 796.2000 | 848.7859 | 25.2929 | M30 bullish FVG [791.0000, 796.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 25 | 2024-12-23 10:00 | 788.8000 | 755.2387 | 2024-12-23 10:30 | 810.6000 | STOCH_EXIT | 1 | 2604.04 | 2.66% | 795.4000 | 855.9226 | 33.5613 | M30 bullish FVG [792.0000, 795.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 26 | 2024-12-23 15:00 | 791.2000 | 757.6387 | 2024-12-23 23:00 | 773.0000 | STOCH_EXIT | 16 | -2410.73 | -2.40% | 825.4000 | 858.3226 | 33.5613 | M30 bullish FVG [816.6000, 825.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 27 | 2024-12-26 10:00 | 785.8000 | 749.3687 | 2024-12-26 10:30 | 800.4000 | STOCH_EXIT | 1 | 1725.86 | 1.76% | 771.2000 | 858.6625 | 36.4313 | M30 bullish FVG [765.0000, 771.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 28 | 2024-12-26 12:00 | 791.0000 | 754.5687 | 2024-12-27 21:30 | 789.6000 | STOCH_EXIT | 53 | -275.98 | -0.28% | 798.8000 | 863.8625 | 36.4313 | M30 bullish FVG [795.0000, 798.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 29 | 2024-12-30 20:00 | 832.6000 | 798.3465 | 2024-12-30 22:30 | 844.6000 | STOCH_EXIT | 5 | 1328.21 | 1.34% | 820.4000 | 901.1070 | 34.2535 | M30 bullish FVG [810.0000, 820.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 30 | 2025-01-08 20:00 | 814.0000 | 779.3703 | 2025-01-09 07:30 | 816.2000 | STOCH_EXIT | 8 | 171.73 | 0.17% | 815.0000 | 883.2594 | 34.6297 | M30 bullish FVG [810.2000, 815.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 31 | 2025-02-20 18:30 | 889.4000 | 859.2026 | 2025-02-21 16:00 | 896.6000 | STOCH_EXIT | 29 | 712.69 | 0.71% | 889.2000 | 949.7948 | 30.1974 | M30 bullish FVG [886.0000, 889.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 32 | 2025-03-07 17:00 | 858.8000 | 826.3378 | 2025-03-07 20:00 | 859.0000 | STOCH_EXIT | 6 | -77.75 | -0.08% | 854.2000 | 923.7243 | 32.4622 | M30 bullish FVG [847.8000, 854.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 33 | 2025-03-14 23:00 | 875.0000 | 844.1336 | 2025-03-15 03:00 | 882.8000 | STOCH_EXIT | 4 | 802.85 | 0.79% | 872.4000 | 936.7328 | 30.8664 | M30 bullish FVG [865.6000, 872.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 34 | 2025-03-18 14:30 | 907.0000 | 875.0897 | 2025-03-18 20:00 | 904.0000 | STOCH_EXIT | 11 | -441.32 | -0.43% | 917.0000 | 970.8206 | 31.9103 | M30 bullish FVG [912.8000, 917.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 35 | 2025-03-19 10:30 | 892.0000 | 859.5833 | 2025-03-19 16:00 | 896.6000 | STOCH_EXIT | 11 | 422.45 | 0.42% | 903.2000 | 956.8335 | 32.4167 | M30 bullish FVG [900.0000, 903.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 36 | 2025-03-28 15:00 | 852.4000 | 824.3853 | 2025-03-28 16:00 | 864.4000 | STOCH_EXIT | 2 | 1336.99 | 1.31% | 867.6000 | 908.4295 | 28.0147 | M30 bullish FVG [862.0000, 867.6000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 37 | 2025-03-28 21:00 | 870.0000 | 841.9853 | 2025-03-29 05:00 | 869.6000 | STOCH_EXIT | 10 | -151.11 | -0.15% | 862.0000 | 926.0295 | 28.0147 | M30 bullish FVG [856.8000, 862.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 38 | 2025-04-02 18:00 | 849.0000 | 820.9398 | 2025-04-02 21:00 | 859.0000 | STOCH_EXIT | 6 | 1115.81 | 1.08% | 864.4000 | 905.1204 | 28.0602 | M30 bullish FVG [859.6000, 864.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 39 | 2025-04-28 12:00 | 865.0000 | 837.9000 | 2025-04-28 15:30 | 871.4000 | STOCH_EXIT | 7 | 669.35 | 0.64% | 875.0000 | 919.1999 | 27.1000 | M30 bullish FVG [868.6000, 875.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 40 | 2025-07-30 07:30 | 774.2000 | 752.6023 | 2025-07-31 12:30 | 764.2000 | STOCH_EXIT | 45 | -1464.61 | -1.39% | 770.0000 | 817.3953 | 21.5977 | M30 bullish FVG [765.2000, 770.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 41 | 2025-08-01 12:00 | 776.2000 | 754.8173 | 2025-08-01 17:30 | 771.2000 | STOCH_EXIT | 11 | -773.68 | -0.74% | 778.2000 | 818.9653 | 21.3827 | M30 bullish FVG [774.8000, 778.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 42 | 2025-08-04 22:00 | 769.2000 | 747.5732 | 2025-08-05 06:30 | 771.0000 | STOCH_EXIT | 4 | 138.01 | 0.13% | 768.6000 | 812.4535 | 21.6268 | M30 bullish FVG [765.6000, 768.6000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 43 | 2025-08-07 10:30 | 767.4000 | 746.7549 | 2025-08-08 08:00 | 771.6000 | STOCH_EXIT | 29 | 463.12 | 0.45% | 772.4000 | 808.6902 | 20.6451 | M30 bullish FVG [769.2000, 772.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 44 | 2025-08-08 16:30 | 774.8000 | 754.2438 | 2025-08-08 20:00 | 775.0000 | STOCH_EXIT | 7 | -77.04 | -0.07% | 778.2000 | 815.9124 | 20.5562 | M30 bullish FVG [775.0000, 778.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 45 | 2025-08-11 19:30 | 806.4000 | 786.3407 | 2025-08-12 07:30 | 827.8000 | STOCH_EXIT | 11 | 2655.19 | 2.55% | 809.0000 | 846.5186 | 20.0593 | M30 bullish FVG [794.4000, 809.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 46 | 2025-08-15 08:30 | 808.0000 | 783.6700 | 2025-08-15 12:00 | 808.8000 | STOCH_EXIT | 7 | -1.11 | -0.00% | 810.0000 | 856.6600 | 24.3300 | M30 bullish FVG [804.0000, 810.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 47 | 2025-08-18 17:30 | 803.2000 | 779.5364 | 2025-08-18 21:30 | 807.4000 | STOCH_EXIT | 8 | 448.10 | 0.42% | 800.4000 | 850.5271 | 23.6636 | M30 bullish FVG [797.6000, 800.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 48 | 2025-08-19 16:00 | 807.4000 | 783.4981 | 2025-08-19 21:30 | 808.8000 | STOCH_EXIT | 11 | 78.13 | 0.07% | 809.2000 | 855.2038 | 23.9019 | M30 bullish FVG [803.0000, 809.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 49 | 2025-08-22 18:00 | 786.0000 | 762.2220 | 2025-08-22 20:00 | 790.4000 | STOCH_EXIT | 4 | 491.20 | 0.46% | 790.0000 | 833.5560 | 23.7780 | M30 bullish FVG [784.2000, 790.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 50 | 2025-08-25 23:00 | 796.8000 | 772.8633 | 2025-08-26 07:00 | 799.8000 | STOCH_EXIT | 3 | 297.23 | 0.28% | 791.4000 | 844.6734 | 23.9367 | M30 bullish FVG [789.0000, 791.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 51 | 2025-08-28 21:00 | 793.6000 | 771.7628 | 2025-08-29 06:30 | 801.6000 | STOCH_EXIT | 6 | 979.53 | 0.91% | 800.0000 | 837.2745 | 21.8372 | M30 bullish FVG [795.8000, 800.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 52 | 2025-09-03 10:00 | 788.8000 | 767.8527 | 2025-09-03 14:30 | 785.6000 | STOCH_EXIT | 9 | -550.23 | -0.51% | 785.4000 | 830.6947 | 20.9473 | M30 bullish FVG [782.8000, 785.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 53 | 2025-09-08 15:30 | 787.0000 | 768.5751 | 2025-09-08 17:00 | 789.2000 | STOCH_EXIT | 3 | 193.43 | 0.18% | 789.8000 | 823.8498 | 18.4249 | M30 bullish FVG [787.8000, 789.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 54 | 2025-09-09 07:00 | 797.0000 | 778.7340 | 2025-09-09 08:00 | 801.0000 | STOCH_EXIT | 2 | 435.34 | 0.40% | 793.6000 | 833.5319 | 18.2660 | M30 bullish FVG [791.4000, 793.6000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 55 | 2025-09-10 19:30 | 791.2000 | 773.7816 | 2025-09-11 16:00 | 787.2000 | STOCH_EXIT | 27 | -656.12 | -0.61% | 797.4000 | 826.0368 | 17.4184 | M30 bullish FVG [795.6000, 797.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 56 | 2025-09-11 22:30 | 788.6000 | 771.7686 | 2025-09-12 07:00 | 791.2000 | STOCH_EXIT | 3 | 247.98 | 0.23% | 792.2000 | 822.2628 | 16.8314 | M30 bullish FVG [788.0000, 792.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 57 | 2025-10-21 23:30 | 726.8000 | 700.4616 | 2025-10-22 07:30 | 737.6000 | STOCH_EXIT | 3 | 1500.10 | 1.38% | 738.4000 | 779.4768 | 26.3384 | M30 bullish FVG [735.6000, 738.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 96391.56
- Макс: 110226.72
- Финал: 110226.72

Полная кривая — в `*_equity.csv`.
