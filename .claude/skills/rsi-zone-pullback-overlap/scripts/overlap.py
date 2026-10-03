#!/usr/bin/env python3
"""Совместная работа rsi_pullback и rsi_zone на одном тикере общего счёта.

Берёт журналы сделок двух одиночных прогонов cmd/backtest (файлы *_trades.csv) одного тикера,
одного окна и одного дня и проигрывает их так, как их исполнил бы живой раннер счёта:

- занятый тикер: пока открыта позиция одной стратегии, вторая не входит;
- приоритет: если обе входят на одном баре, входит rsi_pullback;
- сделка, которой не дали войти, выпадает целиком и тикер не занимает.

Деньги: pnl_pct каждой сделки умножается на долю счёта стратегии (RSI_PULLBACK_BUY_PCT,
RSI_ZONE_BUY_PCT), получается вклад в доходность счёта в процентах; кривая — по дате выхода,
без реинвестирования.

Ограничение модели: журнал соло-прогона не знает о сигналах, которые стратегия пропустила,
потому что сама была в позиции. Если её сделку вытеснили, в живом раннере она могла бы войти
позже внутри того же отрезка — модель этого не видит. Поэтому «украденное» — оценка сверху,
а «совместный счёт» — оценка снизу для вытесненной стратегии.

  python3 overlap.py --pullback P_trades.csv --zone Z_trades.csv \
      [--pullback-pct 50] [--zone-pct 5] [--tail 2026-04-01] [--tail 2025-10-01] [--same-bar-reentry]
"""

import argparse
import csv
from collections import defaultdict
from datetime import datetime


def load(path, tag):
    out = []
    with open(path, encoding="utf-8") as f:
        for r in csv.DictReader(f):
            out.append({
                "s": tag,
                "in": datetime.fromisoformat(r["entry_time"].replace("Z", "+00:00")),
                "out": datetime.fromisoformat(r["exit_time"].replace("Z", "+00:00")),
                "pct": float(r["pnl_pct"]) * 100.0,
                "reason": r["reason"],
            })
    return out


def simulate(trades, same_bar_reentry):
    # pullback раньше zone на одном баре входа
    order = sorted(trades, key=lambda t: (t["in"], 0 if t["s"] == "P" else 1))
    busy = None
    kept, blocked = [], []
    for t in order:
        free = busy is None or t["in"] > busy["out"] or (same_bar_reentry and t["in"] == busy["out"])
        if free:
            kept.append(t)
            busy = t
        else:
            t["by"] = busy["s"]
            blocked.append(t)
    return kept, blocked


def stats(trades, w):
    """w: {'P': доля, 'Z': доля} в долях единицы. Возвращает вклад в % счёта."""
    pts = sorted(((t["out"], t["pct"] * w[t["s"]]) for t in trades), key=lambda x: x[0])
    gp = sum(v for _, v in pts if v > 0)
    gl = -sum(v for _, v in pts if v < 0)
    eq = peak = dd = 0.0
    for _, v in pts:
        eq += v
        peak = max(peak, eq)
        dd = max(dd, peak - eq)
    pf = gp / gl if gl > 0 else float("inf")
    return {"n": len(pts), "net": eq, "pf": pf, "dd": dd}


def pf_str(trades):
    gp = sum(t["pct"] for t in trades if t["pct"] > 0)
    gl = -sum(t["pct"] for t in trades if t["pct"] < 0)
    if not trades:
        return "—"
    return "inf" if gl == 0 else "%.3f" % (gp / gl)


def fmt(s):
    pf = "inf" if s["pf"] == float("inf") else "%.3f" % s["pf"]
    ratio = "inf" if s["dd"] == 0 else "%.2f" % (s["net"] / s["dd"])
    return "%4d сд  net %+7.2f%%  PF %6s  DD %5.2f%%  net/DD %s" % (s["n"], s["net"], pf, s["dd"], ratio)


