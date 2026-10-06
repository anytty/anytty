"""Visual widget toolkit tests (Go ``sdk/widgets`` parity).

Every assertion inspects the produced box tree (flow/children/content/style/
size/pos/input), never rendering. Cases mirror the Go ``*_test.go`` files for
tokens, border, chrome (layout), widgets (basics) and progress.
"""

import unittest

from tui2sdk import builder
from tui2sdk.widgets import basics, border, layout, progress, theme, tokens


def built(widget):
    return widget.build().build()


def find_box(root, node_id):
    if root is None:
        return None
    if root.get("id") == node_id:
        return root
    for child in root.get("children") or []:
        found = find_box(child, node_id)
        if found is not None:
            return found
    return None


def box_text(box):
    if not box:
        return ""
    content = box.get("content") or {}
    return content.get("text", "")


def row_text(box):
    return "".join(box_text(child) for child in box.get("children") or [])


class TestTokens(unittest.TestCase):
    def test_token_constants_match_host_names(self):
        expected = {
            "STYLE_DEFAULT": "default",
            "STYLE_BACKGROUND": "bg",
            "STYLE_FG": "fg",
            "STYLE_FOREGROUND": "foreground",
            "STYLE_STRONG_FOREGROUND": "strong-foreground",
            "STYLE_MUTED": "muted",
            "STYLE_ACCENT": "accent",
            "STYLE_ACCENT_DIM": "accent_dim",
            "STYLE_SUCCESS": "success",
            "STYLE_OK": "ok",
            "STYLE_WARNING": "warning",
            "STYLE_DANGER": "danger",
            "STYLE_INFO": "info",
            "STYLE_CHROME": "chrome",
            "STYLE_CHROME_FOCUS": "chrome_focus",
            "STYLE_HEADER": "header",
            "STYLE_TAB_ACTIVE": "tab_active",
            "STYLE_TAB_INACTIVE": "tab_inactive",
            "STYLE_FOOTER": "footer",
            "STYLE_FOOTER_ACCENT": "footer-accent",
            "STYLE_STATUS": "status",
            "STYLE_OVERLAY": "overlay",
            "STYLE_BORDER": "border",
            "STYLE_BORDER_FOCUS": "border_focus",
            "STYLE_BORDER_DEAD": "border_dead",
            "STYLE_ACTIVE_BORDER": "active-border",
            "STYLE_INACTIVE_BORDER": "inactive-border",
            "STYLE_SELECTION": "selection",
        }
        for name, want in expected.items():
            self.assertEqual(getattr(tokens, name), want, name)

    def test_style_attribute_helpers_are_idempotent(self):
        self.assertEqual(tokens.with_bold(""), "bold")
        self.assertEqual(tokens.with_bold(tokens.with_bold("accent")), "accent;bold")
        self.assertEqual(tokens.with_underline(""), "underline")
        self.assertEqual(tokens.with_reverse(tokens.with_reverse("fg:#101010")),
                         "fg:#101010;reverse")
        self.assertEqual(tokens.with_bold("fg:#111111;bold"), "fg:#111111;bold")

    def test_style_color_helpers(self):
        self.assertEqual(tokens.with_fg("", "#aabbcc"), "fg:#aabbcc")
        self.assertEqual(tokens.with_fg("bold", "#aabbcc"), "fg:#aabbcc;bold")
        self.assertEqual(tokens.with_fg("fg:#111111;bold", "#aabbcc"),
                         "fg:#aabbcc;bold")
        self.assertEqual(tokens.with_bg("fg:#111111;bg:#000000", "#ffffff"),
                         "fg:#111111;bg:#ffffff")
        self.assertEqual(tokens.with_fg("bold", ""), "bold")
        once = tokens.with_fg("bold", "#aabbcc")
        self.assertEqual(tokens.with_fg(once, "#aabbcc"), once)


