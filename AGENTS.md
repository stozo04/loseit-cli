# AGENTS.md — machine contract for loseit-cli

loseit-cli is a read-only Lose It! nutrition extractor. It obtains the data export (a downloaded ZIP or a
cookie fetch), parses the food log, and emits per-day nutrition as JSON. It does **no** writing — no daily
log, no sync, no upsert; the consuming agent does the storing. **stdout is data; stderr is hints/logs/errors.**

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Success |
| `2` | Export / parse failure (no token, expired cookie, missing/invalid ZIP, unreadable export, empty `food-logs.csv`) |
| `64` | Usage error (bad flags, bad `--date`) |
| `78` | Config error (unreadable / invalid `config.json`) |
| `1` | Other failure |

## `days` (the core emit)

Parses the export and prints per-day nutrition. A human table by default; `--json` emits the frozen
contract below.

```sh
loseit-cli days --zip export.zip --json --days 7
loseit-cli days --zip export.zip --date 2026-06-16 --days 1
loseit-cli days --json                 # cookie-fetch path (needs a token)
```

| Flag | Meaning | Default |
|---|---|---|
| `--zip PATH` | parse a downloaded export ZIP instead of fetching | — (cookie fetch) |
| `--date` | civil anchor: `today` \| `yesterday` \| `YYYY-MM-DD` | `today` |
| `--days N` | days back from `--date` (inclusive window) | `7` |
| `--json` | emit the frozen per-day JSON contract | off (human table) |

### `days --json` shape (FROZEN)

A JSON **object keyed by ISO date** → nutrition object. Keys are ascending by date. Key order within each
nutrition object is frozen. An empty selection emits `{}`.

```json
{
  "2026-06-16": {
    "source": "Lose It export",
    "calories_food": 505,
    "protein_g": 75,
    "carbs_g": 36,
    "fat_g": 6,
    "fiber_g": 3,
    "meals": [
      {
        "meal": "Breakfast",
        "calories": 225,
        "protein_g": 23,
        "carbs_g": 36,
        "fat_g": 0,
        "items": [
          { "name": "Greek Yogurt", "qty": "1 cup", "calories": 120 }
        ]
      }
    ],
    "loseit_budget": 1663,
    "loseit_under": 1158,
    "exercise_adjustment": 120
  }
}
```

- `loseit_budget`, `loseit_under` (= `loseit_budget − calories_food`), and `exercise_adjustment` appear
  **only** when `daily-calorie-summary.csv` provides them, and **after** `meals`. `loseit_budget`/`loseit_under`
  are emitted together (under is shown even when zero); `exercise_adjustment` is omitted when the day's
  exercise calories are zero.
- All numbers are integers, rounded **half-to-even** (banker's rounding). Day and per-meal totals round the
  summed raw floats; `items[].calories` rounds each row's calorie value.
- Meal order: `Breakfast, Lunch, Dinner, Snacks` first (that order), then any other meals in first-encounter
  order. A blank meal name becomes `Other`. Deleted rows are excluded.

## `config show --json`

The resolved effective config. Key order frozen:

```json
{
  "token_path": "~/.config/loseit/token",
  "export_url": "https://www.loseit.com/export/data",
  "config_path": "<path>"
}
```

`config path` prints just the `config.json` path in use.

## `doctor`

Reports config + whether a token is present. **No network** (Lose It has no OAuth to validate). Always
exits `0` — lacking a token is not a failure because `--zip` needs none; a stderr hint is printed when no
token is found. Never reveals the cookie value. Key order frozen:

```json
{
  "tokenPresent": false,
  "exportURL": "https://www.loseit.com/export/data",
  "tokenPath": "~/.config/loseit/token",
  "configPath": "<path>",
  "version": "<version>"
}
```

## `version --json`

```json
{ "version": "<version>", "commit": "<sha>", "date": "<iso>", "go": "go1.x" }
```

## Auth model

- `--zip PATH` reads a downloaded export — **no token**. This is the reliable headless/agent path.
- Cookie fetch (`days` without `--zip`) sends `Cookie: liauth=<t>; fn_auth=<t>`; the token comes from
  `LOSEIT_TOKEN` or the `token_path` file. **The `liauth` cookie expires with no auto-refresh**, so headless
  cookie fetches eventually fail (exit `2`, "token probably expired") — fall back to `--zip`.
