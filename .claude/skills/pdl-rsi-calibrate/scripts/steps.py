"""Step-by-step calibration of pdl_rsi (Hour1) for one ticker: coordinate descent over parameter groups.

One group of related parameters is swept per step, everything else pinned at the point chosen so far.
Each step moves the point to the centre of the best plateau of its own grid, and only when that
beats the current point's neighbourhood by MOVE_SHARE; otherwise the point stays.

Usage (from the repo root):
  steps.py init    <T>                 -> data/params/pdl_rsi/<t>/cal/point.json: the open start (filters
                                          and optional exits off, core defaults elsewhere)
  steps.py grid    <T> <step>          -> cal/<nn>_<step>.json: the step's axes over the current point,
                                          every other field pinned; prints the run command
  steps.py pick    <T> <step> <calibration.md>
                                       the plateau centre of the step, the move/keep decision; updates
                                          point.json and writes РЕЗУЛЬТАТ into the step's grid
  steps.py status  <T>                 the current point and the steps already picked
  steps.py wf      <T>                 -> cal/wf_rolling.json: the compact walk-forward grid around the point
  steps.py verdict <walkforward.md>    the pooled OOS gate
  steps.py literal <T>                 the point as a Go literal body for core.Params

Steps, in order (STEPS below): signal, trigger, trend, floor, trigmax, volume, room, risk, exits, then the
second pass signal2, trigger2, risk2. A step cannot be picked before the previous one.
"""
import json, os, re, statistics, sys

# Core defaults (core.DefaultParams) with every filter and optional exit off: the start of the descent.
# The filters start off so the signal and trigger steps see the biggest sample; each filter is switched
# on later only when it pays for itself.
START = dict(HourRSIPeriod=6, RSIArm=30, RSIFloor=0, RSITriggerMax=0, RSIExit=70, StochExitAbove=0,
             StochK=5, StochSmooth=3, StochD=3, StochBars=4, StochCrossMax=40,
             VolumeMult=0, VolBars=20, MinRoomATR=0,
             StopATR=0.5, MaxStopPct=0, TargetR=2, TrendEMA=0, DailyATRPeriod=14)
# Never swept: the stochastic smoothing, the volume base and the ATR period.
FIXED = ("StochSmooth", "StochD", "VolBars", "DailyATRPeriod")

# (name, axes). Order is the method: first what makes a setup (the most trades), then the trigger, then
# the filters one at a time (each removes trades and must earn its place), then the payoff and the exits
# on the final set of trades, then a second pass over the signal, the trigger and the payoff with
# everything else in place.
SIGNAL = {"HourRSIPeriod": [4, 5, 6, 8], "RSIArm": [25, 30, 35]}
TRIGGER = {"StochK": [5, 10, 14], "StochBars": [2, 4, 6], "StochCrossMax": [0, 30, 40, 50]}
RISK = {"StopATR": [0.3, 0.5, 0.7, 1.0], "TargetR": [1.5, 2, 3], "MaxStopPct": [0, 1, 1.5]}
STEPS = [
    ("signal", SIGNAL),
    ("trigger", TRIGGER),
    ("trend", {"TrendEMA": [0, 100, 150, 200]}),
    ("floor", {"RSIFloor": [0, 15, 20, 22]}),
    ("trigmax", {"RSITriggerMax": [0, 40, 50]}),
    ("volume", {"VolumeMult": [0, 0.5, 0.8, 1.2]}),
    ("room", {"MinRoomATR": [0, 0.3, 0.6]}),
    ("risk", RISK),
    ("exits", {"RSIExit": [0, 65, 70, 75], "StochExitAbove": [0, 40, 60]}),
    ("signal2", SIGNAL),
    ("trigger2", TRIGGER),
    ("risk2", RISK),
]
# The walk-forward grid: one step either side of the point on these axes (≤ 27 points).
WF_AXES = ("HourRSIPeriod", "StopATR", "TargetR")

