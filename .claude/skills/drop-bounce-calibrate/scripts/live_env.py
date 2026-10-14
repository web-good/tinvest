"""Writes a ticker into or out of the drop_bounce live universe.

Usage: live_env.py <TICKER> [Minutes30]   (no TF = remove)

drop_bounce runs on 30m only (owner, 2026-10-14: hourly signal, 30m MACD confirmation): Minutes30
is the only timeframe accepted. The _M15/_H1 keys of the old rules are deleted from the files if found.

Sets membership exactly: with Minutes30 the ticker ends up in DROP_BOUNCE_TICKERS_M30, without it the
ticker is taken out. Edits env/prod.env, env/prod.env.example and env/local.env.example (run from
the repo root); a missing key is appended. Order of the other tickers is kept; a new one goes last.
"""
import os, sys

KEY = "DROP_BOUNCE_TICKERS_M30"
LEGACY = {"DROP_BOUNCE_TICKERS_M15", "DROP_BOUNCE_TICKERS_H1"}
FILES = ["env/prod.env", "env/prod.env.example", "env/local.env.example"]

if len(sys.argv) < 2:
    sys.exit(__doc__)
ticker = sys.argv[1].upper()
bad = [tf for tf in sys.argv[2:] if tf != "Minutes30"]
if bad:
    sys.exit(f"timeframe {bad} not traded: drop_bounce is 30m only, want Minutes30")
want = bool(sys.argv[2:])

for f in FILES:
    if not os.path.exists(f):
        print(f"{f}: нет файла, пропущен")
        continue
    out, seen = [], False
    for line in open(f).read().splitlines():
        key, sep, v = line.partition("=")
        key = key.strip()
        if sep and key in LEGACY:
            continue
        if sep and key == KEY:
            seen = True
            tickers = [t.strip() for t in v.split(",") if t.strip() and t.strip() != ticker]
            if want:
                tickers.append(ticker)
            line = f"{KEY}={','.join(tickers)}"
        out.append(line)
    if not seen:
        out.append(f"{KEY}={ticker if want else ''}")
    open(f, "w").write("\n".join(out) + "\n")
    print(f"{f}: " + "; ".join(l for l in out if l.split("=")[0] == KEY))
