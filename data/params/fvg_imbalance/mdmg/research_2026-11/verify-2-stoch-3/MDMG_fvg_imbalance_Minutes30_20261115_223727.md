# Бэктест MDMG (Minutes30)

- Период: 2023-11-15 22:37 — 2025-11-15 22:37
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
| StopATR | 2 |
| TakeProfitATR | 2 |
| HTFTrendEMA | 50 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 152 |
| Выигрышных / проигрышных | 110 / 42 |
| Win rate | 72.37% |
| Gross profit / loss | 151236.22 / 81361.75 |
| Profit factor | 1.859 |
| Чистый PnL | 68805.56 (68.81%) |
| Стартовый / финальный капитал | 100000.00 / 168805.56 |
| Макс. просадка | 15760.46 (13.65%) |
| Средняя прибыль / убыток | 1374.87 / 1937.18 |
| Expectancy | 459.70 |
| Лучшая / худшая сделка | 9408.96 / -11145.32 |
| Exposure | 18.52% |
| CAGR | 29.88% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-12-21 12:00 | 788.5000 | 725.6161 | 2023-12-22 10:00 | 796.6000 | STOCH_EXIT | 15 | 920.74 | 0.93% | 797.6000 | 851.3839 | 31.4419 | M30 bullish FVG [794.0000, 797.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 2 | 2023-12-26 11:30 | 811.0000 | 750.1029 | 2023-12-27 14:00 | 807.9000 | STOCH_EXIT | 24 | -484.77 | -0.48% | 820.4000 | 871.8971 | 30.4486 | M30 bullish FVG [809.0000, 820.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 3 | 2023-12-28 11:30 | 793.8000 | 733.0469 | 2024-01-03 09:30 | 792.6000 | STOCH_EXIT | 34 | -251.14 | -0.25% | 803.2000 | 854.5531 | 30.3766 | M30 bullish FVG [801.0000, 803.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 4 | 2024-01-05 11:30 | 806.5000 | 747.3962 | 2024-01-08 10:00 | 822.9000 | STOCH_EXIT | 16 | 1932.58 | 1.93% | 803.5000 | 865.6038 | 29.5519 | M30 bullish FVG [802.8000, 803.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 5 | 2024-01-08 18:30 | 808.1000 | 750.7893 | 2024-01-10 11:00 | 837.1000 | STOCH_EXIT | 23 | 3550.35 | 3.49% | 818.1000 | 865.4107 | 28.6553 | M30 bullish FVG [815.1000, 818.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 6 | 2024-01-16 10:30 | 817.0000 | 761.5216 | 2024-01-17 16:00 | 817.0000 | STOCH_EXIT | 30 | -105.39 | -0.10% | 825.1000 | 872.4784 | 27.7392 | M30 bullish FVG [825.0000, 825.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 7 | 2024-01-22 13:00 | 800.0000 | 749.9711 | 2024-01-22 16:00 | 808.0000 | STOCH_EXIT | 6 | 942.68 | 0.90% | 804.9000 | 850.0289 | 25.0144 | M30 bullish FVG [800.0000, 804.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 8 | 2024-01-23 17:00 | 807.7000 | 758.5303 | 2024-01-25 10:00 | 818.9000 | STOCH_EXIT | 24 | 1360.66 | 1.29% | 814.0000 | 856.8697 | 24.5848 | M30 bullish FVG [809.0000, 814.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 9 | 2024-02-05 12:00 | 883.3000 | 834.1702 | 2024-02-05 14:00 | 912.4000 | STOCH_EXIT | 4 | 3440.66 | 3.19% | 891.9000 | 932.4298 | 24.5649 | M30 bullish FVG [891.8000, 891.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 10 | 2024-02-07 11:30 | 979.0000 | 917.4636 | 2024-02-09 12:00 | 969.4000 | STOCH_EXIT | 39 | -1194.88 | -1.08% | 992.2000 | 1040.5364 | 30.7682 | M30 bullish FVG [990.2000, 992.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 11 | 2024-02-12 12:30 | 954.3000 | 886.1151 | 2024-02-14 09:30 | 967.0000 | STOCH_EXIT | 29 | 1350.03 | 1.23% | 978.5000 | 1022.4849 | 34.0924 | M30 bullish FVG [975.0000, 978.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 12 | 2024-02-16 14:30 | 993.8000 | 925.9656 | 2024-02-19 14:00 | 1015.4000 | STOCH_EXIT | 18 | 2306.68 | 2.07% | 1011.3000 | 1061.6344 | 33.9172 | M30 bullish FVG [1009.8000, 1011.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 13 | 2024-02-20 10:30 | 984.2000 | 914.2673 | 2024-02-21 10:00 | 914.2673 | SL | 18 | -8151.42 | -7.20% | 1012.9000 | 1054.1327 | 34.9663 | M30 bullish FVG [1009.4000, 1012.9000], stochastic crossed below 20 | FVG stop 914.2673 |
| 14 | 2024-02-28 17:30 | 958.1000 | 887.7372 | 2024-03-01 15:00 | 983.4000 | STOCH_EXIT | 33 | 2676.22 | 2.54% | 991.8000 | 1028.4628 | 35.1814 | M30 bullish FVG [978.9000, 991.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 15 | 2024-03-07 13:00 | 1015.7000 | 942.1195 | 2024-03-07 14:00 | 1036.3000 | STOCH_EXIT | 2 | 2074.84 | 1.93% | 1028.3000 | 1089.2805 | 36.7902 | M30 bullish FVG [1026.0000, 1028.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 16 | 2024-03-11 17:30 | 1025.6000 | 952.7467 | 2024-03-18 10:30 | 995.4000 | STOCH_EXIT | 81 | -3339.52 | -3.04% | 1030.9000 | 1098.4533 | 36.4266 | M30 bullish FVG [1016.6000, 1030.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 17 | 2024-03-20 12:00 | 1001.6000 | 933.4283 | 2024-03-21 11:30 | 1008.9000 | STOCH_EXIT | 18 | 667.24 | 0.63% | 1013.0000 | 1069.7717 | 34.0858 | M30 bullish FVG [1008.7000, 1013.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 18 | 2024-03-22 16:30 | 1001.7000 | 935.7030 | 2024-03-29 12:30 | 988.2000 | STOCH_EXIT | 87 | -1550.96 | -1.45% | 1013.4000 | 1067.6970 | 32.9985 | M30 bullish FVG [1012.9000, 1013.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 19 | 2024-04-04 13:30 | 1041.0000 | 976.9023 | 2024-04-06 10:00 | 1026.0000 | STOCH_EXIT | 30 | -1619.38 | -1.54% | 1070.0000 | 1105.0977 | 32.0489 | M30 bullish FVG [1060.0000, 1070.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 20 | 2024-04-08 10:00 | 1018.5000 | 950.3208 | 2024-04-08 15:00 | 1032.3000 | STOCH_EXIT | 10 | 1303.01 | 1.25% | 1025.1000 | 1086.6792 | 34.0896 | M30 bullish FVG [1025.0000, 1025.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 21 | 2024-04-09 12:30 | 1012.9000 | 946.7622 | 2024-04-11 12:30 | 1006.5000 | STOCH_EXIT | 38 | -770.61 | -0.73% | 1026.0000 | 1079.0378 | 33.0689 | M30 bullish FVG [1025.0000, 1026.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 22 | 2024-04-17 14:30 | 1036.3000 | 975.5958 | 2024-04-17 18:30 | 1047.8000 | STOCH_EXIT | 8 | 1056.25 | 1.01% | 1047.3000 | 1097.0042 | 30.3521 | M30 bullish FVG [1046.5000, 1047.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 23 | 2024-04-19 12:00 | 1038.7000 | 982.2152 | 2024-04-20 12:00 | 1049.3000 | STOCH_EXIT | 18 | 974.71 | 0.92% | 1045.2000 | 1095.1848 | 28.2424 | M30 bullish FVG [1044.1000, 1045.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 24 | 2024-04-23 12:30 | 1046.4000 | 988.7555 | 2024-04-24 13:30 | 1050.6000 | STOCH_EXIT | 21 | 321.45 | 0.30% | 1057.0000 | 1104.0445 | 28.8223 | M30 bullish FVG [1053.8000, 1057.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 25 | 2024-04-30 10:30 | 1104.1000 | 1044.2259 | 2024-05-01 22:30 | 1111.0000 | STOCH_EXIT | 42 | 561.87 | 0.52% | 1115.5000 | 1163.9741 | 29.9371 | M30 bullish FVG [1114.5000, 1115.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 26 | 2024-05-03 13:30 | 1136.7000 | 1072.0341 | 2024-05-04 15:00 | 1154.8000 | STOCH_EXIT | 21 | 1593.70 | 1.49% | 1133.9000 | 1201.3659 | 32.3330 | M30 bullish FVG [1115.0000, 1133.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 27 | 2024-05-07 11:00 | 1122.4000 | 1056.7054 | 2024-05-08 15:30 | 1138.5000 | STOCH_EXIT | 28 | 1452.05 | 1.33% | 1133.0000 | 1188.0946 | 32.8473 | M30 bullish FVG [1130.3000, 1133.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 28 | 2024-05-15 13:00 | 1179.1000 | 1110.6090 | 2024-05-15 18:00 | 1110.6090 | SL | 10 | -6545.77 | -5.90% | 1194.9000 | 1247.5910 | 34.2455 | M30 bullish FVG [1188.5000, 1194.9000], stochastic crossed below 20 | FVG stop 1110.6090 |
| 29 | 2024-05-16 16:00 | 1102.7000 | 1024.8726 | 2024-05-17 18:00 | 1088.1000 | STOCH_EXIT | 23 | -1475.37 | -1.42% | 1118.4000 | 1180.5274 | 38.9137 | M30 bullish FVG [1115.0000, 1118.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 30 | 2024-05-20 14:30 | 1061.3000 | 977.1815 | 2024-05-21 10:00 | 1145.4185 | TP | 10 | 7969.45 | 7.82% | 1064.9000 | 1145.4185 | 42.0592 | M30 bullish FVG [1064.8000, 1064.9000], stochastic crossed below 20 | FVG target 1145.4185 |
| 31 | 2024-05-22 16:00 | 1070.1000 | 977.6346 | 2024-06-17 09:30 | 1162.5654 | TP | 6 | 9408.96 | 8.53% | 1088.3000 | 1162.5654 | 46.2327 | M30 bullish FVG [1087.9000, 1088.3000], stochastic crossed below 20 | FVG target 1162.5654 |
| 32 | 2024-06-17 16:30 | 1101.2000 | 1009.6964 | 2024-06-18 09:30 | 1000.0000 | SL | 5 | -11145.32 | -9.28% | 1271.7000 | 1192.7036 | 45.7518 | M30 bullish FVG [1179.2000, 1271.7000], stochastic crossed below 20 | FVG stop 1009.6964 |
| 33 | 2024-09-23 10:00 | 873.0000 | 826.4835 | 2024-09-23 17:00 | 887.0000 | STOCH_EXIT | 14 | 1640.00 | 1.50% | 873.6000 | 919.5165 | 23.2583 | M30 bullish FVG [873.5000, 873.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 34 | 2024-09-24 10:30 | 861.7000 | 814.6203 | 2024-09-25 09:30 | 873.8000 | STOCH_EXIT | 32 | 1437.73 | 1.30% | 883.2000 | 908.7797 | 23.5398 | M30 bullish FVG [882.9000, 883.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 35 | 2024-09-25 23:30 | 857.6000 | 808.9403 | 2024-09-26 07:00 | 870.3000 | STOCH_EXIT | 1 | 1538.69 | 1.38% | 860.5000 | 906.2597 | 24.3298 | M30 bullish FVG [859.0000, 860.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 36 | 2024-09-26 22:30 | 875.6000 | 827.2017 | 2024-09-27 11:00 | 886.7000 | STOCH_EXIT | 11 | 1318.23 | 1.17% | 879.6000 | 923.9983 | 24.1991 | M30 bullish FVG [877.7000, 879.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 37 | 2024-09-27 21:00 | 884.1000 | 836.6302 | 2024-09-28 13:00 | 891.0000 | STOCH_EXIT | 18 | 781.62 | 0.68% | 890.4000 | 931.5698 | 23.7349 | M30 bullish FVG [888.9000, 890.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 38 | 2024-10-02 18:30 | 886.8000 | 836.7491 | 2024-10-04 07:30 | 881.9000 | STOCH_EXIT | 46 | -751.97 | -0.65% | 904.1000 | 936.8509 | 25.0255 | M30 bullish FVG [903.8000, 904.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 39 | 2024-10-04 13:00 | 870.9000 | 818.7224 | 2024-10-05 07:00 | 876.4000 | STOCH_EXIT | 22 | 610.68 | 0.53% | 882.0000 | 923.0776 | 26.0888 | M30 bullish FVG [880.0000, 882.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 40 | 2024-10-07 11:30 | 875.2000 | 823.8208 | 2024-10-08 13:00 | 869.6000 | STOCH_EXIT | 37 | -854.36 | -0.74% | 881.3000 | 926.5792 | 25.6896 | M30 bullish FVG [877.0000, 881.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 41 | 2024-10-10 11:00 | 868.8000 | 820.3838 | 2024-10-11 16:30 | 875.8000 | STOCH_EXIT | 45 | 808.86 | 0.70% | 870.9000 | 917.2162 | 24.2081 | M30 bullish FVG [870.7000, 870.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 42 | 2024-10-11 22:00 | 870.7000 | 823.6992 | 2024-10-12 23:00 | 868.8000 | STOCH_EXIT | 36 | -365.61 | -0.32% | 872.0000 | 917.7008 | 23.5004 | M30 bullish FVG [871.0000, 872.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 43 | 2024-10-14 08:00 | 869.8000 | 824.8136 | 2024-10-14 09:30 | 873.0000 | STOCH_EXIT | 3 | 307.38 | 0.27% | 870.0000 | 914.7864 | 22.4932 | M30 bullish FVG [869.8000, 870.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 44 | 2024-10-16 10:00 | 882.0000 | 838.7413 | 2024-10-16 11:00 | 891.6000 | STOCH_EXIT | 2 | 1141.43 | 0.99% | 888.5000 | 925.2587 | 21.6294 | M30 bullish FVG [887.0000, 888.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 45 | 2024-10-16 19:30 | 883.2000 | 839.9413 | 2024-10-17 08:30 | 888.9000 | STOCH_EXIT | 12 | 635.44 | 0.54% | 886.3000 | 926.4587 | 21.6294 | M30 bullish FVG [885.4000, 886.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 46 | 2024-10-17 13:00 | 881.9000 | 839.6598 | 2024-10-19 07:00 | 868.3000 | STOCH_EXIT | 56 | -1925.19 | -1.64% | 888.0000 | 924.1402 | 21.1201 | M30 bullish FVG [887.7000, 888.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 47 | 2024-10-31 17:00 | 855.4000 | 791.2179 | 2024-11-01 10:30 | 860.1000 | STOCH_EXIT | 21 | 518.70 | 0.45% | 860.6000 | 919.5821 | 32.0910 | M30 bullish FVG [860.0000, 860.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 48 | 2024-11-13 20:00 | 877.1000 | 821.8723 | 2024-11-14 13:00 | 881.9000 | STOCH_EXIT | 20 | 517.51 | 0.45% | 882.9000 | 932.3277 | 27.6138 | M30 bullish FVG [879.5000, 882.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 49 | 2024-11-14 21:00 | 869.4000 | 816.7457 | 2024-11-15 12:30 | 882.8000 | STOCH_EXIT | 17 | 1678.20 | 1.44% | 880.5000 | 922.0543 | 26.3271 | M30 bullish FVG [879.7000, 880.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 50 | 2024-11-19 12:30 | 875.3000 | 820.6283 | 2024-11-20 10:00 | 884.3000 | STOCH_EXIT | 29 | 1096.23 | 0.93% | 894.6000 | 929.9717 | 27.3358 | M30 bullish FVG [893.8000, 894.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 51 | 2024-11-20 18:30 | 867.5000 | 811.5906 | 2024-11-21 10:00 | 877.1000 | STOCH_EXIT | 17 | 1195.69 | 1.01% | 885.3000 | 923.4094 | 27.9547 | M30 bullish FVG [885.0000, 885.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 52 | 2024-11-21 18:00 | 864.6000 | 809.3270 | 2024-11-22 10:00 | 873.3000 | STOCH_EXIT | 18 | 1088.52 | 0.91% | 873.0000 | 919.8730 | 27.6365 | M30 bullish FVG [869.5000, 873.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 53 | 2024-11-22 20:30 | 867.1000 | 813.0322 | 2024-11-23 14:00 | 874.8000 | STOCH_EXIT | 21 | 956.07 | 0.79% | 857.5000 | 921.1678 | 27.0339 | M30 bullish FVG [857.1000, 857.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 54 | 2024-11-25 18:00 | 872.6000 | 820.3799 | 2024-11-27 14:00 | 868.4000 | STOCH_EXIT | 60 | -709.87 | -0.58% | 885.2000 | 924.8201 | 26.1100 | M30 bullish FVG [883.9000, 885.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 55 | 2024-12-23 15:00 | 868.8000 | 832.2046 | 2024-12-26 09:30 | 859.9000 | STOCH_EXIT | 91 | -1367.01 | -1.12% | 866.2000 | 905.3954 | 18.2977 | M30 bullish FVG [865.3000, 866.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 56 | 2024-12-27 11:30 | 852.5000 | 814.9957 | 2024-12-27 23:00 | 856.7000 | STOCH_EXIT | 23 | 471.70 | 0.39% | 865.0000 | 890.0043 | 18.7522 | M30 bullish FVG [857.7000, 865.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 57 | 2024-12-31 13:00 | 894.7000 | 853.8549 | 2024-12-31 19:00 | 900.9000 | STOCH_EXIT | 12 | 715.80 | 0.59% | 898.0000 | 935.5451 | 20.4225 | M30 bullish FVG [896.5000, 898.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 58 | 2025-01-07 15:00 | 940.9000 | 891.0030 | 2025-01-08 07:30 | 943.8000 | STOCH_EXIT | 19 | 252.54 | 0.21% | 944.8000 | 990.7970 | 24.9485 | M30 bullish FVG [944.6000, 944.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 59 | 2025-01-10 12:00 | 912.9000 | 860.6765 | 2025-01-10 14:00 | 934.6000 | STOCH_EXIT | 4 | 2763.24 | 2.27% | 930.9000 | 965.1235 | 26.1117 | M30 bullish FVG [923.6000, 930.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 60 | 2025-01-15 14:00 | 936.2000 | 884.6276 | 2025-01-15 22:00 | 946.9000 | STOCH_EXIT | 16 | 1297.87 | 1.04% | 940.2000 | 987.7724 | 25.7862 | M30 bullish FVG [937.6000, 940.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 61 | 2025-01-20 23:00 | 928.9000 | 882.9342 | 2025-01-21 19:30 | 940.8000 | STOCH_EXIT | 27 | 1480.30 | 1.18% | 938.5000 | 974.8658 | 22.9829 | M30 bullish FVG [936.5000, 938.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 62 | 2025-01-24 17:30 | 969.9000 | 918.1252 | 2025-01-25 09:30 | 973.5000 | STOCH_EXIT | 28 | 344.31 | 0.27% | 979.3000 | 1021.6748 | 25.8874 | M30 bullish FVG [977.0000, 979.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 63 | 2025-01-27 23:00 | 991.4000 | 940.7520 | 2025-01-28 18:30 | 1010.0000 | STOCH_EXIT | 25 | 2252.71 | 1.77% | 1008.6000 | 1042.0480 | 25.3240 | M30 bullish FVG [1001.9000, 1008.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 64 | 2025-02-07 15:30 | 991.6000 | 933.4767 | 2025-02-09 02:30 | 999.8000 | STOCH_EXIT | 61 | 943.76 | 0.73% | 1003.2000 | 1049.7233 | 29.0617 | M30 bullish FVG [1001.0000, 1003.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 65 | 2025-02-10 23:00 | 1002.6000 | 941.2855 | 2025-02-12 07:00 | 1014.7000 | STOCH_EXIT | 36 | 1441.88 | 1.11% | 1011.8000 | 1063.9145 | 30.6573 | M30 bullish FVG [1011.0000, 1011.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 66 | 2025-02-13 14:00 | 1026.4000 | 969.8934 | 2025-02-15 03:00 | 1039.7000 | STOCH_EXIT | 56 | 1582.44 | 1.19% | 1022.1000 | 1082.9066 | 28.2533 | M30 bullish FVG [1022.0000, 1022.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 67 | 2025-02-17 10:00 | 1036.2000 | 975.5917 | 2025-02-17 12:00 | 1049.6000 | STOCH_EXIT | 4 | 1594.07 | 1.19% | 1036.6000 | 1096.8083 | 30.3041 | M30 bullish FVG [1036.5000, 1036.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 68 | 2025-02-18 14:30 | 1030.0000 | 970.9209 | 2025-02-19 19:00 | 1028.8000 | STOCH_EXIT | 43 | -292.05 | -0.22% | 1029.2000 | 1089.0791 | 29.5396 | M30 bullish FVG [1026.9000, 1029.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 69 | 2025-02-21 21:00 | 1035.9000 | 977.4275 | 2025-02-22 03:00 | 1045.5000 | STOCH_EXIT | 7 | 1112.71 | 0.83% | 1039.0000 | 1094.3725 | 29.2363 | M30 bullish FVG [1036.6000, 1039.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 70 | 2025-02-27 12:00 | 1001.2000 | 945.8601 | 2025-02-27 17:30 | 1027.8000 | STOCH_EXIT | 11 | 3479.63 | 2.55% | 1031.4000 | 1056.5399 | 27.6699 | M30 bullish FVG [1027.5000, 1031.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 71 | 2025-02-28 21:30 | 1008.6000 | 953.2416 | 2025-03-01 12:30 | 1020.5000 | STOCH_EXIT | 26 | 1502.19 | 1.08% | 1017.8000 | 1063.9584 | 27.6792 | M30 bullish FVG [1016.3000, 1017.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 72 | 2025-03-04 15:30 | 1036.2000 | 980.1777 | 2025-03-05 11:00 | 1052.0000 | STOCH_EXIT | 25 | 2006.80 | 1.42% | 1051.3000 | 1092.2223 | 28.0112 | M30 bullish FVG [1048.0000, 1051.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 73 | 2025-03-05 22:30 | 1047.3000 | 991.3221 | 2025-03-06 09:00 | 1056.0000 | STOCH_EXIT | 7 | 1040.18 | 0.73% | 1051.7000 | 1103.2779 | 27.9889 | M30 bullish FVG [1050.8000, 1051.7000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 74 | 2025-03-06 21:30 | 1060.9000 | 1006.7777 | 2025-03-07 23:30 | 1060.1000 | STOCH_EXIT | 38 | -253.03 | -0.18% | 1072.5000 | 1115.0223 | 27.0612 | M30 bullish FVG [1070.6000, 1072.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 75 | 2025-03-11 13:00 | 1059.9000 | 1003.8567 | 2025-03-12 07:30 | 1053.9000 | STOCH_EXIT | 23 | -959.74 | -0.67% | 1067.0000 | 1115.9433 | 28.0216 | M30 bullish FVG [1066.9000, 1067.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 76 | 2025-03-12 19:00 | 1027.6000 | 971.2741 | 2025-03-12 23:00 | 1040.6000 | STOCH_EXIT | 8 | 1663.26 | 1.16% | 1035.5000 | 1083.9259 | 28.1629 | M30 bullish FVG [1032.0000, 1035.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 77 | 2025-03-13 11:00 | 1017.0000 | 961.4117 | 2025-03-14 10:00 | 1033.8000 | STOCH_EXIT | 32 | 2239.99 | 1.55% | 1040.0000 | 1072.5883 | 27.7942 | M30 bullish FVG [1039.7000, 1040.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 78 | 2025-03-17 10:00 | 1041.0000 | 986.4039 | 2025-03-18 08:30 | 1049.2000 | STOCH_EXIT | 31 | 1008.84 | 0.69% | 1043.0000 | 1095.5961 | 27.2980 | M30 bullish FVG [1042.2000, 1043.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 79 | 2025-03-19 11:30 | 1020.2000 | 963.1554 | 2025-03-19 12:30 | 1031.5000 | STOCH_EXIT | 2 | 1489.75 | 1.01% | 1025.0000 | 1077.2446 | 28.5223 | M30 bullish FVG [1024.1000, 1025.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 80 | 2025-03-19 19:00 | 1023.7000 | 966.6554 | 2025-03-20 08:00 | 1028.8000 | STOCH_EXIT | 12 | 594.77 | 0.40% | 1026.1000 | 1080.7446 | 28.5223 | M30 bullish FVG [1026.0000, 1026.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 81 | 2025-03-24 12:30 | 1117.5000 | 1042.9121 | 2025-03-25 08:30 | 1102.7000 | STOCH_EXIT | 27 | -2131.95 | -1.42% | 1126.1000 | 1192.0879 | 37.2940 | M30 bullish FVG [1123.0000, 1126.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 82 | 2025-03-25 13:00 | 1083.1000 | 1002.6969 | 2025-03-25 23:00 | 1083.0000 | STOCH_EXIT | 20 | -160.89 | -0.11% | 1105.1000 | 1163.5031 | 40.2015 | M30 bullish FVG [1103.5000, 1105.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 83 | 2025-03-26 18:00 | 1070.7000 | 988.8400 | 2025-03-28 10:00 | 1059.0000 | STOCH_EXIT | 54 | -1761.55 | -1.19% | 1081.0000 | 1152.5600 | 40.9300 | M30 bullish FVG [1079.7000, 1081.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 84 | 2025-03-28 21:30 | 1050.9000 | 971.5921 | 2025-03-29 02:30 | 1066.9000 | STOCH_EXIT | 6 | 2076.81 | 1.42% | 1058.0000 | 1130.2079 | 39.6539 | M30 bullish FVG [1041.9000, 1058.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 85 | 2025-03-31 11:30 | 1037.3000 | 958.2427 | 2025-04-01 06:30 | 1027.7000 | STOCH_EXIT | 25 | -1509.82 | -1.02% | 1076.0000 | 1116.3573 | 39.5286 | M30 bullish FVG [1067.3000, 1076.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 86 | 2025-04-01 11:30 | 1013.8000 | 924.2611 | 2025-04-02 21:00 | 1005.5000 | STOCH_EXIT | 54 | -1340.59 | -0.92% | 1032.1000 | 1103.3389 | 44.7695 | M30 bullish FVG [1027.7000, 1032.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 87 | 2025-04-14 07:00 | 1013.8000 | 941.5197 | 2025-04-15 09:00 | 1003.3000 | STOCH_EXIT | 39 | -1645.72 | -1.13% | 1021.6000 | 1086.0803 | 36.1401 | M30 bullish FVG [1021.0000, 1021.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 88 | 2025-05-09 03:00 | 999.2000 | 956.7309 | 2025-05-09 07:30 | 1003.8000 | STOCH_EXIT | 9 | 514.59 | 0.36% | 999.2000 | 1041.6691 | 21.2345 | M30 bullish FVG [998.8000, 999.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 89 | 2025-05-09 16:00 | 998.1000 | 955.6309 | 2025-05-09 23:00 | 1002.0000 | STOCH_EXIT | 14 | 417.59 | 0.29% | 1000.0000 | 1040.5691 | 21.2345 | M30 bullish FVG [998.1000, 1000.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 90 | 2025-05-12 12:00 | 1008.2000 | 965.7309 | 2025-05-13 07:30 | 1003.7000 | STOCH_EXIT | 25 | -787.35 | -0.55% | 1006.7000 | 1050.6691 | 21.2345 | M30 bullish FVG [1006.0000, 1006.7000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 91 | 2025-05-13 12:00 | 998.8000 | 955.1216 | 2025-05-14 06:30 | 1000.0000 | STOCH_EXIT | 24 | 28.89 | 0.02% | 999.0000 | 1042.4784 | 21.8392 | M30 bullish FVG [998.9000, 999.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 92 | 2025-06-12 06:30 | 987.6000 | 952.8444 | 2025-06-12 21:00 | 992.4000 | STOCH_EXIT | 29 | 552.45 | 0.39% | 989.0000 | 1022.3556 | 17.3778 | M30 bullish FVG [988.8000, 989.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 93 | 2025-06-13 11:00 | 985.9000 | 951.1444 | 2025-06-14 08:30 | 981.1000 | STOCH_EXIT | 39 | -844.39 | -0.59% | 996.3000 | 1020.6556 | 17.3778 | M30 bullish FVG [994.0000, 996.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 94 | 2025-06-17 07:30 | 986.7000 | 952.3148 | 2025-06-17 13:30 | 990.5000 | STOCH_EXIT | 12 | 407.65 | 0.28% | 987.5000 | 1021.0852 | 17.1926 | M30 bullish FVG [987.2000, 987.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 95 | 2025-06-18 16:30 | 984.1000 | 951.0709 | 2025-06-18 19:00 | 992.3000 | STOCH_EXIT | 5 | 1052.92 | 0.73% | 993.0000 | 1017.1291 | 16.5146 | M30 bullish FVG [991.0000, 993.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 96 | 2025-06-19 14:00 | 990.7000 | 958.5301 | 2025-06-19 18:00 | 994.6000 | STOCH_EXIT | 8 | 424.47 | 0.29% | 991.0000 | 1022.8699 | 16.0849 | M30 bullish FVG [989.8000, 991.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 97 | 2025-06-20 19:30 | 993.4000 | 961.8137 | 2025-06-20 23:30 | 997.1000 | STOCH_EXIT | 8 | 394.89 | 0.27% | 996.3000 | 1024.9863 | 15.7932 | M30 bullish FVG [996.1000, 996.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 98 | 2025-06-24 10:30 | 991.6000 | 961.5250 | 2025-06-24 13:00 | 997.5000 | STOCH_EXIT | 5 | 721.10 | 0.49% | 995.0000 | 1021.6750 | 15.0375 | M30 bullish FVG [993.3000, 995.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 99 | 2025-06-25 07:00 | 997.5000 | 967.3875 | 2025-06-25 19:30 | 1000.8000 | STOCH_EXIT | 25 | 338.22 | 0.23% | 997.8000 | 1027.6125 | 15.0562 | M30 bullish FVG [997.5000, 997.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 100 | 2025-06-26 23:00 | 1004.0000 | 973.7956 | 2025-06-27 11:00 | 1013.6000 | STOCH_EXIT | 10 | 1254.32 | 0.86% | 1011.0000 | 1034.2044 | 15.1022 | M30 bullish FVG [1005.0000, 1011.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 101 | 2025-06-30 07:00 | 1017.0000 | 985.1870 | 2025-06-30 10:30 | 1024.2000 | STOCH_EXIT | 7 | 896.01 | 0.61% | 1014.2000 | 1048.8130 | 15.9065 | M30 bullish FVG [1013.4000, 1014.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 102 | 2025-07-01 10:30 | 1024.1000 | 991.4308 | 2025-07-01 15:00 | 1028.1000 | STOCH_EXIT | 9 | 431.22 | 0.29% | 1029.6000 | 1056.7692 | 16.3346 | M30 bullish FVG [1029.1000, 1029.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 103 | 2025-07-04 07:30 | 1031.6000 | 995.2058 | 2025-07-04 11:30 | 1063.1000 | STOCH_EXIT | 8 | 4385.18 | 2.95% | 1072.0000 | 1067.9942 | 18.1971 | M30 bullish FVG [1062.0000, 1072.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 104 | 2025-07-04 20:00 | 1046.6000 | 1010.2058 | 2025-07-05 08:30 | 1056.1000 | STOCH_EXIT | 20 | 1241.95 | 0.81% | 1050.4000 | 1082.9942 | 18.1971 | M30 bullish FVG [1047.6000, 1050.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 105 | 2025-07-07 07:30 | 1047.3000 | 1006.3483 | 2025-07-07 16:00 | 1053.5000 | STOCH_EXIT | 17 | 762.14 | 0.49% | 1053.9000 | 1088.2517 | 20.4759 | M30 bullish FVG [1053.5000, 1053.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 106 | 2025-07-08 08:30 | 1047.3000 | 1004.0877 | 2025-07-08 17:00 | 1060.6000 | STOCH_EXIT | 17 | 1812.42 | 1.17% | 1053.0000 | 1090.5123 | 21.6062 | M30 bullish FVG [1046.2000, 1053.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 107 | 2025-07-08 22:00 | 1049.8000 | 1006.5877 | 2025-07-09 23:30 | 1039.7000 | STOCH_EXIT | 38 | -1671.71 | -1.06% | 1058.9000 | 1093.0123 | 21.6062 | M30 bullish FVG [1056.1000, 1058.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 108 | 2025-07-11 09:00 | 1059.1000 | 1014.6815 | 2025-07-11 10:30 | 1069.9000 | STOCH_EXIT | 3 | 1431.12 | 0.92% | 1066.5000 | 1103.5185 | 22.2092 | M30 bullish FVG [1064.8000, 1066.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 109 | 2025-07-11 14:30 | 1058.0000 | 1013.5815 | 2025-07-12 10:30 | 1045.0000 | STOCH_EXIT | 35 | -2079.62 | -1.33% | 1066.4000 | 1102.4185 | 22.2092 | M30 bullish FVG [1062.0000, 1066.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 110 | 2025-07-16 11:00 | 1088.0000 | 1033.3364 | 2025-07-16 12:00 | 1099.3000 | STOCH_EXIT | 2 | 1449.30 | 0.94% | 1097.0000 | 1142.6636 | 27.3318 | M30 bullish FVG [1096.0000, 1097.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 111 | 2025-07-17 18:00 | 1103.4000 | 1050.0410 | 2025-07-18 10:30 | 1114.4000 | STOCH_EXIT | 19 | 1404.54 | 0.90% | 1112.2000 | 1156.7590 | 26.6795 | M30 bullish FVG [1110.0000, 1112.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 112 | 2025-07-21 07:00 | 1142.9000 | 1089.6323 | 2025-07-22 15:30 | 1117.5000 | STOCH_EXIT | 52 | -3661.17 | -2.32% | 1148.0000 | 1196.1677 | 26.6339 | M30 bullish FVG [1147.2000, 1148.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 113 | 2025-07-23 18:00 | 1119.0000 | 1062.0079 | 2025-07-24 10:00 | 1133.1000 | STOCH_EXIT | 19 | 1790.41 | 1.16% | 1144.3000 | 1175.9921 | 28.4960 | M30 bullish FVG [1143.8000, 1144.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 114 | 2025-07-24 15:00 | 1125.2000 | 1067.7502 | 2025-07-24 20:00 | 1131.9000 | STOCH_EXIT | 10 | 774.43 | 0.49% | 1132.8000 | 1182.6498 | 28.7249 | M30 bullish FVG [1132.3000, 1132.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 115 | 2025-07-25 18:00 | 1119.0000 | 1063.2824 | 2025-07-26 02:00 | 1138.3000 | STOCH_EXIT | 12 | 2543.99 | 1.62% | 1138.2000 | 1174.7176 | 27.8588 | M30 bullish FVG [1132.9000, 1138.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 116 | 2025-07-28 18:30 | 1146.0000 | 1090.1050 | 2025-07-29 09:30 | 1156.5000 | STOCH_EXIT | 16 | 1299.48 | 0.82% | 1161.0000 | 1201.8950 | 27.9475 | M30 bullish FVG [1160.0000, 1161.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 117 | 2025-08-05 13:00 | 1162.8000 | 1097.1817 | 2025-08-05 20:30 | 1172.5000 | STOCH_EXIT | 15 | 1177.46 | 0.73% | 1178.0000 | 1228.4183 | 32.8091 | M30 bullish FVG [1176.0000, 1178.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 118 | 2025-08-11 07:00 | 1204.4000 | 1139.0273 | 2025-08-11 09:30 | 1217.5000 | STOCH_EXIT | 5 | 1593.13 | 0.99% | 1206.0000 | 1269.7727 | 32.6863 | M30 bullish FVG [1205.9000, 1206.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 119 | 2025-08-11 21:30 | 1224.0000 | 1158.6273 | 2025-08-12 10:30 | 1242.0000 | STOCH_EXIT | 13 | 2230.01 | 1.37% | 1230.9000 | 1289.3727 | 32.6863 | M30 bullish FVG [1229.8000, 1230.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 120 | 2025-08-14 10:30 | 1251.6000 | 1181.6880 | 2025-08-15 10:00 | 1257.3000 | STOCH_EXIT | 34 | 586.81 | 0.36% | 1267.1000 | 1321.5120 | 34.9560 | M30 bullish FVG [1264.8000, 1267.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 121 | 2025-08-15 21:30 | 1257.4000 | 1186.2960 | 2025-08-16 03:30 | 1255.0000 | STOCH_EXIT | 8 | -482.62 | -0.29% | 1264.0000 | 1328.5040 | 35.5520 | M30 bullish FVG [1261.4000, 1264.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 122 | 2025-08-20 14:00 | 1280.3000 | 1212.7464 | 2025-08-22 09:00 | 1263.6000 | STOCH_EXIT | 60 | -2318.38 | -1.40% | 1299.0000 | 1347.8536 | 33.7768 | M30 bullish FVG [1295.0000, 1299.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 123 | 2025-08-26 18:00 | 1258.0000 | 1185.2749 | 2025-08-27 11:30 | 1272.4000 | STOCH_EXIT | 22 | 1707.52 | 1.04% | 1277.9000 | 1330.7251 | 36.3625 | M30 bullish FVG [1275.0000, 1277.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 124 | 2025-08-28 21:30 | 1247.9000 | 1178.3616 | 2025-08-30 02:30 | 1253.9000 | STOCH_EXIT | 41 | 626.88 | 0.38% | 1282.3000 | 1317.4384 | 34.7692 | M30 bullish FVG [1280.5000, 1282.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 125 | 2025-09-03 17:00 | 1238.3000 | 1170.2097 | 2025-09-03 22:30 | 1246.7000 | STOCH_EXIT | 11 | 959.11 | 0.58% | 1245.8000 | 1306.3903 | 34.0452 | M30 bullish FVG [1245.0000, 1245.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 126 | 2025-09-04 15:30 | 1243.9000 | 1178.1590 | 2025-09-05 07:00 | 1248.1000 | STOCH_EXIT | 17 | 395.84 | 0.24% | 1249.8000 | 1309.6410 | 32.8705 | M30 bullish FVG [1249.6000, 1249.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 127 | 2025-09-05 15:00 | 1244.8000 | 1181.9262 | 2025-09-06 15:00 | 1246.6000 | STOCH_EXIT | 42 | 74.28 | 0.04% | 1251.1000 | 1307.6738 | 31.4369 | M30 bullish FVG [1246.5000, 1251.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 128 | 2025-09-09 13:00 | 1268.1000 | 1207.3894 | 2025-09-10 11:00 | 1275.7000 | STOCH_EXIT | 31 | 835.31 | 0.50% | 1270.0000 | 1328.8106 | 30.3553 | M30 bullish FVG [1269.4000, 1270.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 129 | 2025-09-10 22:00 | 1272.8000 | 1213.2545 | 2025-09-12 07:00 | 1265.6000 | STOCH_EXIT | 39 | -1117.93 | -0.67% | 1279.0000 | 1332.3455 | 29.7728 | M30 bullish FVG [1277.8000, 1279.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 130 | 2025-09-12 10:30 | 1243.3000 | 1183.1000 | 2025-09-13 05:00 | 1251.6000 | STOCH_EXIT | 32 | 945.04 | 0.57% | 1259.2000 | 1303.5000 | 30.1000 | M30 bullish FVG [1258.8000, 1259.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 131 | 2025-09-15 09:30 | 1245.0000 | 1184.9572 | 2025-09-16 10:00 | 1250.2000 | STOCH_EXIT | 36 | 533.57 | 0.32% | 1257.1000 | 1305.0428 | 30.0214 | M30 bullish FVG [1257.0000, 1257.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 132 | 2025-09-16 13:00 | 1234.8000 | 1174.0174 | 2025-09-17 07:30 | 1241.1000 | STOCH_EXIT | 24 | 688.44 | 0.41% | 1245.2000 | 1295.5826 | 30.3913 | M30 bullish FVG [1238.7000, 1245.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 133 | 2025-09-17 13:00 | 1232.2000 | 1172.1019 | 2025-09-18 12:00 | 1219.7000 | STOCH_EXIT | 32 | -1880.46 | -1.11% | 1236.8000 | 1292.2981 | 30.0491 | M30 bullish FVG [1236.7000, 1236.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 134 | 2025-09-19 20:00 | 1219.9000 | 1158.0011 | 2025-09-20 13:00 | 1224.1000 | STOCH_EXIT | 29 | 407.99 | 0.24% | 1226.8000 | 1281.7989 | 30.9495 | M30 bullish FVG [1226.0000, 1226.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 135 | 2025-09-25 14:00 | 1236.0000 | 1173.1435 | 2025-09-27 19:30 | 1239.0000 | STOCH_EXIT | 87 | 237.94 | 0.14% | 1234.8000 | 1298.8565 | 31.4283 | M30 bullish FVG [1231.6000, 1234.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 136 | 2025-09-29 17:00 | 1227.1000 | 1167.2931 | 2025-09-30 10:30 | 1225.0000 | STOCH_EXIT | 22 | -455.67 | -0.27% | 1241.1000 | 1286.9069 | 29.9035 | M30 bullish FVG [1234.6000, 1241.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 137 | 2025-10-02 09:30 | 1240.2000 | 1177.5373 | 2025-10-03 11:00 | 1247.8000 | STOCH_EXIT | 38 | 858.06 | 0.51% | 1255.0000 | 1302.8627 | 31.3313 | M30 bullish FVG [1252.7000, 1255.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 138 | 2025-10-03 17:00 | 1238.3000 | 1175.8418 | 2025-10-03 22:30 | 1246.7000 | STOCH_EXIT | 11 | 973.42 | 0.58% | 1245.6000 | 1300.7582 | 31.2291 | M30 bullish FVG [1243.0000, 1245.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 139 | 2025-10-06 19:00 | 1244.4000 | 1182.4317 | 2025-10-06 23:00 | 1257.7000 | STOCH_EXIT | 8 | 1638.66 | 0.97% | 1257.0000 | 1306.3683 | 30.9842 | M30 bullish FVG [1252.0000, 1257.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 140 | 2025-10-07 13:30 | 1247.1000 | 1185.1866 | 2025-10-07 18:30 | 1258.5000 | STOCH_EXIT | 10 | 1390.17 | 0.81% | 1255.7000 | 1309.0134 | 30.9567 | M30 bullish FVG [1253.9000, 1255.7000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 141 | 2025-10-08 11:00 | 1245.6000 | 1185.6661 | 2025-10-09 10:00 | 1245.6000 | STOCH_EXIT | 33 | -171.89 | -0.10% | 1253.4000 | 1305.5339 | 29.9670 | M30 bullish FVG [1251.4000, 1253.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 142 | 2025-10-09 11:30 | 1226.4000 | 1163.3471 | 2025-10-09 15:30 | 1251.3000 | STOCH_EXIT | 8 | 3312.56 | 1.93% | 1242.6000 | 1289.4529 | 31.5265 | M30 bullish FVG [1240.6000, 1242.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 143 | 2025-10-10 14:00 | 1253.2000 | 1187.2509 | 2025-10-11 14:30 | 1246.1000 | STOCH_EXIT | 45 | -1168.95 | -0.67% | 1261.8000 | 1319.1491 | 32.9746 | M30 bullish FVG [1261.4000, 1261.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 144 | 2025-10-14 12:00 | 1269.0000 | 1198.1000 | 2025-10-14 20:30 | 1198.1000 | SL | 17 | -9882.30 | -5.68% | 1285.5000 | 1339.9000 | 35.4500 | M30 bullish FVG [1280.0000, 1285.5000], stochastic crossed below 20 | FVG stop 1198.1000 |
| 145 | 2025-10-20 22:00 | 1241.4000 | 1161.2516 | 2025-10-22 07:30 | 1241.3000 | STOCH_EXIT | 41 | -177.06 | -0.11% | 1243.7000 | 1321.5484 | 40.0742 | M30 bullish FVG [1243.4000, 1243.7000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 146 | 2025-10-22 22:30 | 1232.2000 | 1154.6414 | 2025-10-23 22:00 | 1218.4000 | STOCH_EXIT | 34 | -1998.36 | -1.22% | 1242.8000 | 1309.7586 | 38.7793 | M30 bullish FVG [1241.9000, 1242.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 147 | 2025-11-03 07:00 | 1260.5000 | 1189.1666 | 2025-11-03 07:00 | 1275.5000 | STOCH_EXIT | 0 | 1757.70 | 1.09% | 1267.0000 | 1331.8334 | 35.6667 | M30 bullish FVG [1266.2000, 1267.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 148 | 2025-11-05 17:30 | 1276.6000 | 1203.5904 | 2025-11-06 06:30 | 1304.9000 | STOCH_EXIT | 13 | 3457.18 | 2.11% | 1297.1000 | 1349.6096 | 36.5048 | M30 bullish FVG [1284.9000, 1297.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 149 | 2025-11-06 14:00 | 1289.5000 | 1215.8626 | 2025-11-07 07:00 | 1294.1000 | STOCH_EXIT | 21 | 430.07 | 0.26% | 1297.4000 | 1363.1374 | 36.8187 | M30 bullish FVG [1292.9000, 1297.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 150 | 2025-11-10 18:30 | 1306.7000 | 1235.8769 | 2025-11-11 08:30 | 1318.8000 | STOCH_EXIT | 15 | 1380.77 | 0.83% | 1311.1000 | 1377.5231 | 35.4116 | M30 bullish FVG [1310.7000, 1311.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 151 | 2025-11-12 15:30 | 1310.6000 | 1243.1760 | 2025-11-13 08:30 | 1313.3000 | STOCH_EXIT | 21 | 179.06 | 0.11% | 1325.4000 | 1378.0240 | 33.7120 | M30 bullish FVG [1323.6000, 1325.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 152 | 2025-11-13 14:30 | 1296.5000 | 1227.1491 | 2025-11-14 07:30 | 1298.1000 | STOCH_EXIT | 21 | 39.35 | 0.02% | 1311.8000 | 1365.8509 | 34.6754 | M30 bullish FVG [1306.1000, 1311.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 97777.76
- Макс: 176491.74
- Финал: 168805.56

Полная кривая — в `*_equity.csv`.