class TestBorder(unittest.TestCase):
    def test_border_presets(self):
        normal = border.border_normal()
        self.assertEqual((normal.top_left, normal.top_right,
                          normal.bottom_left, normal.bottom_right),
                         ("┌", "┐", "└", "┘"))
        self.assertEqual((normal.top, normal.bottom, normal.left, normal.right),
                         ("─", "─", "│", "│"))
        rounded = border.border_rounded()
        self.assertEqual((rounded.top_left, rounded.top_right,
                          rounded.bottom_left, rounded.bottom_right),
                         ("╭", "╮", "╰", "╯"))
        thick = border.border_thick()
        self.assertEqual((thick.top_left, thick.top, thick.left), ("┏", "━", "┃"))
        double = border.border_double()
        self.assertEqual((double.top_left, double.top, double.left), ("╔", "═", "║"))
        self.assertEqual(border.border_set_or_default(rounded).top_left, "╭")
        self.assertEqual(border.border_set_or_default(border.BorderSet()).top_left, "┌")

    def test_border_box_draws_frame_and_title(self):
        root = built(border.BorderBox(id="panel", title="hi", width=10, height=4))
        self.assertEqual(root.get("id"), "panel")
        self.assertEqual(root["size"][:2], (10, 4))
        children = root["children"]
        self.assertEqual(len(children), 4)
        top = row_text(children[0])
        self.assertTrue(top.startswith("┌"))
        self.assertTrue(top.endswith("┐"))
        self.assertIn(" hi ", top)
        self.assertEqual(builder.display_width(top), 10)
        self.assertEqual(box_text(children[3]), "└" + "─" * 8 + "┘")
        body = children[1]["children"]
        self.assertEqual(len(body), 3)
        self.assertEqual(box_text(body[0]), "│")
        self.assertEqual(box_text(body[2]), "│")

    def test_border_box_title_too_long_is_truncated(self):
        root = built(border.BorderBox(title="a very long title indeed",
                                      width=8, height=3))
        top = row_text(root["children"][0])
        self.assertEqual(builder.display_width(top), 8)
        self.assertIn(" a ver", top)
        self.assertTrue(top.endswith("┐"))

    def test_border_box_title_cjk_safe(self):
        root = built(border.BorderBox(title="终端列表", width=9, height=3))
        top = row_text(root["children"][0])
        self.assertEqual(builder.display_width(top), 9)
        self.assertTrue(top.endswith("┐"))

    def test_border_box_zero_size_degrades(self):
        child = builder.text("body")
        root = built(border.BorderBox(child=child))
        self.assertEqual(root["size"][:2], (6, 3))
        tiny = built(border.BorderBox(child=child, width=1, height=1))
        self.assertEqual(len(tiny["children"]), 1)
        self.assertEqual(row_text(tiny), "body")
        empty = built(border.BorderBox(width=0, height=0))
        self.assertEqual(empty["size"][:2], (2, 2))

    def test_border_box_style_defaults_and_child(self):
        child = builder.text("x").height(1).flex(1)
        root = built(border.BorderBox(child=child, title="T", width=6, height=3,
                                      style="fg:#ff0000", title_style="accent"))
        top = root["children"][0]["children"]
        self.assertEqual(top[0].get("style"), "fg:#ff0000")
        self.assertEqual(top[1].get("style"), "accent")
        default = built(border.BorderBox(width=4, height=2))
        self.assertEqual(default["children"][0]["children"][0].get("style"),
                         tokens.STYLE_BORDER)


