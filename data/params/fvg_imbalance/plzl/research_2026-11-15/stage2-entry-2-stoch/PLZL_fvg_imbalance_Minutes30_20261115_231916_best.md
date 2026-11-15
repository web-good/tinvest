# Бэктест PLZL (Minutes30)

- Период: 2023-11-15 23:19 — 2025-11-15 23:19
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
| StopATR | 1.7 |
| TakeProfitATR | 1 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 1 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 123 |
| Выигрышных / проигрышных | 80 / 43 |
| Win rate | 65.04% |
| Gross profit / loss | 70591.98 / 58881.90 |
| Profit factor | 1.199 |
| Чистый PnL | 11710.08 (11.71%) |
| Стартовый / финальный капитал | 100000.00 / 111710.08 |
| Макс. просадка | 12436.23 (10.30%) |
| Средняя прибыль / убыток | 882.40 / 1369.35 |
| Expectancy | 95.20 |
| Лучшая / худшая сделка | 3881.86 / -7003.56 |
| Exposure | 14.52% |
| CAGR | 5.68% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-11-22 16:00 | 1132.8000 | 1100.7731 | 2023-11-23 18:30 | 1136.0000 | STOCH_EXIT | 34 | 181.77 | 0.18% | 1133.0000 | 1151.6394 | 18.8394 | M30 bullish FVG [1132.6000, 1133.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 2 | 2023-11-29 13:30 | 1114.4000 | 1080.8882 | 2023-11-30 16:30 | 1080.8882 | SL | 35 | -3080.24 | -3.10% | 1114.6000 | 1134.1128 | 19.7128 | M30 bullish FVG [1111.8000, 1114.6000], stochastic crossed below 20 | FVG stop 1080.8882 |
| 3 | 2023-12-08 19:30 | 1058.2000 | 1017.6986 | 2023-12-09 17:00 | 1054.4000 | STOCH_EXIT | 23 | -441.92 | -0.46% | 1054.0000 | 1082.0244 | 23.8244 | M30 bullish FVG [1053.8000, 1054.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 4 | 2023-12-13 10:30 | 993.8000 | 949.8511 | 2023-12-13 11:00 | 1009.4000 | STOCH_EXIT | 1 | 1416.04 | 1.47% | 994.6000 | 1019.6523 | 25.8523 | M30 bullish FVG [993.6000, 994.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 5 | 2023-12-13 19:30 | 995.6000 | 951.6511 | 2023-12-13 23:00 | 1008.6000 | STOCH_EXIT | 7 | 1175.79 | 1.20% | 996.6000 | 1021.4523 | 25.8523 | M30 bullish FVG [995.6000, 996.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 6 | 2023-12-14 19:00 | 1004.8000 | 961.2460 | 2023-12-15 14:30 | 1008.4000 | STOCH_EXIT | 20 | 254.15 | 0.26% | 1002.6000 | 1030.4200 | 25.6200 | M30 bullish FVG [999.0000, 1002.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 7 | 2023-12-21 11:00 | 1050.6000 | 1007.1736 | 2023-12-21 22:30 | 1063.8000 | STOCH_EXIT | 23 | 1141.42 | 1.16% | 1049.0000 | 1076.1449 | 25.5449 | M30 bullish FVG [1047.0000, 1049.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 8 | 2023-12-22 14:00 | 1054.4000 | 1011.3070 | 2023-12-25 09:30 | 1063.0000 | STOCH_EXIT | 56 | 716.42 | 0.71% | 1055.0000 | 1079.7488 | 25.3488 | M30 bullish FVG [1054.4000, 1055.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 9 | 2023-12-28 21:30 | 1068.8000 | 1029.0134 | 2023-12-29 18:30 | 1067.8000 | STOCH_EXIT | 23 | -194.42 | -0.19% | 1069.6000 | 1092.2039 | 23.4039 | M30 bullish FVG [1067.2000, 1069.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 10 | 2024-01-04 16:30 | 1078.0000 | 1039.5589 | 2024-01-04 20:30 | 1083.8000 | STOCH_EXIT | 8 | 438.88 | 0.44% | 1079.0000 | 1100.6124 | 22.6124 | M30 bullish FVG [1077.0000, 1079.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 11 | 2024-01-05 15:30 | 1080.6000 | 1043.1562 | 2024-01-06 10:00 | 1085.0000 | STOCH_EXIT | 17 | 308.50 | 0.31% | 1081.0000 | 1102.6258 | 22.0258 | M30 bullish FVG [1080.0000, 1081.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 12 | 2024-01-09 17:30 | 1089.0000 | 1053.3229 | 2024-01-11 18:30 | 1087.8000 | STOCH_EXIT | 60 | -212.82 | -0.21% | 1089.8000 | 1109.9865 | 20.9865 | M30 bullish FVG [1088.8000, 1089.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 13 | 2024-01-12 23:30 | 1098.6000 | 1064.8244 | 2024-01-15 23:00 | 1105.0000 | STOCH_EXIT | 64 | 487.43 | 0.48% | 1095.8000 | 1118.4680 | 19.8680 | M30 bullish FVG [1094.4000, 1095.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 14 | 2024-01-29 23:30 | 1098.0000 | 1070.4966 | 2024-01-30 10:00 | 1101.4000 | STOCH_EXIT | 2 | 213.93 | 0.21% | 1098.6000 | 1114.1785 | 16.1785 | M30 bullish FVG [1098.0000, 1098.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 15 | 2024-01-31 15:00 | 1106.2000 | 1079.9405 | 2024-01-31 18:00 | 1121.6467 | TP | 6 | 1318.62 | 1.30% | 1104.2000 | 1121.6467 | 15.4467 | M30 bullish FVG [1103.8000, 1104.2000], stochastic crossed below 20 | FVG target 1121.6467 |
| 16 | 2024-02-01 16:00 | 1136.0000 | 1106.8805 | 2024-02-02 09:30 | 1140.0000 | STOCH_EXIT | 16 | 260.44 | 0.25% | 1127.2000 | 1153.1291 | 17.1291 | M30 bullish FVG [1109.0000, 1127.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 17 | 2024-02-08 11:00 | 1135.0000 | 1106.5261 | 2024-02-09 18:30 | 1119.2000 | STOCH_EXIT | 44 | -1540.37 | -1.49% | 1135.2000 | 1151.7493 | 16.7493 | M30 bullish FVG [1133.0000, 1135.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 18 | 2024-02-13 17:00 | 1108.0000 | 1078.6722 | 2024-02-15 17:00 | 1104.6000 | STOCH_EXIT | 57 | -414.58 | -0.41% | 1107.8000 | 1125.2516 | 17.2516 | M30 bullish FVG [1105.8000, 1107.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 19 | 2024-02-16 14:30 | 1102.2000 | 1073.8951 | 2024-02-17 17:00 | 1088.2000 | STOCH_EXIT | 33 | -1388.76 | -1.37% | 1104.6000 | 1118.8499 | 16.6499 | M30 bullish FVG [1099.0000, 1104.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 20 | 2024-02-19 23:30 | 1094.0000 | 1063.8069 | 2024-02-20 10:00 | 1104.6000 | STOCH_EXIT | 2 | 864.56 | 0.87% | 1087.0000 | 1111.7607 | 17.7607 | M30 bullish FVG [1086.0000, 1087.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 21 | 2024-02-20 16:30 | 1095.4000 | 1064.9592 | 2024-02-20 21:30 | 1064.9592 | SL | 10 | -2899.93 | -2.88% | 1100.0000 | 1113.3063 | 17.9063 | M30 bullish FVG [1095.2000, 1100.0000], stochastic crossed below 20 | FVG stop 1064.9592 |
| 22 | 2024-02-27 13:00 | 1092.2000 | 1057.8124 | 2024-02-28 10:30 | 1096.2000 | STOCH_EXIT | 24 | 261.52 | 0.27% | 1094.6000 | 1112.4280 | 20.2280 | M30 bullish FVG [1090.6000, 1094.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 23 | 2024-03-05 16:00 | 1145.0000 | 1106.2111 | 2024-03-06 17:30 | 1165.8000 | STOCH_EXIT | 32 | 1689.44 | 1.71% | 1152.2000 | 1167.8170 | 22.8170 | M30 bullish FVG [1144.8000, 1152.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 24 | 2024-03-07 14:30 | 1172.0000 | 1129.4023 | 2024-03-07 22:00 | 1172.8000 | STOCH_EXIT | 15 | -31.65 | -0.03% | 1171.0000 | 1197.0575 | 25.0575 | M30 bullish FVG [1167.0000, 1171.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 25 | 2024-03-11 12:00 | 1177.4000 | 1134.4207 | 2024-03-12 12:30 | 1172.4000 | STOCH_EXIT | 30 | -524.87 | -0.52% | 1180.0000 | 1202.6820 | 25.2820 | M30 bullish FVG [1176.0000, 1180.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 26 | 2024-03-14 18:00 | 1177.4000 | 1134.3949 | 2024-03-15 11:30 | 1184.8000 | STOCH_EXIT | 16 | 522.39 | 0.53% | 1177.6000 | 1202.6971 | 25.2971 | M30 bullish FVG [1177.2000, 1177.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 27 | 2024-03-15 19:00 | 1177.2000 | 1134.6439 | 2024-03-18 09:30 | 1178.6000 | STOCH_EXIT | 46 | 18.88 | 0.02% | 1180.0000 | 1202.2330 | 25.0330 | M30 bullish FVG [1175.2000, 1180.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 28 | 2024-03-19 10:00 | 1162.6000 | 1121.8383 | 2024-03-20 21:00 | 1154.4000 | STOCH_EXIT | 51 | -804.83 | -0.80% | 1164.0000 | 1186.5775 | 23.9775 | M30 bullish FVG [1162.6000, 1164.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 29 | 2024-03-21 18:30 | 1163.8000 | 1120.1639 | 2024-03-23 11:30 | 1146.0000 | STOCH_EXIT | 43 | -1611.17 | -1.63% | 1164.6000 | 1189.4683 | 25.6683 | M30 bullish FVG [1160.6000, 1164.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 30 | 2024-03-27 11:30 | 1155.2000 | 1110.1624 | 2024-03-27 21:30 | 1154.0000 | STOCH_EXIT | 20 | -197.79 | -0.20% | 1159.2000 | 1181.6927 | 26.4927 | M30 bullish FVG [1155.0000, 1159.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 31 | 2024-04-04 14:00 | 1311.0000 | 1257.0709 | 2024-04-04 21:30 | 1322.4000 | STOCH_EXIT | 15 | 746.16 | 0.77% | 1308.6000 | 1342.7230 | 31.7230 | M30 bullish FVG [1307.2000, 1308.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 32 | 2024-04-10 13:00 | 1381.4000 | 1319.4441 | 2024-04-11 13:30 | 1387.2000 | STOCH_EXIT | 30 | 313.51 | 0.32% | 1377.4000 | 1417.8447 | 36.4447 | M30 bullish FVG [1376.8000, 1377.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 33 | 2024-04-12 22:30 | 1449.0000 | 1386.2184 | 2024-04-13 23:00 | 1475.0000 | STOCH_EXIT | 29 | 1668.58 | 1.69% | 1453.6000 | 1485.9304 | 36.9304 | M30 bullish FVG [1449.0000, 1453.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 34 | 2024-04-15 10:00 | 1461.8000 | 1395.3185 | 2024-04-15 21:30 | 1455.0000 | STOCH_EXIT | 23 | -561.57 | -0.56% | 1466.0000 | 1500.9068 | 39.1068 | M30 bullish FVG [1460.6000, 1466.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 35 | 2024-04-19 14:30 | 1424.0000 | 1361.3686 | 2024-04-20 10:00 | 1429.6000 | STOCH_EXIT | 19 | 292.12 | 0.29% | 1420.2000 | 1460.8420 | 36.8420 | M30 bullish FVG [1420.0000, 1420.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 36 | 2024-04-22 19:30 | 1383.4000 | 1322.1579 | 2024-04-23 19:30 | 1376.0000 | STOCH_EXIT | 29 | -632.14 | -0.63% | 1390.2000 | 1419.4247 | 36.0247 | M30 bullish FVG [1383.0000, 1390.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 37 | 2024-04-26 21:30 | 1370.0000 | 1307.3022 | 2024-04-27 16:00 | 1381.0000 | STOCH_EXIT | 18 | 692.96 | 0.70% | 1377.8000 | 1406.8811 | 36.8811 | M30 bullish FVG [1363.8000, 1377.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 38 | 2024-05-02 11:00 | 1323.2000 | 1262.0645 | 2024-05-02 20:30 | 1332.4000 | STOCH_EXIT | 19 | 590.42 | 0.59% | 1323.4000 | 1359.1621 | 35.9621 | M30 bullish FVG [1322.6000, 1323.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 39 | 2024-05-07 10:30 | 1353.0000 | 1292.8839 | 2024-05-08 10:00 | 1358.8000 | STOCH_EXIT | 28 | 328.86 | 0.33% | 1353.0000 | 1388.3624 | 35.3624 | M30 bullish FVG [1348.4000, 1353.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 40 | 2024-05-16 13:30 | 1416.6000 | 1359.3328 | 2024-05-17 09:30 | 1422.2000 | STOCH_EXIT | 21 | 296.82 | 0.29% | 1426.0000 | 1450.2866 | 33.6866 | M30 bullish FVG [1415.4000, 1426.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 41 | 2024-05-20 19:30 | 1447.0000 | 1390.4401 | 2024-05-21 12:00 | 1450.6000 | STOCH_EXIT | 14 | 150.58 | 0.15% | 1441.2000 | 1480.2706 | 33.2706 | M30 bullish FVG [1441.0000, 1441.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 42 | 2024-06-21 13:00 | 1167.8000 | 1080.0331 | 2024-06-22 16:00 | 1187.8000 | STOCH_EXIT | 34 | 1637.53 | 1.61% | 1166.0000 | 1219.4276 | 51.6276 | M30 bullish FVG [1153.8000, 1166.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 43 | 2024-06-24 12:00 | 1183.2000 | 1097.2579 | 2024-06-25 20:00 | 1175.0000 | STOCH_EXIT | 45 | -815.98 | -0.79% | 1187.0000 | 1233.7542 | 50.5542 | M30 bullish FVG [1182.0000, 1187.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 44 | 2024-06-28 17:00 | 1208.4000 | 1132.3651 | 2024-06-29 16:30 | 1216.2000 | STOCH_EXIT | 27 | 553.37 | 0.54% | 1209.2000 | 1253.1264 | 44.7264 | M30 bullish FVG [1208.4000, 1209.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 45 | 2024-07-18 20:00 | 1284.4000 | 1203.4900 | 2024-07-20 21:00 | 1274.0000 | STOCH_EXIT | 58 | -934.34 | -0.91% | 1285.6000 | 1331.9941 | 47.5941 | M30 bullish FVG [1280.8000, 1285.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 46 | 2024-07-22 17:00 | 1280.8000 | 1203.5610 | 2024-07-22 23:30 | 1290.8000 | STOCH_EXIT | 13 | 688.42 | 0.68% | 1276.6000 | 1326.2347 | 45.4347 | M30 bullish FVG [1275.6000, 1276.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 47 | 2024-07-24 12:30 | 1295.2000 | 1220.9424 | 2024-07-24 18:30 | 1306.0000 | STOCH_EXIT | 12 | 750.45 | 0.73% | 1298.0000 | 1338.8809 | 43.6809 | M30 bullish FVG [1293.8000, 1298.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 48 | 2024-08-01 20:30 | 1290.8000 | 1219.9895 | 2024-08-03 18:30 | 1260.8000 | STOCH_EXIT | 53 | -2502.06 | -2.42% | 1290.0000 | 1332.4532 | 41.6532 | M30 bullish FVG [1288.0000, 1290.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 49 | 2024-08-08 23:00 | 1232.0000 | 1163.3879 | 2024-08-12 09:30 | 1232.0000 | STOCH_EXIT | 96 | -101.02 | -0.10% | 1232.6000 | 1272.3601 | 40.3601 | M30 bullish FVG [1232.0000, 1232.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 50 | 2024-08-14 11:00 | 1280.0000 | 1215.5046 | 2024-08-15 14:30 | 1263.4000 | STOCH_EXIT | 39 | -1393.99 | -1.40% | 1280.4000 | 1317.9385 | 37.9385 | M30 bullish FVG [1280.0000, 1280.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 51 | 2024-08-21 07:30 | 1301.4000 | 1239.7618 | 2024-08-21 21:00 | 1299.6000 | STOCH_EXIT | 27 | -235.64 | -0.24% | 1294.0000 | 1337.6578 | 36.2578 | M30 bullish FVG [1293.6000, 1294.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 52 | 2024-08-22 13:00 | 1293.0000 | 1232.1216 | 2024-08-23 11:30 | 1275.0000 | STOCH_EXIT | 31 | -1465.58 | -1.49% | 1292.2000 | 1328.8108 | 35.8108 | M30 bullish FVG [1288.0000, 1292.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 53 | 2024-08-26 10:30 | 1292.0000 | 1227.6357 | 2024-08-27 08:30 | 1301.2000 | STOCH_EXIT | 30 | 592.76 | 0.61% | 1288.2000 | 1329.8614 | 37.8614 | M30 bullish FVG [1286.2000, 1288.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 54 | 2024-08-27 15:30 | 1292.4000 | 1227.1689 | 2024-08-28 07:00 | 1296.8000 | STOCH_EXIT | 17 | 236.01 | 0.24% | 1294.6000 | 1330.7713 | 38.3713 | M30 bullish FVG [1290.0000, 1294.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 55 | 2024-09-04 10:00 | 1149.6000 | 1075.4523 | 2024-09-04 10:30 | 1163.0000 | STOCH_EXIT | 1 | 1040.71 | 1.06% | 1145.2000 | 1193.2163 | 43.6163 | M30 bullish FVG [1144.4000, 1145.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 56 | 2024-09-05 22:00 | 1200.6000 | 1124.6572 | 2024-09-06 10:00 | 1217.8000 | STOCH_EXIT | 10 | 1327.24 | 1.33% | 1197.4000 | 1245.2723 | 44.6723 | M30 bullish FVG [1197.0000, 1197.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 57 | 2024-09-09 10:00 | 1211.0000 | 1138.1809 | 2024-09-09 10:30 | 1230.0000 | STOCH_EXIT | 1 | 1475.70 | 1.47% | 1210.4000 | 1253.8348 | 42.8348 | M30 bullish FVG [1210.0000, 1210.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 58 | 2024-09-10 12:00 | 1238.6000 | 1164.8866 | 2024-09-11 07:30 | 1251.8000 | STOCH_EXIT | 24 | 980.29 | 0.96% | 1246.4000 | 1281.9608 | 43.3608 | M30 bullish FVG [1238.2000, 1246.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 59 | 2024-09-11 21:00 | 1248.4000 | 1176.1632 | 2024-09-12 17:30 | 1266.4000 | STOCH_EXIT | 27 | 1372.89 | 1.34% | 1247.2000 | 1290.8922 | 42.4922 | M30 bullish FVG [1245.0000, 1247.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 60 | 2024-09-19 22:00 | 1322.2000 | 1254.1213 | 2024-09-20 10:30 | 1342.8000 | STOCH_EXIT | 11 | 1522.13 | 1.46% | 1323.6000 | 1362.2463 | 40.0463 | M30 bullish FVG [1320.6000, 1323.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 61 | 2024-09-25 11:00 | 1362.4000 | 1297.7631 | 2024-09-25 12:30 | 1381.6000 | STOCH_EXIT | 3 | 1390.58 | 1.31% | 1364.2000 | 1400.4217 | 38.0217 | M30 bullish FVG [1362.0000, 1364.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 62 | 2024-09-30 19:30 | 1324.0000 | 1261.1784 | 2024-10-01 11:30 | 1328.8000 | STOCH_EXIT | 18 | 281.36 | 0.26% | 1323.4000 | 1360.9539 | 36.9539 | M30 bullish FVG [1320.8000, 1323.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 63 | 2024-10-02 15:30 | 1349.0000 | 1284.6636 | 2024-10-03 23:30 | 1344.8000 | STOCH_EXIT | 50 | -443.75 | -0.41% | 1349.2000 | 1386.8449 | 37.8449 | M30 bullish FVG [1349.0000, 1349.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 64 | 2024-10-04 18:00 | 1342.0000 | 1280.3768 | 2024-10-05 23:00 | 1342.8000 | STOCH_EXIT | 42 | -43.39 | -0.04% | 1342.2000 | 1378.2490 | 36.2490 | M30 bullish FVG [1341.6000, 1342.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 65 | 2024-10-22 16:30 | 1449.4000 | 1394.2341 | 2024-10-23 10:30 | 1470.0000 | STOCH_EXIT | 22 | 1416.38 | 1.32% | 1443.0000 | 1481.8505 | 32.4505 | M30 bullish FVG [1439.6000, 1443.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 66 | 2024-11-08 10:30 | 1473.4000 | 1403.3919 | 2024-11-08 22:30 | 1476.2000 | STOCH_EXIT | 24 | 98.06 | 0.09% | 1482.2000 | 1514.5812 | 41.1812 | M30 bullish FVG [1468.8000, 1482.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 67 | 2024-11-13 14:30 | 1434.2000 | 1361.7478 | 2024-11-14 07:00 | 1476.8190 | TP | 19 | 3128.42 | 2.87% | 1439.6000 | 1476.8190 | 42.6190 | M30 bullish FVG [1428.8000, 1439.6000], stochastic crossed below 20 | FVG target 1476.8190 |
| 68 | 2024-11-14 16:00 | 1412.2000 | 1339.6772 | 2024-11-15 10:00 | 1427.8000 | STOCH_EXIT | 22 | 1120.22 | 1.00% | 1409.6000 | 1454.8605 | 42.6605 | M30 bullish FVG [1407.4000, 1409.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 69 | 2024-11-29 15:00 | 1424.0000 | 1351.5072 | 2024-11-30 22:00 | 1415.4000 | STOCH_EXIT | 48 | -791.56 | -0.70% | 1421.8000 | 1466.6429 | 42.6429 | M30 bullish FVG [1416.8000, 1421.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 70 | 2024-12-03 16:00 | 1445.2000 | 1374.1813 | 2024-12-05 17:30 | 1436.4000 | STOCH_EXIT | 71 | -788.54 | -0.71% | 1442.4000 | 1486.9757 | 41.7757 | M30 bullish FVG [1442.0000, 1442.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 71 | 2024-12-06 14:30 | 1429.0000 | 1361.4463 | 2024-12-07 07:00 | 1439.4000 | STOCH_EXIT | 19 | 699.33 | 0.63% | 1432.4000 | 1468.7375 | 39.7375 | M30 bullish FVG [1428.4000, 1432.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 72 | 2024-12-10 10:30 | 1470.2000 | 1404.3004 | 2024-12-10 20:30 | 1471.2000 | STOCH_EXIT | 20 | -35.77 | -0.03% | 1465.4000 | 1508.9645 | 38.7645 | M30 bullish FVG [1462.0000, 1465.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 73 | 2024-12-13 17:00 | 1369.0000 | 1301.9753 | 2024-12-15 21:00 | 1366.2000 | STOCH_EXIT | 76 | -341.74 | -0.30% | 1374.2000 | 1408.4263 | 39.4263 | M30 bullish FVG [1363.0000, 1374.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 74 | 2024-12-18 23:00 | 1382.0000 | 1299.8849 | 2024-12-21 07:00 | 1363.8000 | STOCH_EXIT | 70 | -1585.40 | -1.42% | 1370.4000 | 1430.3030 | 48.3030 | M30 bullish FVG [1369.4000, 1370.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 75 | 2025-01-07 07:30 | 1404.0000 | 1338.0843 | 2025-01-07 17:30 | 1409.0000 | STOCH_EXIT | 20 | 280.29 | 0.26% | 1404.0000 | 1442.7739 | 38.7739 | M30 bullish FVG [1403.6000, 1404.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 76 | 2025-01-09 14:00 | 1436.8000 | 1371.2212 | 2025-01-09 17:00 | 1452.0000 | STOCH_EXIT | 6 | 1059.18 | 0.96% | 1436.2000 | 1475.3758 | 38.5758 | M30 bullish FVG [1436.0000, 1436.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 77 | 2025-01-29 10:30 | 1702.8000 | 1625.5593 | 2025-01-30 06:30 | 1703.6000 | STOCH_EXIT | 27 | -58.71 | -0.05% | 1695.0000 | 1748.2357 | 45.4357 | M30 bullish FVG [1688.2000, 1695.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 78 | 2025-01-31 11:30 | 1724.6000 | 1649.1423 | 2025-02-01 07:30 | 1721.0000 | STOCH_EXIT | 35 | -340.66 | -0.31% | 1715.2000 | 1768.9869 | 44.3869 | M30 bullish FVG [1712.0000, 1715.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 79 | 2025-02-10 21:00 | 1797.0000 | 1721.7949 | 2025-02-11 07:30 | 1807.2000 | STOCH_EXIT | 8 | 520.67 | 0.47% | 1800.0000 | 1841.2383 | 44.2383 | M30 bullish FVG [1795.0000, 1800.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 80 | 2025-02-13 17:00 | 1907.2000 | 1830.2443 | 2025-02-14 07:00 | 1952.4680 | TP | 8 | 2513.62 | 2.27% | 1875.2000 | 1952.4680 | 45.2680 | M30 bullish FVG [1870.0000, 1875.2000], stochastic crossed below 20 | FVG target 1952.4680 |
| 81 | 2025-02-26 14:00 | 1938.6000 | 1842.5104 | 2025-02-28 14:00 | 1927.6000 | STOCH_EXIT | 70 | -763.05 | -0.67% | 1938.8000 | 1995.1233 | 56.5233 | M30 bullish FVG [1938.2000, 1938.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 82 | 2025-03-12 09:00 | 1893.6000 | 1796.8331 | 2025-03-12 18:00 | 1906.2000 | STOCH_EXIT | 18 | 642.01 | 0.56% | 1874.6000 | 1950.5217 | 56.9217 | M30 bullish FVG [1871.0000, 1874.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 83 | 2025-03-13 08:00 | 1892.8000 | 1799.8121 | 2025-03-13 19:00 | 1904.0000 | STOCH_EXIT | 22 | 558.10 | 0.49% | 1892.2000 | 1947.4987 | 54.6987 | M30 bullish FVG [1892.0000, 1892.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 84 | 2025-04-03 09:30 | 1796.8000 | 1690.3336 | 2025-04-04 11:30 | 1750.0000 | STOCH_EXIT | 39 | -3108.70 | -2.70% | 1797.0000 | 1859.4273 | 62.6273 | M30 bullish FVG [1793.0000, 1797.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 85 | 2025-04-08 20:00 | 1751.4000 | 1628.6813 | 2025-04-09 09:30 | 1814.8000 | STOCH_EXIT | 14 | 3881.86 | 3.52% | 1745.2000 | 1823.5875 | 72.1875 | M30 bullish FVG [1736.0000, 1745.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 86 | 2025-04-09 18:00 | 1791.6000 | 1665.4070 | 2025-04-09 20:00 | 1838.6000 | STOCH_EXIT | 4 | 2891.83 | 2.52% | 1799.8000 | 1865.8312 | 74.2312 | M30 bullish FVG [1787.6000, 1799.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 87 | 2025-04-10 12:00 | 1806.6000 | 1678.7593 | 2025-04-10 20:00 | 1832.2000 | STOCH_EXIT | 16 | 1545.74 | 1.32% | 1797.8000 | 1881.8004 | 75.2004 | M30 bullish FVG [1793.0000, 1797.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 88 | 2025-04-17 10:30 | 1929.0000 | 1814.5320 | 2025-04-18 09:00 | 1924.2000 | STOCH_EXIT | 32 | -417.05 | -0.35% | 1929.2000 | 1996.3341 | 67.3341 | M30 bullish FVG [1928.8000, 1929.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 89 | 2025-04-24 21:00 | 1895.0000 | 1785.6727 | 2025-04-29 10:30 | 1785.6727 | SL | 172 | -7003.56 | -5.86% | 1892.2000 | 1959.3102 | 64.3102 | M30 bullish FVG [1890.0000, 1892.2000], stochastic crossed below 20 | FVG stop 1785.6727 |
| 90 | 2025-05-01 09:30 | 1746.8000 | 1638.5777 | 2025-05-02 06:30 | 1751.4000 | STOCH_EXIT | 29 | 182.46 | 0.16% | 1746.0000 | 1810.4602 | 63.6602 | M30 bullish FVG [1745.8000, 1746.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 91 | 2025-05-08 17:00 | 1750.4000 | 1642.3022 | 2025-05-09 03:00 | 1758.6000 | STOCH_EXIT | 16 | 412.51 | 0.37% | 1751.8000 | 1813.9869 | 63.5869 | M30 bullish FVG [1749.8000, 1751.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 92 | 2025-05-14 08:00 | 1749.6000 | 1646.3800 | 2025-05-16 08:00 | 1691.0000 | STOCH_EXIT | 70 | -3860.50 | -3.45% | 1753.8000 | 1810.3177 | 60.7177 | M30 bullish FVG [1749.4000, 1753.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 93 | 2025-05-29 10:00 | 1676.0000 | 1575.9170 | 2025-05-29 13:00 | 1693.2000 | STOCH_EXIT | 6 | 1008.50 | 0.93% | 1673.4000 | 1734.8723 | 58.8723 | M30 bullish FVG [1669.0000, 1673.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 94 | 2025-06-03 18:30 | 1730.0000 | 1633.6511 | 2025-06-04 12:00 | 1762.8000 | STOCH_EXIT | 22 | 1956.38 | 1.79% | 1736.2000 | 1786.6758 | 56.6758 | M30 bullish FVG [1727.6000, 1736.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 95 | 2025-06-04 23:00 | 1758.6000 | 1665.4903 | 2025-06-05 12:00 | 1777.6000 | STOCH_EXIT | 13 | 1085.61 | 0.98% | 1761.6000 | 1813.3704 | 54.7704 | M30 bullish FVG [1756.0000, 1761.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 96 | 2025-06-06 22:30 | 1751.2000 | 1655.5212 | 2025-06-07 06:30 | 1756.0000 | STOCH_EXIT | 12 | 194.97 | 0.17% | 1750.0000 | 1807.4816 | 56.2816 | M30 bullish FVG [1747.6000, 1750.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 97 | 2025-06-10 12:00 | 1701.8000 | 1598.6534 | 2025-06-10 18:00 | 1734.2000 | STOCH_EXIT | 12 | 2025.01 | 1.80% | 1707.8000 | 1762.4745 | 60.6745 | M30 bullish FVG [1701.2000, 1707.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 98 | 2025-06-17 15:00 | 1732.8000 | 1631.2882 | 2025-06-18 06:30 | 1745.0000 | STOCH_EXIT | 18 | 690.43 | 0.60% | 1748.6000 | 1792.5128 | 59.7128 | M30 bullish FVG [1722.0000, 1748.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 99 | 2025-06-27 07:00 | 1768.0000 | 1681.2633 | 2025-06-27 15:30 | 1789.8000 | STOCH_EXIT | 17 | 1301.37 | 1.13% | 1768.2000 | 1819.0216 | 51.0216 | M30 bullish FVG [1766.8000, 1768.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 100 | 2025-07-11 09:00 | 1854.8000 | 1765.9225 | 2025-07-12 06:00 | 1803.8000 | STOCH_EXIT | 38 | -3328.25 | -2.85% | 1859.2000 | 1907.0809 | 52.2809 | M30 bullish FVG [1853.0000, 1859.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 101 | 2025-07-17 09:00 | 1883.2000 | 1785.9907 | 2025-07-17 17:00 | 1896.4000 | STOCH_EXIT | 16 | 678.61 | 0.60% | 1873.8000 | 1940.3820 | 57.1820 | M30 bullish FVG [1872.0000, 1873.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 102 | 2025-07-22 11:00 | 1933.4000 | 1832.0794 | 2025-07-22 16:00 | 1955.4000 | STOCH_EXIT | 10 | 1183.28 | 1.04% | 1922.4000 | 1993.0004 | 59.6004 | M30 bullish FVG [1919.8000, 1922.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 103 | 2025-07-30 15:30 | 1976.6000 | 1878.3641 | 2025-07-31 10:00 | 1964.4000 | STOCH_EXIT | 24 | -821.89 | -0.72% | 1976.8000 | 2034.3858 | 57.7858 | M30 bullish FVG [1976.0000, 1976.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 104 | 2025-08-01 11:30 | 1963.4000 | 1864.0402 | 2025-08-02 05:00 | 1970.8000 | STOCH_EXIT | 31 | 315.11 | 0.28% | 1959.8000 | 2021.8469 | 58.4469 | M30 bullish FVG [1958.2000, 1959.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 105 | 2025-08-05 14:00 | 2027.0000 | 1927.2661 | 2025-08-05 16:30 | 2046.2000 | STOCH_EXIT | 5 | 978.31 | 0.85% | 2026.8000 | 2085.6670 | 58.6670 | M30 bullish FVG [2025.0000, 2026.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 106 | 2025-08-06 13:00 | 2040.0000 | 1941.5857 | 2025-08-06 21:30 | 2056.0000 | STOCH_EXIT | 17 | 795.26 | 0.68% | 2031.4000 | 2097.8908 | 57.8908 | M30 bullish FVG [2025.0000, 2031.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 107 | 2025-08-11 10:30 | 2164.6000 | 2067.2040 | 2025-08-12 11:30 | 2067.2040 | SL | 37 | -5373.64 | -4.59% | 2207.2000 | 2221.8918 | 57.2918 | M30 bullish FVG [2150.0000, 2207.2000], stochastic crossed below 20 | FVG stop 2067.2040 |
| 108 | 2025-08-19 18:30 | 2132.2000 | 2031.4781 | 2025-08-20 11:30 | 2153.6000 | STOCH_EXIT | 21 | 1001.37 | 0.90% | 2104.0000 | 2191.4482 | 59.2482 | M30 bullish FVG [2102.6000, 2104.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 109 | 2025-08-21 19:00 | 2144.0000 | 2046.9045 | 2025-08-22 08:30 | 2162.0000 | STOCH_EXIT | 14 | 824.04 | 0.74% | 2141.4000 | 2201.1150 | 57.1150 | M30 bullish FVG [2141.2000, 2141.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 110 | 2025-09-02 09:30 | 2175.6000 | 2079.0689 | 2025-09-02 15:00 | 2194.0000 | STOCH_EXIT | 11 | 843.19 | 0.74% | 2168.4000 | 2232.3830 | 56.7830 | M30 bullish FVG [2166.4000, 2168.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 111 | 2025-09-23 22:30 | 2325.2000 | 2238.5033 | 2025-09-24 14:30 | 2328.6000 | STOCH_EXIT | 19 | 52.58 | 0.05% | 2319.2000 | 2376.1981 | 50.9981 | M30 bullish FVG [2313.4000, 2319.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 112 | 2025-09-25 07:30 | 2307.0000 | 2221.5380 | 2025-09-25 09:00 | 2320.0000 | STOCH_EXIT | 3 | 523.64 | 0.46% | 2304.6000 | 2357.2718 | 50.2718 | M30 bullish FVG [2304.2000, 2304.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 113 | 2025-09-25 18:30 | 2308.2000 | 2222.7380 | 2025-09-26 09:00 | 2319.4000 | STOCH_EXIT | 16 | 444.31 | 0.38% | 2304.6000 | 2358.4718 | 50.2718 | M30 bullish FVG [2304.2000, 2304.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 114 | 2025-09-30 12:00 | 2372.2000 | 2287.4947 | 2025-10-01 07:00 | 2384.0000 | STOCH_EXIT | 25 | 452.25 | 0.40% | 2376.2000 | 2422.0267 | 49.8267 | M30 bullish FVG [2365.0000, 2376.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 115 | 2025-10-01 17:30 | 2356.2000 | 2265.5236 | 2025-10-02 09:00 | 2374.0000 | STOCH_EXIT | 18 | 756.31 | 0.65% | 2357.2000 | 2409.5390 | 53.3390 | M30 bullish FVG [2355.2000, 2357.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 116 | 2025-10-07 12:30 | 2370.0000 | 2274.9540 | 2025-10-07 21:00 | 2383.6000 | STOCH_EXIT | 17 | 549.94 | 0.47% | 2364.4000 | 2425.9094 | 55.9094 | M30 bullish FVG [2360.0000, 2364.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 117 | 2025-10-17 12:30 | 2253.4000 | 2114.5037 | 2025-10-18 09:00 | 2210.6000 | STOCH_EXIT | 37 | -2341.66 | -2.00% | 2245.0000 | 2335.1037 | 81.7037 | M30 bullish FVG [2243.0000, 2245.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 118 | 2025-10-21 08:30 | 2145.0000 | 1992.4985 | 2025-10-22 10:00 | 2081.8000 | STOCH_EXIT | 38 | -3461.61 | -3.04% | 2160.0000 | 2234.7068 | 89.7068 | M30 bullish FVG [2144.0000, 2160.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 119 | 2025-10-24 15:00 | 2045.0000 | 1889.1600 | 2025-10-26 03:30 | 2050.6000 | STOCH_EXIT | 65 | 191.82 | 0.17% | 2042.4000 | 2136.6706 | 91.6706 | M30 bullish FVG [2042.2000, 2042.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 120 | 2025-10-30 07:00 | 2039.8000 | 1891.6702 | 2025-10-30 09:30 | 2070.2000 | STOCH_EXIT | 5 | 1530.63 | 1.39% | 2044.6000 | 2126.9352 | 87.1352 | M30 bullish FVG [2036.4000, 2044.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 121 | 2025-10-31 14:30 | 2075.6000 | 1931.0566 | 2025-11-01 23:30 | 2062.0000 | STOCH_EXIT | 53 | -846.12 | -0.75% | 2086.2000 | 2160.6255 | 85.0255 | M30 bullish FVG [2073.0000, 2086.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 122 | 2025-11-03 20:30 | 2091.4000 | 1947.6125 | 2025-11-05 06:30 | 2094.6000 | STOCH_EXIT | 51 | 58.67 | 0.05% | 2086.2000 | 2175.9809 | 84.5809 | M30 bullish FVG [2075.0000, 2086.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 123 | 2025-11-13 22:30 | 2176.2000 | 2059.0980 | 2025-11-15 02:30 | 2156.0000 | STOCH_EXIT | 39 | -1140.67 | -1.03% | 2174.4000 | 2245.0835 | 68.8835 | M30 bullish FVG [2168.2000, 2174.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 95797.59
- Макс: 120754.26
- Финал: 111710.08

Полная кривая — в `*_equity.csv`.
