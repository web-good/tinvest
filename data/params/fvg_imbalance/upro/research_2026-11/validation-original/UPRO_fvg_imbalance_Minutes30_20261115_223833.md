# Бэктест UPRO (Minutes30)

- Период: 2025-11-15 22:38 — 2026-11-15 22:38
- Стартовый кэш: 100000.00
- Fraction: 1; Commission: 0.0005

## Параметры стратегии

| Параметр | Значение |
|---|---|
| ATRPeriod | 14 |
| MinGapATR | 0.25 |
| MaxWaitBars | 30 |
| StochPeriod | 14 |
| UseStochExit | 1 |
| StopATR | 1 |
| TakeProfitATR | 2 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 0 |

## Сводка метрик

| Метрика | Значение |
|---|---|
| Всего сделок | 23 |
| Выигрышных / проигрышных | 15 / 8 |
| Win rate | 65.22% |
| Gross profit / loss | 11075.15 / 8847.08 |
| Profit factor | 1.252 |
| Чистый PnL | 2228.08 (2.23%) |
| Стартовый / финальный капитал | 100000.00 / 102228.08 |
| Макс. просадка | 9397.90 (8.75%) |
| Средняя прибыль / убыток | 738.34 / 1105.88 |
| Expectancy | 96.87 |
| Лучшая / худшая сделка | 2109.60 / -2645.12 |
| Exposure | 3.30% |
| CAGR | 2.23% |

## Журнал сделок

