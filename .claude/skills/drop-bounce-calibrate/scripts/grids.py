"""Grid files for drop_bounce calibration of one ticker on one timeframe.

Every grid pins all Params fields explicitly: a registered ticker's ParseParams layers JSON over
its own literal, so an unlisted field would silently come from the old literal.

Usage (from the repo root):
  grids.py sweep    <ticker> <tf>                 -> sweep_<tf>.json (one-phase grid over SWEEP)
  grids.py point    <ticker> <tf> <calib.md> [k]  -> point_<tf>.json (flat Params of combo #k, default 1)
  grids.py defaults <ticker> <tf>                 -> point_<tf>.json (core defaults)
<tf> must be Minutes30: drop_bounce runs on 30m only (owner, 2026-10-14: hourly signal, 30m MACD).
"""
import json, os, re, sys

DEFAULTS = dict(RSIPeriod=6, RSIZone=30, RSICap=50, ZoneEntry=1, EMAPeriod=200, StopDailyATR=1.0,
                BreakevenATR=0.5, TrailATR=0.0,
                DailyATRPeriod=14, RSIExitPeriod=10, RSIExit=70,
                MACDFast=12, MACDSlow=26, MACDSignal=9, ConfirmBars=3, FridayTo=11, StaleBars=0, StaleATR=0.3,
                PrevCloseGate=0, PrevCloseATR=0.0, RSIFloor=0.0, SkipMonday=0,
                EMAGapATR=0.0, MACDGreenOnly=0, BreakevenPct=0.1,
                MACDConfirm=1, ConfirmMoveBars=0, ConfirmMoveATR=0.1)
# Hold-out scheme (owner, 2026-10-14): one in-sample sweep on the first 30 of 36 months, the point is
# the centre of the best plateau, the last 6 months are opened once. Axes (owner, 2026-10-14; rules
# on 30m: hourly RSI signal, 30m MACD confirmation, hourly RSI(RSIExitPeriod) exit): the entry RSI
# length, the exit RSI length and the stop as a share of the daily ATR or a whole one — 3 × 5 × 3 = 45.
# Fixed at the defaults: MACD 12/26/9, ConfirmBars 3, Friday buys before 11:00 (FridayTo 11),
# the real-drop gate off (PrevCloseGate 0: a probe on 10 tickers helped 4 of them, owner decides),
# the entry filters off (RSIFloor 0, SkipMonday 0: better on the dev window, worse on the pooled hold-out).
# The previous axes (EMAPeriod × StaleBars × StopDailyATR) belonged to the rules with the day-range
# gate and the MACD exit. Do not change without a request.
SWEEP = {"RSIPeriod": [4, 5, 6], "RSIExitPeriod": [8, 9, 10, 11, 12], "StopDailyATR": [0.5, 0.7, 1.0]}
HOLDOUT_MONTHS = 6


def path(t, name):
    d = f"data/params/drop_bounce/{t.lower()}"
    os.makedirs(d, exist_ok=True)
    return f"{d}/{name}.json"


def dump(p, doc):
    with open(p, "w") as f:
        json.dump(doc, f, ensure_ascii=False, indent=2)
        f.write("\n")
    print(p)


def cmd(t, tf, name):
    return (f"reports/drop_bounce_cal/backtest -ticker {t} -strategy drop_bounce -interval {tf} "
            f"-calibrate data/params/drop_bounce/{t.lower()}/{name}.json -months 36 -holdout-months {HOLDOUT_MONTHS} "
            f"-metric profit_factor -risk-pct 1 -min-trades 1 -combos 0")


def parse_combo(md, k):
    s = open(md).read()
    m = re.search(rf"^### #{k} .*?(?=^### #|\Z)", s, re.M | re.S)
    if not m:
        sys.exit(f"combo #{k} not found in {md}")
    p = {name: float(v) for name, v in re.findall(r"^\| (\w+) \| ([-\d.]+) \|", m.group(0), re.M)}
    return {k2: (int(v) if v.is_integer() else v) for k2, v in p.items() if k2 in DEFAULTS}


def main():
    if len(sys.argv) < 4:
        sys.exit(__doc__)
    kind, t, tf = sys.argv[1], sys.argv[2].upper(), sys.argv[3]
    if tf != "Minutes30":
        sys.exit("калибруется только Minutes30: drop_bounce работает только на 30m (решение владельца 2026-10-14)")
    low = tf.lower()
    if kind == "sweep":
        g = {k: [v] for k, v in DEFAULTS.items()}
        g.update(SWEEP)
        axes = ", ".join(f"{k} {v}" for k, v in SWEEP.items())
        dump(path(t, f"sweep_{low}"), {
            "_comment": f"drop_bounce {t} {tf}: in-sample перебор на первых 30 из 36 месяцев (последние {HOLDOUT_MONTHS} — hold-out), оси {axes}, остальное — дефолты ядра, все поля прибиты. Запуск: {cmd(t, tf, f'sweep_{low}')}. РЕЗУЛЬТАТ: прогон не делался.",
            "phases": [{"name": "sweep", "grid": g}]})
    elif kind == "point":
        k = int(sys.argv[5]) if len(sys.argv) > 5 else 1
        p = dict(DEFAULTS)
        p.update(parse_combo(sys.argv[4], k))
        if p["StopDailyATR"] <= 0:
            sys.exit("точка без стопа — недопустимо")
        dump(path(t, f"point_{low}"), p)
    elif kind == "defaults":
        dump(path(t, f"point_{low}"), dict(DEFAULTS))
    else:
        sys.exit(__doc__)


if __name__ == "__main__":
    main()
