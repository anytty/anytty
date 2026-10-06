"""Content/interaction/form/chart widget tests (port of the Go widget tests).

Assertions inspect the produced box trees (plain dicts) and pure helper
values; nothing here renders or talks to a host. ``build()`` returns a
composable ``builder.Node``, so a widget's committed box is ``build().build()``
(the Go tests' ``Build().Build()``).
"""

import math
import unittest
import unicodedata
from datetime import datetime, timedelta

from tui2sdk.widgets import chart, contextmenu, datepicker
from tui2sdk.widgets import format as fmt
from tui2sdk.widgets import form, hover, input as input_mod
from tui2sdk.widgets import list as list_mod
from tui2sdk.widgets import modal, mouse, richtext, scrollbar
from tui2sdk.widgets import select as select_mod
from tui2sdk.widgets import table, validate


def box_text(box):
    if not box:
        return ""
    return (box.get("content") or {}).get("text", "")


def row_text(box):
    if not box:
        return ""
    text = box_text(box)
    for child in box.get("children") or []:
        text += row_text(child)
    return text


def find_box(root, id):
    if not root:
        return None
    if root.get("id") == id:
        return root
    for child in root.get("children") or []:
        found = find_box(child, id)
        if found is not None:
            return found
    return None


def children(box):
    return box.get("children") or []


def box_style(box):
    return box.get("style", "")


def box_size(box):
    return box.get("size") or (0, 0, 0)


def display_width(text):
    total = 0
    for ch in text:
        if unicodedata.combining(ch):
            continue
        total += 2 if unicodedata.east_asian_width(ch) in ("W", "F") else 1
    return total


class TestRichText(unittest.TestCase):
    def test_build_merges_and_styles(self):
        root = richtext.line(
            richtext.styled("a", "x"),
            richtext.styled("b", "x"),
            richtext.styled("", "z"),
            richtext.styled("c", "y"),
        ).build().build()
        kids = children(root)
        self.assertEqual(len(kids), 2)
        self.assertEqual(box_text(kids[0]), "ab")
        self.assertEqual(box_style(kids[0]), "x")
        self.assertEqual(box_text(kids[1]), "c")
        self.assertEqual(box_style(kids[1]), "y")

    def test_width_pads_and_text_measures(self):
        rich = richtext.RichText([richtext.styled("ab", "x")], width=5)
        root = rich.build().build()
        self.assertEqual(row_text(root), "ab   ")
        self.assertEqual(rich.text(), "ab")

    def test_empty(self):
        root = richtext.line().build().build()
        self.assertEqual(children(root), [])
        self.assertEqual(richtext.line().text(), "")
        styled = richtext.line(richtext.styled("hi", "a"),
                               richtext.styled("!", "b")).styled("z")
        self.assertEqual(styled.text(), "hi!")
        self.assertEqual(len(styled.spans), 1)
        self.assertEqual(styled.spans[0].style, "z")

    def test_wrap_text_plain(self):
        cases = [
            ("simple", "hello world", 5, ["hello", "world"]),
            ("spaces", "a b c d", 3, ["a b", "c d"]),
            ("hardbreak", "abcd ef", 3, ["abc", "d", "ef"]),
            ("newline", "ab\ncd", 5, ["ab", "cd"]),
            ("blankline", "ab\n\ncd", 5, ["ab", "", "cd"]),
            ("exact", "abcde", 5, ["abcde"]),
            ("one", "a", 1, ["a"]),
            ("empty", "", 5, [""]),
            ("leading spaces", "   x", 5, ["x"]),
        ]
        for name, text, width, want in cases:
            self.assertEqual(richtext.wrap_text(text, width), want, name)
        self.assertIsNone(richtext.wrap_text("abc", 0))
        self.assertIsNone(richtext.wrap_text("abc", -1))

    def test_wrap_text_wide_runes_never_split(self):
        self.assertEqual(richtext.wrap_text("\u4f60\u597d", 3),
                         ["\u4f60", "\u597d"])
        self.assertEqual(richtext.wrap_text("\u4f60\u597d", 4),
                         ["\u4f60\u597d"])
        self.assertEqual(richtext.wrap_text("a\u4f60b", 2),
                         ["a", "\u4f60", "b"])
        for line in richtext.wrap_text("\u4f60\u597d\u4e16\u754c", 3):
            self.assertLessEqual(display_width(line), 3)

    def test_wrap_spans_preserves_styles(self):
        spans = [richtext.Span("ab", "x"), richtext.Span("cd", "y")]
        got = richtext.wrap_spans(spans, 2)
        self.assertEqual(len(got), 2)
        self.assertEqual(got[0].text(), "ab")
        self.assertEqual(got[0].spans[0].style, "x")
        self.assertEqual(got[1].text(), "cd")
        self.assertEqual(got[1].spans[0].style, "y")
        for row in got:
            self.assertEqual(row.width, 2)
        self.assertIsNone(richtext.wrap_spans(spans, 0))

    def test_wrap_spans_splits_one_run_across_lines(self):
        got = richtext.wrap_spans([richtext.Span("abcd", "x")], 3)
        self.assertEqual(len(got), 2)
        self.assertEqual(got[0].text(), "abc")
        self.assertEqual(got[1].text(), "d")
        for row in got:
            self.assertEqual(len(row.spans), 1)
            self.assertEqual(row.spans[0].style, "x")

    def test_rich_text_wrap_spans_marks_wrapped(self):
        rich = richtext.RichText([richtext.styled("abcdef", "x")])
        rows = rich.wrap_spans(3)
        self.assertEqual(len(rows), 2)
        self.assertTrue(rows[0].wrap)
        self.assertTrue(rows[1].wrap)

    def test_parse_emphasis(self):
        got = richtext.parse_emphasis("a **b** c")
        want = [richtext.Span("a ", ""), richtext.Span("b", "bold"),
                richtext.Span(" c", "")]
        self.assertEqual(got, want)
        self.assertEqual(richtext.parse_emphasis("_em_"),
                         [richtext.Span("em", "italic")])
        mixed = richtext.parse_emphasis("**bold** and _em_")
        want = [richtext.Span("bold", "bold"), richtext.Span(" and ", ""),
                richtext.Span("em", "italic")]
        self.assertEqual(mixed, want)

    def test_parse_emphasis_unclosed_and_empty_stay_literal(self):
        cases = {
            "a **b": "a **b",
            "_open": "_open",
            "****": "****",
            "__": "__",
            "a ** b": "a ** b",
            "a _b c": "a _b c",
        }
        for text, want in cases.items():
            got = richtext.parse_emphasis(text)
            self.assertEqual(len(got), 1)
            self.assertEqual(got[0].text, want)
            self.assertEqual(got[0].style, "")

    def test_parse_emphasis_base_style_composes(self):
        got = richtext.parse_emphasis_styled("**x** _y_ z", "muted")
        want = [richtext.Span("x", "muted;bold"), richtext.Span(" ", "muted"),
                richtext.Span("y", "muted;italic"), richtext.Span(" z", "muted")]
        self.assertEqual(got, want)

    def test_with_italic_idempotent(self):
        self.assertEqual(richtext.with_italic(""), "italic")
        self.assertEqual(richtext.with_italic("muted"), "muted;italic")
        self.assertEqual(richtext.with_italic("a;italic;b"), "a;italic;b")


