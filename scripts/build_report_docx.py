#!/usr/bin/env python3
"""Build docs/实验报告-绿行积分.docx from the Markdown working source.

The Markdown file is the single source of truth.  This script keeps the
conversion deliberately small and explicit: it understands only the subset of
Markdown used by the report (headings, paragraphs, pipe tables, images, code
fences, blockquotes and page-break comments).  It does not try to be a general
Markdown renderer.

Usage:
    python scripts/build_report_docx.py
    python scripts/build_report_docx.py --md docs/实验报告-绿行积分.md \
        --out docs/实验报告-绿行积分.docx
"""

from __future__ import annotations

import argparse
import os
import re
import sys
from pathlib import Path

from docx import Document
from docx.enum.section import WD_SECTION
from docx.enum.table import WD_CELL_VERTICAL_ALIGNMENT, WD_TABLE_ALIGNMENT
from docx.enum.text import WD_ALIGN_PARAGRAPH, WD_BREAK
from docx.oxml import OxmlElement
from docx.oxml.ns import qn
from docx.shared import Cm, Pt, RGBColor
from PIL import Image

ROOT = Path(__file__).resolve().parents[1]
DEFAULT_MD = ROOT / "docs" / "实验报告-绿行积分.md"
DEFAULT_OUT = ROOT / "docs" / "实验报告-绿行积分.docx"

BODY_FONT = "宋体"
HEAD_FONT = "黑体"
CODE_FONT = "Consolas"
INK = RGBColor(0x1A, 0x1A, 0x1A)
GRAY = RGBColor(0x59, 0x59, 0x59)
ACCENT = RGBColor(0x17, 0x7A, 0x4D)

IMAGE_RE = re.compile(r"!\[([^\]]*)\]\((?:<([^>]+)>|([^)]+))\)")
INLINE_RE = re.compile(r"(\*\*[^*]+\*\*|`[^`]+`)")
LINK_RE = re.compile(r"\[([^\]]+)\]\(([^)]+)\)")


# ---------------------------------------------------------------------------
# low-level helpers


def set_run_font(run, name: str, size: float | None = None,
                 bold: bool | None = None, color: RGBColor | None = None) -> None:
    run.font.name = name
    run._element.rPr.rFonts.set(qn("w:eastAsia"), name)
    if size is not None:
        run.font.size = Pt(size)
    if bold is not None:
        run.font.bold = bold
    if color is not None:
        run.font.color.rgb = color


def shade(element, fill: str) -> None:
    pPr = element.get_or_add_pPr() if hasattr(element, "get_or_add_pPr") else element
    shd = OxmlElement("w:shd")
    shd.set(qn("w:val"), "clear")
    shd.set(qn("w:color"), "auto")
    shd.set(qn("w:fill"), fill)
    pPr.append(shd)


def set_cell_margins(table, top=30, start=45, bottom=30, end=45) -> None:
    tblPr = table._tbl.tblPr
    mar = OxmlElement("w:tblCellMar")
    for side, value in (("top", top), ("start", start),
                        ("bottom", bottom), ("end", end)):
        el = OxmlElement(f"w:{side}")
        el.set(qn("w:w"), str(value))
        el.set(qn("w:type"), "dxa")
        mar.append(el)
    tblPr.append(mar)


def set_repeat_table_header(row) -> None:
    trPr = row._tr.get_or_add_trPr()
    header = OxmlElement("w:tblHeader")
    header.set(qn("w:val"), "true")
    trPr.append(header)


def prevent_row_split(row) -> None:
    trPr = row._tr.get_or_add_trPr()
    cant = OxmlElement("w:cantSplit")
    cant.set(qn("w:val"), "true")
    trPr.append(cant)


def add_field(paragraph, instr: str, placeholder: str = "") -> None:
    run = paragraph.add_run()
    begin = OxmlElement("w:fldChar")
    begin.set(qn("w:fldCharType"), "begin")
    run._r.append(begin)

    instr_el = OxmlElement("w:instrText")
    instr_el.set(qn("xml:space"), "preserve")
    instr_el.text = instr
    run._r.append(instr_el)

    separate = OxmlElement("w:fldChar")
    separate.set(qn("w:fldCharType"), "separate")
    run._r.append(separate)

    if placeholder:
        run.add_text(placeholder)

    end = OxmlElement("w:fldChar")
    end.set(qn("w:fldCharType"), "end")
    run._r.append(end)


