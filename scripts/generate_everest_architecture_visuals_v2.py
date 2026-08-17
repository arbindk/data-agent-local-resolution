from __future__ import annotations

import math
import os
from pathlib import Path
from typing import Iterable, Sequence

from PIL import Image, ImageDraw, ImageFont


ROOT = Path(__file__).resolve().parents[1]
OUT_DIR = ROOT / "docs" / "architecture_visuals"
PNG_OUT = OUT_DIR / "everest_data_agent_cell_architecture_v2.png"
GIF_OUT = OUT_DIR / "everest_data_agent_cell_sequence_v2.gif"

W, H = 3200, 2100

NAVY = "#08245c"
NAVY2 = "#0b3178"
BLUE = "#0f62c9"
LIGHT_BLUE = "#eaf3ff"
GREEN = "#0b7d4b"
LIGHT_GREEN = "#eaf8f0"
PURPLE = "#5b35b1"
LIGHT_PURPLE = "#f4efff"
ORANGE = "#f59e0b"
LIGHT_ORANGE = "#fff7e6"
RED = "#c0261b"
LIGHT_RED = "#fff0ef"
GRAY = "#64748b"
LIGHT_GRAY = "#f8fafc"
BORDER = "#b7c7df"
TEXT = "#061b46"
MUTED = "#41526e"
WHITE = "#ffffff"


def font(size: int, bold: bool = False) -> ImageFont.FreeTypeFont | ImageFont.ImageFont:
    candidates = [
        "C:/Windows/Fonts/segoeuib.ttf" if bold else "C:/Windows/Fonts/segoeui.ttf",
        "C:/Windows/Fonts/arialbd.ttf" if bold else "C:/Windows/Fonts/arial.ttf",
    ]
    for p in candidates:
        if os.path.exists(p):
            return ImageFont.truetype(p, size=size)
    return ImageFont.load_default()


F = {
    "title": font(40, True),
    "subtitle": font(22, True),
    "h1": font(22, True),
    "h2": font(18, True),
    "h3": font(15, True),
    "body": font(14),
    "body_bold": font(14, True),
    "small": font(12),
    "small_bold": font(12, True),
    "tiny": font(10),
    "tiny_bold": font(10, True),
}


def text_size(draw: ImageDraw.ImageDraw, text: str, fnt: ImageFont.ImageFont) -> tuple[int, int]:
    b = draw.textbbox((0, 0), text, font=fnt)
    return b[2] - b[0], b[3] - b[1]


def wrap(draw: ImageDraw.ImageDraw, text: str, max_width: int, fnt: ImageFont.ImageFont) -> list[str]:
    words = str(text).split()
    if not words:
        return [""]
    out: list[str] = []
    line = words[0]
    for word in words[1:]:
        trial = f"{line} {word}"
        if text_size(draw, trial, fnt)[0] <= max_width:
            line = trial
        else:
            out.append(line)
            line = word
    out.append(line)
    return out


def draw_wrapped(
    draw: ImageDraw.ImageDraw,
    xy: tuple[int, int],
    text: str,
    max_width: int,
    fnt: ImageFont.ImageFont,
    fill: str = TEXT,
    line_gap: int = 4,
    max_lines: int | None = None,
) -> int:
    x, y = xy
    lines = wrap(draw, text, max_width, fnt)
    if max_lines is not None and len(lines) > max_lines:
        lines = lines[:max_lines]
        lines[-1] = lines[-1].rstrip(".") + "..."
    for line in lines:
        draw.text((x, y), line, font=fnt, fill=fill)
        y += text_size(draw, line, fnt)[1] + line_gap
    return y


def round_box(draw: ImageDraw.ImageDraw, box: tuple[int, int, int, int], fill: str, outline: str = BORDER, width: int = 2, radius: int = 16):
    draw.rounded_rectangle(box, radius=radius, fill=fill, outline=outline, width=width)


