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
  steps.py best    <T>                 the best combo of the whole calibration (top PF over every step's
                                          report, ≥ BEST_MIN_TRADES trades): it goes into the literal;
                                          saved as "best" in point.json
  steps.py wf      <T>                 -> cal/wf_rolling.json: the walk-forward grid around the best combo
  steps.py verdict <walkforward.md>    the gate: the pooled OOS and the last FRESH_FOLDS folds in profit
  steps.py literal <T>                 the point as a Go literal body for core.Params

The entry has two modes, each calibrated on its own from the open start (pass the flag anywhere on
the line, every command but verdict):
  (none)       stochastic — %K under 20 (the core's zone)  files cal/,         reports <step>/
  --no-stoch   off        — StochK 0, the RSI cross down    files cal_nostoch/, reports nostoch_<step>/
With the gate the signal is any hour on which the RSI is under RSIArm and %K is under 20 — neither
waits for the other. There is no cross of %K over %D and no tunable level (owner 2026-10-26). The off
mode drops the trigger steps (the stochastic lengths do nothing without the gate).
"""
import json, os, re, statistics, sys

# Core defaults (core.DefaultParams) with every filter and optional exit off: the start of the descent.
# The filters start off so the signal and trigger steps see the biggest sample; each filter is switched
# on later only when it pays for itself.
START = dict(HourRSIPeriod=6, RSIArm=30, RSIFloor=0, RSIExit=70,
             RSITrailFrom=0, RSITrailTo=70, RSITrailStep=10,
             StochK=14, StochSmooth=1, StochD=3,
             VolumeMult=0, VolBars=20, MinRoomATR=0,
             StopATR=0.5, MaxStopPct=0, TargetR=2, TrendEMA=0, DailyATRPeriod=14)
# The entry mode: the stochastic gate (default) or none (--no-stoch: StochK 0 turns the gate off).
# The cross mode was removed with StochBelow and StochCross from the core (owner 2026-10-26).
if "--cross" in sys.argv:
    sys.exit("режима --cross больше нет: крест %K над %D и StochBelow/StochCross убраны из ядра 2026-10-26")
MODE = "level"
if "--no-stoch" in sys.argv:
    sys.argv.remove("--no-stoch")
    MODE = "nostoch"
if MODE == "nostoch":
    START["StochK"] = 0
# Never swept: %D (it does not gate), the volume base and the ATR period (the RSI ladder is swept whole
# on the exits step).
FIXED = ("StochD", "VolBars", "DailyATRPeriod")

# (name, axes). Order is the method: first what makes a setup (the most trades), then the trigger, then
# the filters one at a time (each removes trades and must earn its place), then the payoff and the exits
# on the final set of trades, then a second pass over the signal, the trigger and the payoff with
# everything else in place.
SIGNAL = {"HourRSIPeriod": [4, 5, 6, 8], "RSIArm": [25, 30, 35]}
# The stochastic lengths (owner 2026-10-26): from the start 14/1. StochD stays at 3: the gate compares
# %K alone with 20, %D only shows in the entry reason.
TRIGGER = {"StochK": [8, 9, 10, 11, 12, 13, 14], "StochSmooth": [1, 2, 3]}
RISK = {"StopATR": [0.3, 0.5, 0.7, 1.0], "TargetR": [1.5, 2, 3], "MaxStopPct": [0, 1, 1.5]}
STEPS = [
    ("signal", SIGNAL),
    ("trigger", TRIGGER),
    # TrendEMA 0 is banned after this step (owner 2026-10-26): the trend filter is always on. It starts
    # at 0 so the signal and trigger steps see the biggest sample; the trend step then must pick one of
    # its periods — the off value does not compete and the 5% rule does not keep the point at 0.
    ("trend", {"TrendEMA": [100, 150, 170, 200]}),
    ("floor", {"RSIFloor": [0, 15, 20, 22]}),
    ("volume", {"VolumeMult": [0, 0.5, 0.8, 1.2]}),
    ("room", {"MinRoomATR": [0, 0.3, 0.6]}),
    ("risk", RISK),
    # The soft exits together: the RSI leaving the upper zone (the cross of RSIExit down) and the RSI ladder (RSITrailFrom 0 — off; 50 — the
    # owner's 50/60/70 ladder; 60 — from the second rung) interact, so they are swept jointly. The
    # ladder step (owner 2026-10-26) sets both the rung spacing and how far under a rung the exit sits;
    # the first rung 40 and the top rung 80 (owner 2026-10-26) give the 40/60/80 ladder of step 20 with
    # exits at 20/40/60. The calibrator sweeps a cartesian grid, so they apply to every step.
    # RSIExit 0 is banned (owner 2026-10-26): the strategy is conservative — many small winners and small
    # losers — so the exit on the RSI leaving the upper zone always stays on, even at a lower PF.
    ("exits", {"RSIExit": [65, 70, 75], "RSITrailFrom": [0, 40, 50, 60], "RSITrailStep": [10, 15, 20],
               "RSITrailTo": [70, 80]}),
    ("signal2", SIGNAL),
    ("trigger2", TRIGGER),
    ("risk2", RISK),
]
# The walk-forward grid: one step either side of the point on these axes (≤ 3^5 = 243 points). The
# walk-forward simulates periodic recalibration: each fold picks its own values on its train window, so
# the setup, the trigger and the payoff all stay free per fold; the filters stay pinned at the point.
WF_AXES = ("HourRSIPeriod", "RSIArm", "StochK", "StopATR", "TargetR")
if MODE == "nostoch":
    STEPS = [(n, ax) for n, ax in STEPS if not n.startswith("trigger")]
    WF_AXES = tuple(k for k in WF_AXES if not k.startswith("Stoch"))
VARIANT = "" if MODE == "level" else MODE
FLAG = {"level": "", "nostoch": " --no-stoch"}[MODE]
TITLE = {"level": "стохастик: %K ниже 20", "nostoch": "без стохастика (StochK 0)"}[MODE]

MONTHS = 36
MIN_TRADES = 20      # ≈ 7 a year over 36 months; fewer — the combo is out of the step's choice
BEST_MIN_TRADES = 10 # the best combo for the literal: the owner takes the top PF below 20 trades too, but
                     # under the walk-forward gate's 10 trades a PF is a couple of lucky trades
MOVE_SHARE = 0.05    # the point moves only when the new centre beats the current one by 5%
PF_CAP = 10.0        # an inf PF (no losers) counts as this in a median
WF_MIN_TRADES = 3
GATE_PF, GATE_TRADES = 1.2, 10
FRESH_FOLDS = 12     # the freshest 12 one-month test folds (the last year) must make money together
FRESH_PF = 1.1      # ...and their pooled PF must reach this (owner 2026-10-26, SVCB: +1.5% on PF ~1.0 passed)
BIN = "reports/pdl_rsi_cal/backtest"
# Axes whose off value (0) is banned (owner 2026-10-26): the current 0 never joins their grid, and their
# step always moves the point to the best centre, ignoring MOVE_SHARE.
FORCED = ("TrendEMA",)


def caldir(t):
    d = f"data/params/pdl_rsi/{t.lower()}/cal" + (f"_{VARIANT}" if VARIANT else "")
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
        sys.exit(f"нет {p}: сначала steps.py init {t}{FLAG}")
    st = load(p)
    if st.get("variant", "") != VARIANT:
        sys.exit(f"{p}: режим {st.get('variant') or 'level'}, а команда — {MODE}")
    return p, st


def cmd(t, grid, out, extra=""):
    return (f"taskset -c 0 nice -n 19 env GOMAXPROCS=1 {BIN} -ticker {t} -strategy pdl_rsi -interval Hour1 "
            f"-risk-pct 1 -metric profit_factor -months {MONTHS} -calibrate {grid} -min-trades 1 -combos 0{extra} "
            f"-out reports/{t}_pdl_rsi/{VARIANT + '_' if VARIANT else ''}{out}")


def init(t):
    p = f"{caldir(t)}/point.json"
    if os.path.exists(p):
        sys.exit(f"{p} уже есть: калибровка начата, продолжай с steps.py status {t}")
    dump(p, {"_comment": f"pdl_rsi {t}: текущая точка пошаговой калибровки (steps.py{FLAG}). Шаги: нет.",
             "variant": VARIANT, "_steps": [], "params": START})
    print(p)
    print(f"дальше: steps.py grid {t} {STEPS[0][0]}{FLAG}")


def grid(t, name):
    i = step_index(name)
    _, st = state(t)
    done = [s["step"] for s in st["_steps"]]
    want = [n for n, _ in STEPS[:i]]
    if done != want:
        sys.exit(f"шаг {name}: до него должны быть выбраны {want}, выбраны {done}")
    p = st["params"]
    if i > step_index("trend") and p["TrendEMA"] == 0:
        sys.exit(f"точка с TrendEMA 0 после шага trend: фильтр тренда обязателен (владелец 2026-10-26) — начни калибровку заново с steps.py init {t}{FLAG}")
    if p["RSIExit"] == 0:
        sys.exit(f"точка с RSIExit 0: выход по RSI обязателен (владелец 2026-10-26) — начни калибровку заново с steps.py init {t}{FLAG}")
    g = {k: [v] for k, v in p.items()}
    axes = {}
    for k, vals in STEPS[i][1].items():
        # the current value always competes, except a banned off value
        axes[k] = sorted(set(vals) | ({p[k]} if k not in FORCED else set()))
        g[k] = axes[k]
    if 0 in g["TrendEMA"] and i >= step_index("trend"):
        sys.exit("TrendEMA 0 в сетке запрещён: фильтр тренда обязателен (владелец 2026-10-26)")
    if 0 in g["RSIExit"]:
        sys.exit("RSIExit 0 в сетке запрещён: выход по RSI обязателен (владелец 2026-10-26)")
    gp = grid_path(t, name)
    c = cmd(t, gp, name)
    dump(gp, {"_comment": f"pdl_rsi {t} Hour1, шаг {i + 1} «{name}» пошаговой калибровки, {TITLE} ({MONTHS} мес, -risk-pct 1): оси "
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

    forced = any(k in FORCED and cur[k] == 0 for k in axes)
    if forced:
        rows = [r for r in rows if all(r[4][k] != 0 for k in axes if k in FORCED)]
    ok = [r for r in rows if r[3] >= MIN_TRADES]
    print(f"шаг {i + 1} «{name}»: {len(rows)} комбинаций, с ≥ {MIN_TRADES} сделками — {len(ok)}"
          + ("; выключенное значение запрещено — выбор обязателен" if forced else ""))
    for r in sorted(ok or (rows if forced else []), key=lambda r: (-score(r), -r[3]))[:5]:
        print(f"  #{r[0]}: медиана окрестности {score(r):.3f}, PF {r[1]:.3f}, net {r[2]:+.0f}, сделок {r[3]} — {tag(r[4])}")
    if forced:
        # the off value is banned and not in the grid: the best centre wins outright, MOVE_SHARE aside
        best = max(ok or rows, key=lambda r: (score(r), r[3], -r[0]))
        why = "выключенное значение запрещено — выбран лучший центр" + (
            "" if ok else f" (ни у одной комбинации нет {MIN_TRADES} сделок)")
    else:
        here = next(r for r in rows if all(r[4][k] == cur[k] for k in axes))
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
    st["_comment"] = f"pdl_rsi {t}: текущая точка пошаговой калибровки (steps.py{FLAG}). Шаги: " + "; ".join(
        f"{s['step']} — {s['result']}" for s in st["_steps"]) + "."
    dump(sp, st)
    gp = grid_path(t, name)
    g = load(gp)
    g["_comment"] = g["_comment"].replace("РЕЗУЛЬТАТ: прогон не делался.", "") + f" РЕЗУЛЬТАТ: {res}."
    dump(gp, g)
    nxt = STEPS[i + 1][0] if i + 1 < len(STEPS) else None
    print(f"дальше: steps.py grid {t} {nxt}{FLAG}" if nxt else f"шаги пройдены; дальше: steps.py best {t}{FLAG}")


def status(t):
    _, st = state(t)
    for s in st["_steps"]:
        print(f"  {s['step']}: {s['result']}")
    done = len(st["_steps"])
    print("точка:", ", ".join(f"{k} {v:g}" for k, v in st["params"].items()))
    print(f"дальше: steps.py grid {t} {STEPS[done][0]}{FLAG}" if done < len(STEPS) else f"шаги пройдены; дальше: steps.py best {t}{FLAG}")


def best(t):
    sp, st = state(t)
    if len(st["_steps"]) < len(STEPS):
        sys.exit("лучшая комбинация — после всех шагов")
    top = None
    for name, _ in STEPS:
        d = f"reports/{t}_pdl_rsi/{VARIANT + '_' if VARIANT else ''}{name}"
        mds = sorted(f for f in os.listdir(d) if f.endswith("_calibration.md")) if os.path.isdir(d) else []
        if not mds:
            sys.exit(f"{d}: нет отчёта calibration.md шага {name}")
        for r in combos(f"{d}/{mds[-1]}"):
            if r[4].get("RSIExit", 1) == 0:
                continue
            if r[4].get("TrendEMA", 1) == 0:
                continue  # banned (owner 2026-10-26): the signal and trigger steps run with the filter off  # banned (owner 2026-10-26): a report from before the ban may still hold it
            if r[3] >= BEST_MIN_TRADES and (top is None or (r[1], r[3], r[2]) > (top[1], top[3], top[2])):
                top = (name, *r[1:])
    name, pf, net, n, p = top
    full = dict(st["params"], **p)
    diff = [k for k in full if full[k] != st["params"][k]]
    st["best"] = {"step": name, "pf": pf, "net": net, "trades": n, "params": full}
    dump(sp, st)
    print(f"лучшая комбинация: шаг {name}, PF {pf:.3f}, net {net:+.0f}, сделок {n}"
          + (f" (меньше {MIN_TRADES} — спуск её не выбирал)" if n < MIN_TRADES else ""))
    print("отличие от точки спуска: " + (", ".join(f"{k} {st['params'][k]:g} → {full[k]:g}" for k in diff) or "нет"))
    print(f"дальше: steps.py wf {t}{FLAG}")


def wf(t):
    _, st = state(t)
    if "best" not in st:
        sys.exit(f"walk-forward — вокруг литерала, сначала steps.py best {t}{FLAG}")
    p = st["best"]["params"]
    g = {k: [v] for k, v in p.items()}
    for k in WF_AXES:
        vals = sorted({v for _, ax in STEPS for a, vv in ax.items() if a == k for v in vv} | {p[k]})
        j = vals.index(p[k])
        g[k] = vals[max(j - 1, 0): j + 2]
    gp = f"{caldir(t)}/wf_rolling.json"
    c = cmd(t, gp, "wf", f" -train-months 3 -test-months 1").replace("-min-trades 1", f"-min-trades {WF_MIN_TRADES}")
    dump(gp, {"_comment": f"pdl_rsi {t} Hour1, {TITLE}, скользящий walk-forward (подбор 3 мес, проверка 1 мес, шаг 1 мес, {MONTHS} мес, -risk-pct 1, -min-trades {WF_MIN_TRADES}) вокруг лучшей комбинации калибровки (литерал), каждый фолд выбирает свои значения (имитация периодической перекалибровки): оси "
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
    # A profitable old year can carry the 36-month pool while the ticker loses today: the freshest year
    # of folds must be in profit on its own, and by a margin — a flat year (PF ~1) is not an edge.
    folds = [(int(m.group(1)), float(m.group(2)), m.group(3)) for m in re.finditer(
        r"^\| \d+ \| [^|]+ \| [^|]+ \| [^|]+ \| [^|]+ \| (\d+) \| ([-\d.]+)% \| [^|]+ \|(?: ([\d.]+ / [\d.]+) \|)?", s, re.M)]
    if any(g is None for _, _, g in folds):
        sys.exit("отчёт без колонки OOS gross +/− по фолдам: пересобери бинарь и перезапусти walk-forward")
    fresh = folds[-FRESH_FOLDS:]
    comp = 1.0
    for _, r, _ in fresh:
        comp *= 1 + r / 100
    up = sum(r > 0 for _, r, _ in fresh)
    fg = sum(float(g.split(" / ")[0]) for _, _, g in fresh)
    fl = sum(float(g.split(" / ")[1]) for _, _, g in fresh)
    fpf = fg / fl if fl > 0 else (PF_CAP if fg > 0 else 0.0)
    print(f"последние {len(fresh)} фолдов: {sum(k for k, _, _ in fresh)} сделок, PF {fpf:.3f}, compounded {100 * (comp - 1):+.2f}%, "
          f"в плюсе {up} из {len(fresh)}")
    pool_ok = pf >= GATE_PF and net > 0 and n >= GATE_TRADES
    fresh_ok = len(fresh) == FRESH_FOLDS and comp > 1 and fpf >= FRESH_PF
    if pool_ok and fresh_ok:
        print("ГОДЕН")
    else:
        why = ([] if pool_ok else [f"пул: нужно PF ≥ {GATE_PF}, net > 0, сделок ≥ {GATE_TRADES}"]) + \
              ([] if fresh_ok else [f"последние {FRESH_FOLDS} фолдов: нужно PF ≥ {FRESH_PF} и compounded > 0"])
        print("НЕ ГОДЕН (" + "; ".join(why) + ")")
    # The fold picks: different values across folds are the expected answer to a changing market, not a
    # flaw. The last fold's pick is the one fitted to the freshest months.
    moved = re.findall(r"^- (\w+): (.+)$", s.split("**Гуляли**", 1)[1], re.M) if "**Гуляли**" in s else []
    if not moved:
        print("фолды выбрали одни и те же значения на всех осях")
    for k, vals in moved:
        v = [x.strip() for x in vals.split(",")]
        share = ", ".join(f"{x} ×{v.count(x)}" for x in sorted(set(v), key=lambda x: -v.count(x)))
        print(f"  {k}: {share}; последний фолд — {v[-1]}")


def literal(t):
    _, st = state(t)
    if "best" not in st:
        sys.exit(f"литерал — лучшая комбинация, сначала steps.py best {t}{FLAG}")
    for k, v in st["best"]["params"].items():
        print(f"\t\t{k}: {v},")


if __name__ == "__main__":
    a = sys.argv
    if len(a) < 3:
        sys.exit(__doc__)
    {"init": lambda: init(a[2].upper()), "grid": lambda: grid(a[2].upper(), a[3]),
     "pick": lambda: pick(a[2].upper(), a[3], a[4]), "status": lambda: status(a[2].upper()),
     "best": lambda: best(a[2].upper()), "wf": lambda: wf(a[2].upper()), "verdict": lambda: verdict(a[2]),
     "literal": lambda: literal(a[2].upper())}.get(a[1], lambda: sys.exit(__doc__))()
