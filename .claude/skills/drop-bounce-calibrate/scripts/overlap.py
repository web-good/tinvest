"""Overlap of drop_bounce trades between two timeframes for one ticker.

Usage: overlap.py <dir with both *_trades.csv> <TICKER> [TF_A TF_B]   (default Minutes15 Minutes30; any pair of Minutes15/Minutes30/Hour1)

A trade overlaps when its holding interval [entry, exit] intersects any trade of the other
timeframe. Also simulates one position per ticker: trades in time order, a trade is skipped
while another timeframe's trade on the same ticker is open (first come first served).
"""
import csv, glob, sys
from datetime import datetime

d, ticker = sys.argv[1], sys.argv[2].upper()
tfa, tfb = sys.argv[3:5] if len(sys.argv) > 4 else ("Minutes15", "Minutes30")

def load(tk, tf):
    f = sorted(glob.glob(f"{d}/{tk}_drop_bounce_{tf}_*_trades.csv"))[-1]
    out = []
    for r in csv.DictReader(open(f)):
        t = lambda s: datetime.fromisoformat(s.replace("Z", "+00:00"))
        out.append((t(r["entry_time"]), t(r["exit_time"]), float(r["pnl_pct"]), tf))
    return out

def pf(xs):
    g = sum(x for x in xs if x > 0); l = -sum(x for x in xs if x < 0)
    return g / l if l else float("inf")

for tk in [ticker]:
    a, b = load(tk, tfa), load(tk, tfb)
    ov = lambda x, ys: any(x[0] <= y[1] and y[0] <= x[1] for y in ys)
    oa = [x for x in a if ov(x, b)]
    ob = [x for x in b if ov(x, a)]
    allt = sorted(a + b, key=lambda x: x[0])
    taken, busy = [], None
    for x in allt:
        if busy and x[0] <= busy:
            continue
        taken.append(x); busy = x[1]
    s = lambda xs: sum(x[2] for x in xs) * 100
    print(f"{tk}: {tfa} {len(a)} сд (PF {pf([x[2] for x in a]):.2f}, сумма {s(a):.1f}%), "
          f"{tfb} {len(b)} сд (PF {pf([x[2] for x in b]):.2f}, сумма {s(b):.1f}%); "
          f"пересекаются {tfa} {len(oa)}/{len(a)}, {tfb} {len(ob)}/{len(b)}; "
          f"одна позиция: {len(taken)} сд, PF {pf([x[2] for x in taken]):.2f}, сумма {s(taken):.1f}% "
          f"(против {s(a)+s(b):.1f}% по отдельности)")