class TestList(unittest.TestCase):
    def test_renders_only_visible_rows(self):
        items = ["item-%d" % i for i in range(100000)]
        lst = list_mod.List(items=items, height=10, width=20, selected=50000,
                            follow=True)
        start, end = lst.visible_range()
        self.assertEqual(end - start, 10)
        root = lst.build().build()
        self.assertEqual(len(children(root)), 10)
        self.assertIn("item-49991", box_text(children(root)[0]))
        last = box_text(children(root)[9])
        self.assertIn("item-50000", last)
        self.assertTrue(last.startswith(list_mod.DEFAULT_LIST_MARKER))

    def test_helpers(self):
        lst = list_mod.List(items=["a", "b", "c", "d", "e"], height=2)
        lst.move(1)
        self.assertEqual(lst.selected, 1)
        lst.move(99)
        self.assertEqual(lst.selected, 4)
        lst.page_up()
        self.assertEqual(lst.selected, 2)
        lst.page_down()
        self.assertEqual(lst.selected, 4)
        lst.top()
        self.assertEqual((lst.selected, lst.offset), (0, 0))
        lst.bottom()
        self.assertEqual((lst.selected, lst.offset), (4, 3))
        lst.offset = 100
        lst.selected = 100
        lst.ensure_visible()
        self.assertEqual((lst.selected, lst.offset), (4, 3))

    def test_empty_and_height_one(self):
        empty = list_mod.List()
        self.assertEqual(empty.visible_range(), (0, 0))
        empty.move(1)
        empty.page_up()
        empty.page_down()
        empty.top()
        empty.bottom()
        self.assertEqual(empty.selected, 0)
        self.assertEqual(len(children(empty.build().build())), 0)
        placeholder = list_mod.List(empty="no items", height=3,
                                    width=10).build().build()
        self.assertEqual(len(children(placeholder)), 1)
        self.assertEqual(box_text(children(placeholder)[0]), "no items  ")
        one = list_mod.List(items=["only"], height=1, width=8).build().build()
        self.assertEqual(len(children(one)), 1)
        self.assertEqual(display_width(box_text(children(one)[0])), 8)

    def test_header_footer_window(self):
        lst = list_mod.List(items=["a", "b", "c", "d"], height=4,
                            header="HDR", footer="FTR", header_style="h",
                            footer_style="f")
        self.assertEqual(lst.visible_range(), (0, 2))
        kids = children(lst.build().build())
        self.assertEqual(len(kids), 4)
        self.assertEqual(box_text(kids[0]), "HDR")
        self.assertEqual(box_style(kids[0]), "h")
        self.assertEqual(box_text(kids[3]), "FTR")
        self.assertEqual(box_style(kids[3]), "f")

    def test_follow_false_keeps_offset(self):
        lst = list_mod.List(items=["a", "b", "c", "d", "e"], height=2,
                            selected=4)
        self.assertEqual(lst.visible_range()[1], 2)
        lst.follow = True
        self.assertEqual(lst.visible_range()[0], 3)

    def test_row_id_style_and_cjk_truncation(self):
        lst = list_mod.List(
            items=["\u4e2d\u6587\u4e2d\u6587", "short"],
            height=2, width=6, selected=1, style="base",
            selected_style="sel",
            row_id=lambda index: "row:%d" % index,
        )
        kids = children(lst.build().build())
        cell = box_text(kids[0])
        self.assertEqual(display_width(cell), 6)
        self.assertIn("\u4e2d\u6587", cell)
        self.assertEqual(kids[0].get("id"), "row:0")
        self.assertEqual(kids[0].get("input"), ["mouse"])
        self.assertTrue(box_text(kids[1]).startswith(list_mod.DEFAULT_LIST_MARKER))
        self.assertEqual(box_style(kids[1]), "sel")

    def test_virtual_list_rows_and_disabled(self):
        lst = list_mod.VirtualList(
            id="vl",
            rows=[
                list_mod.ListRow(text="one", id="vl:0", style="s0"),
                list_mod.ListRow(text="two", id="vl:1", disabled=True),
                list_mod.ListRow(text="three", id="vl:2"),
            ],
            height=2, width=8, offset=1, selected=2, follow=True,
            disabled_style="off", selected_style="sel",
        )
        self.assertEqual(lst.visible_range(), (1, 3))
        kids = children(lst.build().build())
        self.assertEqual(len(kids), 2)
        self.assertEqual(box_style(kids[0]), "off")
        self.assertEqual(kids[0].get("id"), "vl:1")
        self.assertEqual(box_style(kids[1]), "sel")
        self.assertTrue(box_text(kids[1]).startswith(list_mod.DEFAULT_LIST_MARKER))
        empty = list_mod.VirtualList()
        empty.move(3)
        empty.bottom()
        self.assertEqual(empty.selected, 0)
        self.assertEqual(empty.build().build().get("id"), None)


class TestTable(unittest.TestCase):
    def test_column_widths(self):
        tbl = table.Table(
            columns=[table.Column(title="Name"),
                     table.Column(title="N", min_width=4)],
            rows=[["alpha", "1"], ["b", "12345"]],
        )
        self.assertEqual(tbl.column_widths(), [5, 5])
        tbl.columns[0].width = 3
        self.assertEqual(tbl.column_widths(), [3, 5])

    def test_flex_and_shrink(self):
        flex = table.Table(width=20,
                           columns=[table.Column(title="A"),
                                    table.Column(title="B", flex=1)],
                           rows=[["x", "y"]])
        widths = flex.column_widths()
        self.assertEqual(widths[0] + widths[1] + flex._sep_width(), 20)
        self.assertGreater(widths[1], widths[0])
        shrink = table.Table(width=6,
                             columns=[table.Column(title="A"),
                                      table.Column(title="B")],
                             rows=[["aaaa", "bbbb"]])
        widths = shrink.column_widths()
        self.assertEqual(widths[0] + widths[1] + shrink._sep_width(), 6)
        self.assertEqual(widths, [4, 1])

    def test_build_header_selection_zebra_footer(self):
        tbl = table.Table(
            id="tbl",
            columns=[table.Column(title="Name", width=4),
                     table.Column(title="V", width=3,
                                  align=table.ALIGN_RIGHT)],
            rows=[["ab", "7"], ["c", "12"], ["dd", "3"]],
            selected=2, zebra=True, zebra_style="zebra",
            selected_style="sel", header_style="hdr", footer_style="ftr",
            rule=True, footer=["F", "9"],
            row_id=lambda index: "row:%d" % index,
        )
        kids = children(tbl.build().build())
        self.assertEqual(len(kids), 6)
        header = children(kids[0])
        self.assertEqual(len(header), 3)
        self.assertEqual(box_text(header[0]), "Name")
        self.assertEqual(box_text(header[2]), "  V")
        self.assertEqual(box_style(header[0]), "hdr")
        self.assertEqual(box_style(kids[1]), "hdr")
        self.assertEqual(box_style(children(kids[2])[0]), "")
        self.assertEqual(box_style(children(kids[3])[0]), "zebra")
        self.assertEqual(kids[4].get("id"), "row:2")
        self.assertEqual(box_style(children(kids[4])[0]), "sel")
        self.assertEqual(kids[4].get("input"), ["mouse"])
        footer = children(kids[5])
        self.assertEqual(box_text(footer[0]), "F   ")
        self.assertEqual(box_style(footer[0]), "ftr")

    def test_cell_truncation_and_alignment(self):
        tbl = table.Table(
            hide_header=True,
            columns=[table.Column(title="L", width=4),
                     table.Column(title="R", width=4,
                                  align=table.ALIGN_RIGHT),
                     table.Column(title="C", width=4,
                                  align=table.ALIGN_CENTER)],
            rows=[["\u4e2d\u6587\u4f60\u597d", "ab", "ab"]],
        )
        row = children(children(tbl.build().build())[0])
        self.assertEqual(box_text(row[0]), "\u4e2d\u6587")
        self.assertEqual(box_text(row[2]), "  ab")
        self.assertEqual(box_text(row[4]), " ab ")

    def test_records_format(self):
        tbl = table.Table(
            columns=[table.Column(title="N",
                                  format=lambda value: "<%s>" % value),
                     table.Column(title="D")],
            records=[[42, "x"], ["\u4e03", 7]],
            rows=[["ignored", "ignored"]],
        )
        self.assertEqual(tbl.row_count(), 2)
        self.assertEqual(tbl.cell(0, 0), "<42>")
        self.assertEqual(tbl.cell(1, 0), "<\u4e03>")
        self.assertEqual(tbl.cell(0, 1), "x")
        self.assertEqual(tbl.cell(9, 9), "")

    def test_empty_and_short_rows(self):
        empty = table.Table()
        self.assertEqual(children(empty.build().build()), [])
        short = table.Table(
            columns=[table.Column(title="A", width=2),
                     table.Column(title="B", width=2)],
            rows=[["1"]], hide_header=True, width=5,
        )
        kids = children(short.build().build())
        self.assertEqual(len(kids), 1)
        cells = children(kids[0])
        self.assertEqual(box_text(cells[2]), "  ")


class TestTextInput(unittest.TestCase):
    def test_editing(self):
        inp = input_mod.TextInput()
        self.assertTrue(inp.insert_string("hello"))
        self.assertEqual(inp.text(), "hello")
        self.assertEqual(inp.cursor, 5)
        self.assertTrue(inp.left())
        self.assertEqual(inp.cursor, 4)
        self.assertTrue(inp.insert_rune("X"))
        self.assertEqual(inp.text(), "hellXo")
        self.assertEqual(inp.cursor, 5)
        self.assertTrue(inp.backspace())
        self.assertEqual(inp.text(), "hello")
        self.assertEqual(inp.cursor, 4)
        self.assertTrue(inp.delete())
        self.assertEqual(inp.text(), "hell")
        self.assertEqual(inp.cursor, 4)
        self.assertTrue(inp.home())
        self.assertEqual(inp.cursor, 0)
        self.assertFalse(inp.home())
        self.assertFalse(inp.left())
        self.assertTrue(inp.right())
        self.assertEqual(inp.cursor, 1)
        self.assertTrue(inp.end())
        self.assertEqual(inp.cursor, 4)
        empty = input_mod.TextInput()
        self.assertFalse(empty.backspace())
        self.assertFalse(empty.delete())
        self.assertFalse(empty.left())
        self.assertFalse(empty.right())
        self.assertFalse(empty.delete_to_end())

    def test_words_and_limits(self):
        inp = input_mod.TextInput(value="foo bar  baz")
        inp.end()
        self.assertTrue(inp.word_left())
        self.assertEqual(inp.cursor, 9)
        self.assertTrue(inp.word_left())
        self.assertEqual(inp.cursor, 4)
        self.assertTrue(inp.word_right())
        self.assertEqual(inp.cursor, 9)
        inp.home()
        inp.end()
        self.assertTrue(inp.delete_word_left())
        self.assertEqual(inp.text(), "foo bar  ")
        self.assertEqual(inp.cursor, 9)
        limited = input_mod.TextInput(max_len=3)
        self.assertTrue(limited.insert_string("abcdef"))
        self.assertEqual(limited.text(), "abc")
        self.assertFalse(limited.insert_rune("x"))
        self.assertFalse(limited.insert_string("de"))
        limited.set_value("toolong")
        self.assertEqual(limited.text(), "too")
        self.assertEqual(limited.cursor, 3)

    def test_build_mask_placeholder_and_cursor(self):
        masked = input_mod.TextInput(value="secret", mask="\u2022", width=10,
                                     focused=True, cursor=6)
        root = masked.build().build()
        self.assertEqual(row_text(root), "\u2022" * 6 + " " * 4)
        cursor = root.get("cursor")
        self.assertIsNotNone(cursor)
        self.assertEqual(cursor.get("col"), 6)
        self.assertEqual(cursor.get("row"), 0)
        self.assertTrue(root.get("focused"))
        unfocused = input_mod.TextInput(value="secret", mask="\u2022", width=4)
        self.assertEqual(row_text(unfocused.build().build()), "\u2022" * 4)
        self.assertIsNone(unfocused.build().build().get("cursor"))
        placeholder = input_mod.TextInput(placeholder="name", width=6,
                                          focused=True)
        proot = placeholder.build().build()
        self.assertEqual(row_text(proot), "name  ")
        self.assertEqual(proot.get("cursor").get("col"), 0)

    def test_horizontal_scroll(self):
        scrolled = input_mod.TextInput(value="abcdefghij", cursor=10, width=5,
                                       focused=True)
        root = scrolled.build().build()
        self.assertEqual(row_text(root), "ghij ")
        self.assertEqual(root.get("cursor").get("col"), 4)
        cjk = input_mod.TextInput(value="\u4e2d\u6587\u4e2d\u6587\u4e2d\u6587",
                                  cursor=6, width=5, focused=True)
        cjk_root = cjk.build().build()
        self.assertEqual(display_width(row_text(cjk_root)), 5)
        self.assertEqual(row_text(cjk_root), "\u4e2d\u6587 ")
        self.assertEqual(cjk_root.get("cursor").get("col"), 4)

    def test_handle_key(self):
        inp = input_mod.TextInput()
        self.assertTrue(inp.handle_key({"key": "a", "char": "a"}))
        self.assertEqual(inp.text(), "a")
        self.assertTrue(inp.handle_key({"key": "left"}))
        self.assertEqual(inp.cursor, 0)
        self.assertTrue(inp.handle_key({"key": "\u4e2d", "char": "\u4e2d"}))
        self.assertEqual(inp.text(), "\u4e2da")
        self.assertFalse(inp.handle_key({"key": "enter"}))
        self.assertFalse(inp.handle_key({"key": "tab"}))
        self.assertFalse(inp.handle_key(None))
        self.assertTrue(inp.handle_key({"key": "ctrl-a"}))
        self.assertEqual(inp.cursor, 0)
        self.assertTrue(inp.handle_key({"key": "delete"}))
        self.assertEqual(inp.text(), "a")


