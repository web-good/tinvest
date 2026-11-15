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
| StopATR | 2 |
| TakeProfitATR | 1 |
| HTFTrendEMA | 120 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 57 |
| Выигрышных / проигрышных | 36 / 21 |
| Win rate | 63.16% |
| Gross profit / loss | 32578.42 / 18632.07 |
| Profit factor | 1.749 |
| Чистый PnL | 13946.35 (13.95%) |
| Стартовый / финальный капитал | 100000.00 / 113946.35 |
| Макс. просадка | 7379.18 (6.88%) |
| Средняя прибыль / убыток | 904.96 / 887.24 |
| Expectancy | 244.67 |
| Лучшая / худшая сделка | 3385.56 / -2486.66 |
| Exposure | 3.64% |
| CAGR | 6.74% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-01-09 11:30 | 675.2000 | 630.6838 | 2024-01-09 16:00 | 679.0000 | STOCH_EXIT | 9 | 462.19 | 0.46% | 672.6000 | 697.4581 | 22.2581 | M30 bullish FVG [667.4000, 672.6000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 2 | 2024-01-10 15:00 | 695.8000 | 650.8064 | 2024-01-12 10:00 | 692.0000 | STOCH_EXIT | 28 | -647.12 | -0.65% | 679.0000 | 718.2968 | 22.4968 | M30 bullish FVG [675.6000, 679.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 3 | 2024-01-17 18:00 | 715.0000 | 669.7246 | 2024-01-18 12:30 | 718.8000 | STOCH_EXIT | 8 | 428.55 | 0.43% | 712.0000 | 737.6377 | 22.6377 | M30 bullish FVG [703.2000, 712.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 4 | 2024-01-18 16:30 | 718.8000 | 673.5585 | 2024-01-19 09:30 | 722.6000 | STOCH_EXIT | 5 | 428.02 | 0.43% | 717.0000 | 741.4207 | 22.6207 | M30 bullish FVG [714.4000, 717.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 5 | 2024-01-23 10:30 | 745.6000 | 700.7554 | 2024-01-24 17:30 | 747.4000 | STOCH_EXIT | 33 | 141.17 | 0.14% | 739.4000 | 768.0223 | 22.4223 | M30 bullish FVG [729.8000, 739.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 6 | 2024-02-05 15:00 | 757.0000 | 718.9374 | 2024-02-06 18:00 | 760.2000 | STOCH_EXIT | 25 | 324.71 | 0.32% | 761.4000 | 776.0313 | 19.0313 | M30 bullish FVG [758.6000, 761.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 7 | 2024-02-08 12:30 | 767.2000 | 728.0860 | 2024-02-10 11:00 | 755.6000 | STOCH_EXIT | 33 | -1619.34 | -1.61% | 770.8000 | 786.7570 | 19.5570 | M30 bullish FVG [768.8000, 770.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 8 | 2024-02-14 17:00 | 811.0000 | 768.7444 | 2024-02-15 11:00 | 832.1278 | TP | 7 | 2477.36 | 2.50% | 820.8000 | 832.1278 | 21.1278 | M30 bullish FVG [817.6000, 820.8000], stochastic crossed below 20 | FVG target 832.1278 |
| 9 | 2024-02-15 16:00 | 816.4000 | 770.8198 | 2024-02-16 18:00 | 817.4000 | STOCH_EXIT | 23 | 22.70 | 0.02% | 824.2000 | 839.1901 | 22.7901 | M30 bullish FVG [813.0000, 824.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 10 | 2024-02-27 11:30 | 868.0000 | 808.3614 | 2024-02-27 12:00 | 897.8193 | TP | 1 | 3385.56 | 3.33% | 862.8000 | 897.8193 | 29.8193 | M30 bullish FVG [854.6000, 862.8000], stochastic crossed below 20 | FVG target 897.8193 |
| 11 | 2024-02-28 16:00 | 1015.4000 | 945.4499 | 2024-03-01 12:30 | 992.6000 | STOCH_EXIT | 31 | -2451.81 | -2.34% | 1023.8000 | 1050.3750 | 34.9750 | M30 bullish FVG [974.0000, 1023.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 12 | 2024-03-04 16:30 | 993.0000 | 913.4954 | 2024-03-05 14:00 | 1004.0000 | STOCH_EXIT | 14 | 1030.15 | 1.01% | 996.2000 | 1032.7523 | 39.7523 | M30 bullish FVG [985.0000, 996.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 13 | 2024-03-06 10:30 | 997.0000 | 916.9598 | 2024-03-06 11:00 | 1021.0000 | STOCH_EXIT | 1 | 2391.06 | 2.30% | 996.0000 | 1037.0201 | 40.0201 | M30 bullish FVG [987.6000, 996.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 14 | 2024-03-11 17:00 | 991.6000 | 914.9327 | 2024-03-12 17:00 | 987.0000 | STOCH_EXIT | 19 | -598.06 | -0.56% | 991.2000 | 1029.9337 | 38.3337 | M30 bullish FVG [985.0000, 991.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 15 | 2024-03-14 16:00 | 998.2000 | 929.3698 | 2024-03-15 09:30 | 1005.2000 | STOCH_EXIT | 6 | 629.82 | 0.60% | 1003.0000 | 1032.6151 | 34.4151 | M30 bullish FVG [994.8000, 1003.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 16 | 2024-04-02 14:00 | 999.4000 | 930.4549 | 2024-04-03 09:30 | 1008.0000 | STOCH_EXIT | 10 | 805.21 | 0.76% | 989.8000 | 1033.8725 | 34.4725 | M30 bullish FVG [979.4000, 989.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 17 | 2024-04-11 12:30 | 1143.2000 | 1075.3637 | 2024-04-11 18:30 | 1130.0000 | STOCH_EXIT | 12 | -1333.30 | -1.25% | 1135.0000 | 1177.1182 | 33.9182 | M30 bullish FVG [1130.2000, 1135.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 18 | 2024-05-07 15:00 | 947.0000 | 864.6601 | 2024-05-08 18:30 | 943.0000 | STOCH_EXIT | 26 | -548.90 | -0.52% | 947.0000 | 988.1700 | 41.1700 | M30 bullish FVG [942.0000, 947.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 19 | 2024-05-15 13:00 | 943.2000 | 878.0603 | 2024-05-16 12:30 | 937.0000 | STOCH_EXIT | 18 | -792.55 | -0.76% | 948.6000 | 975.7698 | 32.5698 | M30 bullish FVG [943.0000, 948.6000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 20 | 2024-11-08 12:00 | 735.0000 | 692.5258 | 2024-11-08 19:00 | 732.2000 | STOCH_EXIT | 14 | -501.77 | -0.48% | 732.2000 | 756.2371 | 21.2371 | M30 bullish FVG [730.0000, 732.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 21 | 2024-11-12 08:30 | 746.0000 | 703.5687 | 2024-11-12 11:00 | 748.0000 | STOCH_EXIT | 5 | 174.17 | 0.17% | 737.6000 | 767.2157 | 21.2157 | M30 bullish FVG [735.0000, 737.6000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 22 | 2024-11-12 22:30 | 778.4000 | 735.9687 | 2024-11-14 09:30 | 766.0000 | STOCH_EXIT | 42 | -1751.90 | -1.69% | 784.0000 | 799.6157 | 21.2157 | M30 bullish FVG [781.0000, 784.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 23 | 2024-11-15 09:00 | 783.4000 | 734.9229 | 2024-11-15 18:00 | 774.4000 | STOCH_EXIT | 18 | -1271.26 | -1.25% | 785.8000 | 807.6386 | 24.2386 | M30 bullish FVG [778.0000, 785.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 24 | 2024-11-18 22:00 | 798.2000 | 747.6141 | 2024-11-19 08:30 | 803.8000 | STOCH_EXIT | 7 | 604.67 | 0.60% | 796.2000 | 823.4929 | 25.2929 | M30 bullish FVG [791.0000, 796.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 25 | 2024-12-23 10:00 | 788.8000 | 721.6774 | 2024-12-23 10:30 | 810.6000 | STOCH_EXIT | 1 | 2688.04 | 2.66% | 795.4000 | 822.3613 | 33.5613 | M30 bullish FVG [792.0000, 795.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 26 | 2024-12-23 15:00 | 791.2000 | 724.0774 | 2024-12-23 23:00 | 773.0000 | STOCH_EXIT | 16 | -2486.66 | -2.40% | 825.4000 | 824.7613 | 33.5613 | M30 bullish FVG [816.6000, 825.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 27 | 2024-12-26 10:00 | 785.8000 | 712.9375 | 2024-12-26 10:30 | 800.4000 | STOCH_EXIT | 1 | 1781.09 | 1.76% | 771.2000 | 822.2313 | 36.4313 | M30 bullish FVG [765.0000, 771.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 28 | 2024-12-26 12:00 | 791.0000 | 718.1375 | 2024-12-27 21:30 | 789.6000 | STOCH_EXIT | 53 | -286.93 | -0.28% | 798.8000 | 827.4313 | 36.4313 | M30 bullish FVG [795.0000, 798.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 29 | 2024-12-30 20:00 | 832.6000 | 764.0930 | 2024-12-30 22:30 | 844.6000 | STOCH_EXIT | 5 | 1384.01 | 1.34% | 820.4000 | 866.8535 | 34.2535 | M30 bullish FVG [810.0000, 820.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 30 | 2025-01-08 20:00 | 814.0000 | 744.7406 | 2025-01-09 07:30 | 816.2000 | STOCH_EXIT | 8 | 177.27 | 0.17% | 815.0000 | 848.6297 | 34.6297 | M30 bullish FVG [810.2000, 815.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 31 | 2025-02-20 18:30 | 889.4000 | 829.0052 | 2025-02-21 16:00 | 896.6000 | STOCH_EXIT | 29 | 744.23 | 0.71% | 889.2000 | 919.5974 | 30.1974 | M30 bullish FVG [886.0000, 889.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 32 | 2025-03-07 17:00 | 858.8000 | 793.8757 | 2025-03-07 20:00 | 859.0000 | STOCH_EXIT | 6 | -81.04 | -0.08% | 854.2000 | 891.2622 | 32.4622 | M30 bullish FVG [847.8000, 854.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 33 | 2025-03-14 23:00 | 875.0000 | 813.2672 | 2025-03-15 03:00 | 882.8000 | STOCH_EXIT | 4 | 830.53 | 0.79% | 872.4000 | 905.8664 | 30.8664 | M30 bullish FVG [865.6000, 872.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 34 | 2025-03-18 14:30 | 907.0000 | 843.1794 | 2025-03-18 20:00 | 904.0000 | STOCH_EXIT | 11 | -456.94 | -0.43% | 917.0000 | 938.9103 | 31.9103 | M30 bullish FVG [912.8000, 917.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 35 | 2025-03-19 10:30 | 892.0000 | 827.1665 | 2025-03-19 16:00 | 896.6000 | STOCH_EXIT | 11 | 437.27 | 0.42% | 903.2000 | 924.4167 | 32.4167 | M30 bullish FVG [900.0000, 903.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 36 | 2025-03-28 15:00 | 852.4000 | 796.3705 | 2025-03-28 16:00 | 864.4000 | STOCH_EXIT | 2 | 1381.56 | 1.31% | 867.6000 | 880.4147 | 28.0147 | M30 bullish FVG [862.0000, 867.6000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 37 | 2025-03-28 21:00 | 870.0000 | 813.9705 | 2025-03-29 05:00 | 869.6000 | STOCH_EXIT | 10 | -156.19 | -0.15% | 862.0000 | 898.0147 | 28.0147 | M30 bullish FVG [856.8000, 862.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 38 | 2025-04-02 18:00 | 849.0000 | 792.8796 | 2025-04-02 21:00 | 859.0000 | STOCH_EXIT | 6 | 1152.40 | 1.08% | 864.4000 | 877.0602 | 28.0602 | M30 bullish FVG [859.6000, 864.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 39 | 2025-04-28 12:00 | 865.0000 | 810.8001 | 2025-04-28 15:30 | 871.4000 | STOCH_EXIT | 7 | 691.48 | 0.64% | 875.0000 | 892.1000 | 27.1000 | M30 bullish FVG [868.6000, 875.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 40 | 2025-07-30 07:30 | 774.2000 | 731.0047 | 2025-07-31 12:30 | 764.2000 | STOCH_EXIT | 45 | -1518.46 | -1.39% | 770.0000 | 795.7977 | 21.5977 | M30 bullish FVG [765.2000, 770.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 41 | 2025-08-01 12:00 | 776.2000 | 733.4347 | 2025-08-01 17:30 | 771.2000 | STOCH_EXIT | 11 | -802.54 | -0.74% | 778.2000 | 797.5827 | 21.3827 | M30 bullish FVG [774.8000, 778.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 42 | 2025-08-04 22:00 | 769.2000 | 725.9465 | 2025-08-05 06:30 | 771.0000 | STOCH_EXIT | 4 | 143.16 | 0.13% | 768.6000 | 790.8268 | 21.6268 | M30 bullish FVG [765.6000, 768.6000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 43 | 2025-08-07 10:30 | 767.4000 | 726.1098 | 2025-08-08 08:00 | 771.6000 | STOCH_EXIT | 29 | 476.84 | 0.45% | 772.4000 | 788.0451 | 20.6451 | M30 bullish FVG [769.2000, 772.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 44 | 2025-08-08 16:30 | 774.8000 | 733.6876 | 2025-08-08 20:00 | 775.0000 | STOCH_EXIT | 7 | -79.91 | -0.07% | 778.2000 | 795.3562 | 20.5562 | M30 bullish FVG [775.0000, 778.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 45 | 2025-08-11 19:30 | 806.4000 | 766.2814 | 2025-08-12 07:30 | 826.4593 | TP | 11 | 2559.30 | 2.39% | 809.0000 | 826.4593 | 20.0593 | M30 bullish FVG [794.4000, 809.0000], stochastic crossed below 20 | FVG target 826.4593 |
| 46 | 2025-08-15 08:30 | 808.0000 | 759.3400 | 2025-08-15 12:00 | 808.8000 | STOCH_EXIT | 7 | -1.14 | -0.00% | 810.0000 | 832.3300 | 24.3300 | M30 bullish FVG [804.0000, 810.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 47 | 2025-08-18 17:30 | 803.2000 | 755.8729 | 2025-08-18 21:30 | 807.4000 | STOCH_EXIT | 8 | 465.07 | 0.42% | 800.4000 | 826.8636 | 23.6636 | M30 bullish FVG [797.6000, 800.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 48 | 2025-08-19 16:00 | 807.4000 | 759.5962 | 2025-08-19 21:30 | 808.8000 | STOCH_EXIT | 11 | 81.09 | 0.07% | 809.2000 | 831.3019 | 23.9019 | M30 bullish FVG [803.0000, 809.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 49 | 2025-08-22 18:00 | 786.0000 | 738.4440 | 2025-08-22 20:00 | 790.4000 | STOCH_EXIT | 4 | 509.26 | 0.46% | 790.0000 | 809.7780 | 23.7780 | M30 bullish FVG [784.2000, 790.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 50 | 2025-08-25 23:00 | 796.8000 | 748.9266 | 2025-08-26 07:00 | 799.8000 | STOCH_EXIT | 3 | 306.04 | 0.28% | 791.4000 | 820.7367 | 23.9367 | M30 bullish FVG [789.0000, 791.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 51 | 2025-08-28 21:00 | 793.6000 | 749.9255 | 2025-08-29 06:30 | 801.6000 | STOCH_EXIT | 6 | 1008.34 | 0.91% | 800.0000 | 815.4372 | 21.8372 | M30 bullish FVG [795.8000, 800.0000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 52 | 2025-09-03 10:00 | 788.8000 | 746.9053 | 2025-09-03 14:30 | 785.6000 | STOCH_EXIT | 9 | -566.18 | -0.51% | 785.4000 | 809.7473 | 20.9473 | M30 bullish FVG [782.8000, 785.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 53 | 2025-09-08 15:30 | 787.0000 | 750.1502 | 2025-09-08 17:00 | 789.2000 | STOCH_EXIT | 3 | 200.49 | 0.18% | 789.8000 | 805.4249 | 18.4249 | M30 bullish FVG [787.8000, 789.8000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 54 | 2025-09-09 07:00 | 797.0000 | 760.4681 | 2025-09-09 08:00 | 801.0000 | STOCH_EXIT | 2 | 448.14 | 0.40% | 793.6000 | 815.2660 | 18.2660 | M30 bullish FVG [791.4000, 793.6000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 55 | 2025-09-10 19:30 | 791.2000 | 756.3632 | 2025-09-11 16:00 | 787.2000 | STOCH_EXIT | 27 | -680.07 | -0.61% | 797.4000 | 808.6184 | 17.4184 | M30 bullish FVG [795.6000, 797.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 56 | 2025-09-11 22:30 | 788.6000 | 754.9372 | 2025-09-12 07:00 | 791.2000 | STOCH_EXIT | 3 | 257.03 | 0.23% | 792.2000 | 805.4314 | 16.8314 | M30 bullish FVG [788.0000, 792.2000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |
| 57 | 2025-10-21 23:30 | 726.8000 | 674.1232 | 2025-10-22 07:30 | 737.6000 | STOCH_EXIT | 3 | 1550.44 | 1.38% | 738.4000 | 753.1384 | 26.3384 | M30 bullish FVG [735.6000, 738.4000], stochastic crossed below 20 | FVG stochastic %K(9) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 98021.86
- Макс: 113946.35
- Финал: 113946.35

Полная кривая — в `*_equity.csv`.
