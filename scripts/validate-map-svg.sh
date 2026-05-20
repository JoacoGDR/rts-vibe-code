#!/usr/bin/env bash
# Validates web/public/maps/{id}.svg province ids match embedded YAML maps.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
exec python3 "$ROOT/scripts/validate_map_svg.py"