class TestTextArea(unittest.TestCase):
    def test_editing_and_cursor(self):
        area = input_mod.TextArea()
        self.assertTrue(area.insert_string("ab\r\ncd"))
        self.assertEqual(area.text(), "ab\ncd")
        self.assertEqual(area.row_col(), (1, 2))
        self.assertTrue(area.up())
        self.assertEqual(area.row_col(), (0, 2))
        self.assertFalse(area.up())
        self.assertTrue(area.down())
        self.assertFalse(area.down())
        self.assertTrue(area.home())
        self.assertEqual(area.cursor, 3)
        self.assertTrue(area.end())
        self.assertEqual(area.cursor, 5)
        self.assertTrue(area.insert_rune("\n"))
        self.assertEqual(area.text(), "ab\ncd\n")
        self.assertEqual(area.row_col(), (2, 0))
        self.assertTrue(area.backspace())
        self.assertEqual(area.text(), "ab\ncd")
        self.assertEqual(area.cursor, 5)
        self.assertEqual(area.line(0), "ab")
        self.assertEqual(area.line(7), "")

    def test_scroll_and_cursor_protocol(self):
        area = input_mod.TextArea(value="l0\nl1\nl2\nl3\nl4", height=2,
                                  cursor=14, focused=True)
        root = area.build().build()
        kids = children(root)
        self.assertEqual(len(kids), 2)
        self.assertEqual(row_text(kids[0]), "l3")
        self.assertEqual(row_text(kids[1]), "l4 ")
        self.assertEqual(root.get("cursor").get("row"), 1)
        self.assertEqual(root.get("cursor").get("col"), 2)
        cjk = input_mod.TextArea(value="\u4e2d\u6587\u4e2d\u6587\n\u77ed",
                                 width=5, focused=True, cursor=4)
        cjk_root = cjk.build().build()
        self.assertEqual(row_text(children(cjk_root)[0]), "\u4e2d\u6587 ")
        self.assertEqual(cjk_root.get("cursor").get("col"), 4)

    def test_mask_and_page_keys(self):
        masked = input_mod.TextArea(value="ab\ncd", mask="*", width=4,
                                    height=2, focused=True, cursor=2)
        root = masked.build().build()
        self.assertEqual(row_text(children(root)[0]), "**  ")
        self.assertEqual(row_text(children(root)[1]), "**  ")
        area = input_mod.TextArea(value="a\nb\nc\nd\ne\nf", height=2)
        area.cursor = 0
        self.assertTrue(area.page_down())
        self.assertGreaterEqual(area.cursor, 2)
        self.assertTrue(area.page_up())
        self.assertEqual(area.cursor, 0)
        self.assertTrue(area.handle_key({"key": "enter"}))
        self.assertEqual(area.cursor, 1)
        self.assertFalse(area.handle_key({"key": "esc"}))

    def test_placeholder_empty_build(self):
        area = input_mod.TextArea(placeholder="notes", width=7, focused=True)
        root = area.build().build()
        self.assertEqual(len(children(root)), 1)
        self.assertEqual(row_text(children(root)[0]), "notes  ")
        self.assertIsNotNone(root.get("cursor"))


class TestModal(unittest.TestCase):
    def test_position_and_backdrop(self):
        dlg = modal.Modal(
            id="confirm", title="Confirm", width=20, height=6,
            center=True, parent_width=80, parent_height=24,
            backdrop=True, backdrop_style="dim",
            rows=[modal.FrameRow(text="ok", id="confirm:ok",
                                 input=["mouse"])],
        )
        self.assertEqual(dlg.position(), (30, 9))
        root = dlg.build().build()
        backdrop = find_box(root, "confirm:backdrop")
        self.assertIsNotNone(backdrop)
        self.assertEqual(box_style(backdrop), "dim")
        self.assertEqual(box_size(backdrop)[:2], (80, 24))
        layer = find_box(root, "confirm")
        self.assertIsNotNone(layer)
        self.assertEqual(layer.get("pos"), (30, 9))
        self.assertIsNotNone(find_box(root, "confirm:ok"))

    def test_position_clamps_and_offsets(self):
        dlg = modal.Modal(width=60, height=40, center=True, parent_width=40,
                          parent_height=20)
        self.assertEqual(dlg.position(), (0, 0))
        offset = modal.Modal(x=3, y=2)
        self.assertEqual(offset.position(), (3, 2))
        self.assertIsNone(find_box(offset.build().build(), "confirm:backdrop"))


class TestMenu(unittest.TestCase):
    def test_move_skips_and_selects(self):
        menu = modal.Menu(items=[
            modal.MenuItem(id="open", label="Open", hotkey="o"),
            modal.MenuItem(separator=True),
            modal.MenuItem(id="del", label="Delete", hotkey="d",
                           disabled=True),
            modal.MenuItem(id="quit", label="Quit", hotkey="q"),
        ], selected=0)
        menu.move(1)
        self.assertEqual(menu.selected, 3)
        menu.move(1)
        self.assertEqual(menu.selected, 3)
        menu.move(-1)
        self.assertEqual(menu.selected, 0)
        menu.move(-1)
        self.assertEqual(menu.selected, 0)
        self.assertFalse(menu.select(1))
        self.assertFalse(menu.select(2))
        self.assertTrue(menu.select(3))
        self.assertEqual(menu.value(), "quit")

    def test_hotkey_and_build(self):
        menu = modal.Menu(
            id="menu", title="Actions", width=16, x=1, y=1,
            selected_style="sel", disabled_style="off",
            items=[
                modal.MenuItem(id="open", label="Open", hotkey="o"),
                modal.MenuItem(id="del", label="Delete", hotkey="d",
                               disabled=True),
                modal.MenuItem(id="quit", label="Quit", hotkey="q"),
            ],
        )
        value, ok = menu.hotkey({"key": "d", "char": "d"})
        self.assertFalse(ok)
        self.assertEqual(value, "")
        value, ok = menu.hotkey({"key": "Q", "char": "Q"})
        self.assertTrue(ok)
        self.assertEqual(value, "quit")
        self.assertEqual(menu.selected, 2)
        _, ok = menu.hotkey({"key": "x", "char": "x"})
        self.assertFalse(ok)
        root = menu.build().build()
        self.assertEqual(root.get("pos"), (1, 1))
        open_row = find_box(root, "open")
        self.assertIsNotNone(open_row)
        self.assertIn("Open", box_text(open_row))
        self.assertEqual(open_row.get("input"), ["mouse"])
        selected = find_box(root, "quit")
        self.assertIsNotNone(selected)
        self.assertTrue(box_text(selected).startswith(
            list_mod.DEFAULT_LIST_MARKER))
        self.assertEqual(box_style(selected), "sel")
        empty = modal.Menu()
        empty.move(1)
        self.assertEqual(empty.value(), "")
        self.assertIsNotNone(empty.build().build())


