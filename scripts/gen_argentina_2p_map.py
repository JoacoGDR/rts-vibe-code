#!/usr/bin/env python3
"""Generate pkg/shared/maps/argentina-2p.yaml and web/public/maps/argentina-2p.svg."""
from __future__ import annotations

import re
import xml.etree.ElementTree as ET
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SRC_SVG = ROOT / "web" / "public" / "maps" / "argentina.svg"
YAML_PATH = ROOT / "pkg" / "shared" / "maps" / "argentina-2p.yaml"
SVG_PATH = ROOT / "web" / "public" / "maps" / "argentina-2p.svg"

MAP_ID = "argentina-2p"
VIEW_W, VIEW_H = 361.54608, 792.57880

NORTH = frozenset({
    "AR-Y", "AR-A", "AR-K", "AR-T", "AR-F", "AR-G",
    "AR-P", "AR-H", "AR-N", "AR-W", "AR-E", "AR-S",
})
SOUTH = frozenset({
    "AR-B", "AR-C", "AR-L", "AR-X", "AR-M", "AR-J",
    "AR-D", "AR-Q", "AR-R", "AR-U", "AR-Z", "AR-V",
})

# Geometry may miss thin borders; these are always included.
MANUAL_EDGES = [
    ("AR-C", "AR-B"),  # CABA enclave
    ("AR-V", "AR-Z"),  # Magellan strait link
    # Norte / Sur frontier and key chokepoints
    ("AR-S", "AR-X"),
    ("AR-S", "AR-B"),
    ("AR-E", "AR-B"),
    ("AR-E", "AR-S"),
    ("AR-G", "AR-X"),
    ("AR-G", "AR-D"),
    ("AR-F", "AR-J"),
    ("AR-F", "AR-D"),
    ("AR-F", "AR-X"),
    ("AR-A", "AR-J"),
    ("AR-K", "AR-J"),
    ("AR-K", "AR-X"),
    ("AR-T", "AR-X"),
    ("AR-H", "AR-S"),
    ("AR-W", "AR-S"),
    ("AR-P", "AR-A"),
    ("AR-B", "AR-L"),
    ("AR-L", "AR-S"),
    ("AR-Q", "AR-D"),
    ("AR-Q", "AR-X"),
    ("AR-R", "AR-B"),
]

NUM_RE = re.compile(r"[-+]?(?:\d+\.\d+|\d+)(?:[eE][-+]?\d+)?")


def path_subpaths(d: str) -> list[list[tuple[float, float]]]:
    """Split an SVG path into subpaths (each M/m starts a new contour)."""
    subpaths: list[list[tuple[float, float]]] = []
    for part in re.split(r"(?=[Mm])", d):
        part = part.strip()
        if not part:
            continue
        nums = [float(x) for x in NUM_RE.findall(part)]
        pts: list[tuple[float, float]] = []
        for i in range(0, len(nums) - 1, 2):
            pts.append((nums[i], nums[i + 1]))
        if pts:
            subpaths.append(pts)
    return subpaths


def path_points(d: str) -> list[tuple[float, float]]:
    pts: list[tuple[float, float]] = []
    for sub in path_subpaths(d):
        pts.extend(sub)
    return pts


def bbox_center(pts: list[tuple[float, float]]) -> tuple[float, float]:
    xs = [p[0] for p in pts]
    ys = [p[1] for p in pts]
    return ((min(xs) + max(xs)) / 2, (min(ys) + max(ys)) / 2)


def centroid(d: str) -> tuple[float, float]:
    subpaths = path_subpaths(d)
    if not subpaths:
        return (0.0, 0.0)
    # Use the largest subpath (main province shape, not exclaves/islands).
    main = max(subpaths, key=len)
    return bbox_center(main)


def quantize(pt: tuple[float, float], step: float = 0.75) -> tuple[float, float]:
    return (round(pt[0] / step) * step, round(pt[1] / step) * step)


def boundary_points(pts: list[tuple[float, float]], step: float = 0.75) -> set[tuple[float, float]]:
    return {quantize(p, step) for p in pts}


def dist(a: tuple[float, float], b: tuple[float, float]) -> float:
    return ((a[0] - b[0]) ** 2 + (a[1] - b[1]) ** 2) ** 0.5


def detect_edges(
    point_sets: dict[str, set[tuple[float, float]]],
    centers: dict[str, tuple[float, float]],
    min_shared: int = 6,
    max_center_dist: float = 200.0,
) -> set[tuple[str, str]]:
    edges: set[tuple[str, str]] = set()
    ids = sorted(point_sets.keys())
    for i, a in enumerate(ids):
        for b in ids[i + 1 :]:
            if len(point_sets[a] & point_sets[b]) < min_shared:
                continue
            if dist(centers[a], centers[b]) > max_center_dist:
                continue
            edges.add((a, b) if a < b else (b, a))
    return edges