class TestDistributeAndSplitLayout(unittest.TestCase):
    def test_distribute(self):
        cases = [
            (10, [1, 1], [5, 5]),
            (10, [1, 2], [3, 7]),
            (5, [1, 1, 1], [1, 1, 3]),
            (2, [1, 1, 1], [1, 1, 1]),
            (0, [1, 1], [1, 1]),
            (7, None, []),
            (7, [1], [7]),
        ]
        for avail, ratios, want in cases:
            self.assertEqual(layout.distribute(avail, ratios), want,
                             "distribute(%r, %r)" % (avail, ratios))

    def test_split_layout_rects(self):
        split = layout.SplitLayout(orient="row", weights=[1, 2], gap=1)
        panes, dividers = split.rects(10, 3)
        self.assertEqual(len(panes), 2)
        self.assertEqual(len(dividers), 1)
        self.assertEqual(panes[0], layout.Rect(x=0, y=0, w=3, h=3))
        self.assertEqual(dividers[0], layout.Rect(x=3, y=0, w=1, h=3))
        self.assertEqual((panes[1].x, panes[1].w), (4, 6))

        stacked = layout.SplitLayout(orient="col", weights=[1, 1], gap=1)
        panes, dividers = stacked.rects(4, 5)
        self.assertEqual(panes[0].h, 2)
        self.assertEqual(dividers[0].y, 2)
        self.assertEqual(panes[1].y, 3)

    def test_split_layout_defaults(self):
        split = layout.SplitLayout()
        self.assertEqual(split.axis(), "row")
        self.assertEqual(split.gap_width(), 1)
        panes, dividers = split.rects(7, 2)
        self.assertEqual(panes, [layout.Rect(x=0, y=0, w=7, h=2)])
        self.assertEqual(dividers, [])


class TestChromeWidgets(unittest.TestCase):
    def test_title_bar(self):
        title = layout.TitleBar(
            id="title", width=20,
            left=[basics.Segment(text=" pane ")],
            buttons=[basics.Button(id="btn:close", text="x",
                                   input=["mouse"])],
        )
        self.assertEqual(len(title.line()), 20)
        box = built(title)
        self.assertEqual(box.get("id"), "title")
        self.assertGreater(len(box.get("children") or []), 0)

    def test_footer(self):
        footer = layout.Footer(
            id="footer", width=30, has_badge=True,
            badge=basics.Segment(text=" CTRL "),
            groups=[basics.Segment(text=" P PANE"),
                    basics.Segment(text=" T TAB")],
            right=[basics.Segment(text=" ws:main ")],
        )
        self.assertEqual(builder.display_width(footer.line()), 30)
        self.assertEqual(built(footer).get("id"), "footer")

    def test_picker(self):
        picker = layout.Picker(
            id="picker", title="Terminals", width=24,
            rows=[
                layout.PickerRow(text="term-1", id="picker:0", selectable=True),
                layout.PickerRow(text="term-2", id="picker:1", selected=True,
                                 selectable=True),
            ],
        )
        box = built(picker)
        self.assertEqual(box.get("id"), "picker")
        self.assertGreaterEqual(len(box.get("children") or []), 3)
        selected = find_box(box, "picker:1")
        self.assertIsNotNone(selected)
        self.assertTrue(box_text(selected).startswith("▸ "))
        unselected = find_box(box, "picker:0")
        # Go's Frame pads every row to the inner width (24 - 2 = 22).
        self.assertEqual(box_text(unselected), "  term-1" + " " * 14)
        self.assertEqual(builder.display_width(box_text(unselected)), 22)

    def test_floating_layer(self):
        floating = layout.FloatingLayer(id="float-1", title="float", x=3, y=2,
                                        width=20, height=6,
                                        rows=[basics.FrameRow(text="body")])
        float_box = built(floating)
        self.assertEqual(float_box.get("pos"), (3, 2))
        collapsed = layout.FloatingLayer(id="float-1", title="float", width=20,
                                         height=6, collapsed=True)
        self.assertEqual(built(collapsed)["size"][1], 1)

    def test_toast(self):
        toast = layout.Toast(id="toast", text="saved", style="fg:#fff")
        box = built(toast)
        self.assertEqual(box.get("id"), "toast")
        self.assertEqual(box.get("style"), "fg:#fff")
        self.assertEqual(box_text(box), "saved")