def header_panel(
    draw: ImageDraw.ImageDraw,
    box: tuple[int, int, int, int],
    title: str,
    fill: str = WHITE,
    outline: str = BORDER,
    header: str = NAVY,
    title_color: str = WHITE,
    number: str | None = None,
):
    x1, y1, x2, y2 = box
    round_box(draw, box, fill, outline, 2, 12)
    draw.rounded_rectangle((x1, y1, x2, y1 + 44), radius=12, fill=header, outline=header)
    draw.rectangle((x1, y1 + 26, x2, y1 + 44), fill=header)
    tx = x1 + 16
    if number:
        draw.ellipse((x1 + 12, y1 + 9, x1 + 36, y1 + 33), fill=WHITE)
        draw.text((x1 + 21, y1 + 11), number, font=F["tiny_bold"], fill=header)
        tx = x1 + 46
    draw_wrapped(draw, (tx, y1 + 10), title, x2 - tx - 10, F["h3"], title_color, 1, 1)


def bullets(
    draw: ImageDraw.ImageDraw,
    xy: tuple[int, int],
    items: Sequence[str],
    max_width: int,
    color: str = TEXT,
    bullet_color: str = BLUE,
    fnt: ImageFont.ImageFont = F["small"],
    line_gap: int = 3,
    item_gap: int = 6,
) -> int:
    x, y = xy
    for item in items:
        draw.ellipse((x, y + 5, x + 6, y + 11), fill=bullet_color)
        y = draw_wrapped(draw, (x + 16, y), item, max_width - 16, fnt, color, line_gap)
        y += item_gap
    return y


