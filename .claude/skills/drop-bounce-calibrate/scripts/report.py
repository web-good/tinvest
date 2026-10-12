"""Read drop_bounce backtest reports and apply the skill's gates.

Usage (from the repo root):
  report.py wf <walkforward.md>                 pooled OOS, folds, gate verdict
  report.py plateau <calib.md> <point.json>     every combo's PF vs the point, plateau verdict
  report.py cost <backtest.md>                  PF and net under doubled commission, verdict

Degenerate fold: fewer than 5 OOS trades or no losing trade (OOS PF >= 1000).
WF gate: pooled OOS PF >= 1.5, pooled trades >= 20, >= 3 non-degenerate folds and all but at most
one of them above 1.0 (3 of 4, or 3 of 3).
Plateau: every neighbour PF >= 1.0 and the neighbours' median PF >= 0.7 x the point's PF.
Cost: PF >= 1.3 and net PnL > 0 with -commission 0.001.
"""
import json, re, statistics, sys


def wf(md):
    s = open(md).read()
    pf = float(re.search(r"\| Profit factor \| ([\d.]+) \|", s).group(1))
    n = int(re.search(r"\| Всего сделок \| (\d+) \|", s).group(1))
    comp = re.search(r"\| Compounded return \| ([-\d.]+)% \|", s).group(1)
    folds = re.findall(r"^\| \d \| [^|]+\| [^|]+\| [\d.]+ \| ([\d.]+) \| (\d+) \| ([-\d.]+)% \| ([\d.]+)% \|", s, re.M)
    live = [(float(p), int(t)) for p, t, _, _ in folds if int(t) >= 5 and float(p) < 1000]
    above = sum(p > 1.0 for p, _ in live)
    ok = pf >= 1.5 and n >= 20 and len(live) >= 3 and above >= len(live) - 1 and above >= 3
    fs = " / ".join(f"{float(p):.3f}({t}){'*' if int(t) < 5 or float(p) >= 1000 else ''} dd {dd}%" for p, t, _, dd in folds)
    print(f"pooled PF {pf:.3f} на {n} сделках, compounded {comp}%")
    print(f"фолды: {fs}")
    print(f"живых фолдов {len(live)}, выше 1.0: {above}")
    walk = re.findall(r"^- ([A-Za-z]+): (.+)$", s, re.M)
    if walk:
        print("гуляли: " + "; ".join(f"{k} {v}" for k, v in walk))
    print("ПЛАНКА ВЗЯТА" if ok else "ПЛАНКА НЕ ВЗЯТА")


def combos(md):
    s = open(md).read()
    for m in re.finditer(r"^### #\d+ — profit_factor ([\d.]+), сделок (\d+)\n(.*?)(?=^### #|\Z)", s, re.M | re.S):
        p = {k: float(v) for k, v in re.findall(r"^\| (\w+) \| ([-\d.]+) \|", m.group(3), re.M)}
        yield float(m.group(1)), int(m.group(2)), p


def plateau(md, point):
    pt = {k: float(v) for k, v in json.load(open(point)).items()}
    total = int(re.search(r"Всего комбинаций: (\d+)", open(md).read()).group(1))
    rows = list(combos(md))
    if len(rows) < total:
        sys.exit(f"в отчёте {len(rows)} из {total} комбинаций — сетка плато должна быть ≤ 20")
    centre = [r for r in rows if all(abs(r[2].get(k, v) - v) < 1e-9 for k, v in pt.items())]
    if not centre:
        sys.exit("точки нет в сетке плато")
    cpf = centre[0][0]
    nb = [r for r in rows if r is not centre[0]]
    for pf, n, p in rows:
        diff = {k: p[k] for k in ("MinDropPct", "RSIEntry", "StopDailyATR", "MaxDayRangeATR", "VolMult") if abs(p[k] - pt[k]) > 1e-9}
        print(f"  PF {pf:.3f} / {n}  {diff or 'ТОЧКА'}")
    med = statistics.median(r[0] for r in nb)
    ok = all(r[0] >= 1.0 for r in nb) and med >= 0.7 * cpf
    print(f"точка {cpf:.3f}, медиана соседей {med:.3f} (порог {0.7 * cpf:.3f}), минимум {min(r[0] for r in nb):.3f}")
    print("ПЛАТО ЕСТЬ" if ok else "ОСТРЫЙ ПИК")


def cost(md):
    s = open(md).read()
    pf = float(re.search(r"\| Profit factor \| ([\d.]+) \|", s).group(1))
    net = float(re.search(r"\| Чистый PnL \| ([-\d.]+)", s).group(1))
    n = int(re.search(r"\| Всего сделок \| (\d+) \|", s).group(1))
    print(f"удвоенная комиссия: PF {pf:.3f}, net {net:.0f} ₽ на {n} сделках")
    print("ИЗДЕРЖКИ ВЫДЕРЖАНЫ" if pf >= 1.3 and net > 0 else "ИЗДЕРЖКИ СЪЕДАЮТ КРАЙ")


if __name__ == "__main__":
    {"wf": lambda: wf(sys.argv[2]), "plateau": lambda: plateau(sys.argv[2], sys.argv[3]),
     "cost": lambda: cost(sys.argv[2])}.get(sys.argv[1] if len(sys.argv) > 1 else "", lambda: sys.exit(__doc__))()
