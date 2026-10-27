"""Grid files for drop_bounce calibration of one ticker on one timeframe.

Every grid pins all Params fields explicitly: a registered ticker's ParseParams layers JSON over
its own literal, so an unlisted field would silently come from the old literal.

Usage (from the repo root):
  grids.py macd     <ticker> <tf>                 -> macd_<tf>_<pair>.json, one per MACD pair (HOUR_MACD × M15_MACD),
                                                   one run each at the core defaults; prints the run commands
  grids.py sweep    <ticker> <tf> <pair>          -> sweep_<tf>.json (one-phase grid over SWEEP, MACD pinned to
                                                   <pair>, e.g. h8-26-5_m12-26-9 — the one report.py macd picked)
  grids.py point    <ticker> <tf> <calib.md> [k]  -> point_<tf>.json (flat Params of combo #k, default 1)
  grids.py cand     <ticker> <tf> <calib.md> <k>  -> cand<k>_<tf>.json (a tie-break candidate, see report.py center)
  grids.py defaults <ticker> <tf>                 -> point_<tf>.json (core defaults)
<tf> must be Minutes15: drop_bounce runs on 15m only (owner, 2026-10-15: hourly signal, hourly MACD, 15m MACD).
"""
import json, os, re, sys

DEFAULTS = dict(RSIPeriod=6, RSIZone=30, RSICap=50, ZoneEntry=1, EMAPeriod=200, StopDailyATR=1.0,
                DailyATRPeriod=14, RSIExitPeriod=10, RSIExit=70,
                HourMACDFast=8, HourMACDSlow=26, HourMACDSignal=5, HourMACDHours=20,
                MACDFast=12, MACDSlow=26, MACDSignal=9, ConfirmBars=6, FridayTo=11)
# Hold-out scheme (owner, 2026-10-14): one in-sample sweep on the first 30 of 36 months, the point is
# the centre of the best plateau, the last 6 months are opened once. Axes (owner, 2026-10-14; rules
# on 15m since 2026-10-15: hourly RSI signal, hourly MACD(8,26,5) within 20 hours, 15m MACD
# confirmation, hourly RSI(RSIExitPeriod) exit): the entry RSI length, the exit RSI length and the
# stop as a share of the daily ATR or a whole one — 3 × 5 × 3 = 45, swept once per MACD pair (HOUR_MACD
# × M15_MACD below). Fixed at the defaults: the 20-hour wait for the hourly MACD, ConfirmBars 6, Friday
# buys before 11:00 (FridayTo 11).
# Owner 2026-10-15: the breakeven, the trailing stop and every off-by-default filter were removed from
# the core; the stop stays where it was set.
# The previous axes (EMAPeriod × StaleBars × StopDailyATR) belonged to the rules with the day-range
# gate and the MACD exit. Do not change without a request.
SWEEP = {"RSIPeriod": [4, 5, 6], "RSIExitPeriod": [8, 9, 10, 11, 12], "StopDailyATR": [0.5, 0.7, 1.0]}
# MACD setups (owner, 2026-10-15: "concrete MACD setups for 15m and for the hour, the calibration picks the
# best"). A setup is a (fast, slow, signal) triple, never a cartesian product of its numbers: the engine
# expands a grid field by field, so each pair of setups gets its own grid file and its own sweep run.
# Hour: 8/26/5 the owner's, 12/26/9 the classic, 5/35/5 slow (trend). 15m: 12/26/9 the classic,
# 8/17/9 fast (Appel), 3/10/16 (Raschke).
# The pair is a separate step before the sweep (owner, 2026-10-15: 9 × 45 = 405 runs took ~45 minutes):
# every pair runs once at the core defaults (RSI 6, exit RSI 10, stop 1.0; MACD_PROBE empty — 9 runs),
# report.py macd picks the pair by PF, then SWEEP runs on that pair only (45 runs). Known cost (owner
# accepted): the pair is chosen at uncalibrated RSI and stop.
HOUR_MACD = [(8, 26, 5), (12, 26, 9), (5, 35, 5)]
M15_MACD = [(12, 26, 9), (8, 17, 9), (3, 10, 16)]
MACD_PROBE = {}
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