def arrow(draw: ImageDraw.ImageDraw, start: tuple[int, int], end: tuple[int, int], color: str = NAVY, width: int = 3, dashed: bool = False):
    x1, y1 = start
    x2, y2 = end
    if dashed:
        dx, dy = x2 - x1, y2 - y1
        dist = max(1.0, math.hypot(dx, dy))
        segs = max(2, int(dist // 18))
        for i in range(segs):
            if i % 2 == 0:
                a, b = i / segs, min(1, (i + 1) / segs)
                draw.line((x1 + dx * a, y1 + dy * a, x1 + dx * b, y1 + dy * b), fill=color, width=width)
    else:
        draw.line((x1, y1, x2, y2), fill=color, width=width)
    ang = math.atan2(y2 - y1, x2 - x1)
    size = 12
    pts = [
        (x2, y2),
        (x2 - size * math.cos(ang - math.pi / 6), y2 - size * math.sin(ang - math.pi / 6)),
        (x2 - size * math.cos(ang + math.pi / 6), y2 - size * math.sin(ang + math.pi / 6)),
    ]
    draw.polygon(pts, fill=color)


def mini_card(
    draw: ImageDraw.ImageDraw,
    box: tuple[int, int, int, int],
    title: str,
    items: Sequence[str],
    fill: str = WHITE,
    outline: str = BORDER,
    accent: str = BLUE,
    number: str | None = None,
):
    x1, y1, x2, y2 = box
    round_box(draw, box, fill, outline, 2, 10)
    if number:
        draw.ellipse((x1 + 12, y1 + 12, x1 + 36, y1 + 36), fill=accent)
        draw.text((x1 + 21, y1 + 14), number, font=F["tiny_bold"], fill=WHITE)
        draw_wrapped(draw, (x1 + 46, y1 + 12), title, x2 - x1 - 58, F["small_bold"], TEXT, 1, 2)
        by = y1 + 52
    else:
        draw_wrapped(draw, (x1 + 14, y1 + 12), title, x2 - x1 - 28, F["small_bold"], TEXT, 1, 2)
        by = y1 + 44
    bullets(draw, (x1 + 16, by), items, x2 - x1 - 28, MUTED, accent, F["tiny"], 2, 4)


def top_badge(draw: ImageDraw.ImageDraw, x: int, title: str, body: str, color: str):
    round_box(draw, (x, 20, x + 360, 96), "#071b45", "#a6b8d8", 2, 12)
    draw.ellipse((x + 18, 42, x + 46, 70), outline=WHITE, width=3)
    draw.ellipse((x + 26, 50, x + 38, 62), fill=color)
    draw.text((x + 62, 30), title, font=F["small_bold"], fill=WHITE)
    draw_wrapped(draw, (x + 62, 52), body, 275, F["tiny"], "#dce8ff", 2, 2)


def draw_icon_database(draw: ImageDraw.ImageDraw, cx: int, cy: int, color: str):
    draw.ellipse((cx - 24, cy - 20, cx + 24, cy), outline=color, width=3)
    draw.rectangle((cx - 24, cy - 10, cx + 24, cy + 25), outline=color, width=3)
    draw.arc((cx - 24, cy + 10, cx + 24, cy + 30), 0, 180, fill=color, width=3)


def draw_icon_doc(draw: ImageDraw.ImageDraw, x: int, y: int, color: str):
    draw.rectangle((x, y, x + 42, y + 54), outline=color, width=3)
    draw.line((x + 10, y + 18, x + 32, y + 18), fill=color, width=2)
    draw.line((x + 10, y + 30, x + 32, y + 30), fill=color, width=2)
    draw.line((x + 10, y + 42, x + 26, y + 42), fill=color, width=2)


def create_board() -> Image.Image:
    img = Image.new("RGB", (W, H), WHITE)
    draw = ImageDraw.Draw(img)
    draw.rectangle((0, 0, W, 130), fill=NAVY)
    draw.text((34, 22), "EVEREST DATA AGENT CELL - AUTONOMOUS MVS1 ARCHITECTURE", font=F["title"], fill=WHITE)
    draw.text((36, 72), "Zero-copy by default. Orchestrator-first. Guardian-enforced. Generic connectors. Decision provenance for every mutation.", font=F["subtitle"], fill="#dbeafe")
    top_badge(draw, 2040, "Colocated With Data", "Near data sources to minimize movement and latency.", GREEN)
    top_badge(draw, 2410, "Autonomous Guardrails", "Auto-action only when policy explicitly allows.", ORANGE)
    top_badge(draw, 2780, "Policy Always Enforced", "No mutation bypasses Guardian.", RED)

    # Left column
    header_panel(draw, (20, 155, 500, 455), "1. PURPOSE", LIGHT_GRAY, "#d6e2f2", BLUE, number="1")
    draw_wrapped(draw, (45, 215), "The Data Agent Cell is a colocated runtime that receives MVS1 intent, discovers and analyzes data through generic connectors, obtains Guardian decisions, executes approved autonomous tagging, creates HITL tasks when required, and records provenance for every decision.", 420, F["body"], TEXT, 5)

    header_panel(draw, (20, 475, 500, 820), "2. SCOPE - MVS1 AUTONOMOUS DSPM", LIGHT_GREEN, "#c9e8d7", GREEN, number="2")
    draw.text((48, 535), "Autonomous DSPM Actions", font=F["small_bold"], fill=GREEN)
    bullets(draw, (48, 565), [
        "Discover sources, containers, objects and metadata",
        "Classify, score, infer labels and protection flags",
        "Extract bounded content features when policy allows",
        "Auto-tag allowed objects with dd_protection_flags",
        "Create HITL tasks for exceptions and obligations",
    ], 405, TEXT, GREEN, F["small"])
    draw.text((48, 705), "Advanced / Expert Actions", font=F["small_bold"], fill=GREEN)
    bullets(draw, (48, 735), [
        "Workbench investigation",
        "Connector diagnostics",
        "Evidence and provenance review",
    ], 405, TEXT, GREEN, F["small"])

    header_panel(draw, (20, 840, 500, 1160), "3. WHY NO EXTERNAL QUEUES / BUS", WHITE, "#d6e2f2", NAVY, number="3")
    bullets(draw, (48, 900), [
        "Direct orchestrated control path keeps latency and failure modes simple",
        "Durable local state is allowed: leases, checkpoints, cursors, retries",
        "No pub/sub dependency for the mandatory enforcement path",
        "Deterministic lifecycle: request, decision, execution, provenance",
    ], 405, TEXT, NAVY, F["small"])

    # Main architecture panel
    header_panel(draw, (540, 155, 2445, 1190), "4. DATA AGENT CELL - INTERNAL ARCHITECTURE (All communication via ORCHESTRATOR)", WHITE, "#9bb6d9", BLUE, number="4")
    round_box(draw, (575, 215, 2040, 1135), "#fbfdff", NAVY, 5, 18)
    draw.rectangle((575, 215, 2040, 285), fill=NAVY)
    draw.text((1170, 230), "ORCHESTRATOR", font=F["h1"], fill=WHITE)
    draw.text((1050, 256), "Single control plane and only sanctioned runtime entry/exit", font=F["small_bold"], fill="#dbeafe")

    top_y = 315
    card_w = 210
    gap = 16
    modules = [
        ("A", "REQUEST INTAKE", ["MVS1 intent", "authn / actor", "request envelope", "correlation ID"]),
        ("B", "CONTEXT BUILDER", ["identity resolution", "source + capability", "classification context", "canonical request"]),
        ("C", "POLICY CLIENT", ["bundle verify", "call Guardian", "cache decisions", "fail closed"]),
        ("D", "WORKFLOW & STATE", ["orchestration steps", "leases/checkpoints", "retries/backoff", "compensation"]),
        ("E", "ROUTING & COORDINATION", ["route worker", "load manager", "cursor/range scheduler", "backpressure"]),
        ("F", "RESPONSE COMPOSER", ["aggregate results", "apply obligations", "return receipt", "surface next task"]),
    ]
    for i, (num, title, items) in enumerate(modules):
        x = 595 + i * (card_w + gap)
        mini_card(draw, (x, top_y, x + card_w, top_y + 230), title, items, LIGHT_BLUE, "#c3d9f4", BLUE, num)

    round_box(draw, (595, 590, 2018, 665), LIGHT_PURPLE, PURPLE, 2, 10)
    draw.text((1040, 607), "FUNCTION CALLING (In-Process, Orchestrator-Mediated)", font=F["h3"], fill=PURPLE)
    draw.text((915, 632), "Direct synchronous calls to components. No external bus on the enforcement path.", font=F["tiny_bold"], fill=MUTED)
    for x in [700, 925, 1150, 1375, 1600, 1825]:
        arrow(draw, (x, top_y + 230), (x, 590), PURPLE, 2, dashed=True)

    worker_y = 705
    worker_w = 225
    workers = [
        ("1", "CONNECTOR ADAPTERS", ["Azure Blob first", "files / DBs / APIs", "capabilities", "object refs"]),
        ("2", "ZERO-COPY ACCESSOR", ["handle/reference", "stream/range/cursor", "bounded preview", "ephemeral extraction"]),
        ("3", "DISCOVERY & ANALYSIS", ["discover/scan", "metadata inference", "feature extraction", "scoring"]),
        ("4", "DUCKDB WORKING MEMORY", ["normalized metadata", "batch features", "scores/aggregates", "reset per batch"]),
        ("5", "TELEMETRY COLLECTOR", ["metrics", "logs", "traces", "health signals"]),
        ("6", "PROVENANCE WRITER", ["decision record", "action receipt", "policy hash", "tamper evidence"]),
    ]
    for i, (num, title, items) in enumerate(workers):
        x = 595 + i * (worker_w + 9)
        color = [ORANGE, GREEN, BLUE, GREEN, BLUE, PURPLE][i]
        fill = [LIGHT_ORANGE, LIGHT_GREEN, LIGHT_BLUE, LIGHT_GREEN, LIGHT_BLUE, LIGHT_PURPLE][i]
        mini_card(draw, (x, worker_y, x + worker_w, worker_y + 260), title, items, fill, color, color, num)
        arrow(draw, (x + worker_w // 2, 665), (x + worker_w // 2, worker_y), PURPLE, 2, dashed=True)

    round_box(draw, (595, 1000, 2018, 1115), "#f4f0ff", PURPLE, 2, 12)
    draw.text((1040, 1024), "LOCAL STATE (Embedded)", font=F["h3"], fill=PURPLE)
    draw.text((920, 1054), "Config references, checkpoints, leases, keys, bounded cache, batch-local DuckDB state", font=F["small"], fill=MUTED)

    # Guardian and upstream side
    header_panel(draw, (2075, 365, 2415, 750), "GUARDIAN AGENT", LIGHT_GREEN, "#8ed1aa", GREEN)
    draw.text((2165, 420), "Authoritative Enforcement", font=F["small_bold"], fill=GREEN)
    bullets(draw, (2110, 470), [
        "authorization decision",
        "risk and compliance scoring",
        "allow / deny / modify / HITL",
        "obligations: tag, mask, notify, limit",
        "policy bundle version + hash",
        "owns decision provenance",
    ], 260, TEXT, GREEN, F["small"])
    arrow(draw, (2040, 450), (2075, 450), RED, 4)
    draw.text((2000, 410), "Decision request", font=F["tiny_bold"], fill=RED)
    arrow(draw, (2075, 545), (2040, 545), RED, 4)
    draw.text((2000, 560), "Decision + obligations", font=F["tiny_bold"], fill=RED)

    header_panel(draw, (2075, 790, 2415, 1135), "UPSTREAM SYSTEMS", LIGHT_BLUE, "#a8c8ee", BLUE)
    bullets(draw, (2110, 850), [
        "policy service / registry",
        "identity provider",
        "key management",
        "task service for HITL",
        "Elasticsearch summary index",
        "evidence export / SIEM",
    ], 260, TEXT, BLUE, F["small"])
    arrow(draw, (1880, 960), (2075, 960), BLUE, 3, dashed=True)

    # Data lane under main
    round_box(draw, (540, 1220, 2040, 1445), LIGHT_GREEN, GREEN, 2, 14)
    draw.text((960, 1240), "COLOCATED WITH DATA - ZERO-COPY ACCESS (IN-PLACE)", font=F["h3"], fill=GREEN)
    data_cards = [
        ("FILE / OBJECT DATA", ["Azure Blob", "Object stores", "file shares"]),
        ("DATABASES", ["structured", "unstructured", "query/cursor"]),
        ("SAAS / APIS / STREAMS", ["SharePoint", "custom APIs", "streams"]),
    ]
    for i, (title, items) in enumerate(data_cards):
        x = 600 + i * 465
        mini_card(draw, (x, 1295, x + 390, 1408), title, items, WHITE, "#9dd7b6", GREEN)
    arrow(draw, (760, 1220), (760, 965), GREEN, 3, dashed=True)
    arrow(draw, (1000, 1220), (1000, 965), GREEN, 3, dashed=True)
    arrow(draw, (1360, 1220), (1360, 965), GREEN, 3, dashed=True)

    # Right column
    header_panel(draw, (2470, 155, 3180, 590), "5. ZERO-COPY DATA ACCESS - HOW IT WORKS", WHITE, "#d6e2f2", BLUE, number="5")
    mini_card(draw, (2510, 230, 2755, 310), "Data Source", ["file / DB / object / API"], LIGHT_BLUE, "#9bb6d9", BLUE)
    mini_card(draw, (2510, 350, 2755, 430), "Connector Adapter", ["opens handle/session", "returns object reference"], WHITE, "#9bb6d9", BLUE)
    mini_card(draw, (2510, 470, 2755, 550), "Zero-Copy Accessor", ["stream/range/cursor", "bounded preview only"], LIGHT_GREEN, "#8ed1aa", GREEN)
    arrow(draw, (2630, 310), (2630, 350), BLUE)
    arrow(draw, (2630, 430), (2630, 470), GREEN)
    bullets(draw, (2790, 235), [
        "No persistent full-object copy by default",
        "Feature extraction is ephemeral",
        "Explicit export/download needs Guardian approval",
        "Offsets, ranges, hashes and counts recorded",
    ], 340, TEXT, BLUE, F["small"])

    header_panel(draw, (2470, 615, 3180, 1035), "6. ZERO-COPY CONCURRENCY MODEL", LIGHT_GREEN, "#c9e8d7", GREEN, number="6")
    mini_card(draw, (2510, 680, 2825, 845), "Read Paths", [
        "immutable read operations",
        "shared concurrent workers",
        "disjoint range reads",
        "no locks required"
    ], WHITE, "#8ed1aa", GREEN)
    mini_card(draw, (2845, 680, 3140, 845), "Write / Mutation Paths", [
        "controlled by Orchestrator",
        "copy-on-write if required",
        "append-only preferred",
        "exclusive locks for rare conflicts"
    ], WHITE, "#8ed1aa", GREEN)
    round_box(draw, (2535, 900, 3115, 990), WHITE, "#8ed1aa", 2, 10)
    draw.text((2700, 922), "Shared Data, No Shared Mutable State", font=F["small_bold"], fill=GREEN)
    draw.text((2590, 952), "Zero-copy reads, safe writes only when required and approved", font=F["small"], fill=MUTED)

    header_panel(draw, (2470, 1060, 3180, 1445), "7. LIMITATIONS & TRADE-OFFS", WHITE, "#f3cf98", ORANGE, number="7")
    bullets(draw, (2505, 1125), [
        "Some formats/protocols require controlled temporary copies",
        "High concurrency can require buffers for ranges and maps",
        "Connector/source behavior affects performance",
        "Guardian unavailable behavior is fail-closed unless configured",
        "Full content export must be approved and audited",
    ], 620, TEXT, ORANGE, F["small"])

    # Bottom row: contract, lifecycle, provenance
    header_panel(draw, (20, 1490, 1120, 1910), "8. GUARDIAN & DATA AGENT CONTRACT (ENFORCEMENT GUARANTEE)", LIGHT_RED, "#f1aaa6", RED, number="8")
    mini_card(draw, (55, 1560, 340, 1820), "Hard Enforcement Rules", [
        "No local allow rules",
        "No cached allow bypass",
        "No executor call without decision",
        "Fail closed by default",
        "All calls through Orchestrator"
    ], WHITE, "#f1aaa6", RED)
    mini_card(draw, (410, 1560, 675, 1820), "Data Agent Caller", [
        "decision request",
        "context: actor/source/action",
        "obligation enforcement",
        "action receipt"
    ], WHITE, "#9bb6d9", BLUE)
    mini_card(draw, (750, 1560, 1085, 1820), "Guardian Agent", [
        "evaluate policy",
        "return decision",
        "return obligations",
        "record provenance",
        "fully auditable"
    ], WHITE, "#8ed1aa", GREEN)
    arrow(draw, (340, 1640), (410, 1640), BLUE)
    arrow(draw, (675, 1640), (750, 1640), RED)
    arrow(draw, (750, 1720), (675, 1720), RED)
    draw.text((175, 1865), "Orchestrator will not dispatch an executor call without a valid Guardian ALLOW/MODIFY decision.", font=F["small_bold"], fill=RED)

    header_panel(draw, (1150, 1490, 1775, 1910), "9. ACTION LIFECYCLE (END-TO-END)", WHITE, "#d6e2f2", BLUE, number="9")
    lifecycle = [
        "Client/schedule submits intent to Orchestrator",
        "Orchestrator creates request envelope",
        "Connector discovers source/object metadata",
        "Zero-copy accessor extracts bounded features",
        "DuckDB aggregates batch signals",
        "Guardian returns decision + obligations",
        "Allowed actions execute; HITL tasks created if required",
        "Guardian writes decision provenance",
        "Orchestrator submits summary to Elasticsearch",
        "Portal returns response and next task"
    ]
    y = 1552
    for i, item in enumerate(lifecycle, 1):
        draw.ellipse((1180, y, 1204, y + 24), fill=BLUE)
        draw.text((1188 if i < 10 else 1184, y + 3), str(i), font=F["tiny_bold"], fill=WHITE)
        y = draw_wrapped(draw, (1218, y), item, 510, F["small"], TEXT, 2, 1) + 5

    header_panel(draw, (1805, 1490, 2380, 1910), "10. PROVENANCE (WHAT IS RECORDED)", WHITE, "#d6e2f2", BLUE, number="10")
    bullets(draw, (1840, 1552), [
        "Request: actor, tenant, source, operation, scope",
        "Decision: allow/deny/modify/HITL, obligations",
        "Policy: bundle version, hash, rules applied",
        "Feature extraction: offsets/ranges/hashes/counts",
        "Action receipt: executor, parameters, result",
        "Integrity: timestamp, signature/hash, correlation ID",
    ], 490, TEXT, BLUE, F["small"])

    header_panel(draw, (2410, 1490, 3180, 1910), "11. MVS1 AUTONOMOUS AUTO-TAGGING", LIGHT_GREEN, "#c9e8d7", GREEN, number="11")
    bullets(draw, (2445, 1552), [
        "Default posture: autonomous within policy guardrails",
        "Allowed: auto-apply dd_protection_flags and approved metadata/tags",
        "HITL: created when Guardian requires human review",
        "AI: recommend, explain, classify, extract features, prefill HITL",
        "Never allowed: AI or Workbench direct mutation bypass",
        "Goal: HITL volume diminishes as policies mature",
    ], 680, TEXT, GREEN, F["small"])

    # Legend and key takeaway
    round_box(draw, (20, 1940, 710, 2070), WHITE, "#d6e2f2", 2, 10)
    draw.text((45, 1960), "LEGEND", font=F["small_bold"], fill=NAVY)
    legend = [(BLUE, "Orchestrator functions"), (ORANGE, "Connector/workers"), (GREEN, "Data/source/working state"), (RED, "Guardian enforcement"), (PURPLE, "Provenance/indexing")]
    lx, ly = 45, 1995
    for color, name in legend:
        draw.rectangle((lx, ly, lx + 24, ly + 18), outline=color, width=3)
        draw.text((lx + 34, ly - 1), name, font=F["tiny"], fill=TEXT)
        lx += 210
        if lx > 600:
            lx = 45
            ly += 34

    round_box(draw, (740, 1940, 2280, 2070), WHITE, "#d6e2f2", 2, 10)
    draw.text((760, 1960), "COMMUNICATION", font=F["small_bold"], fill=NAVY)
    arrow(draw, (760, 2015), (850, 2015), BLUE)
    draw.text((865, 2004), "Synchronous in-process call", font=F["tiny"], fill=TEXT)
    arrow(draw, (1080, 2015), (1170, 2015), BLUE, dashed=True)
    draw.text((1185, 2004), "State/evidence/indexing handoff", font=F["tiny"], fill=TEXT)
    arrow(draw, (1510, 2015), (1600, 2015), RED)
    draw.text((1615, 2004), "Guardian decision boundary", font=F["tiny"], fill=TEXT)

    round_box(draw, (2310, 1940, 3180, 2070), NAVY, NAVY, 2, 10)
    draw.text((2335, 1960), "KEY TAKEAWAY", font=F["small_bold"], fill=WHITE)
    draw_wrapped(draw, (2335, 1995), "Everest MVS1 is an autonomous, orchestrator-first Data Agent Cell. It can act automatically only when Guardian policy explicitly allows it; otherwise it creates HITL tasks and records decision provenance.", 800, F["small"], "#dbeafe", 4)

    return img


SEQ_NODES = {
    "intent": (50, 150, 315, 265, "Portal / Intent"),
    "envelope": (385, 150, 650, 265, "Request Envelope"),
    "orch": (720, 115, 1045, 300, "Data Agent Cell Orchestrator"),
    "conn": (50, 420, 315, 535, "Generic Connector"),
    "access": (385, 420, 650, 535, "Zero-Copy Accessor"),
    "duck": (720, 420, 1045, 535, "DuckDB Working Memory"),
    "guardian": (1115, 150, 1435, 330, "Guardian Decision"),
    "execute": (1115, 420, 1435, 535, "Approved Action / HITL"),
    "prov": (1500, 150, 1810, 330, "Decision Provenance"),
    "elastic": (1500, 420, 1810, 535, "Elasticsearch Summary"),
}


SEQ_FRAMES = [
    ("1. Intent", "User, schedule, or autonomous policy trigger submits MVS1 intent. Portal does not execute directly.", ["intent", "envelope", "orch"]),
    ("2. Normalize Request", "Orchestrator converts intent into a request envelope with actor, tenant, source, operation, scope and policy context.", ["envelope", "orch"]),
    ("3. Discover By Generic Connector", "Connector discovers stores, containers, objects, metadata, tags and capabilities through a normalized contract.", ["orch", "conn"]),
    ("4. Zero-Copy Access", "Accessor reads by handle, stream, range, cursor or bounded preview. Feature extraction is ephemeral.", ["conn", "access"]),
    ("5. Analyze In DuckDB", "DuckDB holds transient normalized metadata, feature vectors, scores and batch aggregation state.", ["access", "duck"]),
    ("6. Ask Guardian", "Orchestrator sends context to Guardian. Guardian evaluates policy, risk, obligations and HITL requirement.", ["orch", "guardian"]),
    ("7. Autonomous Or HITL", "Allowed actions execute automatically. HITL tasks are created when Guardian requires review.", ["guardian", "execute"]),
    ("8. Record Provenance", "Guardian records policy version/hash, rules, reasoning, decision, obligations, actor and timestamp.", ["guardian", "prov"]),
    ("9. Submit Batch Summary", "Orchestrator submits aggregated results and normalized signals to Elasticsearch at batch completion.", ["duck", "elastic", "prov"]),
    ("10. Evidence And Next Task", "Portal shows evidence, action receipt, HITL status, provenance and recommended next action.", ["intent", "prov", "elastic"]),
]


def draw_sequence_frame(title: str, subtitle: str, active: Iterable[str], idx: int) -> Image.Image:
    img = Image.new("RGB", (1900, 1080), WHITE)
    draw = ImageDraw.Draw(img)
    draw.rectangle((0, 0, 1900, 115), fill=NAVY)
    draw.text((34, 22), "EVEREST DATA AGENT CELL - AUTONOMOUS SEQUENCE FLOW", font=font(34, True), fill=WHITE)
    draw.text((1600, 35), f"{idx}/10", font=F["subtitle"], fill="#dbeafe")
    draw.text((45, 140), title, font=font(30, True), fill=TEXT)
    draw_wrapped(draw, (45, 182), subtitle, 1250, F["body"], MUTED, 5, 2)

    active_set = set(active)
    for key, (x1, y1, x2, y2, label) in SEQ_NODES.items():
        is_active = key in active_set
        fill = LIGHT_ORANGE if is_active else LIGHT_GRAY
        outline = ORANGE if is_active else BORDER
        width = 5 if is_active else 2
        round_box(draw, (x1, y1, x2, y2), fill, outline, width, 16)
        draw_wrapped(draw, (x1 + 18, y1 + 30), label, x2 - x1 - 36, F["h2"], TEXT, 4, 2)

    # Main communication lines
    lines = [
        ("intent", "envelope", BLUE),
        ("envelope", "orch", BLUE),
        ("orch", "conn", BLUE),
        ("conn", "access", GREEN),
        ("access", "duck", GREEN),
        ("orch", "guardian", RED),
        ("guardian", "execute", RED),
        ("guardian", "prov", PURPLE),
        ("execute", "prov", PURPLE),
        ("duck", "elastic", PURPLE),
        ("elastic", "prov", PURPLE),
    ]
    centers = {k: ((v[0] + v[2]) // 2, (v[1] + v[3]) // 2) for k, v in SEQ_NODES.items()}
    for a, b, c in lines:
        ca, cb = centers[a], centers[b]
        color = c if a in active_set or b in active_set else "#9aa8bc"
        arrow(draw, ca, cb, color, 4 if color == c else 2, dashed=False)

    # Invariant strip
    round_box(draw, (50, 720, 1810, 965), "#fbfdff", "#d6e2f2", 2, 18)
    draw.text((80, 745), "Enforced Invariants In This Flow", font=F["h1"], fill=NAVY)
    invariants = [
        "Portal cannot mutate directly",
        "AI cannot mutate directly",
        "Guardian required before mutation",
        "Policy version/hash on decision",
        "DuckDB is transient",
        "No persistent full data copy by default",
        "Every action attempt has provenance",
        "Azure Blob is one connector implementation",
    ]
    x, y = 80, 800
    for inv in invariants:
        draw.ellipse((x, y + 4, x + 20, y + 24), fill=GREEN)
        draw.line((x + 5, y + 14, x + 9, y + 19, x + 16, y + 9), fill=WHITE, width=3)
        draw_wrapped(draw, (x + 30, y), inv, 380, F["body"], TEXT, 4, 2)
        x += 430
        if x > 1600:
            x = 80
            y += 70

    draw.rectangle((0, 1030, 1900, 1080), fill=NAVY)
    draw.text((45, 1042), "Primary MVS1 path: autonomous auto-tagging within Guardian policy guardrails; HITL only when required.", font=F["body_bold"], fill=WHITE)
    return img


def create_gif() -> list[Image.Image]:
    return [draw_sequence_frame(title, body, active, i) for i, (title, body, active) in enumerate(SEQ_FRAMES, 1)]


def main() -> None:
    OUT_DIR.mkdir(parents=True, exist_ok=True)
    board = create_board()
    board.save(PNG_OUT)
    frames = create_gif()
    frames[0].save(
        GIF_OUT,
        save_all=True,
        append_images=frames[1:],
        duration=1650,
        loop=0,
        optimize=False,
    )
    print(f"created {PNG_OUT}")
    print(f"created {GIF_OUT}")


if __name__ == "__main__":
    main()