| № | Вход | Цена входа | Стоп | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2025-11-21 12:00 | 1.4460 | 1.3988 | 2025-11-21 17:30 | 1.4590 | STOCH_EXIT | 11 | 796.78 | 0.80% | 1.4580 | 1.5403 | 0.0472 | M30 bullish FVG [1.4460, 1.4580], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 2 | 2025-12-10 12:00 | 1.4970 | 1.4498 | 2025-12-10 19:30 | 1.5050 | STOCH_EXIT | 15 | 435.43 | 0.43% | 1.5020 | 1.5913 | 0.0472 | M30 bullish FVG [1.4900, 1.5020], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 3 | 2025-12-12 09:30 | 1.5080 | 1.4628 | 2025-12-12 11:00 | 1.5180 | STOCH_EXIT | 3 | 568.63 | 0.56% | 1.5210 | 1.5984 | 0.0452 | M30 bullish FVG [1.5080, 1.5210], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 4 | 2025-12-29 10:30 | 1.4970 | 1.4577 | 2025-12-29 11:30 | 1.5300 | STOCH_EXIT | 2 | 2109.60 | 2.10% | 1.5140 | 1.5755 | 0.0393 | M30 bullish FVG [1.5040, 1.5140], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 5 | 2026-01-14 19:30 | 1.4430 | 1.4087 | 2026-01-15 09:30 | 1.4440 | STOCH_EXIT | 14 | -31.49 | -0.03% | 1.4510 | 1.5115 | 0.0343 | M30 bullish FVG [1.4410, 1.4510], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 6 | 2026-01-23 08:00 | 1.5550 | 1.5181 | 2026-01-23 13:30 | 1.5720 | STOCH_EXIT | 11 | 1018.81 | 0.99% | 1.5650 | 1.6288 | 0.0369 | M30 bullish FVG [1.5220, 1.5650], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 7 | 2026-01-28 10:30 | 1.6260 | 1.5835 | 2026-01-28 20:30 | 1.6310 | STOCH_EXIT | 20 | 215.78 | 0.21% | 1.6030 | 1.7111 | 0.0425 | M30 bullish FVG [1.5710, 1.6030], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 8 | 2026-02-18 07:30 | 1.6265 | 1.5813 | 2026-02-19 07:30 | 1.6360 | STOCH_EXIT | 35 | 503.60 | 0.48% | 1.6405 | 1.7169 | 0.0452 | M30 bullish FVG [1.6225, 1.6405], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 9 | 2026-02-19 14:30 | 1.6350 | 1.5882 | 2026-02-20 18:30 | 1.6185 | STOCH_EXIT | 42 | -1160.11 | -1.11% | 1.6340 | 1.7285 | 0.0468 | M30 bullish FVG [1.6185, 1.6340], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 10 | 2026-03-25 19:30 | 1.4870 | 1.4517 | 2026-03-26 07:00 | 1.4990 | STOCH_EXIT | 10 | 735.49 | 0.71% | 1.4970 | 1.5576 | 0.0353 | M30 bullish FVG [1.4830, 1.4970], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 11 | 2026-03-30 17:30 | 1.4360 | 1.3997 | 2026-03-31 08:00 | 1.4450 | STOCH_EXIT | 15 | 551.84 | 0.53% | 1.4525 | 1.5086 | 0.0363 | M30 bullish FVG [1.4400, 1.4525], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 12 | 2026-04-10 11:30 | 1.4350 | 1.4007 | 2026-04-10 14:00 | 1.4540 | STOCH_EXIT | 5 | 1281.55 | 1.22% | 1.4515 | 1.5035 | 0.0343 | M30 bullish FVG [1.4315, 1.4515], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 13 | 2026-04-20 12:00 | 1.4215 | 1.3887 | 2026-04-20 17:30 | 1.4260 | STOCH_EXIT | 11 | 230.72 | 0.22% | 1.4285 | 1.4871 | 0.0328 | M30 bullish FVG [1.4200, 1.4285], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 14 | 2026-04-22 07:30 | 1.4295 | 1.3975 | 2026-04-22 10:00 | 1.4335 | STOCH_EXIT | 5 | 190.07 | 0.18% | 1.4315 | 1.4936 | 0.0320 | M30 bullish FVG [1.4155, 1.4315], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 15 | 2026-05-07 21:30 | 1.3770 | 1.3490 | 2026-05-08 18:00 | 1.3490 | SL | 27 | -2257.43 | -2.13% | 1.3805 | 1.4329 | 0.0280 | M30 bullish FVG [1.3695, 1.3805], stochastic crossed below 20 | FVG stop 1.3490 |
| 16 | 2026-05-13 07:00 | 1.4300 | 1.3999 | 2026-05-13 07:00 | 1.4305 | STOCH_EXIT | 0 | -67.91 | -0.07% | 1.4190 | 1.4902 | 0.0301 | M30 bullish FVG [1.4110, 1.4190], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 17 | 2026-05-14 14:30 | 1.4115 | 1.3816 | 2026-05-15 10:00 | 1.3995 | STOCH_EXIT | 25 | -992.01 | -0.95% | 1.4260 | 1.4713 | 0.0299 | M30 bullish FVG [1.4120, 1.4260], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 18 | 2026-06-11 15:00 | 1.1665 | 1.1287 | 2026-06-12 02:30 | 1.1785 | STOCH_EXIT | 18 | 963.65 | 0.93% | 1.1570 | 1.2421 | 0.0378 | M30 bullish FVG [1.1475, 1.1570], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 19 | 2026-06-16 17:00 | 1.1810 | 1.1408 | 2026-06-18 20:30 | 1.1630 | STOCH_EXIT | 76 | -1687.14 | -1.62% | 1.1910 | 1.2613 | 0.0402 | M30 bullish FVG [1.1810, 1.1910], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 20 | 2026-07-27 22:30 | 1.0600 | 1.0031 | 2026-07-28 15:00 | 1.0610 | STOCH_EXIT | 20 | -5.87 | -0.01% | 1.0135 | 1.1738 | 0.0569 | M30 bullish FVG [0.9575, 1.0135], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 21 | 2026-08-26 07:30 | 0.9660 | 0.9038 | 2026-08-27 11:00 | 0.9420 | STOCH_EXIT | 42 | -2645.12 | -2.58% | 0.9815 | 1.0903 | 0.0622 | M30 bullish FVG [0.9465, 0.9815], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 22 | 2026-09-01 16:30 | 0.9875 | 0.9246 | 2026-09-02 07:30 | 0.9935 | STOCH_EXIT | 17 | 505.96 | 0.51% | 1.0750 | 1.1132 | 0.0629 | M30 bullish FVG [1.0435, 1.0750], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |
| 23 | 2026-10-13 19:00 | 1.1260 | 1.0863 | 2026-10-13 20:00 | 1.1380 | STOCH_EXIT | 2 | 967.25 | 0.96% | 1.1200 | 1.2054 | 0.0397 | M30 bullish FVG [1.1040, 1.1200], stochastic crossed below 20 | FVG stochastic %K(14) crossed above 80 |

## Движение капитала

- Старт: 100000.00
- Мин: 98048.79
- Макс: 107446.69
- Финал: 102228.08

Полная кривая — в `*_equity.csv`.