MONTHS = 36
MIN_TRADES = 20      # ≈ 7 a year over 36 months; fewer — the combo is out of the choice
MOVE_SHARE = 0.05    # the point moves only when the new centre beats the current one by 5%
PF_CAP = 10.0        # an inf PF (no losers) counts as this in a median
WF_MIN_TRADES = 3
GATE_PF, GATE_TRADES = 1.2, 10
BIN = "reports/pdl_rsi_cal/backtest"


def caldir(t):
    d = f"data/params/pdl_rsi/{t.lower()}/cal"
    os.makedirs(d, exist_ok=True)
    return d


def load(p):
    with open(p) as f:
        return json.load(f)


def dump(p, doc):
    with open(p, "w") as f:
        json.dump(doc, f, ensure_ascii=False, indent=2)
        f.write("\n")


def num(v):
    return int(v) if float(v).is_integer() else v


def step_index(name):
    for i, (n, _) in enumerate(STEPS):
        if n == name:
            return i
    sys.exit(f"шаг {name!r}: нет такого, шаги — " + ", ".join(n for n, _ in STEPS))


def grid_path(t, name):
    return f"{caldir(t)}/{step_index(name) + 1:02d}_{name}.json"


def state(t):
    p = f"{caldir(t)}/point.json"
    if not os.path.exists(p):
        sys.exit(f"нет {p}: сначала steps.py init {t}")
    return p, load(p)


def cmd(t, grid, out, extra=""):
    return (f"taskset -c 0 nice -n 19 env GOMAXPROCS=1 {BIN} -ticker {t} -strategy pdl_rsi -interval Hour1 "
            f"-risk-pct 1 -metric profit_factor -months {MONTHS} -calibrate {grid} -min-trades 1 -combos 0{extra} "
            f"-out reports/{t}_pdl_rsi/{out}")


def init(t):
    p = f"{caldir(t)}/point.json"
    if os.path.exists(p):
        sys.exit(f"{p} уже есть: калибровка начата, продолжай с steps.py status {t}")
    dump(p, {"_comment": f"pdl_rsi {t}: текущая точка пошаговой калибровки (steps.py). Шаги: нет.",
             "_steps": [], "params": START})
    print(p)
    print("дальше: steps.py grid", t, STEPS[0][0])


def grid(t, name):
    i = step_index(name)
    _, st = state(t)
    done = [s["step"] for s in st["_steps"]]
    want = [n for n, _ in STEPS[:i]]
    if done != want:
        sys.exit(f"шаг {name}: до него должны быть выбраны {want}, выбраны {done}")
    p = st["params"]
    g = {k: [v] for k, v in p.items()}
    axes = {}
    for k, vals in STEPS[i][1].items():
        axes[k] = sorted(set(vals) | {p[k]})  # the current value always competes
        g[k] = axes[k]
    gp = grid_path(t, name)
    c = cmd(t, gp, name)
    dump(gp, {"_comment": f"pdl_rsi {t} Hour1, шаг {i + 1} «{name}» пошаговой калибровки ({MONTHS} мес, -risk-pct 1): оси "
                          + ", ".join(f"{k} {v}" for k, v in axes.items())
                          + f", остальные поля прибиты к точке шагов {done or 'старта'}. Запуск: {c}. РЕЗУЛЬТАТ: прогон не делался.",
              "phases": [{"name": name, "grid": g}]})
    print(gp)
    print(c)