class TestBasicsWidgets(unittest.TestCase):
    def test_tab_bar_structure(self):
        bar = basics.TabBar(
            left=basics.Segment(text=" local ", style="chrome", id="workspace"),
            items=[basics.TabItem(id="tab:0", title="1:1", active=True),
                   basics.TabItem(id="tab:1", title="2:2")],
            plus=True,
            plus_id="tab:new",
        )
        root = bar.build().id("header").height(1).build()

        workspace = find_box(root, "workspace")
        self.assertIsNotNone(workspace)
        self.assertEqual(workspace.get("style"), "chrome")
        active = find_box(root, "tab:0")
        self.assertEqual(active.get("style"), basics.DEFAULT_ACTIVE_STYLE)
        self.assertEqual(box_text(active), "[1:1]")
        self.assertEqual(active.get("input"), ["mouse"])
        inactive = find_box(root, "tab:1")
        self.assertEqual(inactive.get("style"), basics.DEFAULT_INACTIVE_STYLE)
        self.assertEqual(box_text(inactive), " 2:2 ")
        plus = find_box(root, "tab:new")
        self.assertEqual(box_text(plus), " + ")

    def test_status_bar_structure_and_alignment(self):
        bar = basics.StatusBar(
            left=[basics.Segment(text="NORMAL", style="status")],
            right=[basics.Segment(text="tab 1/2", style="status"),
                   basics.Segment(text="12:00", style="muted")],
            width=30,
        )
        root = built(bar)
        children = root["children"]
        self.assertEqual(len(children), 6)
        self.assertEqual(box_text(children[0]), "NORMAL")
        self.assertEqual(box_text(children[1]), " " + basics.DEFAULT_SEPARATOR + " ")
        spacer = box_text(children[2])
        self.assertGreater(builder.display_width(spacer), 0)
        right = box_text(children[3]) + box_text(children[4]) + box_text(children[5])
        joined = ("NORMAL" + " " + basics.DEFAULT_SEPARATOR + " " + spacer + right)
        self.assertEqual(builder.display_width(joined), 30)
        self.assertEqual(
            bar.text(),
            "NORMAL " + basics.DEFAULT_SEPARATOR + " tab 1/2 "
            + basics.DEFAULT_SEPARATOR + " 12:00")

    def test_status_bar_trims_left_first(self):
        bar = basics.StatusBar(
            left=[basics.Segment(text="LONGHINTS", style="muted"),
                  basics.Segment(text="extra", style="muted")],
            right=[basics.Segment(text="12:00", style="muted")],
            width=12,
        )
        root = built(bar)
        children = root["children"]
        text = "".join(box_text(child) for child in children)
        self.assertLessEqual(builder.display_width(text), 12)
        self.assertIn("12:00", box_text(children[-1]))

    def test_card_centers_and_borders(self):
        card = basics.Card(
            id="slot-1", title="空槽",
            lines=["Ctrl-F 选择终端", "Ctrl-P 面板命令"],
            width=20, height=8, style=basics.DEFAULT_BORDER_STYLE, center=True,
        )
        root = built(card)
        self.assertEqual(root["size"][:2], (20, 8))
        children = root["children"]
        self.assertEqual(len(children), 8)
        top = box_text(children[0])
        self.assertIn("空槽", top)
        self.assertTrue(top.startswith("┌─"))
        self.assertTrue(top.endswith("┐"))
        self.assertEqual(children[0].get("style"), basics.DEFAULT_BORDER_STYLE)
        self.assertEqual(box_text(children[7]), "└" + "─" * 18 + "┘")
        row = children[3]["children"]
        self.assertEqual(len(row), 3)
        self.assertEqual(box_text(row[0]), "│")
        self.assertEqual(box_text(row[2]), "│")
        hint = box_text(row[1])
        self.assertIn("Ctrl-F 选择终端", hint)
        self.assertTrue(hint.startswith(" "))
        self.assertEqual(builder.display_width(hint), 18)

    def test_divider_builds_gutter_run(self):
        vertical = built(basics.Divider(id="divider:0", vertical=True, length=4,
                                        style="fg:#aabbcc", input=["mouse"]))
        self.assertEqual(vertical.get("id"), "divider:0")
        self.assertEqual(box_text(vertical), "││││")
        self.assertEqual(vertical["size"][:2], (1, 4))
        self.assertEqual(vertical.get("style"), "fg:#aabbcc")
        self.assertEqual(vertical.get("input"), ["mouse"])
        horizontal = built(basics.Divider(length=3))
        self.assertEqual(box_text(horizontal), "───")
        self.assertEqual(horizontal["size"][1], 1)
        self.assertEqual(horizontal.get("style"), basics.DEFAULT_SEP_STYLE)

    def test_frame_draws_chrome_and_row_ids(self):
        frame = basics.Frame(
            id="overlay:x", title="terminals", width=20, height=5,
            style="fg:#ffffff",
            rows=[basics.FrameRow(text="select", id="pick:0", input=["mouse"])],
        )
        root = built(frame)
        children = root["children"]
        self.assertEqual(len(children), 5)
        top = box_text(children[0])
        self.assertIn("terminals", top)
        self.assertEqual(children[0].get("style"), "fg:#ffffff")
        self.assertEqual(box_text(children[4]), "└" + "─" * 18 + "┘")
        row = find_box(root, "pick:0")
        self.assertIsNotNone(row)
        self.assertEqual(row["size"][:2], (18, 1))
        self.assertEqual(row.get("input"), ["mouse"])

    def test_button_hot_key_and_click_handling(self):
        button = basics.Button(id="btn:new", text="New terminal", hot="N")
        box = button.build().build()
        self.assertEqual(box.get("id"), "btn:new")
        children = box["children"]
        self.assertEqual(len(children), 2)
        self.assertEqual(box_text(children[0]), "N")
        self.assertEqual(children[0].get("style"), basics.DEFAULT_BUTTON_STYLE)
        self.assertEqual(box_text(children[1]), "ew terminal")

        container = builder.row(button.build(), builder.text("   ")).width(20).build()
        first = container["children"][0]
        self.assertEqual(first.get("id"), "btn:new")
        self.assertEqual(first.get("input"), ["mouse"])

    def test_key_hint_segments(self):
        hint = basics.KeyHint(mode="SCROLL",
                              keys=["PgUp/PgDn scroll", "y copy"],
                              id="keyhint")
        root = built(hint)
        children = root["children"]
        self.assertEqual(len(children), 3)
        self.assertEqual(box_text(children[0]), "SCROLL")
        self.assertEqual(children[0].get("style"), basics.DEFAULT_STATUS_STYLE)
        self.assertEqual(box_text(children[2]), "PgUp/PgDn scroll · y copy")
        self.assertEqual(
            hint.text(),
            "SCROLL " + basics.DEFAULT_SEPARATOR + " PgUp/PgDn scroll · y copy")