class TestScrollbar(unittest.TestCase):
    def test_thumb_proportional(self):
        bar = scrollbar.Scrollbar(total=100, visible=10, height=10)
        self.assertEqual(bar.thumb(), (0, 1))
        bar.offset = 90
        self.assertEqual(bar.thumb(), (9, 1))
        bar.offset = 45
        self.assertEqual(bar.thumb()[0], 4)
        half = scrollbar.Scrollbar(total=100, visible=50, height=10)
        self.assertEqual(half.thumb(), (0, 5))

    def test_thumb_degenerate(self):
        fits = scrollbar.Scrollbar(total=5, visible=10, height=10)
        self.assertEqual(fits.thumb(), (0, 10))
        self.assertEqual(fits.offset_at(7), 0)
        empty = scrollbar.Scrollbar(total=0, visible=0, height=4)
        self.assertEqual(empty.thumb(), (0, 4))
        zero_visible = scrollbar.Scrollbar(total=10, visible=0, height=10,
                                           offset=5)
        self.assertEqual(zero_visible.thumb()[1], 1)
        no_track = scrollbar.Scrollbar(total=10, visible=3)
        self.assertEqual(no_track.thumb(), (0, 0))

    def test_clamps_offset(self):
        bar = scrollbar.Scrollbar(total=100, visible=10, height=10,
                                  offset=1000)
        self.assertEqual(bar.thumb()[0], 9)
        bar.offset = -50
        self.assertEqual(bar.thumb()[0], 0)

    def test_offset_at(self):
        bar = scrollbar.Scrollbar(total=100, visible=10, height=10)
        self.assertEqual(bar.offset_at(0), 0)
        self.assertEqual(bar.offset_at(9), 90)
        self.assertEqual(bar.offset_at(100), 90)
        wide = scrollbar.Scrollbar(total=40, visible=10, height=10)
        self.assertEqual(wide.thumb(), (0, 2))
        mid = wide.offset_at(5)
        self.assertGreater(mid, 0)
        self.assertLess(mid, 30)

    def test_horizontal_build(self):
        bar = scrollbar.Scrollbar(id="sb", total=20, visible=5, width=10)
        self.assertFalse(bar.is_vertical())
        self.assertEqual(bar.track_length(), 10)
        row = bar.build().build()
        self.assertEqual(len(children(row)), 10)
        self.assertEqual(box_text(children(row)[0]),
                         scrollbar.SCROLLBAR_THUMB_HORIZONTAL)
        self.assertEqual(bar.thumb(), (0, 2))

    def test_vertical_build_caps(self):
        bar = scrollbar.Scrollbar(id="sb", total=10, visible=2, height=5,
                                  up="\u25b2", down="\u25bc")
        kids = children(bar.build().build())
        self.assertEqual(len(kids), 7)
        self.assertEqual(box_text(kids[0]), "\u25b2")
        self.assertEqual(box_text(kids[6]), "\u25bc")
        self.assertEqual(bar.thumb(), (0, 1))


class TestMouse(unittest.TestCase):
    def test_hit_precedence_topmost_wins(self):
        regions = [
            mouse.HitRegion(id="below", x=0, y=0, w=10, h=10),
            mouse.HitRegion(id="above", x=2, y=2, w=3, h=3),
        ]
        region, ok = mouse.hit(regions, 3, 3)
        self.assertTrue(ok)
        self.assertEqual(region.id, "above")
        region, ok = mouse.hit(regions, 0, 0)
        self.assertTrue(ok)
        self.assertEqual(region.id, "below")
        _, ok = mouse.hit(regions, 20, 20)
        self.assertFalse(ok)
        self.assertEqual(mouse.hit_id(regions, 3, 3), ("above", True))

    def test_hit_region_unbounded_axis(self):
        region = mouse.HitRegion(id="row", x=0, y=4, w=0, h=1)
        self.assertTrue(region.contains(999, 4))
        self.assertFalse(region.contains(0, 5))

    def test_list_regions_and_row_at(self):
        lst = list_mod.List(
            items=["a", "b", "c", "d", "e"], height=3, header="H", offset=1,
            row_id=lambda i: "row:" + chr(ord("a") + i),
        )
        regions = mouse.list_regions(lst, 0, 0, 20)
        self.assertEqual(len(regions), 2)
        self.assertEqual(regions[0].y, 1)
        self.assertEqual(regions[0].data, 1)
        self.assertEqual(regions[0].id, "row:b")
        self.assertEqual(regions[1].y, 2)
        self.assertEqual(regions[1].data, 2)
        self.assertEqual(lst.row_at(2), (2, True))
        self.assertEqual(lst.row_at(0), (0, False))
        self.assertEqual(lst.row_at(9), (0, False))

    def test_virtual_list_regions(self):
        lst = list_mod.VirtualList(
            rows=[list_mod.ListRow(text="a", id="a"),
                  list_mod.ListRow(text="b", id="b"),
                  list_mod.ListRow(text="c", id="c")],
            height=2,
        )
        regions = mouse.virtual_list_regions(lst, 3, 5, 8)
        self.assertEqual(len(regions), 2)
        self.assertEqual((regions[0].x, regions[0].y), (3, 5))
        self.assertEqual(regions[1].id, "b")
        self.assertEqual(regions[1].data, 1)

    def test_table_regions_and_row_at(self):
        tbl = table.Table(
            columns=[table.Column(title="A", width=2),
                     table.Column(title="B", width=3)],
            rows=[["x", "y"], ["z", "w"]],
            row_id=lambda i: "r",
        )
        regions = mouse.table_regions(tbl, 0, 0)
        self.assertEqual(len(regions), 4)
        self.assertEqual(regions[0].data, (0, 0))
        self.assertEqual(regions[1].data, (0, 1))
        self.assertEqual(regions[1].x, 3)
        self.assertEqual(regions[2].y, 2)
        self.assertEqual(tbl.row_at(1), (0, True))
        self.assertEqual(tbl.row_at(0), (0, False))
        self.assertEqual(tbl.row_at(9), (0, False))

    def test_apply_wheel_clamps(self):
        self.assertEqual(mouse.apply_wheel(0, 100, 10, 3), 3)
        self.assertEqual(mouse.apply_wheel(5, 100, 10, -100), 0)
        self.assertEqual(mouse.apply_wheel(80, 100, 10, 999), 90)
        self.assertEqual(mouse.apply_wheel(4, 5, 10, 3), 0)

    def test_on_wheel_methods(self):
        lst = list_mod.List(items=[""] * 50, height=5)
        lst.on_wheel({"delta": 4})
        self.assertEqual(lst.offset, 4)
        lst.on_wheel(None)
        self.assertEqual(lst.offset, 4)
        virtual = list_mod.VirtualList(rows=[list_mod.ListRow()] * 50,
                                       height=5)
        virtual.on_wheel({"delta": 3})
        self.assertEqual(virtual.offset, 3)
        self.assertEqual(mouse.table_scroll(0, 100, 10, 7), 7)

    def test_drag_rect_normalization(self):
        cases = [
            ("down-right", 2, 3, 5, 8, mouse.Rect(2, 3, 4, 6)),
            ("up-left", 5, 8, 2, 3, mouse.Rect(2, 3, 4, 6)),
            ("up-right", 5, 3, 2, 8, mouse.Rect(2, 3, 4, 6)),
            ("down-left", 2, 8, 5, 3, mouse.Rect(2, 3, 4, 6)),
        ]
        for name, sx, sy, x, y, want in cases:
            drag = mouse.Drag()
            drag.begin(sx, sy, "left")
            drag.update(x, y)
            rect, ok = drag.end()
            self.assertTrue(ok, name)
            self.assertEqual(rect, want, name)
            self.assertFalse(drag.active, name)
        _, ok = mouse.Drag().end()
        self.assertFalse(ok)

    def test_select_range_and_list_range(self):
        self.assertEqual(mouse.select_range(9, 2), (2, 9))
        lst = list_mod.List(items=["a", "b", "c", "d", "e"], height=3)
        self.assertEqual(mouse.list_select_range(lst, 2, 0), (0, 2, True))
        self.assertEqual(mouse.list_select_range(lst, 5, 7), (0, 0, False))
        virtual = list_mod.VirtualList(rows=[list_mod.ListRow()] * 9,
                                       height=4)
        self.assertEqual(mouse.virtual_list_select_range(virtual, 0, 3),
                         (0, 3, True))

    def test_click_tracker(self):
        base = datetime(1970, 1, 1)
        tracker = mouse.ClickTracker(threshold=timedelta(milliseconds=400))
        self.assertEqual(tracker.click(1, 1, base), 1)
        self.assertEqual(
            tracker.click(1, 1, base + timedelta(milliseconds=100)), 2)
        self.assertEqual(
            tracker.click(1, 1, base + timedelta(milliseconds=150)), 3)
        self.assertEqual(
            tracker.click(1, 1, base + timedelta(milliseconds=160)), 1)
        self.assertEqual(tracker.click(1, 1, base + timedelta(seconds=1)), 1)
        self.assertEqual(
            tracker.click(1, 1, base + timedelta(seconds=1, milliseconds=100)),
            2)
        self.assertEqual(
            tracker.click(5, 5, base + timedelta(seconds=1, milliseconds=150)),
            1)
        zero = mouse.ClickTracker()
        self.assertEqual(zero.click(0, 0, base), 1)
        self.assertEqual(
            zero.click(0, 0, base + timedelta(milliseconds=10)), 2)