def enable_update_fields(doc: Document) -> None:
    settings = doc.settings.element
    update = OxmlElement("w:updateFields")
    update.set(qn("w:val"), "true")
    settings.append(update)


def add_page_number_footer(section) -> None:
    section.different_first_page_header_footer = True
    footer = section.footer
    footer.is_linked_to_previous = False
    p = footer.paragraphs[0]
    p.alignment = WD_ALIGN_PARAGRAPH.CENTER
    run = p.add_run("第 ")
    set_run_font(run, BODY_FONT, 9, color=GRAY)
    add_field(p, " PAGE ", "1")
    run = p.add_run(" 页")
    set_run_font(run, BODY_FONT, 9, color=GRAY)


# ---------------------------------------------------------------------------
# inline text


def clean_links(text: str) -> str:
    def repl(match: re.Match) -> str:
        label, url = match.group(1), match.group(2)
        if url.startswith(("http://", "https://")):
            return f"{label}（{url}）"
        return label

    return LINK_RE.sub(repl, text)


def add_inline(paragraph, text: str, size: float = 10.5,
               color: RGBColor = INK) -> None:
    text = clean_links(text.replace("<br>", "\n"))
    for token in INLINE_RE.split(text):
        if not token:
            continue
        if token.startswith("**") and token.endswith("**") and len(token) > 4:
            run = paragraph.add_run(token[2:-2])
            set_run_font(run, BODY_FONT, size, bold=True, color=color)
        elif token.startswith("`") and token.endswith("`") and len(token) > 2:
            run = paragraph.add_run(token[1:-1])
            set_run_font(run, CODE_FONT, size - 0.5, color=color)
        else:
            run = paragraph.add_run(token)
            set_run_font(run, BODY_FONT, size, color=color)


# ---------------------------------------------------------------------------
# blocks


def add_cover(doc: Document, lines: list[str]) -> None:
    fields: list[tuple[str, str]] = []
    for line in lines:
        if "：" in line:
            key, value = line.split("：", 1)
            fields.append((key.strip(), value.strip()))

    values = dict(fields)
    title = values.get("实验题目", "碳中和城市绿色出行积分系统")

    for _ in range(3):
        doc.add_paragraph()

    p = doc.add_paragraph()
    p.alignment = WD_ALIGN_PARAGRAPH.CENTER
    run = p.add_run(values.get("学校", "河北工业大学"))
    set_run_font(run, HEAD_FONT, 22, bold=True)

    p = doc.add_paragraph()
    p.alignment = WD_ALIGN_PARAGRAPH.CENTER
    p.paragraph_format.space_before = Pt(4)
    run = p.add_run("软件工程实验")
    set_run_font(run, HEAD_FONT, 16, color=GRAY)

    p = doc.add_paragraph()
    p.alignment = WD_ALIGN_PARAGRAPH.CENTER
    p.paragraph_format.space_before = Pt(14)
    run = p.add_run("实 验 报 告")
    set_run_font(run, HEAD_FONT, 32, bold=True)

    p = doc.add_paragraph()
    p.alignment = WD_ALIGN_PARAGRAPH.CENTER
    p.paragraph_format.space_before = Pt(18)
    p.paragraph_format.space_after = Pt(26)
    run = p.add_run(title)
    set_run_font(run, HEAD_FONT, 20, bold=True, color=ACCENT)

    for key, value in fields:
        if key in {"学校", "实验题目"}:
            continue
        p = doc.add_paragraph()
        p.paragraph_format.left_indent = Cm(3.4)
        p.paragraph_format.space_after = Pt(4)
        label = p.add_run(f"{key}：")
        set_run_font(label, BODY_FONT, 11, bold=True)
        body = p.add_run(value)
        set_run_font(body, BODY_FONT, 11 if key != "小组成员" else 10.5)

    doc.add_page_break()