class TestProgressWidgets(unittest.TestCase):
    def test_progress_bar_fraction_and_bar(self):
        bar = progress.ProgressBar(value=50, max=200, width=10, label="load ",
                                   style="fill", track_style="track",
                                   percent_style="pct")
        self.assertEqual(bar.fraction(), 0.25)
        self.assertEqual(bar.percent(), 25)
        text = bar.bar_text()
        self.assertEqual(len(text), 10)
        self.assertEqual(text.count("█"), 3)
        children = built(bar)["children"]
        self.assertEqual(len(children), 4)
        self.assertEqual(box_text(children[0]), "load ")
        self.assertEqual(children[0].get("style"), "")
        self.assertEqual(box_text(children[1]), "███")
        self.assertEqual(children[1].get("style"), "fill")
        self.assertEqual(box_text(children[2]), "░" * 7)
        self.assertEqual(children[2].get("style"), "track")
        self.assertEqual(box_text(children[3]), "25%")
        self.assertEqual(children[3].get("style"), "pct")
        self.assertEqual(len(progress.ProgressBar().bar_text()),
                         progress.DEFAULT_PROGRESS_WIDTH)

    def test_progress_bar_clamps_and_rounds(self):
        cases = [
            (-5, 10, 0),
            (30, 10, 100),
            (1, 3, 33),
            (2, 3, 67),
            (0, 0, 0),
            (5, -1, 0),
        ]
        for value, maximum, percent in cases:
            bar = progress.ProgressBar(value=value, max=maximum)
            self.assertEqual(bar.percent(), percent,
                             "percent(%d/%d)" % (value, maximum))
        hidden = built(progress.ProgressBar(value=3, max=4, hide_percent=True))
        self.assertEqual(len(hidden["children"]), 2)

    def test_spinner_frame_and_label(self):
        spinner = progress.Spinner(frame=11, frames=["a", "b", "c"], style="spin",
                                   label="working", label_style="muted")
        self.assertEqual(spinner.frame_text(), "c")
        self.assertEqual(progress.Spinner(frame=-1,
                                          frames=["a", "b", "c"]).frame_text(), "c")
        self.assertNotEqual(progress.Spinner().frame_text(), "")
        children = built(spinner)["children"]
        self.assertEqual(len(children), 2)
        self.assertEqual(box_text(children[0]), "c")
        self.assertEqual(children[0].get("style"), "spin")
        self.assertEqual(box_text(children[1]), "working")
        self.assertEqual(children[1].get("style"), "muted")

    def test_badge_and_tags(self):
        badge = built(progress.Badge(id="badge:1", text="3", style="accent",
                                     input=["mouse"]))
        self.assertEqual(badge.get("id"), "badge:1")
        self.assertEqual(box_text(badge), "3")
        self.assertEqual(badge.get("style"), "accent")
        self.assertEqual(badge.get("input"), ["mouse"])

        tags = built(progress.Tags(id="tags", style="base", sep_style="sep",
                                   items=[
                                       progress.Tag(text="go", id="tag:go"),
                                       progress.Tag(text="ui", style="hot"),
                                       progress.Tag(text="cjk 中文"),
                                   ]))
        children = tags["children"]
        self.assertEqual(len(children), 5)
        self.assertEqual(box_text(children[0]), "go")
        self.assertEqual(children[0].get("style"), "base")
        self.assertEqual(children[0].get("id"), "tag:go")
        self.assertEqual(children[2].get("style"), "hot")
        self.assertEqual(builder.display_width(row_text(tags)), 14)


