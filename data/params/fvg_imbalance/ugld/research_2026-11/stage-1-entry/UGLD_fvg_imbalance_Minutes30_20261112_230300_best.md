# Бэктест UGLD (Minutes30)

- Период: 2023-11-12 23:02 — 2025-11-12 23:02
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
| Всего сделок | 78 |
| Выигрышных / проигрышных | 49 / 29 |
| Win rate | 62.82% |
| Gross profit / loss | 56362.14 / 56788.49 |
| Profit factor | 0.992 |
| Чистый PnL | -426.36 (-0.43%) |
| Стартовый / финальный капитал | 100000.00 / 99573.64 |
| Макс. просадка | 24411.91 (21.16%) |
| Средняя прибыль / убыток | 1150.25 / 1958.22 |
| Expectancy | -5.47 |
| Лучшая / худшая сделка | 4985.49 / -5259.79 |
| Exposure | 4.89% |
| CAGR | -0.21% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2024-01-29 18:00 | 0.6433 | 0.6269 | 2024-01-30 10:30 | 0.6607 | STOCH_EXIT | 4 | 2595.94 | 2.60% | 0.6439 | 0.6761 | 0.0164 | M30 bullish FVG [0.6400, 0.6439], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 2 | 2024-01-31 11:00 | 0.6519 | 0.6332 | 2024-01-31 12:30 | 0.6557 | STOCH_EXIT | 3 | 493.95 | 0.48% | 0.6530 | 0.6894 | 0.0187 | M30 bullish FVG [0.6510, 0.6530], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 3 | 2024-02-01 18:00 | 0.6790 | 0.6603 | 2024-02-02 16:00 | 0.6837 | STOCH_EXIT | 15 | 606.82 | 0.59% | 0.6755 | 0.7165 | 0.0187 | M30 bullish FVG [0.6640, 0.6755], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 4 | 2024-02-14 14:30 | 0.7465 | 0.7223 | 2024-02-14 17:30 | 0.7223 | SL | 6 | -3438.74 | -3.34% | 0.7425 | 0.7949 | 0.0242 | M30 bullish FVG [0.7398, 0.7425], stochastic crossed below 20 | FVG stop 0.7223 |
| 5 | 2024-02-15 17:30 | 0.7490 | 0.7226 | 2024-02-16 09:30 | 0.7560 | STOCH_EXIT | 3 | 830.92 | 0.83% | 0.7363 | 0.8019 | 0.0264 | M30 bullish FVG [0.7323, 0.7363], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 6 | 2024-02-22 16:30 | 0.7851 | 0.7530 | 2024-02-26 10:00 | 0.7530 | SL | 6 | -4204.45 | -4.18% | 0.7747 | 0.8493 | 0.0321 | M30 bullish FVG [0.7642, 0.7747], stochastic crossed below 20 | FVG stop 0.7530 |
| 7 | 2024-03-01 15:30 | 0.7537 | 0.7204 | 2024-03-01 18:30 | 0.7646 | STOCH_EXIT | 6 | 1298.03 | 1.34% | 0.7452 | 0.8203 | 0.0333 | M30 bullish FVG [0.7399, 0.7452], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 8 | 2024-03-05 11:00 | 0.7817 | 0.7492 | 2024-03-05 17:00 | 0.7918 | STOCH_EXIT | 12 | 1164.16 | 1.19% | 0.7750 | 0.8467 | 0.0325 | M30 bullish FVG [0.7676, 0.7750], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 9 | 2024-03-18 10:30 | 0.7700 | 0.7438 | 2024-03-19 11:00 | 0.7721 | STOCH_EXIT | 20 | 170.11 | 0.17% | 0.7748 | 0.8224 | 0.0262 | M30 bullish FVG [0.7624, 0.7748], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 10 | 2024-03-21 15:30 | 0.7763 | 0.7512 | 2024-03-21 18:30 | 0.7846 | STOCH_EXIT | 6 | 962.50 | 0.97% | 0.7584 | 0.8264 | 0.0251 | M30 bullish FVG [0.7556, 0.7584], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 11 | 2024-04-02 14:30 | 0.9384 | 0.9051 | 2024-04-03 09:30 | 0.9600 | STOCH_EXIT | 9 | 2209.64 | 2.20% | 0.9440 | 1.0050 | 0.0333 | M30 bullish FVG [0.9352, 0.9440], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 12 | 2024-04-09 16:00 | 0.9977 | 0.9627 | 2024-04-10 10:30 | 1.0476 | STOCH_EXIT | 8 | 4985.49 | 4.90% | 0.9971 | 1.0678 | 0.0350 | M30 bullish FVG [0.9915, 0.9971], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 13 | 2024-04-11 17:30 | 1.0840 | 1.0417 | 2024-04-11 18:00 | 1.0417 | SL | 1 | -4292.30 | -4.00% | 1.1020 | 1.1686 | 0.0423 | M30 bullish FVG [1.0649, 1.1020], stochastic crossed below 20 | FVG stop 1.0417 |
| 14 | 2024-04-19 17:00 | 1.0656 | 1.0155 | 2024-04-19 18:30 | 1.0688 | STOCH_EXIT | 3 | 204.75 | 0.20% | 1.0600 | 1.1658 | 0.0501 | M30 bullish FVG [1.0546, 1.0600], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 15 | 2024-05-07 11:00 | 0.9274 | 0.8838 | 2024-05-07 17:00 | 0.9410 | STOCH_EXIT | 12 | 1405.90 | 1.37% | 0.9373 | 1.0146 | 0.0436 | M30 bullish FVG [0.9200, 0.9373], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 16 | 2024-05-15 15:00 | 0.9931 | 0.9523 | 2024-05-15 18:00 | 1.0037 | STOCH_EXIT | 6 | 1008.17 | 0.97% | 0.9901 | 1.0748 | 0.0408 | M30 bullish FVG [0.9841, 0.9901], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 17 | 2024-06-17 12:00 | 0.9195 | 0.8793 | 2024-06-17 19:00 | 0.9153 | STOCH_EXIT | 14 | -588.50 | -0.56% | 0.9231 | 0.9998 | 0.0402 | M30 bullish FVG [0.9165, 0.9231], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 18 | 2024-06-21 12:00 | 0.8622 | 0.8219 | 2024-06-22 15:30 | 0.8624 | STOCH_EXIT | 35 | -80.80 | -0.08% | 0.8576 | 0.9427 | 0.0403 | M30 bullish FVG [0.8535, 0.8576], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 19 | 2024-07-02 17:30 | 0.8585 | 0.8229 | 2024-07-02 19:30 | 0.8645 | STOCH_EXIT | 4 | 626.90 | 0.60% | 0.8644 | 0.9297 | 0.0356 | M30 bullish FVG [0.8556, 0.8644], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 20 | 2024-07-10 16:30 | 0.8058 | 0.7727 | 2024-07-11 10:30 | 0.8206 | STOCH_EXIT | 17 | 1832.27 | 1.73% | 0.8063 | 0.8720 | 0.0331 | M30 bullish FVG [0.8029, 0.8063], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 21 | 2024-07-12 11:00 | 0.8321 | 0.7964 | 2024-07-12 20:30 | 0.8321 | STOCH_EXIT | 19 | -107.34 | -0.10% | 0.8259 | 0.9036 | 0.0357 | M30 bullish FVG [0.8218, 0.8259], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 22 | 2024-07-15 23:30 | 0.8291 | 0.7929 | 2024-07-16 10:30 | 0.8527 | STOCH_EXIT | 3 | 2935.92 | 2.74% | 0.8250 | 0.9014 | 0.0362 | M30 bullish FVG [0.8190, 0.8250], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 23 | 2024-07-31 19:30 | 0.8394 | 0.8085 | 2024-07-31 22:00 | 0.8444 | STOCH_EXIT | 5 | 544.71 | 0.50% | 0.8400 | 0.9012 | 0.0309 | M30 bullish FVG [0.8342, 0.8400], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 24 | 2024-08-01 14:00 | 0.8516 | 0.8213 | 2024-08-01 16:00 | 0.8608 | STOCH_EXIT | 4 | 1084.69 | 0.98% | 0.8607 | 0.9123 | 0.0303 | M30 bullish FVG [0.8499, 0.8607], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 25 | 2024-08-22 16:00 | 0.7539 | 0.7245 | 2024-08-23 09:30 | 0.7614 | STOCH_EXIT | 16 | 997.87 | 0.89% | 0.7481 | 0.8126 | 0.0294 | M30 bullish FVG [0.7410, 0.7481], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 26 | 2024-09-05 16:30 | 0.6534 | 0.6144 | 2024-09-05 23:00 | 0.6364 | STOCH_EXIT | 13 | -3052.57 | -2.70% | 0.6745 | 0.7314 | 0.0390 | M30 bullish FVG [0.6476, 0.6745], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 27 | 2024-09-09 16:30 | 0.6428 | 0.6011 | 2024-09-09 18:30 | 0.6641 | STOCH_EXIT | 4 | 3530.56 | 3.21% | 0.6432 | 0.7263 | 0.0417 | M30 bullish FVG [0.6386, 0.6432], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 28 | 2024-09-19 16:00 | 0.6987 | 0.6606 | 2024-09-19 19:30 | 0.7095 | STOCH_EXIT | 7 | 1635.54 | 1.44% | 0.6930 | 0.7749 | 0.0381 | M30 bullish FVG [0.6845, 0.6930], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 29 | 2024-09-27 08:30 | 0.7715 | 0.7327 | 2024-09-28 13:00 | 0.7507 | STOCH_EXIT | 43 | -3212.60 | -2.79% | 0.7677 | 0.8492 | 0.0388 | M30 bullish FVG [0.7428, 0.7677], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 30 | 2024-10-04 13:00 | 0.7032 | 0.6659 | 2024-10-04 20:00 | 0.7044 | STOCH_EXIT | 14 | 78.90 | 0.07% | 0.7027 | 0.7779 | 0.0373 | M30 bullish FVG [0.6987, 0.7027], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 31 | 2024-10-15 18:30 | 0.6773 | 0.6477 | 2024-10-15 23:00 | 0.6794 | STOCH_EXIT | 9 | 234.57 | 0.21% | 0.6821 | 0.7365 | 0.0296 | M30 bullish FVG [0.6755, 0.6821], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 32 | 2024-10-16 18:00 | 0.6908 | 0.6613 | 2024-10-17 08:30 | 0.6948 | STOCH_EXIT | 15 | 535.77 | 0.48% | 0.7000 | 0.7499 | 0.0295 | M30 bullish FVG [0.6900, 0.7000], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 33 | 2024-10-22 20:30 | 0.7068 | 0.6774 | 2024-10-23 09:30 | 0.7125 | STOCH_EXIT | 12 | 793.47 | 0.71% | 0.7100 | 0.7656 | 0.0294 | M30 bullish FVG [0.7018, 0.7100], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 34 | 2024-10-23 17:30 | 0.7090 | 0.6800 | 2024-10-24 16:30 | 0.6885 | STOCH_EXIT | 32 | -3391.80 | -2.99% | 0.7110 | 0.7670 | 0.0290 | M30 bullish FVG [0.7071, 0.7110], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 35 | 2024-11-13 10:00 | 0.6714 | 0.6402 | 2024-11-14 07:00 | 0.6402 | SL | 28 | -5231.16 | -4.75% | 0.6762 | 0.7339 | 0.0312 | M30 bullish FVG [0.6712, 0.6762], stochastic crossed below 20 | FVG stop 0.6402 |
| 36 | 2024-11-15 23:00 | 0.6647 | 0.6345 | 2024-11-16 00:00 | 0.6670 | STOCH_EXIT | 2 | 258.20 | 0.25% | 0.6680 | 0.7250 | 0.0302 | M30 bullish FVG [0.6630, 0.6680], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 37 | 2024-11-18 15:30 | 0.6589 | 0.6285 | 2024-11-18 18:30 | 0.6681 | STOCH_EXIT | 6 | 1357.30 | 1.29% | 0.6610 | 0.7196 | 0.0304 | M30 bullish FVG [0.6574, 0.6610], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 38 | 2024-11-27 17:30 | 0.6555 | 0.6250 | 2024-11-28 19:00 | 0.6494 | STOCH_EXIT | 37 | -1093.90 | -1.03% | 0.6659 | 0.7165 | 0.0305 | M30 bullish FVG [0.6477, 0.6659], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 39 | 2024-12-06 10:30 | 0.6285 | 0.5996 | 2024-12-06 18:30 | 0.6310 | STOCH_EXIT | 16 | 314.20 | 0.30% | 0.6292 | 0.6862 | 0.0289 | M30 bullish FVG [0.6262, 0.6292], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 40 | 2024-12-12 08:00 | 0.6523 | 0.6261 | 2024-12-13 10:00 | 0.6417 | STOCH_EXIT | 38 | -1822.01 | -1.72% | 0.6505 | 0.7046 | 0.0262 | M30 bullish FVG [0.6464, 0.6505], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 41 | 2024-12-20 10:00 | 0.6798 | 0.6557 | 2024-12-20 10:00 | 0.6870 | STOCH_EXIT | 0 | 997.04 | 0.96% | 0.6753 | 0.7279 | 0.0241 | M30 bullish FVG [0.6726, 0.6753], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 42 | 2024-12-20 20:00 | 0.7254 | 0.7013 | 2024-12-20 23:00 | 0.7398 | STOCH_EXIT | 6 | 1968.11 | 1.88% | 0.7302 | 0.7735 | 0.0241 | M30 bullish FVG [0.6929, 0.7302], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 43 | 2024-12-24 10:30 | 0.7842 | 0.7526 | 2024-12-24 17:00 | 0.7526 | SL | 13 | -4405.29 | -4.13% | 0.7980 | 0.8474 | 0.0316 | M30 bullish FVG [0.7510, 0.7980], stochastic crossed below 20 | FVG stop 0.7526 |
| 44 | 2025-01-14 19:00 | 0.7750 | 0.7448 | 2025-01-14 21:00 | 0.7749 | STOCH_EXIT | 4 | -115.49 | -0.11% | 0.7720 | 0.8353 | 0.0302 | M30 bullish FVG [0.7669, 0.7720], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 45 | 2025-01-15 21:00 | 0.7817 | 0.7519 | 2025-01-16 07:00 | 0.7877 | STOCH_EXIT | 6 | 683.20 | 0.67% | 0.7854 | 0.8413 | 0.0298 | M30 bullish FVG [0.7811, 0.7854], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 46 | 2025-01-16 17:00 | 0.7946 | 0.7653 | 2025-01-16 21:00 | 0.7987 | STOCH_EXIT | 8 | 426.13 | 0.42% | 0.7927 | 0.8532 | 0.0293 | M30 bullish FVG [0.7896, 0.7927], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 47 | 2025-01-21 15:30 | 0.8083 | 0.7780 | 2025-01-21 16:00 | 0.8189 | STOCH_EXIT | 1 | 1252.66 | 1.21% | 0.8059 | 0.8688 | 0.0303 | M30 bullish FVG [0.8023, 0.8059], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 48 | 2025-02-03 22:30 | 0.8665 | 0.8366 | 2025-02-04 15:30 | 0.8715 | STOCH_EXIT | 21 | 499.85 | 0.48% | 0.8687 | 0.9262 | 0.0299 | M30 bullish FVG [0.8639, 0.8687], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 49 | 2025-02-06 11:30 | 0.8993 | 0.8694 | 2025-02-06 14:00 | 0.8694 | SL | 5 | -3603.15 | -3.42% | 0.9106 | 0.9591 | 0.0299 | M30 bullish FVG [0.8988, 0.9106], stochastic crossed below 20 | FVG stop 0.8694 |
| 50 | 2025-02-13 10:30 | 0.7798 | 0.7425 | 2025-02-14 07:00 | 0.7401 | SL | 21 | -5259.79 | -5.19% | 0.7639 | 0.8544 | 0.0373 | M30 bullish FVG [0.7600, 0.7639], stochastic crossed below 20 | FVG stop 0.7425 |
| 51 | 2025-02-20 17:00 | 0.7743 | 0.7368 | 2025-02-21 07:00 | 0.7728 | STOCH_EXIT | 15 | -281.92 | -0.29% | 0.7745 | 0.8492 | 0.0375 | M30 bullish FVG [0.7699, 0.7745], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 52 | 2025-03-04 23:30 | 0.7387 | 0.7060 | 2025-03-05 09:00 | 0.7440 | STOCH_EXIT | 6 | 592.62 | 0.62% | 0.7390 | 0.8042 | 0.0327 | M30 bullish FVG [0.7347, 0.7390], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 53 | 2025-03-05 22:30 | 0.7597 | 0.7267 | 2025-03-06 08:00 | 0.7663 | STOCH_EXIT | 6 | 741.30 | 0.77% | 0.7680 | 0.8258 | 0.0330 | M30 bullish FVG [0.7498, 0.7680], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 54 | 2025-03-10 17:30 | 0.7417 | 0.7079 | 2025-03-11 07:00 | 0.7414 | STOCH_EXIT | 14 | -136.44 | -0.14% | 0.7411 | 0.8094 | 0.0338 | M30 bullish FVG [0.7365, 0.7411], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 55 | 2025-03-20 13:00 | 0.7669 | 0.7393 | 2025-03-22 05:00 | 0.7525 | STOCH_EXIT | 63 | -1925.28 | -1.98% | 0.7693 | 0.8221 | 0.0276 | M30 bullish FVG [0.7662, 0.7693], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 56 | 2025-03-26 11:30 | 0.7411 | 0.7153 | 2025-03-26 19:30 | 0.7153 | SL | 16 | -3395.35 | -3.58% | 0.7427 | 0.7927 | 0.0258 | M30 bullish FVG [0.7399, 0.7427], stochastic crossed below 20 | FVG stop 0.7153 |
| 57 | 2025-04-01 14:30 | 0.7365 | 0.7045 | 2025-04-01 17:30 | 0.7446 | STOCH_EXIT | 6 | 919.93 | 1.00% | 0.7423 | 0.8006 | 0.0320 | M30 bullish FVG [0.7349, 0.7423], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 58 | 2025-04-07 22:00 | 0.6674 | 0.6339 | 2025-04-08 06:30 | 0.6793 | STOCH_EXIT | 4 | 1560.50 | 1.68% | 0.6644 | 0.7343 | 0.0335 | M30 bullish FVG [0.6575, 0.6644], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 59 | 2025-04-25 13:30 | 0.6859 | 0.6543 | 2025-04-25 21:00 | 0.6849 | STOCH_EXIT | 15 | -230.90 | -0.25% | 0.6913 | 0.7490 | 0.0316 | M30 bullish FVG [0.6851, 0.6913], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 60 | 2025-05-27 16:30 | 0.5945 | 0.5737 | 2025-05-27 17:30 | 0.6018 | STOCH_EXIT | 2 | 1058.89 | 1.13% | 0.5821 | 0.6361 | 0.0208 | M30 bullish FVG [0.5800, 0.5821], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 61 | 2025-06-02 22:30 | 0.6340 | 0.6122 | 2025-06-03 07:00 | 0.6472 | STOCH_EXIT | 4 | 1883.91 | 1.98% | 0.6206 | 0.6776 | 0.0218 | M30 bullish FVG [0.6180, 0.6206], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 62 | 2025-06-05 18:30 | 0.6691 | 0.6464 | 2025-06-06 07:00 | 0.6797 | STOCH_EXIT | 12 | 1439.21 | 1.48% | 0.6663 | 0.7144 | 0.0227 | M30 bullish FVG [0.6600, 0.6663], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 63 | 2025-06-06 11:30 | 0.6740 | 0.6512 | 2025-06-06 17:30 | 0.6512 | SL | 12 | -3423.94 | -3.48% | 0.6746 | 0.7196 | 0.0228 | M30 bullish FVG [0.6712, 0.6746], stochastic crossed below 20 | FVG stop 0.6512 |
| 64 | 2025-06-10 10:30 | 0.6398 | 0.6148 | 2025-06-10 17:30 | 0.6360 | STOCH_EXIT | 14 | -661.25 | -0.69% | 0.6340 | 0.6898 | 0.0250 | M30 bullish FVG [0.6312, 0.6340], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 65 | 2025-07-01 22:00 | 0.6721 | 0.6530 | 2025-07-02 06:30 | 0.6771 | STOCH_EXIT | 4 | 605.56 | 0.64% | 0.6712 | 0.7103 | 0.0191 | M30 bullish FVG [0.6647, 0.6712], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 66 | 2025-07-16 15:00 | 0.5910 | 0.5530 | 2025-07-16 17:00 | 0.6106 | STOCH_EXIT | 4 | 3058.87 | 3.21% | 0.5900 | 0.6671 | 0.0380 | M30 bullish FVG [0.5850, 0.5900], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 67 | 2025-07-22 20:00 | 0.5883 | 0.5529 | 2025-07-22 23:30 | 0.5899 | STOCH_EXIT | 7 | 168.82 | 0.17% | 0.5876 | 0.6592 | 0.0354 | M30 bullish FVG [0.5837, 0.5876], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 68 | 2025-08-08 22:30 | 0.6176 | 0.5936 | 2025-08-11 07:00 | 0.6319 | STOCH_EXIT | 4 | 2174.36 | 2.21% | 0.6192 | 0.6656 | 0.0240 | M30 bullish FVG [0.6117, 0.6192], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 69 | 2025-08-12 18:30 | 0.6109 | 0.5844 | 2025-08-12 23:30 | 0.6132 | STOCH_EXIT | 10 | 276.82 | 0.28% | 0.6129 | 0.6639 | 0.0265 | M30 bullish FVG [0.6097, 0.6129], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 70 | 2025-08-25 21:30 | 0.5870 | 0.5661 | 2025-08-26 06:30 | 0.5870 | STOCH_EXIT | 5 | -100.96 | -0.10% | 0.5880 | 0.6287 | 0.0209 | M30 bullish FVG [0.5859, 0.5880], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 71 | 2025-09-09 18:00 | 0.6172 | 0.6003 | 2025-09-10 18:00 | 0.6171 | STOCH_EXIT | 35 | -116.90 | -0.12% | 0.6101 | 0.6509 | 0.0169 | M30 bullish FVG [0.6068, 0.6101], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 72 | 2025-09-12 11:30 | 0.6336 | 0.6146 | 2025-09-12 23:00 | 0.6283 | STOCH_EXIT | 23 | -943.02 | -0.94% | 0.6414 | 0.6716 | 0.0190 | M30 bullish FVG [0.6335, 0.6414], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 73 | 2025-09-24 20:00 | 0.6210 | 0.5997 | 2025-09-24 23:00 | 0.6234 | STOCH_EXIT | 6 | 284.45 | 0.29% | 0.6235 | 0.6636 | 0.0213 | M30 bullish FVG [0.6207, 0.6235], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 74 | 2025-10-09 11:30 | 0.5198 | 0.4923 | 2025-10-09 23:30 | 0.5222 | STOCH_EXIT | 24 | 360.77 | 0.36% | 0.5142 | 0.5748 | 0.0275 | M30 bullish FVG [0.5040, 0.5142], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 75 | 2025-10-17 11:00 | 0.5178 | 0.4925 | 2025-10-17 15:30 | 0.5179 | STOCH_EXIT | 9 | -81.06 | -0.08% | 0.5182 | 0.5685 | 0.0253 | M30 bullish FVG [0.5151, 0.5182], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 76 | 2025-10-22 16:00 | 0.5211 | 0.4979 | 2025-10-23 16:00 | 0.5145 | STOCH_EXIT | 35 | -1366.62 | -1.37% | 0.5191 | 0.5675 | 0.0232 | M30 bullish FVG [0.5147, 0.5191], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 77 | 2025-11-06 20:00 | 0.4755 | 0.4576 | 2025-11-07 10:00 | 0.4794 | STOCH_EXIT | 15 | 711.89 | 0.72% | 0.4751 | 0.5112 | 0.0179 | M30 bullish FVG [0.4729, 0.4751], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |
| 78 | 2025-11-07 16:00 | 0.4767 | 0.4595 | 2025-11-07 22:00 | 0.4761 | STOCH_EXIT | 12 | -224.97 | -0.23% | 0.4788 | 0.5112 | 0.0172 | M30 bullish FVG [0.4760, 0.4788], stochastic crossed below 20 | FVG stochastic %K(10) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 90948.23
- Макс: 115360.14
- Финал: 99573.64

Полная кривая — в `*_equity.csv`.
