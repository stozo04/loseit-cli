"""loseit-cli — pull Lose It! nutrition into DAILY_LOG.json.

Commands:
  days     Show parsed nutrition for recent days (no writing).
  sync     Write recent days' nutrition into DAILY_LOG.json.

Source of the export ZIP: --zip PATH (a downloaded export) or, by default,
a cookie-authenticated fetch from loseit.com.
"""
import sys
import json
import argparse
from datetime import date, datetime, timedelta

from . import config as cfgmod
from . import export as exp
from . import nutrition as nut
from . import daily_log as logmod


def _err(*a):
    print(*a, file=sys.stderr)


def _resolve_date(s):
    if not s or s == "today":
        return date.today()
    if s == "yesterday":
        return date.today() - timedelta(days=1)
    return datetime.strptime(s, "%Y-%m-%d").date()


def _wanted_dates(target, days):
    return {(target - timedelta(days=i)).isoformat() for i in range(days)}


def _load_nutrition(cfg, zip_path):
    data = exp.load_zip_bytes(cfg, zip_path)
    food = exp.read_csv(data, "food-logs.csv")
    summary = exp.read_csv(data, "daily-calorie-summary.csv")
    if not food:
        raise exp.LoseItError("food-logs.csv not found / empty in the export.")
    return nut.build_nutrition_by_day(food, summary)


def cmd_days(args):
    cfg = cfgmod.load_config()
    by_day = _load_nutrition(cfg, args.zip)
    target = _resolve_date(args.date)
    wanted = sorted(_wanted_dates(target, args.days))
    rows = [(d, by_day[d]) for d in wanted if d in by_day]
    if args.json:
        print(json.dumps({d: n for d, n in rows}, ensure_ascii=False, indent=2))
        return
    if not rows:
        print(f"No Lose It entries for the last {args.days} day(s) ending {target}.")
        return
    print(f"Lose It nutrition, last {args.days} day(s) ending {target}:\n")
    for d, n in rows:
        print(f"  {d}: {n['calories_food']} cal, "
              f"{n['protein_g']}g protein, {n['fiber_g']}g fiber, "
              f"{n['carbs_g']}g carb, {n['fat_g']}g fat  "
              f"({len(n['meals'])} meals)")


def cmd_sync(args):
    cfg = cfgmod.load_config()
    by_day = _load_nutrition(cfg, args.zip)
    target = _resolve_date(args.date)
    wanted = _wanted_dates(target, args.days)
    todo = sorted(d for d in wanted if d in by_day)

    if not todo:
        print(f"No Lose It entries to log for the last {args.days} day(s) ending {target}.")
        return

    doc = logmod.load(cfg["daily_log"])
    summary = []
    for d in todo:
        if args.dry_run:
            summary.append((d, by_day[d], "dry-run"))
            continue
        _, status = logmod.upsert_nutrition(doc, d, by_day[d])
        summary.append((d, by_day[d], status))

    if not args.dry_run:
        logmod.save(cfg["daily_log"], doc)

    tag = "[dry-run] would write" if args.dry_run else "synced"
    print(f"{tag} nutrition for {len(summary)} day(s):\n")
    FLAG = {"created": "  (new day)", "updated": "  (replaced)",
            "added": "  (added)", "dry-run": ""}
    for d, n, status in summary:
        print(f"  {d}: {n['calories_food']} cal, {n['protein_g']}g protein, "
              f"{n['fiber_g']}g fiber{FLAG.get(status, '')}")


def main(argv=None):
    p = argparse.ArgumentParser(prog="loseit",
                                description="Log Lose It! nutrition into DAILY_LOG.json.")
    sub = p.add_subparsers(dest="cmd", required=True)

    sp = sub.add_parser("days", help="Show parsed nutrition for recent days")
    sp.add_argument("--zip", help="parse a downloaded export ZIP instead of fetching")
    sp.add_argument("--date", default="today", help="today | yesterday | YYYY-MM-DD")
    sp.add_argument("--days", type=int, default=7)
    sp.add_argument("--json", action="store_true")
    sp.set_defaults(func=cmd_days)

    sp = sub.add_parser("sync", help="Write recent days' nutrition into DAILY_LOG.json")
    sp.add_argument("--zip", help="parse a downloaded export ZIP instead of fetching")
    sp.add_argument("--date", default="today", help="today | yesterday | YYYY-MM-DD")
    sp.add_argument("--days", type=int, default=3)
    sp.add_argument("--dry-run", action="store_true")
    sp.set_defaults(func=cmd_sync)

    args = p.parse_args(argv)
    try:
        args.func(args)
    except exp.LoseItError as e:
        _err(f"loseit error: {e}")
        sys.exit(2)


if __name__ == "__main__":
    main()
