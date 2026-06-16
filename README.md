# loseit-cli

The **nutrition** counterpart to [`speediance-cli`](https://github.com/stozo04/speediance-cli) (strength) and [`google-health-cli`](https://github.com/stozo04/google-health-cli) (cardio).

It reads Steven's **Lose It!** data export and writes per-day nutrition — calories, **protein**, carbs, fat, **fiber**, and a per-meal breakdown — into `DAILY_LOG.json`, matching the existing `nutrition` schema.

## How it works

```
Lose It export ZIP  ── from --zip PATH, or GET /export/data with the liauth cookie
        │
        ▼
  food-logs.csv  (Date, Meal, Calories, Protein, Carbs, Fat, Fiber, …)
        │
        ▼
  aggregate per day + per meal  →  nutrition object
        │
        ▼
  upsert into DAILY_LOG.json  (idempotent; preserves training / final_note / body / watch)
```

The export endpoint (`GET https://www.loseit.com/export/data`) and the `liauth`
cookie scheme were learned from the MIT-licensed
[RichClarkeAI/loseit-cli](https://github.com/RichClarkeAI/loseit-cli). This is an
independent implementation — no GWT-RPC, no Playwright, pure stdlib: just
download, unzip, parse.

## Two ways to supply the export

1. **Downloaded ZIP (no token):**
   ```sh
   python -m loseit sync --zip ~/Downloads/loseit-export.zip
   ```
2. **Cookie fetch:** save your `liauth` cookie (loseit.com → F12 → Application →
   Cookies) to `~/.config/loseit/token`, then:
   ```sh
   python -m loseit sync
   ```
   The session cookie expires periodically and has no auto-refresh, so you'll
   re-grab it occasionally — or just use the `--zip` path with a fresh download.

## Usage

```sh
python -m loseit days --zip export.zip --days 7   # preview parsed nutrition
python -m loseit sync --dry-run                   # show what would be written
python -m loseit sync --date yesterday            # write yesterday
python -m loseit sync --days 7                    # backfill a week
```

`sync` is idempotent per day and never clobbers non-nutrition data on a day.

## What it writes

```json
"nutrition": {
  "source": "Lose It export",
  "calories_food": 1540,
  "protein_g": 112,
  "carbs_g": 139,
  "fat_g": 54,
  "fiber_g": 17,
  "loseit_budget": 1663,
  "loseit_under": 123,
  "meals": [
    { "meal": "Breakfast", "calories": 389, "protein_g": 28,
      "carbs_g": 30, "fat_g": 12, "items": [ { "name": "...", "qty": "1 cup", "calories": 120 } ] }
  ]
}
```

## Notes

- Read-only use of your own Lose It data. The `liauth` token (if used) sits at
  `~/.config/loseit/token` — gitignored, don't commit it.
- Unofficial path; a Lose It web change could break it (same risk class as the
  Speediance unofficial API). The `/export/data` endpoint is first-party, so the
  read path is fairly stable.