def parse_source_svg() -> list[dict]:
    tree = ET.parse(SRC_SVG)
    root = tree.getroot()
    ns = {"svg": "http://www.w3.org/2000/svg"}
    provinces: list[dict] = []
    for path in root.findall(".//svg:path", ns) or root.findall(".//path"):
        pid = path.get("id") or ""
        if not pid.startswith("AR-"):
            continue
        title = path.get("title") or pid
        d = path.get("d") or ""
        pts = path_points(d)
        cx, cy = centroid(d)
        slot = "north" if pid in NORTH else "south"
        provinces.append({
            "id": pid,
            "name": title,
            "x": round(cx, 1),
            "y": round(cy, 1),
            "home_slot": slot,
            "points": pts,
            "d": d,
        })
    provinces.sort(key=lambda p: p["id"])
    return provinces


def build_edges(provinces: list[dict]) -> list[tuple[str, str]]:
    point_sets = {p["id"]: boundary_points(p["points"]) for p in provinces}
    centers = {p["id"]: (p["x"], p["y"]) for p in provinces}
    edges = detect_edges(point_sets, centers, min_shared=6)
    for a, b in MANUAL_EDGES:
        edges.add((a, b) if a < b else (b, a))
    return sorted(edges)


def yaml_quote_name(name: str) -> str:
    if "'" in name:
        return repr(name)
    return f"'{name}'"


def write_yaml(provinces: list[dict], edges: list[tuple[str, str]]) -> None:
    lines = [
        f"id: {MAP_ID}",
        'name: "Argentina — Norte vs Sur"',
        'rules_summary: "2 commanders · 24 provincias · Norte contra Sur"',
        "provinces:",
    ]
    for p in provinces:
        lines.append(
            f"  - {{ id: {p['id']}, name: {yaml_quote_name(p['name'])}, "
            f"x: {p['x']}, y: {p['y']}, home_slot: {p['home_slot']} }}"
        )
    lines.append("edges:")
    for a, b in edges:
        lines.append(f"  - {{ from: {a}, to: {b} }}")
    lines.append("slots:")
    lines.append('  - { id: north, color: "#c45c26", capital: AR-T }')
    lines.append('  - { id: south, color: "#2d6a9f", capital: AR-C }')
    lines.append("starting_units:")
    lines.append("  - { slot: north, type: infantry, province: AR-T, hp: 100 }")
    lines.append("  - { slot: south, type: infantry, province: AR-C, hp: 100 }")
    YAML_PATH.write_text("\n".join(lines) + "\n")


def write_svg(provinces: list[dict]) -> None:
    paths_xml: list[str] = []
    for p in provinces:
        paths_xml.append(
            f'    <path id="{p["id"]}" data-province-id="{p["id"]}" '
            f'title="{p["name"]}" d="{p["d"]}"/>'
        )
    svg = f"""<?xml version="1.0" encoding="utf-8"?>
<svg viewBox="0 0 {VIEW_W} {VIEW_H}" width="{VIEW_W}" height="{VIEW_H}"
  xmlns="http://www.w3.org/2000/svg">
  <rect width="{VIEW_W}" height="{VIEW_H}" fill="#0c0f14"/>
  <g class="province-layer" stroke-width="1.5" fill-opacity="0.88">
{chr(10).join(paths_xml)}
  </g>
</svg>
"""
    SVG_PATH.write_text(svg)


def print_adjacency_report(provinces: list[dict], edges: list[tuple[str, str]]) -> None:
    neighbors: dict[str, list[str]] = {p["id"]: [] for p in provinces}
    for a, b in edges:
        neighbors[a].append(b)
        neighbors[b].append(a)
    for pid in sorted(neighbors):
        nb = sorted(neighbors[pid])
        print(f"  {pid}: {len(nb)} neighbors -> {', '.join(nb)}")


def main() -> None:
    provinces = parse_source_svg()
    if len(provinces) != 24:
        raise SystemExit(f"expected 24 provinces, got {len(provinces)}")

    north_count = sum(1 for p in provinces if p["home_slot"] == "north")
    south_count = sum(1 for p in provinces if p["home_slot"] == "south")
    if north_count != 12 or south_count != 12:
        raise SystemExit(f"bad split: north={north_count} south={south_count}")

    edges = build_edges(provinces)
    write_yaml(provinces, edges)
    write_svg(provinces)

    print(f"wrote {YAML_PATH}")
    print(f"wrote {SVG_PATH}")
    print(f"provinces={len(provinces)} edges={len(edges)}")
    print("adjacency per province:")
    print_adjacency_report(provinces, edges)


if __name__ == "__main__":
    main()
