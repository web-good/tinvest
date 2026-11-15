# Бэктест PLZL (Minutes30)

- Период: 2023-11-15 23:18 — 2025-11-15 23:18
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
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 1 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 124 |
| Выигрышных / проигрышных | 79 / 45 |
| Win rate | 63.71% |
| Gross profit / loss | 63266.41 / 60733.47 |
| Profit factor | 1.042 |
| Чистый PnL | 2532.94 (2.53%) |
| Стартовый / финальный капитал | 100000.00 / 102532.94 |
| Макс. просадка | 10499.54 (9.59%) |
| Средняя прибыль / убыток | 800.84 / 1349.63 |
| Expectancy | 20.43 |
| Лучшая / худшая сделка | 3450.55 / -4590.35 |
| Exposure | 12.92% |
| CAGR | 1.26% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-11-22 16:00 | 1132.8000 | 1113.9606 | 2023-11-23 18:30 | 1136.0000 | STOCH_EXIT | 34 | 181.77 | 0.18% | 1133.0000 | 1170.4787 | 18.8394 | M30 bullish FVG [1132.6000, 1133.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 2 | 2023-11-29 13:30 | 1114.4000 | 1094.6872 | 2023-11-30 11:00 | 1094.6872 | SL | 24 | -1852.74 | -1.87% | 1114.6000 | 1153.8256 | 19.7128 | M30 bullish FVG [1111.8000, 1114.6000], stochastic crossed below 20 | FVG stop 1094.6872 |
| 3 | 2023-12-08 19:30 | 1058.2000 | 1034.3756 | 2023-12-09 17:00 | 1054.4000 | STOCH_EXIT | 23 | -446.78 | -0.46% | 1054.0000 | 1105.8487 | 23.8244 | M30 bullish FVG [1053.8000, 1054.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 4 | 2023-12-13 10:30 | 993.8000 | 967.9477 | 2023-12-13 11:00 | 1009.4000 | STOCH_EXIT | 1 | 1430.64 | 1.47% | 994.6000 | 1045.5046 | 25.8523 | M30 bullish FVG [993.6000, 994.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 5 | 2023-12-13 19:30 | 995.6000 | 969.7477 | 2023-12-13 23:00 | 1008.6000 | STOCH_EXIT | 7 | 1187.79 | 1.20% | 996.6000 | 1047.3046 | 25.8523 | M30 bullish FVG [995.6000, 996.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 6 | 2023-12-14 19:00 | 1004.8000 | 979.1800 | 2023-12-15 14:30 | 1008.4000 | STOCH_EXIT | 20 | 256.75 | 0.26% | 1002.6000 | 1056.0400 | 25.6200 | M30 bullish FVG [999.0000, 1002.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 7 | 2023-12-21 11:00 | 1050.6000 | 1025.0551 | 2023-12-21 22:30 | 1063.8000 | STOCH_EXIT | 23 | 1153.57 | 1.16% | 1049.0000 | 1101.6898 | 25.5449 | M30 bullish FVG [1047.0000, 1049.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 8 | 2023-12-22 14:00 | 1054.4000 | 1029.0512 | 2023-12-25 09:30 | 1063.0000 | STOCH_EXIT | 56 | 723.96 | 0.71% | 1055.0000 | 1105.0977 | 25.3488 | M30 bullish FVG [1054.4000, 1055.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 9 | 2023-12-28 21:30 | 1068.8000 | 1045.3961 | 2023-12-29 18:30 | 1067.8000 | STOCH_EXIT | 23 | -196.49 | -0.19% | 1069.6000 | 1115.6078 | 23.4039 | M30 bullish FVG [1067.2000, 1069.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 10 | 2024-01-04 16:30 | 1078.0000 | 1055.3876 | 2024-01-04 20:30 | 1083.8000 | STOCH_EXIT | 8 | 443.60 | 0.44% | 1079.0000 | 1123.2248 | 22.6124 | M30 bullish FVG [1077.0000, 1079.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 11 | 2024-01-05 15:30 | 1080.6000 | 1058.5742 | 2024-01-06 10:00 | 1085.0000 | STOCH_EXIT | 17 | 315.13 | 0.31% | 1081.0000 | 1124.6516 | 22.0258 | M30 bullish FVG [1080.0000, 1081.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 12 | 2024-01-09 17:30 | 1089.0000 | 1068.0135 | 2024-01-11 18:30 | 1087.8000 | STOCH_EXIT | 60 | -215.11 | -0.21% | 1089.8000 | 1130.9730 | 20.9865 | M30 bullish FVG [1088.8000, 1089.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 13 | 2024-01-12 23:30 | 1098.6000 | 1078.7320 | 2024-01-15 23:00 | 1105.0000 | STOCH_EXIT | 64 | 492.73 | 0.48% | 1095.8000 | 1138.3360 | 19.8680 | M30 bullish FVG [1094.4000, 1095.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 14 | 2024-01-29 23:30 | 1098.0000 | 1081.8215 | 2024-01-30 10:00 | 1101.4000 | STOCH_EXIT | 2 | 216.23 | 0.21% | 1098.6000 | 1130.3569 | 16.1785 | M30 bullish FVG [1098.0000, 1098.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 15 | 2024-01-31 15:00 | 1106.2000 | 1090.7533 | 2024-01-31 18:00 | 1131.0000 | STOCH_EXIT | 6 | 2202.37 | 2.14% | 1104.2000 | 1137.0935 | 15.4467 | M30 bullish FVG [1103.8000, 1104.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 16 | 2024-02-01 16:00 | 1136.0000 | 1118.8709 | 2024-02-02 09:30 | 1140.0000 | STOCH_EXIT | 16 | 266.17 | 0.25% | 1127.2000 | 1170.2582 | 17.1291 | M30 bullish FVG [1109.0000, 1127.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 17 | 2024-02-08 11:00 | 1135.0000 | 1118.2507 | 2024-02-08 12:00 | 1118.2507 | SL | 2 | -1662.47 | -1.57% | 1135.2000 | 1168.4987 | 16.7493 | M30 bullish FVG [1133.0000, 1135.2000], stochastic crossed below 20 | FVG stop 1118.2507 |
| 18 | 2024-02-13 17:00 | 1108.0000 | 1090.7484 | 2024-02-15 17:00 | 1104.6000 | STOCH_EXIT | 57 | -423.59 | -0.41% | 1107.8000 | 1142.5033 | 17.2516 | M30 bullish FVG [1105.8000, 1107.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 19 | 2024-02-16 14:30 | 1102.2000 | 1085.5501 | 2024-02-16 16:30 | 1085.5501 | SL | 4 | -1667.92 | -1.61% | 1104.6000 | 1135.4999 | 16.6499 | M30 bullish FVG [1099.0000, 1104.6000], stochastic crossed below 20 | FVG stop 1085.5501 |
| 20 | 2024-02-16 23:30 | 1082.4000 | 1065.7501 | 2024-02-17 17:00 | 1088.2000 | STOCH_EXIT | 15 | 443.18 | 0.44% | 1084.6000 | 1115.6999 | 16.6499 | M30 bullish FVG [1082.0000, 1084.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 21 | 2024-02-19 23:30 | 1094.0000 | 1076.2393 | 2024-02-20 10:00 | 1104.6000 | STOCH_EXIT | 2 | 883.57 | 0.87% | 1087.0000 | 1129.5213 | 17.7607 | M30 bullish FVG [1086.0000, 1087.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 22 | 2024-02-20 16:30 | 1095.4000 | 1077.4937 | 2024-02-20 20:00 | 1077.4937 | SL | 7 | -1785.32 | -1.73% | 1100.0000 | 1131.2127 | 17.9063 | M30 bullish FVG [1095.2000, 1100.0000], stochastic crossed below 20 | FVG stop 1077.4937 |
| 23 | 2024-02-27 13:00 | 1092.2000 | 1071.9720 | 2024-02-28 10:30 | 1096.2000 | STOCH_EXIT | 24 | 270.24 | 0.27% | 1094.6000 | 1132.6560 | 20.2280 | M30 bullish FVG [1090.6000, 1094.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 24 | 2024-03-05 16:00 | 1145.0000 | 1122.1830 | 2024-03-06 17:30 | 1165.8000 | STOCH_EXIT | 32 | 1748.37 | 1.71% | 1152.2000 | 1190.6339 | 22.8170 | M30 bullish FVG [1144.8000, 1152.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 25 | 2024-03-07 14:30 | 1172.0000 | 1146.9425 | 2024-03-07 22:00 | 1172.8000 | STOCH_EXIT | 15 | -32.77 | -0.03% | 1171.0000 | 1222.1150 | 25.0575 | M30 bullish FVG [1167.0000, 1171.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 26 | 2024-03-11 12:00 | 1177.4000 | 1152.1180 | 2024-03-12 12:30 | 1172.4000 | STOCH_EXIT | 30 | -543.39 | -0.52% | 1180.0000 | 1227.9639 | 25.2820 | M30 bullish FVG [1176.0000, 1180.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 27 | 2024-03-14 18:00 | 1177.4000 | 1152.1029 | 2024-03-15 11:30 | 1184.8000 | STOCH_EXIT | 16 | 541.04 | 0.53% | 1177.6000 | 1227.9942 | 25.2971 | M30 bullish FVG [1177.2000, 1177.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 28 | 2024-03-15 19:00 | 1177.2000 | 1152.1670 | 2024-03-18 09:30 | 1178.6000 | STOCH_EXIT | 46 | 19.54 | 0.02% | 1180.0000 | 1227.2661 | 25.0330 | M30 bullish FVG [1175.2000, 1180.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 29 | 2024-03-19 10:00 | 1162.6000 | 1138.6225 | 2024-03-20 15:00 | 1138.6225 | SL | 39 | -2236.40 | -2.16% | 1164.0000 | 1210.5549 | 23.9775 | M30 bullish FVG [1162.6000, 1164.0000], stochastic crossed below 20 | FVG stop 1138.6225 |
| 30 | 2024-03-21 18:30 | 1163.8000 | 1138.1317 | 2024-03-22 20:30 | 1138.1317 | SL | 33 | -2333.28 | -2.30% | 1164.6000 | 1215.1366 | 25.6683 | M30 bullish FVG [1160.6000, 1164.6000], stochastic crossed below 20 | FVG stop 1138.1317 |
| 31 | 2024-03-27 11:30 | 1155.2000 | 1128.7073 | 2024-03-27 21:30 | 1154.0000 | STOCH_EXIT | 20 | -200.14 | -0.20% | 1159.2000 | 1208.1854 | 26.4927 | M30 bullish FVG [1155.0000, 1159.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 32 | 2024-04-04 14:00 | 1311.0000 | 1279.2770 | 2024-04-04 21:30 | 1322.4000 | STOCH_EXIT | 15 | 756.25 | 0.77% | 1308.6000 | 1374.4460 | 31.7230 | M30 bullish FVG [1307.2000, 1308.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 33 | 2024-04-10 13:00 | 1381.4000 | 1344.9553 | 2024-04-11 13:30 | 1387.2000 | STOCH_EXIT | 30 | 317.93 | 0.32% | 1377.4000 | 1454.2893 | 36.4447 | M30 bullish FVG [1376.8000, 1377.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 34 | 2024-04-12 22:30 | 1449.0000 | 1412.0696 | 2024-04-13 23:00 | 1475.0000 | STOCH_EXIT | 29 | 1693.12 | 1.69% | 1453.6000 | 1522.8607 | 36.9304 | M30 bullish FVG [1449.0000, 1453.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 35 | 2024-04-15 10:00 | 1461.8000 | 1422.6932 | 2024-04-15 21:30 | 1455.0000 | STOCH_EXIT | 23 | -569.83 | -0.56% | 1466.0000 | 1540.0135 | 39.1068 | M30 bullish FVG [1460.6000, 1466.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 36 | 2024-04-19 14:30 | 1424.0000 | 1387.1580 | 2024-04-20 10:00 | 1429.6000 | STOCH_EXIT | 19 | 296.30 | 0.29% | 1420.2000 | 1497.6841 | 36.8420 | M30 bullish FVG [1420.0000, 1420.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 37 | 2024-04-22 19:30 | 1383.4000 | 1347.3753 | 2024-04-23 19:30 | 1376.0000 | STOCH_EXIT | 29 | -640.92 | -0.63% | 1390.2000 | 1455.4495 | 36.0247 | M30 bullish FVG [1383.0000, 1390.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 38 | 2024-04-26 21:30 | 1370.0000 | 1333.1189 | 2024-04-27 16:00 | 1381.0000 | STOCH_EXIT | 18 | 702.59 | 0.70% | 1377.8000 | 1443.7621 | 36.8811 | M30 bullish FVG [1363.8000, 1377.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 39 | 2024-05-02 11:00 | 1323.2000 | 1287.2379 | 2024-05-02 20:30 | 1332.4000 | STOCH_EXIT | 19 | 598.29 | 0.59% | 1323.4000 | 1395.1241 | 35.9621 | M30 bullish FVG [1322.6000, 1323.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 40 | 2024-05-07 10:30 | 1353.0000 | 1317.6376 | 2024-05-08 10:00 | 1358.8000 | STOCH_EXIT | 28 | 333.31 | 0.33% | 1353.0000 | 1423.7248 | 35.3624 | M30 bullish FVG [1348.4000, 1353.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 41 | 2024-05-16 13:30 | 1416.6000 | 1382.9134 | 2024-05-17 09:30 | 1422.2000 | STOCH_EXIT | 21 | 301.00 | 0.29% | 1426.0000 | 1483.9731 | 33.6866 | M30 bullish FVG [1415.4000, 1426.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 42 | 2024-05-20 19:30 | 1447.0000 | 1413.7294 | 2024-05-21 12:00 | 1450.6000 | STOCH_EXIT | 14 | 152.74 | 0.15% | 1441.2000 | 1513.5411 | 33.2706 | M30 bullish FVG [1441.0000, 1441.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 43 | 2024-06-21 13:00 | 1167.8000 | 1116.1724 | 2024-06-22 16:00 | 1187.8000 | STOCH_EXIT | 34 | 1656.35 | 1.61% | 1166.0000 | 1271.0552 | 51.6276 | M30 bullish FVG [1153.8000, 1166.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 44 | 2024-06-24 12:00 | 1183.2000 | 1132.6458 | 2024-06-25 20:00 | 1175.0000 | STOCH_EXIT | 45 | -825.36 | -0.79% | 1187.0000 | 1284.3084 | 50.5542 | M30 bullish FVG [1182.0000, 1187.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 45 | 2024-06-28 17:00 | 1208.4000 | 1163.6736 | 2024-06-29 16:30 | 1216.2000 | STOCH_EXIT | 27 | 559.95 | 0.54% | 1209.2000 | 1297.8528 | 44.7264 | M30 bullish FVG [1208.4000, 1209.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 46 | 2024-07-18 20:00 | 1284.4000 | 1236.8059 | 2024-07-20 21:00 | 1274.0000 | STOCH_EXIT | 58 | -946.02 | -0.91% | 1285.6000 | 1379.5882 | 47.5941 | M30 bullish FVG [1280.8000, 1285.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 47 | 2024-07-22 17:00 | 1280.8000 | 1235.3653 | 2024-07-22 23:30 | 1290.8000 | STOCH_EXIT | 13 | 697.14 | 0.68% | 1276.6000 | 1371.6694 | 45.4347 | M30 bullish FVG [1275.6000, 1276.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 48 | 2024-07-24 12:30 | 1295.2000 | 1251.5191 | 2024-07-24 18:30 | 1306.0000 | STOCH_EXIT | 12 | 759.95 | 0.73% | 1298.0000 | 1382.5619 | 43.6809 | M30 bullish FVG [1293.8000, 1298.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 49 | 2024-08-01 20:30 | 1290.8000 | 1249.1468 | 2024-08-03 18:30 | 1260.8000 | STOCH_EXIT | 53 | -2533.34 | -2.42% | 1290.0000 | 1374.1064 | 41.6532 | M30 bullish FVG [1288.0000, 1290.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 50 | 2024-08-08 23:00 | 1232.0000 | 1191.6399 | 2024-08-12 09:30 | 1232.0000 | STOCH_EXIT | 96 | -102.26 | -0.10% | 1232.6000 | 1312.7201 | 40.3601 | M30 bullish FVG [1232.0000, 1232.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 51 | 2024-08-14 11:00 | 1280.0000 | 1242.0615 | 2024-08-15 14:30 | 1263.4000 | STOCH_EXIT | 39 | -1411.86 | -1.40% | 1280.4000 | 1355.8769 | 37.9385 | M30 bullish FVG [1280.0000, 1280.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 52 | 2024-08-21 07:30 | 1301.4000 | 1265.1422 | 2024-08-21 21:00 | 1299.6000 | STOCH_EXIT | 27 | -238.74 | -0.24% | 1294.0000 | 1373.9156 | 36.2578 | M30 bullish FVG [1293.6000, 1294.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 53 | 2024-08-22 13:00 | 1293.0000 | 1257.1892 | 2024-08-22 20:30 | 1257.1892 | SL | 15 | -2855.61 | -2.87% | 1292.2000 | 1364.6216 | 35.8108 | M30 bullish FVG [1288.0000, 1292.2000], stochastic crossed below 20 | FVG stop 1257.1892 |
| 54 | 2024-08-26 10:30 | 1292.0000 | 1254.1386 | 2024-08-27 08:30 | 1301.2000 | STOCH_EXIT | 30 | 592.76 | 0.61% | 1288.2000 | 1367.7227 | 37.8614 | M30 bullish FVG [1286.2000, 1288.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 55 | 2024-08-27 15:30 | 1292.4000 | 1254.0287 | 2024-08-28 07:00 | 1296.8000 | STOCH_EXIT | 17 | 236.01 | 0.24% | 1294.6000 | 1369.1425 | 38.3713 | M30 bullish FVG [1290.0000, 1294.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 56 | 2024-09-04 10:00 | 1149.6000 | 1105.9837 | 2024-09-04 10:30 | 1163.0000 | STOCH_EXIT | 1 | 1040.71 | 1.06% | 1145.2000 | 1236.8326 | 43.6163 | M30 bullish FVG [1144.4000, 1145.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 57 | 2024-09-05 22:00 | 1200.6000 | 1155.9277 | 2024-09-06 10:00 | 1217.8000 | STOCH_EXIT | 10 | 1327.24 | 1.33% | 1197.4000 | 1289.9445 | 44.6723 | M30 bullish FVG [1197.0000, 1197.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 58 | 2024-09-09 10:00 | 1211.0000 | 1168.1652 | 2024-09-09 10:30 | 1230.0000 | STOCH_EXIT | 1 | 1475.70 | 1.47% | 1210.4000 | 1296.6695 | 42.8348 | M30 bullish FVG [1210.0000, 1210.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 59 | 2024-09-10 12:00 | 1238.6000 | 1195.2392 | 2024-09-11 07:30 | 1251.8000 | STOCH_EXIT | 24 | 980.29 | 0.96% | 1246.4000 | 1325.3217 | 43.3608 | M30 bullish FVG [1238.2000, 1246.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 60 | 2024-09-11 21:00 | 1248.4000 | 1205.9078 | 2024-09-12 17:30 | 1266.4000 | STOCH_EXIT | 27 | 1372.89 | 1.34% | 1247.2000 | 1333.3844 | 42.4922 | M30 bullish FVG [1245.0000, 1247.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 61 | 2024-09-19 22:00 | 1322.2000 | 1282.1537 | 2024-09-20 10:30 | 1342.8000 | STOCH_EXIT | 11 | 1522.13 | 1.46% | 1323.6000 | 1402.2926 | 40.0463 | M30 bullish FVG [1320.6000, 1323.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 62 | 2024-09-25 11:00 | 1362.4000 | 1324.3783 | 2024-09-25 12:30 | 1381.6000 | STOCH_EXIT | 3 | 1390.58 | 1.31% | 1364.2000 | 1438.4434 | 38.0217 | M30 bullish FVG [1362.0000, 1364.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 63 | 2024-09-30 19:30 | 1324.0000 | 1287.0461 | 2024-10-01 11:30 | 1328.8000 | STOCH_EXIT | 18 | 281.36 | 0.26% | 1323.4000 | 1397.9078 | 36.9539 | M30 bullish FVG [1320.8000, 1323.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 64 | 2024-10-02 15:30 | 1349.0000 | 1311.1551 | 2024-10-03 23:30 | 1344.8000 | STOCH_EXIT | 50 | -443.75 | -0.41% | 1349.2000 | 1424.6899 | 37.8449 | M30 bullish FVG [1349.0000, 1349.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 65 | 2024-10-04 18:00 | 1342.0000 | 1305.7510 | 2024-10-05 23:00 | 1342.8000 | STOCH_EXIT | 42 | -43.39 | -0.04% | 1342.2000 | 1414.4979 | 36.2490 | M30 bullish FVG [1341.6000, 1342.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 66 | 2024-10-22 16:30 | 1449.4000 | 1416.9495 | 2024-10-23 10:30 | 1470.0000 | STOCH_EXIT | 22 | 1416.38 | 1.32% | 1443.0000 | 1514.3011 | 32.4505 | M30 bullish FVG [1439.6000, 1443.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 67 | 2024-11-08 10:30 | 1473.4000 | 1432.2188 | 2024-11-08 22:30 | 1476.2000 | STOCH_EXIT | 24 | 96.74 | 0.09% | 1482.2000 | 1555.7625 | 41.1812 | M30 bullish FVG [1468.8000, 1482.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 68 | 2024-11-13 14:30 | 1434.2000 | 1391.5810 | 2024-11-15 10:00 | 1427.8000 | STOCH_EXIT | 59 | -595.16 | -0.55% | 1439.6000 | 1519.4379 | 42.6190 | M30 bullish FVG [1428.8000, 1439.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 69 | 2024-11-29 15:00 | 1424.0000 | 1381.3571 | 2024-11-30 22:00 | 1415.4000 | STOCH_EXIT | 48 | -761.50 | -0.70% | 1421.8000 | 1509.2857 | 42.6429 | M30 bullish FVG [1416.8000, 1421.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 70 | 2024-12-03 16:00 | 1445.2000 | 1403.4243 | 2024-12-05 17:30 | 1436.4000 | STOCH_EXIT | 71 | -757.82 | -0.71% | 1442.4000 | 1528.7514 | 41.7757 | M30 bullish FVG [1442.0000, 1442.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 71 | 2024-12-06 14:30 | 1429.0000 | 1389.2625 | 2024-12-07 07:00 | 1439.4000 | STOCH_EXIT | 19 | 663.47 | 0.63% | 1432.4000 | 1508.4750 | 39.7375 | M30 bullish FVG [1428.4000, 1432.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 72 | 2024-12-10 10:30 | 1470.2000 | 1431.4355 | 2024-12-10 20:30 | 1471.2000 | STOCH_EXIT | 20 | -34.36 | -0.03% | 1465.4000 | 1547.7289 | 38.7645 | M30 bullish FVG [1462.0000, 1465.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 73 | 2024-12-13 17:00 | 1369.0000 | 1329.5737 | 2024-12-15 21:00 | 1366.2000 | STOCH_EXIT | 76 | -325.07 | -0.30% | 1374.2000 | 1447.8526 | 39.4263 | M30 bullish FVG [1363.0000, 1374.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 74 | 2024-12-18 23:00 | 1382.0000 | 1333.6970 | 2024-12-19 15:30 | 1333.6970 | SL | 19 | -3823.88 | -3.59% | 1370.4000 | 1478.6059 | 48.3030 | M30 bullish FVG [1369.4000, 1370.4000], stochastic crossed below 20 | FVG stop 1333.6970 |
| 75 | 2025-01-07 07:30 | 1404.0000 | 1365.2261 | 2025-01-07 17:30 | 1409.0000 | STOCH_EXIT | 20 | 262.33 | 0.26% | 1404.0000 | 1481.5479 | 38.7739 | M30 bullish FVG [1403.6000, 1404.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 76 | 2025-01-09 14:00 | 1436.8000 | 1398.2242 | 2025-01-09 17:00 | 1452.0000 | STOCH_EXIT | 6 | 990.40 | 0.96% | 1436.2000 | 1513.9516 | 38.5758 | M30 bullish FVG [1436.0000, 1436.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 77 | 2025-01-29 10:30 | 1702.8000 | 1657.3643 | 2025-01-30 06:30 | 1703.6000 | STOCH_EXIT | 27 | -55.10 | -0.05% | 1695.0000 | 1793.6714 | 45.4357 | M30 bullish FVG [1688.2000, 1695.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 78 | 2025-01-31 11:30 | 1724.6000 | 1680.2131 | 2025-02-01 07:30 | 1721.0000 | STOCH_EXIT | 35 | -319.37 | -0.31% | 1715.2000 | 1813.3738 | 44.3869 | M30 bullish FVG [1712.0000, 1715.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 79 | 2025-02-10 21:00 | 1797.0000 | 1752.7617 | 2025-02-11 07:30 | 1807.2000 | STOCH_EXIT | 8 | 487.08 | 0.47% | 1800.0000 | 1885.4766 | 44.2383 | M30 bullish FVG [1795.0000, 1800.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 80 | 2025-02-13 17:00 | 1907.2000 | 1861.9320 | 2025-02-14 17:00 | 1861.9320 | SL | 28 | -2546.24 | -2.47% | 1875.2000 | 1997.7361 | 45.2680 | M30 bullish FVG [1870.0000, 1875.2000], stochastic crossed below 20 | FVG stop 1861.9320 |
| 81 | 2025-02-26 14:00 | 1938.6000 | 1882.0767 | 2025-02-28 14:00 | 1927.6000 | STOCH_EXIT | 70 | -672.52 | -0.67% | 1938.8000 | 2051.6466 | 56.5233 | M30 bullish FVG [1938.2000, 1938.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 82 | 2025-03-12 09:00 | 1893.6000 | 1836.6783 | 2025-03-12 18:00 | 1906.2000 | STOCH_EXIT | 18 | 567.11 | 0.56% | 1874.6000 | 2007.4434 | 56.9217 | M30 bullish FVG [1871.0000, 1874.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 83 | 2025-03-13 08:00 | 1892.8000 | 1838.1013 | 2025-03-13 19:00 | 1904.0000 | STOCH_EXIT | 22 | 492.98 | 0.49% | 1892.2000 | 2002.1975 | 54.6987 | M30 bullish FVG [1892.0000, 1892.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 84 | 2025-04-03 09:30 | 1796.8000 | 1734.1727 | 2025-04-03 14:30 | 1734.1727 | SL | 10 | -3670.39 | -3.58% | 1797.0000 | 1922.0546 | 62.6273 | M30 bullish FVG [1793.0000, 1797.0000], stochastic crossed below 20 | FVG stop 1734.1727 |
| 85 | 2025-04-08 20:00 | 1751.4000 | 1679.2125 | 2025-04-09 09:30 | 1814.8000 | STOCH_EXIT | 14 | 3450.55 | 3.52% | 1745.2000 | 1895.7749 | 72.1875 | M30 bullish FVG [1736.0000, 1745.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 86 | 2025-04-09 18:00 | 1791.6000 | 1717.3688 | 2025-04-09 20:00 | 1838.6000 | STOCH_EXIT | 4 | 2575.54 | 2.52% | 1799.8000 | 1940.0624 | 74.2312 | M30 bullish FVG [1787.6000, 1799.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 87 | 2025-04-10 12:00 | 1806.6000 | 1731.3996 | 2025-04-10 20:00 | 1832.2000 | STOCH_EXIT | 16 | 1379.27 | 1.32% | 1797.8000 | 1957.0008 | 75.2004 | M30 bullish FVG [1793.0000, 1797.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 88 | 2025-04-17 10:30 | 1929.0000 | 1861.6659 | 2025-04-18 09:00 | 1924.2000 | STOCH_EXIT | 32 | -369.96 | -0.35% | 1929.2000 | 2063.6682 | 67.3341 | M30 bullish FVG [1928.8000, 1929.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 89 | 2025-04-24 21:00 | 1895.0000 | 1830.6898 | 2025-04-25 18:00 | 1830.6898 | SL | 29 | -3639.52 | -3.49% | 1892.2000 | 2023.6204 | 64.3102 | M30 bullish FVG [1890.0000, 1892.2000], stochastic crossed below 20 | FVG stop 1830.6898 |
| 90 | 2025-05-01 09:30 | 1746.8000 | 1683.1398 | 2025-05-02 06:30 | 1751.4000 | STOCH_EXIT | 29 | 165.35 | 0.16% | 1746.0000 | 1874.1203 | 63.6602 | M30 bullish FVG [1745.8000, 1746.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 91 | 2025-05-08 17:00 | 1750.4000 | 1686.8131 | 2025-05-09 03:00 | 1758.6000 | STOCH_EXIT | 16 | 373.84 | 0.37% | 1751.8000 | 1877.5738 | 63.5869 | M30 bullish FVG [1749.8000, 1751.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 92 | 2025-05-14 08:00 | 1749.6000 | 1688.8823 | 2025-05-15 10:00 | 1688.8823 | SL | 39 | -3621.34 | -3.57% | 1753.8000 | 1871.0353 | 60.7177 | M30 bullish FVG [1749.4000, 1753.8000], stochastic crossed below 20 | FVG stop 1688.8823 |
| 93 | 2025-05-29 10:00 | 1676.0000 | 1617.1277 | 2025-05-29 13:00 | 1693.2000 | STOCH_EXIT | 6 | 915.41 | 0.93% | 1673.4000 | 1793.7446 | 58.8723 | M30 bullish FVG [1669.0000, 1673.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 94 | 2025-06-03 18:30 | 1730.0000 | 1673.3242 | 2025-06-04 12:00 | 1762.8000 | STOCH_EXIT | 22 | 1770.06 | 1.79% | 1736.2000 | 1843.3517 | 56.6758 | M30 bullish FVG [1727.6000, 1736.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 95 | 2025-06-04 23:00 | 1758.6000 | 1703.8296 | 2025-06-05 12:00 | 1777.6000 | STOCH_EXIT | 13 | 982.22 | 0.98% | 1761.6000 | 1868.1408 | 54.7704 | M30 bullish FVG [1756.0000, 1761.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 96 | 2025-06-06 22:30 | 1751.2000 | 1694.9184 | 2025-06-07 06:30 | 1756.0000 | STOCH_EXIT | 12 | 176.69 | 0.17% | 1750.0000 | 1863.7633 | 56.2816 | M30 bullish FVG [1747.6000, 1750.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 97 | 2025-06-10 12:00 | 1701.8000 | 1641.1255 | 2025-06-10 18:00 | 1734.2000 | STOCH_EXIT | 12 | 1840.92 | 1.80% | 1707.8000 | 1823.1489 | 60.6745 | M30 bullish FVG [1701.2000, 1707.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 98 | 2025-06-17 15:00 | 1732.8000 | 1673.0872 | 2025-06-18 06:30 | 1745.0000 | STOCH_EXIT | 18 | 627.67 | 0.60% | 1748.6000 | 1852.2256 | 59.7128 | M30 bullish FVG [1722.0000, 1748.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 99 | 2025-06-27 07:00 | 1768.0000 | 1716.9784 | 2025-06-27 15:30 | 1789.8000 | STOCH_EXIT | 17 | 1181.24 | 1.13% | 1768.2000 | 1870.0431 | 51.0216 | M30 bullish FVG [1766.8000, 1768.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 100 | 2025-07-11 09:00 | 1854.8000 | 1802.5191 | 2025-07-11 18:00 | 1802.5191 | SL | 18 | -3084.24 | -2.92% | 1859.2000 | 1959.3618 | 52.2809 | M30 bullish FVG [1853.0000, 1859.2000], stochastic crossed below 20 | FVG stop 1802.5191 |
| 101 | 2025-07-11 23:00 | 1789.4000 | 1737.1191 | 2025-07-12 06:00 | 1803.8000 | STOCH_EXIT | 10 | 718.39 | 0.70% | 1790.2000 | 1893.9618 | 52.2809 | M30 bullish FVG [1787.6000, 1790.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 102 | 2025-07-17 09:00 | 1883.2000 | 1826.0180 | 2025-07-17 17:00 | 1896.4000 | STOCH_EXIT | 16 | 622.06 | 0.60% | 1873.8000 | 1997.5639 | 57.1820 | M30 bullish FVG [1872.0000, 1873.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 103 | 2025-07-22 11:00 | 1933.4000 | 1873.7996 | 2025-07-22 16:00 | 1955.4000 | STOCH_EXIT | 10 | 1083.00 | 1.04% | 1922.4000 | 2052.6007 | 59.6004 | M30 bullish FVG [1919.8000, 1922.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 104 | 2025-07-30 15:30 | 1976.6000 | 1918.8142 | 2025-07-31 10:00 | 1964.4000 | STOCH_EXIT | 24 | -751.04 | -0.72% | 1976.8000 | 2092.1716 | 57.7858 | M30 bullish FVG [1976.0000, 1976.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 105 | 2025-08-01 11:30 | 1963.4000 | 1904.9531 | 2025-08-02 05:00 | 1970.8000 | STOCH_EXIT | 31 | 287.94 | 0.28% | 1959.8000 | 2080.2939 | 58.4469 | M30 bullish FVG [1958.2000, 1959.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 106 | 2025-08-05 14:00 | 2027.0000 | 1968.3330 | 2025-08-05 16:30 | 2046.2000 | STOCH_EXIT | 5 | 892.50 | 0.85% | 2026.8000 | 2144.3340 | 58.6670 | M30 bullish FVG [2025.0000, 2026.8000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 107 | 2025-08-06 13:00 | 2040.0000 | 1982.1092 | 2025-08-06 21:30 | 2056.0000 | STOCH_EXIT | 17 | 725.50 | 0.68% | 2031.4000 | 2155.7816 | 57.8908 | M30 bullish FVG [2025.0000, 2031.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 108 | 2025-08-11 10:30 | 2164.6000 | 2107.3082 | 2025-08-11 16:30 | 2107.3082 | SL | 12 | -2911.96 | -2.74% | 2207.2000 | 2279.1836 | 57.2918 | M30 bullish FVG [2150.0000, 2207.2000], stochastic crossed below 20 | FVG stop 2107.3082 |
| 109 | 2025-08-19 18:30 | 2132.2000 | 2072.9518 | 2025-08-20 11:30 | 2153.6000 | STOCH_EXIT | 21 | 924.34 | 0.90% | 2104.0000 | 2250.6963 | 59.2482 | M30 bullish FVG [2102.6000, 2104.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 110 | 2025-08-21 19:00 | 2144.0000 | 2086.8850 | 2025-08-22 08:30 | 2162.0000 | STOCH_EXIT | 14 | 776.50 | 0.74% | 2141.4000 | 2258.2300 | 57.1150 | M30 bullish FVG [2141.2000, 2141.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 111 | 2025-09-02 09:30 | 2175.6000 | 2118.8170 | 2025-09-02 15:00 | 2194.0000 | STOCH_EXIT | 11 | 778.33 | 0.74% | 2168.4000 | 2289.1660 | 56.7830 | M30 bullish FVG [2166.4000, 2168.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 112 | 2025-09-23 22:30 | 2325.2000 | 2274.2019 | 2025-09-24 14:30 | 2328.6000 | STOCH_EXIT | 19 | 48.29 | 0.05% | 2319.2000 | 2427.1961 | 50.9981 | M30 bullish FVG [2313.4000, 2319.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 113 | 2025-09-25 07:30 | 2307.0000 | 2256.7282 | 2025-09-25 09:00 | 2320.0000 | STOCH_EXIT | 3 | 491.58 | 0.46% | 2304.6000 | 2407.5436 | 50.2718 | M30 bullish FVG [2304.2000, 2304.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 114 | 2025-09-25 18:30 | 2308.2000 | 2257.9282 | 2025-09-26 09:00 | 2319.4000 | STOCH_EXIT | 16 | 408.77 | 0.38% | 2304.6000 | 2408.7436 | 50.2718 | M30 bullish FVG [2304.2000, 2304.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 115 | 2025-09-30 12:00 | 2372.2000 | 2322.3733 | 2025-10-01 07:00 | 2384.0000 | STOCH_EXIT | 25 | 423.99 | 0.40% | 2376.2000 | 2471.8533 | 49.8267 | M30 bullish FVG [2365.0000, 2376.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 116 | 2025-10-01 17:30 | 2356.2000 | 2302.8610 | 2025-10-02 09:00 | 2374.0000 | STOCH_EXIT | 18 | 694.57 | 0.65% | 2357.2000 | 2462.8781 | 53.3390 | M30 bullish FVG [2355.2000, 2357.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 117 | 2025-10-07 12:30 | 2370.0000 | 2314.0906 | 2025-10-07 21:00 | 2383.6000 | STOCH_EXIT | 17 | 505.04 | 0.47% | 2364.4000 | 2481.8189 | 55.9094 | M30 bullish FVG [2360.0000, 2364.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 118 | 2025-10-17 12:30 | 2253.4000 | 2171.6963 | 2025-10-18 09:00 | 2210.6000 | STOCH_EXIT | 37 | -2161.54 | -2.00% | 2245.0000 | 2416.8074 | 81.7037 | M30 bullish FVG [2243.0000, 2245.0000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 119 | 2025-10-21 08:30 | 2145.0000 | 2055.2932 | 2025-10-21 17:00 | 2055.2932 | SL | 17 | -4590.35 | -4.28% | 2160.0000 | 2324.4135 | 89.7068 | M30 bullish FVG [2144.0000, 2160.0000], stochastic crossed below 20 | FVG stop 2055.2932 |
| 120 | 2025-10-24 15:00 | 2045.0000 | 1953.3294 | 2025-10-26 03:30 | 2050.6000 | STOCH_EXIT | 65 | 177.61 | 0.17% | 2042.4000 | 2228.3412 | 91.6706 | M30 bullish FVG [2042.2000, 2042.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 121 | 2025-10-30 07:00 | 2039.8000 | 1952.6648 | 2025-10-30 09:30 | 2070.2000 | STOCH_EXIT | 5 | 1417.25 | 1.39% | 2044.6000 | 2214.0704 | 87.1352 | M30 bullish FVG [2036.4000, 2044.6000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 122 | 2025-10-31 14:30 | 2075.6000 | 1990.5745 | 2025-11-01 23:30 | 2062.0000 | STOCH_EXIT | 53 | -783.44 | -0.75% | 2086.2000 | 2245.6511 | 85.0255 | M30 bullish FVG [2073.0000, 2086.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 123 | 2025-11-03 20:30 | 2091.4000 | 2006.8191 | 2025-11-05 06:30 | 2094.6000 | STOCH_EXIT | 51 | 54.24 | 0.05% | 2086.2000 | 2260.5617 | 84.5809 | M30 bullish FVG [2075.0000, 2086.2000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |
| 124 | 2025-11-13 22:30 | 2176.2000 | 2107.3165 | 2025-11-15 02:30 | 2156.0000 | STOCH_EXIT | 39 | -1051.21 | -1.03% | 2174.4000 | 2313.9670 | 68.8835 | M30 bullish FVG [2168.2000, 2174.4000], stochastic crossed below 20 | FVG stochastic %K(21) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 97010.75
- Макс: 110299.48
- Финал: 102532.94

Полная кривая — в `*_equity.csv`.