def combos(md):
    s = open(md).read()
    total = int(re.search(r"Всего комбинаций: (\d+)", s).group(1))
    table = {}
    for m in re.finditer(r"^\| (\d+) \| [^|]+ \| ([-+\w.]+) \| ([-\d.]+) \| [^|]+ \| [^|]+ \| (\d+) \|$", s, re.M):
        f = float(m.group(2))
        table[int(m.group(1))] = (min(f, PF_CAP), float(m.group(3)), int(m.group(4)))
    rows = []
    for m in re.finditer(r"^### #(\d+) — .*?\n(.*?)(?=^### #|\Z)", s, re.M | re.S):
        k = int(m.group(1))
        p = {a: num(float(v)) for a, v in re.findall(r"^\| (\w+) \| ([-\d.]+) \|", m.group(2), re.M)}
        rows.append((k, *table[k], p))
    if len(rows) < total:
        sys.exit(f"{md}: в отчёте {len(rows)} из {total} комбинаций — запусти с -combos 0")
    return rows


def pick(t, name, md):
    i = step_index(name)
    sp, st = state(t)
    done = [s["step"] for s in st["_steps"]]
    if done != [n for n, _ in STEPS[:i]]:
        sys.exit(f"шаг {name}: выбраны {done} — шаги идут по порядку, повторно шаг не выбирают")
    axes = {k: sorted({r[4][k] for r in combos(md)}) for k in STEPS[i][1]}
    rows = combos(md)
    cur = st["params"]

    def neighbours(p):
        out = []
        for r in rows:
            diff = [k for k in axes if r[4][k] != p[k]]
            if len(diff) == 1 and abs(axes[diff[0]].index(r[4][diff[0]]) - axes[diff[0]].index(p[diff[0]])) == 1:
                out.append(r)
        return out

    def score(r):
        return statistics.median([r[1]] + [n[1] for n in neighbours(r[4])])

    def tag(p):
        return ", ".join(f"{k} {p[k]:g}" for k in axes)

    here = next(r for r in rows if all(r[4][k] == cur[k] for k in axes))
    ok = [r for r in rows if r[3] >= MIN_TRADES]
    print(f"шаг {i + 1} «{name}»: {len(rows)} комбинаций, с ≥ {MIN_TRADES} сделками — {len(ok)}")
    for r in sorted(ok, key=lambda r: (-score(r), -r[3]))[:5]:
        print(f"  #{r[0]}: медиана окрестности {score(r):.3f}, PF {r[1]:.3f}, net {r[2]:+.0f}, сделок {r[3]} — {tag(r[4])}")
    hs = score(here) if here[3] >= MIN_TRADES else float("-inf")
    print(f"  текущая точка #{here[0]}: медиана {score(here):.3f}, PF {here[1]:.3f}, сделок {here[3]} — {tag(cur)}"
          + ("" if here[3] >= MIN_TRADES else f" (меньше {MIN_TRADES} сделок)"))
    if not ok:
        best, why = here, f"ни у одной комбинации нет {MIN_TRADES} сделок — точка не двигается"
    else:
        top = max(ok, key=lambda r: (score(r), r[3], -r[0]))
        if all(top[4][k] == cur[k] for k in axes) or (hs > float("-inf") and score(top) < hs * (1 + MOVE_SHARE)):
            best, why = here, f"лучший центр {tag(top[4])} ({score(top):.3f}) не выше текущей точки на {MOVE_SHARE:.0%} — точка остаётся"
        else:
            best, why = top, "точка сдвинута"
    nb = neighbours(best[4])
    plateau = all(n[1] >= 1.0 for n in nb)
    print(f"ВЫБОР: {tag(best[4])}, PF {best[1]:.3f}, net {best[2]:+.0f}, сделок {best[3]} — {why}")
    for n in nb:
        d = next(k for k in axes if n[4][k] != best[4][k])
        print(f"  сосед {d} {n[4][d]:g}: PF {n[1]:.3f}/{n[3]}")
    print("ПЛАТО ЕСТЬ" if plateau else "ОСТРЫЙ ПИК (соседи ниже PF 1)")
    for k in axes:
        cur[k] = best[4][k]
    res = f"{tag(best[4])}, PF {best[1]:.3f}, net {best[2]:+.0f}, сделок {best[3]}, {'плато' if plateau else 'острый пик'}; {why}"
    st["_steps"].append({"step": name, "result": res})
    st["_comment"] = f"pdl_rsi {t}: текущая точка пошаговой калибровки (steps.py). Шаги: " + "; ".join(
        f"{s['step']} — {s['result']}" for s in st["_steps"]) + "."
    dump(sp, st)
    gp = grid_path(t, name)
    g = load(gp)
    g["_comment"] = g["_comment"].replace("РЕЗУЛЬТАТ: прогон не делался.", "") + f" РЕЗУЛЬТАТ: {res}."
    dump(gp, g)
    nxt = STEPS[i + 1][0] if i + 1 < len(STEPS) else None
    print(f"дальше: steps.py grid {t} {nxt}" if nxt else f"шаги пройдены; дальше: steps.py wf {t}")


