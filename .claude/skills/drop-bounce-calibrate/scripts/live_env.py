"""Writes a ticker's accepted timeframes into the drop_bounce live universes.

Usage: live_env.py <TICKER> [Minutes15]   (no TF = remove everywhere)

drop_bounce is traded on 15m only (owner, 2026-10-13): Minutes15 is the only timeframe accepted;
the _M30/_H1 keys are still rewritten so a ticker left there by the old rules is taken out.

Sets membership exactly: the ticker ends up in DROP_BOUNCE_TICKERS_<tf> of every listed timeframe
and in no other, so a recalibration that rejects a timeframe also takes the ticker out of it.
Edits env/prod.env, env/prod.env.example and env/local.env.example (run from the repo root);
a key missing from a file is appended. Order of the other tickers is kept; a new one goes last.
"""
import os, sys

KEYS = {"Minutes15": "DROP_BOUNCE_TICKERS_M15", "Minutes30": "DROP_BOUNCE_TICKERS_M30", "Hour1": "DROP_BOUNCE_TICKERS_H1"}
FILES = ["env/prod.env", "env/prod.env.example", "env/local.env.example"]

if len(sys.argv) < 2:
    sys.exit(__doc__)
ticker = sys.argv[1].upper()
bad = [tf for tf in sys.argv[2:] if tf != "Minutes15"]
if bad:
    sys.exit(f"timeframe {bad} not traded: drop_bounce is 15m only, want Minutes15")
want = {KEYS[tf] for tf in sys.argv[2:]}

for f in FILES:
    if not os.path.exists(f):
        print(f"{f}: нет файла, пропущен")
        continue
    lines = open(f).read().splitlines()
    seen = set()
    for i, line in enumerate(lines):
        key, sep, v = line.partition("=")
        if not sep or key.strip() not in KEYS.values():
            continue
        key = key.strip()
        seen.add(key)
        tickers = [t.strip() for t in v.split(",") if t.strip() and t.strip() != ticker]
        if key in want:
            tickers.append(ticker)
        lines[i] = f"{key}={','.join(tickers)}"
    for key in KEYS.values():
        if key not in seen:
            lines.append(f"{key}={ticker if key in want else ''}")
    open(f, "w").write("\n".join(lines) + "\n")
    print(f"{f}: " + "; ".join(l for l in lines if l.split("=")[0] in KEYS.values()))
