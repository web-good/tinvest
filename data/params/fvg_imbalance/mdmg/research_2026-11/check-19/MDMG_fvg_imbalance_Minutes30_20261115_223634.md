# Бэктест MDMG (Minutes30)

- Период: 2023-11-15 22:36 — 2025-11-15 22:36
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.1 |
| MaxWaitBars | 30 |
| StochPeriod | 10 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 2 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 1 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 87 |
| Выигрышных / проигрышных | 67 / 20 |
| Win rate | 77.01% |
| Gross profit / loss | 50493.18 / 29511.10 |
| Profit factor | 1.711 |
| Чистый PnL | 20982.08 (20.98%) |
| Стартовый / финальный капитал | 100000.00 / 120982.08 |
| Макс. просадка | 10024.34 (9.45%) |
| Средняя прибыль / убыток | 753.63 / 1475.55 |
| Expectancy | 241.17 |
| Лучшая / худшая сделка | 3498.42 / -3942.28 |
| Exposure | 5.46% |
| CAGR | 9.98% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-11-20 18:30 | 796.4000 | 770.7541 | 2023-11-21 17:00 | 799.1000 | STOCH_EXIT | 16 | 237.78 | 0.24% | 797.1000 | 847.6918 | 25.6459 | M30 bullish FVG [793.3000, 797.1000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 2 | 2023-11-23 10:30 | 815.1000 | 789.2775 | 2023-11-24 10:00 | 816.0000 | STOCH_EXIT | 18 | 10.30 | 0.01% | 814.0000 | 866.7450 | 25.8225 | M30 bullish FVG [808.3000, 814.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 3 | 2023-12-06 10:30 | 732.0000 | 705.6074 | 2023-12-06 16:30 | 728.9000 | STOCH_EXIT | 12 | -520.94 | -0.52% | 730.5000 | 784.7852 | 26.3926 | M30 bullish FVG [725.2000, 730.5000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 4 | 2023-12-11 10:30 | 712.3000 | 684.5008 | 2023-12-12 10:00 | 712.0000 | STOCH_EXIT | 18 | -140.69 | -0.14% | 705.0000 | 767.8984 | 27.7992 | M30 bullish FVG [701.8000, 705.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 5 | 2023-12-13 17:00 | 721.9000 | 694.0573 | 2023-12-15 14:00 | 730.8000 | STOCH_EXIT | 32 | 1119.79 | 1.13% | 716.9000 | 777.5854 | 27.8427 | M30 bullish FVG [713.5000, 716.9000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 6 | 2023-12-20 18:30 | 796.6000 | 764.1702 | 2023-12-21 09:30 | 800.9000 | STOCH_EXIT | 1 | 441.16 | 0.44% | 797.6000 | 861.4596 | 32.4298 | M30 bullish FVG [794.0000, 797.6000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 7 | 2023-12-22 17:30 | 806.8000 | 775.7253 | 2023-12-22 18:30 | 810.0000 | STOCH_EXIT | 2 | 298.95 | 0.30% | 805.5000 | 868.9493 | 31.0747 | M30 bullish FVG [796.5000, 805.5000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 8 | 2023-12-25 15:30 | 820.0000 | 788.5092 | 2023-12-26 16:30 | 807.4000 | STOCH_EXIT | 21 | -1649.89 | -1.63% | 820.4000 | 882.9815 | 31.4908 | M30 bullish FVG [809.0000, 820.4000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 9 | 2024-01-09 18:00 | 824.6000 | 796.9272 | 2024-01-10 09:30 | 835.0000 | STOCH_EXIT | 2 | 1148.42 | 1.16% | 823.1000 | 879.9456 | 27.6728 | M30 bullish FVG [816.7000, 823.1000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 10 | 2024-01-22 13:00 | 800.0000 | 774.9856 | 2024-01-22 14:00 | 807.7000 | STOCH_EXIT | 2 | 868.91 | 0.86% | 804.9000 | 850.0289 | 25.0144 | M30 bullish FVG [800.0000, 804.9000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 11 | 2024-01-23 16:00 | 809.8000 | 785.2152 | 2024-01-25 10:00 | 818.9000 | STOCH_EXIT | 26 | 1035.71 | 1.02% | 814.0000 | 858.9697 | 24.5848 | M30 bullish FVG [809.0000, 814.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 12 | 2024-02-06 18:00 | 986.6000 | 958.9112 | 2024-02-08 10:00 | 958.9112 | SL | 22 | -2980.80 | -2.90% | 974.5000 | 1041.9776 | 27.6888 | M30 bullish FVG [946.3000, 974.5000], stochastic crossed below 20 | FVG stop 958.9112 |
| 13 | 2024-02-12 10:30 | 956.2000 | 922.1076 | 2024-02-12 16:30 | 959.5000 | STOCH_EXIT | 12 | 243.58 | 0.24% | 962.3000 | 1024.3849 | 34.0924 | M30 bullish FVG [948.8000, 962.3000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 14 | 2024-02-16 12:00 | 993.9000 | 959.9828 | 2024-02-16 17:30 | 996.3000 | STOCH_EXIT | 11 | 140.49 | 0.14% | 990.1000 | 1061.7344 | 33.9172 | M30 bullish FVG [981.8000, 990.1000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 15 | 2024-02-26 17:00 | 957.0000 | 920.7222 | 2024-02-26 18:00 | 962.9000 | STOCH_EXIT | 2 | 513.77 | 0.52% | 945.0000 | 1029.5557 | 36.2778 | M30 bullish FVG [941.0000, 945.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 16 | 2024-02-28 16:00 | 964.3000 | 929.1186 | 2024-02-29 11:30 | 971.1000 | STOCH_EXIT | 10 | 606.56 | 0.60% | 976.1000 | 1034.6628 | 35.1814 | M30 bullish FVG [958.7000, 976.1000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 17 | 2024-03-07 13:00 | 1015.7000 | 978.9098 | 2024-03-07 14:00 | 1036.3000 | STOCH_EXIT | 2 | 1937.83 | 1.93% | 1026.8000 | 1089.2805 | 36.7902 | M30 bullish FVG [1007.3000, 1026.8000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 18 | 2024-03-11 10:00 | 1027.0000 | 990.5734 | 2024-03-14 10:00 | 990.5734 | SL | 57 | -3743.54 | -3.64% | 1030.9000 | 1099.8533 | 36.4266 | M30 bullish FVG [1016.6000, 1030.9000], stochastic crossed below 20 | FVG stop 990.5734 |
| 19 | 2024-03-21 17:00 | 998.9000 | 965.2632 | 2024-03-22 11:00 | 1019.0000 | STOCH_EXIT | 7 | 1890.01 | 1.91% | 999.0000 | 1066.1737 | 33.6368 | M30 bullish FVG [995.3000, 999.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 20 | 2024-04-02 11:00 | 1018.5000 | 987.8664 | 2024-04-02 14:00 | 1029.0000 | STOCH_EXIT | 6 | 938.15 | 0.93% | 1020.2000 | 1079.7672 | 30.6336 | M30 bullish FVG [1009.0000, 1020.2000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 21 | 2024-05-03 13:00 | 1136.9000 | 1104.5670 | 2024-05-04 15:00 | 1154.8000 | STOCH_EXIT | 22 | 1507.87 | 1.47% | 1133.9000 | 1201.5659 | 32.3330 | M30 bullish FVG [1115.0000, 1133.9000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 22 | 2024-05-09 14:00 | 1131.9000 | 1099.6286 | 2024-05-09 18:30 | 1139.4000 | STOCH_EXIT | 9 | 579.16 | 0.56% | 1122.0000 | 1196.4428 | 32.2714 | M30 bullish FVG [1116.6000, 1122.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 23 | 2024-05-15 12:30 | 1180.5000 | 1146.2545 | 2024-05-15 15:00 | 1146.2545 | SL | 5 | -3115.98 | -3.00% | 1172.8000 | 1248.9910 | 34.2455 | M30 bullish FVG [1165.1000, 1172.8000], stochastic crossed below 20 | FVG stop 1146.2545 |
| 24 | 2024-06-17 15:30 | 1203.0000 | 1157.2482 | 2024-06-17 16:00 | 1157.2482 | SL | 1 | -3942.28 | -3.90% | 1271.7000 | 1294.5036 | 45.7518 | M30 bullish FVG [1179.2000, 1271.7000], stochastic crossed below 20 | FVG stop 1157.2482 |
| 25 | 2024-06-18 14:00 | 1003.6000 | 936.0876 | 2024-06-19 09:30 | 1027.0000 | STOCH_EXIT | 10 | 2171.32 | 2.23% | 1021.7000 | 1138.6248 | 67.5124 | M30 bullish FVG [1000.0000, 1021.7000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 26 | 2024-07-02 16:00 | 829.4000 | 775.6755 | 2024-07-03 15:30 | 827.9000 | STOCH_EXIT | 18 | -279.44 | -0.28% | 834.0000 | 936.8491 | 53.7245 | M30 bullish FVG [826.7000, 834.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 27 | 2024-07-22 13:00 | 841.5000 | 804.7961 | 2024-07-22 14:00 | 858.0000 | STOCH_EXIT | 2 | 1831.08 | 1.86% | 838.2000 | 914.9077 | 36.7039 | M30 bullish FVG [830.2000, 838.2000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 28 | 2024-07-31 15:30 | 871.7000 | 838.0355 | 2024-08-01 09:30 | 880.0000 | STOCH_EXIT | 7 | 853.78 | 0.85% | 850.0000 | 939.0290 | 33.6645 | M30 bullish FVG [843.0000, 850.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 29 | 2024-09-10 16:00 | 833.1000 | 805.6392 | 2024-09-11 08:00 | 837.2000 | STOCH_EXIT | 16 | 398.31 | 0.39% | 833.0000 | 888.0215 | 27.4608 | M30 bullish FVG [829.9000, 833.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 30 | 2024-09-26 16:30 | 880.0000 | 855.8009 | 2024-09-26 20:00 | 883.6000 | STOCH_EXIT | 7 | 315.31 | 0.31% | 880.9000 | 928.3983 | 24.1991 | M30 bullish FVG [873.5000, 880.9000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 31 | 2024-09-27 18:30 | 886.6000 | 862.8651 | 2024-09-28 07:30 | 890.0000 | STOCH_EXIT | 12 | 288.85 | 0.28% | 882.4000 | 934.0698 | 23.7349 | M30 bullish FVG [880.0000, 882.4000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 32 | 2024-09-30 16:30 | 901.5000 | 878.0533 | 2024-09-30 19:00 | 908.9000 | STOCH_EXIT | 5 | 740.41 | 0.72% | 907.0000 | 948.3934 | 23.4467 | M30 bullish FVG [896.9000, 907.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 33 | 2024-10-15 18:30 | 884.0000 | 861.8992 | 2024-10-15 22:00 | 889.3000 | STOCH_EXIT | 7 | 516.36 | 0.50% | 884.1000 | 928.2017 | 22.1008 | M30 bullish FVG [880.0000, 884.1000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 34 | 2024-10-25 11:00 | 850.7000 | 828.8579 | 2024-10-25 12:00 | 859.0000 | STOCH_EXIT | 2 | 908.31 | 0.87% | 851.2000 | 894.3841 | 21.8421 | M30 bullish FVG [848.0000, 851.2000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 35 | 2024-10-30 14:30 | 862.8000 | 838.9097 | 2024-10-31 08:30 | 875.1000 | STOCH_EXIT | 22 | 1383.16 | 1.32% | 869.0000 | 910.5807 | 23.8903 | M30 bullish FVG [843.2000, 869.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 36 | 2024-11-27 18:00 | 854.5000 | 829.1286 | 2024-11-27 23:00 | 869.9000 | STOCH_EXIT | 10 | 1802.69 | 1.70% | 858.0000 | 905.2428 | 25.3714 | M30 bullish FVG [853.8000, 858.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 37 | 2024-11-29 17:00 | 854.0000 | 827.6206 | 2024-11-29 19:00 | 860.4000 | STOCH_EXIT | 4 | 698.39 | 0.65% | 860.0000 | 906.7589 | 26.3794 | M30 bullish FVG [853.3000, 860.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 38 | 2024-12-11 23:30 | 842.1000 | 821.0619 | 2024-12-12 08:00 | 846.6000 | STOCH_EXIT | 3 | 471.58 | 0.43% | 844.2000 | 884.1763 | 21.0381 | M30 bullish FVG [841.0000, 844.2000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 39 | 2025-01-08 20:30 | 941.0000 | 916.0515 | 2025-01-09 10:30 | 916.0515 | SL | 14 | -3001.74 | -2.75% | 941.2000 | 990.8970 | 24.9485 | M30 bullish FVG [936.0000, 941.2000], stochastic crossed below 20 | FVG stop 916.0515 |
| 40 | 2025-01-09 19:00 | 929.0000 | 903.6335 | 2025-01-09 19:30 | 931.0000 | STOCH_EXIT | 1 | 121.98 | 0.12% | 930.9000 | 979.7330 | 25.3665 | M30 bullish FVG [923.6000, 930.9000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 41 | 2025-01-22 16:00 | 948.0000 | 924.2637 | 2025-01-23 11:00 | 980.2000 | STOCH_EXIT | 24 | 3498.42 | 3.29% | 951.6000 | 995.4726 | 23.7363 | M30 bullish FVG [944.6000, 951.6000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 42 | 2025-01-23 18:00 | 964.0000 | 939.7520 | 2025-01-23 22:00 | 969.0000 | STOCH_EXIT | 8 | 459.82 | 0.42% | 955.7000 | 1012.4960 | 24.2480 | M30 bullish FVG [950.0000, 955.7000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 43 | 2025-01-27 18:00 | 993.8000 | 968.4760 | 2025-01-28 08:30 | 1003.1000 | STOCH_EXIT | 15 | 921.47 | 0.83% | 992.0000 | 1044.4480 | 25.3240 | M30 bullish FVG [979.8000, 992.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 44 | 2025-02-19 23:30 | 1024.6000 | 994.4633 | 2025-02-20 12:00 | 1036.5000 | STOCH_EXIT | 11 | 1173.90 | 1.06% | 1021.3000 | 1084.8735 | 30.1367 | M30 bullish FVG [1013.0000, 1021.3000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 45 | 2025-02-25 18:30 | 1030.9000 | 1002.4325 | 2025-02-25 21:30 | 1035.9000 | STOCH_EXIT | 6 | 432.36 | 0.38% | 1034.0000 | 1087.8350 | 28.4675 | M30 bullish FVG [1030.2000, 1034.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 46 | 2025-03-06 16:00 | 1066.0000 | 1038.9388 | 2025-03-07 10:00 | 1071.6000 | STOCH_EXIT | 22 | 480.31 | 0.42% | 1063.1000 | 1120.1223 | 27.0612 | M30 bullish FVG [1058.0000, 1063.1000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 47 | 2025-03-20 23:30 | 1085.0000 | 1057.4579 | 2025-03-21 10:30 | 1094.0000 | STOCH_EXIT | 9 | 822.69 | 0.73% | 1085.2000 | 1140.0842 | 27.5421 | M30 bullish FVG [1070.0000, 1085.2000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 48 | 2025-03-21 18:30 | 1114.0000 | 1082.7680 | 2025-03-22 03:00 | 1138.0000 | STOCH_EXIT | 13 | 2333.15 | 2.05% | 1134.1000 | 1176.4639 | 31.2320 | M30 bullish FVG [1109.0000, 1134.1000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 49 | 2025-03-28 16:00 | 1060.7000 | 1021.0461 | 2025-03-28 23:30 | 1060.3000 | STOCH_EXIT | 15 | -160.65 | -0.14% | 1058.0000 | 1140.0079 | 39.6539 | M30 bullish FVG [1041.9000, 1058.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 50 | 2025-04-03 11:00 | 1002.4000 | 958.7289 | 2025-04-03 22:30 | 998.0000 | STOCH_EXIT | 23 | -626.42 | -0.54% | 1007.1000 | 1089.7422 | 43.6711 | M30 bullish FVG [1001.0000, 1007.1000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 51 | 2025-04-08 15:30 | 989.7000 | 949.8609 | 2025-04-08 18:30 | 997.0000 | STOCH_EXIT | 6 | 737.88 | 0.64% | 984.6000 | 1069.3782 | 39.8391 | M30 bullish FVG [978.2000, 984.6000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 52 | 2025-04-10 17:00 | 997.3000 | 958.5020 | 2025-04-10 21:00 | 1003.7000 | STOCH_EXIT | 8 | 626.34 | 0.54% | 999.0000 | 1074.8960 | 38.7980 | M30 bullish FVG [994.7000, 999.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 53 | 2025-04-11 22:00 | 1008.8000 | 971.3876 | 2025-04-12 05:00 | 1012.7000 | STOCH_EXIT | 9 | 335.15 | 0.29% | 1011.5000 | 1083.6249 | 37.4124 | M30 bullish FVG [1007.5000, 1011.5000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 54 | 2025-04-16 23:00 | 1004.6000 | 970.5266 | 2025-04-17 07:00 | 1010.5000 | STOCH_EXIT | 2 | 572.42 | 0.49% | 1005.3000 | 1072.7467 | 34.0734 | M30 bullish FVG [1001.6000, 1005.3000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 55 | 2025-04-29 13:00 | 992.0000 | 967.4787 | 2025-04-29 23:30 | 992.8000 | STOCH_EXIT | 21 | -22.90 | -0.02% | 991.0000 | 1041.0426 | 24.5213 | M30 bullish FVG [987.0000, 991.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 56 | 2025-05-06 19:00 | 989.0000 | 965.7022 | 2025-05-06 20:00 | 991.7000 | STOCH_EXIT | 2 | 203.45 | 0.17% | 986.7000 | 1035.5955 | 23.2978 | M30 bullish FVG [984.0000, 986.7000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 57 | 2025-05-16 07:00 | 985.2000 | 964.4131 | 2025-05-16 08:00 | 994.8000 | STOCH_EXIT | 2 | 1033.20 | 0.87% | 986.9000 | 1026.7737 | 20.7869 | M30 bullish FVG [980.8000, 986.9000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 58 | 2025-05-23 21:00 | 975.3000 | 953.4608 | 2025-05-24 08:00 | 979.1000 | STOCH_EXIT | 17 | 344.38 | 0.29% | 974.7000 | 1018.9783 | 21.8392 | M30 bullish FVG [969.9000, 974.7000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 59 | 2025-05-27 11:00 | 966.3000 | 944.0249 | 2025-05-27 16:30 | 973.3000 | STOCH_EXIT | 11 | 741.71 | 0.62% | 972.9000 | 1010.8502 | 22.2751 | M30 bullish FVG [961.9000, 972.9000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 60 | 2025-05-28 12:30 | 982.7000 | 960.3874 | 2025-05-28 19:30 | 982.6000 | STOCH_EXIT | 14 | -132.08 | -0.11% | 984.4000 | 1027.3252 | 22.3126 | M30 bullish FVG [982.0000, 984.4000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 61 | 2025-06-03 16:00 | 975.1000 | 955.1481 | 2025-06-03 20:30 | 980.5000 | STOCH_EXIT | 9 | 543.93 | 0.45% | 970.3000 | 1015.0038 | 19.9519 | M30 bullish FVG [967.0000, 970.3000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 62 | 2025-06-05 14:30 | 976.5000 | 957.3349 | 2025-06-05 21:30 | 977.5000 | STOCH_EXIT | 14 | 2.83 | 0.00% | 978.0000 | 1014.8303 | 19.1651 | M30 bullish FVG [975.9000, 978.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 63 | 2025-06-25 09:00 | 998.8000 | 983.7438 | 2025-06-25 19:30 | 1000.8000 | STOCH_EXIT | 21 | 121.02 | 0.10% | 1003.0000 | 1028.9125 | 15.0562 | M30 bullish FVG [997.5000, 1003.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 64 | 2025-06-26 21:30 | 1005.6000 | 990.4978 | 2025-06-27 09:00 | 1009.0000 | STOCH_EXIT | 9 | 287.12 | 0.24% | 1011.0000 | 1035.8044 | 15.1022 | M30 bullish FVG [1005.0000, 1011.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 65 | 2025-07-03 21:00 | 1062.6000 | 1046.5416 | 2025-07-04 07:00 | 1046.5416 | SL | 7 | -1950.88 | -1.61% | 1072.0000 | 1094.7168 | 16.0584 | M30 bullish FVG [1062.0000, 1072.0000], stochastic crossed below 20 | FVG stop 1046.5416 |
| 66 | 2025-07-07 21:30 | 1049.0000 | 1028.5241 | 2025-07-08 15:30 | 1058.9000 | STOCH_EXIT | 23 | 999.60 | 0.84% | 1053.0000 | 1089.9517 | 20.4759 | M30 bullish FVG [1046.2000, 1053.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 67 | 2025-07-08 21:00 | 1056.0000 | 1034.3938 | 2025-07-09 07:30 | 1055.6000 | STOCH_EXIT | 8 | -164.51 | -0.14% | 1053.5000 | 1099.2123 | 21.6062 | M30 bullish FVG [1048.6000, 1053.5000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 68 | 2025-07-10 21:30 | 1064.1000 | 1042.3977 | 2025-07-11 10:30 | 1069.9000 | STOCH_EXIT | 13 | 530.10 | 0.44% | 1061.9000 | 1107.5045 | 21.7023 | M30 bullish FVG [1059.2000, 1061.9000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 69 | 2025-07-17 16:30 | 1107.0000 | 1080.3205 | 2025-07-18 10:30 | 1114.4000 | STOCH_EXIT | 22 | 685.53 | 0.57% | 1105.5000 | 1160.3590 | 26.6795 | M30 bullish FVG [1098.0000, 1105.5000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 70 | 2025-07-23 07:30 | 1133.2000 | 1104.7040 | 2025-07-23 10:00 | 1137.5000 | STOCH_EXIT | 5 | 338.62 | 0.28% | 1131.0000 | 1190.1921 | 28.4960 | M30 bullish FVG [1126.3000, 1131.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 71 | 2025-07-31 22:30 | 1125.2000 | 1093.6325 | 2025-07-31 23:30 | 1129.5000 | STOCH_EXIT | 2 | 342.65 | 0.28% | 1117.8000 | 1188.3350 | 31.5675 | M30 bullish FVG [1114.0000, 1117.8000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 72 | 2025-08-06 08:30 | 1174.3000 | 1139.8629 | 2025-08-06 20:00 | 1170.5000 | STOCH_EXIT | 23 | -512.16 | -0.42% | 1160.0000 | 1243.1741 | 34.4371 | M30 bullish FVG [1154.0000, 1160.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 73 | 2025-08-13 20:00 | 1247.9000 | 1213.6704 | 2025-08-14 06:30 | 1261.0000 | STOCH_EXIT | 8 | 1149.02 | 0.95% | 1248.0000 | 1316.3591 | 34.2296 | M30 bullish FVG [1243.0000, 1248.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 74 | 2025-08-15 19:30 | 1260.0000 | 1224.4480 | 2025-08-15 23:30 | 1261.4000 | STOCH_EXIT | 8 | 13.51 | 0.01% | 1253.8000 | 1331.1040 | 35.5520 | M30 bullish FVG [1242.5000, 1253.8000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 75 | 2025-09-10 22:00 | 1272.8000 | 1243.0272 | 2025-09-11 08:00 | 1279.0000 | STOCH_EXIT | 7 | 472.71 | 0.39% | 1273.5000 | 1332.3455 | 29.7728 | M30 bullish FVG [1267.2000, 1273.5000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 76 | 2025-09-16 12:30 | 1241.2000 | 1210.8087 | 2025-09-17 07:30 | 1241.1000 | STOCH_EXIT | 25 | -132.77 | -0.11% | 1245.2000 | 1301.9826 | 30.3913 | M30 bullish FVG [1238.7000, 1245.2000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 77 | 2025-09-17 10:00 | 1234.8000 | 1204.7509 | 2025-09-18 09:00 | 1204.7509 | SL | 32 | -3095.62 | -2.53% | 1235.4000 | 1294.8981 | 30.0491 | M30 bullish FVG [1230.0000, 1235.4000], stochastic crossed below 20 | FVG stop 1204.7509 |
| 78 | 2025-09-25 13:00 | 1240.0000 | 1208.5717 | 2025-09-27 15:00 | 1237.6000 | STOCH_EXIT | 80 | -349.32 | -0.29% | 1234.8000 | 1302.8565 | 31.4283 | M30 bullish FVG [1231.6000, 1234.8000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 79 | 2025-09-29 07:30 | 1237.2000 | 1207.2965 | 2025-09-29 11:00 | 1244.9000 | STOCH_EXIT | 7 | 620.06 | 0.52% | 1235.3000 | 1297.0069 | 29.9035 | M30 bullish FVG [1231.8000, 1235.3000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 80 | 2025-09-29 16:00 | 1241.5000 | 1211.5965 | 2025-09-29 22:00 | 1211.5965 | SL | 12 | -2988.48 | -2.51% | 1241.1000 | 1301.3069 | 29.9035 | M30 bullish FVG [1234.6000, 1241.1000], stochastic crossed below 20 | FVG stop 1211.5965 |
| 81 | 2025-09-30 16:00 | 1220.0000 | 1189.3968 | 2025-09-30 18:00 | 1235.2000 | STOCH_EXIT | 4 | 1341.35 | 1.14% | 1221.3000 | 1281.2064 | 30.6032 | M30 bullish FVG [1217.4000, 1221.3000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 82 | 2025-10-01 15:30 | 1240.5000 | 1209.5970 | 2025-10-01 20:30 | 1249.5000 | STOCH_EXIT | 10 | 736.73 | 0.62% | 1235.0000 | 1302.3060 | 30.9030 | M30 bullish FVG [1228.9000, 1235.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 83 | 2025-10-15 14:30 | 1213.0000 | 1172.0607 | 2025-10-15 22:00 | 1221.7000 | STOCH_EXIT | 15 | 733.30 | 0.62% | 1216.3000 | 1294.8786 | 40.9393 | M30 bullish FVG [1211.7000, 1216.3000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 84 | 2025-10-16 11:30 | 1216.2000 | 1175.4849 | 2025-10-16 15:00 | 1219.4000 | STOCH_EXIT | 7 | 194.26 | 0.16% | 1219.0000 | 1297.6301 | 40.7151 | M30 bullish FVG [1214.1000, 1219.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 85 | 2025-10-20 18:30 | 1241.7000 | 1201.6258 | 2025-10-20 22:30 | 1243.5000 | STOCH_EXIT | 8 | 53.51 | 0.04% | 1241.4000 | 1321.8484 | 40.0742 | M30 bullish FVG [1233.0000, 1241.4000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 86 | 2025-10-28 16:30 | 1191.8000 | 1154.8945 | 2025-10-28 20:00 | 1197.1000 | STOCH_EXIT | 7 | 410.55 | 0.34% | 1185.0000 | 1265.6109 | 36.9055 | M30 bullish FVG [1178.9000, 1185.0000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 87 | 2025-11-05 17:00 | 1281.8000 | 1245.2952 | 2025-11-05 22:00 | 1285.0000 | STOCH_EXIT | 10 | 180.16 | 0.15% | 1282.4000 | 1354.8096 | 36.5048 | M30 bullish FVG [1276.7000, 1282.4000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 96095.24
- Макс: 123480.46
- Финал: 120982.08

Полная кривая — в `*_equity.csv`.
