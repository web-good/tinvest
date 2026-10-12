"""Grid files for drop_bounce calibration of one ticker on one timeframe.

Every grid pins all Params fields explicitly: a registered ticker's ParseParams layers JSON over
its own literal, so an unlisted field would silently come from the old literal.

Usage (from the repo root):
  grids.py baseline <ticker> <tf>                 -> cal_<tf>_baseline.json (core defaults, one point)
  grids.py calstop  <ticker> <tf>                 -> calstop_<tf>.json (phased grid, stop never 0)
  grids.py point    <ticker> <tf> <calib.md> [k]  -> point_<tf>.json (flat Params of combo #k, default 1)
  grids.py plateau  <ticker> <tf>                 -> plateau_<tf>_entry.json, plateau_<tf>_stop.json,
                                                     plateau_<tf>_day.json, plateau_<tf>_volume.json (only for a gate
                                                     the point has on)
<tf> is Minutes5 or Minutes15; Minutes30 and Hour1 only on the owner's explicit request.
"""
import json, os, re, sys

DEFAULTS = dict(MinDropPct=0.6, RSIPeriod=6, RSIEntry=30, RSIExit=70, ExitInProfit=1, StochPeriod=14, StochSmooth=3, StochExit=80,
                ProfitPct=1, StopDailyATR=0.5, DailyATRPeriod=14, MaxDayRangeATR=0, VolMult=0, VolBars=20,
                StartHour=8)
DROP = {"Minutes5": [0.3, 0.4, 0.5, 0.7], "Minutes15": [0.4, 0.6, 0.8, 1.0],
        "Minutes30": [0.5, 0.8, 1.0, 1.3], "Hour1": [0.6, 1.0, 1.4, 2.0]}
# plateau step of MinDropPct past the grid's edge
DROP_STEP = {"Minutes5": 0.1, "Minutes15": 0.2, "Minutes30": 0.3, "Hour1": 0.4}
STOPS = [0.25, 0.4, 0.5, 0.75, 1.0]  # daily ATRs
DAY_RANGE = [0, 0.5, 0.75, 1.0, 1.5]  # daily ATRs, 0 = gate off
VOL_MULT = [0, 1.5, 2.0, 3.0]  # x mean volume of the VolBars bars before the signal, 0 = gate off
PHASES = lambda tf: [
    ("entry", {"MinDropPct": DROP[tf], "RSIPeriod": [3, 6, 9], "RSIEntry": [20, 25, 30]}),
    ("risk", {"StopDailyATR": STOPS, "ProfitPct": [0, 0.5, 1.0, 1.5, 2.0, 3.0]}),
    # a phase of its own: inside risk it would multiply that phase fivefold
    ("day", {"MaxDayRangeATR": DAY_RANGE}),
    ("volume", {"VolMult": VOL_MULT, "VolBars": [20, 50]}),
    ("exits", {"RSIExit": [0, 65, 70, 75], "ExitInProfit": [0, 1], "StochExit": [0, 80, 90]}),
]


def path(t, name):
    d = f"data/params/drop_bounce/{t.lower()}"
    os.makedirs(d, exist_ok=True)
    return f"{d}/{name}.json"


def dump(p, doc):
    with open(p, "w") as f:
        json.dump(doc, f, ensure_ascii=False, indent=2)
        f.write("\n")
    print(p)


def cmd(t, tf, name, wf=True, extra=""):
    s = (f"reports/drop_bounce_cal/backtest -ticker {t} -strategy drop_bounce -interval {tf} "
         f"-calibrate data/params/drop_bounce/{t.lower()}/{name}.json -months 36 -metric profit_factor")
    return s + (" -train-months 12 -test-months 6 -min-trades 10" if wf else " -min-trades 1") + extra


def pinned(over):
    g = {k: [v] for k, v in DEFAULTS.items()}
    g.update(over)
    return g


def parse_combo(md, k):
    s = open(md).read()
    m = re.search(rf"^### #{k} .*?(?=^### #|\Z)", s, re.M | re.S)
    if not m:
        sys.exit(f"combo #{k} not found in {md}")
    p = {name: float(v) for name, v in re.findall(r"^\| (\w+) \| ([-\d.]+) \|", m.group(0), re.M)}
    return {k2: (int(v) if v.is_integer() else v) for k2, v in p.items() if k2 in DEFAULTS}


def neighbours(values, x, step):
    """x with one step on each side; past the grid's edge the step continues outward."""
    values = sorted(set(values) | {x})
    i = values.index(x)
    lo = values[i - 1] if i > 0 else round(x - step, 2)
    hi = values[i + 1] if i + 1 < len(values) else round(x + step, 2)
    return [v for v in (lo, x, hi) if v > 0]


