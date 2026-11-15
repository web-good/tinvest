# Бэктест MDMG (Minutes30)

- Период: 2023-11-15 22:36 — 2025-11-15 22:36
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005
- ⚠️ На конце прогона осталась открытая позиция (оценена mark-to-market)

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0 |
| MaxWaitBars | 30 |
| StochPeriod | 21 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 2 |
| HTFTrendEMA | 50 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 161 |
| Выигрышных / проигрышных | 118 / 43 |
| Win rate | 73.29% |
| Gross profit / loss | 163737.08 / 99764.43 |
| Profit factor | 1.641 |
| Чистый PnL | 62943.92 (62.94%) |
| Стартовый / финальный капитал | 100000.00 / 162943.92 |
| Макс. просадка | 9798.09 (6.13%) |
| Средняя прибыль / убыток | 1387.60 / 2320.10 |
| Expectancy | 397.35 |
| Лучшая / худшая сделка | 9591.66 / -5198.14 |
| Exposure | 17.45% |
| CAGR | 27.61% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-12-21 12:00 | 788.5000 | 757.0581 | 2023-12-22 10:00 | 796.6000 | STOCH_EXIT | 15 | 920.74 | 0.93% | 797.6000 | 851.3839 | 31.4419 | M30 bullish FVG [794.0000, 797.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 2 | 2023-12-26 11:30 | 811.0000 | 780.5514 | 2023-12-27 14:00 | 807.9000 | STOCH_EXIT | 24 | -484.77 | -0.48% | 820.4000 | 871.8971 | 30.4486 | M30 bullish FVG [809.0000, 820.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 3 | 2023-12-28 11:30 | 793.8000 | 763.4234 | 2023-12-29 14:30 | 763.4234 | SL | 25 | -3925.55 | -3.92% | 803.2000 | 854.5531 | 30.3766 | M30 bullish FVG [801.0000, 803.2000], stochastic crossed below 20 | FVG stop 763.4234 |
| 4 | 2024-01-05 11:30 | 806.5000 | 776.9481 | 2024-01-08 10:00 | 822.9000 | STOCH_EXIT | 16 | 1854.65 | 1.93% | 803.5000 | 865.6038 | 29.5519 | M30 bullish FVG [802.8000, 803.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 5 | 2024-01-08 18:30 | 808.1000 | 779.4447 | 2024-01-10 11:00 | 837.1000 | STOCH_EXIT | 23 | 3409.47 | 3.49% | 818.1000 | 865.4107 | 28.6553 | M30 bullish FVG [815.1000, 818.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 6 | 2024-01-16 10:30 | 817.0000 | 789.2608 | 2024-01-17 16:00 | 817.0000 | STOCH_EXIT | 30 | -101.31 | -0.10% | 825.1000 | 872.4784 | 27.7392 | M30 bullish FVG [825.0000, 825.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 7 | 2024-01-22 13:00 | 800.0000 | 774.9856 | 2024-01-22 16:00 | 808.0000 | STOCH_EXIT | 6 | 913.89 | 0.90% | 804.9000 | 850.0289 | 25.0144 | M30 bullish FVG [800.0000, 804.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 8 | 2024-01-23 17:00 | 807.7000 | 783.1152 | 2024-01-25 10:00 | 818.9000 | STOCH_EXIT | 24 | 1308.72 | 1.29% | 814.0000 | 856.8697 | 24.5848 | M30 bullish FVG [809.0000, 814.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 9 | 2024-02-05 12:00 | 883.3000 | 858.7351 | 2024-02-05 14:00 | 912.4000 | STOCH_EXIT | 4 | 3299.65 | 3.19% | 891.9000 | 932.4298 | 24.5649 | M30 bullish FVG [891.8000, 891.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 10 | 2024-02-07 11:30 | 979.0000 | 948.2318 | 2024-02-08 10:30 | 948.2318 | SL | 17 | -3458.77 | -3.24% | 992.2000 | 1040.5364 | 30.7682 | M30 bullish FVG [990.2000, 992.2000], stochastic crossed below 20 | FVG stop 948.2318 |
| 11 | 2024-02-08 13:30 | 944.2000 | 911.8224 | 2024-02-09 12:00 | 969.4000 | STOCH_EXIT | 16 | 2642.51 | 2.57% | 992.2000 | 1008.9552 | 32.3776 | M30 bullish FVG [990.2000, 992.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 12 | 2024-02-12 12:30 | 954.3000 | 920.2076 | 2024-02-14 09:30 | 967.0000 | STOCH_EXIT | 29 | 1303.07 | 1.23% | 978.5000 | 1022.4849 | 34.0924 | M30 bullish FVG [975.0000, 978.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 13 | 2024-02-16 14:30 | 993.8000 | 959.8828 | 2024-02-19 14:00 | 1015.4000 | STOCH_EXIT | 18 | 2224.30 | 2.07% | 1011.3000 | 1061.6344 | 33.9172 | M30 bullish FVG [1009.8000, 1011.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 14 | 2024-02-20 10:30 | 984.2000 | 949.2337 | 2024-02-21 10:00 | 949.2337 | SL | 18 | -3988.57 | -3.65% | 1012.9000 | 1054.1327 | 34.9663 | M30 bullish FVG [1009.4000, 1012.9000], stochastic crossed below 20 | FVG stop 949.2337 |
| 15 | 2024-02-28 17:30 | 958.1000 | 922.9186 | 2024-03-01 15:00 | 983.4000 | STOCH_EXIT | 33 | 2676.22 | 2.54% | 991.8000 | 1028.4628 | 35.1814 | M30 bullish FVG [978.9000, 991.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 16 | 2024-03-07 13:00 | 1015.7000 | 978.9098 | 2024-03-07 14:00 | 1036.3000 | STOCH_EXIT | 2 | 2074.84 | 1.93% | 1028.3000 | 1089.2805 | 36.7902 | M30 bullish FVG [1026.0000, 1028.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 17 | 2024-03-11 17:30 | 1025.6000 | 989.1734 | 2024-03-14 10:00 | 989.1734 | SL | 42 | -4005.44 | -3.65% | 1030.9000 | 1098.4533 | 36.4266 | M30 bullish FVG [1016.6000, 1030.9000], stochastic crossed below 20 | FVG stop 989.1734 |
| 18 | 2024-03-20 12:00 | 1001.6000 | 967.5142 | 2024-03-21 11:30 | 1008.9000 | STOCH_EXIT | 18 | 667.24 | 0.63% | 1013.0000 | 1069.7717 | 34.0858 | M30 bullish FVG [1008.7000, 1013.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 19 | 2024-03-22 16:30 | 1001.7000 | 968.7015 | 2024-03-29 12:30 | 988.2000 | STOCH_EXIT | 87 | -1550.96 | -1.45% | 1013.4000 | 1067.6970 | 32.9985 | M30 bullish FVG [1012.9000, 1013.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 20 | 2024-04-04 13:30 | 1041.0000 | 1008.9511 | 2024-04-04 17:30 | 1008.9511 | SL | 8 | -3340.46 | -3.18% | 1070.0000 | 1105.0977 | 32.0489 | M30 bullish FVG [1060.0000, 1070.0000], stochastic crossed below 20 | FVG stop 1008.9511 |
| 21 | 2024-04-05 13:00 | 1012.9000 | 977.7189 | 2024-04-06 10:00 | 1026.0000 | STOCH_EXIT | 12 | 1220.14 | 1.19% | 1019.3000 | 1083.2622 | 35.1811 | M30 bullish FVG [1012.6000, 1019.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 22 | 2024-04-08 10:00 | 1018.5000 | 984.4104 | 2024-04-08 15:00 | 1032.3000 | STOCH_EXIT | 10 | 1290.23 | 1.25% | 1025.1000 | 1086.6792 | 34.0896 | M30 bullish FVG [1025.0000, 1025.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 23 | 2024-04-09 12:30 | 1012.9000 | 979.8311 | 2024-04-11 12:30 | 1006.5000 | STOCH_EXIT | 38 | -763.20 | -0.73% | 1026.0000 | 1079.0378 | 33.0689 | M30 bullish FVG [1025.0000, 1026.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 24 | 2024-04-17 14:30 | 1036.3000 | 1005.9479 | 2024-04-17 18:30 | 1047.8000 | STOCH_EXIT | 8 | 1045.80 | 1.01% | 1047.3000 | 1097.0042 | 30.3521 | M30 bullish FVG [1046.5000, 1047.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 25 | 2024-04-19 12:00 | 1038.7000 | 1010.4576 | 2024-04-20 12:00 | 1049.3000 | STOCH_EXIT | 18 | 965.16 | 0.92% | 1045.2000 | 1095.1848 | 28.2424 | M30 bullish FVG [1044.1000, 1045.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 26 | 2024-04-23 12:30 | 1046.4000 | 1017.5777 | 2024-04-24 13:30 | 1050.6000 | STOCH_EXIT | 21 | 318.30 | 0.30% | 1057.0000 | 1104.0445 | 28.8223 | M30 bullish FVG [1053.8000, 1057.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 27 | 2024-04-30 10:30 | 1104.1000 | 1074.1629 | 2024-05-01 22:30 | 1111.0000 | STOCH_EXIT | 42 | 556.08 | 0.52% | 1115.5000 | 1163.9741 | 29.9371 | M30 bullish FVG [1114.5000, 1115.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 28 | 2024-05-03 13:30 | 1136.7000 | 1104.3670 | 2024-05-04 15:00 | 1154.8000 | STOCH_EXIT | 21 | 1593.70 | 1.49% | 1133.9000 | 1201.3659 | 32.3330 | M30 bullish FVG [1115.0000, 1133.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 29 | 2024-05-07 11:00 | 1122.4000 | 1089.5527 | 2024-05-08 15:30 | 1138.5000 | STOCH_EXIT | 28 | 1437.08 | 1.33% | 1133.0000 | 1188.0946 | 32.8473 | M30 bullish FVG [1130.3000, 1133.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 30 | 2024-05-15 13:00 | 1179.1000 | 1144.8545 | 2024-05-15 15:00 | 1144.8545 | SL | 4 | -3292.90 | -3.00% | 1194.9000 | 1247.5910 | 34.2455 | M30 bullish FVG [1188.5000, 1194.9000], stochastic crossed below 20 | FVG stop 1144.8545 |
| 31 | 2024-05-16 16:00 | 1102.7000 | 1063.7863 | 2024-05-16 17:00 | 1063.7863 | SL | 2 | -3839.70 | -3.63% | 1118.4000 | 1180.5274 | 38.9137 | M30 bullish FVG [1115.0000, 1118.4000], stochastic crossed below 20 | FVG stop 1063.7863 |
| 32 | 2024-05-17 11:30 | 1064.1000 | 1022.1516 | 2024-05-17 18:00 | 1088.1000 | STOCH_EXIT | 13 | 2200.69 | 2.15% | 1074.0000 | 1147.9968 | 41.9484 | M30 bullish FVG [1056.7000, 1074.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 33 | 2024-05-20 14:30 | 1061.3000 | 1019.2408 | 2024-05-21 10:00 | 1145.4185 | TP | 10 | 8218.50 | 7.82% | 1064.9000 | 1145.4185 | 42.0592 | M30 bullish FVG [1064.8000, 1064.9000], stochastic crossed below 20 | FVG target 1145.4185 |
| 34 | 2024-05-22 16:00 | 1070.1000 | 1023.8673 | 2024-06-17 09:30 | 1162.5654 | TP | 6 | 9591.66 | 8.53% | 1088.3000 | 1162.5654 | 46.2327 | M30 bullish FVG [1087.9000, 1088.3000], stochastic crossed below 20 | FVG target 1162.5654 |
| 35 | 2024-06-17 16:30 | 1101.2000 | 1055.4482 | 2024-06-17 18:00 | 1055.4482 | SL | 3 | -5198.14 | -4.25% | 1271.7000 | 1192.7036 | 45.7518 | M30 bullish FVG [1179.2000, 1271.7000], stochastic crossed below 20 | FVG stop 1055.4482 |
| 36 | 2024-09-23 10:00 | 873.0000 | 849.7417 | 2024-09-23 17:00 | 887.0000 | STOCH_EXIT | 14 | 1758.08 | 1.50% | 873.6000 | 919.5165 | 23.2583 | M30 bullish FVG [873.5000, 873.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 37 | 2024-09-24 10:30 | 861.7000 | 838.1602 | 2024-09-25 09:30 | 873.8000 | STOCH_EXIT | 32 | 1550.05 | 1.30% | 883.2000 | 908.7797 | 23.5398 | M30 bullish FVG [882.9000, 883.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 38 | 2024-09-25 23:30 | 857.6000 | 833.2702 | 2024-09-26 07:00 | 870.3000 | STOCH_EXIT | 1 | 1668.88 | 1.38% | 860.5000 | 906.2597 | 24.3298 | M30 bullish FVG [859.0000, 860.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 39 | 2024-09-26 22:30 | 875.6000 | 851.4009 | 2024-09-27 11:00 | 886.7000 | STOCH_EXIT | 11 | 1430.64 | 1.17% | 879.6000 | 923.9983 | 24.1991 | M30 bullish FVG [877.7000, 879.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 40 | 2024-09-27 21:00 | 884.1000 | 860.3651 | 2024-09-28 13:00 | 891.0000 | STOCH_EXIT | 18 | 841.74 | 0.68% | 890.4000 | 931.5698 | 23.7349 | M30 bullish FVG [888.9000, 890.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 41 | 2024-10-02 18:30 | 886.8000 | 861.7745 | 2024-10-03 15:30 | 861.7745 | SL | 28 | -3625.97 | -2.92% | 904.1000 | 936.8509 | 25.0255 | M30 bullish FVG [903.8000, 904.1000], stochastic crossed below 20 | FVG stop 861.7745 |
| 42 | 2024-10-04 13:00 | 870.9000 | 844.8112 | 2024-10-05 07:00 | 876.4000 | STOCH_EXIT | 22 | 643.06 | 0.53% | 882.0000 | 923.0776 | 26.0888 | M30 bullish FVG [880.0000, 882.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 43 | 2024-10-07 11:30 | 875.2000 | 849.5104 | 2024-10-08 13:00 | 869.6000 | STOCH_EXIT | 37 | -899.66 | -0.74% | 881.3000 | 926.5792 | 25.6896 | M30 bullish FVG [877.0000, 881.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 44 | 2024-10-10 11:00 | 868.8000 | 844.5919 | 2024-10-11 16:30 | 875.8000 | STOCH_EXIT | 45 | 851.75 | 0.70% | 870.9000 | 917.2162 | 24.2081 | M30 bullish FVG [870.7000, 870.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 45 | 2024-10-11 22:00 | 870.7000 | 847.1996 | 2024-10-12 23:00 | 868.8000 | STOCH_EXIT | 36 | -387.76 | -0.32% | 872.0000 | 917.7008 | 23.5004 | M30 bullish FVG [871.0000, 872.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 46 | 2024-10-14 08:00 | 869.8000 | 847.3068 | 2024-10-14 09:30 | 873.0000 | STOCH_EXIT | 3 | 323.68 | 0.27% | 870.0000 | 914.7864 | 22.4932 | M30 bullish FVG [869.8000, 870.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 47 | 2024-10-16 10:00 | 882.0000 | 860.3706 | 2024-10-16 11:00 | 891.6000 | STOCH_EXIT | 2 | 1202.42 | 0.99% | 888.5000 | 925.2587 | 21.6294 | M30 bullish FVG [887.0000, 888.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 48 | 2024-10-16 19:30 | 883.2000 | 861.5706 | 2024-10-17 08:30 | 888.9000 | STOCH_EXIT | 12 | 669.14 | 0.54% | 886.3000 | 926.4587 | 21.6294 | M30 bullish FVG [885.4000, 886.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 49 | 2024-10-17 13:00 | 881.9000 | 860.7799 | 2024-10-17 19:00 | 860.3000 | SL | 12 | -3145.95 | -2.55% | 888.0000 | 924.1402 | 21.1201 | M30 bullish FVG [887.7000, 888.0000], stochastic crossed below 20 | FVG stop 860.7799 |
| 50 | 2024-10-18 11:00 | 864.8000 | 843.7813 | 2024-10-19 07:00 | 868.3000 | STOCH_EXIT | 26 | 366.05 | 0.30% | 871.0000 | 906.8374 | 21.0187 | M30 bullish FVG [870.0000, 871.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 51 | 2024-10-31 17:00 | 855.4000 | 823.3090 | 2024-11-01 10:30 | 860.1000 | STOCH_EXIT | 21 | 541.76 | 0.45% | 860.6000 | 919.5821 | 32.0910 | M30 bullish FVG [860.0000, 860.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 52 | 2024-11-13 20:00 | 877.1000 | 849.4862 | 2024-11-14 13:00 | 881.9000 | STOCH_EXIT | 20 | 541.03 | 0.45% | 882.9000 | 932.3277 | 27.6138 | M30 bullish FVG [879.5000, 882.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 53 | 2024-11-14 21:00 | 869.4000 | 843.0729 | 2024-11-15 12:30 | 882.8000 | STOCH_EXIT | 17 | 1753.35 | 1.44% | 880.5000 | 922.0543 | 26.3271 | M30 bullish FVG [879.7000, 880.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 54 | 2024-11-19 12:30 | 875.3000 | 847.9642 | 2024-11-20 10:00 | 884.3000 | STOCH_EXIT | 29 | 1144.95 | 0.93% | 894.6000 | 929.9717 | 27.3358 | M30 bullish FVG [893.8000, 894.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 55 | 2024-11-20 18:30 | 867.5000 | 839.5453 | 2024-11-21 10:00 | 877.1000 | STOCH_EXIT | 17 | 1256.79 | 1.01% | 885.3000 | 923.4094 | 27.9547 | M30 bullish FVG [885.0000, 885.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 56 | 2024-11-21 18:00 | 864.6000 | 836.9635 | 2024-11-22 10:00 | 873.3000 | STOCH_EXIT | 18 | 1135.50 | 0.91% | 873.0000 | 919.8730 | 27.6365 | M30 bullish FVG [869.5000, 873.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 57 | 2024-11-22 20:30 | 867.1000 | 840.0661 | 2024-11-23 14:00 | 874.8000 | STOCH_EXIT | 21 | 997.04 | 0.79% | 857.5000 | 921.1678 | 27.0339 | M30 bullish FVG [857.1000, 857.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 58 | 2024-11-25 18:00 | 872.6000 | 846.4900 | 2024-11-26 19:00 | 840.2000 | SL | 36 | -4888.69 | -3.81% | 885.2000 | 924.8201 | 26.1100 | M30 bullish FVG [883.9000, 885.2000], stochastic crossed below 20 | FVG stop 846.4900 |
| 59 | 2024-12-23 15:00 | 868.8000 | 850.5023 | 2024-12-24 14:30 | 850.5023 | SL | 33 | -2720.35 | -2.20% | 866.2000 | 905.3954 | 18.2977 | M30 bullish FVG [865.3000, 866.2000], stochastic crossed below 20 | FVG stop 850.5023 |
| 60 | 2024-12-24 18:30 | 853.5000 | 835.2236 | 2024-12-26 09:30 | 859.9000 | STOCH_EXIT | 50 | 781.61 | 0.65% | 853.5000 | 890.0529 | 18.2764 | M30 bullish FVG [851.8000, 853.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 61 | 2024-12-27 11:30 | 852.5000 | 833.7478 | 2024-12-27 23:00 | 856.7000 | STOCH_EXIT | 23 | 475.05 | 0.39% | 865.0000 | 890.0043 | 18.7522 | M30 bullish FVG [857.7000, 865.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 62 | 2024-12-31 13:00 | 894.7000 | 874.2775 | 2024-12-31 19:00 | 900.9000 | STOCH_EXIT | 12 | 721.10 | 0.59% | 898.0000 | 935.5451 | 20.4225 | M30 bullish FVG [896.5000, 898.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 63 | 2025-01-07 15:00 | 940.9000 | 915.9515 | 2025-01-08 07:30 | 943.8000 | STOCH_EXIT | 19 | 254.49 | 0.21% | 944.8000 | 990.7970 | 24.9485 | M30 bullish FVG [944.6000, 944.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 64 | 2025-01-10 12:00 | 912.9000 | 886.7883 | 2025-01-10 14:00 | 934.6000 | STOCH_EXIT | 4 | 2784.02 | 2.27% | 930.9000 | 965.1235 | 26.1117 | M30 bullish FVG [923.6000, 930.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 65 | 2025-01-15 14:00 | 936.2000 | 910.4138 | 2025-01-15 22:00 | 946.9000 | STOCH_EXIT | 16 | 1307.63 | 1.04% | 940.2000 | 987.7724 | 25.7862 | M30 bullish FVG [937.6000, 940.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 66 | 2025-01-20 23:00 | 928.9000 | 905.9171 | 2025-01-21 19:30 | 940.8000 | STOCH_EXIT | 27 | 1491.26 | 1.18% | 938.5000 | 974.8658 | 22.9829 | M30 bullish FVG [936.5000, 938.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 67 | 2025-01-24 17:30 | 969.9000 | 944.0126 | 2025-01-25 09:30 | 973.5000 | STOCH_EXIT | 28 | 346.94 | 0.27% | 979.3000 | 1021.6748 | 25.8874 | M30 bullish FVG [977.0000, 979.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 68 | 2025-01-27 23:00 | 991.4000 | 966.0760 | 2025-01-28 18:30 | 1010.0000 | STOCH_EXIT | 25 | 2287.91 | 1.77% | 1008.6000 | 1042.0480 | 25.3240 | M30 bullish FVG [1001.9000, 1008.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 69 | 2025-02-07 15:30 | 991.6000 | 962.5383 | 2025-02-09 02:30 | 999.8000 | STOCH_EXIT | 61 | 950.97 | 0.73% | 1003.2000 | 1049.7233 | 29.0617 | M30 bullish FVG [1001.0000, 1003.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 70 | 2025-02-10 23:00 | 1002.6000 | 971.9427 | 2025-02-12 07:00 | 1014.7000 | STOCH_EXIT | 36 | 1452.97 | 1.11% | 1011.8000 | 1063.9145 | 30.6573 | M30 bullish FVG [1011.0000, 1011.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 71 | 2025-02-13 14:00 | 1026.4000 | 998.1467 | 2025-02-15 03:00 | 1039.7000 | STOCH_EXIT | 56 | 1594.70 | 1.19% | 1022.1000 | 1082.9066 | 28.2533 | M30 bullish FVG [1022.0000, 1022.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 72 | 2025-02-17 10:00 | 1036.2000 | 1005.8959 | 2025-02-17 12:00 | 1049.6000 | STOCH_EXIT | 4 | 1606.42 | 1.19% | 1036.6000 | 1096.8083 | 30.3041 | M30 bullish FVG [1036.5000, 1036.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 73 | 2025-02-18 14:30 | 1030.0000 | 1000.4604 | 2025-02-19 19:00 | 1028.8000 | STOCH_EXIT | 43 | -294.28 | -0.22% | 1029.2000 | 1089.0791 | 29.5396 | M30 bullish FVG [1026.9000, 1029.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 74 | 2025-02-21 21:00 | 1035.9000 | 1006.6637 | 2025-02-22 03:00 | 1045.5000 | STOCH_EXIT | 7 | 1121.27 | 0.83% | 1039.0000 | 1094.3725 | 29.2363 | M30 bullish FVG [1036.6000, 1039.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 75 | 2025-02-27 12:00 | 1001.2000 | 973.5301 | 2025-02-27 17:30 | 1027.8000 | STOCH_EXIT | 11 | 3505.21 | 2.55% | 1031.4000 | 1056.5399 | 27.6699 | M30 bullish FVG [1027.5000, 1031.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 76 | 2025-02-28 21:30 | 1008.6000 | 980.9208 | 2025-03-01 12:30 | 1020.5000 | STOCH_EXIT | 26 | 1513.08 | 1.08% | 1017.8000 | 1063.9584 | 27.6792 | M30 bullish FVG [1016.3000, 1017.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 77 | 2025-03-04 15:30 | 1036.2000 | 1008.1888 | 2025-03-05 11:00 | 1052.0000 | STOCH_EXIT | 25 | 2021.56 | 1.42% | 1051.3000 | 1092.2223 | 28.0112 | M30 bullish FVG [1048.0000, 1051.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 78 | 2025-03-05 22:30 | 1047.3000 | 1019.3111 | 2025-03-06 09:00 | 1056.0000 | STOCH_EXIT | 7 | 1055.47 | 0.73% | 1051.7000 | 1103.2779 | 27.9889 | M30 bullish FVG [1050.8000, 1051.7000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 79 | 2025-03-06 21:30 | 1060.9000 | 1033.8388 | 2025-03-07 17:00 | 1033.8388 | SL | 25 | -3850.87 | -2.65% | 1072.5000 | 1115.0223 | 27.0612 | M30 bullish FVG [1070.6000, 1072.5000], stochastic crossed below 20 | FVG stop 1033.8388 |
| 80 | 2025-03-11 13:00 | 1059.9000 | 1031.8784 | 2025-03-12 07:30 | 1053.9000 | STOCH_EXIT | 23 | -938.57 | -0.67% | 1067.0000 | 1115.9433 | 28.0216 | M30 bullish FVG [1066.9000, 1067.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 81 | 2025-03-12 19:00 | 1027.6000 | 999.4371 | 2025-03-12 23:00 | 1040.6000 | STOCH_EXIT | 8 | 1639.33 | 1.16% | 1035.5000 | 1083.9259 | 28.1629 | M30 bullish FVG [1032.0000, 1035.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 82 | 2025-03-13 11:00 | 1017.0000 | 989.2058 | 2025-03-14 10:00 | 1033.8000 | STOCH_EXIT | 32 | 2208.44 | 1.55% | 1040.0000 | 1072.5883 | 27.7942 | M30 bullish FVG [1039.7000, 1040.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 83 | 2025-03-17 10:00 | 1041.0000 | 1013.7020 | 2025-03-18 08:30 | 1049.2000 | STOCH_EXIT | 31 | 994.53 | 0.69% | 1043.0000 | 1095.5961 | 27.2980 | M30 bullish FVG [1042.2000, 1043.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 84 | 2025-03-19 11:30 | 1020.2000 | 991.6777 | 2025-03-19 12:30 | 1031.5000 | STOCH_EXIT | 2 | 1458.93 | 1.01% | 1025.0000 | 1077.2446 | 28.5223 | M30 bullish FVG [1024.1000, 1025.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 85 | 2025-03-19 19:00 | 1023.7000 | 995.1777 | 2025-03-20 08:00 | 1028.8000 | STOCH_EXIT | 12 | 582.55 | 0.40% | 1026.1000 | 1080.7446 | 28.5223 | M30 bullish FVG [1026.0000, 1026.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 86 | 2025-03-24 12:30 | 1117.5000 | 1080.2060 | 2025-03-24 17:00 | 1080.2060 | SL | 9 | -5067.85 | -3.43% | 1126.1000 | 1192.0879 | 37.2940 | M30 bullish FVG [1123.0000, 1126.1000], stochastic crossed below 20 | FVG stop 1080.2060 |
| 87 | 2025-03-24 21:30 | 1084.5000 | 1047.2060 | 2025-03-25 08:30 | 1102.7000 | STOCH_EXIT | 9 | 2240.94 | 1.58% | 1093.0000 | 1159.0879 | 37.2940 | M30 bullish FVG [1090.9000, 1093.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 88 | 2025-03-25 13:00 | 1083.1000 | 1042.8985 | 2025-03-25 23:00 | 1083.0000 | STOCH_EXIT | 20 | -157.35 | -0.11% | 1105.1000 | 1163.5031 | 40.2015 | M30 bullish FVG [1103.5000, 1105.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 89 | 2025-03-26 18:00 | 1070.7000 | 1029.7700 | 2025-03-28 10:00 | 1059.0000 | STOCH_EXIT | 54 | -1723.25 | -1.19% | 1081.0000 | 1152.5600 | 40.9300 | M30 bullish FVG [1079.7000, 1081.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 90 | 2025-03-28 21:30 | 1050.9000 | 1011.2461 | 2025-03-29 02:30 | 1066.9000 | STOCH_EXIT | 6 | 2031.99 | 1.42% | 1058.0000 | 1130.2079 | 39.6539 | M30 bullish FVG [1041.9000, 1058.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 91 | 2025-03-31 11:30 | 1037.3000 | 997.7714 | 2025-04-01 06:30 | 1027.7000 | STOCH_EXIT | 25 | -1477.92 | -1.02% | 1076.0000 | 1116.3573 | 39.5286 | M30 bullish FVG [1067.3000, 1076.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 92 | 2025-04-01 11:30 | 1013.8000 | 969.0305 | 2025-04-02 21:00 | 1005.5000 | STOCH_EXIT | 54 | -1312.66 | -0.92% | 1032.1000 | 1103.3389 | 44.7695 | M30 bullish FVG [1027.7000, 1032.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 93 | 2025-04-14 07:00 | 1013.8000 | 977.6599 | 2025-04-15 09:00 | 1003.3000 | STOCH_EXIT | 39 | -1611.20 | -1.13% | 1021.6000 | 1086.0803 | 36.1401 | M30 bullish FVG [1021.0000, 1021.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 94 | 2025-05-09 03:00 | 999.2000 | 977.9655 | 2025-05-09 07:30 | 1003.8000 | STOCH_EXIT | 9 | 503.79 | 0.36% | 999.2000 | 1041.6691 | 21.2345 | M30 bullish FVG [998.8000, 999.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 95 | 2025-05-09 16:00 | 998.1000 | 976.8655 | 2025-05-09 23:00 | 1002.0000 | STOCH_EXIT | 14 | 408.89 | 0.29% | 1000.0000 | 1040.5691 | 21.2345 | M30 bullish FVG [998.1000, 1000.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 96 | 2025-05-12 12:00 | 1008.2000 | 986.9655 | 2025-05-13 07:30 | 1003.7000 | STOCH_EXIT | 25 | -770.83 | -0.55% | 1006.7000 | 1050.6691 | 21.2345 | M30 bullish FVG [1006.0000, 1006.7000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 97 | 2025-05-13 12:00 | 998.8000 | 976.9608 | 2025-05-14 06:30 | 1000.0000 | STOCH_EXIT | 24 | 28.28 | 0.02% | 999.0000 | 1042.4784 | 21.8392 | M30 bullish FVG [998.9000, 999.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 98 | 2025-06-12 06:30 | 987.6000 | 970.2222 | 2025-06-12 21:00 | 992.4000 | STOCH_EXIT | 29 | 541.02 | 0.39% | 989.0000 | 1022.3556 | 17.3778 | M30 bullish FVG [988.8000, 989.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 99 | 2025-06-13 11:00 | 985.9000 | 968.5222 | 2025-06-14 08:30 | 981.1000 | STOCH_EXIT | 39 | -827.04 | -0.59% | 996.3000 | 1020.6556 | 17.3778 | M30 bullish FVG [994.0000, 996.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 100 | 2025-06-17 07:30 | 986.7000 | 969.5074 | 2025-06-17 13:30 | 990.5000 | STOCH_EXIT | 12 | 399.22 | 0.28% | 987.5000 | 1021.0852 | 17.1926 | M30 bullish FVG [987.2000, 987.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 101 | 2025-06-18 16:30 | 984.1000 | 967.5854 | 2025-06-18 19:00 | 992.3000 | STOCH_EXIT | 5 | 1031.29 | 0.73% | 993.0000 | 1017.1291 | 16.5146 | M30 bullish FVG [991.0000, 993.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 102 | 2025-06-19 14:00 | 990.7000 | 974.6151 | 2025-06-19 18:00 | 994.6000 | STOCH_EXIT | 8 | 415.75 | 0.29% | 991.0000 | 1022.8699 | 16.0849 | M30 bullish FVG [989.8000, 991.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 103 | 2025-06-20 19:30 | 993.4000 | 977.6068 | 2025-06-20 23:30 | 997.1000 | STOCH_EXIT | 8 | 386.78 | 0.27% | 996.3000 | 1024.9863 | 15.7932 | M30 bullish FVG [996.1000, 996.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 104 | 2025-06-24 10:30 | 991.6000 | 976.5625 | 2025-06-24 13:00 | 997.5000 | STOCH_EXIT | 5 | 706.38 | 0.49% | 995.0000 | 1021.6750 | 15.0375 | M30 bullish FVG [993.3000, 995.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 105 | 2025-06-25 07:00 | 997.5000 | 982.4438 | 2025-06-25 19:30 | 1000.8000 | STOCH_EXIT | 25 | 329.02 | 0.23% | 997.8000 | 1027.6125 | 15.0562 | M30 bullish FVG [997.5000, 997.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 106 | 2025-06-26 23:00 | 1004.0000 | 988.8978 | 2025-06-27 11:00 | 1013.6000 | STOCH_EXIT | 10 | 1228.54 | 0.86% | 1011.0000 | 1034.2044 | 15.1022 | M30 bullish FVG [1005.0000, 1011.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 107 | 2025-06-30 07:00 | 1017.0000 | 1001.0935 | 2025-06-30 10:30 | 1024.2000 | STOCH_EXIT | 7 | 877.47 | 0.61% | 1014.2000 | 1048.8130 | 15.9065 | M30 bullish FVG [1013.4000, 1014.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 108 | 2025-07-01 10:30 | 1024.1000 | 1007.7654 | 2025-07-01 15:00 | 1028.1000 | STOCH_EXIT | 9 | 422.29 | 0.29% | 1029.6000 | 1056.7692 | 16.3346 | M30 bullish FVG [1029.1000, 1029.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 109 | 2025-07-04 07:30 | 1031.6000 | 1013.4029 | 2025-07-04 11:30 | 1063.1000 | STOCH_EXIT | 8 | 4293.82 | 2.95% | 1072.0000 | 1067.9942 | 18.1971 | M30 bullish FVG [1062.0000, 1072.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 110 | 2025-07-04 20:00 | 1046.6000 | 1028.4029 | 2025-07-05 08:30 | 1056.1000 | STOCH_EXIT | 20 | 1208.16 | 0.81% | 1050.4000 | 1082.9942 | 18.1971 | M30 bullish FVG [1047.6000, 1050.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 111 | 2025-07-07 07:30 | 1047.3000 | 1026.8241 | 2025-07-07 16:00 | 1053.5000 | STOCH_EXIT | 17 | 746.69 | 0.49% | 1053.9000 | 1088.2517 | 20.4759 | M30 bullish FVG [1053.5000, 1053.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 112 | 2025-07-08 08:30 | 1047.3000 | 1025.6938 | 2025-07-08 17:00 | 1060.6000 | STOCH_EXIT | 17 | 1775.68 | 1.17% | 1053.0000 | 1090.5123 | 21.6062 | M30 bullish FVG [1046.2000, 1053.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 113 | 2025-07-08 22:00 | 1049.8000 | 1028.1938 | 2025-07-09 23:30 | 1039.7000 | STOCH_EXIT | 38 | -1638.28 | -1.06% | 1058.9000 | 1093.0123 | 21.6062 | M30 bullish FVG [1056.1000, 1058.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 114 | 2025-07-11 09:00 | 1059.1000 | 1036.8908 | 2025-07-11 10:30 | 1069.9000 | STOCH_EXIT | 3 | 1401.91 | 0.92% | 1066.5000 | 1103.5185 | 22.2092 | M30 bullish FVG [1064.8000, 1066.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 115 | 2025-07-11 14:30 | 1058.0000 | 1035.7908 | 2025-07-11 18:30 | 1035.7908 | SL | 8 | -3372.14 | -2.20% | 1066.4000 | 1102.4185 | 22.2092 | M30 bullish FVG [1062.0000, 1066.4000], stochastic crossed below 20 | FVG stop 1035.7908 |
| 116 | 2025-07-11 21:00 | 1041.0000 | 1018.7908 | 2025-07-12 05:00 | 1018.7908 | SL | 11 | -3346.44 | -2.23% | 1044.1000 | 1085.4185 | 22.2092 | M30 bullish FVG [1043.0000, 1044.1000], stochastic crossed below 20 | FVG stop 1018.7908 |
| 117 | 2025-07-16 11:00 | 1088.0000 | 1060.6682 | 2025-07-16 12:00 | 1099.3000 | STOCH_EXIT | 2 | 1377.86 | 0.94% | 1097.0000 | 1142.6636 | 27.3318 | M30 bullish FVG [1096.0000, 1097.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 118 | 2025-07-17 18:00 | 1103.4000 | 1076.7205 | 2025-07-18 10:30 | 1114.4000 | STOCH_EXIT | 19 | 1325.41 | 0.90% | 1112.2000 | 1156.7590 | 26.6795 | M30 bullish FVG [1110.0000, 1112.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 119 | 2025-07-21 07:00 | 1142.9000 | 1116.2661 | 2025-07-21 22:00 | 1116.2661 | SL | 30 | -3637.01 | -2.43% | 1148.0000 | 1196.1677 | 26.6339 | M30 bullish FVG [1147.2000, 1148.0000], stochastic crossed below 20 | FVG stop 1116.2661 |
| 120 | 2025-07-22 10:00 | 1106.5000 | 1078.9043 | 2025-07-22 15:30 | 1117.5000 | STOCH_EXIT | 11 | 1305.22 | 0.89% | 1125.1000 | 1161.6915 | 27.5957 | M30 bullish FVG [1124.9000, 1125.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 121 | 2025-07-23 18:00 | 1119.0000 | 1090.5040 | 2025-07-24 10:00 | 1133.1000 | STOCH_EXIT | 19 | 1712.56 | 1.16% | 1144.3000 | 1175.9921 | 28.4960 | M30 bullish FVG [1143.8000, 1144.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 122 | 2025-07-24 15:00 | 1125.2000 | 1096.4751 | 2025-07-24 20:00 | 1131.9000 | STOCH_EXIT | 10 | 735.43 | 0.49% | 1132.8000 | 1182.6498 | 28.7249 | M30 bullish FVG [1132.3000, 1132.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 123 | 2025-07-25 18:00 | 1119.0000 | 1091.1412 | 2025-07-26 02:00 | 1138.3000 | STOCH_EXIT | 12 | 2434.96 | 1.62% | 1138.2000 | 1174.7176 | 27.8588 | M30 bullish FVG [1132.9000, 1138.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 124 | 2025-07-28 18:30 | 1146.0000 | 1118.0525 | 2025-07-29 09:30 | 1156.5000 | STOCH_EXIT | 16 | 1243.38 | 0.82% | 1161.0000 | 1201.8950 | 27.9475 | M30 bullish FVG [1160.0000, 1161.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 125 | 2025-08-05 13:00 | 1162.8000 | 1129.9909 | 2025-08-05 20:30 | 1172.5000 | STOCH_EXIT | 15 | 1126.27 | 0.73% | 1178.0000 | 1228.4183 | 32.8091 | M30 bullish FVG [1176.0000, 1178.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 126 | 2025-08-11 07:00 | 1204.4000 | 1171.7137 | 2025-08-11 09:30 | 1217.5000 | STOCH_EXIT | 5 | 1521.80 | 0.99% | 1206.0000 | 1269.7727 | 32.6863 | M30 bullish FVG [1205.9000, 1206.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 127 | 2025-08-11 21:30 | 1224.0000 | 1191.3137 | 2025-08-12 10:30 | 1242.0000 | STOCH_EXIT | 13 | 2129.41 | 1.37% | 1230.9000 | 1289.3727 | 32.6863 | M30 bullish FVG [1229.8000, 1230.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 128 | 2025-08-14 10:30 | 1251.6000 | 1216.6440 | 2025-08-15 10:00 | 1257.3000 | STOCH_EXIT | 34 | 560.14 | 0.36% | 1267.1000 | 1321.5120 | 34.9560 | M30 bullish FVG [1264.8000, 1267.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 129 | 2025-08-15 21:30 | 1257.4000 | 1221.8480 | 2025-08-16 02:00 | 1221.8480 | SL | 5 | -4635.75 | -2.92% | 1264.0000 | 1328.5040 | 35.5520 | M30 bullish FVG [1261.4000, 1264.0000], stochastic crossed below 20 | FVG stop 1221.8480 |
| 130 | 2025-08-20 14:00 | 1280.3000 | 1246.5232 | 2025-08-21 17:00 | 1246.5232 | SL | 41 | -4204.83 | -2.74% | 1299.0000 | 1347.8536 | 33.7768 | M30 bullish FVG [1295.0000, 1299.0000], stochastic crossed below 20 | FVG stop 1246.5232 |
| 131 | 2025-08-21 21:30 | 1236.5000 | 1200.8144 | 2025-08-22 09:00 | 1263.6000 | STOCH_EXIT | 10 | 3127.84 | 2.09% | 1235.1000 | 1307.8712 | 35.6856 | M30 bullish FVG [1232.1000, 1235.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 132 | 2025-08-26 18:00 | 1258.0000 | 1221.6375 | 2025-08-27 11:30 | 1272.4000 | STOCH_EXIT | 22 | 1602.45 | 1.04% | 1277.9000 | 1330.7251 | 36.3625 | M30 bullish FVG [1275.0000, 1277.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 133 | 2025-08-28 21:30 | 1247.9000 | 1213.1308 | 2025-08-30 02:30 | 1253.9000 | STOCH_EXIT | 41 | 588.89 | 0.38% | 1282.3000 | 1317.4384 | 34.7692 | M30 bullish FVG [1280.5000, 1282.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 134 | 2025-09-03 17:00 | 1238.3000 | 1204.2548 | 2025-09-03 22:30 | 1246.7000 | STOCH_EXIT | 11 | 894.69 | 0.58% | 1245.8000 | 1306.3903 | 34.0452 | M30 bullish FVG [1245.0000, 1245.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 135 | 2025-09-04 15:30 | 1243.9000 | 1211.0295 | 2025-09-05 07:00 | 1248.1000 | STOCH_EXIT | 17 | 369.25 | 0.24% | 1249.8000 | 1309.6410 | 32.8705 | M30 bullish FVG [1249.6000, 1249.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 136 | 2025-09-05 15:00 | 1244.8000 | 1213.3631 | 2025-09-06 15:00 | 1246.6000 | STOCH_EXIT | 42 | 69.84 | 0.04% | 1251.1000 | 1307.6738 | 31.4369 | M30 bullish FVG [1246.5000, 1251.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 137 | 2025-09-09 13:00 | 1268.1000 | 1237.7447 | 2025-09-10 11:00 | 1275.7000 | STOCH_EXIT | 31 | 778.36 | 0.50% | 1270.0000 | 1328.8106 | 30.3553 | M30 bullish FVG [1269.4000, 1270.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 138 | 2025-09-10 22:00 | 1272.8000 | 1243.0272 | 2025-09-12 07:00 | 1265.6000 | STOCH_EXIT | 39 | -1050.18 | -0.67% | 1279.0000 | 1332.3455 | 29.7728 | M30 bullish FVG [1277.8000, 1279.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 139 | 2025-09-12 10:30 | 1243.3000 | 1213.2000 | 2025-09-13 05:00 | 1251.6000 | STOCH_EXIT | 32 | 888.62 | 0.57% | 1259.2000 | 1303.5000 | 30.1000 | M30 bullish FVG [1258.8000, 1259.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 140 | 2025-09-15 09:30 | 1245.0000 | 1214.9786 | 2025-09-16 10:00 | 1250.2000 | STOCH_EXIT | 36 | 498.00 | 0.32% | 1257.1000 | 1305.0428 | 30.0214 | M30 bullish FVG [1257.0000, 1257.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 141 | 2025-09-16 13:00 | 1234.8000 | 1204.4087 | 2025-09-17 07:30 | 1241.1000 | STOCH_EXIT | 24 | 647.94 | 0.41% | 1245.2000 | 1295.5826 | 30.3913 | M30 bullish FVG [1238.7000, 1245.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 142 | 2025-09-17 13:00 | 1232.2000 | 1202.1509 | 2025-09-18 12:00 | 1219.7000 | STOCH_EXIT | 32 | -1756.92 | -1.11% | 1236.8000 | 1292.2981 | 30.0491 | M30 bullish FVG [1236.7000, 1236.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 143 | 2025-09-19 20:00 | 1219.9000 | 1188.9505 | 2025-09-20 13:00 | 1224.1000 | STOCH_EXIT | 29 | 381.18 | 0.24% | 1226.8000 | 1281.7989 | 30.9495 | M30 bullish FVG [1226.0000, 1226.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 144 | 2025-09-25 14:00 | 1236.0000 | 1204.5717 | 2025-09-27 19:30 | 1239.0000 | STOCH_EXIT | 87 | 223.84 | 0.14% | 1234.8000 | 1298.8565 | 31.4283 | M30 bullish FVG [1231.6000, 1234.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 145 | 2025-09-29 17:00 | 1227.1000 | 1197.1965 | 2025-09-30 10:30 | 1225.0000 | STOCH_EXIT | 22 | -425.73 | -0.27% | 1241.1000 | 1286.9069 | 29.9035 | M30 bullish FVG [1234.6000, 1241.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 146 | 2025-10-02 09:30 | 1240.2000 | 1208.8687 | 2025-10-03 11:00 | 1247.8000 | STOCH_EXIT | 38 | 800.86 | 0.51% | 1255.0000 | 1302.8627 | 31.3313 | M30 bullish FVG [1252.7000, 1255.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 147 | 2025-10-03 17:00 | 1238.3000 | 1207.0709 | 2025-10-03 22:30 | 1246.7000 | STOCH_EXIT | 11 | 909.00 | 0.58% | 1245.6000 | 1300.7582 | 31.2291 | M30 bullish FVG [1243.0000, 1245.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 148 | 2025-10-06 19:00 | 1244.4000 | 1213.4158 | 2025-10-06 23:00 | 1257.7000 | STOCH_EXIT | 8 | 1530.22 | 0.97% | 1257.0000 | 1306.3683 | 30.9842 | M30 bullish FVG [1252.0000, 1257.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 149 | 2025-10-07 13:30 | 1247.1000 | 1216.1433 | 2025-10-07 18:30 | 1258.5000 | STOCH_EXIT | 10 | 1298.84 | 0.81% | 1255.7000 | 1309.0134 | 30.9567 | M30 bullish FVG [1253.9000, 1255.7000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 150 | 2025-10-08 11:00 | 1245.6000 | 1215.6330 | 2025-10-09 10:00 | 1245.6000 | STOCH_EXIT | 33 | -160.68 | -0.10% | 1253.4000 | 1305.5339 | 29.9670 | M30 bullish FVG [1251.4000, 1253.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 151 | 2025-10-09 11:30 | 1226.4000 | 1194.8735 | 2025-10-09 15:30 | 1251.3000 | STOCH_EXIT | 8 | 3099.61 | 1.93% | 1242.6000 | 1289.4529 | 31.5265 | M30 bullish FVG [1240.6000, 1242.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 152 | 2025-10-10 14:00 | 1253.2000 | 1220.2254 | 2025-10-11 14:30 | 1246.1000 | STOCH_EXIT | 45 | -1093.80 | -0.67% | 1261.8000 | 1319.1491 | 32.9746 | M30 bullish FVG [1261.4000, 1261.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 153 | 2025-10-14 12:00 | 1269.0000 | 1233.5500 | 2025-10-14 18:00 | 1233.5500 | SL | 12 | -4697.76 | -2.89% | 1285.5000 | 1339.9000 | 35.4500 | M30 bullish FVG [1280.0000, 1285.5000], stochastic crossed below 20 | FVG stop 1233.5500 |
| 154 | 2025-10-20 22:00 | 1241.4000 | 1201.3258 | 2025-10-22 07:30 | 1241.3000 | STOCH_EXIT | 41 | -171.69 | -0.11% | 1243.7000 | 1321.5484 | 40.0742 | M30 bullish FVG [1243.4000, 1243.7000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 155 | 2025-10-22 22:30 | 1232.2000 | 1193.4207 | 2025-10-23 22:00 | 1218.4000 | STOCH_EXIT | 34 | -1923.24 | -1.22% | 1242.8000 | 1309.7586 | 38.7793 | M30 bullish FVG [1241.9000, 1242.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 156 | 2025-11-03 07:00 | 1260.5000 | 1224.8333 | 2025-11-03 07:00 | 1275.5000 | STOCH_EXIT | 0 | 1702.77 | 1.09% | 1267.0000 | 1331.8334 | 35.6667 | M30 bullish FVG [1266.2000, 1267.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 157 | 2025-11-05 17:30 | 1276.6000 | 1240.0952 | 2025-11-06 06:30 | 1304.9000 | STOCH_EXIT | 13 | 3349.15 | 2.11% | 1297.1000 | 1349.6096 | 36.5048 | M30 bullish FVG [1284.9000, 1297.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 158 | 2025-11-06 14:00 | 1289.5000 | 1252.6813 | 2025-11-07 07:00 | 1294.1000 | STOCH_EXIT | 21 | 413.52 | 0.26% | 1297.4000 | 1363.1374 | 36.8187 | M30 bullish FVG [1292.9000, 1297.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 159 | 2025-11-10 18:30 | 1306.7000 | 1271.2884 | 2025-11-11 08:30 | 1318.8000 | STOCH_EXIT | 15 | 1337.62 | 0.83% | 1311.1000 | 1377.5231 | 35.4116 | M30 bullish FVG [1310.7000, 1311.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 160 | 2025-11-12 15:30 | 1310.6000 | 1276.8880 | 2025-11-13 08:30 | 1313.3000 | STOCH_EXIT | 21 | 172.12 | 0.11% | 1325.4000 | 1378.0240 | 33.7120 | M30 bullish FVG [1323.6000, 1325.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 161 | 2025-11-13 14:30 | 1296.5000 | 1261.8246 | 2025-11-14 07:30 | 1298.1000 | STOCH_EXIT | 21 | 38.14 | 0.02% | 1311.8000 | 1365.8509 | 34.6754 | M30 bullish FVG [1306.1000, 1311.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 96510.42
- Макс: 165437.21
- Финал: 162943.92

Полная кривая — в `*_equity.csv`.
