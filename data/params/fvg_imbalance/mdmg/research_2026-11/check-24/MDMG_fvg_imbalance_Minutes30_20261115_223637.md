# Бэктест MDMG (Minutes30)

- Период: 2023-11-15 22:36 — 2025-11-15 22:36
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

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
| RequireEntryAboveFVGLower | 1 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 65 |
| Выигрышных / проигрышных | 47 / 18 |
| Win rate | 72.31% |
| Gross profit / loss | 50237.11 / 35120.38 |
| Profit factor | 1.430 |
| Чистый PnL | 15116.73 (15.12%) |
| Стартовый / финальный капитал | 100000.00 / 115116.73 |
| Макс. просадка | 10965.16 (10.35%) |
| Средняя прибыль / убыток | 1068.87 / 1951.13 |
| Expectancy | 232.57 |
| Лучшая / худшая сделка | 7637.39 / -3818.27 |
| Exposure | 6.97% |
| CAGR | 7.28% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-12-26 11:30 | 811.0000 | 780.5514 | 2023-12-27 14:00 | 807.9000 | STOCH_EXIT | 24 | -480.86 | -0.48% | 820.4000 | 871.8971 | 30.4486 | M30 bullish FVG [809.0000, 820.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 2 | 2024-01-05 11:30 | 806.5000 | 776.9481 | 2024-01-08 10:00 | 822.9000 | STOCH_EXIT | 16 | 1916.99 | 1.93% | 803.5000 | 865.6038 | 29.5519 | M30 bullish FVG [802.8000, 803.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 3 | 2024-01-22 13:00 | 800.0000 | 774.9856 | 2024-01-22 16:00 | 808.0000 | STOCH_EXIT | 6 | 906.70 | 0.90% | 804.9000 | 850.0289 | 25.0144 | M30 bullish FVG [800.0000, 804.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 4 | 2024-01-23 17:00 | 807.7000 | 783.1152 | 2024-01-25 10:00 | 818.9000 | STOCH_EXIT | 24 | 1308.72 | 1.29% | 805.9000 | 856.8697 | 24.5848 | M30 bullish FVG [804.9000, 805.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 5 | 2024-02-07 11:30 | 979.0000 | 948.2318 | 2024-02-08 10:30 | 948.2318 | SL | 17 | -3331.84 | -3.24% | 974.5000 | 1040.5364 | 30.7682 | M30 bullish FVG [946.3000, 974.5000], stochastic crossed below 20 | FVG stop 948.2318 |
| 6 | 2024-02-12 12:30 | 954.3000 | 920.2076 | 2024-02-14 09:30 | 967.0000 | STOCH_EXIT | 29 | 1232.63 | 1.23% | 962.3000 | 1022.4849 | 34.0924 | M30 bullish FVG [948.8000, 962.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 7 | 2024-02-16 14:30 | 993.8000 | 959.8828 | 2024-02-19 14:00 | 1015.4000 | STOCH_EXIT | 18 | 2100.73 | 2.07% | 990.1000 | 1061.6344 | 33.9172 | M30 bullish FVG [981.8000, 990.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 8 | 2024-03-07 13:00 | 1015.7000 | 978.9098 | 2024-03-07 14:00 | 1036.3000 | STOCH_EXIT | 2 | 1976.97 | 1.93% | 1026.8000 | 1089.2805 | 36.7902 | M30 bullish FVG [1007.3000, 1026.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 9 | 2024-03-11 17:30 | 1025.6000 | 989.1734 | 2024-03-14 10:00 | 989.1734 | SL | 42 | -3818.27 | -3.65% | 1030.9000 | 1098.4533 | 36.4266 | M30 bullish FVG [1016.6000, 1030.9000], stochastic crossed below 20 | FVG stop 989.1734 |
| 10 | 2024-03-22 16:30 | 1001.7000 | 968.7015 | 2024-03-29 12:30 | 988.2000 | STOCH_EXIT | 87 | -1463.99 | -1.45% | 999.0000 | 1067.6970 | 32.9985 | M30 bullish FVG [995.3000, 999.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 11 | 2024-04-04 13:30 | 1041.0000 | 1008.9511 | 2024-04-04 17:30 | 1008.9511 | SL | 8 | -3175.09 | -3.18% | 1032.5000 | 1105.0977 | 32.0489 | M30 bullish FVG [1029.8000, 1032.5000], stochastic crossed below 20 | FVG stop 1008.9511 |
| 12 | 2024-04-05 13:00 | 1012.9000 | 977.7189 | 2024-04-06 10:00 | 1026.0000 | STOCH_EXIT | 12 | 1147.65 | 1.19% | 1019.3000 | 1083.2622 | 35.1811 | M30 bullish FVG [1012.6000, 1019.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 13 | 2024-04-08 12:30 | 1020.0000 | 985.9104 | 2024-04-08 15:00 | 1032.3000 | STOCH_EXIT | 5 | 1082.29 | 1.10% | 1029.1000 | 1088.1792 | 34.0896 | M30 bullish FVG [1020.0000, 1029.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 14 | 2024-04-17 14:30 | 1036.3000 | 1005.9479 | 2024-04-17 18:30 | 1047.8000 | STOCH_EXIT | 8 | 993.51 | 1.01% | 1038.0000 | 1097.0042 | 30.3521 | M30 bullish FVG [1035.8000, 1038.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 15 | 2024-05-01 14:30 | 1100.8000 | 1070.9799 | 2024-05-01 22:30 | 1111.0000 | STOCH_EXIT | 16 | 827.56 | 0.83% | 1100.0000 | 1160.4402 | 29.8201 | M30 bullish FVG [1098.2000, 1100.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 16 | 2024-05-03 13:30 | 1136.7000 | 1104.3670 | 2024-05-04 15:00 | 1154.8000 | STOCH_EXIT | 21 | 1508.93 | 1.49% | 1133.9000 | 1201.3659 | 32.3330 | M30 bullish FVG [1115.0000, 1133.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 17 | 2024-05-15 13:00 | 1179.1000 | 1144.8545 | 2024-05-15 15:00 | 1144.8545 | SL | 4 | -3080.45 | -3.00% | 1177.0000 | 1247.5910 | 34.2455 | M30 bullish FVG [1174.9000, 1177.0000], stochastic crossed below 20 | FVG stop 1144.8545 |
| 18 | 2024-05-16 16:00 | 1102.7000 | 1063.7863 | 2024-05-16 17:00 | 1063.7863 | SL | 2 | -3599.72 | -3.63% | 1100.1000 | 1180.5274 | 38.9137 | M30 bullish FVG [1099.0000, 1100.1000], stochastic crossed below 20 | FVG stop 1063.7863 |
| 19 | 2024-05-17 11:30 | 1064.1000 | 1022.1516 | 2024-05-17 18:00 | 1088.1000 | STOCH_EXIT | 13 | 2063.15 | 2.15% | 1074.0000 | 1147.9968 | 41.9484 | M30 bullish FVG [1056.7000, 1074.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 20 | 2024-05-20 14:30 | 1061.3000 | 1019.2408 | 2024-05-21 10:00 | 1145.4185 | TP | 10 | 7637.39 | 7.82% | 1062.0000 | 1145.4185 | 42.0592 | M30 bullish FVG [1058.4000, 1062.0000], stochastic crossed below 20 | FVG target 1145.4185 |
| 21 | 2024-09-27 21:00 | 884.1000 | 860.3651 | 2024-09-28 13:00 | 891.0000 | STOCH_EXIT | 18 | 715.48 | 0.68% | 882.4000 | 931.5698 | 23.7349 | M30 bullish FVG [880.0000, 882.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 22 | 2024-10-04 13:00 | 870.9000 | 844.8112 | 2024-10-05 07:00 | 876.4000 | STOCH_EXIT | 22 | 564.41 | 0.53% | 870.8000 | 923.0776 | 26.0888 | M30 bullish FVG [869.5000, 870.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 23 | 2024-11-21 18:00 | 864.6000 | 836.9635 | 2024-11-22 10:00 | 873.3000 | STOCH_EXIT | 18 | 963.22 | 0.91% | 863.1000 | 919.8730 | 27.6365 | M30 bullish FVG [862.9000, 863.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 24 | 2024-11-22 20:30 | 867.1000 | 840.0661 | 2024-11-23 14:00 | 874.8000 | STOCH_EXIT | 21 | 846.80 | 0.79% | 857.5000 | 921.1678 | 27.0339 | M30 bullish FVG [857.1000, 857.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 25 | 2024-12-23 15:00 | 868.8000 | 850.5023 | 2024-12-24 14:30 | 850.5023 | SL | 33 | -2394.67 | -2.20% | 866.2000 | 905.3954 | 18.2977 | M30 bullish FVG [865.3000, 866.2000], stochastic crossed below 20 | FVG stop 850.5023 |
| 26 | 2024-12-24 18:30 | 853.5000 | 835.2236 | 2024-12-26 09:30 | 859.9000 | STOCH_EXIT | 50 | 687.37 | 0.65% | 853.5000 | 890.0529 | 18.2764 | M30 bullish FVG [851.8000, 853.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 27 | 2025-01-15 14:00 | 936.2000 | 910.4138 | 2025-01-15 22:00 | 946.9000 | STOCH_EXIT | 16 | 1112.46 | 1.04% | 936.3000 | 987.7724 | 25.7862 | M30 bullish FVG [935.4000, 936.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 28 | 2025-01-24 17:30 | 969.9000 | 944.0126 | 2025-01-25 09:30 | 973.5000 | STOCH_EXIT | 28 | 291.74 | 0.27% | 965.0000 | 1021.6748 | 25.8874 | M30 bullish FVG [964.0000, 965.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 29 | 2025-01-27 23:00 | 991.4000 | 966.0760 | 2025-01-28 18:30 | 1010.0000 | STOCH_EXIT | 25 | 1918.32 | 1.77% | 992.0000 | 1042.0480 | 25.3240 | M30 bullish FVG [979.8000, 992.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 30 | 2025-02-13 14:00 | 1026.4000 | 998.1467 | 2025-02-15 03:00 | 1039.7000 | STOCH_EXIT | 56 | 1312.56 | 1.19% | 1022.1000 | 1082.9066 | 28.2533 | M30 bullish FVG [1022.0000, 1022.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 31 | 2025-02-18 14:30 | 1030.0000 | 1000.4604 | 2025-02-19 19:00 | 1028.8000 | STOCH_EXIT | 43 | -240.78 | -0.22% | 1029.2000 | 1089.0791 | 29.5396 | M30 bullish FVG [1026.9000, 1029.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 32 | 2025-03-06 21:30 | 1060.9000 | 1033.8388 | 2025-03-07 17:00 | 1033.8388 | SL | 25 | -2951.40 | -2.65% | 1063.1000 | 1115.0223 | 27.0612 | M30 bullish FVG [1058.0000, 1063.1000], stochastic crossed below 20 | FVG stop 1033.8388 |
| 33 | 2025-03-28 21:30 | 1050.9000 | 1011.2461 | 2025-03-29 02:30 | 1066.9000 | STOCH_EXIT | 6 | 1538.93 | 1.42% | 1058.0000 | 1130.2079 | 39.6539 | M30 bullish FVG [1041.9000, 1058.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 34 | 2025-05-09 16:00 | 998.1000 | 976.8655 | 2025-05-09 23:00 | 1002.0000 | STOCH_EXIT | 14 | 318.99 | 0.29% | 1000.0000 | 1040.5691 | 21.2345 | M30 bullish FVG [998.1000, 1000.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 35 | 2025-05-12 12:00 | 1008.2000 | 986.9655 | 2025-05-13 07:30 | 1003.7000 | STOCH_EXIT | 25 | -600.15 | -0.55% | 1006.7000 | 1050.6691 | 21.2345 | M30 bullish FVG [1006.0000, 1006.7000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 36 | 2025-06-17 07:30 | 986.7000 | 969.5074 | 2025-06-17 13:30 | 990.5000 | STOCH_EXIT | 12 | 312.07 | 0.28% | 986.9000 | 1021.0852 | 17.1926 | M30 bullish FVG [985.2000, 986.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 37 | 2025-06-19 14:00 | 990.7000 | 974.6151 | 2025-06-19 18:00 | 994.6000 | STOCH_EXIT | 8 | 322.72 | 0.29% | 991.0000 | 1022.8699 | 16.0849 | M30 bullish FVG [989.8000, 991.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 38 | 2025-06-20 19:30 | 993.4000 | 977.6068 | 2025-06-20 23:30 | 997.1000 | STOCH_EXIT | 8 | 300.23 | 0.27% | 994.1000 | 1024.9863 | 15.7932 | M30 bullish FVG [993.4000, 994.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 39 | 2025-06-24 10:30 | 991.6000 | 976.5625 | 2025-06-24 13:00 | 997.5000 | STOCH_EXIT | 5 | 544.50 | 0.49% | 988.5000 | 1021.6750 | 15.0375 | M30 bullish FVG [988.3000, 988.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 40 | 2025-06-25 07:00 | 997.5000 | 982.4438 | 2025-06-25 19:30 | 1000.8000 | STOCH_EXIT | 25 | 255.39 | 0.23% | 997.8000 | 1027.6125 | 15.0562 | M30 bullish FVG [997.5000, 997.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 41 | 2025-06-26 23:00 | 1004.0000 | 988.8978 | 2025-06-27 11:00 | 1013.6000 | STOCH_EXIT | 10 | 953.62 | 0.86% | 1002.4000 | 1034.2044 | 15.1022 | M30 bullish FVG [1002.0000, 1002.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 42 | 2025-07-08 08:30 | 1047.3000 | 1025.6938 | 2025-07-08 17:00 | 1060.6000 | STOCH_EXIT | 17 | 1310.33 | 1.17% | 1053.0000 | 1090.5123 | 21.6062 | M30 bullish FVG [1046.2000, 1053.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 43 | 2025-07-08 22:00 | 1049.8000 | 1028.1938 | 2025-07-09 23:30 | 1039.7000 | STOCH_EXIT | 38 | -1203.63 | -1.06% | 1053.5000 | 1093.0123 | 21.6062 | M30 bullish FVG [1048.6000, 1053.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 44 | 2025-07-11 09:00 | 1059.1000 | 1036.8908 | 2025-07-11 10:30 | 1069.9000 | STOCH_EXIT | 3 | 1031.96 | 0.92% | 1056.9000 | 1103.5185 | 22.2092 | M30 bullish FVG [1056.5000, 1056.9000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 45 | 2025-07-11 21:00 | 1041.0000 | 1018.7908 | 2025-07-12 05:00 | 1018.7908 | SL | 11 | -2533.07 | -2.23% | 1040.2000 | 1085.4185 | 22.2092 | M30 bullish FVG [1039.8000, 1040.2000], stochastic crossed below 20 | FVG stop 1018.7908 |
| 46 | 2025-07-17 18:00 | 1103.4000 | 1076.7205 | 2025-07-18 10:30 | 1114.4000 | STOCH_EXIT | 19 | 989.11 | 0.90% | 1105.5000 | 1156.7590 | 26.6795 | M30 bullish FVG [1098.0000, 1105.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 47 | 2025-07-28 18:30 | 1146.0000 | 1118.0525 | 2025-07-29 09:30 | 1156.5000 | STOCH_EXIT | 16 | 906.83 | 0.82% | 1146.5000 | 1201.8950 | 27.9475 | M30 bullish FVG [1146.0000, 1146.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 48 | 2025-08-05 13:00 | 1162.8000 | 1129.9909 | 2025-08-05 20:30 | 1172.5000 | STOCH_EXIT | 15 | 827.64 | 0.73% | 1147.2000 | 1228.4183 | 32.8091 | M30 bullish FVG [1145.5000, 1147.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 49 | 2025-08-11 21:30 | 1224.0000 | 1191.3137 | 2025-08-12 10:30 | 1242.0000 | STOCH_EXIT | 13 | 1542.56 | 1.37% | 1220.5000 | 1289.3727 | 32.6863 | M30 bullish FVG [1217.5000, 1220.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 50 | 2025-08-15 21:30 | 1257.4000 | 1221.8480 | 2025-08-16 02:00 | 1221.8480 | SL | 5 | -3348.04 | -2.92% | 1255.0000 | 1328.5040 | 35.5520 | M30 bullish FVG [1251.8000, 1255.0000], stochastic crossed below 20 | FVG stop 1221.8480 |
| 51 | 2025-08-21 14:00 | 1255.1000 | 1219.4144 | 2025-08-22 09:00 | 1263.6000 | STOCH_EXIT | 25 | 644.42 | 0.58% | 1255.0000 | 1326.4712 | 35.6856 | M30 bullish FVG [1253.4000, 1255.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 52 | 2025-08-26 18:00 | 1258.0000 | 1221.6375 | 2025-08-27 11:30 | 1272.4000 | STOCH_EXIT | 22 | 1169.00 | 1.04% | 1258.3000 | 1330.7251 | 36.3625 | M30 bullish FVG [1257.0000, 1258.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 53 | 2025-09-04 15:30 | 1243.9000 | 1211.0295 | 2025-09-05 07:00 | 1248.1000 | STOCH_EXIT | 17 | 268.81 | 0.24% | 1242.3000 | 1309.6410 | 32.8705 | M30 bullish FVG [1241.9000, 1242.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 54 | 2025-09-05 15:00 | 1244.8000 | 1213.3631 | 2025-09-06 15:00 | 1246.6000 | STOCH_EXIT | 42 | 50.44 | 0.04% | 1246.3000 | 1307.6738 | 31.4369 | M30 bullish FVG [1244.0000, 1246.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 55 | 2025-09-10 07:00 | 1273.2000 | 1243.4272 | 2025-09-10 11:00 | 1275.7000 | STOCH_EXIT | 8 | 109.07 | 0.10% | 1271.1000 | 1332.7455 | 29.7728 | M30 bullish FVG [1270.3000, 1271.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 56 | 2025-09-10 22:00 | 1272.8000 | 1243.0272 | 2025-09-12 07:00 | 1265.6000 | STOCH_EXIT | 39 | -753.76 | -0.67% | 1273.5000 | 1332.3455 | 29.7728 | M30 bullish FVG [1267.2000, 1273.5000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 57 | 2025-09-12 23:00 | 1242.0000 | 1211.9000 | 2025-09-13 05:00 | 1251.6000 | STOCH_EXIT | 7 | 760.14 | 0.67% | 1242.3000 | 1302.2000 | 30.1000 | M30 bullish FVG [1241.5000, 1242.3000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 58 | 2025-09-17 13:00 | 1232.2000 | 1202.1509 | 2025-09-18 12:00 | 1219.7000 | STOCH_EXIT | 32 | -1262.79 | -1.11% | 1235.4000 | 1292.2981 | 30.0491 | M30 bullish FVG [1230.0000, 1235.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 59 | 2025-09-25 14:00 | 1236.0000 | 1204.5717 | 2025-09-27 19:30 | 1239.0000 | STOCH_EXIT | 87 | 160.39 | 0.14% | 1234.8000 | 1298.8565 | 31.4283 | M30 bullish FVG [1231.6000, 1234.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 60 | 2025-10-03 17:00 | 1238.3000 | 1207.0709 | 2025-10-03 22:30 | 1246.7000 | STOCH_EXIT | 11 | 651.33 | 0.58% | 1238.1000 | 1300.7582 | 31.2291 | M30 bullish FVG [1237.4000, 1238.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 61 | 2025-10-07 13:30 | 1247.1000 | 1216.1433 | 2025-10-07 18:30 | 1258.5000 | STOCH_EXIT | 10 | 923.40 | 0.81% | 1248.0000 | 1309.0134 | 30.9567 | M30 bullish FVG [1246.0000, 1248.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 62 | 2025-10-10 14:00 | 1253.2000 | 1220.2254 | 2025-10-11 14:30 | 1246.1000 | STOCH_EXIT | 45 | -759.82 | -0.67% | 1251.4000 | 1319.1491 | 32.9746 | M30 bullish FVG [1250.4000, 1251.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 63 | 2025-10-20 22:00 | 1241.4000 | 1201.3258 | 2025-10-22 07:30 | 1241.3000 | STOCH_EXIT | 41 | -122.06 | -0.11% | 1241.4000 | 1321.5484 | 40.0742 | M30 bullish FVG [1233.0000, 1241.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 64 | 2025-11-06 14:00 | 1289.5000 | 1252.6813 | 2025-11-07 07:00 | 1294.1000 | STOCH_EXIT | 21 | 291.12 | 0.26% | 1285.0000 | 1363.1374 | 36.8187 | M30 bullish FVG [1284.0000, 1285.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 65 | 2025-11-10 18:30 | 1306.7000 | 1271.2884 | 2025-11-11 08:30 | 1318.8000 | STOCH_EXIT | 15 | 938.49 | 0.83% | 1303.1000 | 1377.5231 | 35.4116 | M30 bullish FVG [1303.0000, 1303.1000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 94969.58
- Макс: 115703.37
- Финал: 115116.73

Полная кривая — в `*_equity.csv`.