def add_toc(doc: Document) -> None:
    p = doc.add_paragraph()
    p.alignment = WD_ALIGN_PARAGRAPH.CENTER
    p.paragraph_format.space_after = Pt(12)
    run = p.add_run("目　录")
    set_run_font(run, HEAD_FONT, 18, bold=True)

    p = doc.add_paragraph()
    p.paragraph_format.space_after = Pt(6)
    add_field(p, 'TOC \\o "1-3" \\h \\z \\u', "目录将在打开 Word 后自动更新")

    p = doc.add_paragraph()
    p.alignment = WD_ALIGN_PARAGRAPH.CENTER
    run = p.add_run("（打开后按 Ctrl+A、F9 更新域，生成带页码的目录）")
    set_run_font(run, BODY_FONT, 9, color=GRAY)
    doc.add_page_break()


def add_heading(doc: Document, text: str, level: int) -> None:
    p = doc.add_paragraph(style=f"Heading {level}")
    p.paragraph_format.keep_with_next = True
    p.paragraph_format.space_before = Pt({1: 16, 2: 12, 3: 9, 4: 7}[level])
    p.paragraph_format.space_after = Pt({1: 8, 2: 6, 3: 4, 4: 3}[level])
    run = p.add_run(text)
    set_run_font(run, HEAD_FONT, {1: 17, 2: 14, 3: 12, 4: 11}[level],
                 bold=True, color=RGBColor(0, 0, 0) if level > 1 else ACCENT)
    if level == 1:
        p.paragraph_format.page_break_before = False


def add_paragraph(doc: Document, text: str) -> None:
    p = doc.add_paragraph()
    p.paragraph_format.space_after = Pt(4)
    p.paragraph_format.line_spacing = 1.35
    add_inline(p, text)


def add_quote(doc: Document, text: str) -> None:
    p = doc.add_paragraph()
    p.paragraph_format.left_indent = Cm(0.6)
    p.paragraph_format.space_before = Pt(2)
    p.paragraph_format.space_after = Pt(5)
    p.paragraph_format.line_spacing = 1.25
    add_inline(p, text, size=9.5, color=GRAY)


def add_code(doc: Document, lines: list[str]) -> None:
    p = doc.add_paragraph()
    p.paragraph_format.left_indent = Cm(0.4)
    p.paragraph_format.space_before = Pt(4)
    p.paragraph_format.space_after = Pt(6)
    shade(p._p, "F2F2F2")
    for index, line in enumerate(lines):
        if index:
            p.add_run().add_break()
        run = p.add_run(line)
        set_run_font(run, CODE_FONT, 8.5, color=INK)


def image_size(path: Path) -> tuple[float, float]:
    with Image.open(path) as img:
        width, height = img.size
    max_w, max_h = 16.8, 22.0
    ratio = min(max_w / width, max_h / height, 1.0)
    return width * ratio, height * ratio


def add_image(doc: Document, alt: str, raw_path: str) -> bool:
    path = Path(raw_path)
    if not path.is_absolute():
        path = ROOT / path
    if not path.exists():
        print(f"WARN missing image: {raw_path}", file=sys.stderr)
        return False
    try:
        width_cm, height_cm = image_size(path)
        p = doc.add_paragraph()
        p.alignment = WD_ALIGN_PARAGRAPH.CENTER
        p.paragraph_format.space_before = Pt(3)
        p.paragraph_format.space_after = Pt(6)
        run = p.add_run()
        run.add_picture(str(path), width=Cm(width_cm), height=Cm(height_cm))
        if alt and not alt.startswith("图"):
            cap = doc.add_paragraph()
            cap.alignment = WD_ALIGN_PARAGRAPH.CENTER
            run = cap.add_run(alt)
            set_run_font(run, BODY_FONT, 9, color=GRAY)
        return True
    except Exception as exc:  # noqa: BLE001 - report and continue
        print(f"WARN image failed {raw_path}: {exc}", file=sys.stderr)
        return False


