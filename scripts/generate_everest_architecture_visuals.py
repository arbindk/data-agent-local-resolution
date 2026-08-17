from __future__ import annotations

import math
import os
from pathlib import Path
from typing import Iterable, Sequence

from PIL import Image, ImageDraw, ImageFont


ROOT = Path(__file__).resolve().parents[1]
OUT_DIR = ROOT / "docs" / "architecture_visuals"
PNG_OUT = OUT_DIR / "everest_final_architecture_summary.png"
GIF_OUT = OUT_DIR / "everest_autonomous_flow_sequence.gif"


NAVY = "#08245c"
DEEP = "#071b45"
BLUE = "#0f5fbf"
LIGHT_BLUE = "#eaf3ff"
GREEN = "#0b7d4b"
LIGHT_GREEN = "#eaf8f0"
PURPLE = "#5b35b1"
LIGHT_PURPLE = "#f2edff"
ORANGE = "#f59e0b"
LIGHT_ORANGE = "#fff7e6"
RED = "#b42318"
LIGHT_RED = "#fff0ef"
GRAY = "#64748b"
LIGHT_GRAY = "#f7f9fc"
BORDER = "#b8c7df"
TEXT = "#071b45"
MUTED = "#475569"
WHITE = "#ffffff"


def font(size: int, bold: bool = False) -> ImageFont.FreeTypeFont | ImageFont.ImageFont:
    candidates = [
        "C:/Windows/Fonts/segoeuib.ttf" if bold else "C:/Windows/Fonts/segoeui.ttf",
        "C:/Windows/Fonts/arialbd.ttf" if bold else "C:/Windows/Fonts/arial.ttf",
    ]
    for path in candidates:
        if os.path.exists(path):
            return ImageFont.truetype(path, size=size)
    return ImageFont.load_default()


F = {
    "title": font(42, True),
    "subtitle": font(21, True),
    "section": font(19, True),
    "body": font(16),
    "body_bold": font(16, True),
    "small": font(13),
    "tiny": font(11),
    "tiny_bold": font(11, True),
}


def wrap_text(draw: ImageDraw.ImageDraw, text: str, max_width: int, fnt: ImageFont.ImageFont) -> list[str]:
    words = str(text).split()
    if not words:
        return [""]
    lines: list[str] = []
    current = words[0]
    for word in words[1:]:
        trial = f"{current} {word}"
        if draw.textbbox((0, 0), trial, font=fnt)[2] <= max_width:
            current = trial
        else:
            lines.append(current)
            current = word
    lines.append(current)
    return lines


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
    lines = wrap_text(draw, text, max_width, fnt)
    if max_lines is not None and len(lines) > max_lines:
        lines = lines[:max_lines]
        lines[-1] = lines[-1].rstrip(".") + "..."
    for line in lines:
        draw.text((x, y), line, font=fnt, fill=fill)
        y += draw.textbbox((0, 0), line, font=fnt)[3] + line_gap
    return y


def rounded(draw: ImageDraw.ImageDraw, box: tuple[int, int, int, int], fill: str, outline: str = BORDER, width: int = 2, radius: int = 18):
    draw.rounded_rectangle(box, radius=radius, fill=fill, outline=outline, width=width)


def label_box(
    draw: ImageDraw.ImageDraw,
    box: tuple[int, int, int, int],
    title: str,
    body: str | Sequence[str] = "",
    fill: str = WHITE,
    outline: str = BORDER,
    title_fill: str = NAVY,
    number: str | None = None,
    accent: str | None = None,
):
    x1, y1, x2, y2 = box
    rounded(draw, box, fill, outline, 2, 16)
    tx = x1 + 18
    ty = y1 + 15
    if number:
        draw.ellipse((x1 + 14, y1 + 14, x1 + 44, y1 + 44), fill=accent or BLUE)
        draw.text((x1 + 24, y1 + 17), number, font=F["tiny_bold"], fill=WHITE)
        tx = x1 + 54
    draw_wrapped(draw, (tx, ty), title, x2 - tx - 14, F["body_bold"], title_fill, 3, max_lines=2)
    body_y = y1 + 56 if number else y1 + 48
    if isinstance(body, str):
        items = [body] if body else []
    else:
        items = list(body)
    for item in items:
        if item.startswith("•"):
            draw.text((x1 + 18, body_y + 2), "•", font=F["small"], fill=title_fill)
            body_y = draw_wrapped(draw, (x1 + 36, body_y), item[1:].strip(), x2 - x1 - 52, F["small"], MUTED, 3, max_lines=2)
        else:
            body_y = draw_wrapped(draw, (x1 + 18, body_y), item, x2 - x1 - 36, F["small"], MUTED, 3, max_lines=3)