class TestHover(unittest.TestCase):
    def test_enter_leave_and_style(self):
        h = hover.Hover(style="hover", regions=[
            mouse.HitRegion(id="a", x=0, y=0, w=4, h=1),
            mouse.HitRegion(id="b", x=4, y=0, w=4, h=1),
        ])
        self.assertEqual(h.update(1, 0), "a")
        self.assertEqual(h.style_for("a"), "hover")
        self.assertEqual(h.style_for("b"), "")
        self.assertEqual(h.update(5, 0), "b")
        self.assertEqual(h.style_for("a"), "")
        self.assertEqual(h.update(20, 0), "")
        self.assertEqual(h.hovered(), "")
        self.assertEqual(h.style_for("b"), "")
        h.clear()
        self.assertEqual(h.current_style, "")

    def test_node_fallback(self):
        h = hover.Hover(node="surface", style="hover",
                        regions=[mouse.HitRegion(x=0, y=0, w=10, h=10)])
        self.assertEqual(h.update(2, 2), "surface")
        self.assertEqual(h.style_for("surface"), "hover")


class TestContextMenu(unittest.TestCase):
    def _menu(self):
        return modal.Menu(id="ctx", width=10, items=[
            modal.MenuItem(id="copy", label="Copy", hotkey="c"),
            modal.MenuItem(id="paste", label="Paste", hotkey="p"),
        ])

    def test_open_clamps_inside_parent(self):
        menu = contextmenu.ContextMenu(menu=self._menu(), parent_width=20,
                                       parent_height=10)
        x, y = menu.open(18, 8)
        self.assertTrue(menu.visible)
        self.assertEqual((x, y), (10, 6))
        self.assertEqual((menu.anchor_x, menu.anchor_y), (x, y))
        box = menu.build().build()
        self.assertEqual(box.get("pos"), (10, 6))

    def test_open_without_parent(self):
        menu = contextmenu.ContextMenu(menu=self._menu())
        self.assertEqual(menu.open(100, 50), (100, 50))

    def test_margin_and_negative(self):
        menu = contextmenu.ContextMenu(menu=self._menu(), parent_width=12,
                                       parent_height=6, margin=1)
        self.assertEqual(menu.open(11, 5), (1, 1))
        menu.margin = 0
        self.assertEqual(menu.open(-4, -4)[0], 0)

    def test_close_and_height(self):
        menu = contextmenu.ContextMenu(menu=self._menu())
        menu.open(3, 4)
        self.assertEqual(menu.menu_height(), 4)
        menu.close()
        self.assertFalse(menu.visible)
        box = menu.build().build()
        self.assertIs(box.get("visible"), False)
        self.assertEqual(len(children(box)), 0)

    def test_keyboard_drives_embedded_menu(self):
        menu = contextmenu.ContextMenu(menu=self._menu())
        menu.open(0, 0)
        menu.move(1)
        self.assertEqual(menu.selected, 1)
        value, ok = menu.hotkey({"key": "c"})
        self.assertTrue(ok)
        self.assertEqual(value, "copy")
        self.assertEqual(menu.value(), "copy")


class TestValidate(unittest.TestCase):
    def test_required(self):
        v = validate.required("required")
        self.assertEqual(v(""), "required")
        self.assertEqual(v("   "), "required")
        self.assertEqual(v(" ok "), "")

    def test_min_max_len(self):
        v = validate.min_len(3, "short")
        self.assertEqual(v(""), "")
        self.assertEqual(v("ab"), "short")
        self.assertEqual(v("abc"), "")
        self.assertEqual(v("\u4e2d\u4e2d\u4e2d"), "")
        self.assertEqual(validate.min_len(0, "short")("x"), "")
        v = validate.max_len(3, "long")
        self.assertEqual(v(""), "")
        self.assertEqual(v("abcd"), "long")
        self.assertEqual(v("abc"), "")
        self.assertEqual(validate.max_len(0, "long")("anything"), "")

    def test_pattern(self):
        digits = validate.pattern(r"^\d+$", "digits only")
        self.assertEqual(digits(""), "")
        self.assertEqual(digits("123"), "")
        self.assertEqual(digits("12a"), "digits only")
        broken = validate.pattern("[", "bad pattern")
        self.assertEqual(broken(""), "")
        self.assertEqual(broken("x"), "bad pattern")

    def test_email(self):
        v = validate.email("bad email")
        cases = {
            "": "",
            "a@b.co": "",
            "first.last+tag@x.io": "",
            "no-at-sign": "bad email",
            "a@b": "bad email",
            "a@b.": "bad email",
            "a b@c.d": "bad email",
        }
        for value, want in cases.items():
            self.assertEqual(v(value), want, value)

    def test_int_range(self):
        v = validate.int_range(1, 10, "out of range")
        cases = {
            "": "",
            "1": "",
            "10": "",
            " 5 ": "",
            "0": "out of range",
            "11": "out of range",
            "x": "out of range",
            "1.5": "out of range",
        }
        for value, want in cases.items():
            self.assertEqual(v(value), want, value)

    def test_one_of(self):
        v = validate.one_of(["red", "green", "blue"], "pick one")
        self.assertEqual(v(""), "")
        self.assertEqual(v("green"), "")
        self.assertEqual(v("yellow"), "pick one")

    def test_all_and_custom(self):
        v = validate.all_of(validate.required("required"),
                            validate.min_len(3, "short"),
                            validate.email("bad email"))
        self.assertEqual(v(""), "required")
        self.assertEqual(v("ab"), "short")
        self.assertEqual(v("not-an-email"), "bad email")
        self.assertEqual(v("a@b.co"), "")
        with_nil = validate.all_of(None, validate.required("required"))
        self.assertEqual(with_nil(""), "required")
        self.assertEqual(validate.all_of()("anything"), "")
        self.assertEqual(validate.custom(None)("x"), "")
        self.assertEqual(validate.custom(lambda value: "nope")("x"), "nope")


class TestForm(unittest.TestCase):
    def _new_form(self):
        return form.Form(
            id="login", width=30,
            fields=[
                form.Field(id="user", label="User",
                           input=input_mod.TextInput(id="user"),
                           required=True),
                form.Field(id="skip", label="Skip",
                           input=input_mod.TextInput(id="skip"),
                           disabled=True),
                form.Field(id="pass", label="Pass",
                           input=input_mod.TextInput(id="pass"),
                           validate=validate.min_len(4, "too short")),
            ],
        )

    def test_navigation_skips_disabled_and_readonly(self):
        f = self._new_form()
        self.assertEqual(f.focus_id(), "user")
        f.next()
        self.assertEqual(f.focus_id(), "pass")
        f.next()
        self.assertEqual(f.focus_id(), "user")
        f.prev()
        self.assertEqual(f.focus_id(), "pass")
        f.fields[2].readonly = True
        self.assertEqual(f.focus_id(), "pass")
        f.prev()
        self.assertEqual(f.focus_id(), "user")

    def test_navigation_all_disabled_is_stable(self):
        f = form.Form(fields=[
            form.Field(id="a", disabled=True),
            form.Field(id="b", readonly=True),
        ])
        f.next()
        self.assertEqual(f.focus_id(), "a")
        empty = form.Form()
        empty.next()
        empty.prev()
        self.assertEqual(empty.focus_id(), "")

    def test_values_and_input(self):
        f = self._new_form()
        f.set_values({"user": "ada", "pass": "hunter2"})
        values = f.values()
        self.assertEqual(values["user"], "ada")
        self.assertEqual(values["pass"], "hunter2")
        self.assertIsNotNone(f.input("user"))
        self.assertEqual(f.input("user").text(), "ada")
        self.assertIsNone(f.input("missing"))
        f.input("user").insert_rune("!")
        self.assertEqual(f.values()["user"], "ada!")

    def test_validate_aggregation(self):
        f = form.Form(fields=[
            form.Field(id="user", input=input_mod.TextInput(), required=True),
            form.Field(id="pass", input=input_mod.TextInput(),
                       validate=validate.min_len(4, "too short")),
        ])
        self.assertFalse(f.validate())
        self.assertEqual(f.fields[0].error, form.FORM_REQUIRED_MESSAGE)
        self.assertEqual(f.fields[1].error, "")
        f.set_values({"user": "ada", "pass": "x"})
        self.assertFalse(f.validate())
        self.assertEqual(f.fields[0].error, "")
        self.assertEqual(f.fields[1].error, "too short")
        f.set_values({"pass": "longenough"})
        self.assertTrue(f.validate())
        self.assertEqual(f.fields[0].error, "")
        self.assertEqual(f.fields[1].error, "")
        required_overrides = form.Form(fields=[form.Field(
            input=input_mod.TextInput(), required=True,
            validate=lambda value: "custom")])
        required_overrides.validate()
        self.assertEqual(required_overrides.fields[0].error,
                         form.FORM_REQUIRED_MESSAGE)

    def test_handle_key_routing(self):
        f = self._new_form()
        self.assertFalse(f.handle_key("unknown", {"key": "a", "char": "a"}))
        self.assertEqual(f.input("user").text(), "")
        self.assertTrue(f.handle_key("user", {"key": "a", "char": "a"}))
        self.assertEqual(f.input("user").text(), "a")
        self.assertTrue(f.handle_key("user", {"key": "tab"}))
        self.assertEqual(f.focus_id(), "pass")
        self.assertTrue(f.handle_key("", {"key": "shift-tab"}))
        self.assertEqual(f.focus_id(), "user")
        self.assertTrue(f.handle_key("user", {"key": "enter"}))
        self.assertEqual(f.focus_id(), "pass")
        self.assertFalse(f.handle_key("user", None))

    def test_validate_on_change(self):
        f = form.Form(validate_on_change=True, fields=[form.Field(
            id="email", input=input_mod.TextInput(id="email"), required=True,
            validate=validate.email("bad email"))])
        f.fields[0].error = form.FORM_REQUIRED_MESSAGE
        f.handle_key("email", {"key": "a", "char": "a"})
        self.assertEqual(f.fields[0].error, "bad email")
        f.input("email").set_value("a@b.co")
        f.fields[0].error = "stale"
        f.handle_key("email", {"key": "x", "char": "x"})
        self.assertEqual(f.fields[0].error, "")
        f.input("email").set_value("ab")
        f.fields[0].error = "stale"
        f.handle_key("email", {"key": "backspace"})
        self.assertEqual(f.fields[0].error, "bad email")

    def test_and_field_build(self):
        f = self._new_form()
        f.fields[0].error = "user is required"
        f.fields[0].help = "your login"
        root = f.build().build()
        self.assertEqual(root.get("id"), "login")
        self.assertEqual(len(children(root)), len(f.fields))
        field = children(root)[0]
        self.assertEqual(box_text(children(field)[0]), "User *")
        self.assertEqual(box_text(children(field)[2]), "user is required")
        f.fields[0].error = ""
        help_root = f.build().build()
        help_line = children(children(help_root)[0])[2]
        self.assertEqual(box_text(help_line), "your login")
        built = form.Field().build().build()
        self.assertEqual(len(children(built)), 0)

    def test_error_style_propagation(self):
        f = form.Form(error_style="danger", fields=[form.Field(
            id="a", error="boom", input=input_mod.TextInput())])
        field = children(f.build().build())[0]
        self.assertEqual(box_style(children(field)[1]), "danger")

    def test_width_propagates_to_input(self):
        f = form.Form(id="f", width=20, fields=[form.Field(
            id="a", input=input_mod.TextInput(id="a", width=20))])
        root = f.build().build()
        inp = find_box(root, "a")
        self.assertIsNotNone(inp)
        self.assertEqual(box_size(inp)[0], 20)
        self.assertEqual(display_width(row_text(inp)), 20)


