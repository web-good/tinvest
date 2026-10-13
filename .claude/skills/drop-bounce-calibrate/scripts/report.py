"""Read drop_bounce backtest reports and apply the skill's gates.

Usage (from the repo root):
  report.py wf <walkforward.md>                 pooled OOS, folds, gate verdict
  report.py plateau <calib.md> <point.json>     every combo's PF vs the point, plateau verdict
  report.py cost <backtest.md>                  PF and net under doubled commission, verdict

WF gate (owner, 2026-10-13), judged on the pooled trades, not on a count of folds:
  A pooled OOS PF >= 1.2 and pooled trades >= 30;
  B the folds' OOS NetPnL% summed without the best fold > 0 (no single half-year carries it);
  C the last two folds' OOS NetPnL% summed > 0 (the edge has not faded in the last 12 months);
  D every fold with >= 10 OOS trades has PF >= 0.8 (no big fold deep in the red).
Plateau: every neighbour PF >= 1.0 and the neighbours' median PF >= 0.7 x the point's PF.
Cost: PF >= 1.2 and net PnL > 0 with -commission 0.001.
"""
import json, re, statistics, sys


def wf(md):
    s = open(md).read()
    pf = float(re.search(r"\| Profit factor \| ([\d.]+) \|", s).group(1))
    n = int(re.search(r"\| Всего сделок \| (\d+) \|", s).group(1))
    comp = re.search(r"\| Compounded return \| ([-\d.]+)% \|", s).group(1)
    folds = re.findall(r"^\| \d \| [^|]+\| [^|]+\| [\d.]+ \| ([\d.]+) \| (\d+) \| ([-\d.]+)% \| ([\d.]+)% \|", s, re.M)
    nets = [float(r) for _, _, r, _ in folds]
    a = pf >= 1.2 and n >= 30
    b = sum(nets) - max(nets) > 0
    c = sum(nets[-2:]) > 0
    big = [(float(p), int(t)) for p, t, _, _ in folds if int(t) >= 10]
    d = all(p >= 0.8 for p, _ in big)
    ok = a and b and c and d
    fs = " / ".join(f"{float(p):.3f}({t}) net {r}% dd {dd}%" for p, t, r, dd in folds)
    print(f"pooled PF {pf:.3f} на {n} сделках, compounded {comp}%")
    print(f"фолды: {fs}")
    print(f"A pooled PF ≥ 1.2 и ≥ 30 сделок: {'да' if a else 'нет'}")
    print(f"B net без лучшего фолда {sum(nets) - max(nets):+.2f}% > 0: {'да' if b else 'нет'}")
    print(f"C net двух последних фолдов {sum(nets[-2:]):+.2f}% > 0: {'да' if c else 'нет'}")
    print(f"D фолды с ≥ 10 сделками PF ≥ 0.8 {[round(p, 3) for p, _ in big]}: {'да' if d else 'нет'}")
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
        diff = {k: p[k] for k in ("RSIPeriod", "RSIZone", "RSICap", "EMAPeriod", "DaySpentATR", "StopDailyATR", "RSIExit") if abs(p[k] - pt[k]) > 1e-9}
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
    print("ИЗДЕРЖКИ ВЫДЕРЖАНЫ" if pf >= 1.2 and net > 0 else "ИЗДЕРЖКИ СЪЕДАЮТ КРАЙ")


if __name__ == "__main__":
    {"wf": lambda: wf(sys.argv[2]), "plateau": lambda: plateau(sys.argv[2], sys.argv[3]),
     "cost": lambda: cost(sys.argv[2])}.get(sys.argv[1] if len(sys.argv) > 1 else "", lambda: sys.exit(__doc__))()
