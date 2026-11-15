# Бэктест MDMG (Minutes30)

- Период: 2023-11-15 22:37 — 2025-11-15 22:37
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.1 |
| MaxWaitBars | 30 |
| StochPeriod | 11 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 1 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 133 |
| Выигрышных / проигрышных | 98 / 35 |
| Win rate | 73.68% |
| Gross profit / loss | 94238.95 / 56013.15 |
| Profit factor | 1.682 |
| Чистый PnL | 38225.80 (38.23%) |
| Стартовый / финальный капитал | 100000.00 / 138225.80 |
| Макс. просадка | 8799.10 (7.91%) |
| Средняя прибыль / убыток | 961.62 / 1600.38 |
| Expectancy | 287.41 |
| Лучшая / худшая сделка | 4468.81 / -4409.63 |
| Exposure | 8.62% |
| CAGR | 17.54% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-11-20 18:30 | 796.4000 | 770.7541 | 2023-11-21 17:00 | 799.1000 | STOCH_EXIT | 16 | 237.78 | 0.24% | 797.1000 | 822.0459 | 25.6459 | M30 bullish FVG [793.3000, 797.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 2 | 2023-11-23 10:30 | 815.1000 | 789.2775 | 2023-11-24 10:00 | 816.0000 | STOCH_EXIT | 18 | 10.30 | 0.01% | 814.0000 | 840.9225 | 25.8225 | M30 bullish FVG [808.3000, 814.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 3 | 2023-11-24 14:30 | 805.6000 | 780.0577 | 2023-11-27 09:30 | 813.3000 | STOCH_EXIT | 9 | 854.43 | 0.85% | 815.0000 | 831.1423 | 25.5423 | M30 bullish FVG [811.3000, 815.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 4 | 2023-12-06 10:30 | 732.0000 | 705.6074 | 2023-12-06 16:30 | 728.9000 | STOCH_EXIT | 12 | -528.60 | -0.52% | 744.0000 | 758.3926 | 26.3926 | M30 bullish FVG [737.2000, 744.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 5 | 2023-12-11 10:30 | 712.3000 | 684.5008 | 2023-12-12 10:00 | 712.0000 | STOCH_EXIT | 18 | -142.71 | -0.14% | 705.0000 | 740.0992 | 27.7992 | M30 bullish FVG [701.8000, 705.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 6 | 2023-12-13 17:00 | 721.9000 | 694.0573 | 2023-12-15 14:00 | 730.8000 | STOCH_EXIT | 32 | 1136.14 | 1.13% | 727.1000 | 749.7427 | 27.8427 | M30 bullish FVG [724.2000, 727.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 7 | 2023-12-22 17:00 | 807.3000 | 776.2253 | 2023-12-25 10:00 | 821.2000 | STOCH_EXIT | 5 | 1635.72 | 1.62% | 805.5000 | 838.3747 | 31.0747 | M30 bullish FVG [796.5000, 805.5000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 8 | 2023-12-25 16:00 | 820.2000 | 788.7092 | 2023-12-26 17:00 | 808.6000 | STOCH_EXIT | 21 | -1551.80 | -1.51% | 820.4000 | 851.6908 | 31.4908 | M30 bullish FVG [809.0000, 820.4000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 9 | 2024-01-08 18:30 | 808.1000 | 779.4447 | 2024-01-09 12:00 | 836.7553 | TP | 6 | 3479.11 | 3.44% | 818.1000 | 836.7553 | 28.6553 | M30 bullish FVG [815.1000, 818.1000], stochastic crossed below 20 | FVG target 836.7553 |
| 10 | 2024-01-09 18:00 | 824.6000 | 796.9272 | 2024-01-10 09:30 | 835.0000 | STOCH_EXIT | 2 | 1215.42 | 1.16% | 823.1000 | 852.2728 | 27.6728 | M30 bullish FVG [816.7000, 823.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 11 | 2024-01-22 13:00 | 800.0000 | 774.9856 | 2024-01-22 14:00 | 807.7000 | STOCH_EXIT | 2 | 910.29 | 0.86% | 804.9000 | 825.0144 | 25.0144 | M30 bullish FVG [800.0000, 804.9000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 12 | 2024-01-23 16:00 | 809.8000 | 785.2152 | 2024-01-25 10:00 | 818.9000 | STOCH_EXIT | 26 | 1093.71 | 1.02% | 814.0000 | 834.3848 | 24.5848 | M30 bullish FVG [809.0000, 814.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 13 | 2024-02-07 10:30 | 986.6000 | 955.8318 | 2024-02-08 10:00 | 955.8318 | SL | 18 | -3459.60 | -3.22% | 974.5000 | 1017.3682 | 30.7682 | M30 bullish FVG [946.3000, 974.5000], stochastic crossed below 20 | FVG stop 955.8318 |
| 14 | 2024-02-12 10:30 | 956.2000 | 922.1076 | 2024-02-12 16:00 | 957.8000 | STOCH_EXIT | 11 | 70.09 | 0.07% | 978.5000 | 990.2924 | 34.0924 | M30 bullish FVG [975.0000, 978.5000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 15 | 2024-02-16 12:00 | 993.9000 | 959.9828 | 2024-02-19 14:00 | 1015.4000 | STOCH_EXIT | 23 | 2152.01 | 2.06% | 990.1000 | 1027.8172 | 33.9172 | M30 bullish FVG [981.8000, 990.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 16 | 2024-02-20 10:30 | 984.2000 | 949.2337 | 2024-02-21 10:00 | 949.2337 | SL | 18 | -3880.77 | -3.65% | 1008.7000 | 1019.1663 | 34.9663 | M30 bullish FVG [1003.0000, 1008.7000], stochastic crossed below 20 | FVG stop 949.2337 |
| 17 | 2024-02-26 17:00 | 957.0000 | 920.7222 | 2024-02-27 09:30 | 968.3000 | STOCH_EXIT | 4 | 1106.10 | 1.08% | 945.0000 | 993.2778 | 36.2778 | M30 bullish FVG [941.0000, 945.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 18 | 2024-02-28 16:30 | 962.7000 | 927.5186 | 2024-02-29 11:30 | 971.1000 | STOCH_EXIT | 9 | 802.77 | 0.77% | 991.8000 | 997.8814 | 35.1814 | M30 bullish FVG [978.9000, 991.8000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 19 | 2024-03-07 13:00 | 1015.7000 | 978.9098 | 2024-03-07 14:00 | 1036.3000 | STOCH_EXIT | 2 | 2016.12 | 1.93% | 1026.8000 | 1052.4902 | 36.7902 | M30 bullish FVG [1007.3000, 1026.8000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 20 | 2024-03-11 16:30 | 1045.1000 | 1008.6734 | 2024-03-12 10:00 | 1008.6734 | SL | 6 | -3820.26 | -3.58% | 1030.9000 | 1081.5266 | 36.4266 | M30 bullish FVG [1016.6000, 1030.9000], stochastic crossed below 20 | FVG stop 1008.6734 |
| 21 | 2024-03-19 17:30 | 1006.1000 | 971.4691 | 2024-03-21 10:30 | 1001.5000 | STOCH_EXIT | 24 | -571.59 | -0.56% | 1013.0000 | 1040.7309 | 34.6309 | M30 bullish FVG [1008.7000, 1013.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 22 | 2024-03-21 17:00 | 998.9000 | 965.2632 | 2024-03-22 11:00 | 1019.0000 | STOCH_EXIT | 7 | 1947.29 | 1.91% | 1008.2000 | 1032.5368 | 33.6368 | M30 bullish FVG [1002.0000, 1008.2000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 23 | 2024-04-03 15:00 | 1049.6000 | 1019.5474 | 2024-04-03 18:30 | 1058.0000 | STOCH_EXIT | 7 | 727.27 | 0.70% | 1070.0000 | 1079.6526 | 30.0526 | M30 bullish FVG [1060.0000, 1070.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 24 | 2024-04-09 15:30 | 1013.1000 | 980.0311 | 2024-04-10 09:30 | 1017.6000 | STOCH_EXIT | 7 | 362.40 | 0.34% | 1018.0000 | 1046.1689 | 33.0689 | M30 bullish FVG [1014.0000, 1018.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 25 | 2024-04-23 12:00 | 1051.1000 | 1022.2777 | 2024-04-24 13:30 | 1050.6000 | STOCH_EXIT | 22 | -155.08 | -0.15% | 1057.0000 | 1079.9223 | 28.8223 | M30 bullish FVG [1053.8000, 1057.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 26 | 2024-05-03 13:00 | 1136.9000 | 1104.5670 | 2024-05-04 15:00 | 1154.8000 | STOCH_EXIT | 22 | 1541.38 | 1.47% | 1133.9000 | 1169.2330 | 32.3330 | M30 bullish FVG [1115.0000, 1133.9000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 27 | 2024-05-09 14:00 | 1131.9000 | 1099.6286 | 2024-05-09 18:30 | 1139.4000 | STOCH_EXIT | 9 | 598.25 | 0.56% | 1122.0000 | 1164.1714 | 32.2714 | M30 bullish FVG [1116.6000, 1122.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 28 | 2024-05-15 13:00 | 1179.1000 | 1144.8545 | 2024-05-15 15:00 | 1144.8545 | SL | 4 | -3222.08 | -3.00% | 1194.9000 | 1213.3455 | 34.2455 | M30 bullish FVG [1188.5000, 1194.9000], stochastic crossed below 20 | FVG stop 1144.8545 |
| 29 | 2024-05-17 11:30 | 1064.1000 | 1022.1516 | 2024-05-17 18:00 | 1088.1000 | STOCH_EXIT | 13 | 2246.54 | 2.15% | 1074.0000 | 1106.0484 | 41.9484 | M30 bullish FVG [1056.7000, 1074.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 30 | 2024-05-22 16:00 | 1070.1000 | 1023.8673 | 2024-06-17 09:30 | 1116.3327 | TP | 6 | 4468.81 | 4.22% | 1083.5000 | 1116.3327 | 46.2327 | M30 bullish FVG [1078.4000, 1083.5000], stochastic crossed below 20 | FVG target 1116.3327 |
| 31 | 2024-06-17 16:00 | 1182.0000 | 1136.2482 | 2024-06-17 16:00 | 1136.2482 | SL | 0 | -4409.63 | -3.97% | 1271.7000 | 1227.7518 | 45.7518 | M30 bullish FVG [1179.2000, 1271.7000], stochastic crossed below 20 | FVG stop 1136.2482 |
| 32 | 2024-06-18 11:30 | 1032.5000 | 964.9876 | 2024-06-19 09:30 | 1027.0000 | STOCH_EXIT | 15 | -672.56 | -0.63% | 1021.7000 | 1100.0124 | 67.5124 | M30 bullish FVG [1000.0000, 1021.7000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 33 | 2024-07-02 16:00 | 829.4000 | 775.6755 | 2024-07-03 16:00 | 830.4000 | STOCH_EXIT | 19 | 21.60 | 0.02% | 834.0000 | 883.1245 | 53.7245 | M30 bullish FVG [826.7000, 834.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 34 | 2024-07-22 13:00 | 841.5000 | 804.7961 | 2024-07-22 14:00 | 858.0000 | STOCH_EXIT | 2 | 1971.93 | 1.86% | 838.2000 | 878.2039 | 36.7039 | M30 bullish FVG [830.2000, 838.2000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 35 | 2024-07-31 16:00 | 872.5000 | 838.8355 | 2024-08-01 09:30 | 880.0000 | STOCH_EXIT | 6 | 814.72 | 0.76% | 850.0000 | 906.1645 | 33.6645 | M30 bullish FVG [843.0000, 850.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 36 | 2024-09-19 08:30 | 840.1000 | 816.0336 | 2024-09-19 08:30 | 841.2000 | STOCH_EXIT | 0 | 33.46 | 0.03% | 839.6000 | 864.1664 | 24.0664 | M30 bullish FVG [837.2000, 839.6000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 37 | 2024-09-24 10:30 | 861.7000 | 838.1602 | 2024-09-25 18:30 | 869.7000 | STOCH_EXIT | 50 | 898.92 | 0.83% | 879.3000 | 885.2398 | 23.5398 | M30 bullish FVG [876.7000, 879.3000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 38 | 2024-09-26 17:00 | 880.0000 | 855.8009 | 2024-09-26 20:00 | 883.6000 | STOCH_EXIT | 6 | 337.06 | 0.31% | 880.9000 | 904.1991 | 24.1991 | M30 bullish FVG [873.5000, 880.9000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 39 | 2024-09-27 18:30 | 886.6000 | 862.8651 | 2024-09-27 19:00 | 910.3349 | TP | 1 | 2831.72 | 2.57% | 882.4000 | 910.3349 | 23.7349 | M30 bullish FVG [880.0000, 882.4000], stochastic crossed below 20 | FVG target 910.3349 |
| 40 | 2024-09-30 18:00 | 901.3000 | 877.8533 | 2024-09-30 19:00 | 908.9000 | STOCH_EXIT | 2 | 836.86 | 0.74% | 907.0000 | 924.7467 | 23.4467 | M30 bullish FVG [896.9000, 907.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 41 | 2024-10-02 18:30 | 886.8000 | 861.7745 | 2024-10-03 15:30 | 861.7745 | SL | 28 | -3315.17 | -2.92% | 905.0000 | 911.8255 | 25.0255 | M30 bullish FVG [895.8000, 905.0000], stochastic crossed below 20 | FVG stop 861.7745 |
| 42 | 2024-10-04 13:00 | 870.9000 | 844.8112 | 2024-10-04 20:30 | 874.8000 | STOCH_EXIT | 15 | 381.42 | 0.35% | 879.0000 | 896.9888 | 26.0888 | M30 bullish FVG [873.9000, 879.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 43 | 2024-10-07 11:30 | 875.2000 | 849.5104 | 2024-10-08 08:30 | 869.7000 | STOCH_EXIT | 28 | -802.93 | -0.73% | 881.3000 | 900.8896 | 25.6896 | M30 bullish FVG [877.0000, 881.3000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 44 | 2024-10-15 18:30 | 884.0000 | 861.8992 | 2024-10-15 22:00 | 889.3000 | STOCH_EXIT | 7 | 547.26 | 0.50% | 884.1000 | 906.1008 | 22.1008 | M30 bullish FVG [880.0000, 884.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 45 | 2024-10-23 20:00 | 840.3000 | 819.5370 | 2024-10-24 09:30 | 846.0000 | STOCH_EXIT | 13 | 636.25 | 0.58% | 843.3000 | 861.0630 | 20.7630 | M30 bullish FVG [841.0000, 843.3000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 46 | 2024-10-25 11:00 | 850.7000 | 828.8579 | 2024-10-25 12:00 | 859.0000 | STOCH_EXIT | 2 | 967.87 | 0.87% | 851.2000 | 872.5421 | 21.8421 | M30 bullish FVG [848.0000, 851.2000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 47 | 2024-10-30 14:30 | 862.8000 | 838.9097 | 2024-10-30 14:30 | 886.6903 | TP | 0 | 2992.03 | 2.67% | 869.0000 | 886.6903 | 23.8903 | M30 bullish FVG [843.2000, 869.0000], stochastic crossed below 20 | FVG target 886.6903 |
| 48 | 2024-11-12 19:00 | 878.8000 | 850.5466 | 2024-11-13 18:30 | 883.0000 | STOCH_EXIT | 33 | 434.80 | 0.38% | 883.6000 | 907.0534 | 28.2534 | M30 bullish FVG [880.3000, 883.6000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 49 | 2024-11-13 20:00 | 877.1000 | 849.4862 | 2024-11-14 10:30 | 878.2000 | STOCH_EXIT | 15 | 29.13 | 0.03% | 882.9000 | 904.7138 | 27.6138 | M30 bullish FVG [879.5000, 882.9000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 50 | 2024-11-21 18:00 | 864.6000 | 836.9635 | 2024-11-21 23:00 | 870.0000 | STOCH_EXIT | 10 | 602.85 | 0.52% | 873.0000 | 892.2365 | 27.6365 | M30 bullish FVG [869.5000, 873.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 51 | 2024-11-27 18:00 | 854.5000 | 829.1286 | 2024-11-27 23:00 | 869.9000 | STOCH_EXIT | 10 | 1977.14 | 1.70% | 858.0000 | 879.8714 | 25.3714 | M30 bullish FVG [853.8000, 858.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 52 | 2024-11-29 21:30 | 850.8000 | 824.4206 | 2024-11-30 08:30 | 856.9000 | STOCH_EXIT | 8 | 729.21 | 0.62% | 860.0000 | 877.1794 | 26.3794 | M30 bullish FVG [853.3000, 860.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 53 | 2024-12-26 18:00 | 853.0000 | 834.9361 | 2024-12-27 17:30 | 853.6000 | STOCH_EXIT | 33 | -35.21 | -0.03% | 865.0000 | 871.0639 | 18.0639 | M30 bullish FVG [857.7000, 865.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 54 | 2025-01-08 20:30 | 941.0000 | 916.0515 | 2025-01-09 10:30 | 916.0515 | SL | 14 | -3260.51 | -2.75% | 941.2000 | 965.9485 | 24.9485 | M30 bullish FVG [936.0000, 941.2000], stochastic crossed below 20 | FVG stop 916.0515 |
| 55 | 2025-01-09 19:00 | 929.0000 | 903.6335 | 2025-01-09 19:30 | 931.0000 | STOCH_EXIT | 1 | 132.68 | 0.12% | 930.9000 | 954.3665 | 25.3665 | M30 bullish FVG [923.6000, 930.9000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 56 | 2025-01-22 20:00 | 943.9000 | 920.1637 | 2025-01-23 11:00 | 967.6363 | TP | 16 | 2779.22 | 2.41% | 951.6000 | 967.6363 | 23.7363 | M30 bullish FVG [944.6000, 951.6000], stochastic crossed below 20 | FVG target 967.6363 |
| 57 | 2025-01-23 18:00 | 964.0000 | 939.7520 | 2025-01-23 22:00 | 969.0000 | STOCH_EXIT | 8 | 496.12 | 0.42% | 970.5000 | 988.2480 | 24.2480 | M30 bullish FVG [963.9000, 970.5000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 58 | 2025-01-27 21:30 | 994.6000 | 969.2760 | 2025-01-28 08:30 | 1003.1000 | STOCH_EXIT | 8 | 892.64 | 0.75% | 1008.6000 | 1019.9240 | 25.3240 | M30 bullish FVG [1001.9000, 1008.6000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 59 | 2025-02-10 23:00 | 1002.6000 | 971.9427 | 2025-02-11 12:30 | 1014.0000 | STOCH_EXIT | 13 | 1236.61 | 1.04% | 1015.0000 | 1033.2573 | 30.6573 | M30 bullish FVG [1011.0000, 1015.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 60 | 2025-02-11 17:00 | 1005.9000 | 975.6825 | 2025-02-12 07:00 | 1014.7000 | STOCH_EXIT | 14 | 934.76 | 0.77% | 1013.2000 | 1036.1175 | 30.2175 | M30 bullish FVG [1010.0000, 1013.2000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 61 | 2025-02-20 10:00 | 1024.8000 | 995.2302 | 2025-02-20 12:00 | 1036.5000 | STOCH_EXIT | 4 | 1269.65 | 1.04% | 1021.3000 | 1054.3698 | 29.5698 | M30 bullish FVG [1013.0000, 1021.3000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 62 | 2025-02-25 19:00 | 1027.1000 | 998.6325 | 2025-02-25 21:30 | 1035.9000 | STOCH_EXIT | 5 | 932.22 | 0.76% | 1034.0000 | 1055.5675 | 28.4675 | M30 bullish FVG [1030.2000, 1034.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 63 | 2025-02-27 12:00 | 1001.2000 | 973.5301 | 2025-02-27 16:30 | 1010.7000 | STOCH_EXIT | 9 | 1053.26 | 0.85% | 1031.4000 | 1028.8699 | 27.6699 | M30 bullish FVG [1027.5000, 1031.4000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 64 | 2025-02-27 23:00 | 1016.7000 | 989.0301 | 2025-02-28 08:30 | 1033.7000 | STOCH_EXIT | 5 | 1964.90 | 1.57% | 1019.5000 | 1044.3699 | 27.6699 | M30 bullish FVG [1016.0000, 1019.5000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 65 | 2025-02-28 13:30 | 1015.1000 | 987.4208 | 2025-03-01 10:30 | 1020.6000 | STOCH_EXIT | 38 | 560.27 | 0.44% | 1022.0000 | 1042.7792 | 27.6792 | M30 bullish FVG [1015.4000, 1022.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 66 | 2025-03-04 15:30 | 1036.2000 | 1008.1888 | 2025-03-04 18:30 | 1047.8000 | STOCH_EXIT | 6 | 1298.63 | 1.02% | 1051.3000 | 1064.2112 | 28.0112 | M30 bullish FVG [1048.0000, 1051.3000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 67 | 2025-03-06 16:00 | 1066.0000 | 1038.9388 | 2025-03-07 10:00 | 1071.6000 | STOCH_EXIT | 22 | 548.28 | 0.42% | 1063.1000 | 1093.0612 | 27.0612 | M30 bullish FVG [1058.0000, 1063.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 68 | 2025-03-13 10:00 | 1031.6000 | 1003.8058 | 2025-03-13 17:00 | 1025.0000 | STOCH_EXIT | 14 | -953.54 | -0.74% | 1035.5000 | 1059.3942 | 27.7942 | M30 bullish FVG [1032.0000, 1035.5000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 69 | 2025-03-21 19:00 | 1111.2000 | 1079.9680 | 2025-03-22 02:00 | 1135.0000 | STOCH_EXIT | 10 | 2630.52 | 2.04% | 1134.1000 | 1142.4320 | 31.2320 | M30 bullish FVG [1109.0000, 1134.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 70 | 2025-03-27 10:30 | 1067.5000 | 1027.0650 | 2025-03-28 10:00 | 1059.0000 | STOCH_EXIT | 34 | -1176.28 | -0.90% | 1075.1000 | 1107.9350 | 40.4350 | M30 bullish FVG [1069.0000, 1075.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 71 | 2025-03-28 16:00 | 1060.7000 | 1021.0461 | 2025-03-28 23:30 | 1060.3000 | STOCH_EXIT | 15 | -178.18 | -0.14% | 1058.0000 | 1100.3539 | 39.6539 | M30 bullish FVG [1041.9000, 1058.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 72 | 2025-03-31 11:30 | 1037.3000 | 997.7714 | 2025-03-31 20:30 | 1023.4000 | STOCH_EXIT | 18 | -1866.29 | -1.44% | 1076.0000 | 1076.8286 | 39.5286 | M30 bullish FVG [1067.3000, 1076.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 73 | 2025-04-03 11:00 | 1002.4000 | 958.7289 | 2025-04-03 23:00 | 998.4000 | STOCH_EXIT | 24 | -640.05 | -0.50% | 1007.1000 | 1046.0711 | 43.6711 | M30 bullish FVG [1001.0000, 1007.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 74 | 2025-04-07 15:30 | 973.8000 | 933.8963 | 2025-04-07 20:00 | 985.7000 | STOCH_EXIT | 9 | 1430.55 | 1.12% | 986.0000 | 1013.7037 | 39.9037 | M30 bullish FVG [981.6000, 986.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 75 | 2025-04-08 15:30 | 989.7000 | 949.8609 | 2025-04-08 18:30 | 997.0000 | STOCH_EXIT | 6 | 819.86 | 0.64% | 984.6000 | 1029.5391 | 39.8391 | M30 bullish FVG [978.2000, 984.6000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 76 | 2025-04-10 17:00 | 997.3000 | 958.5020 | 2025-04-10 21:00 | 1003.7000 | STOCH_EXIT | 8 | 701.94 | 0.54% | 999.0000 | 1036.0980 | 38.7980 | M30 bullish FVG [994.7000, 999.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 77 | 2025-04-11 22:00 | 1008.8000 | 971.3876 | 2025-04-12 05:30 | 1012.0000 | STOCH_EXIT | 10 | 282.46 | 0.22% | 1011.5000 | 1046.2124 | 37.4124 | M30 bullish FVG [1007.5000, 1011.5000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 78 | 2025-04-16 23:30 | 1004.5000 | 970.4266 | 2025-04-17 07:00 | 1010.5000 | STOCH_EXIT | 1 | 649.03 | 0.50% | 1005.3000 | 1038.5734 | 34.0734 | M30 bullish FVG [1001.6000, 1005.3000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 79 | 2025-04-29 13:00 | 992.0000 | 967.4787 | 2025-04-29 23:30 | 992.8000 | STOCH_EXIT | 21 | -25.40 | -0.02% | 991.0000 | 1016.5213 | 24.5213 | M30 bullish FVG [987.0000, 991.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 80 | 2025-05-06 19:00 | 989.0000 | 965.7022 | 2025-05-06 20:00 | 991.7000 | STOCH_EXIT | 2 | 225.67 | 0.17% | 986.7000 | 1012.2978 | 23.2978 | M30 bullish FVG [984.0000, 986.7000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 81 | 2025-05-16 13:30 | 988.2000 | 967.4131 | 2025-05-16 16:30 | 995.0000 | STOCH_EXIT | 6 | 772.52 | 0.59% | 986.9000 | 1008.9869 | 20.7869 | M30 bullish FVG [980.8000, 986.9000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 82 | 2025-05-23 19:00 | 977.5000 | 955.6608 | 2025-05-23 19:30 | 981.3000 | STOCH_EXIT | 1 | 380.78 | 0.29% | 974.7000 | 999.3392 | 21.8392 | M30 bullish FVG [969.9000, 974.7000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 83 | 2025-05-27 11:00 | 966.3000 | 944.0249 | 2025-05-27 16:30 | 973.3000 | STOCH_EXIT | 11 | 826.14 | 0.62% | 972.9000 | 988.5751 | 22.2751 | M30 bullish FVG [961.9000, 972.9000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 84 | 2025-05-28 13:00 | 981.2000 | 958.8874 | 2025-05-28 19:30 | 982.6000 | STOCH_EXIT | 13 | 56.86 | 0.04% | 984.4000 | 1003.5126 | 22.3126 | M30 bullish FVG [982.0000, 984.4000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 85 | 2025-06-03 16:00 | 975.1000 | 955.1481 | 2025-06-03 20:30 | 980.5000 | STOCH_EXIT | 9 | 605.84 | 0.45% | 970.3000 | 995.0519 | 19.9519 | M30 bullish FVG [967.0000, 970.3000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 86 | 2025-06-05 14:30 | 976.5000 | 957.3349 | 2025-06-05 22:00 | 979.6000 | STOCH_EXIT | 15 | 290.71 | 0.22% | 978.0000 | 995.6651 | 19.1651 | M30 bullish FVG [975.9000, 978.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 87 | 2025-06-13 10:00 | 992.5000 | 975.1222 | 2025-06-14 02:00 | 975.1222 | SL | 28 | -2478.82 | -1.85% | 996.3000 | 1009.8778 | 17.3778 | M30 bullish FVG [994.0000, 996.3000], stochastic crossed below 20 | FVG stop 975.1222 |
| 88 | 2025-06-18 16:30 | 984.1000 | 967.5854 | 2025-06-18 19:00 | 992.3000 | STOCH_EXIT | 5 | 966.38 | 0.73% | 993.0000 | 1000.6146 | 16.5146 | M30 bullish FVG [991.0000, 993.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 89 | 2025-06-24 10:30 | 991.6000 | 976.5625 | 2025-06-24 13:00 | 997.5000 | STOCH_EXIT | 5 | 657.33 | 0.49% | 995.0000 | 1006.6375 | 15.0375 | M30 bullish FVG [993.3000, 995.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 90 | 2025-06-25 09:00 | 998.8000 | 983.7438 | 2025-06-25 19:30 | 1000.8000 | STOCH_EXIT | 21 | 133.03 | 0.10% | 1003.0000 | 1013.8562 | 15.0562 | M30 bullish FVG [997.5000, 1003.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 91 | 2025-06-26 21:30 | 1005.6000 | 990.4978 | 2025-06-27 09:00 | 1009.0000 | STOCH_EXIT | 9 | 318.23 | 0.24% | 1011.0000 | 1020.7022 | 15.1022 | M30 bullish FVG [1005.0000, 1011.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 92 | 2025-07-03 21:30 | 1062.4000 | 1046.3416 | 2025-07-04 07:00 | 1046.3416 | SL | 6 | -2156.21 | -1.61% | 1072.0000 | 1078.4584 | 16.0584 | M30 bullish FVG [1062.0000, 1072.0000], stochastic crossed below 20 | FVG stop 1046.3416 |
| 93 | 2025-07-04 17:00 | 1048.0000 | 1029.8029 | 2025-07-05 08:30 | 1056.1000 | STOCH_EXIT | 26 | 888.04 | 0.67% | 1068.7000 | 1066.1971 | 18.1971 | M30 bullish FVG [1066.4000, 1068.7000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 94 | 2025-07-07 21:30 | 1049.0000 | 1028.5241 | 2025-07-08 15:30 | 1058.9000 | STOCH_EXIT | 23 | 1114.60 | 0.84% | 1053.0000 | 1069.4759 | 20.4759 | M30 bullish FVG [1046.2000, 1053.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 95 | 2025-07-08 21:00 | 1056.0000 | 1034.3938 | 2025-07-09 13:00 | 1034.3938 | SL | 19 | -2876.72 | -2.14% | 1058.9000 | 1077.6062 | 21.6062 | M30 bullish FVG [1056.1000, 1058.9000], stochastic crossed below 20 | FVG stop 1034.3938 |
| 96 | 2025-07-09 17:00 | 1035.0000 | 1013.5514 | 2025-07-09 22:30 | 1039.1000 | STOCH_EXIT | 11 | 385.93 | 0.30% | 1038.3000 | 1056.4486 | 21.4486 | M30 bullish FVG [1035.5000, 1038.3000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 97 | 2025-07-10 22:00 | 1063.0000 | 1041.2977 | 2025-07-11 10:30 | 1069.9000 | STOCH_EXIT | 12 | 717.53 | 0.55% | 1061.9000 | 1084.7023 | 21.7023 | M30 bullish FVG [1059.2000, 1061.9000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 98 | 2025-07-11 14:30 | 1058.0000 | 1035.7908 | 2025-07-11 18:30 | 1035.7908 | SL | 8 | -2907.02 | -2.20% | 1066.4000 | 1080.2092 | 22.2092 | M30 bullish FVG [1062.0000, 1066.4000], stochastic crossed below 20 | FVG stop 1035.7908 |
| 99 | 2025-07-17 16:30 | 1107.0000 | 1080.3205 | 2025-07-18 10:30 | 1114.4000 | STOCH_EXIT | 22 | 729.56 | 0.57% | 1105.5000 | 1133.6795 | 26.6795 | M30 bullish FVG [1098.0000, 1105.5000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 100 | 2025-07-23 17:30 | 1133.6000 | 1105.1040 | 2025-07-23 22:30 | 1133.0000 | STOCH_EXIT | 10 | -197.60 | -0.15% | 1131.0000 | 1162.0960 | 28.4960 | M30 bullish FVG [1126.3000, 1131.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 101 | 2025-07-25 18:00 | 1119.0000 | 1091.1412 | 2025-07-25 23:00 | 1129.9000 | STOCH_EXIT | 10 | 1133.96 | 0.87% | 1138.2000 | 1146.8588 | 27.8588 | M30 bullish FVG [1132.9000, 1138.2000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 102 | 2025-07-28 17:30 | 1149.6000 | 1121.6525 | 2025-07-29 07:00 | 1149.0000 | STOCH_EXIT | 13 | -199.42 | -0.15% | 1158.8000 | 1177.5475 | 27.9475 | M30 bullish FVG [1150.4000, 1158.8000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 103 | 2025-07-31 22:30 | 1125.2000 | 1093.6325 | 2025-07-31 23:30 | 1129.5000 | STOCH_EXIT | 2 | 368.03 | 0.28% | 1117.8000 | 1156.7675 | 31.5675 | M30 bullish FVG [1114.0000, 1117.8000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 104 | 2025-08-06 09:00 | 1174.4000 | 1139.9629 | 2025-08-06 20:00 | 1170.5000 | STOCH_EXIT | 22 | -563.04 | -0.43% | 1160.0000 | 1208.8371 | 34.4371 | M30 bullish FVG [1154.0000, 1160.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 105 | 2025-08-12 14:30 | 1216.4000 | 1182.6913 | 2025-08-12 17:00 | 1240.9000 | STOCH_EXIT | 5 | 2490.03 | 1.91% | 1240.1000 | 1250.1087 | 33.7087 | M30 bullish FVG [1232.3000, 1240.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 106 | 2025-08-13 20:00 | 1247.9000 | 1213.6704 | 2025-08-14 07:00 | 1265.2000 | STOCH_EXIT | 9 | 1700.61 | 1.28% | 1248.0000 | 1282.1296 | 34.2296 | M30 bullish FVG [1243.0000, 1248.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 107 | 2025-08-14 21:00 | 1239.3000 | 1204.3440 | 2025-08-15 07:00 | 1254.8000 | STOCH_EXIT | 7 | 1539.32 | 1.15% | 1254.8000 | 1274.2560 | 34.9560 | M30 bullish FVG [1250.0000, 1254.8000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 108 | 2025-08-15 19:30 | 1260.0000 | 1224.4480 | 2025-08-15 23:30 | 1261.4000 | STOCH_EXIT | 8 | 15.04 | 0.01% | 1253.8000 | 1295.5520 | 35.5520 | M30 bullish FVG [1242.5000, 1253.8000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 109 | 2025-08-20 14:00 | 1280.3000 | 1246.5232 | 2025-08-20 20:30 | 1263.0000 | STOCH_EXIT | 13 | -1968.59 | -1.45% | 1299.0000 | 1314.0768 | 33.7768 | M30 bullish FVG [1295.0000, 1299.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 110 | 2025-08-21 11:00 | 1251.9000 | 1216.2144 | 2025-08-22 08:30 | 1254.6000 | STOCH_EXIT | 30 | 154.80 | 0.12% | 1269.7000 | 1287.5856 | 35.6856 | M30 bullish FVG [1261.5000, 1269.7000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 111 | 2025-08-29 17:00 | 1241.2000 | 1204.3143 | 2025-08-29 18:30 | 1249.6000 | STOCH_EXIT | 3 | 772.70 | 0.58% | 1253.8000 | 1278.0857 | 36.8857 | M30 bullish FVG [1248.6000, 1253.8000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 112 | 2025-09-05 15:00 | 1244.8000 | 1213.3631 | 2025-09-05 22:30 | 1246.2000 | STOCH_EXIT | 15 | 16.69 | 0.01% | 1251.1000 | 1276.2369 | 31.4369 | M30 bullish FVG [1246.5000, 1251.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 113 | 2025-09-10 22:00 | 1272.8000 | 1243.0272 | 2025-09-11 08:00 | 1279.0000 | STOCH_EXIT | 7 | 521.95 | 0.39% | 1273.5000 | 1302.5728 | 29.7728 | M30 bullish FVG [1267.2000, 1273.5000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 114 | 2025-09-11 11:00 | 1265.8000 | 1235.7539 | 2025-09-11 22:00 | 1260.4000 | STOCH_EXIT | 22 | -712.95 | -0.53% | 1277.1000 | 1295.8461 | 30.0461 | M30 bullish FVG [1273.0000, 1277.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 115 | 2025-09-16 13:00 | 1234.8000 | 1204.4087 | 2025-09-17 07:30 | 1241.1000 | STOCH_EXIT | 24 | 551.76 | 0.41% | 1245.2000 | 1265.1913 | 30.3913 | M30 bullish FVG [1238.7000, 1245.2000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 116 | 2025-09-17 10:00 | 1234.8000 | 1204.7509 | 2025-09-18 09:00 | 1204.7509 | SL | 32 | -3408.30 | -2.53% | 1235.4000 | 1264.8491 | 30.0491 | M30 bullish FVG [1230.0000, 1235.4000], stochastic crossed below 20 | FVG stop 1204.7509 |
| 117 | 2025-09-18 15:00 | 1207.1000 | 1176.4544 | 2025-09-18 17:30 | 1221.0000 | STOCH_EXIT | 5 | 1382.77 | 1.05% | 1216.4000 | 1237.7456 | 30.6456 | M30 bullish FVG [1210.0000, 1216.4000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 118 | 2025-09-22 11:30 | 1205.7000 | 1175.0184 | 2025-09-22 17:30 | 1206.3000 | STOCH_EXIT | 12 | -66.66 | -0.05% | 1222.0000 | 1236.3816 | 30.6816 | M30 bullish FVG [1214.0000, 1222.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 119 | 2025-09-25 13:00 | 1240.0000 | 1208.5717 | 2025-09-27 15:30 | 1237.2000 | STOCH_EXIT | 81 | -432.13 | -0.33% | 1234.8000 | 1271.4283 | 31.4283 | M30 bullish FVG [1231.6000, 1234.8000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 120 | 2025-09-29 07:30 | 1237.2000 | 1207.2965 | 2025-09-29 11:00 | 1244.9000 | STOCH_EXIT | 7 | 691.11 | 0.52% | 1241.0000 | 1267.1035 | 29.9035 | M30 bullish FVG [1236.8000, 1241.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 121 | 2025-09-29 16:30 | 1240.5000 | 1210.5965 | 2025-09-29 22:00 | 1210.5965 | SL | 11 | -3361.93 | -2.51% | 1241.1000 | 1270.4035 | 29.9035 | M30 bullish FVG [1234.6000, 1241.1000], stochastic crossed below 20 | FVG stop 1210.5965 |
| 122 | 2025-09-30 16:00 | 1220.0000 | 1189.3968 | 2025-09-30 18:00 | 1235.2000 | STOCH_EXIT | 4 | 1495.05 | 1.14% | 1231.0000 | 1250.6032 | 30.6032 | M30 bullish FVG [1225.5000, 1231.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 123 | 2025-10-01 16:00 | 1241.0000 | 1210.0970 | 2025-10-01 20:30 | 1249.5000 | STOCH_EXIT | 9 | 769.00 | 0.58% | 1258.1000 | 1271.9030 | 30.9030 | M30 bullish FVG [1252.1000, 1258.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 124 | 2025-10-06 18:00 | 1245.9000 | 1214.9158 | 2025-10-06 22:30 | 1256.6000 | STOCH_EXIT | 9 | 1001.57 | 0.76% | 1257.0000 | 1276.8842 | 30.9842 | M30 bullish FVG [1252.0000, 1257.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 125 | 2025-10-14 11:30 | 1272.8000 | 1237.3500 | 2025-10-14 14:30 | 1281.2000 | STOCH_EXIT | 6 | 747.92 | 0.56% | 1285.5000 | 1308.2500 | 35.4500 | M30 bullish FVG [1280.0000, 1285.5000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 126 | 2025-10-15 14:30 | 1213.0000 | 1172.0607 | 2025-10-15 22:00 | 1221.7000 | STOCH_EXIT | 15 | 823.09 | 0.62% | 1216.3000 | 1253.9393 | 40.9393 | M30 bullish FVG [1211.7000, 1216.3000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 127 | 2025-10-16 11:30 | 1216.2000 | 1175.4849 | 2025-10-16 15:00 | 1219.4000 | STOCH_EXIT | 7 | 220.02 | 0.16% | 1219.0000 | 1256.9151 | 40.7151 | M30 bullish FVG [1214.1000, 1219.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 128 | 2025-10-20 18:30 | 1241.7000 | 1201.6258 | 2025-10-20 23:30 | 1242.8000 | STOCH_EXIT | 10 | -15.51 | -0.01% | 1241.4000 | 1281.7742 | 40.0742 | M30 bullish FVG [1233.0000, 1241.4000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 129 | 2025-10-28 16:30 | 1191.8000 | 1154.8945 | 2025-10-28 20:00 | 1197.1000 | STOCH_EXIT | 7 | 463.93 | 0.34% | 1185.0000 | 1228.7055 | 36.9055 | M30 bullish FVG [1178.9000, 1185.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 130 | 2025-11-03 17:00 | 1272.4000 | 1236.7333 | 2025-11-03 20:30 | 1278.4000 | STOCH_EXIT | 7 | 500.81 | 0.37% | 1280.8000 | 1308.0667 | 35.6667 | M30 bullish FVG [1275.4000, 1280.8000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 131 | 2025-11-05 17:00 | 1281.8000 | 1245.2952 | 2025-11-05 22:30 | 1291.6000 | STOCH_EXIT | 11 | 902.41 | 0.66% | 1297.1000 | 1318.3048 | 36.5048 | M30 bullish FVG [1284.9000, 1297.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 132 | 2025-11-06 11:00 | 1285.4000 | 1248.5813 | 2025-11-06 16:30 | 1292.5000 | STOCH_EXIT | 11 | 615.97 | 0.45% | 1297.4000 | 1322.2187 | 36.8187 | M30 bullish FVG [1292.9000, 1297.4000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 133 | 2025-11-13 14:30 | 1296.5000 | 1261.8246 | 2025-11-13 19:30 | 1298.0000 | STOCH_EXIT | 10 | 21.49 | 0.02% | 1311.8000 | 1331.1754 | 34.6754 | M30 bullish FVG [1306.1000, 1311.8000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 98803.49
- Макс: 138389.41
- Финал: 138225.80

Полная кривая — в `*_equity.csv`.