def split_table_row(line: str) -> list[str]:
    line = line.strip()
    if line.startswith("|"):
        line = line[1:]
    if line.endswith("|"):
        line = line[:-1]
    cells = re.split(r"(?<!\\)\|", line)
    return [cell.replace("\\|", "|").strip() for cell in cells]


def is_separator(line: str) -> bool:
    cells = split_table_row(line)
    return bool(cells) and all(re.fullmatch(r":?-{3,}:?", c.strip()) for c in cells)


def add_table(doc: Document, rows: list[list[str]]) -> None:
    if not rows:
        return
    cols = max(len(row) for row in rows)
    normalized = [row + [""] * (cols - len(row)) for row in rows]
    table = doc.add_table(rows=len(normalized), cols=cols)
    table.style = "Table Grid"
    table.alignment = WD_TABLE_ALIGNMENT.CENTER
    table.autofit = False
    table.allow_autofit = False
    set_cell_margins(table)

    font_size = 8.5 if cols <= 5 else 8.0 if cols <= 8 else 7.5 if cols == 9 else 7.0

    # Weight columns by their longest cell, but keep every column wide enough.
    weights: list[float] = []
    for c in range(cols):
        longest = max(len(normalized[r][c]) for r in range(len(normalized)))
        weights.append(max(4.0, min(longest, 40.0)) ** 0.75)
    total = sum(weights)
    available = 18.0
    widths = [max(1.0, available * w / total) for w in weights]
    scale = available / sum(widths)
    widths = [w * scale for w in widths]

    for r, row in enumerate(normalized):
        table_row = table.rows[r]
        prevent_row_split(table_row)
        if r == 0:
            set_repeat_table_header(table_row)
        for c, value in enumerate(row):
            cell = table_row.cells[c]
            cell.width = Cm(widths[c])
            cell.vertical_alignment = (
                WD_CELL_VERTICAL_ALIGNMENT.CENTER if r == 0
                else WD_CELL_VERTICAL_ALIGNMENT.TOP
            )
            p = cell.paragraphs[0]
            p.paragraph_format.space_before = Pt(0)
            p.paragraph_format.space_after = Pt(0)
            p.paragraph_format.line_spacing = 1.0
            if r == 0:
                p.alignment = WD_ALIGN_PARAGRAPH.CENTER
                for token in INLINE_RE.split(value):
                    if not token:
                        continue
                    run = p.add_run(token.strip("*`"))
                    set_run_font(run, BODY_FONT, font_size, bold=True)
                shade(cell._tc.get_or_add_tcPr(), "D9E2F3")
            else:
                add_inline(p, value, size=font_size)

    for c in range(cols):
        table.columns[c].width = Cm(widths[c])

    # A small gap after the table; the following caption/paragraph supplies
    # its own space before.
    spacer = doc.add_paragraph()
    spacer.paragraph_format.space_after = Pt(2)
    spacer.paragraph_format.line_spacing = 1.0


# ---------------------------------------------------------------------------
# main parse loop