def tag(p):
    """File tag of a MACD pair: h8-26-5_m12-26-9."""
    return (f"h{p['HourMACDFast']}-{p['HourMACDSlow']}-{p['HourMACDSignal']}"
            f"_m{p['MACDFast']}-{p['MACDSlow']}-{p['MACDSignal']}")


def macd(h, m):
    return dict(HourMACDFast=h[0], HourMACDSlow=h[1], HourMACDSignal=h[2], MACDFast=m[0], MACDSlow=m[1], MACDSignal=m[2])


def untag(s):
    """MACD fields of a pair tag h8-26-5_m12-26-9."""
    m = re.fullmatch(r"h(\d+)-(\d+)-(\d+)_m(\d+)-(\d+)-(\d+)", s)
    if not m:
        sys.exit(f"пара MACD {s!r}: нужен вид h8-26-5_m12-26-9")
    v = [int(x) for x in m.groups()]
    return macd(v[:3], v[3:])


def cmd(t, tf, name, out):
    return (f"reports/drop_bounce_cal/backtest -ticker {t} -strategy drop_bounce -interval {tf} "
            f"-calibrate data/params/drop_bounce/{t.lower()}/{name}.json -months 36 -holdout-months {HOLDOUT_MONTHS} "
            f"-metric profit_factor -risk-pct 1 -min-trades 1 -combos 0 -out reports/{t}_drop_bounce/{out}/{tf}")


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
    if tf != "Minutes15":
        sys.exit("калибруется только Minutes15: drop_bounce работает только на 15m (решение владельца 2026-10-15)")
    low = tf.lower()
    if kind == "macd":
        runs = []
        for h in HOUR_MACD:
            for m in M15_MACD:
                g = {k: [v] for k, v in DEFAULTS.items()}
                g.update({k: [v] for k, v in macd(h, m).items()})
                g.update(MACD_PROBE)
                pair = tag(macd(h, m))
                name = f"macd_{low}_{pair}"
                c = cmd(t, tf, name, f"macd/{pair}")
                dump(path(t, name), {
                    "_comment": f"drop_bounce {t} {tf}: шаг выбора пары MACD на первых 30 из 36 месяцев (последние {HOLDOUT_MONTHS} — hold-out), MACD 1h {h[0]}/{h[1]}/{h[2]}, MACD 15m {m[0]}/{m[1]}/{m[2]}, одна точка — дефолты ядра (RSI 6, RSI выхода 10, стоп 1.0), все поля прибиты. Одна из {len(HOUR_MACD) * len(M15_MACD)} пар: выбирает report.py macd <все calibration.md>. Запуск: {c}. РЕЗУЛЬТАТ: прогон не делался.",
                    "phases": [{"name": "macd", "grid": g}]})
                runs.append(c)
        print("запуск, строго по очереди:")
        print("\n".join(runs))
    elif kind == "sweep":
        if len(sys.argv) < 5:
            sys.exit("grids.py sweep <T> Minutes15 <пара MACD из report.py macd>")
        pair = untag(sys.argv[4])
        g = {k: [v] for k, v in DEFAULTS.items()}
        g.update({k: [v] for k, v in pair.items()})
        g.update(SWEEP)
        axes = ", ".join(f"{k} {v}" for k, v in SWEEP.items())
        dump(path(t, f"sweep_{low}"), {
            "_comment": f"drop_bounce {t} {tf}: in-sample перебор на первых 30 из 36 месяцев (последние {HOLDOUT_MONTHS} — hold-out), пара MACD {sys.argv[4]} (шаг MACD), оси {axes}, остальное — дефолты ядра, все поля прибиты. Запуск: {cmd(t, tf, f'sweep_{low}', 'sweep')}. РЕЗУЛЬТАТ: прогон не делался.",
            "phases": [{"name": "sweep", "grid": g}]})
    elif kind in ("point", "cand"):
        k = int(sys.argv[5]) if len(sys.argv) > 5 else 1
        p = dict(DEFAULTS)
        p.update(parse_combo(sys.argv[4], k))
        if p["StopDailyATR"] <= 0:
            sys.exit("точка без стопа — недопустимо")
        dump(path(t, f"point_{low}" if kind == "point" else f"cand{k}_{low}"), p)
    elif kind == "defaults":
        dump(path(t, f"point_{low}"), dict(DEFAULTS))
    else:
        sys.exit(__doc__)


if __name__ == "__main__":
    main()
