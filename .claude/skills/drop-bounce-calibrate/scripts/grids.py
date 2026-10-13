"""Grid files for drop_bounce calibration of one ticker on one timeframe.

Every grid pins all Params fields explicitly: a registered ticker's ParseParams layers JSON over
its own literal, so an unlisted field would silently come from the old literal.

Usage (from the repo root):
  grids.py baseline <ticker> <tf>                 -> cal_<tf>_baseline.json (core defaults, one point)
  grids.py calstop  <ticker> <tf>                 -> calstop_<tf>.json (phased grid, stop never 0)
  grids.py point    <ticker> <tf> <calib.md> [k]  -> point_<tf>.json (flat Params of combo #k, default 1)
  grids.py consensus <ticker> <tf> <walkforward.md> -> point_<tf>.json (weighted mean of the folds' picks)
  grids.py plateau  <ticker> <tf>                 -> plateau_<tf>_entry.json, plateau_<tf>_stop.json
<tf> must be Minutes15: drop_bounce is traded on 15m only (owner, 2026-10-13).
"""
import json, os, re, sys

DEFAULTS = dict(VolDailyMult=0.4, VolAvgPeriod=14, RSIPeriod=6, RSIExit=70, StopDailyATR=1.0,
                DailyATRPeriod=14, StartHour=7)
# share of an average weekday daily volume traded by the signal bar
VOL = {"Minutes5": [0.02, 0.04, 0.06, 0.1, 0.15, 0.2],
       "Minutes15": [0.05, 0.1, 0.15, 0.2, 0.25, 0.3, 0.4, 0.5, 0.6, 0.8],
       "Minutes30": [0.1, 0.15, 0.2, 0.25, 0.3, 0.4, 0.5, 0.6, 0.8, 1.0],
       "Hour1": [0.15, 0.2, 0.25, 0.3, 0.4, 0.5, 0.6, 0.8, 1.0, 1.2]}
# plateau step of VolDailyMult past the grid's edge
VOL_STEP = {"Minutes5": 0.02, "Minutes15": 0.05, "Minutes30": 0.1, "Hour1": 0.1}
STOPS = [0.3, 0.5, 0.75, 1.0, 1.25, 1.5, 2.0, 2.5, 3.0]  # daily ATRs
RSI_EXITS = [50, 55, 60, 65, 70, 75, 80, 85, 90]
PHASES = lambda tf: [
    ("entry", {"VolDailyMult": VOL[tf], "VolAvgPeriod": [5, 10, 14]}),
    ("risk", {"StopDailyATR": STOPS, "DailyATRPeriod": [7, 10, 14]}),
    ("exits", {"RSIPeriod": [4, 5, 6, 7], "RSIExit": RSI_EXITS}),
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


def fold_picks(md):
    """Per-field values the walk-forward folds picked, oldest fold first."""
    s = open(md).read()
    folds = len(re.findall(r"^\| \d+ \| \d{4}-", s, re.M))
    picks = {}
    stable = re.search(r"\*\*Стабильные\*\*[^:]*: (.+)$", s, re.M)
    if stable:
        for k, v in re.findall(r"(\w+)=([-\d.]+)", stable.group(1)):
            picks[k] = [float(v)] * folds
    for k, vs in re.findall(r"^- ([A-Za-z]+): (.+)$", s, re.M):
        picks[k] = [float(v) for v in vs.split(",")]
    return picks


def consensus(picks):
    """Weighted mean of each field's fold picks; a value picked k times weighs k² (repeats count
    more than in a plain mean). Integer fields round to int, the rest to 0.01."""
    out = {}
    for k, default in DEFAULTS.items():
        vs = picks.get(k)
        if not vs:
            out[k] = default
            continue
        w = {v: vs.count(v) ** 2 for v in set(vs)}
        m = sum(v * c for v, c in w.items()) / sum(w.values())
        out[k] = int(round(m)) if isinstance(default, int) else round(m, 2)
    return out


def neighbours(values, x, step):
    """x with one step on each side; past the grid's edge the step continues outward."""
    values = sorted(set(values) | {x})
    i = values.index(x)
    lo = values[i - 1] if i > 0 else round(x - step, 2)
    hi = values[i + 1] if i + 1 < len(values) else round(x + step, 2)
    return [v for v in (lo, x, hi) if v > 0]


def main():
    kind, t, tf = sys.argv[1], sys.argv[2].upper(), sys.argv[3]
    if tf != "Minutes15":
        sys.exit("калибруется только Minutes15: drop_bounce торгуем только на 15m (решение владельца 2026-10-13)")
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
    elif kind == "consensus":
        picks = fold_picks(sys.argv[4])
        for k, vs in sorted(picks.items()):
            print(f"  {k}: фолды {vs}")
        p = consensus(picks)
        if p["StopDailyATR"] <= 0:
            sys.exit("точка без стопа — недопустимо")
        dump(path(t, f"point_{low}"), p)
        print(json.dumps(p, ensure_ascii=False))
    elif kind == "plateau":
        p = json.load(open(path(t, f"point_{low}")))
        e = pinned({k: [v] for k, v in p.items()})
        e["VolDailyMult"] = neighbours(VOL[tf], p["VolDailyMult"], VOL_STEP[tf])
        e["RSIExit"] = neighbours(RSI_EXITS, p["RSIExit"], 5)
        dump(path(t, f"plateau_{low}_entry"), {
            "_comment": f"drop_bounce {t} {tf}: плато точки по входу и выходу (±1 шаг VolDailyMult × RSIExit), in-sample. Запуск: {cmd(t, tf, f'plateau_{low}_entry', wf=False)}. РЕЗУЛЬТАТ: прогон не делался.",
            "phases": [{"name": "plateau_entry", "grid": e}]})
        s = pinned({k: [v] for k, v in p.items()})
        s["StopDailyATR"] = neighbours(STOPS, p["StopDailyATR"], 0.25)
        dump(path(t, f"plateau_{low}_stop"), {
            "_comment": f"drop_bounce {t} {tf}: плато точки по стопу (±1 шаг StopDailyATR), in-sample. Запуск: {cmd(t, tf, f'plateau_{low}_stop', wf=False)}. РЕЗУЛЬТАТ: прогон не делался.",
            "phases": [{"name": "plateau_stop", "grid": s}]})
    else:
        sys.exit(__doc__)


if __name__ == "__main__":
    main()