class TestSelect(unittest.TestCase):
    def _test_select(self):
        return select_mod.Select(
            id="color", label="Color", width=20,
            options=[
                select_mod.Option(value="red", label="Red"),
                select_mod.Option(value="green", label="Green", disabled=True),
                select_mod.Option(value="grey", label="Grey"),
                select_mod.Option(value="blue", label="Blue"),
            ],
        )

    def test_selected(self):
        sel = self._test_select()
        self.assertFalse(sel.selected()[1])
        self.assertEqual(sel.index(), -1)
        self.assertTrue(sel.select_index(3))
        self.assertEqual(sel.value, "blue")
        option, ok = sel.selected()
        self.assertTrue(ok)
        self.assertEqual(option.value, "blue")
        self.assertFalse(sel.select_index(1))
        self.assertFalse(sel.select_index(len(sel.options)))
        self.assertFalse(sel.select_index(-1))
        sel.value = "green"
        option, ok = sel.selected()
        self.assertFalse(ok)
        self.assertIsNone(option)

    def test_move_skips_disabled(self):
        sel = self._test_select()
        sel.select_index(0)
        sel.move(1)
        self.assertEqual(sel.value, "grey")
        sel.move(1)
        self.assertEqual(sel.value, "blue")
        sel.move(1)
        self.assertEqual(sel.value, "blue")
        sel.move(-1)
        self.assertEqual(sel.value, "grey")
        sel.move(-3)
        self.assertEqual(sel.value, "red")
        unselected = self._test_select()
        unselected.move(1)
        self.assertEqual(unselected.value, "red")
        back = self._test_select()
        back.move(-1)
        self.assertEqual(back.value, "blue")
        empty = select_mod.Select()
        empty.move(1)
        self.assertEqual(empty.value, "")
        empty.move(0)

    def test_typeahead(self):
        sel = self._test_select()
        sel.select_index(0)
        self.assertTrue(sel.typeahead("gr"))
        self.assertEqual(sel.value, "grey")
        self.assertTrue(sel.typeahead("b"))
        self.assertEqual(sel.value, "blue")
        self.assertTrue(sel.typeahead("Re"))
        self.assertEqual(sel.value, "red")
        self.assertFalse(sel.typeahead("zzz"))
        self.assertFalse(sel.typeahead(""))
        self.assertFalse(select_mod.Select().typeahead("a"))
        values = select_mod.Select(options=[
            select_mod.Option(value="alpha"),
            select_mod.Option(value="beta", label="B"),
        ])
        self.assertTrue(values.typeahead("be"))
        self.assertEqual(values.value, "beta")

    def test_build_closed(self):
        sel = self._test_select()
        sel.value = "blue"
        root = sel.build().build()
        self.assertEqual(root.get("id"), "color")
        self.assertEqual(box_text(children(root)[0]), "Color")
        row = children(root)[1]
        text = row_text(row)
        self.assertEqual(display_width(text), 20)
        self.assertEqual(text[:4], "Blue")
        self.assertTrue(text.rstrip(" ").endswith("\u25be"))
        empty = select_mod.Select(placeholder="choose", indicator="!")
        self.assertEqual(row_text(empty.build().build()), "choose!")

    def test_build_dropdown(self):
        sel = self._test_select()
        sel.open = True
        sel.value = "grey"
        root = sel.build().build()
        self.assertEqual(len(children(root)), len(sel.options))
        selected = row_text(children(root)[2])
        self.assertTrue(selected.startswith(list_mod.DEFAULT_LIST_MARKER))
        disabled = children(root)[1]
        self.assertTrue(box_text(disabled).startswith("  "))
        sel.disabled_style = "muted"
        dropdown = sel.build_dropdown().build()
        self.assertEqual(box_style(children(dropdown)[1]), "muted")

    def test_unicode_labels(self):
        sel = select_mod.Select(
            options=[select_mod.Option(value="cn", label="\u4e2d\u6587"),
                     select_mod.Option(value="jp",
                                       label="\u65e5\u672c\u8a9e")],
            placeholder="\u9009\u62e9",
        )
        self.assertTrue(sel.typeahead("\u4e2d"))
        self.assertEqual(sel.value, "cn")
        sel.value = "jp"
        text = row_text(sel.build().build())
        self.assertEqual(display_width(text),
                         display_width("\u65e5\u672c\u8a9e \u25be"))


class TestDate(unittest.TestCase):
    def test_days_in_month_and_is_leap(self):
        cases = [
            (2023, 1, 31, False),
            (2023, 2, 28, False),
            (2024, 2, 29, True),
            (1900, 2, 28, False),
            (2000, 2, 29, True),
            (2023, 4, 30, False),
            (2023, 12, 31, False),
        ]
        for year, month, days, leap in cases:
            date = datepicker.Date(year, month, 1)
            self.assertEqual(date.days_in_month(), days, str(date))
            self.assertEqual(date.is_leap(), leap, year)
        self.assertEqual(datepicker.Date(2024, 13, 1).days_in_month(), 0)

    def test_add_days_across_boundaries(self):
        cases = [
            (datepicker.Date(2024, 2, 28), 1, datepicker.Date(2024, 2, 29)),
            (datepicker.Date(2024, 2, 29), 1, datepicker.Date(2024, 3, 1)),
            (datepicker.Date(2023, 2, 28), 1, datepicker.Date(2023, 3, 1)),
            (datepicker.Date(2023, 12, 31), 1, datepicker.Date(2024, 1, 1)),
            (datepicker.Date(2024, 1, 1), -1, datepicker.Date(2023, 12, 31)),
            (datepicker.Date(2024, 3, 1), -1, datepicker.Date(2024, 2, 29)),
            (datepicker.Date(2024, 1, 1), 60, datepicker.Date(2024, 3, 1)),
            (datepicker.Date(2024, 1, 1), 0, datepicker.Date(2024, 1, 1)),
        ]
        for start, days, want in cases:
            self.assertEqual(start.add_days(days), want, str(start))
        zero = datepicker.Date()
        self.assertEqual(zero.add_days(1), zero)
        self.assertEqual(zero.add_months(1), zero)

    def test_add_months_clamps(self):
        cases = [
            (datepicker.Date(2024, 1, 31), 1, datepicker.Date(2024, 2, 29)),
            (datepicker.Date(2023, 1, 31), 1, datepicker.Date(2023, 2, 28)),
            (datepicker.Date(2024, 1, 15), 1, datepicker.Date(2024, 2, 15)),
            (datepicker.Date(2024, 12, 15), 1, datepicker.Date(2025, 1, 15)),
            (datepicker.Date(2024, 1, 15), -1, datepicker.Date(2023, 12, 15)),
            (datepicker.Date(2024, 1, 15), 12, datepicker.Date(2025, 1, 15)),
            (datepicker.Date(2024, 1, 15), -13, datepicker.Date(2022, 12, 15)),
        ]
        for start, months, want in cases:
            self.assertEqual(start.add_months(months), want, str(start))

    def test_compare_weekday_string(self):
        a, b = datepicker.Date(2024, 1, 1), datepicker.Date(2024, 1, 2)
        self.assertEqual(a.compare(b), -1)
        self.assertEqual(b.compare(a), 1)
        self.assertEqual(a.compare(a), 0)
        self.assertTrue(a.before(b))
        self.assertFalse(a.after(b))
        self.assertTrue(a.equal(datepicker.Date(2024, 1, 1)))
        self.assertEqual(datepicker.Date(2024, 1, 1).weekday_name(), "Monday")
        self.assertEqual(datepicker.Date(2024, 2, 29).weekday_name(),
                         "Thursday")
        self.assertEqual(str(datepicker.Date(2024, 1, 2)), "2024-01-02")
        self.assertEqual(str(datepicker.Date()), "")

    def test_parse_date(self):
        self.assertEqual(datepicker.parse_date("2024-02-29"),
                         datepicker.Date(2024, 2, 29))
        for bad in ["", "2024-2-9", "2024-02-30", "2023-02-29", "2023-13-01",
                    "2024-00-10", "2024-01-32", "not-a-date",
                    "2024-01-01T00:00:00Z"]:
            with self.assertRaises(ValueError, msg=bad):
                datepicker.parse_date(bad)