class TestTheme(unittest.TestCase):
    def test_theme_variants(self):
        self.assertEqual(theme.default_theme(), theme.dark_theme())
        self.assertNotEqual(theme.dark_theme(), theme.light_theme())
        _, ok = theme.theme_by_name("light")
        self.assertTrue(ok)
        _, ok = theme.theme_by_name("neon")
        self.assertFalse(ok)

        dark = theme.dark_theme()
        for name in ("title", "accent", "muted", "danger", "success",
                     "warning", "border", "border_focus", "selection",
                     "selection_text", "placeholder", "cursor", "status_bar",
                     "tab_active", "tab_inactive", "header", "footer", "toast",
                     "overlay", "backdrop", "marker", "separator", "zebra"):
            self.assertNotEqual(getattr(dark, name), "", name)
        light = theme.light_theme()
        for name in ("text", "accent", "selection", "overlay"):
            self.assertNotEqual(getattr(light, name), "", name)

    def test_themed_visual_constructors(self):
        current = theme.light_theme()
        bar = theme.themed_progress_bar(current)
        self.assertEqual(bar.style, current.accent)
        self.assertEqual(bar.track_style, current.muted)
        spinner = theme.themed_spinner(current)
        self.assertEqual(spinner.style, current.accent)
        badge = theme.themed_badge(current)
        self.assertEqual(badge.style, current.accent)
        tags = theme.themed_tags(current)
        self.assertEqual(tags.style, current.text)
        self.assertEqual(tags.sep_style, current.separator)

    def test_theme_is_overridable(self):
        previous = theme.DEFAULT_THEME_VALUE
        try:
            theme.DEFAULT_THEME_VALUE = theme.light_theme()
            self.assertEqual(theme.default_theme().name, "light")
        finally:
            theme.DEFAULT_THEME_VALUE = previous


if __name__ == "__main__":
    unittest.main()