def block(title, P, Z, w, same_bar):
    kept, blocked = simulate([dict(t) for t in P + Z], same_bar)
    kp = [t for t in kept if t["s"] == "P"]
    kz = [t for t in kept if t["s"] == "Z"]
    bp = [t for t in blocked if t["s"] == "P"]
    bz = [t for t in blocked if t["s"] == "Z"]
    solo_p, comb = stats(P, w), stats(kept, w)
    print("== %s" % title)
    print("  pullback соло      " + fmt(solo_p))
    print("  zone соло          " + fmt(stats(Z, w)))
    print("  совместный счёт    " + fmt(comb))
    print("    из них pullback  " + fmt(stats(kp, w)))
    print("    из них zone      " + fmt(stats(kz, w)))
    steal = sum(t["pct"] * w["P"] for t in bp)
    lost_z = sum(t["pct"] * w["Z"] for t in bz)
    print("  вытеснено pullback: %d сд (вклад соло %+.2f%%, из них прибыльных %d)"
          % (len(bp), steal, sum(1 for t in bp if t["pct"] > 0)))
    print("  вытеснено zone:     %d сд (вклад соло %+.2f%%)" % (len(bz), lost_z))
    # кого отдаёт каждая сторона: PF вытесненных против оставшихся, в % позиции
    for name, gone, left in (("pullback", bp, kp), ("zone", bz, kz)):
        if gone:
            print("    %s: PF вытесненных %s (%d сд) против оставшихся %s (%d сд)" % (
                name, pf_str(gone), len(gone), pf_str(left), len(left)))
    p_in = {t["in"] for t in P}
    same = sum(1 for t in Z if t["in"] in p_in)
    inside = sum(1 for z in Z if z["in"] not in p_in and any(p["in"] < z["in"] <= p["out"] for p in P))
    print("  общий сигнал: zone входит на том же баре, что pullback, — %d из %d входов zone;"
          " ещё %d входов zone внутри позиции pullback" % (same, len(Z), inside))
    print("  прирост к pullback соло: net %+.2f%%  DD %+.2f%%"
          % (comb["net"] - solo_p["net"], comb["dd"] - solo_p["dd"]))
    if bp:
        print("  вытесненные сделки pullback:")
        for t in sorted(bp, key=lambda t: t["in"]):
            print("    %s → %s  %s  %+.2f%% позиции" % (
                t["in"].strftime("%Y-%m-%d %H:%M"), t["out"].strftime("%Y-%m-%d %H:%M"), t["reason"], t["pct"]))
    years = defaultdict(lambda: [0.0, 0.0])
    for t in P:
        years[t["out"].year][0] += t["pct"] * w["P"]
    for t in kept:
        years[t["out"].year][1] += t["pct"] * w[t["s"]]
    print("  по годам выхода (pullback соло → совместный): " + "  ".join(
        "%d %+.2f→%+.2f" % (y, a, b) for y, (a, b) in sorted(years.items())))
    print()


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--pullback", required=True)
    ap.add_argument("--zone", required=True)
    ap.add_argument("--pullback-pct", type=float, default=50.0)
    ap.add_argument("--zone-pct", type=float, default=5.0)
    ap.add_argument("--tail", action="append", default=[], help="YYYY-MM-DD, срез по дате входа; можно несколько")
    ap.add_argument("--same-bar-reentry", action="store_true",
                    help="вход на баре выхода другой стратегии считается разрешённым")
    ap.add_argument("--from", dest="date_from", help="YYYY-MM-DD: брать сделки со входом не раньше")
    ap.add_argument("--to", dest="date_to", help="YYYY-MM-DD: брать сделки со входом раньше этой даты")
    a = ap.parse_args()

    w = {"P": a.pullback_pct / 100.0, "Z": a.zone_pct / 100.0}
    P, Z = load(a.pullback, "P"), load(a.zone, "Z")
    # общее окно для журналов разных прогонов (разные дни, разная длина)
    lo = datetime.fromisoformat(a.date_from + "T00:00:00+00:00") if a.date_from else None
    hi = datetime.fromisoformat(a.date_to + "T00:00:00+00:00") if a.date_to else None
    def within(t):
        return (lo is None or t["in"] >= lo) and (hi is None or t["in"] < hi)
    P, Z = [t for t in P if within(t)], [t for t in Z if within(t)]
    if P and Z:
        print("журналы: pullback %s … %s, zone %s … %s (по входам)" % (
            min(t["in"] for t in P).date(), max(t["in"] for t in P).date(),
            min(t["in"] for t in Z).date(), max(t["in"] for t in Z).date()))
    print("доли счёта: pullback %.0f%%, zone %.0f%%; повторный вход на баре выхода: %s\n"
          % (a.pullback_pct, a.zone_pct, "да" if a.same_bar_reentry else "нет"))
    block("полное окно", P, Z, w, a.same_bar_reentry)
    for d in a.tail:
        cut = datetime.fromisoformat(d + "T00:00:00+00:00")
        block("вход с %s" % d, [t for t in P if t["in"] >= cut], [t for t in Z if t["in"] >= cut], w, a.same_bar_reentry)

    # чувствительность к доле zone: при какой доле совместный счёт перестаёт выигрывать
    print("== чувствительность к доле zone (полное окно), прирост net к pullback соло:")
    solo = stats(P, w)["net"]
    row = []
    for zp in (2, 5, 10, 20, 30, 50):
        ww = {"P": w["P"], "Z": zp / 100.0}
        kept, _ = simulate([dict(t) for t in P + Z], a.same_bar_reentry)
        row.append("%d%% %+.2f" % (zp, stats(kept, ww)["net"] - solo))
    print("  " + "  ".join(row))


if __name__ == "__main__":
    main()