def build(md_path: Path, out_path: Path) -> tuple[int, int, int]:
    text = md_path.read_text(encoding="utf-8")
    lines = text.splitlines()

    doc = Document()
    section = doc.sections[0]
    section.page_width = Cm(21.0)
    section.page_height = Cm(29.7)
    section.top_margin = Cm(2.0)
    section.bottom_margin = Cm(2.0)
    section.left_margin = Cm(1.4)
    section.right_margin = Cm(1.4)
    add_page_number_footer(section)

    normal = doc.styles["Normal"]
    normal.font.name = BODY_FONT
    normal.font.size = Pt(10.5)
    normal._element.rPr.rFonts.set(qn("w:eastAsia"), BODY_FONT)
    normal.paragraph_format.space_after = Pt(4)

    for level in (1, 2, 3, 4):
        style = doc.styles[f"Heading {level}"]
        style.font.name = HEAD_FONT
        style._element.rPr.rFonts.set(qn("w:eastAsia"), HEAD_FONT)
        style.font.color.rgb = RGBColor(0, 0, 0)
        style.font.bold = True

    enable_update_fields(doc)

    # Cover block
    cover_lines: list[str] = []
    i = 0
    while i < len(lines) and lines[i].strip() != "<!-- /COVER -->":
        if lines[i].strip() not in {"<!-- COVER -->"} and lines[i].strip():
            cover_lines.append(lines[i].strip())
        i += 1
    i += 1
    add_cover(doc, cover_lines)

    toc_done = False
    image_count = 0
    table_count = 0

    while i < len(lines):
        line = lines[i]
        stripped = line.strip()

        if stripped in {"<!-- /COVER -->", "<!-- BODY -->", "<!-- COVER -->"}:
            i += 1
            continue
        if stripped == "<!-- TOC -->":
            add_toc(doc)
            toc_done = True
            i += 1
            continue
        if stripped in {"<!-- pagebreak -->", "<!-- 分页 -->"}:
            doc.add_page_break()
            i += 1
            continue
        if stripped.startswith("<!--") and stripped.endswith("-->"):
            i += 1
            continue
        if not stripped:
            i += 1
            continue

        if stripped.startswith("```"):
            i += 1
            code_lines: list[str] = []
            while i < len(lines) and not lines[i].strip().startswith("```"):
                code_lines.append(lines[i])
                i += 1
            i += 1
            add_code(doc, code_lines)
            continue

        heading = re.match(r"^(#{1,6})\s+(.*)$", stripped)
        if heading:
            level = min(len(heading.group(1)), 4)
            add_heading(doc, heading.group(2).strip(), level)
            i += 1
            continue

        image = IMAGE_RE.fullmatch(stripped)
        if image:
            alt = image.group(1).strip()
            raw = (image.group(2) or image.group(3) or "").strip()
            if add_image(doc, alt, raw):
                image_count += 1
            i += 1
            continue

        if stripped.startswith("|") and stripped.endswith("|"):
            table_lines: list[str] = []
            while i < len(lines) and lines[i].strip().startswith("|") \
                    and lines[i].strip().endswith("|"):
                table_lines.append(lines[i].strip())
                i += 1
            rows = [split_table_row(t) for t in table_lines if not is_separator(t)]
            rows = [r for r in rows if any(c for c in r)]
            if rows:
                add_table(doc, rows)
                table_count += 1
            continue

        if stripped.startswith(">"):
            quote_lines: list[str] = []
            while i < len(lines) and lines[i].strip().startswith(">"):
                quote_lines.append(lines[i].strip()[1:].strip())
                i += 1
            for q in quote_lines:
                if q:
                    add_quote(doc, q)
            continue

        # Figure/table captions: centered, bold, keep with next block.
        if stripped.startswith("**图") or stripped.startswith("**表"):
            p = doc.add_paragraph()
            p.alignment = WD_ALIGN_PARAGRAPH.CENTER
            p.paragraph_format.space_before = Pt(7)
            p.paragraph_format.space_after = Pt(3)
            p.paragraph_format.keep_with_next = True
            add_inline(p, stripped, size=9.5)
            i += 1
            continue

        add_paragraph(doc, stripped)
        i += 1

    if not toc_done:
        print("WARN no TOC marker found", file=sys.stderr)

    doc.core_properties.title = "碳中和城市绿色出行积分系统 实验报告"
    doc.core_properties.author = "李响（P1）"
    doc.core_properties.subject = "《软件工程实验》2026 版第 14 题"
    doc.core_properties.comments = "由 docs/实验报告-绿行积分.md 生成"

    out_path.parent.mkdir(parents=True, exist_ok=True)
    doc.save(str(out_path))
    return len(doc.paragraphs), len(doc.tables), image_count


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--md", type=Path, default=DEFAULT_MD)
    parser.add_argument("--out", type=Path, default=DEFAULT_OUT)
    args = parser.parse_args()

    paragraphs, tables, images = build(args.md, args.out)
    size = args.out.stat().st_size
    print(f"WROTE {args.out}")
    print(f"paragraphs={paragraphs} tables={tables} images={images} bytes={size}")


if __name__ == "__main__":
    main()
