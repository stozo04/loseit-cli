"""loseit-cli — log Lose It! nutrition into DAILY_LOG.json.

The nutrition counterpart to speediance-cli (strength) and google-health-cli (cardio).
It reads Steven's Lose It! data export — a ZIP of CSVs from the first-party
endpoint `GET https://www.loseit.com/export/data` — and writes per-day nutrition
(calories, protein, carbs, fat, fiber + per-meal breakdown) into DAILY_LOG.json.

Two ways to get the export ZIP:
  * --zip PATH   : parse a ZIP you already downloaded (no token needed).
  * cookie fetch : download it using your `liauth` session cookie.

The endpoint + cookie scheme were learned from the MIT-licensed
RichClarkeAI/loseit-cli; this implementation is independent (no GWT-RPC,
no Playwright — just download, unzip, parse).
"""

__version__ = "0.1.0"