class TestCalendar(unittest.TestCase):
    def test_grid_alignment(self):
        cases = [
            (datepicker.Date(2024, 1, 1), datepicker.Date(2024, 1, 1),
             datepicker.Date(2024, 2, 11)),
            (datepicker.Date(2024, 2, 1), datepicker.Date(2024, 1, 29),
             datepicker.Date(2024, 3, 10)),
            (datepicker.Date(2023, 12, 1), datepicker.Date(2023, 11, 27),
             datepicker.Date(2024, 1, 7)),
            (datepicker.Date(2024, 9, 1), datepicker.Date(2024, 8, 26),
             datepicker.Date(2024, 10, 6)),
        ]
        for month, first_cell, last_cell in cases:
            cal = datepicker.Calendar(month=month)
            grid = cal.grid()
            self.assertEqual(len(grid), 6)
            for row in grid:
                self.assertEqual(len(row), 7)
            self.assertEqual(grid[0][0], first_cell)
            self.assertEqual(grid[5][6], last_cell)
            for row in grid:
                self.assertEqual(row[0].weekday_name(), "Monday")
                for col in range(1, 7):
                    self.assertEqual(row[col], row[col - 1].add_days(1))
        self.assertIsNone(datepicker.Calendar().grid())

    def test_move_cursor(self):
        cal = datepicker.Calendar(month=datepicker.Date(2024, 1, 1),
                                  selected=datepicker.Date(2024, 1, 31))
        self.assertTrue(cal.move_cursor(1))
        self.assertEqual(cal.selected, datepicker.Date(2024, 2, 1))
        self.assertEqual((cal.month.month, cal.month.day), (2, 1))
        self.assertTrue(cal.move_cursor(-1))
        self.assertEqual(cal.selected, datepicker.Date(2024, 1, 31))
        self.assertEqual(cal.month.month, 1)
        self.assertFalse(cal.move_cursor(0))
        cursorless = datepicker.Calendar(month=datepicker.Date(2024, 3, 1))
        cursorless.move_cursor(7)
        self.assertEqual(cursorless.selected, datepicker.Date(2024, 3, 8))

    def test_build(self):
        cal = datepicker.Calendar(
            id="cal", month=datepicker.Date(2024, 2, 1),
            selected=datepicker.Date(2024, 2, 14),
            marked={datepicker.Date(2024, 2, 20): "\u2605"},
            today=datepicker.Date(2024, 2, 1),
            selected_style="reverse", marked_style="accent",
            today_style="bold",
        )
        root = cal.build().build()
        self.assertEqual(root.get("id"), "cal")
        self.assertEqual(box_text(children(root)[0]), "February 2024")
        weekdays = row_text(children(root)[1])
        self.assertEqual(display_width(weekdays), 21)
        trimmed = weekdays.strip()
        self.assertTrue(trimmed.startswith("Mo"))
        self.assertTrue(trimmed.endswith("Su"))
        grid = children(root)[2:]
        self.assertEqual(len(grid), 6)
        widths = {display_width(row_text(row)) for row in grid}
        self.assertEqual(widths, {21})
        self.assertEqual(_find_marked_style(root), "accent")
        self.assertNotEqual(_find_styled_cell(root, "reverse"), "")
        unset = datepicker.Calendar(month=datepicker.Date(2024, 2, 1))
        self.assertEqual(len(children(unset.build().build())), 8)

    def test_cjk_marked_width_stable(self):
        cal = datepicker.Calendar(
            month=datepicker.Date(2024, 2, 1),
            marked={datepicker.Date(2024, 2, 1): "\u6625\u8282",
                    datepicker.Date(2024, 2, 2): "\u2605"},
        )
        root = cal.build().build()
        widths = [display_width(row_text(row))
                  for row in children(root)[2:]]
        self.assertEqual(len(set(widths)), 1)


def _find_marked_style(root):
    for row in children(root):
        for cell in children(row):
            if box_style(cell) == "accent":
                return box_style(cell)
    return ""


def _find_styled_cell(root, style):
    for row in children(root):
        for cell in children(row):
            if box_style(cell) == style:
                return box_text(cell)
    return ""


class TestDayValidatorAndDateField(unittest.TestCase):
    def test_day_validator(self):
        v = datepicker.day_validator(datepicker.Date(2024, 1, 1),
                                     datepicker.Date(2024, 12, 31),
                                     "bad range")
        self.assertEqual(v(""), "")
        self.assertEqual(v("2024-06-15"), "")
        self.assertEqual(v("2023-12-31"), "bad range")
        self.assertEqual(v("2025-01-01"), "bad range")
        self.assertEqual(v("2024-02-30"), "bad range")
        open_min = datepicker.day_validator(datepicker.Date(),
                                            datepicker.Date(2024, 12, 31),
                                            "bad range")
        self.assertEqual(open_min("1900-01-01"), "")
        open_max = datepicker.day_validator(datepicker.Date(2024, 1, 1),
                                            datepicker.Date(), "bad range")
        self.assertEqual(open_max("2999-12-31"), "")

    def test_date_field(self):
        inp = input_mod.TextInput(id="due", placeholder="YYYY-MM-DD")
        field = datepicker.new_date_field("due", "Due", inp)
        self.assertEqual(field.id, "due")
        self.assertEqual(field.label, "Due")
        self.assertIsNotNone(field.validate)
        self.assertEqual(field.validate("2024-02-29"), "")
        self.assertEqual(field.validate("2024-02-30"),
                         datepicker.DATE_FIELD_MESSAGE)
        self.assertFalse(field.date()[1])
        inp.set_value("2024-07-04")
        date, ok = field.date()
        self.assertTrue(ok)
        self.assertEqual(date, datepicker.Date(2024, 7, 4))
        required = datepicker.new_date_field("due", "Due", inp)
        required.required = True
        f = form.Form(fields=[required])
        inp.set_value("")
        self.assertFalse(f.validate())
        self.assertEqual(f.fields[0].error, form.FORM_REQUIRED_MESSAGE)
        inp.set_value("2024-02-30")
        self.assertFalse(f.validate())
        self.assertEqual(f.fields[0].error, datepicker.DATE_FIELD_MESSAGE)


class TestSparkline(unittest.TestCase):
    def test_glyph_mapping(self):
        self.assertEqual(
            chart.Sparkline(values=[0, 1, 2, 3, 4, 5, 6, 7]).line(),
            "\u2581\u2582\u2583\u2584\u2585\u2586\u2587\u2588")
        self.assertEqual(chart.Sparkline(values=[5, 5, 5]).line(),
                         "\u2584\u2584\u2584")
        self.assertEqual(chart.Sparkline(values=[42]).line(), "\u2584")
        self.assertEqual(chart.Sparkline().line(), "")
        self.assertEqual(
            chart.Sparkline(values=[0, 5, 10], min=0.0, max=10.0).line(),
            "\u2581\u2585\u2588")
        self.assertEqual(
            chart.Sparkline(values=[0, 0, 10, 10], width=2).line(),
            "\u2581\u2588")
        built = chart.Sparkline(values=[0, 7],
                                style="accent").build().build()
        self.assertEqual(box_text(built), "\u2581\u2588")
        self.assertEqual(box_style(built), "accent")


class TestBarChart(unittest.TestCase):
    def test_vertical_dimensions(self):
        bar = chart.BarChart(values=[1, 2, 3, 4], width=7, style="bar",
                             label_style="lbl", labels=["a", "b", "c", "d"])
        self.assertEqual(bar.max_value(), 4)
        lines = bar.vertical_lines()
        self.assertEqual(len(lines), 2)
        self.assertEqual(lines[0], "\u2582 \u2584 \u2586 \u2588")
        self.assertEqual(lines[1], "a b c d")
        built = bar.build().build()
        kids = children(built)
        self.assertEqual(len(kids), 2)
        self.assertEqual(box_text(children(kids[0])[0]), "\u2582")
        self.assertEqual(box_style(children(kids[0])[0]), "bar")
        self.assertEqual(box_text(children(kids[1])[0]), "a")
        self.assertEqual(box_style(children(kids[1])[0]), "lbl")

    def test_vertical_partial_and_height(self):
        self.assertEqual(
            chart.BarChart(values=[0.5], max=1, width=1).vertical_lines()[0],
            "\u2584")
        lines = chart.BarChart(values=[4], max=4, width=1,
                               height=2).vertical_lines()
        self.assertEqual(lines, ["\u2588", "\u2588"])
        lines = chart.BarChart(values=[2], max=4, width=1,
                               height=2).vertical_lines()
        self.assertEqual(lines, [" ", "\u2588"])
        narrow = chart.BarChart(values=[1, 2, 3, 4], width=3).vertical_lines()
        self.assertGreater(len(narrow[0]), 0)
        self.assertIsNone(chart.BarChart().vertical_lines())

    def test_horizontal(self):
        lines = chart.BarChart(values=[50, 100], width=10,
                               horizontal=True).horizontal_lines()
        self.assertEqual(lines, ["\u2588" * 5, "\u2588" * 10])
        lines = chart.BarChart(values=[50, 100], width=10, horizontal=True,
                               labels=["a", "bb"],
                               label_style="lbl").horizontal_lines()
        self.assertEqual(lines[0], "a  \u2588\u2588\u2588\u2588")
        self.assertEqual(lines[1], "bb " + "\u2588" * 7)
        capped = chart.BarChart(values=[1, 2, 3], height=2,
                                horizontal=True).horizontal_lines()
        self.assertEqual(len(capped), 2)
        blank = chart.BarChart(values=[-1, 0], width=4,
                               horizontal=True).horizontal_lines()
        self.assertEqual(blank[0].strip(), "")
        self.assertEqual(blank[1].strip(), "")

    def test_selected_style(self):
        bar = chart.BarChart(values=[1, 1], width=3, style="base",
                             selected=1, selected_style="hot")
        kids = children(children(bar.build().build())[0])
        self.assertEqual(box_style(kids[0]), "base")
        self.assertEqual(box_style(kids[2]), "hot")


