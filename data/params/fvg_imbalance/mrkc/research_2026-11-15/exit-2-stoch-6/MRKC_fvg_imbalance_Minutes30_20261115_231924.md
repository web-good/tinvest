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
| StopATR | 0.5 |
| TakeProfitATR | 1 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 49 |
| Выигрышных / проигрышных | 34 / 15 |
| Win rate | 69.39% |
| Gross profit / loss | 36188.41 / 21325.42 |
| Profit factor | 1.697 |
| Чистый PnL | 14862.99 (14.86%) |
| Стартовый / финальный капитал | 100000.00 / 114862.99 |
| Макс. просадка | 5675.61 (5.51%) |
| Средняя прибыль / убыток | 1064.36 / 1421.69 |
| Expectancy | 303.33 |
| Лучшая / худшая сделка | 3504.55 / -2217.61 |
| Exposure | 2.23% |
| CAGR | 7.16% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2023-11-22 17:30 | 0.6012 | 0.5920 | 2023-11-24 10:00 | 0.5920 | SL | 23 | -1624.47 | -1.63% | 0.6044 | 0.6196 | 0.0184 | M30 bullish FVG [0.5982, 0.6044], stochastic crossed below 20 | FVG stop 0.5920 |
| 2 | 2023-12-08 15:30 | 0.5580 | 0.5486 | 2023-12-08 16:30 | 0.5730 | STOCH_EXIT | 2 | 2540.47 | 2.59% | 0.5632 | 0.5768 | 0.0188 | M30 bullish FVG [0.5564, 0.5632], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 3 | 2023-12-19 17:30 | 0.5776 | 0.5677 | 2023-12-20 16:00 | 0.5810 | STOCH_EXIT | 16 | 490.80 | 0.49% | 0.5816 | 0.5975 | 0.0199 | M30 bullish FVG [0.5750, 0.5816], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 4 | 2024-01-10 14:00 | 0.6160 | 0.6076 | 2024-01-11 10:30 | 0.6076 | SL | 12 | -1476.11 | -1.46% | 0.6140 | 0.6328 | 0.0168 | M30 bullish FVG [0.6100, 0.6140], stochastic crossed below 20 | FVG stop 0.6076 |
| 5 | 2024-01-23 15:00 | 0.6106 | 0.6027 | 2024-01-24 10:00 | 0.6240 | STOCH_EXIT | 9 | 2083.58 | 2.09% | 0.6214 | 0.6263 | 0.0157 | M30 bullish FVG [0.6174, 0.6214], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 6 | 2024-02-22 10:00 | 0.5568 | 0.5496 | 2024-02-22 10:30 | 0.5626 | STOCH_EXIT | 1 | 958.97 | 0.94% | 0.5564 | 0.5713 | 0.0145 | M30 bullish FVG [0.5524, 0.5564], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 7 | 2024-03-11 16:00 | 0.5816 | 0.5756 | 2024-03-12 10:00 | 0.5756 | SL | 7 | -1160.20 | -1.13% | 0.5826 | 0.5936 | 0.0120 | M30 bullish FVG [0.5794, 0.5826], stochastic crossed below 20 | FVG stop 0.5756 |
| 8 | 2024-03-13 14:30 | 0.5826 | 0.5766 | 2024-03-14 10:30 | 0.5766 | SL | 11 | -1136.57 | -1.12% | 0.5834 | 0.5945 | 0.0119 | M30 bullish FVG [0.5784, 0.5834], stochastic crossed below 20 | FVG stop 0.5766 |
| 9 | 2024-03-19 10:30 | 0.5682 | 0.5614 | 2024-03-19 13:00 | 0.5710 | STOCH_EXIT | 5 | 394.78 | 0.39% | 0.5700 | 0.5819 | 0.0137 | M30 bullish FVG [0.5664, 0.5700], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 10 | 2024-04-22 17:30 | 0.6148 | 0.6093 | 2024-04-23 09:30 | 0.6198 | STOCH_EXIT | 3 | 718.76 | 0.71% | 0.6130 | 0.6258 | 0.0110 | M30 bullish FVG [0.6098, 0.6130], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 11 | 2024-04-26 11:00 | 0.6048 | 0.5992 | 2024-04-26 15:30 | 0.5992 | SL | 9 | -1047.14 | -1.03% | 0.6074 | 0.6161 | 0.0113 | M30 bullish FVG [0.6028, 0.6074], stochastic crossed below 20 | FVG stop 0.5992 |
| 12 | 2024-04-29 15:00 | 0.6268 | 0.6211 | 2024-04-29 18:30 | 0.6296 | STOCH_EXIT | 7 | 347.49 | 0.35% | 0.6304 | 0.6382 | 0.0114 | M30 bullish FVG [0.6262, 0.6304], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 13 | 2024-05-03 17:30 | 0.6350 | 0.6283 | 2024-05-04 10:30 | 0.6390 | STOCH_EXIT | 4 | 534.72 | 0.53% | 0.6360 | 0.6485 | 0.0135 | M30 bullish FVG [0.6300, 0.6360], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 14 | 2024-05-08 17:00 | 0.6430 | 0.6356 | 2024-05-08 17:00 | 0.6356 | SL | 0 | -1258.37 | -1.25% | 0.6484 | 0.6578 | 0.0148 | M30 bullish FVG [0.6406, 0.6484], stochastic crossed below 20 | FVG stop 0.6356 |
| 15 | 2024-05-30 16:00 | 0.5664 | 0.5568 | 2024-05-31 10:00 | 0.5568 | SL | 7 | -1805.10 | -1.80% | 0.5700 | 0.5857 | 0.0193 | M30 bullish FVG [0.5642, 0.5700], stochastic crossed below 20 | FVG stop 0.5568 |
| 16 | 2024-07-05 23:00 | 0.5016 | 0.4917 | 2024-07-06 18:00 | 0.5066 | STOCH_EXIT | 18 | 881.20 | 0.90% | 0.4992 | 0.5213 | 0.0197 | M30 bullish FVG [0.4942, 0.4992], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 17 | 2024-08-08 14:30 | 0.4440 | 0.4365 | 2024-08-08 18:00 | 0.4365 | SL | 7 | -1781.55 | -1.80% | 0.4486 | 0.4591 | 0.0151 | M30 bullish FVG [0.4444, 0.4486], stochastic crossed below 20 | FVG stop 0.4365 |
| 18 | 2024-08-12 20:30 | 0.4322 | 0.4247 | 2024-08-13 16:00 | 0.4364 | STOCH_EXIT | 20 | 847.28 | 0.87% | 0.4294 | 0.4473 | 0.0151 | M30 bullish FVG [0.4240, 0.4294], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 19 | 2024-09-20 16:00 | 0.3860 | 0.3800 | 2024-09-20 17:00 | 0.3906 | STOCH_EXIT | 2 | 1073.98 | 1.09% | 0.3860 | 0.3980 | 0.0120 | M30 bullish FVG [0.3822, 0.3860], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 20 | 2024-10-25 16:00 | 0.3832 | 0.3796 | 2024-10-25 19:00 | 0.3905 | TP | 6 | 1778.85 | 1.79% | 0.3880 | 0.3905 | 0.0073 | M30 bullish FVG [0.3860, 0.3880], stochastic crossed below 20 | FVG target 0.3905 |
| 21 | 2024-10-28 16:30 | 0.3860 | 0.3821 | 2024-10-28 19:30 | 0.3937 | TP | 6 | 1921.15 | 1.90% | 0.3924 | 0.3937 | 0.0077 | M30 bullish FVG [0.3896, 0.3924], stochastic crossed below 20 | FVG target 0.3937 |
| 22 | 2024-11-04 12:00 | 0.4144 | 0.4086 | 2024-11-04 20:30 | 0.4148 | STOCH_EXIT | 16 | -3.64 | -0.00% | 0.4200 | 0.4259 | 0.0115 | M30 bullish FVG [0.4140, 0.4200], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 23 | 2024-11-11 20:00 | 0.4550 | 0.4490 | 2024-11-11 21:00 | 0.4602 | STOCH_EXIT | 2 | 1071.78 | 1.04% | 0.4516 | 0.4670 | 0.0120 | M30 bullish FVG [0.4446, 0.4516], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 24 | 2024-11-13 19:30 | 0.4712 | 0.4644 | 2024-11-14 11:00 | 0.4718 | STOCH_EXIT | 17 | 28.40 | 0.03% | 0.4630 | 0.4849 | 0.0137 | M30 bullish FVG [0.4542, 0.4630], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 25 | 2024-11-19 12:30 | 0.4790 | 0.4713 | 2024-11-19 17:00 | 0.4713 | SL | 9 | -1772.10 | -1.70% | 0.4922 | 0.4944 | 0.0154 | M30 bullish FVG [0.4878, 0.4922], stochastic crossed below 20 | FVG stop 0.4713 |
| 26 | 2024-12-02 13:30 | 0.4360 | 0.4266 | 2024-12-02 17:30 | 0.4482 | STOCH_EXIT | 8 | 2763.11 | 2.70% | 0.4480 | 0.4548 | 0.0188 | M30 bullish FVG [0.4424, 0.4480], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 27 | 2024-12-20 21:30 | 0.4508 | 0.4420 | 2024-12-21 07:30 | 0.4578 | STOCH_EXIT | 6 | 1525.15 | 1.45% | 0.4402 | 0.4684 | 0.0176 | M30 bullish FVG [0.4348, 0.4402], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 28 | 2024-12-24 10:30 | 0.4726 | 0.4627 | 2024-12-24 15:30 | 0.4738 | STOCH_EXIT | 10 | 164.26 | 0.15% | 0.4792 | 0.4925 | 0.0199 | M30 bullish FVG [0.4746, 0.4792], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 29 | 2025-01-17 23:30 | 0.4952 | 0.4872 | 2025-01-18 09:30 | 0.4996 | STOCH_EXIT | 6 | 842.96 | 0.79% | 0.5002 | 0.5113 | 0.0161 | M30 bullish FVG [0.4944, 0.5002], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 30 | 2025-02-03 20:30 | 0.5188 | 0.5116 | 2025-02-04 10:00 | 0.5288 | STOCH_EXIT | 13 | 1961.57 | 1.83% | 0.5096 | 0.5332 | 0.0144 | M30 bullish FVG [0.5020, 0.5096], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 31 | 2025-02-13 13:30 | 0.5424 | 0.5355 | 2025-02-13 18:30 | 0.5482 | STOCH_EXIT | 10 | 1061.45 | 0.97% | 0.5512 | 0.5561 | 0.0137 | M30 bullish FVG [0.5420, 0.5512], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 32 | 2025-02-17 16:00 | 0.5616 | 0.5540 | 2025-02-17 18:30 | 0.5628 | STOCH_EXIT | 5 | 125.65 | 0.11% | 0.5604 | 0.5768 | 0.0152 | M30 bullish FVG [0.5524, 0.5604], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 33 | 2025-02-26 22:00 | 0.6504 | 0.6401 | 2025-02-27 09:00 | 0.6604 | STOCH_EXIT | 8 | 1588.58 | 1.44% | 0.6654 | 0.6710 | 0.0206 | M30 bullish FVG [0.6600, 0.6654], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 34 | 2025-03-07 16:00 | 0.6484 | 0.6362 | 2025-03-07 17:30 | 0.6362 | SL | 3 | -2217.61 | -1.98% | 0.6520 | 0.6728 | 0.0244 | M30 bullish FVG [0.6450, 0.6520], stochastic crossed below 20 | FVG stop 0.6362 |
| 35 | 2025-03-24 16:00 | 0.6302 | 0.6195 | 2025-03-24 18:00 | 0.6358 | STOCH_EXIT | 4 | 869.23 | 0.79% | 0.6336 | 0.6515 | 0.0213 | M30 bullish FVG [0.6268, 0.6336], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 36 | 2025-05-05 16:00 | 0.5306 | 0.5213 | 2025-05-05 22:30 | 0.5416 | STOCH_EXIT | 13 | 2186.96 | 1.97% | 0.5300 | 0.5492 | 0.0186 | M30 bullish FVG [0.5252, 0.5300], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 37 | 2025-05-27 07:30 | 0.6000 | 0.5905 | 2025-05-27 09:00 | 0.6036 | STOCH_EXIT | 3 | 566.66 | 0.50% | 0.5972 | 0.6190 | 0.0190 | M30 bullish FVG [0.5918, 0.5972], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 38 | 2025-05-27 23:00 | 0.6278 | 0.6183 | 2025-05-28 16:30 | 0.6328 | STOCH_EXIT | 22 | 790.92 | 0.70% | 0.6100 | 0.6468 | 0.0190 | M30 bullish FVG [0.6048, 0.6100], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 39 | 2025-06-09 13:30 | 0.6610 | 0.6524 | 2025-06-09 20:30 | 0.6614 | STOCH_EXIT | 14 | -45.19 | -0.04% | 0.6650 | 0.6783 | 0.0173 | M30 bullish FVG [0.6580, 0.6650], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 40 | 2025-07-11 09:30 | 0.6214 | 0.6107 | 2025-07-11 17:30 | 0.6107 | SL | 16 | -2086.03 | -1.82% | 0.6220 | 0.6428 | 0.0214 | M30 bullish FVG [0.6136, 0.6220], stochastic crossed below 20 | FVG stop 0.6107 |
| 41 | 2025-07-17 17:00 | 0.6466 | 0.6368 | 2025-07-17 23:30 | 0.6490 | STOCH_EXIT | 13 | 304.88 | 0.27% | 0.6576 | 0.6661 | 0.0195 | M30 bullish FVG [0.6484, 0.6576], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 42 | 2025-07-28 19:00 | 0.7066 | 0.6977 | 2025-07-28 22:00 | 0.7106 | STOCH_EXIT | 6 | 523.33 | 0.47% | 0.7088 | 0.7243 | 0.0177 | M30 bullish FVG [0.7010, 0.7088], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 43 | 2025-07-30 20:00 | 0.6888 | 0.6777 | 2025-07-31 09:00 | 0.6902 | STOCH_EXIT | 13 | 116.52 | 0.10% | 0.6940 | 0.7110 | 0.0222 | M30 bullish FVG [0.6856, 0.6940], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 44 | 2025-08-05 11:00 | 0.7398 | 0.7277 | 2025-08-05 15:00 | 0.7277 | SL | 8 | -1969.69 | -1.74% | 0.7388 | 0.7641 | 0.0243 | M30 bullish FVG [0.7288, 0.7388], stochastic crossed below 20 | FVG stop 0.7277 |
| 45 | 2025-09-29 13:30 | 0.7860 | 0.7738 | 2025-09-29 23:30 | 0.7914 | STOCH_EXIT | 20 | 654.80 | 0.59% | 0.7900 | 0.8104 | 0.0244 | M30 bullish FVG [0.7838, 0.7900], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 46 | 2025-10-01 15:30 | 0.8102 | 0.7969 | 2025-10-02 08:00 | 0.7969 | SL | 20 | -1941.65 | -1.74% | 0.7972 | 0.8367 | 0.0265 | M30 bullish FVG [0.7896, 0.7972], stochastic crossed below 20 | FVG stop 0.7969 |
| 47 | 2025-10-07 22:00 | 0.8292 | 0.8154 | 2025-10-08 07:30 | 0.8320 | STOCH_EXIT | 6 | 261.93 | 0.24% | 0.8354 | 0.8567 | 0.0275 | M30 bullish FVG [0.8270, 0.8354], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |
| 48 | 2025-10-09 23:30 | 0.8552 | 0.8412 | 2025-10-10 07:00 | 0.8832 | TP | 2 | 3504.55 | 3.18% | 0.8882 | 0.8832 | 0.0280 | M30 bullish FVG [0.8776, 0.8882], stochastic crossed below 20 | FVG target 0.8832 |
| 49 | 2025-10-16 10:30 | 0.7774 | 0.7518 | 2025-10-16 16:00 | 0.7830 | STOCH_EXIT | 11 | 703.69 | 0.62% | 0.7780 | 0.8285 | 0.0511 | M30 bullish FVG [0.7642, 0.7780], stochastic crossed below 20 | FVG stochastic %K(11) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 97297.63
- Макс: 114929.31
- Финал: 114862.99

Полная кривая — в `*_equity.csv`.
