"""Read drop_bounce backtest reports and apply the skill's gates (hold-out scheme, owner 2026-10-14).

Usage (from the repo root):
  report.py center <sweep_calibration.md>        every combo scored by the median PF of itself and its
                                                grid neighbours; prints the winner's #k and plateau verdict
  report.py recent <dev1.md> [dev2.md ...]       the tie-break among CANDIDATES: each candidate's dev run,
                                                PF and net over the last 12 and 6 months of the dev window
  report.py stress <dev.md>                      the point on the development window (-holdout-months 6):
                                                PF, trades, without the best 1 / 3 trades, 6-month blocks
  report.py cost <dev_cost.md>                   the same window with -commission 0.001
  report.py holdout <dev.md> <full_trades.csv>   trades of the 36-month run entered after the development
                                                window's end: opened once, after the point is fixed

Gates on the development window (30 months):
  plateau  every grid neighbour of the point (one step on one axis) has PF >= 1.0;
  S1 PF >= 1.3 and trades >= 25 (about 10 a year is enough, owner 2026-10-14);
  S2 PF without the best trade >= 1.15;
  S3 PF without the three best trades >= 1.0;
  S4 at least 4 of the 5 six-month blocks have net > 0;
  cost PF >= 1.2 and net > 0 with -commission 0.001.
Tie-break (owner 2026-10-15: price behaviour drifts and the ticker is recalibrated often): combos whose
neighbourhood median is within TIE_SHARE of the best, with a plateau, are CANDIDATES; among them the one
with the highest PF over the last RECENT_MONTHS of the development window wins if its net there is > 0.
The hold-out is never used for it.
Hold-out: net > 0 (6 months hold ~5–10 trades: a sanity check, not a PF measurement).
"""
import csv, re, statistics, sys
from datetime import datetime, timezone

MIN_TRADES = 25
MIN_NEIGHBOUR_TRADES = 10
TIE_SHARE = 0.10
MAX_CANDIDATES = 5
RECENT_MONTHS = 12


def pf(v):
    g = sum(x for x in v if x > 0)
    l = -sum(x for x in v if x < 0)
    return g / l if l else float("inf") if g else 0.0


def combos(md):
    s = open(md).read()
    for m in re.finditer(r"^### #(\d+) — profit_factor ([\d.]+), сделок (\d+)\n(.*?)(?=^### #|\Z)", s, re.M | re.S):
        p = {k: float(v) for k, v in re.findall(r"^\| (\w+) \| ([-\d.]+) \|", m.group(4), re.M)}
        yield int(m.group(1)), float(m.group(2)), int(m.group(3)), p


def center(md):
    total = int(re.search(r"Всего комбинаций: (\d+)", open(md).read()).group(1))
    rows = list(combos(md))
    if len(rows) < total:
        sys.exit(f"в отчёте {len(rows)} из {total} комбинаций — запусти перебор с -combos 0")
    axes = {k: sorted({r[3][k] for r in rows}) for k in rows[0][3] if len({r[3][k] for r in rows}) > 1}

    def neighbours(p):
        out = []
        for r in rows:
            diff = [k for k in axes if r[3][k] != p[k]]
            if len(diff) == 1:
                k = diff[0]
                if abs(axes[k].index(r[3][k]) - axes[k].index(p[k])) == 1:
                    out.append(r)
        return out

    scored = []
    for k, f, n, p in rows:
        if n < MIN_TRADES:
            continue
        nb = neighbours(p)
        scored.append((statistics.median([f] + [r[1] for r in nb]), k, f, n, p, nb))
    if not scored:
        sys.exit(f"ни одной комбинации с ≥ {MIN_TRADES} сделками")
    scored.sort(key=lambda x: -x[0])
    print("топ-5 по медиане окрестности:")
    for med, k, f, n, p, _ in scored[:5]:
        print(f"  #{k}: медиана {med:.3f}, PF {f:.3f}/{n}, " + ", ".join(f"{a} {p[a]:g}" for a in axes))
    med, k, f, n, p, nb = scored[0]
    print(f"ЦЕНТР: #{k} — " + ", ".join(f"{a} {p[a]:g}" for a in axes) + f", PF {f:.3f} на {n} сделках")
    for r in nb:
        d = next(a for a in axes if r[3][a] != p[a])
        print(f"  сосед {d} {r[3][d]:g}: PF {r[1]:.3f}/{r[2]}")
    weak = [r for r in nb if r[2] < MIN_NEIGHBOUR_TRADES]
    if weak:
        print(f"  (у {len(weak)} соседей < {MIN_NEIGHBOUR_TRADES} сделок — их PF шумный)")
    ok = all(r[1] >= 1.0 for r in nb)
    print("ПЛАТО ЕСТЬ" if ok else "ОСТРЫЙ ПИК")
    cands = [x for x in scored if x[0] >= med * (1 - TIE_SHARE) and all(r[1] >= 1.0 for r in x[5])][:MAX_CANDIDATES]
    if ok and len(cands) > 1:
        print(f"КАНДИДАТЫ (медиана в пределах {TIE_SHARE:.0%} от лучшей, с плато): " + " ".join(f"#{x[1]}" for x in cands))
        print("спорная ситуация: прогони каждого кандидата на окне разработки и выбери по свежим месяцам:")
        for x in cands:
            print(f"  grids.py cand <T> Minutes30 {md} {x[1]}")
        print("  report.py recent <dev каждого кандидата, центр первым>")
    else:
        print(f"дальше: grids.py point <T> Minutes30 {md} {k}")


