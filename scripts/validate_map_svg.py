#!/usr/bin/env python3
"""Validate map SVG province ids against pkg/shared/maps YAML."""
from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
YAML_DIR = ROOT / "pkg" / "shared" / "maps"
SVG_DIR = ROOT / "web" / "public" / "maps"

PROV_RE = re.compile(r"^\s+-\s+\{\s*id:\s*([\w-]+)")


def province_ids(yaml_path: Path) -> list[str]:
    ids: list[str] = []
    in_provinces = False
    for line in yaml_path.read_text().splitlines():
        if line.startswith("provinces:"):
            in_provinces = True
            continue
        if in_provinces:
            if line and not line.startswith(" "):
                break
            m = PROV_RE.match(line)
            if m:
                ids.append(m.group(1))
    return ids


def main() -> int:
    fail = 0
    for yaml_path in sorted(YAML_DIR.glob("*.yaml")):
        map_id = None
        for line in yaml_path.read_text().splitlines():
            if line.startswith("id:"):
                map_id = line.split(":", 1)[1].strip()
                break
        if not map_id:
            print(f"skip {yaml_path}: no id", file=sys.stderr)
            continue
        svg_path = SVG_DIR / f"{map_id}.svg"
        if not svg_path.exists():
            print(f"missing {svg_path}", file=sys.stderr)
            fail += 1
            continue
        svg_text = svg_path.read_text()
        for pid in province_ids(yaml_path):
            needle = f'data-province-id="{pid}"'
            if needle not in svg_text:
                print(f"{svg_path.name}: missing {needle}", file=sys.stderr)
                fail += 1
    if fail:
        return 1
    print("map SVG validation OK")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
