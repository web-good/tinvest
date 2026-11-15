# Бэктест MRKC (Minutes30)

- Период: 2023-11-15 23:19 — 2025-11-15 23:19
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.25 |
| MaxWaitBars | 30 |
| StochPeriod | 11 |
| UseStochExit | 1 |
| StopATR | 1.7 |
| TakeProfitATR | 1 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 49 |
| Выигрышных / проигрышных | 37 / 12 |
| Win rate | 75.51% |
| Gross profit / loss | 38332.49 / 15800.46 |
| Profit factor | 2.426 |
| Чистый PnL | 22532.03 (22.53%) |
| Стартовый / финальный капитал | 100000.00 / 122532.03 |
| Макс. просадка | 8848.85 (8.49%) |
| Средняя прибыль / убыток | 1036.01 / 1316.70 |
| Expectancy | 459.84 |
| Лучшая / худшая сделка | 3721.88 / -4139.02 |
| Exposure | 2.74% |
| CAGR | 10.68% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-11-22 17:30 | 0.6012 | 0.5700 | 2023-11-24 16:00 | 0.5952 | STOCH_EXIT | 35 | -1095.30 | -1.10% | 0.6044 | 0.6196 | 0.0184 | M30 bullish FVG [0.5982, 0.6044], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 2 | 2023-12-08 15:30 | 0.5580 | 0.5261 | 2023-12-08 16:30 | 0.5730 | STOCH_EXIT | 2 | 2554.91 | 2.59% | 0.5632 | 0.5768 | 0.0188 | M30 bullish FVG [0.5564, 0.5632], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 3 | 2023-12-19 17:30 | 0.5776 | 0.5438 | 2023-12-20 16:00 | 0.5810 | STOCH_EXIT | 16 | 493.62 | 0.49% | 0.5816 | 0.5975 | 0.0199 | M30 bullish FVG [0.5750, 0.5816], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 4 | 2024-01-10 14:00 | 0.6160 | 0.5875 | 2024-01-11 16:00 | 0.6064 | STOCH_EXIT | 23 | -1684.85 | -1.66% | 0.6140 | 0.6328 | 0.0168 | M30 bullish FVG [0.6100, 0.6140], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 5 | 2024-01-23 15:00 | 0.6106 | 0.5838 | 2024-01-24 10:00 | 0.6240 | STOCH_EXIT | 9 | 2096.36 | 2.09% | 0.6214 | 0.6263 | 0.0157 | M30 bullish FVG [0.6174, 0.6214], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 6 | 2024-02-22 10:00 | 0.5568 | 0.5322 | 2024-02-22 10:30 | 0.5626 | STOCH_EXIT | 1 | 958.97 | 0.94% | 0.5564 | 0.5713 | 0.0145 | M30 bullish FVG [0.5524, 0.5564], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 7 | 2024-03-11 16:00 | 0.5816 | 0.5612 | 2024-03-12 11:00 | 0.5848 | STOCH_EXIT | 9 | 463.17 | 0.45% | 0.5826 | 0.5936 | 0.0120 | M30 bullish FVG [0.5794, 0.5826], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 8 | 2024-03-13 14:30 | 0.5826 | 0.5624 | 2024-03-15 10:30 | 0.5748 | STOCH_EXIT | 30 | -1491.41 | -1.44% | 0.5834 | 0.5945 | 0.0119 | M30 bullish FVG [0.5784, 0.5834], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 9 | 2024-03-19 10:30 | 0.5682 | 0.5450 | 2024-03-19 13:00 | 0.5710 | STOCH_EXIT | 5 | 399.24 | 0.39% | 0.5700 | 0.5819 | 0.0137 | M30 bullish FVG [0.5664, 0.5700], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 10 | 2024-04-22 17:30 | 0.6148 | 0.5962 | 2024-04-23 09:30 | 0.6198 | STOCH_EXIT | 3 | 727.53 | 0.71% | 0.6130 | 0.6258 | 0.0110 | M30 bullish FVG [0.6098, 0.6130], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 11 | 2024-04-26 11:00 | 0.6048 | 0.5857 | 2024-04-26 18:00 | 0.6036 | STOCH_EXIT | 14 | -306.71 | -0.30% | 0.6074 | 0.6161 | 0.0113 | M30 bullish FVG [0.6028, 0.6074], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 12 | 2024-04-29 15:00 | 0.6268 | 0.6074 | 2024-04-29 18:30 | 0.6296 | STOCH_EXIT | 7 | 356.18 | 0.35% | 0.6304 | 0.6382 | 0.0114 | M30 bullish FVG [0.6262, 0.6304], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 13 | 2024-05-03 17:30 | 0.6350 | 0.6121 | 2024-05-04 10:30 | 0.6390 | STOCH_EXIT | 4 | 544.81 | 0.53% | 0.6360 | 0.6485 | 0.0135 | M30 bullish FVG [0.6300, 0.6360], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 14 | 2024-05-08 17:00 | 0.6430 | 0.6179 | 2024-05-08 17:30 | 0.6179 | SL | 1 | -4139.02 | -4.00% | 0.6484 | 0.6578 | 0.0148 | M30 bullish FVG [0.6406, 0.6484], stochastic crossed below 20 | FVG stop 0.6179 |
| 15 | 2024-05-30 16:00 | 0.5664 | 0.5336 | 2024-05-31 16:00 | 0.5542 | STOCH_EXIT | 19 | -2245.81 | -2.25% | 0.5700 | 0.5857 | 0.0193 | M30 bullish FVG [0.5642, 0.5700], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 16 | 2024-07-05 23:00 | 0.5016 | 0.4681 | 2024-07-06 18:00 | 0.5066 | STOCH_EXIT | 18 | 872.20 | 0.90% | 0.4992 | 0.5213 | 0.0197 | M30 bullish FVG [0.4942, 0.4992], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 17 | 2024-08-08 14:30 | 0.4440 | 0.4183 | 2024-08-09 10:00 | 0.4350 | STOCH_EXIT | 20 | -2086.13 | -2.12% | 0.4486 | 0.4591 | 0.0151 | M30 bullish FVG [0.4444, 0.4486], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 18 | 2024-08-12 20:30 | 0.4322 | 0.4066 | 2024-08-13 16:00 | 0.4364 | STOCH_EXIT | 20 | 835.99 | 0.87% | 0.4294 | 0.4473 | 0.0151 | M30 bullish FVG [0.4240, 0.4294], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 19 | 2024-09-20 16:00 | 0.3860 | 0.3657 | 2024-09-20 17:00 | 0.3906 | STOCH_EXIT | 2 | 1057.14 | 1.09% | 0.3860 | 0.3980 | 0.0120 | M30 bullish FVG [0.3822, 0.3860], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 20 | 2024-10-25 16:00 | 0.3832 | 0.3709 | 2024-10-25 19:00 | 0.3905 | TP | 6 | 1758.25 | 1.79% | 0.3880 | 0.3905 | 0.0073 | M30 bullish FVG [0.3860, 0.3880], stochastic crossed below 20 | FVG target 0.3905 |
| 21 | 2024-10-28 16:30 | 0.3860 | 0.3729 | 2024-10-28 19:30 | 0.3937 | TP | 6 | 1899.15 | 1.90% | 0.3924 | 0.3937 | 0.0077 | M30 bullish FVG [0.3896, 0.3924], stochastic crossed below 20 | FVG target 0.3937 |
| 22 | 2024-11-04 12:00 | 0.4144 | 0.3948 | 2024-11-04 20:30 | 0.4148 | STOCH_EXIT | 16 | -3.58 | -0.00% | 0.4200 | 0.4259 | 0.0115 | M30 bullish FVG [0.4140, 0.4200], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 23 | 2024-11-11 20:00 | 0.4550 | 0.4346 | 2024-11-11 21:00 | 0.4602 | STOCH_EXIT | 2 | 1057.56 | 1.04% | 0.4516 | 0.4670 | 0.0120 | M30 bullish FVG [0.4446, 0.4516], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 24 | 2024-11-13 19:30 | 0.4712 | 0.4480 | 2024-11-14 11:00 | 0.4718 | STOCH_EXIT | 17 | 28.01 | 0.03% | 0.4630 | 0.4849 | 0.0137 | M30 bullish FVG [0.4542, 0.4630], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 25 | 2024-11-19 12:30 | 0.4790 | 0.4528 | 2024-11-19 20:30 | 0.4812 | STOCH_EXIT | 16 | 369.78 | 0.36% | 0.4922 | 0.4944 | 0.0154 | M30 bullish FVG [0.4878, 0.4922], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 26 | 2024-12-02 13:30 | 0.4360 | 0.4040 | 2024-12-02 17:30 | 0.4482 | STOCH_EXIT | 8 | 2786.62 | 2.70% | 0.4480 | 0.4548 | 0.0188 | M30 bullish FVG [0.4424, 0.4480], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 27 | 2024-12-20 21:30 | 0.4508 | 0.4209 | 2024-12-21 07:30 | 0.4578 | STOCH_EXIT | 6 | 1538.24 | 1.45% | 0.4402 | 0.4684 | 0.0176 | M30 bullish FVG [0.4348, 0.4402], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 28 | 2024-12-24 10:30 | 0.4726 | 0.4388 | 2024-12-24 15:30 | 0.4738 | STOCH_EXIT | 10 | 164.98 | 0.15% | 0.4792 | 0.4925 | 0.0199 | M30 bullish FVG [0.4746, 0.4792], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 29 | 2025-01-17 23:30 | 0.4952 | 0.4678 | 2025-01-18 09:30 | 0.4996 | STOCH_EXIT | 6 | 846.86 | 0.79% | 0.5002 | 0.5113 | 0.0161 | M30 bullish FVG [0.4944, 0.5002], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 30 | 2025-02-03 20:30 | 0.5188 | 0.4944 | 2025-02-04 10:00 | 0.5288 | STOCH_EXIT | 13 | 1980.53 | 1.83% | 0.5096 | 0.5332 | 0.0144 | M30 bullish FVG [0.5020, 0.5096], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 31 | 2025-02-13 13:30 | 0.5424 | 0.5191 | 2025-02-13 18:30 | 0.5482 | STOCH_EXIT | 10 | 1071.96 | 0.97% | 0.5512 | 0.5561 | 0.0137 | M30 bullish FVG [0.5420, 0.5512], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 32 | 2025-02-17 16:00 | 0.5616 | 0.5357 | 2025-02-17 18:30 | 0.5628 | STOCH_EXIT | 5 | 126.28 | 0.11% | 0.5604 | 0.5768 | 0.0152 | M30 bullish FVG [0.5524, 0.5604], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 33 | 2025-02-26 22:00 | 0.6504 | 0.6153 | 2025-02-27 09:00 | 0.6604 | STOCH_EXIT | 8 | 1607.27 | 1.44% | 0.6654 | 0.6710 | 0.0206 | M30 bullish FVG [0.6600, 0.6654], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 34 | 2025-03-07 16:00 | 0.6484 | 0.6070 | 2025-03-07 21:30 | 0.6454 | STOCH_EXIT | 11 | -638.21 | -0.56% | 0.6520 | 0.6728 | 0.0244 | M30 bullish FVG [0.6450, 0.6520], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 35 | 2025-03-24 16:00 | 0.6302 | 0.5939 | 2025-03-24 18:00 | 0.6358 | STOCH_EXIT | 4 | 889.09 | 0.79% | 0.6336 | 0.6515 | 0.0213 | M30 bullish FVG [0.6268, 0.6336], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 36 | 2025-05-05 16:00 | 0.5306 | 0.4990 | 2025-05-05 22:30 | 0.5416 | STOCH_EXIT | 13 | 2239.27 | 1.97% | 0.5300 | 0.5492 | 0.0186 | M30 bullish FVG [0.5252, 0.5300], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 37 | 2025-05-27 07:30 | 0.6000 | 0.5677 | 2025-05-27 09:00 | 0.6036 | STOCH_EXIT | 3 | 578.65 | 0.50% | 0.5972 | 0.6190 | 0.0190 | M30 bullish FVG [0.5918, 0.5972], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 38 | 2025-05-27 23:00 | 0.6278 | 0.5955 | 2025-05-28 16:30 | 0.6328 | STOCH_EXIT | 22 | 808.39 | 0.70% | 0.6100 | 0.6468 | 0.0190 | M30 bullish FVG [0.6048, 0.6100], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 39 | 2025-06-09 13:30 | 0.6610 | 0.6316 | 2025-06-09 20:30 | 0.6614 | STOCH_EXIT | 14 | -46.23 | -0.04% | 0.6650 | 0.6783 | 0.0173 | M30 bullish FVG [0.6580, 0.6650], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 40 | 2025-07-11 09:30 | 0.6214 | 0.5849 | 2025-07-11 19:00 | 0.6184 | STOCH_EXIT | 19 | -680.54 | -0.58% | 0.6220 | 0.6428 | 0.0214 | M30 bullish FVG [0.6136, 0.6220], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 41 | 2025-07-17 17:00 | 0.6466 | 0.6134 | 2025-07-17 23:30 | 0.6490 | STOCH_EXIT | 13 | 315.40 | 0.27% | 0.6576 | 0.6661 | 0.0195 | M30 bullish FVG [0.6484, 0.6576], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 42 | 2025-07-28 19:00 | 0.7066 | 0.6764 | 2025-07-28 22:00 | 0.7106 | STOCH_EXIT | 6 | 543.08 | 0.47% | 0.7088 | 0.7243 | 0.0177 | M30 bullish FVG [0.7010, 0.7088], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 43 | 2025-07-30 20:00 | 0.6888 | 0.6510 | 2025-07-31 09:00 | 0.6902 | STOCH_EXIT | 13 | 120.79 | 0.10% | 0.6940 | 0.7110 | 0.0222 | M30 bullish FVG [0.6856, 0.6940], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 44 | 2025-08-05 11:00 | 0.7398 | 0.6985 | 2025-08-05 16:30 | 0.7456 | STOCH_EXIT | 11 | 799.05 | 0.68% | 0.7388 | 0.7641 | 0.0243 | M30 bullish FVG [0.7288, 0.7388], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 45 | 2025-09-29 13:30 | 0.7860 | 0.7445 | 2025-09-29 23:30 | 0.7914 | STOCH_EXIT | 20 | 691.70 | 0.59% | 0.7900 | 0.8104 | 0.0244 | M30 bullish FVG [0.7838, 0.7900], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 46 | 2025-10-01 15:30 | 0.8102 | 0.7651 | 2025-10-02 12:30 | 0.8016 | STOCH_EXIT | 29 | -1382.67 | -1.16% | 0.7972 | 0.8367 | 0.0265 | M30 bullish FVG [0.7896, 0.7972], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 47 | 2025-10-07 22:00 | 0.8292 | 0.7824 | 2025-10-08 07:30 | 0.8320 | STOCH_EXIT | 6 | 277.69 | 0.24% | 0.8354 | 0.8567 | 0.0275 | M30 bullish FVG [0.8270, 0.8354], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 48 | 2025-10-09 23:30 | 0.8552 | 0.8075 | 2025-10-10 07:00 | 0.8832 | TP | 2 | 3721.88 | 3.18% | 0.8882 | 0.8832 | 0.0280 | M30 bullish FVG [0.8776, 0.8882], stochastic crossed below 20 | FVG target 0.8832 |
| 49 | 2025-10-16 10:30 | 0.7774 | 0.6905 | 2025-10-16 16:00 | 0.7830 | STOCH_EXIT | 11 | 751.89 | 0.62% | 0.7780 | 0.8285 | 0.0511 | M30 bullish FVG [0.7642, 0.7780], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 95316.63
- Макс: 122532.03
- Финал: 122532.03

Полная кривая — в `*_equity.csv`.
