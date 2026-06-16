"""Offline tests — no network. Builds a synthetic export ZIP shaped like the
real Lose It! export (confirmed columns) and exercises parse -> aggregate ->
upsert."""
import io
import zipfile
from loseit import export as exp, nutrition as nut, daily_log as logmod

FOOD_HEADER = ("Date,Name,Icon,Meal,Quantity,Units,Calories,Deleted,"
               "Fat (g),Protein (g),Carbohydrates (g),Saturated Fat (g),"
               "Sugars (g),Fiber (g),Cholesterol (mg),Sodium (mg)")
FOOD_ROWS = [
    "2026-06-16,Greek Yogurt,icon,Breakfast,1,cup,120,false,0,22,9,0,9,0,10,80",
    "2026-06-16,Banana,icon,Breakfast,1,each,105,false,0,1,27,0,14,3,0,1",
    "2026-06-16,Chicken Breast,icon,Lunch,6,oz,280,false,6,52,0,2,0,0,120,150",
    "2026-06-16,Old Deleted Food,icon,Lunch,1,each,999,true,50,0,50,0,0,0,0,0",
    "2026-06-15,Protein Shake,icon,Snacks,1,scoop,120,false,1,24,3,0,2,1,5,90",
]
SUMMARY = ("Date,Food cals,Exercise cals,Budget cals,EER\n"
           "2026-06-16,505,120,1663,2450\n2026-06-15,120,0,1663,2450\n")


def _zip():
    buf = io.BytesIO()
    with zipfile.ZipFile(buf, "w") as z:
        z.writestr("food-logs.csv", FOOD_HEADER + "\n" + "\n".join(FOOD_ROWS) + "\n")
        z.writestr("daily-calorie-summary.csv", SUMMARY)
    return buf.getvalue()


def _parse():
    data = _zip()
    food = exp.read_csv(data, "food-logs.csv")
    summary = exp.read_csv(data, "daily-calorie-summary.csv")
    return nut.build_nutrition_by_day(food, summary)


def test_aggregation_sums_macros_and_skips_deleted():
    days = _parse()
    n = days["2026-06-16"]
    # 120+105+280 = 505 (the 999 deleted row excluded)
    assert n["calories_food"] == 505
    assert n["protein_g"] == 75      # 22+1+52
    assert n["fiber_g"] == 3         # 0+3+0
    assert n["carbs_g"] == 36        # 9+27+0
    assert n["source"] == "Lose It export"


def test_meals_grouped_and_ordered():
    n = _parse()["2026-06-16"]
    meals = [m["meal"] for m in n["meals"]]
    assert meals == ["Breakfast", "Lunch"]          # Breakfast before Lunch
    breakfast = n["meals"][0]
    assert breakfast["calories"] == 225             # 120+105
    assert {i["name"] for i in breakfast["items"]} == {"Greek Yogurt", "Banana"}


def test_summary_fields_attached():
    n = _parse()["2026-06-16"]
    assert n["loseit_budget"] == 1663
    assert n["loseit_under"] == 1663 - 505
    assert n["exercise_adjustment"] == 120


def test_upsert_preserves_other_day_data():
    days = _parse()
    doc = {"days": [{"date": "2026-06-16", "weekday": "Tue",
                     "training": {"session": "Push", "source": "manual"},
                     "final_note": "felt good"}]}
    _, status = logmod.upsert_nutrition(doc, "2026-06-16", days["2026-06-16"])
    assert status == "added"
    day = logmod.find_day(doc, "2026-06-16")
    assert day["nutrition"]["protein_g"] == 75
    assert day["training"]["session"] == "Push"     # untouched
    assert day["final_note"] == "felt good"          # untouched


def test_upsert_idempotent_and_creates_in_order():
    days = _parse()
    doc = {"days": [{"date": "2026-06-14"}, {"date": "2026-06-18"}]}
    _, s1 = logmod.upsert_nutrition(doc, "2026-06-16", days["2026-06-16"])
    _, s2 = logmod.upsert_nutrition(doc, "2026-06-16", days["2026-06-16"])
    assert s1 == "created" and s2 == "updated"
    assert [d["date"] for d in doc["days"]] == ["2026-06-14", "2026-06-16", "2026-06-18"]


if __name__ == "__main__":
    fns = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    for fn in fns:
        fn()
        print(f"  ok  {fn.__name__}")
    print(f"\n{len(fns)} passed.")