def period(md):
    m = re.search(r"Период: (\d{4}-\d\d-\d\d \d\d:\d\d) — (\d{4}-\d\d-\d\d \d\d:\d\d)", open(md).read())
    f = lambda s: datetime.strptime(s, "%Y-%m-%d %H:%M").astimezone()
    return f(m.group(1)), f(m.group(2))


def trades(csv_path):
    out = []
    with open(csv_path) as fh:
        for r in csv.DictReader(fh):
            t = datetime.strptime(r["entry_time"], "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=timezone.utc)
            out.append((t, float(r["pnl"])))
    return out


def add_months(d, n):
    y, m = divmod(d.month - 1 + n, 12)
    return d.replace(year=d.year + y, month=m + 1, day=min(d.day, 28))


def stress(md):
    start, end = period(md)
    tr = trades(md[:-3] + "_trades.csv")
    p = [x for _, x in tr]
    s = sorted(p, reverse=True)
    f, f1, f3 = pf(p), pf(s[1:]), pf(s[3:])
    print(f"окно разработки {start:%Y-%m-%d} — {end:%Y-%m-%d}: {len(p)} сделок, PF {f:.3f}, net {sum(p):+.0f} ₽")
    blocks, b0 = [], start
    while b0 < end:
        b1 = min(add_months(b0, 6), end)
        v = [x for t, x in tr if b0 <= t < b1]
        blocks.append((b0, b1, v))
        b0 = b1
    for b0, b1, v in blocks:
        print(f"  {b0:%Y-%m} — {b1:%Y-%m}: {len(v)} сделок, PF {pf(v):.3f}, net {sum(v):+.0f} ₽")
    plus = sum(1 for *_, v in blocks if sum(v) > 0)
    need = len(blocks) - 1
    checks = [
        (f"S1 PF {f:.3f} ≥ 1.3 и {len(p)} ≥ {MIN_TRADES} сделок", f >= 1.3 and len(p) >= MIN_TRADES),
        (f"S2 без лучшей сделки ({s[0]:+.0f} ₽) PF {f1:.3f} ≥ 1.15", f1 >= 1.15),
        (f"S3 без трёх лучших PF {f3:.3f} ≥ 1.0", f3 >= 1.0),
        (f"S4 полугодий в плюсе {plus} из {len(blocks)} (нужно ≥ {need})", plus >= need),
    ]
    for t, ok in checks:
        print(f"{t}: {'да' if ok else 'нет'}")
    print("СТРЕСС ВЫДЕРЖАН" if all(ok for _, ok in checks) else "СТРЕСС НЕ ВЫДЕРЖАН")


def recent(mds):
    rows = []
    for md in mds:
        start, end = period(md)
        tr = trades(md[:-3] + "_trades.csv")
        from12, from6 = add_months(end, -RECENT_MONTHS), add_months(end, -6)
        v12 = [x for t, x in tr if t >= from12]
        v6 = [x for t, x in tr if t >= from6]
        rows.append((md, pf([x for _, x in tr]), len(tr), pf(v12), sum(v12), len(v12), pf(v6), sum(v6), len(v6)))
        print(f"{md}: всё окно PF {rows[-1][1]:.3f}/{len(tr)}; последние {RECENT_MONTHS} мес PF {pf(v12):.3f}, "
              f"net {sum(v12):+.0f} ₽, {len(v12)} сделок; последние 6 мес PF {pf(v6):.3f}, net {sum(v6):+.0f} ₽, {len(v6)} сделок")
    live = [r for r in rows if r[4] > 0]
    if not live:
        print(f"СВЕЖИЕ МЕСЯЦЫ В МИНУСЕ У ВСЕХ — остаётся центр: {rows[0][0]}")
        return
    best = max(live, key=lambda r: (r[3], r[7]))
    print(f"ВЫБОР ПО СВЕЖИМ МЕСЯЦАМ: {best[0]}")


def cost(md):
    s = open(md).read()
    f = float(re.search(r"\| Profit factor \| ([\d.]+) \|", s).group(1))
    net = float(re.search(r"\| Чистый PnL \| ([-\d.]+)", s).group(1))
    n = int(re.search(r"\| Всего сделок \| (\d+) \|", s).group(1))
    print(f"удвоенная комиссия: PF {f:.3f}, net {net:.0f} ₽ на {n} сделках")
    print("ИЗДЕРЖКИ ВЫДЕРЖАНЫ" if f >= 1.2 and net > 0 else "ИЗДЕРЖКИ СЪЕДАЮТ КРАЙ")


def holdout(md, csv_path):
    _, end = period(md)
    v = [x for t, x in trades(csv_path) if t >= end]
    print(f"hold-out с {end:%Y-%m-%d}: {len(v)} сделок, PF {pf(v):.3f}, net {sum(v):+.0f} ₽, "
          f"лучшая {max(v, default=0):+.0f}, худшая {min(v, default=0):+.0f}")
    print("HOLD-OUT В ПЛЮСЕ" if sum(v) > 0 else "HOLD-OUT В МИНУСЕ")


if __name__ == "__main__":
    a = sys.argv
    {"center": lambda: center(a[2]), "stress": lambda: stress(a[2]), "cost": lambda: cost(a[2]),
     "holdout": lambda: holdout(a[2], a[3]), "recent": lambda: recent(a[2:])}.get(a[1] if len(a) > 1 else "", lambda: sys.exit(__doc__))()
