"""Read / merge / write DAILY_LOG.json — the `nutrition` object per day.

Lose It! is the source of truth for nutrition, so a sync overwrites the day's
`nutrition` with fresh export numbers (idempotent). Everything else on the day
— `training`, `body`, `watch`, `final_note` — is left untouched.
"""
import json
from datetime import date

WEEKDAYS = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"]


def load(path):
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)


def save(path, doc):
    with open(path, "w", encoding="utf-8") as f:
        json.dump(doc, f, ensure_ascii=False, indent=2)
        f.write("\n")


def find_day(doc, day_iso):
    for d in doc.get("days", []):
        if d.get("date") == day_iso:
            return d
    return None


def upsert_nutrition(doc, day_iso, nutrition):
    """Set day[day_iso].nutrition, creating the day in date order if needed.

    Returns (day_obj, status): "created" (new day) | "updated" (replaced
    existing nutrition) | "added" (day existed, had no nutrition yet).
    """
    day = find_day(doc, day_iso)
    if day is not None:
        status = "updated" if day.get("nutrition") else "added"
        day["nutrition"] = nutrition
        return day, status

    wd = WEEKDAYS[date.fromisoformat(day_iso).weekday()]
    day = {
        "date": day_iso,
        "weekday": wd,
        "partial": True,
        "nutrition": nutrition,
        "body": {"weight_lb": None, "waist_in": None},
        "watch": {"sleep_hrs": None, "resting_hr": None, "steps": None},
    }
    days = doc.setdefault("days", [])
    for i, d in enumerate(days):
        if d.get("date", "") > day_iso:
            days.insert(i, day)
            break
    else:
        days.append(day)
    return day, "created"