class TestHeatmap(unittest.TestCase):
    def test_shade_ramp_and_ragged(self):
        heat = chart.Heatmap(values=[[0, 1], [2, 3]])
        self.assertIsNone(heat.shades)
        grid = heat.grid()
        self.assertEqual(grid, [" \u2591", "\u2592\u2588"])
        self.assertEqual(heat.level(-100), 0)
        self.assertEqual(heat.level(100), 5)
        self.assertEqual(chart.Heatmap(values=[[5, 5]]).level(5), 3)
        ragged = chart.Heatmap(values=[[1, 2, 3], [4]],
                               col_labels=["x", "y", "z"],
                               row_labels=["r1", "r2"])
        self.assertEqual((ragged.rows(), ragged.cols()), (2, 3))
        self.assertEqual(ragged.cell(1, 1), " ")
        grid = ragged.grid()
        self.assertEqual(len(grid), 3)
        self.assertTrue(grid[1].startswith("r1 "))
        custom = chart.Heatmap(values=[[1], [2]],
                               shades=["a", "b", "c"]).grid()
        self.assertEqual(custom, ["a", "c"])
        self.assertIsNone(chart.Heatmap().grid())
        built = chart.Heatmap(values=[[1]], style="heat", label_style="lbl",
                              col_labels=["c"]).build().build()
        kids = children(built)
        self.assertEqual(len(kids), 2)
        self.assertEqual(box_style(kids[0]), "lbl")
        self.assertEqual(box_style(kids[1]), "heat")


class TestMeterAndLegend(unittest.TestCase):
    def test_meter_and_gauge(self):
        meter = chart.Meter(value=3, max=10, width=20, style="fill",
                            track_style="track", show_value=True,
                            label="cpu ")
        self.assertEqual(meter.fraction(), 0.3)
        self.assertEqual(meter.bar_text().count("\u2588"), 6)
        self.assertEqual(len(meter.bar_text()), 20)
        self.assertEqual(meter.value_text(), "3/10")
        kids = children(meter.build().build())
        self.assertEqual(len(kids), 4)
        self.assertEqual(chart.Gauge(value=1, max=0).fraction(), 0)
        self.assertEqual(chart.Meter(value=20, max=10).fraction(), 1)
        self.assertEqual(chart.Meter(value=1, max=2).value_text(), "")

    def test_legend(self):
        legend = chart.Legend(style="muted", items=[
            chart.LegendItem(label="go", color="accent", marker="x"),
            chart.LegendItem(label="ui"),
        ])
        self.assertEqual(legend.text(), "x go \u25a0 ui")
        kids = children(legend.build().build())
        self.assertEqual(len(kids), 5)
        self.assertEqual(box_text(kids[0]), "x")
        self.assertEqual(box_style(kids[0]), "accent")
        self.assertEqual(box_text(kids[1]), " go")
        self.assertEqual(box_style(kids[1]), "muted")
        self.assertEqual(box_text(kids[2]), " ")
        self.assertEqual(box_text(kids[3]), "\u25a0")
        self.assertEqual(box_style(kids[3]), "")
        self.assertEqual(chart.Legend(separator=" | ", items=[
            chart.LegendItem(label="a")]).text(), "\u25a0 a")
        self.assertEqual(chart.Legend(items=[
            chart.LegendItem(label="a"), chart.LegendItem(label="b")],
            separator="|").text(), "\u25a0 a|\u25a0 b")

    def test_chart_plain_text_is_ansi_free(self):
        texts = [
            chart.Sparkline(values=[1, 2, 3]).line(),
            "\n".join(chart.BarChart(values=[1, 2], width=3,
                                     horizontal=True).horizontal_lines()),
            "\n".join(chart.Heatmap(values=[[1, 2]]).grid()),
            chart.Legend(items=[chart.LegendItem(label="x")]).text(),
        ]
        for text in texts:
            self.assertNotIn("\x1b", text)
            self.assertGreaterEqual(display_width(text), 0)


class TestFormat(unittest.TestCase):
    def test_format_bytes(self):
        cases = [
            (0, "0 B"),
            (1, "1 B"),
            (512, "512 B"),
            (1023, "1023 B"),
            (1024, "1 KB"),
            (1536, "1.5 KB"),
            (1024 * 1024, "1 MB"),
            (3 * 1024 * 1024, "3 MB"),
            (1024 * 1024 * 1024, "1 GB"),
            (5 * 1024 * 1024 * 1024 + 512 * 1024 * 1024, "5.5 GB"),
            (-2048, "-2 KB"),
            (2 ** 63 - 1, "8 EB"),
        ]
        for value, want in cases:
            self.assertEqual(fmt.format_bytes(value), want, value)

    def test_format_count(self):
        cases = [
            (0, "0"),
            (7, "7"),
            (999, "999"),
            (1000, "1,000"),
            (1234, "1,234"),
            (1234567, "1,234,567"),
            (2 ** 63 - 1, "9,223,372,036,854,775,807"),
            (-1000, "-1,000"),
            (-1, "-1"),
        ]
        for value, want in cases:
            self.assertEqual(fmt.format_count(value), want, value)

    def test_format_duration_boundaries(self):
        cases = [
            (0, "0s"),
            (500, "500ns"),
            (999, "999ns"),
            (1500, "2\u00b5s"),
            (250 * 1000, "250\u00b5s"),
            (1500 * 1000, "1.5ms"),
            (250 * 1000 * 1000, "250ms"),
            (1000 * 1000 * 1000, "1s"),
            (1500 * 1000 * 1000, "1.5s"),
            (30 * 1000 * 1000 * 1000, "30s"),
            (90 * 1000 * 1000 * 1000, "1m30s"),
            (60 * 1000 * 1000 * 1000, "1m"),
            (3600 * 1000 * 1000 * 1000, "1h"),
            (3661 * 1000 * 1000 * 1000, "1h1m1s"),
            (3605 * 1000 * 1000 * 1000, "1h5s"),
            (-90 * 1000 * 1000 * 1000, "-1m30s"),
        ]
        for value, want in cases:
            self.assertEqual(fmt.format_duration(value), want, value)

    def test_format_percent_and_float(self):
        percent = [
            (0, 0, "0%"),
            (0.125, 1, "12.5%"),
            (1, 0, "100%"),
            (-0.5, 0, "-50%"),
            (1.0 / 3.0, 2, "33.33%"),
            (0.5, -1, "50%"),
        ]
        for fraction, digits, want in percent:
            self.assertEqual(fmt.format_percent(fraction, digits), want,
                             fraction)
        self.assertEqual(fmt.format_float(3.14159, 2, 0), "3.14")
        self.assertEqual(fmt.format_float(3.5, 0, 6), "     4")

    def test_format_non_finite(self):
        self.assertEqual(fmt.format_percent(math.nan, 1), "NaN%")
        self.assertEqual(fmt.format_percent(math.inf, 0), "+Inf%")
        self.assertEqual(fmt.format_float(math.nan, 2, 0), "NaN")
        self.assertEqual(fmt.format_float(-math.inf, 1, 0), "-Inf")
        self.assertEqual(fmt.pad_left("x", 4), "   x")
        self.assertEqual(fmt.pad_right("x", 4), "x   ")
        self.assertEqual(fmt.pad_left("toolong", 2), "toolong")

    def test_scale_value(self):
        cases = [
            (0, 0, ""),
            (999, 999, ""),
            (1500, 1.5, "K"),
            (2_000_000, 2, "M"),
            (1_000_000_000, 1, "G"),
            (1_000_000_000_000, 1, "T"),
            (1_000_000_000_000_000, 1, "P"),
            (1e18, 1, "E"),
            (-1500, -1.5, "K"),
        ]
        for value, want, suffix in cases:
            self.assertEqual(fmt.scale_value(value), (want, suffix), value)
        scaled, suffix = fmt.scale_value(math.nan)
        self.assertTrue(math.isnan(scaled))
        self.assertEqual(suffix, "")
        scaled, suffix = fmt.scale_value(math.inf)
        self.assertTrue(math.isinf(scaled) and scaled > 0)
        self.assertEqual(suffix, "")


if __name__ == "__main__":
    unittest.main()
