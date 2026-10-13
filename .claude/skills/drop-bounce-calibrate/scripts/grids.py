"""Grid files for drop_bounce calibration of one ticker on one timeframe.

Every grid pins all Params fields explicitly: a registered ticker's ParseParams layers JSON over
its own literal, so an unlisted field would silently come from the old literal.

Usage (from the repo root):
  grids.py baseline <ticker> <tf>                 -> cal_<tf>_baseline.json (core defaults, one point)
  grids.py calstop  <ticker> <tf>                 -> calstop_<tf>.json (phased grid, stop never 0)
  grids.py point    <ticker> <tf> <calib.md> [k]  -> point_<tf>.json (flat Params of combo #k, default 1)
  grids.py consensus <ticker> <tf> <walkforward.md> -> point_<tf>.json (weighted mean of the folds' picks)
  grids.py plateau  <ticker> <tf>                 -> plateau_<tf>_entry.json, plateau_<tf>_stop.json
<tf> must be Hour1: drop_bounce runs on 1h only (owner, 2026-10-13).
"""
import json, os, re, sys

DEFAULTS = dict(RSIPeriod=6, RSIZone=30, RSICap=50, EMAPeriod=200, DaySpentATR=0.7,
                StopDailyATR=1.0, DailyATRPeriod=14, RSIExit=70, StartHour=7)
ZONES = [25, 30, 35]
CAPS = [40, 50, 60]
DAY_SPENT = [0.3, 0.5, 0.7, 1.0]
STOPS = [0.5, 0.7, 1.0, 1.5, 2.0]  # daily ATRs
RSI_EXITS = [60, 65, 70, 75, 80]
# owner's lists, 2026-10-13; do not change without a request
PHASES = [
    ("entry", {"RSIPeriod": [4, 5, 6, 7], "RSIZone": ZONES, "RSICap": CAPS}),
    ("filter", {"EMAPeriod": [100, 150, 200], "DaySpentATR": DAY_SPENT}),
    ("risk", {"StopDailyATR": STOPS, "DailyATRPeriod": [7, 10, 14]}),
    ("exit", {"RSIExit": RSI_EXITS}),
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
    if tf != "Hour1":
        sys.exit("калибруется только Hour1: drop_bounce работает только на 1h (решение владельца 2026-10-13)")
    low = tf.lower()
    if kind == "baseline":
        dump(path(t, f"cal_{low}_baseline"), {
            "_comment": f"drop_bounce {t} {tf}: baseline — одна точка, дефолты ядра. Запуск: {cmd(t, tf, f'cal_{low}_baseline')}. РЕЗУЛЬТАТ: прогон не делался.",
            "phases": [{"name": "baseline", "grid": pinned({})}]})
    elif kind == "calstop":
        phases = PHASES
        # the first phase pins every field, so fields swept later start from the core defaults
        doc = [{"name": phases[0][0], "keepTop": 3, "grid": pinned(phases[0][1])}]
        doc += [{"name": n, "keepTop": 3, "grid": g} for n, g in phases[1:]]
        dump(path(t, f"calstop_{low}"), {
            "_comment": f"drop_bounce {t} {tf}: фазовая калибровка entry → filter → risk → exit (keepTop 3), стоп без нуля, все поля прибиты. Запуск WF: {cmd(t, tf, f'calstop_{low}')}; in-sample: то же без -train/-test, -min-trades 30. РЕЗУЛЬТАТ: прогон не делался.",
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
        e["RSIZone"] = neighbours(ZONES, p["RSIZone"], 5)
        e["DaySpentATR"] = neighbours(DAY_SPENT, p["DaySpentATR"], 0.1)
        dump(path(t, f"plateau_{low}_entry"), {
            "_comment": f"drop_bounce {t} {tf}: плато точки по входу и фильтру (±1 шаг RSIZone × DaySpentATR), in-sample. Запуск: {cmd(t, tf, f'plateau_{low}_entry', wf=False)}. РЕЗУЛЬТАТ: прогон не делался.",
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
