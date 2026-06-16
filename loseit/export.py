"""Get the Lose It! export ZIP and read its CSVs.

Export ZIP layout (confirmed from Steven's real export):
  food-logs.csv            Date,Name,Icon,Meal,Quantity,Units,Calories,Deleted,
                           Fat (g),Protein (g),Carbohydrates (g),Saturated Fat (g),
                           Sugars (g),Fiber (g),Cholesterol (mg),Sodium (mg)
  daily-calorie-summary.csv  Date,Food cals,Exercise cals,Budget cals,EER
"""
import io
import os
import csv
import zipfile
import urllib.request


class LoseItError(RuntimeError):
    pass


def _read_token(cfg):
    token = os.environ.get("LOSEIT_TOKEN")
    if token:
        return token.strip()
    path = os.path.expanduser(cfg["token_path"])
    if os.path.exists(path):
        with open(path, "r", encoding="utf-8") as f:
            return f.read().strip()
    return None


def fetch_zip(cfg):
    """Download the export ZIP using the liauth session cookie."""
    token = _read_token(cfg)
    if not token:
        raise LoseItError(
            f"No Lose It token. Save the `liauth` cookie to {cfg['token_path']} "
            "or set LOSEIT_TOKEN — or pass --zip PATH to parse a downloaded export."
        )
    req = urllib.request.Request(
        cfg["export_url"],
        headers={"Cookie": f"liauth={token}; fn_auth={token}",
                 "User-Agent": "loseit-cli/0.1"},
    )
    with urllib.request.urlopen(req, timeout=120) as resp:
        data = resp.read()
    if not (len(data) > 1000 and data[:2] == b"PK"):
        raise LoseItError(
            "Export response wasn't a ZIP — the liauth token is probably expired. "
            "Re-grab it from loseit.com, or use --zip PATH with a fresh download."
        )
    return data


def load_zip_bytes(cfg, zip_path=None):
    if zip_path:
        with open(os.path.expanduser(zip_path), "rb") as f:
            data = f.read()
        if data[:2] != b"PK":
            raise LoseItError(f"{zip_path} is not a ZIP file.")
        return data
    return fetch_zip(cfg)


def read_csv(zip_bytes, name):
    """Return a list of dict rows for `name` inside the ZIP (empty if absent)."""
    with zipfile.ZipFile(io.BytesIO(zip_bytes)) as z:
        if name not in z.namelist():
            return []
        with z.open(name) as f:
            text = io.TextIOWrapper(f, encoding="utf-8-sig", newline="")
            return list(csv.DictReader(text))
