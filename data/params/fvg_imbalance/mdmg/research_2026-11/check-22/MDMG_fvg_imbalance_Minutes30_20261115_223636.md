# Бэктест MDMG (Minutes30)

- Период: 2023-11-15 22:36 — 2025-11-15 22:36
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.25 |
| MaxWaitBars | 30 |
| StochPeriod | 10 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 2 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 44 |
| Выигрышных / проигрышных | 32 / 12 |
| Win rate | 72.73% |
| Gross profit / loss | 27939.54 / 17640.92 |
| Profit factor | 1.584 |
| Чистый PnL | 10298.62 (10.30%) |
| Стартовый / финальный капитал | 100000.00 / 110298.62 |
| Макс. просадка | 6291.00 (6.24%) |
| Средняя прибыль / убыток | 873.11 / 1470.08 |
| Expectancy | 234.06 |
| Лучшая / худшая сделка | 3248.53 / -3895.35 |
| Exposure | 2.87% |
| CAGR | 5.02% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-12-06 10:30 | 732.0000 | 705.6074 | 2023-12-06 16:30 | 728.9000 | STOCH_EXIT | 12 | -520.94 | -0.52% | 744.0000 | 784.7852 | 26.3926 | M30 bullish FVG [737.2000, 744.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 2 | 2023-12-22 17:30 | 806.8000 | 775.7253 | 2023-12-22 18:30 | 810.0000 | STOCH_EXIT | 2 | 294.17 | 0.30% | 805.5000 | 868.9493 | 31.0747 | M30 bullish FVG [796.5000, 805.5000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 3 | 2023-12-25 15:30 | 820.0000 | 788.5092 | 2023-12-26 16:30 | 807.4000 | STOCH_EXIT | 21 | -1623.06 | -1.63% | 820.4000 | 882.9815 | 31.4908 | M30 bullish FVG [809.0000, 820.4000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 4 | 2024-02-06 18:00 | 986.6000 | 958.9112 | 2024-02-08 10:00 | 958.9112 | SL | 22 | -2837.50 | -2.90% | 974.5000 | 1041.9776 | 27.6888 | M30 bullish FVG [946.3000, 974.5000], stochastic crossed below 20 | FVG stop 958.9112 |
| 5 | 2024-02-12 10:30 | 956.2000 | 922.1076 | 2024-02-12 16:30 | 959.5000 | STOCH_EXIT | 12 | 231.87 | 0.24% | 962.3000 | 1024.3849 | 34.0924 | M30 bullish FVG [948.8000, 962.3000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 6 | 2024-02-16 12:00 | 993.9000 | 959.9828 | 2024-02-16 17:30 | 996.3000 | STOCH_EXIT | 11 | 134.87 | 0.14% | 990.1000 | 1061.7344 | 33.9172 | M30 bullish FVG [981.8000, 990.1000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 7 | 2024-02-28 16:00 | 964.3000 | 929.1186 | 2024-02-29 11:30 | 971.1000 | STOCH_EXIT | 10 | 577.40 | 0.60% | 991.8000 | 1034.6628 | 35.1814 | M30 bullish FVG [978.9000, 991.8000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 8 | 2024-03-07 13:00 | 1015.7000 | 978.9098 | 2024-03-07 14:00 | 1036.3000 | STOCH_EXIT | 2 | 1839.96 | 1.93% | 1026.8000 | 1089.2805 | 36.7902 | M30 bullish FVG [1007.3000, 1026.8000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 9 | 2024-03-11 10:00 | 1027.0000 | 990.5734 | 2024-03-14 10:00 | 990.5734 | SL | 57 | -3556.37 | -3.64% | 1030.9000 | 1099.8533 | 36.4266 | M30 bullish FVG [1016.6000, 1030.9000], stochastic crossed below 20 | FVG stop 990.5734 |
| 10 | 2024-04-02 11:00 | 1018.5000 | 987.8664 | 2024-04-02 14:00 | 1029.0000 | STOCH_EXIT | 6 | 871.82 | 0.93% | 1020.2000 | 1079.7672 | 30.6336 | M30 bullish FVG [1009.0000, 1020.2000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 11 | 2024-04-03 15:00 | 1049.6000 | 1019.5474 | 2024-04-03 18:00 | 1057.1000 | STOCH_EXIT | 6 | 580.20 | 0.61% | 1070.0000 | 1109.7052 | 30.0526 | M30 bullish FVG [1060.0000, 1070.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 12 | 2024-05-03 13:00 | 1136.9000 | 1104.5670 | 2024-05-04 15:00 | 1154.8000 | STOCH_EXIT | 22 | 1407.35 | 1.47% | 1133.9000 | 1201.5659 | 32.3330 | M30 bullish FVG [1115.0000, 1133.9000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 13 | 2024-05-17 12:00 | 1055.8000 | 1013.8516 | 2024-05-17 18:00 | 1088.1000 | STOCH_EXIT | 12 | 2872.98 | 2.96% | 1074.0000 | 1139.6968 | 41.9484 | M30 bullish FVG [1056.7000, 1074.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 14 | 2024-06-17 15:30 | 1203.0000 | 1157.2482 | 2024-06-17 16:00 | 1157.2482 | SL | 1 | -3895.35 | -3.90% | 1271.7000 | 1294.5036 | 45.7518 | M30 bullish FVG [1179.2000, 1271.7000], stochastic crossed below 20 | FVG stop 1157.2482 |
| 15 | 2024-06-18 14:00 | 1003.6000 | 936.0876 | 2024-06-19 09:30 | 1027.0000 | STOCH_EXIT | 10 | 2126.55 | 2.23% | 1021.7000 | 1138.6248 | 67.5124 | M30 bullish FVG [1000.0000, 1021.7000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 16 | 2024-07-31 15:30 | 871.7000 | 838.0355 | 2024-08-01 09:30 | 880.0000 | STOCH_EXIT | 7 | 831.50 | 0.85% | 840.0000 | 939.0290 | 33.6645 | M30 bullish FVG [830.6000, 840.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 17 | 2024-09-26 16:30 | 880.0000 | 855.8009 | 2024-09-26 20:00 | 883.6000 | STOCH_EXIT | 7 | 304.44 | 0.31% | 880.9000 | 928.3983 | 24.1991 | M30 bullish FVG [873.5000, 880.9000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 18 | 2024-09-30 16:30 | 901.5000 | 878.0533 | 2024-09-30 19:00 | 908.9000 | STOCH_EXIT | 5 | 714.43 | 0.72% | 907.0000 | 948.3934 | 23.4467 | M30 bullish FVG [896.9000, 907.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 19 | 2024-10-02 18:30 | 886.8000 | 861.7745 | 2024-10-03 15:30 | 861.7745 | SL | 28 | -2926.67 | -2.92% | 905.0000 | 936.8509 | 25.0255 | M30 bullish FVG [895.8000, 905.0000], stochastic crossed below 20 | FVG stop 861.7745 |
| 20 | 2024-10-30 14:30 | 862.8000 | 838.9097 | 2024-10-31 08:30 | 875.1000 | STOCH_EXIT | 22 | 1280.28 | 1.32% | 869.0000 | 910.5807 | 23.8903 | M30 bullish FVG [843.2000, 869.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 21 | 2024-11-29 17:00 | 854.0000 | 827.6206 | 2024-11-29 19:00 | 860.4000 | STOCH_EXIT | 4 | 637.42 | 0.65% | 860.0000 | 906.7589 | 26.3794 | M30 bullish FVG [853.3000, 860.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 22 | 2024-12-26 18:00 | 853.0000 | 834.9361 | 2024-12-27 17:30 | 853.6000 | STOCH_EXIT | 33 | -29.38 | -0.03% | 865.0000 | 889.1278 | 18.0639 | M30 bullish FVG [857.7000, 865.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 23 | 2025-01-09 19:00 | 929.0000 | 903.6335 | 2025-01-09 19:30 | 931.0000 | STOCH_EXIT | 1 | 113.42 | 0.12% | 930.9000 | 979.7330 | 25.3665 | M30 bullish FVG [923.6000, 930.9000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 24 | 2025-01-22 16:00 | 948.0000 | 924.2637 | 2025-01-23 11:00 | 980.2000 | STOCH_EXIT | 24 | 3248.53 | 3.29% | 951.6000 | 995.4726 | 23.7363 | M30 bullish FVG [944.6000, 951.6000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 25 | 2025-01-23 18:00 | 964.0000 | 939.7520 | 2025-01-23 22:00 | 969.0000 | STOCH_EXIT | 8 | 427.55 | 0.42% | 970.5000 | 1012.4960 | 24.2480 | M30 bullish FVG [963.9000, 970.5000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 26 | 2025-01-27 18:00 | 993.8000 | 968.4760 | 2025-01-28 08:30 | 1003.1000 | STOCH_EXIT | 15 | 855.06 | 0.83% | 1008.6000 | 1044.4480 | 25.3240 | M30 bullish FVG [1001.9000, 1008.6000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 27 | 2025-02-19 23:30 | 1024.6000 | 994.4633 | 2025-02-20 12:00 | 1036.5000 | STOCH_EXIT | 11 | 1097.81 | 1.06% | 1021.3000 | 1084.8735 | 30.1367 | M30 bullish FVG [1013.0000, 1021.3000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 28 | 2025-03-20 23:30 | 1085.0000 | 1057.4579 | 2025-03-21 10:30 | 1094.0000 | STOCH_EXIT | 9 | 759.41 | 0.73% | 1085.2000 | 1140.0842 | 27.5421 | M30 bullish FVG [1070.0000, 1085.2000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 29 | 2025-03-21 18:30 | 1114.0000 | 1082.7680 | 2025-03-22 03:00 | 1138.0000 | STOCH_EXIT | 13 | 2150.16 | 2.05% | 1134.1000 | 1176.4639 | 31.2320 | M30 bullish FVG [1109.0000, 1134.1000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 30 | 2025-03-28 16:00 | 1060.7000 | 1021.0461 | 2025-03-28 23:30 | 1060.3000 | STOCH_EXIT | 15 | -147.51 | -0.14% | 1058.0000 | 1140.0079 | 39.6539 | M30 bullish FVG [1041.9000, 1058.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 31 | 2025-04-10 17:00 | 997.3000 | 958.5020 | 2025-04-10 21:00 | 1003.7000 | STOCH_EXIT | 8 | 583.15 | 0.54% | 988.4000 | 1074.8960 | 38.7980 | M30 bullish FVG [977.0000, 988.4000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 32 | 2025-05-16 07:00 | 985.2000 | 964.4131 | 2025-05-16 08:00 | 994.8000 | STOCH_EXIT | 2 | 938.49 | 0.87% | 986.9000 | 1026.7737 | 20.7869 | M30 bullish FVG [980.8000, 986.9000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 33 | 2025-05-27 11:00 | 966.3000 | 944.0249 | 2025-05-27 16:30 | 973.3000 | STOCH_EXIT | 11 | 681.41 | 0.62% | 972.9000 | 1010.8502 | 22.2751 | M30 bullish FVG [961.9000, 972.9000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 34 | 2025-05-28 12:30 | 982.7000 | 960.3874 | 2025-05-28 19:30 | 982.6000 | STOCH_EXIT | 14 | -120.17 | -0.11% | 972.2000 | 1027.3252 | 22.3126 | M30 bullish FVG [965.2000, 972.2000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 35 | 2025-06-25 09:00 | 998.8000 | 983.7438 | 2025-06-25 19:30 | 1000.8000 | STOCH_EXIT | 21 | 109.02 | 0.10% | 1003.0000 | 1028.9125 | 15.0562 | M30 bullish FVG [997.5000, 1003.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 36 | 2025-06-26 21:30 | 1005.6000 | 990.4978 | 2025-06-27 09:00 | 1009.0000 | STOCH_EXIT | 9 | 260.80 | 0.24% | 1011.0000 | 1035.8044 | 15.1022 | M30 bullish FVG [1005.0000, 1011.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 37 | 2025-07-03 21:00 | 1062.6000 | 1046.5416 | 2025-07-04 07:00 | 1046.5416 | SL | 7 | -1762.64 | -1.61% | 1072.0000 | 1094.7168 | 16.0584 | M30 bullish FVG [1062.0000, 1072.0000], stochastic crossed below 20 | FVG stop 1046.5416 |
| 38 | 2025-07-07 21:30 | 1049.0000 | 1028.5241 | 2025-07-08 15:30 | 1058.9000 | STOCH_EXIT | 23 | 911.14 | 0.84% | 1053.0000 | 1089.9517 | 20.4759 | M30 bullish FVG [1046.2000, 1053.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 39 | 2025-07-17 16:30 | 1107.0000 | 1080.3205 | 2025-07-18 10:30 | 1114.4000 | STOCH_EXIT | 22 | 616.35 | 0.57% | 1105.5000 | 1160.3590 | 26.6795 | M30 bullish FVG [1098.0000, 1105.5000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 40 | 2025-07-23 07:30 | 1133.2000 | 1104.7040 | 2025-07-23 10:00 | 1137.5000 | STOCH_EXIT | 5 | 306.97 | 0.28% | 1113.3000 | 1190.1921 | 28.4960 | M30 bullish FVG [1105.0000, 1113.3000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 41 | 2025-07-28 17:30 | 1149.6000 | 1121.6525 | 2025-07-29 07:00 | 1149.0000 | STOCH_EXIT | 13 | -166.18 | -0.15% | 1158.8000 | 1205.4950 | 27.9475 | M30 bullish FVG [1150.4000, 1158.8000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 42 | 2025-08-15 19:30 | 1260.0000 | 1224.4480 | 2025-08-15 23:30 | 1261.4000 | STOCH_EXIT | 8 | 12.12 | 0.01% | 1253.8000 | 1331.1040 | 35.5520 | M30 bullish FVG [1242.5000, 1253.8000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 43 | 2025-09-22 11:30 | 1205.7000 | 1175.0184 | 2025-09-22 17:30 | 1206.3000 | STOCH_EXIT | 12 | -55.15 | -0.05% | 1222.0000 | 1267.0633 | 30.6816 | M30 bullish FVG [1214.0000, 1222.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 44 | 2025-11-05 17:00 | 1281.8000 | 1245.2952 | 2025-11-05 22:00 | 1285.0000 | STOCH_EXIT | 10 | 162.91 | 0.15% | 1297.1000 | 1354.8096 | 36.5048 | M30 bullish FVG [1284.9000, 1297.1000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 94473.99
- Макс: 110556.31
- Финал: 110298.62

Полная кривая — в `*_equity.csv`.
