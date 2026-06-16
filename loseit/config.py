"""Config loading: JSON file + environment-variable overrides.

  * daily_log  - path to the Workout DAILY_LOG.json we write nutrition into.
  * token_path - file holding the Lose It! `liauth` session cookie (only used
                 for the cookie-fetch path; the --zip path needs no token).
  * export_url - Lose It's first-party data-export endpoint.

The local date is taken straight from the export's `Date` column, so no
timezone handling is needed.
"""
import os
import json

# Project root = the loseit-cli/ folder that contains this package.
_PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))


def _default_config_path():
    if os.path.exists("config.json"):
        return "config.json"
    return os.path.join(_PROJECT_ROOT, "config.json")


CONFIG_PATH = os.environ.get("LOSEIT_CONFIG", _default_config_path())

DEFAULTS = {
    "daily_log": "",
    "token_path": "~/.config/loseit/token",
    "export_url": "https://www.loseit.com/export/data",
}


def load_config():
    cfg = dict(DEFAULTS)
    if os.path.exists(CONFIG_PATH):
        with open(CONFIG_PATH, "r", encoding="utf-8") as f:
            cfg.update(json.load(f))
    cfg["daily_log"] = os.environ.get("LOSEIT_DAILY_LOG", cfg["daily_log"])
    if not cfg["daily_log"]:
        raise SystemExit(
            "Missing daily_log path. Set it in config.json or via LOSEIT_DAILY_LOG. "
            "It should point at the Workout project's DAILY_LOG.json."
        )
    return cfg
