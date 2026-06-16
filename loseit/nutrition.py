"""Aggregate Lose It! food-log rows into per-day nutrition objects.

Output matches the DAILY_LOG.json `nutrition` shape already in use:
  { source, calories_food, protein_g, carbs_g, fat_g, fiber_g,
    loseit_budget, exercise_adjustment, loseit_under, meals: [...] }
plus fiber_g (new; Steven tracks a 30g fiber target).
"""
from datetime import datetime

MEAL_ORDER = ["Breakfast", "Lunch", "Dinner", "Snacks"]


def _num(v):
    try:
        return float(v)
    except (TypeError, ValueError):
        return 0.0


def _norm_date(s):
    s = (s or "").strip()
    for fmt in ("%Y-%m-%d", "%m/%d/%Y", "%m/%d/%y", "%Y/%m/%d"):
        try:
            return datetime.strptime(s, fmt).date().isoformat()
        except ValueError:
            continue
    return None


def _is_deleted(row):
    return str(row.get("Deleted", "")).strip().lower() in ("true", "1", "yes")


def build_nutrition_by_day(food_rows, summary_rows):
    """Return {date_iso: nutrition_dict} aggregated from the export rows."""
    days = {}
    for r in food_rows:
        if _is_deleted(r):
            continue
        d = _norm_date(r.get("Date"))
        if not d:
            continue
        cals, pro = _num(r.get("Calories")), _num(r.get("Protein (g)"))
        carb, fat = _num(r.get("Carbohydrates (g)")), _num(r.get("Fat (g)"))
        fib = _num(r.get("Fiber (g)"))

        day = days.setdefault(d, {"calories_food": 0.0, "protein_g": 0.0,
                                  "carbs_g": 0.0, "fat_g": 0.0, "fiber_g": 0.0,
                                  "meals": {}})
        day["calories_food"] += cals
        day["protein_g"] += pro
        day["carbs_g"] += carb
        day["fat_g"] += fat
        day["fiber_g"] += fib

        meal_name = (r.get("Meal") or "Other").strip() or "Other"
        meal = day["meals"].setdefault(meal_name, {
            "meal": meal_name, "calories": 0.0, "protein_g": 0.0,
            "carbs_g": 0.0, "fat_g": 0.0, "items": []})
        meal["calories"] += cals
        meal["protein_g"] += pro
        meal["carbs_g"] += carb
        meal["fat_g"] += fat
        qty = " ".join(p for p in (str(r.get("Quantity", "")).strip(),
                                   str(r.get("Units", "")).strip()) if p)
        meal["items"].append({"name": (r.get("Name") or "").strip(),
                              "qty": qty, "calories": round(cals)})

    summary = {}
    for r in summary_rows:
        d = _norm_date(r.get("Date"))
        if d:
            summary[d] = r

    out = {}
    for d, day in days.items():
        ordered = [m for m in MEAL_ORDER if m in day["meals"]]
        ordered += [m for m in day["meals"] if m not in MEAL_ORDER]
        meals = [{
            "meal": day["meals"][m]["meal"],
            "calories": round(day["meals"][m]["calories"]),
            "protein_g": round(day["meals"][m]["protein_g"]),
            "carbs_g": round(day["meals"][m]["carbs_g"]),
            "fat_g": round(day["meals"][m]["fat_g"]),
            "items": day["meals"][m]["items"],
        } for m in ordered]

        nut = {
            "source": "Lose It export",
            "calories_food": round(day["calories_food"]),
            "protein_g": round(day["protein_g"]),
            "carbs_g": round(day["carbs_g"]),
            "fat_g": round(day["fat_g"]),
            "fiber_g": round(day["fiber_g"]),
            "meals": meals,
        }
        s = summary.get(d)
        if s:
            budget = round(_num(s.get("Budget cals")))
            if budget:
                nut["loseit_budget"] = budget
                nut["loseit_under"] = budget - nut["calories_food"]
            ex = round(_num(s.get("Exercise cals")))
            if ex:
                nut["exercise_adjustment"] = ex
        out[d] = nut
    return out