def arrow(draw: ImageDraw.ImageDraw, start: tuple[int, int], end: tuple[int, int], color: str = NAVY, width: int = 4, dashed: bool = False):
    x1, y1 = start
    x2, y2 = end
    if dashed:
        dx, dy = x2 - x1, y2 - y1
        dist = max(1, math.hypot(dx, dy))
        steps = int(dist // 18)
        for i in range(steps):
            if i % 2 == 0:
                a = i / steps
                b = min(1, (i + 1) / steps)
                draw.line((x1 + dx * a, y1 + dy * a, x1 + dx * b, y1 + dy * b), fill=color, width=width)
    else:
        draw.line((x1, y1, x2, y2), fill=color, width=width)
    ang = math.atan2(y2 - y1, x2 - x1)
    size = 14
    p1 = (x2, y2)
    p2 = (x2 - size * math.cos(ang - math.pi / 6), y2 - size * math.sin(ang - math.pi / 6))
    p3 = (x2 - size * math.cos(ang + math.pi / 6), y2 - size * math.sin(ang + math.pi / 6))
    draw.polygon([p1, p2, p3], fill=color)


def chip(draw: ImageDraw.ImageDraw, xy: tuple[int, int], text: str, color: str, w: int = 275):
    x, y = xy
    rounded(draw, (x, y, x + w, y + 54), fill="#f8fbff", outline=color, width=2, radius=14)
    draw.ellipse((x + 14, y + 14, x + 38, y + 38), fill=color)
    draw.text((x + 48, y + 10), text, font=F["small"], fill=TEXT)


def create_summary_png() -> Image.Image:
    img = Image.new("RGB", (2400, 1600), WHITE)
    draw = ImageDraw.Draw(img)
    draw.rectangle((0, 0, 2400, 130), fill=NAVY)
    draw.text((36, 24), "EVEREST MVS1 AUTONOMOUS DSPM ARCHITECTURE", font=F["title"], fill=WHITE)
    draw.text((38, 78), "Orchestrator-first. Guardian-enforced. Zero-copy by default. Generic connectors. Provenance for every decision.", font=F["subtitle"], fill="#dbeafe")
    chip(draw, (1450, 24), "Autonomous by default", GREEN, 280)
    chip(draw, (1745, 24), "Guardian before mutation", RED, 310)
    chip(draw, (2070, 24), "No persistent data copies", BLUE, 300)

    # Policy lane
    label_box(draw, (40, 160, 370, 390), "1. Policy Authoring", [
        "• Central governance definitions",
        "• Versioned and reviewed",
        "• Signed policy source"
    ], LIGHT_BLUE, "#91b9ef", number="1", accent=BLUE)
    label_box(draw, (410, 160, 740, 390), "2. Compile & Sign", [
        "• Cedar authorization bundle",
        "• Python compliance/risk rules",
        "• Policy bundle hash"
    ], LIGHT_PURPLE, "#b9a7ef", number="2", accent=PURPLE)
    label_box(draw, (780, 160, 1110, 390), "3. Distribution", [
        "• Immutable bundle",
        "• Pulled by Data Agent Cell",
        "• Verified before use"
    ], LIGHT_BLUE, "#91b9ef", number="3", accent=BLUE)
    arrow(draw, (370, 275), (410, 275), GRAY)
    arrow(draw, (740, 275), (780, 275), GRAY)
    arrow(draw, (1110, 275), (1810, 510), PURPLE, width=3, dashed=True)

    # Data agent cell
    rounded(draw, (40, 430, 1730, 1110), fill="#f8fbff", outline=NAVY, width=4, radius=24)
    draw.rectangle((40, 430, 1730, 500), fill=NAVY)
    draw.text((70, 448), "DATA AGENT CELL - ORCHESTRATOR-FIRST RUNTIME", font=F["section"], fill=WHITE)
    draw.text((70, 474), "Only sanctioned path for scans, reads, AI recommendations, HITL tasks, and actions", font=F["small"], fill="#dbeafe")

    label_box(draw, (80, 535, 390, 735), "Request Envelope", [
        "• actor / role / tenant",
        "• source + connector",
        "• operation + scope",
        "• object references"
    ], WHITE, "#9bb6d9", number="A", accent=NAVY)
    label_box(draw, (430, 535, 740, 735), "Context Builder", [
        "• metadata inference",
        "• policy context",
        "• capability lookup",
        "• correlation ID"
    ], WHITE, "#9bb6d9", number="B", accent=NAVY)
    label_box(draw, (780, 535, 1090, 735), "Routing & State", [
        "• leases/checkpoints",
        "• retries/backoff",
        "• no external queue required",
        "• batch lifecycle"
    ], WHITE, "#9bb6d9", number="C", accent=NAVY)
    label_box(draw, (1130, 535, 1440, 735), "Guardian Client", [
        "• request decision",
        "• receive obligations",
        "• fail closed",
        "• no bypass"
    ], WHITE, "#9bb6d9", number="D", accent=NAVY)

    label_box(draw, (80, 780, 390, 1015), "Generic Connector Adapter", [
        "• Azure Blob first",
        "• file shares / DBs / SaaS",
        "• normalized object refs",
        "• capabilities + actions"
    ], LIGHT_ORANGE, "#f2c36b", number="1", accent=ORANGE)
    label_box(draw, (430, 780, 740, 1015), "Zero-Copy Accessor", [
        "• handle/reference",
        "• stream/range/cursor",
        "• bounded preview",
        "• ephemeral extraction"
    ], LIGHT_GREEN, "#8ed1aa", number="2", accent=GREEN)
    label_box(draw, (780, 780, 1090, 1015), "Discovery & Analysis", [
        "• discover + scan",
        "• classify + score",
        "• feature extraction",
        "• AI recommendations"
    ], LIGHT_BLUE, "#91b9ef", number="3", accent=BLUE)
    label_box(draw, (1130, 780, 1440, 1015), "Approved Execution", [
        "• auto-tag allowed items",
        "• create HITL tasks",
        "• enforce obligations",
        "• action receipts"
    ], LIGHT_RED, "#ee9a94", number="4", accent=RED)

    rounded(draw, (1480, 535, 1690, 1015), fill="#eef8f3", outline=GREEN, width=3, radius=20)
    draw.text((1510, 558), "DuckDB", font=F["section"], fill=GREEN)
    draw_wrapped(draw, (1510, 592), "Transient working memory", 150, F["body_bold"], TEXT)
    for i, item in enumerate(["Normalized metadata", "Feature vectors", "Risk scoring", "Batch aggregation", "Reset/expire per batch"]):
        draw.text((1510, 660 + i * 48), f"• {item}", font=F["small"], fill=MUTED)

    # flow inside cell
    arrow(draw, (390, 635), (430, 635), BLUE)
    arrow(draw, (740, 635), (780, 635), BLUE)
    arrow(draw, (1090, 635), (1130, 635), BLUE)
    arrow(draw, (235, 735), (235, 780), BLUE)
    arrow(draw, (390, 897), (430, 897), BLUE)
    arrow(draw, (740, 897), (780, 897), BLUE)
    arrow(draw, (1090, 897), (1130, 897), RED)
    arrow(draw, (1090, 897), (1480, 760), GREEN, dashed=True)
    arrow(draw, (1480, 790), (1090, 897), GREEN, dashed=True)

    # Guardian and outputs
    label_box(draw, (1810, 430, 2320, 760), "GUARDIAN AGENT - MANDATORY ENFORCEMENT POINT", [
        "• allow / deny / modify / HITL",
        "• policy version + hash required",
        "• obligations returned to Orchestrator",
        "• owns decision provenance"
    ], LIGHT_RED, RED, number="G", accent=RED)
    label_box(draw, (1810, 800, 2070, 1055), "Decision Provenance", [
        "• actor / request context",
        "• rules applied + reasoning",
        "• decision outcome",
        "• signature or hash"
    ], LIGHT_BLUE, "#91b9ef", number="P", accent=BLUE)
    label_box(draw, (2110, 800, 2320, 1055), "Elasticsearch", [
        "• batch summary",
        "• normalized signals",
        "• evidence references",
        "• indexing mode controls findings"
    ], LIGHT_PURPLE, "#b9a7ef", number="E", accent=PURPLE)

    arrow(draw, (1440, 635), (1810, 560), RED, width=5)
    arrow(draw, (1810, 650), (1440, 900), RED, width=5)
    arrow(draw, (1440, 900), (1810, 915), BLUE, dashed=True)
    arrow(draw, (2070, 930), (2110, 930), PURPLE)

    # Data/source lane
    rounded(draw, (40, 1160, 1730, 1430), fill="#f6fffa", outline=GREEN, width=3, radius=24)
    draw.text((70, 1185), "COLOCATED WITH DATA - ZERO-COPY BY DEFAULT", font=F["section"], fill=GREEN)
    label_box(draw, (80, 1235, 395, 1395), "Azure Blob Storage", ["first complete connector", "metadata/tags/tier/content refs"], WHITE, "#8ed1aa")
    label_box(draw, (430, 1235, 745, 1395), "Future Object Stores", ["S3, GCS, MinIO, other object APIs"], WHITE, "#8ed1aa")
    label_box(draw, (780, 1235, 1095, 1395), "File Shares", ["SMB, NFS, Windows file shares"], WHITE, "#8ed1aa")
    label_box(draw, (1130, 1235, 1445, 1395), "Databases / SaaS / APIs", ["structured, semi-structured, SaaS repositories"], WHITE, "#8ed1aa")
    label_box(draw, (1480, 1235, 1690, 1395), "Shared Contract", ["same envelopes, refs, receipts, evidence"], WHITE, "#8ed1aa")
    arrow(draw, (235, 1160), (235, 1015), GREEN, dashed=True)
    arrow(draw, (585, 1160), (585, 1015), GREEN, dashed=True)
    arrow(draw, (1485, 1160), (1585, 1015), GREEN, dashed=True)

    # Bottom invariants
    rounded(draw, (40, 1470, 2320, 1570), fill=NAVY, outline=NAVY, radius=18)
    invariants = [
        "Orchestrator is mandatory",
        "Guardian before mutation",
        "AI cannot mutate directly",
        "DuckDB is transient",
        "No persistent full duplicate copies",
        "Policy hash on every decision",
        "Azure Blob is one connector"
    ]
    x = 70
    for inv in invariants:
        draw.ellipse((x, 1499, x + 20, 1519), fill="#93c5fd")
        draw.line((x + 5, 1509, x + 9, 1514, x + 16, 1504), fill=NAVY, width=3)
        draw_wrapped(draw, (x + 28, 1496), inv, 270, F["small"], WHITE, 2, max_lines=2)
        x += 315
    return img


NODES = {
    "portal": (60, 165, 315, 275, "Portal / Intent"),
    "orch": (420, 95, 850, 345, "Data Agent Cell Orchestrator"),
    "conn": (60, 410, 315, 520, "Generic Connector"),
    "zero": (380, 410, 635, 520, "Zero-Copy Accessor"),
    "duck": (700, 410, 955, 520, "DuckDB Transient Memory"),
    "guardian": (1040, 140, 1335, 345, "Guardian Agent"),
    "action": (1040, 410, 1335, 520, "Approved Action / HITL"),
    "prov": (1390, 165, 1570, 275, "Decision Provenance"),
    "elastic": (1390, 410, 1570, 520, "Elasticsearch Summary"),
}


FRAMES = [
    ("Intent enters Orchestrator", "Portal creates a request envelope. Portal does not execute directly.", ["portal", "orch"]),
    ("Context and control state", "Orchestrator normalizes actor, source, connector, operation, scope, and policy context.", ["orch"]),
    ("Generic connector discovery", "Connector discovers source, object metadata, capabilities, and object references.", ["orch", "conn"]),
    ("Zero-copy feature extraction", "Accessor reads by handle, stream, range, cursor, or bounded preview. No persistent full copy.", ["conn", "zero"]),
    ("Transient scoring in DuckDB", "DuckDB holds normalized batch signals, features, scores, and aggregation state.", ["zero", "duck"]),
    ("Guardian decision", "Guardian returns allow, deny, modify, obligations, or HITL with policy version/hash.", ["orch", "guardian"]),
    ("Autonomous action or HITL", "Allowed actions execute automatically. HITL creates user tasks when required.", ["guardian", "action"]),
    ("Decision provenance", "Guardian records policy hash, rules, reasoning, obligations, outcome, actor, and timestamp.", ["guardian", "prov"]),
    ("Batch summary indexing", "Orchestrator sends aggregated summaries and normalized signals to Elasticsearch.", ["duck", "elastic", "prov"]),
    ("Evidence-ready outcome", "Evidence Center can show decision, policy hash, action receipt, HITL status, and audit trail.", ["portal", "prov", "elastic"]),
]


def draw_sequence_base(highlight: Iterable[str], title: str, subtitle: str, frame_no: int) -> Image.Image:
    img = Image.new("RGB", (1600, 900), WHITE)
    draw = ImageDraw.Draw(img)
    draw.rectangle((0, 0, 1600, 95), fill=NAVY)
    draw.text((34, 22), "EVEREST MVS1 AUTONOMOUS FLOW", font=font(34, True), fill=WHITE)
    draw.text((1040, 30), f"Frame {frame_no}/10", font=F["subtitle"], fill="#dbeafe")
    draw.text((34, 112), title, font=font(28, True), fill=TEXT)
    draw_wrapped(draw, (34, 152), subtitle, 900, F["body"], MUTED, 5, max_lines=2)

    hl = set(highlight)
    for key, (x1, y1, x2, y2, label) in NODES.items():
        active = key in hl
        fill = "#fff7e6" if active else "#f8fbff"
        outline = ORANGE if active else BORDER
        width = 5 if active else 2
        rounded(draw, (x1, y1, x2, y2), fill=fill, outline=outline, width=width, radius=18)
        draw_wrapped(draw, (x1 + 18, y1 + 22), label, x2 - x1 - 36, F["body_bold"], TEXT, 4, max_lines=2)

    # Connector arrows
    arrow(draw, (315, 220), (420, 220), BLUE if "portal" in hl or "orch" in hl else GRAY, 4)
    arrow(draw, (535, 345), (190, 410), BLUE if "conn" in hl else GRAY, 4)
    arrow(draw, (315, 465), (380, 465), BLUE if "zero" in hl else GRAY, 4)
    arrow(draw, (635, 465), (700, 465), GREEN if "duck" in hl else GRAY, 4)
    arrow(draw, (850, 220), (1040, 220), RED if "guardian" in hl else GRAY, 5)
    arrow(draw, (1188, 345), (1188, 410), RED if "action" in hl else GRAY, 4)
    arrow(draw, (1335, 220), (1390, 220), BLUE if "prov" in hl else GRAY, 4)
    arrow(draw, (1335, 465), (1390, 465), PURPLE if "elastic" in hl else GRAY, 4)
    arrow(draw, (955, 465), (1390, 465), PURPLE if "elastic" in hl else GRAY, 3, dashed=True)

    # Detail panel
    rounded(draw, (60, 610, 1540, 835), fill=LIGHT_GRAY, outline="#d4deef", width=2, radius=18)
    draw.text((90, 635), "What this proves", font=F["section"], fill=NAVY)
    bullets = [
        "Portal remains an intent surface; the Orchestrator owns runtime control.",
        "Read/discovery/analysis can occur before Guardian, but mutation cannot.",
        "Guardian decision and provenance are mandatory for autonomous trust.",
        "Azure Blob is one connector behind a reusable multi-store contract.",
    ]
    x = 90
    y = 680
    for b in bullets:
        draw.text((x, y + 2), "•", font=F["body_bold"], fill=GREEN)
        y = draw_wrapped(draw, (x + 24, y), b, 665, F["body"], TEXT, 5, max_lines=2)
        if y > 790:
            x = 835
            y = 680

    draw.rectangle((0, 865, 1600, 900), fill=NAVY)
    draw.text((34, 872), "Invariant: no mutation proceeds without Orchestrator + Guardian allow/modify decision.", font=F["small"], fill=WHITE)
    return img


def create_gif_frames() -> list[Image.Image]:
    frames: list[Image.Image] = []
    for idx, (title, subtitle, highlight) in enumerate(FRAMES, start=1):
        frames.append(draw_sequence_base(highlight, title, subtitle, idx))
    return frames


def main() -> None:
    OUT_DIR.mkdir(parents=True, exist_ok=True)
    summary = create_summary_png()
    summary.save(PNG_OUT)
    frames = create_gif_frames()
    frames[0].save(
        GIF_OUT,
        save_all=True,
        append_images=frames[1:],
        duration=1450,
        loop=0,
        optimize=False,
    )
    print(f"created {PNG_OUT}")
    print(f"created {GIF_OUT}")


if __name__ == "__main__":
    main()
