#!/usr/bin/env python3
"""Generate pkg/shared/maps/classic-4p.yaml and web/public/maps/classic-4p.svg."""
from __future__ import annotations

import math
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
YAML_PATH = ROOT / "pkg" / "shared" / "maps" / "classic-4p.yaml"
SVG_PATH = ROOT / "web" / "public" / "maps" / "classic-4p.svg"

W, H = 1200, 900

QUADRANTS = [
    ("red", "R", 200, 180, 1, 1),
    ("blue", "B", 1000, 180, -1, 1),
    ("green", "G", 200, 720, 1, -1),
    ("yellow", "Y", 1000, 720, -1, -1),
]

SLOTS = [
    ("red", "#e8523a", "R_CAP"),
    ("blue", "#3a8ce8", "B_CAP"),
    ("green", "#3ae87a", "G_CAP"),
    ("yellow", "#e8c43a", "Y_CAP"),
]

BORDER_EDGES = [
    ("R08", "B02"),
    ("R04", "B05"),
    ("R05", "G04"),
    ("R_CAP", "G_CAP"),
    ("B08", "Y02"),
    ("B04", "Y05"),
    ("G08", "Y02"),
    ("G04", "Y05"),
]


def province_grid(cx: float, cy: float, sx: int, sy: int) -> list[tuple[float, float]]:
    dx, dy = 85 * sx, 70 * sy
    cols = [cx - 1.5 * dx, cx - 0.5 * dx, cx + 0.5 * dx, cx + 1.5 * dx]
    rows = [cy - 0.5 * dy, cy + 0.5 * dy]
    out: list[tuple[float, float]] = []
    for y in rows:
        for x in cols:
            out.append((x, y))
    return out


def build_quadrant(slot: str, prefix: str, cx: float, cy: float, sx: int, sy: int) -> list[dict]:
    coords = province_grid(cx, cy, sx, sy)
    ids = [f"{prefix}_CAP"] + [f"{prefix}{i:02d}" for i in range(2, 9)]
    provinces = []
    for i, (pid, (x, y)) in enumerate(zip(ids, coords)):
        name = f"{prefix.title()} Capital" if i == 0 else f"{prefix.title()} {i + 1:02d}"
        provinces.append({
            "id": pid,
            "name": name,
            "x": round(x, 1),
            "y": round(y, 1),
            "home_slot": slot,
        })
    return provinces


def manhattan(a: dict, b: dict) -> float:
    return abs(a["x"] - b["x"]) + abs(a["y"] - b["y"])


def edges_for_cluster(provs: list[dict], max_dist: float = 125) -> list[tuple[str, str]]:
    edges: set[tuple[str, str]] = set()
    for i, a in enumerate(provs):
        for b in provs[i + 1 :]:
            if manhattan(a, b) <= max_dist:
                edges.add(tuple(sorted((a["id"], b["id"]))))
    return sorted(edges)


def hex_poly(cx: float, cy: float, r: float = 40) -> str:
    pts = []
    for i in range(6):
        ang = math.pi / 6 + i * math.pi / 3
        pts.append(f"{cx + r * math.cos(ang):.1f},{cy + r * math.sin(ang):.1f}")
    return " ".join(pts)


def write_yaml(provinces: list[dict], edges: list[tuple[str, str]]) -> None:
    lines = [
        "id: classic-4p",
        'name: "Classic 4-Player"',
        'rules_summary: "4 commanders · 8 provinces each · continental war"',
        "provinces:",
    ]
    for p in provinces:
        lines.append(
            f"  - {{ id: {p['id']}, name: {p['name']!r}, x: {p['x']}, y: {p['y']}, home_slot: {p['home_slot']} }}"
        )
    lines.append("edges:")
    for a, b in edges:
        lines.append(f"  - {{ from: {a}, to: {b} }}")
    lines.append("slots:")
    for sid, color, cap in SLOTS:
        lines.append(f'  - {{ id: {sid}, color: "{color}", capital: {cap} }}')
    lines.append("starting_units:")
    for sid, _, cap in SLOTS:
        lines.append(f"  - {{ slot: {sid}, type: infantry, province: {cap}, hp: 100 }}")
    YAML_PATH.write_text("\n".join(lines) + "\n")


def write_svg(provinces: list[dict], edges: list[tuple[str, str]]) -> None:
    by_id = {p["id"]: p for p in provinces}
    route_lines = []
    for a, b in edges:
        pa, pb = by_id[a], by_id[b]
        route_lines.append(
            f'    <line x1="{pa["x"]}" y1="{pa["y"]}" x2="{pb["x"]}" y2="{pb["y"]}" />'
        )
    polys = []
    for p in provinces:
        pts = hex_poly(p["x"], p["y"])
        polys.append(f'    <polygon data-province-id="{p["id"]}" points="{pts}" />')
    svg = f"""<svg viewBox="0 0 {W} {H}" xmlns="http://www.w3.org/2000/svg">
  <rect width="{W}" height="{H}" fill="#0c0f14" />
  <g fill="none" stroke="#252b38" stroke-width="2" opacity="0.7">
{chr(10).join(route_lines)}
  </g>
  <g stroke="#3d4555" stroke-width="1.5" fill-opacity="0.88">
{chr(10).join(polys)}
  </g>
</svg>
"""
    SVG_PATH.write_text(svg)


def main() -> None:
    all_provinces: list[dict] = []
    edges: set[tuple[str, str]] = set()
    for slot, prefix, cx, cy, sx, sy in QUADRANTS:
        provs = build_quadrant(slot, prefix, cx, cy, sx, sy)
        all_provinces.extend(provs)
        edges.update(edges_for_cluster(provs))
    for a, b in BORDER_EDGES:
        edges.add(tuple(sorted((a, b))))
    edges_sorted = sorted(edges)
    write_yaml(all_provinces, edges_sorted)
    write_svg(all_provinces, edges_sorted)
    print(f"wrote {YAML_PATH}")
    print(f"wrote {SVG_PATH}")
    print(f"provinces={len(all_provinces)} edges={len(edges_sorted)}")


if __name__ == "__main__":
    main()
