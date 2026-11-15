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
| StopATR | 1.2 |
| TakeProfitATR | 0 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 131 |
| Выигрышных / проигрышных | 96 / 35 |
| Win rate | 73.28% |
| Gross profit / loss | 93556.65 / 57362.50 |
| Profit factor | 1.631 |
| Чистый PnL | 36194.15 (36.19%) |
| Стартовый / финальный капитал | 100000.00 / 136194.15 |
| Макс. просадка | 9757.48 (8.65%) |
| Средняя прибыль / убыток | 974.55 / 1638.93 |
| Expectancy | 276.29 |
| Лучшая / худшая сделка | 10257.66 / -5325.39 |
| Exposure | 9.09% |
| CAGR | 16.68% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-11-20 18:30 | 796.4000 | 765.6249 | 2023-11-21 17:00 | 799.1000 | STOCH_EXIT | 16 | 237.78 | 0.24% | 797.1000 | 0.0000 | 25.6459 | M30 bullish FVG [793.3000, 797.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 2 | 2023-11-23 10:30 | 815.1000 | 784.1130 | 2023-11-24 10:00 | 816.0000 | STOCH_EXIT | 18 | 10.30 | 0.01% | 814.0000 | 0.0000 | 25.8225 | M30 bullish FVG [808.3000, 814.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 3 | 2023-11-24 14:30 | 805.6000 | 774.9492 | 2023-11-27 09:30 | 813.3000 | STOCH_EXIT | 9 | 854.43 | 0.85% | 815.0000 | 0.0000 | 25.5423 | M30 bullish FVG [811.3000, 815.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 4 | 2023-12-06 10:30 | 732.0000 | 700.3289 | 2023-12-06 16:30 | 728.9000 | STOCH_EXIT | 12 | -528.60 | -0.52% | 744.0000 | 0.0000 | 26.3926 | M30 bullish FVG [737.2000, 744.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 5 | 2023-12-11 10:30 | 712.3000 | 678.9410 | 2023-12-12 10:00 | 712.0000 | STOCH_EXIT | 18 | -142.71 | -0.14% | 705.0000 | 0.0000 | 27.7992 | M30 bullish FVG [701.8000, 705.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 6 | 2023-12-13 17:00 | 721.9000 | 688.4888 | 2023-12-15 14:00 | 730.8000 | STOCH_EXIT | 32 | 1136.14 | 1.13% | 727.1000 | 0.0000 | 27.8427 | M30 bullish FVG [724.2000, 727.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 7 | 2023-12-22 17:00 | 807.3000 | 770.0104 | 2023-12-25 10:00 | 821.2000 | STOCH_EXIT | 5 | 1635.72 | 1.62% | 805.5000 | 0.0000 | 31.0747 | M30 bullish FVG [796.5000, 805.5000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 8 | 2023-12-25 16:00 | 820.2000 | 782.4111 | 2023-12-26 17:00 | 808.6000 | STOCH_EXIT | 21 | -1551.80 | -1.51% | 820.4000 | 0.0000 | 31.4908 | M30 bullish FVG [809.0000, 820.4000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 9 | 2024-01-08 18:30 | 808.1000 | 773.7136 | 2024-01-10 09:30 | 835.0000 | STOCH_EXIT | 20 | 3259.81 | 3.23% | 818.1000 | 0.0000 | 28.6553 | M30 bullish FVG [815.1000, 818.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 10 | 2024-01-22 13:00 | 800.0000 | 769.9827 | 2024-01-22 14:00 | 807.7000 | STOCH_EXIT | 2 | 903.40 | 0.86% | 804.9000 | 0.0000 | 25.0144 | M30 bullish FVG [800.0000, 804.9000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 11 | 2024-01-23 16:00 | 809.8000 | 780.2982 | 2024-01-25 10:00 | 818.9000 | STOCH_EXIT | 26 | 1077.13 | 1.02% | 814.0000 | 0.0000 | 24.5848 | M30 bullish FVG [809.0000, 814.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 12 | 2024-02-07 10:30 | 986.6000 | 949.6782 | 2024-02-08 10:30 | 949.6782 | SL | 19 | -4092.12 | -3.84% | 974.5000 | 0.0000 | 30.7682 | M30 bullish FVG [946.3000, 974.5000], stochastic crossed below 20 | FVG stop 949.6782 |
| 13 | 2024-02-12 10:30 | 956.2000 | 915.2891 | 2024-02-12 16:00 | 957.8000 | STOCH_EXIT | 11 | 68.80 | 0.07% | 978.5000 | 0.0000 | 34.0924 | M30 bullish FVG [975.0000, 978.5000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 14 | 2024-02-16 12:00 | 993.9000 | 953.1994 | 2024-02-19 14:00 | 1015.4000 | STOCH_EXIT | 23 | 2111.02 | 2.06% | 990.1000 | 0.0000 | 33.9172 | M30 bullish FVG [981.8000, 990.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 15 | 2024-02-20 10:30 | 984.2000 | 942.2404 | 2024-02-21 10:00 | 942.2404 | SL | 18 | -4549.82 | -4.36% | 1008.7000 | 0.0000 | 34.9663 | M30 bullish FVG [1003.0000, 1008.7000], stochastic crossed below 20 | FVG stop 942.2404 |
| 16 | 2024-02-26 17:00 | 957.0000 | 913.4666 | 2024-02-27 09:30 | 968.3000 | STOCH_EXIT | 4 | 1075.08 | 1.08% | 945.0000 | 0.0000 | 36.2778 | M30 bullish FVG [941.0000, 945.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 17 | 2024-02-28 16:30 | 962.7000 | 920.4823 | 2024-02-29 11:30 | 971.1000 | STOCH_EXIT | 9 | 780.48 | 0.77% | 991.8000 | 0.0000 | 35.1814 | M30 bullish FVG [978.9000, 991.8000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 18 | 2024-03-07 13:00 | 1015.7000 | 971.5517 | 2024-03-07 14:00 | 1036.3000 | STOCH_EXIT | 2 | 1957.40 | 1.93% | 1026.8000 | 0.0000 | 36.7902 | M30 bullish FVG [1007.3000, 1026.8000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 19 | 2024-03-11 16:30 | 1045.1000 | 1001.3880 | 2024-03-13 10:00 | 1001.3880 | SL | 25 | -4428.79 | -4.28% | 1030.9000 | 0.0000 | 36.4266 | M30 bullish FVG [1016.6000, 1030.9000], stochastic crossed below 20 | FVG stop 1001.3880 |
| 20 | 2024-03-19 17:30 | 1006.1000 | 964.5429 | 2024-03-21 10:30 | 1001.5000 | STOCH_EXIT | 24 | -554.78 | -0.56% | 1013.0000 | 0.0000 | 34.6309 | M30 bullish FVG [1008.7000, 1013.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 21 | 2024-03-21 17:00 | 998.9000 | 958.5358 | 2024-03-22 11:00 | 1019.0000 | STOCH_EXIT | 7 | 1890.01 | 1.91% | 1008.2000 | 0.0000 | 33.6368 | M30 bullish FVG [1002.0000, 1008.2000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 22 | 2024-04-03 15:00 | 1049.6000 | 1013.5369 | 2024-04-03 18:30 | 1058.0000 | STOCH_EXIT | 7 | 705.24 | 0.70% | 1070.0000 | 0.0000 | 30.0526 | M30 bullish FVG [1060.0000, 1070.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 23 | 2024-04-09 15:30 | 1013.1000 | 973.4173 | 2024-04-10 09:30 | 1017.6000 | STOCH_EXIT | 7 | 348.46 | 0.34% | 1018.0000 | 0.0000 | 33.0689 | M30 bullish FVG [1014.0000, 1018.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 24 | 2024-04-23 12:00 | 1051.1000 | 1016.5133 | 2024-04-24 13:30 | 1050.6000 | STOCH_EXIT | 22 | -150.43 | -0.15% | 1057.0000 | 0.0000 | 28.8223 | M30 bullish FVG [1053.8000, 1057.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 25 | 2024-05-03 13:00 | 1136.9000 | 1098.1004 | 2024-05-04 15:00 | 1154.8000 | STOCH_EXIT | 22 | 1491.12 | 1.47% | 1133.9000 | 0.0000 | 32.3330 | M30 bullish FVG [1115.0000, 1133.9000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 26 | 2024-05-09 14:00 | 1131.9000 | 1093.1743 | 2024-05-09 18:30 | 1139.4000 | STOCH_EXIT | 9 | 579.16 | 0.56% | 1122.0000 | 0.0000 | 32.2714 | M30 bullish FVG [1116.6000, 1122.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 27 | 2024-05-15 13:00 | 1179.1000 | 1138.0054 | 2024-05-15 15:00 | 1138.0054 | SL | 4 | -3718.28 | -3.58% | 1194.9000 | 0.0000 | 34.2455 | M30 bullish FVG [1188.5000, 1194.9000], stochastic crossed below 20 | FVG stop 1138.0054 |
| 28 | 2024-05-17 11:30 | 1064.1000 | 1013.7619 | 2024-05-17 18:00 | 1088.1000 | STOCH_EXIT | 13 | 2154.85 | 2.15% | 1074.0000 | 0.0000 | 41.9484 | M30 bullish FVG [1056.7000, 1074.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 29 | 2024-05-22 16:00 | 1070.1000 | 1014.6207 | 2024-06-17 09:30 | 1179.2000 | STOCH_EXIT | 6 | 10257.66 | 10.09% | 1083.5000 | 0.0000 | 46.2327 | M30 bullish FVG [1078.4000, 1083.5000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 30 | 2024-06-17 16:00 | 1182.0000 | 1127.0978 | 2024-06-17 16:00 | 1127.0978 | SL | 0 | -5325.39 | -4.74% | 1271.7000 | 0.0000 | 45.7518 | M30 bullish FVG [1179.2000, 1271.7000], stochastic crossed below 20 | FVG stop 1127.0978 |
| 31 | 2024-06-18 11:30 | 1032.5000 | 951.4851 | 2024-06-19 09:30 | 1027.0000 | STOCH_EXIT | 15 | -679.09 | -0.63% | 1021.7000 | 0.0000 | 67.5124 | M30 bullish FVG [1000.0000, 1021.7000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 32 | 2024-07-02 16:00 | 829.4000 | 764.9306 | 2024-07-03 16:00 | 830.4000 | STOCH_EXIT | 19 | 21.77 | 0.02% | 834.0000 | 0.0000 | 53.7245 | M30 bullish FVG [826.7000, 834.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 33 | 2024-07-22 13:00 | 841.5000 | 797.4554 | 2024-07-22 14:00 | 858.0000 | STOCH_EXIT | 2 | 1971.93 | 1.86% | 838.2000 | 0.0000 | 36.7039 | M30 bullish FVG [830.2000, 838.2000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 34 | 2024-07-31 16:00 | 872.5000 | 832.1026 | 2024-08-01 09:30 | 880.0000 | STOCH_EXIT | 6 | 821.35 | 0.76% | 850.0000 | 0.0000 | 33.6645 | M30 bullish FVG [843.0000, 850.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 35 | 2024-09-19 08:30 | 840.1000 | 811.2203 | 2024-09-19 08:30 | 841.2000 | STOCH_EXIT | 0 | 33.72 | 0.03% | 839.6000 | 0.0000 | 24.0664 | M30 bullish FVG [837.2000, 839.6000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 36 | 2024-09-24 10:30 | 861.7000 | 833.4522 | 2024-09-25 18:30 | 869.7000 | STOCH_EXIT | 50 | 906.06 | 0.83% | 879.3000 | 0.0000 | 23.5398 | M30 bullish FVG [876.7000, 879.3000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 37 | 2024-09-26 17:00 | 880.0000 | 850.9610 | 2024-09-26 20:00 | 883.6000 | STOCH_EXIT | 6 | 339.77 | 0.31% | 880.9000 | 0.0000 | 24.1991 | M30 bullish FVG [873.5000, 880.9000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 38 | 2024-09-27 18:30 | 886.6000 | 858.1181 | 2024-09-28 07:30 | 890.0000 | STOCH_EXIT | 12 | 313.96 | 0.28% | 882.4000 | 0.0000 | 23.7349 | M30 bullish FVG [880.0000, 882.4000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 39 | 2024-09-30 18:00 | 901.3000 | 873.1640 | 2024-09-30 19:00 | 908.9000 | STOCH_EXIT | 2 | 823.47 | 0.74% | 907.0000 | 0.0000 | 23.4467 | M30 bullish FVG [896.9000, 907.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 40 | 2024-10-02 18:30 | 886.8000 | 856.7694 | 2024-10-03 15:30 | 856.7694 | SL | 28 | -3893.70 | -3.48% | 905.0000 | 0.0000 | 25.0255 | M30 bullish FVG [895.8000, 905.0000], stochastic crossed below 20 | FVG stop 856.7694 |
| 41 | 2024-10-04 13:00 | 870.9000 | 839.5934 | 2024-10-04 20:30 | 874.8000 | STOCH_EXIT | 15 | 375.37 | 0.35% | 879.0000 | 0.0000 | 26.0888 | M30 bullish FVG [873.9000, 879.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 42 | 2024-10-07 11:30 | 875.2000 | 844.3725 | 2024-10-08 08:30 | 869.7000 | STOCH_EXIT | 28 | -783.81 | -0.73% | 881.3000 | 0.0000 | 25.6896 | M30 bullish FVG [877.0000, 881.3000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 43 | 2024-10-15 18:30 | 884.0000 | 857.4790 | 2024-10-15 22:00 | 889.3000 | STOCH_EXIT | 7 | 534.02 | 0.50% | 884.1000 | 0.0000 | 22.1008 | M30 bullish FVG [880.0000, 884.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 44 | 2024-10-23 20:00 | 840.3000 | 815.3844 | 2024-10-24 09:30 | 846.0000 | STOCH_EXIT | 13 | 621.68 | 0.58% | 843.3000 | 0.0000 | 20.7630 | M30 bullish FVG [841.0000, 843.3000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 45 | 2024-10-25 11:00 | 850.7000 | 824.4895 | 2024-10-25 12:00 | 859.0000 | STOCH_EXIT | 2 | 945.53 | 0.87% | 851.2000 | 0.0000 | 21.8421 | M30 bullish FVG [848.0000, 851.2000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 46 | 2024-10-30 14:30 | 862.8000 | 834.1316 | 2024-10-31 08:30 | 875.1000 | STOCH_EXIT | 22 | 1451.74 | 1.32% | 869.0000 | 0.0000 | 23.8903 | M30 bullish FVG [843.2000, 869.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 47 | 2024-11-12 19:00 | 878.8000 | 844.8960 | 2024-11-13 18:30 | 883.0000 | STOCH_EXIT | 33 | 418.21 | 0.38% | 883.6000 | 0.0000 | 28.2534 | M30 bullish FVG [880.3000, 883.6000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 48 | 2024-11-13 20:00 | 877.1000 | 843.9634 | 2024-11-14 10:30 | 878.2000 | STOCH_EXIT | 15 | 28.24 | 0.03% | 882.9000 | 0.0000 | 27.6138 | M30 bullish FVG [879.5000, 882.9000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 49 | 2024-11-21 18:00 | 864.6000 | 831.4362 | 2024-11-21 23:00 | 870.0000 | STOCH_EXIT | 10 | 584.72 | 0.52% | 873.0000 | 0.0000 | 27.6365 | M30 bullish FVG [869.5000, 873.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 50 | 2024-11-27 18:00 | 854.5000 | 824.0543 | 2024-11-27 23:00 | 869.9000 | STOCH_EXIT | 10 | 1904.45 | 1.70% | 858.0000 | 0.0000 | 25.3714 | M30 bullish FVG [853.8000, 858.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 51 | 2024-11-29 21:30 | 850.8000 | 819.1447 | 2024-11-30 08:30 | 856.9000 | STOCH_EXIT | 8 | 702.98 | 0.62% | 860.0000 | 0.0000 | 26.3794 | M30 bullish FVG [853.3000, 860.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 52 | 2024-12-26 18:00 | 853.0000 | 831.3233 | 2024-12-27 17:30 | 853.6000 | STOCH_EXIT | 33 | -33.94 | -0.03% | 865.0000 | 0.0000 | 18.0639 | M30 bullish FVG [857.7000, 865.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 53 | 2025-01-08 20:30 | 941.0000 | 911.0618 | 2025-01-09 11:30 | 911.0618 | SL | 16 | -3765.44 | -3.28% | 941.2000 | 0.0000 | 24.9485 | M30 bullish FVG [936.0000, 941.2000], stochastic crossed below 20 | FVG stop 911.0618 |
| 54 | 2025-01-09 19:00 | 929.0000 | 898.5602 | 2025-01-09 19:30 | 931.0000 | STOCH_EXIT | 1 | 127.33 | 0.12% | 930.9000 | 0.0000 | 25.3665 | M30 bullish FVG [923.6000, 930.9000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 55 | 2025-01-22 20:00 | 943.9000 | 915.4165 | 2025-01-23 11:00 | 980.2000 | STOCH_EXIT | 16 | 4134.54 | 3.74% | 951.6000 | 0.0000 | 23.7363 | M30 bullish FVG [944.6000, 951.6000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 56 | 2025-01-23 18:00 | 964.0000 | 934.9024 | 2025-01-23 22:00 | 969.0000 | STOCH_EXIT | 8 | 479.99 | 0.42% | 970.5000 | 0.0000 | 24.2480 | M30 bullish FVG [963.9000, 970.5000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 57 | 2025-01-27 21:30 | 994.6000 | 964.2112 | 2025-01-28 08:30 | 1003.1000 | STOCH_EXIT | 8 | 870.13 | 0.75% | 1008.6000 | 0.0000 | 25.3240 | M30 bullish FVG [1001.9000, 1008.6000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 58 | 2025-02-10 23:00 | 1002.6000 | 965.8113 | 2025-02-11 12:30 | 1014.0000 | STOCH_EXIT | 13 | 1205.44 | 1.04% | 1015.0000 | 0.0000 | 30.6573 | M30 bullish FVG [1011.0000, 1015.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 59 | 2025-02-11 17:00 | 1005.9000 | 969.6391 | 2025-02-12 07:00 | 1014.7000 | STOCH_EXIT | 14 | 911.39 | 0.77% | 1013.2000 | 0.0000 | 30.2175 | M30 bullish FVG [1010.0000, 1013.2000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 60 | 2025-02-20 10:00 | 1024.8000 | 989.3162 | 2025-02-20 12:00 | 1036.5000 | STOCH_EXIT | 4 | 1226.98 | 1.04% | 1021.3000 | 0.0000 | 29.5698 | M30 bullish FVG [1013.0000, 1021.3000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 61 | 2025-02-25 19:00 | 1027.1000 | 992.9390 | 2025-02-25 21:30 | 1035.9000 | STOCH_EXIT | 5 | 901.15 | 0.76% | 1034.0000 | 0.0000 | 28.4675 | M30 bullish FVG [1030.2000, 1034.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 62 | 2025-02-27 12:00 | 1001.2000 | 967.9961 | 2025-02-27 16:30 | 1010.7000 | STOCH_EXIT | 9 | 1019.29 | 0.85% | 1031.4000 | 0.0000 | 27.6699 | M30 bullish FVG [1027.5000, 1031.4000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 63 | 2025-02-27 23:00 | 1016.7000 | 983.4961 | 2025-02-28 08:30 | 1033.7000 | STOCH_EXIT | 5 | 1901.00 | 1.57% | 1019.5000 | 0.0000 | 27.6699 | M30 bullish FVG [1016.0000, 1019.5000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 64 | 2025-02-28 13:30 | 1015.1000 | 981.8849 | 2025-03-01 10:30 | 1020.6000 | STOCH_EXIT | 38 | 546.82 | 0.44% | 1022.0000 | 0.0000 | 27.6792 | M30 bullish FVG [1015.4000, 1022.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 65 | 2025-03-04 15:30 | 1036.2000 | 1002.5866 | 2025-03-04 18:30 | 1047.8000 | STOCH_EXIT | 6 | 1266.96 | 1.02% | 1051.3000 | 0.0000 | 28.0112 | M30 bullish FVG [1048.0000, 1051.3000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 66 | 2025-03-06 16:00 | 1066.0000 | 1033.5266 | 2025-03-07 10:00 | 1071.6000 | STOCH_EXIT | 22 | 530.15 | 0.42% | 1063.1000 | 0.0000 | 27.0612 | M30 bullish FVG [1058.0000, 1063.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 67 | 2025-03-13 10:00 | 1031.6000 | 998.2470 | 2025-03-13 17:00 | 1025.0000 | STOCH_EXIT | 14 | -930.65 | -0.74% | 1035.5000 | 0.0000 | 27.7942 | M30 bullish FVG [1032.0000, 1035.5000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 68 | 2025-03-21 19:00 | 1111.2000 | 1073.7216 | 2025-03-22 02:00 | 1135.0000 | STOCH_EXIT | 10 | 2539.81 | 2.04% | 1134.1000 | 0.0000 | 31.2320 | M30 bullish FVG [1109.0000, 1134.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 69 | 2025-03-27 10:30 | 1067.5000 | 1018.9780 | 2025-03-28 10:00 | 1059.0000 | STOCH_EXIT | 34 | -1138.03 | -0.90% | 1075.1000 | 0.0000 | 40.4350 | M30 bullish FVG [1069.0000, 1075.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 70 | 2025-03-28 16:00 | 1060.7000 | 1013.1153 | 2025-03-28 23:30 | 1060.3000 | STOCH_EXIT | 15 | -173.80 | -0.14% | 1058.0000 | 0.0000 | 39.6539 | M30 bullish FVG [1041.9000, 1058.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 71 | 2025-03-31 11:30 | 1037.3000 | 989.8656 | 2025-03-31 20:30 | 1023.4000 | STOCH_EXIT | 18 | -1806.57 | -1.44% | 1076.0000 | 0.0000 | 39.5286 | M30 bullish FVG [1067.3000, 1076.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 72 | 2025-04-03 11:00 | 1002.4000 | 949.9947 | 2025-04-03 23:00 | 998.4000 | STOCH_EXIT | 24 | -620.05 | -0.50% | 1007.1000 | 0.0000 | 43.6711 | M30 bullish FVG [1001.0000, 1007.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 73 | 2025-04-07 15:30 | 973.8000 | 925.9156 | 2025-04-07 20:00 | 985.7000 | STOCH_EXIT | 9 | 1386.87 | 1.12% | 986.0000 | 0.0000 | 39.9037 | M30 bullish FVG [981.6000, 986.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 74 | 2025-04-08 15:30 | 989.7000 | 941.8931 | 2025-04-08 18:30 | 997.0000 | STOCH_EXIT | 6 | 794.64 | 0.64% | 984.6000 | 0.0000 | 39.8391 | M30 bullish FVG [978.2000, 984.6000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 75 | 2025-04-10 17:00 | 997.3000 | 950.7424 | 2025-04-10 21:00 | 1003.7000 | STOCH_EXIT | 8 | 680.34 | 0.54% | 999.0000 | 0.0000 | 38.7980 | M30 bullish FVG [994.7000, 999.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 76 | 2025-04-11 22:00 | 1008.8000 | 963.9051 | 2025-04-12 05:30 | 1012.0000 | STOCH_EXIT | 10 | 273.70 | 0.22% | 1011.5000 | 0.0000 | 37.4124 | M30 bullish FVG [1007.5000, 1011.5000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 77 | 2025-04-16 23:30 | 1004.5000 | 963.6120 | 2025-04-17 07:00 | 1010.5000 | STOCH_EXIT | 1 | 629.06 | 0.50% | 1005.3000 | 0.0000 | 34.0734 | M30 bullish FVG [1001.6000, 1005.3000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 78 | 2025-04-29 13:00 | 992.0000 | 962.5744 | 2025-04-29 23:30 | 992.8000 | STOCH_EXIT | 21 | -24.63 | -0.02% | 991.0000 | 0.0000 | 24.5213 | M30 bullish FVG [987.0000, 991.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 79 | 2025-05-06 19:00 | 989.0000 | 961.0427 | 2025-05-06 20:00 | 991.7000 | STOCH_EXIT | 2 | 220.54 | 0.17% | 986.7000 | 0.0000 | 23.2978 | M30 bullish FVG [984.0000, 986.7000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 80 | 2025-05-16 13:30 | 988.2000 | 963.2558 | 2025-05-16 16:30 | 995.0000 | STOCH_EXIT | 6 | 749.28 | 0.59% | 986.9000 | 0.0000 | 20.7869 | M30 bullish FVG [980.8000, 986.9000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 81 | 2025-05-23 19:00 | 977.5000 | 951.2930 | 2025-05-23 19:30 | 981.3000 | STOCH_EXIT | 1 | 369.50 | 0.29% | 974.7000 | 0.0000 | 21.8392 | M30 bullish FVG [969.9000, 974.7000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 82 | 2025-05-27 11:00 | 966.3000 | 939.5699 | 2025-05-27 16:30 | 973.3000 | STOCH_EXIT | 11 | 802.02 | 0.62% | 972.9000 | 0.0000 | 22.2751 | M30 bullish FVG [961.9000, 972.9000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 83 | 2025-05-28 13:00 | 981.2000 | 954.4249 | 2025-05-28 19:30 | 982.6000 | STOCH_EXIT | 13 | 55.19 | 0.04% | 984.4000 | 0.0000 | 22.3126 | M30 bullish FVG [982.0000, 984.4000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 84 | 2025-06-03 16:00 | 975.1000 | 951.1577 | 2025-06-03 20:30 | 980.5000 | STOCH_EXIT | 9 | 588.15 | 0.45% | 970.3000 | 0.0000 | 19.9519 | M30 bullish FVG [967.0000, 970.3000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 85 | 2025-06-05 14:30 | 976.5000 | 953.5018 | 2025-06-05 22:00 | 979.6000 | STOCH_EXIT | 15 | 282.22 | 0.22% | 978.0000 | 0.0000 | 19.1651 | M30 bullish FVG [975.9000, 978.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 86 | 2025-06-13 10:00 | 992.5000 | 971.6466 | 2025-06-14 02:00 | 971.6466 | SL | 28 | -2860.44 | -2.20% | 996.3000 | 0.0000 | 17.3778 | M30 bullish FVG [994.0000, 996.3000], stochastic crossed below 20 | FVG stop 971.6466 |
| 87 | 2025-06-18 16:30 | 984.1000 | 964.2825 | 2025-06-18 19:00 | 992.3000 | STOCH_EXIT | 5 | 937.53 | 0.73% | 993.0000 | 0.0000 | 16.5146 | M30 bullish FVG [991.0000, 993.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 88 | 2025-06-24 10:30 | 991.6000 | 973.5550 | 2025-06-24 13:00 | 997.5000 | STOCH_EXIT | 5 | 637.71 | 0.49% | 995.0000 | 0.0000 | 15.0375 | M30 bullish FVG [993.3000, 995.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 89 | 2025-06-25 09:00 | 998.8000 | 980.7325 | 2025-06-25 19:30 | 1000.8000 | STOCH_EXIT | 21 | 129.03 | 0.10% | 1003.0000 | 0.0000 | 15.0562 | M30 bullish FVG [997.5000, 1003.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 90 | 2025-06-26 21:30 | 1005.6000 | 987.4773 | 2025-06-27 09:00 | 1009.0000 | STOCH_EXIT | 9 | 306.27 | 0.24% | 1011.0000 | 0.0000 | 15.1022 | M30 bullish FVG [1005.0000, 1011.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 91 | 2025-07-03 21:30 | 1062.4000 | 1043.1299 | 2025-07-04 07:00 | 1043.1299 | SL | 6 | -2479.39 | -1.91% | 1072.0000 | 0.0000 | 16.0584 | M30 bullish FVG [1062.0000, 1072.0000], stochastic crossed below 20 | FVG stop 1043.1299 |
| 92 | 2025-07-04 17:00 | 1048.0000 | 1026.1635 | 2025-07-05 08:30 | 1056.1000 | STOCH_EXIT | 26 | 852.80 | 0.67% | 1068.7000 | 0.0000 | 18.1971 | M30 bullish FVG [1066.4000, 1068.7000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 93 | 2025-07-07 21:30 | 1049.0000 | 1024.4290 | 2025-07-08 15:30 | 1058.9000 | STOCH_EXIT | 23 | 1079.22 | 0.84% | 1053.0000 | 0.0000 | 20.4759 | M30 bullish FVG [1046.2000, 1053.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 94 | 2025-07-08 21:00 | 1056.0000 | 1030.0726 | 2025-07-09 22:30 | 1039.1000 | STOCH_EXIT | 38 | -2189.60 | -1.70% | 1058.9000 | 0.0000 | 21.6062 | M30 bullish FVG [1056.1000, 1058.9000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 95 | 2025-07-10 22:00 | 1063.0000 | 1036.9573 | 2025-07-11 10:30 | 1069.9000 | STOCH_EXIT | 12 | 694.19 | 0.55% | 1061.9000 | 0.0000 | 21.7023 | M30 bullish FVG [1059.2000, 1061.9000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 96 | 2025-07-11 14:30 | 1058.0000 | 1031.3489 | 2025-07-11 21:30 | 1031.3489 | SL | 14 | -3323.49 | -2.62% | 1066.4000 | 0.0000 | 22.2092 | M30 bullish FVG [1062.0000, 1066.4000], stochastic crossed below 20 | FVG stop 1031.3489 |
| 97 | 2025-07-17 16:30 | 1107.0000 | 1074.9846 | 2025-07-18 10:30 | 1114.4000 | STOCH_EXIT | 22 | 704.40 | 0.57% | 1105.5000 | 0.0000 | 26.6795 | M30 bullish FVG [1098.0000, 1105.5000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 98 | 2025-07-23 17:30 | 1133.6000 | 1099.4048 | 2025-07-23 22:30 | 1133.0000 | STOCH_EXIT | 10 | -190.66 | -0.15% | 1131.0000 | 0.0000 | 28.4960 | M30 bullish FVG [1126.3000, 1131.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 99 | 2025-07-25 18:00 | 1119.0000 | 1085.5694 | 2025-07-25 23:00 | 1129.9000 | STOCH_EXIT | 10 | 1085.09 | 0.87% | 1138.2000 | 0.0000 | 27.8588 | M30 bullish FVG [1132.9000, 1138.2000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 100 | 2025-07-28 17:30 | 1149.6000 | 1116.0630 | 2025-07-29 07:00 | 1149.0000 | STOCH_EXIT | 13 | -190.67 | -0.15% | 1158.8000 | 0.0000 | 27.9475 | M30 bullish FVG [1150.4000, 1158.8000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 101 | 2025-07-31 22:30 | 1125.2000 | 1087.3190 | 2025-07-31 23:30 | 1129.5000 | STOCH_EXIT | 2 | 355.34 | 0.28% | 1117.8000 | 0.0000 | 31.5675 | M30 bullish FVG [1114.0000, 1117.8000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 102 | 2025-08-06 09:00 | 1174.4000 | 1133.0755 | 2025-08-06 20:00 | 1170.5000 | STOCH_EXIT | 22 | -542.75 | -0.43% | 1160.0000 | 0.0000 | 34.4371 | M30 bullish FVG [1154.0000, 1160.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 103 | 2025-08-12 14:30 | 1216.4000 | 1175.9495 | 2025-08-12 17:00 | 1240.9000 | STOCH_EXIT | 5 | 2396.95 | 1.91% | 1240.1000 | 0.0000 | 33.7087 | M30 bullish FVG [1232.3000, 1240.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 104 | 2025-08-13 20:00 | 1247.9000 | 1206.8245 | 2025-08-14 07:00 | 1265.2000 | STOCH_EXIT | 9 | 1636.43 | 1.28% | 1248.0000 | 0.0000 | 34.2296 | M30 bullish FVG [1243.0000, 1248.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 105 | 2025-08-14 21:00 | 1239.3000 | 1197.3528 | 2025-08-15 07:00 | 1254.8000 | STOCH_EXIT | 7 | 1482.31 | 1.15% | 1254.8000 | 0.0000 | 34.9560 | M30 bullish FVG [1250.0000, 1254.8000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 106 | 2025-08-15 19:30 | 1260.0000 | 1217.3376 | 2025-08-15 23:30 | 1261.4000 | STOCH_EXIT | 8 | 14.49 | 0.01% | 1253.8000 | 0.0000 | 35.5520 | M30 bullish FVG [1242.5000, 1253.8000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 107 | 2025-08-20 14:00 | 1280.3000 | 1239.7678 | 2025-08-20 20:30 | 1263.0000 | STOCH_EXIT | 13 | -1894.31 | -1.45% | 1299.0000 | 0.0000 | 33.7768 | M30 bullish FVG [1295.0000, 1299.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 108 | 2025-08-21 11:00 | 1251.9000 | 1209.0773 | 2025-08-22 08:30 | 1254.6000 | STOCH_EXIT | 30 | 149.02 | 0.12% | 1269.7000 | 0.0000 | 35.6856 | M30 bullish FVG [1261.5000, 1269.7000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 109 | 2025-08-29 17:00 | 1241.2000 | 1196.9372 | 2025-08-29 18:30 | 1249.6000 | STOCH_EXIT | 3 | 744.08 | 0.58% | 1253.8000 | 0.0000 | 36.8857 | M30 bullish FVG [1248.6000, 1253.8000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 110 | 2025-09-05 15:00 | 1244.8000 | 1207.0757 | 2025-09-05 22:30 | 1246.2000 | STOCH_EXIT | 15 | 16.07 | 0.01% | 1251.1000 | 0.0000 | 31.4369 | M30 bullish FVG [1246.5000, 1251.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 111 | 2025-09-10 22:00 | 1272.8000 | 1237.0727 | 2025-09-11 08:00 | 1279.0000 | STOCH_EXIT | 7 | 502.26 | 0.39% | 1273.5000 | 0.0000 | 29.7728 | M30 bullish FVG [1267.2000, 1273.5000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 112 | 2025-09-11 11:00 | 1265.8000 | 1229.7446 | 2025-09-11 22:00 | 1260.4000 | STOCH_EXIT | 22 | -686.30 | -0.53% | 1277.1000 | 0.0000 | 30.0461 | M30 bullish FVG [1273.0000, 1277.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 113 | 2025-09-16 13:00 | 1234.8000 | 1198.3304 | 2025-09-17 07:30 | 1241.1000 | STOCH_EXIT | 24 | 531.52 | 0.41% | 1245.2000 | 0.0000 | 30.3913 | M30 bullish FVG [1238.7000, 1245.2000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 114 | 2025-09-17 10:00 | 1234.8000 | 1198.7411 | 2025-09-18 12:00 | 1219.7000 | STOCH_EXIT | 38 | -1714.36 | -1.32% | 1235.4000 | 0.0000 | 30.0491 | M30 bullish FVG [1230.0000, 1235.4000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 115 | 2025-09-18 15:00 | 1207.1000 | 1170.3253 | 2025-09-18 17:30 | 1221.0000 | STOCH_EXIT | 5 | 1344.71 | 1.05% | 1216.4000 | 0.0000 | 30.6456 | M30 bullish FVG [1210.0000, 1216.4000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 116 | 2025-09-22 11:30 | 1205.7000 | 1168.8820 | 2025-09-22 17:30 | 1206.3000 | STOCH_EXIT | 12 | -65.45 | -0.05% | 1222.0000 | 0.0000 | 30.6816 | M30 bullish FVG [1214.0000, 1222.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 117 | 2025-09-25 13:00 | 1240.0000 | 1202.2861 | 2025-09-27 15:30 | 1237.2000 | STOCH_EXIT | 81 | -424.05 | -0.33% | 1234.8000 | 0.0000 | 31.4283 | M30 bullish FVG [1231.6000, 1234.8000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 118 | 2025-09-29 07:30 | 1237.2000 | 1201.3159 | 2025-09-29 11:00 | 1244.9000 | STOCH_EXIT | 7 | 671.73 | 0.52% | 1241.0000 | 0.0000 | 29.9035 | M30 bullish FVG [1236.8000, 1241.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 119 | 2025-09-29 16:30 | 1240.5000 | 1204.6159 | 2025-09-30 07:30 | 1223.7000 | STOCH_EXIT | 17 | -1893.37 | -1.45% | 1241.1000 | 0.0000 | 29.9035 | M30 bullish FVG [1234.6000, 1241.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 120 | 2025-09-30 16:00 | 1220.0000 | 1183.2762 | 2025-09-30 18:00 | 1235.2000 | STOCH_EXIT | 4 | 1467.10 | 1.14% | 1231.0000 | 0.0000 | 30.6032 | M30 bullish FVG [1225.5000, 1231.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 121 | 2025-10-01 16:00 | 1241.0000 | 1203.9164 | 2025-10-01 20:30 | 1249.5000 | STOCH_EXIT | 9 | 754.49 | 0.58% | 1258.1000 | 0.0000 | 30.9030 | M30 bullish FVG [1252.1000, 1258.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 122 | 2025-10-06 18:00 | 1245.9000 | 1208.7190 | 2025-10-06 22:30 | 1256.6000 | STOCH_EXIT | 9 | 992.12 | 0.76% | 1257.0000 | 0.0000 | 30.9842 | M30 bullish FVG [1252.0000, 1257.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 123 | 2025-10-14 11:30 | 1272.8000 | 1230.2600 | 2025-10-14 14:30 | 1281.2000 | STOCH_EXIT | 6 | 733.67 | 0.56% | 1285.5000 | 0.0000 | 35.4500 | M30 bullish FVG [1280.0000, 1285.5000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 124 | 2025-10-15 14:30 | 1213.0000 | 1163.8728 | 2025-10-15 22:00 | 1221.7000 | STOCH_EXIT | 15 | 815.61 | 0.62% | 1216.3000 | 0.0000 | 40.9393 | M30 bullish FVG [1211.7000, 1216.3000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 125 | 2025-10-16 11:30 | 1216.2000 | 1167.3419 | 2025-10-16 15:00 | 1219.4000 | STOCH_EXIT | 7 | 216.06 | 0.16% | 1219.0000 | 0.0000 | 40.7151 | M30 bullish FVG [1214.1000, 1219.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 126 | 2025-10-20 18:30 | 1241.7000 | 1193.6109 | 2025-10-20 23:30 | 1242.8000 | STOCH_EXIT | 10 | -15.22 | -0.01% | 1241.4000 | 0.0000 | 40.0742 | M30 bullish FVG [1233.0000, 1241.4000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 127 | 2025-10-28 16:30 | 1191.8000 | 1147.5135 | 2025-10-28 20:00 | 1197.1000 | STOCH_EXIT | 7 | 459.82 | 0.34% | 1185.0000 | 0.0000 | 36.9055 | M30 bullish FVG [1178.9000, 1185.0000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 128 | 2025-11-03 17:00 | 1272.4000 | 1229.6000 | 2025-11-03 20:30 | 1278.4000 | STOCH_EXIT | 7 | 496.08 | 0.37% | 1280.8000 | 0.0000 | 35.6667 | M30 bullish FVG [1275.4000, 1280.8000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 129 | 2025-11-05 17:00 | 1281.8000 | 1237.9943 | 2025-11-05 22:30 | 1291.6000 | STOCH_EXIT | 11 | 893.90 | 0.66% | 1297.1000 | 0.0000 | 36.5048 | M30 bullish FVG [1284.9000, 1297.1000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 130 | 2025-11-06 11:00 | 1285.4000 | 1241.2175 | 2025-11-06 16:30 | 1292.5000 | STOCH_EXIT | 11 | 610.16 | 0.45% | 1297.4000 | 0.0000 | 36.8187 | M30 bullish FVG [1292.9000, 1297.4000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 131 | 2025-11-13 14:30 | 1296.5000 | 1254.8895 | 2025-11-13 19:30 | 1298.0000 | STOCH_EXIT | 10 | 21.09 | 0.02% | 1311.8000 | 0.0000 | 34.6754 | M30 bullish FVG [1306.1000, 1311.8000], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 98268.95
- Макс: 136356.42
- Финал: 136194.15

Полная кривая — в `*_equity.csv`.