def status(t):
    _, st = state(t)
    for s in st["_steps"]:
        print(f"  {s['step']}: {s['result']}")
    done = len(st["_steps"])
    print("точка:", ", ".join(f"{k} {v:g}" for k, v in st["params"].items()))
    print(f"дальше: steps.py grid {t} {STEPS[done][0]}" if done < len(STEPS) else f"шаги пройдены; дальше: steps.py wf {t}")


def wf(t):
    _, st = state(t)
    if len(st["_steps"]) < len(STEPS):
        sys.exit("walk-forward — после всех шагов")
    p = st["params"]
    g = {k: [v] for k, v in p.items()}
    for k in WF_AXES:
        vals = sorted({v for _, ax in STEPS for a, vv in ax.items() if a == k for v in vv} | {p[k]})
        j = vals.index(p[k])
        g[k] = vals[max(j - 1, 0): j + 2]
    gp = f"{caldir(t)}/wf_rolling.json"
    c = cmd(t, gp, "wf", f" -train-months 3 -test-months 1").replace("-min-trades 1", f"-min-trades {WF_MIN_TRADES}")
    dump(gp, {"_comment": f"pdl_rsi {t} Hour1, скользящий walk-forward (подбор 3 мес, проверка 1 мес, шаг 1 мес, {MONTHS} мес, -risk-pct 1, -min-trades {WF_MIN_TRADES}) вокруг точки пошаговой калибровки: оси "
                          + ", ".join(f"{k} {g[k]}" for k in WF_AXES) + f", остальное прибито к точке. Запуск: {c}. РЕЗУЛЬТАТ: прогон не делался.",
              "phases": [{"name": "wf", "grid": g}]})
    print(gp)
    print(c)


def verdict(md):
    s = open(md).read()
    f = lambda label: re.search(rf"\| {label} \| ([^|]+) \|", s).group(1).strip()
    n, pf = int(f("Всего сделок")), float(f("Profit factor"))
    gp, gl = (float(x) for x in f("Gross profit / loss").split(" / "))
    net = gp - gl if gl >= 0 else gp + gl
    print(f"пул OOS: {n} сделок, PF {pf:.3f}, net {net:+.0f} ₽, compounded {f('Compounded return')}")
    ok = pf >= GATE_PF and net > 0 and n >= GATE_TRADES
    print("ГОДЕН" if ok else f"НЕ ГОДЕН (нужно PF ≥ {GATE_PF}, net > 0, сделок ≥ {GATE_TRADES})")


def literal(t):
    _, st = state(t)
    for k, v in st["params"].items():
        print(f"\t\t{k}: {v},")


if __name__ == "__main__":
    a = sys.argv
    if len(a) < 3:
        sys.exit(__doc__)
    {"init": lambda: init(a[2].upper()), "grid": lambda: grid(a[2].upper(), a[3]),
     "pick": lambda: pick(a[2].upper(), a[3], a[4]), "status": lambda: status(a[2].upper()),
     "wf": lambda: wf(a[2].upper()), "verdict": lambda: verdict(a[2]),
     "literal": lambda: literal(a[2].upper())}.get(a[1], lambda: sys.exit(__doc__))()
