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

С --md PATH пишет ещё и отчёт владельцу в Markdown: сколько денег (₽ на счёт --capital и % счёта)
принесла каждая стратегия соло и вместе, и какими сделками совместный счёт отличается от соло —
вытесненные с каждой стороны, общие сигналы, сделки zone, которые добавляются к pullback.

  python3 overlap.py --pullback P_trades.csv --zone Z_trades.csv \
      [--pullback-pct 50] [--zone-pct 5] [--tail 2026-04-01] [--tail 2025-10-01] [--same-bar-reentry] \
      [--md report.md --ticker DOMRF --capital 100000]
"""

import argparse
import csv
from collections import defaultdict
from datetime import datetime, timedelta, timezone

MSK = timezone(timedelta(hours=3))


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
                "bars": int(float(r.get("bars_held") or 0)),
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


def block(title, P, Z, w, same_bar, sections=None):
    kept, blocked = simulate([dict(t) for t in P + Z], same_bar)
    kp = [t for t in kept if t["s"] == "P"]
    kz = [t for t in kept if t["s"] == "Z"]
    bp = [t for t in blocked if t["s"] == "P"]
    bz = [t for t in blocked if t["s"] == "Z"]
    solo_p, comb = stats(P, w), stats(kept, w)
    if sections is not None:
        sections.append({"title": title, "P": P, "Z": Z, "kept": kept, "kp": kp, "kz": kz, "bp": bp, "bz": bz})
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


def rub(pct, capital):
    return "%+.0f ₽" % (pct / 100.0 * capital)


def metrics(trades, w, cap):
    """Полный набор метрик набора сделок: деньги — вклад в счёт (доля стратегии × % позиции)."""
    vals = [t["pct"] * w[t["s"]] / 100.0 * cap for t in sorted(trades, key=lambda t: t["out"])]
    st = stats(trades, w)
    wins = [v for v in vals if v > 0]
    losses = [v for v in vals if v <= 0]
    streak = run = 0
    for v in vals:
        run = run + 1 if v <= 0 else 0
        streak = max(streak, run)
    reasons = defaultdict(int)
    for t in trades:
        reasons[t["reason"]] += 1
    bars = sorted(t["bars"] for t in trades)
    n = len(vals)
    return {
        "n": n, "wins": len(wins), "losses": len(losses),
        "wr": len(wins) / n * 100 if n else 0.0,
        "net": sum(vals), "netpct": st["net"],
        "gp": sum(wins), "gl": -sum(losses),
        "pf": st["pf"] if n else None,
        "avg": sum(vals) / n if n else 0.0,
        "avgw": sum(wins) / len(wins) if wins else 0.0,
        "avgl": sum(losses) / len(losses) if losses else 0.0,
        "best": max(vals) if vals else 0.0, "worst": min(vals) if vals else 0.0,
        "avgpos": sum(t["pct"] for t in trades) / n if n else 0.0,
        "dd": st["dd"] / 100.0 * cap, "ddpct": st["dd"],
        "ratio": (st["net"] / st["dd"]) if st["dd"] > 0 else None,
        "streak": streak,
        "hold": bars[n // 2] if n else 0, "holdmax": bars[-1] if n else 0,
        "reasons": ", ".join("%s %d" % (k, v) for k, v in sorted(reasons.items(), key=lambda kv: -kv[1])) or "—",
    }


def md_metrics_table(cols, w, cap):
    """cols: [(заголовок, сделки)] → транспонированная таблица метрик."""
    ms = [metrics(tr, w, cap) for _, tr in cols]
    def f(x, spec="%+.0f ₽"):
        return spec % x
    rows = [
        ("Сделок", lambda m: "%d" % m["n"]),
        ("Прибыльных / убыточных", lambda m: "%d / %d" % (m["wins"], m["losses"])),
        ("Win rate", lambda m: "%.1f%%" % m["wr"]),
        ("**Деньги (net)**", lambda m: "**%s**" % f(m["net"])),
        ("% счёта", lambda m: "%+.2f%%" % m["netpct"]),
        ("Profit factor", lambda m: "—" if m["pf"] is None else ("inf" if m["pf"] == float("inf") else "%.3f" % m["pf"])),
        ("Валовая прибыль / убыток", lambda m: "%s / %s" % (f(m["gp"], "%.0f ₽"), f(m["gl"], "%.0f ₽"))),
        ("Средняя сделка", lambda m: f(m["avg"])),
        ("Средняя прибыльная / убыточная", lambda m: "%s / %s" % (f(m["avgw"]), f(m["avgl"]))),
        ("Лучшая / худшая сделка", lambda m: "%s / %s" % (f(m["best"]), f(m["worst"]))),
        ("Средний результат, % позиции", lambda m: "%+.2f%%" % m["avgpos"]),
        ("Max просадка", lambda m: "%s (%.2f%%)" % (f(m["dd"], "%.0f ₽"), m["ddpct"])),
        ("Net / просадка", lambda m: "—" if m["ratio"] is None else "%.2f" % m["ratio"]),
        ("Убыточных подряд, макс.", lambda m: "%d" % m["streak"]),
        ("Удержание, баров 30m: медиана / макс.", lambda m: "%d / %d" % (m["hold"], m["holdmax"])),
        ("Выходы", lambda m: m["reasons"]),
    ]
    out = ["| Метрика | " + " | ".join(c for c, _ in cols) + " |", "|---" * (len(cols) + 1) + "|"]
    for name, fn in rows:
        out.append("| %s | %s |" % (name, " | ".join(fn(m) for m in ms)))
    return out


def md_stats_row(name, trades, w, capital):
    s = stats(trades, w)
    pf = "—" if not trades else ("inf" if s["pf"] == float("inf") else "%.3f" % s["pf"])
    return "| %s | %d | %s | %+.2f%% | %s | %.2f%% |" % (
        name, s["n"], rub(s["net"], capital), s["net"], pf, s["dd"])


def md_trades(trades, w, capital, note=None):
    if not trades:
        return ["нет", ""]
    out = ["| Стратегия | Вход | Выход | Причина | % позиции | ₽ на счёт |" + (" %s |" % note[0] if note else ""),
           "|---|---|---|---|---|---|" + ("---|" if note else "")]
    for t in sorted(trades, key=lambda t: t["in"]):
        row = "| %s | %s | %s | %s | %+.2f%% | %s |" % (
            "pullback" if t["s"] == "P" else "zone", t["in"].astimezone(MSK).strftime("%Y-%m-%d %H:%M"),
            t["out"].astimezone(MSK).strftime("%Y-%m-%d %H:%M"), t["reason"], t["pct"], rub(t["pct"] * w[t["s"]], capital))
        if note:
            row += " %s |" % note[1](t)
        out.append(row)
    total = sum(t["pct"] * w[t["s"]] for t in trades)
    out += ["", "Итого: %d сд, %s (%+.2f%% счёта)." % (len(trades), rub(total, capital), total), ""]
    return out


def write_md(path, a, w, sections, sens):
    cap = a.capital
    L = ["# rsi_zone рядом с rsi_pullback: %s" % (a.ticker or "тикер"), "",
         "Журналы: `%s` (pullback), `%s` (zone)." % (a.pullback, a.zone),
         "Доли счёта: pullback %.0f%%, zone %.0f%%; счёт для пересчёта в рубли — %s ₽; "
         "деньги — вклад в счёт без реинвестирования, по дате выхода; время сделок — МСК."
         % (a.pullback_pct, a.zone_pct, format(int(cap), ",").replace(",", " ")), "",
         "## Вердикт", "", "_заполняется по правилам скилла_", ""]
    f0 = sections[0]
    mp, mz, mc = (metrics(f0["P"], w, cap), metrics(f0["Z"], w, cap), metrics(f0["kept"], w, cap))
    def pfs(m):
        return "—" if m["pf"] is None else ("inf" if m["pf"] == float("inf") else "%.2f" % m["pf"])
    L += ["## Коротко (%s)" % f0["title"], "",
          "| | Деньги | Сделок | Win rate | PF | Max просадка |", "|---|---|---|---|---|---|"]
    for name, m in (("pullback соло", mp), ("zone соло", mz), ("**вместе**", mc)):
        L.append("| %s | %+.0f ₽ (%+.2f%%) | %d | %.1f%% | %s | %.0f ₽ (%.2f%%) |"
                 % (name, m["net"], m["netpct"], m["n"], m["wr"], pfs(m), m["dd"], m["ddpct"]))
    L += ["", "- Zone к pullback на общем счёте: **%+.0f ₽** (%+.2f%% счёта), просадка %+.0f ₽."
          % (mc["net"] - mp["net"], mc["netpct"] - mp["netpct"], mc["dd"] - mp["dd"]),
          "- Вытеснено: pullback — %d сд (%+.0f ₽), zone — %d сд (%+.0f ₽); добавлено сделок zone: %d (%+.0f ₽)."
          % (len(f0["bp"]), metrics(f0["bp"], w, cap)["net"], len(f0["bz"]), metrics(f0["bz"], w, cap)["net"],
             len(f0["kz"]), metrics(f0["kz"], w, cap)["net"]), ""]
    L += ["## Деньги: каждая стратегия отдельно и вместе", ""]
    for sec in sections:
        P, Z, kept = sec["P"], sec["Z"], sec["kept"]
        sp, sc = stats(P, w), stats(kept, w)
        L += ["### %s" % sec["title"], ""]
        L += md_metrics_table([("pullback соло", P), ("zone соло", Z), ("**вместе**", kept),
                               ("вместе: pullback", sec["kp"]), ("вместе: zone", sec["kz"])], w, cap)
        L += ["",
              "Разница «вместе − pullback соло»: **%s** (%+.2f%% счёта), DD %+.2f п.п. "
              "Сумма соло обеих стратегий — %s; совместный счёт отстаёт от неё на %.0f ₽ из-за вытесненных сделок."
              % (rub(sc["net"] - sp["net"], cap), sc["net"] - sp["net"], sc["dd"] - sp["dd"],
                 rub(sp["net"] + stats(Z, w)["net"], cap),
                 (sp["net"] + stats(Z, w)["net"] - sc["net"]) / 100.0 * cap), ""]
    full = sections[0]
    P, Z, bp, bz, kz = full["P"], full["Z"], full["bp"], full["bz"], full["kz"]
    p_in = {t["in"] for t in P}
    def overlap_kind(z):
        if z["in"] in p_in:
            return "тот же бар"
        if any(p["in"] < z["in"] <= p["out"] for p in P):
            return "внутри позиции pullback"
        return "—"
    L += ["## Чем отличаются сделки (%s)" % full["title"], ""]
    L += md_metrics_table([("pullback: вытеснены zone", bp), ("pullback: остались", full["kp"]),
                           ("zone: вытеснены pullback", bz), ("zone: добавлены", kz)], w, cap)
    L += ["", "Если у вытесненных сделок PF и win rate выше, чем у оставшихся, вытесняются лучшие сделки.", "",
          "### Сделки pullback, которые вытеснила zone", "",
          "Pullback соло их делает, вместе — нет: тикер в этот момент занят позицией zone.", ""]
    L += md_trades(bp, w, cap)
    L += ["### Сделки zone, которые вытеснил pullback", "",
          "Zone соло их делает, вместе — нет: тикер занят pullback или pullback вошёл на том же баре.", ""]
    L += md_trades(bz, w, cap, ("Пересечение", overlap_kind))
    L += ["### Сделки zone, которые добавляются к pullback", "",
          "Их нет у pullback соло — это и есть добавка zone к счёту.", ""]
    L += md_trades(kz, w, cap)
    common = sum(1 for z in Z if overlap_kind(z) != "—")
    L += ["### Общий сигнал", "",
          "Из %d входов zone %d приходятся на бар входа pullback или внутрь его позиции; "
          "сделок pullback, не затронутых zone: %d из %d." % (len(Z), common, len(P) - len(bp), len(P)), ""]
    def period_table(title, key):
        groups = defaultdict(lambda: ([], [], []))
        for i, src in enumerate((P, Z, full["kept"])):
            for t in src:
                groups[key(t["out"].astimezone(MSK))][i].append(t)
        out = ["## %s (%s)" % (title, full["title"]), "",
               "| Период | pullback соло | zone соло | вместе | разница к pullback | сделок P / Z / вместе | PF вместе |",
               "|---|---|---|---|---|---|---|"]
        for k, (gp_, gz_, gc_) in sorted(groups.items()):
            mp_, mz_, mc_ = metrics(gp_, w, cap), metrics(gz_, w, cap), metrics(gc_, w, cap)
            out.append("| %s | %+.0f ₽ | %+.0f ₽ | %+.0f ₽ | %+.0f ₽ | %d / %d / %d | %s |" % (
                k, mp_["net"], mz_["net"], mc_["net"], mc_["net"] - mp_["net"], mp_["n"], mz_["n"], mc_["n"], pfs(mc_)))
        return out + [""]
    L += period_table("По годам выхода", lambda d: d.strftime("%Y"))
    L += period_table("По месяцам выхода", lambda d: d.strftime("%Y-%m"))
    L += ["## Чувствительность к доле zone (%s)" % full["title"], "",
          "| Доля zone | Разница к pullback соло |", "|---|---|"]
    for zp, d in sens:
        L.append("| %d%% | %s (%+.2f%%) |" % (zp, rub(d, cap), d))
    L += ["", "## Ограничение модели", "",
          "Соло-журнал не знает сигналов, пропущенных стратегией, пока она сидела в своей позиции: "
          "вытесненная сделка выпадает целиком, хотя вживую вход мог случиться позже. "
          "«Вытеснено» — оценка сверху, совместный счёт — оценка снизу.", ""]
    with open(path, "w", encoding="utf-8") as f:
        f.write("\n".join(L))


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
    ap.add_argument("--md", help="путь Markdown-отчёта владельцу")
    ap.add_argument("--ticker", default="", help="тикер для заголовка отчёта")
    ap.add_argument("--capital", type=float, default=100000.0, help="размер счёта в ₽ для пересчёта процентов в деньги")
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
    sections = []
    block("полное окно", P, Z, w, a.same_bar_reentry, sections)
    for d in a.tail:
        cut = datetime.fromisoformat(d + "T00:00:00+00:00")
        block("вход с %s" % d, [t for t in P if t["in"] >= cut], [t for t in Z if t["in"] >= cut], w,
              a.same_bar_reentry, sections)

    # чувствительность к доле zone: при какой доле совместный счёт перестаёт выигрывать
    print("== чувствительность к доле zone (полное окно), прирост net к pullback соло:")
    solo = stats(P, w)["net"]
    row, sens = [], []
    for zp in (2, 5, 10, 20, 30, 50):
        ww = {"P": w["P"], "Z": zp / 100.0}
        kept, _ = simulate([dict(t) for t in P + Z], a.same_bar_reentry)
        d = stats(kept, ww)["net"] - solo
        sens.append((zp, d))
        row.append("%d%% %+.2f" % (zp, d))
    print("  " + "  ".join(row))
    if a.md:
        write_md(a.md, a, w, sections, sens)
        print("\nотчёт: %s" % a.md)


if __name__ == "__main__":
    main()