def gate_neighbours(grid, x, step):
    """A gate threshold with one step on each side; the lower neighbour may be 0 (gate off)."""
    values = sorted(set(grid) | {x})
    i = values.index(x)
    lo = values[i - 1] if i > 0 else 0
    hi = values[i + 1] if i + 1 < len(values) else round(x + step, 2)
    return [lo, x, hi]


def main():
    kind, t, tf = sys.argv[1], sys.argv[2].upper(), sys.argv[3]
    if tf not in DROP:
        sys.exit("tf must be Minutes5, Minutes15, Minutes30 or Hour1")
    low = tf.lower()
    if kind == "baseline":
        dump(path(t, f"cal_{low}_baseline"), {
            "_comment": f"drop_bounce {t} {tf}: baseline — одна точка, дефолты ядра. Запуск: {cmd(t, tf, f'cal_{low}_baseline')}. РЕЗУЛЬТАТ: прогон не делался.",
            "phases": [{"name": "baseline", "grid": pinned({})}]})
    elif kind == "calstop":
        phases = PHASES(tf)
        # the first phase pins every field, so fields swept later start from the core defaults
        doc = [{"name": phases[0][0], "keepTop": 3, "grid": pinned(phases[0][1])}]
        doc += [{"name": n, "keepTop": 3, "grid": g} for n, g in phases[1:]]
        dump(path(t, f"calstop_{low}"), {
            "_comment": f"drop_bounce {t} {tf}: фазовая калибровка entry → risk → exits (keepTop 3), стоп без нуля, все поля прибиты. Запуск WF: {cmd(t, tf, f'calstop_{low}')}; in-sample: то же без -train/-test, -min-trades 30. РЕЗУЛЬТАТ: прогон не делался.",
            "phases": doc})
    elif kind == "point":
        k = int(sys.argv[5]) if len(sys.argv) > 5 else 1
        p = dict(DEFAULTS)
        p.update(parse_combo(sys.argv[4], k))
        if p["StopDailyATR"] <= 0:
            sys.exit("точка без стопа — недопустимо")
        dump(path(t, f"point_{low}"), p)
    elif kind == "plateau":
        p = json.load(open(path(t, f"point_{low}")))
        e = pinned({k: [v] for k, v in p.items()})
        e["MinDropPct"] = neighbours(DROP[tf], p["MinDropPct"], DROP_STEP[tf])
        e["RSIEntry"] = neighbours([15, 20, 25, 30, 35], p["RSIEntry"], 5)
        dump(path(t, f"plateau_{low}_entry"), {
            "_comment": f"drop_bounce {t} {tf}: плато точки по входу (±1 шаг MinDropPct × RSIEntry), in-sample. Запуск: {cmd(t, tf, f'plateau_{low}_entry', wf=False)}. РЕЗУЛЬТАТ: прогон не делался.",
            "phases": [{"name": "plateau_entry", "grid": e}]})
        s = pinned({k: [v] for k, v in p.items()})
        s["StopDailyATR"] = neighbours(STOPS, p["StopDailyATR"], 0.25)
        dump(path(t, f"plateau_{low}_stop"), {
            "_comment": f"drop_bounce {t} {tf}: плато точки по стопу (±1 шаг StopDailyATR), in-sample. Запуск: {cmd(t, tf, f'plateau_{low}_stop', wf=False)}. РЕЗУЛЬТАТ: прогон не делался.",
            "phases": [{"name": "plateau_stop", "grid": s}]})
        if p["MaxDayRangeATR"] > 0:
            d = pinned({k: [v] for k, v in p.items()})
            d["MaxDayRangeATR"] = gate_neighbours(DAY_RANGE, p["MaxDayRangeATR"], 0.5)
            dump(path(t, f"plateau_{low}_day"), {
                "_comment": f"drop_bounce {t} {tf}: плато точки по фильтру размаха дня (±1 шаг MaxDayRangeATR, нижний сосед может быть 0 — фильтр выключен), in-sample. Запуск: {cmd(t, tf, f'plateau_{low}_day', wf=False)}. РЕЗУЛЬТАТ: прогон не делался.",
                "phases": [{"name": "plateau_day", "grid": d}]})
        if p["VolMult"] > 0:
            v = pinned({k: [v] for k, v in p.items()})
            v["VolMult"] = gate_neighbours(VOL_MULT, p["VolMult"], 1.0)
            dump(path(t, f"plateau_{low}_volume"), {
                "_comment": f"drop_bounce {t} {tf}: плато точки по объёмному фильтру (±1 шаг VolMult при VolBars точки, нижний сосед может быть 0 — фильтр выключен), in-sample. Запуск: {cmd(t, tf, f'plateau_{low}_volume', wf=False)}. РЕЗУЛЬТАТ: прогон не делался.",
                "phases": [{"name": "plateau_volume", "grid": v}]})
    else:
        sys.exit(__doc__)


if __name__ == "__main__":
    main()
