#!/usr/bin/env python3
"""anytty 展示树程序最小示例（非 Go 语言）。

宿主的世界里只有盒子（rect/div）：程序声明盒子怎么摆、画什么、有没有边框。
宿主通过 stdin 发 NDJSON 事件，本程序通过 stdout 写展示树 JSON。
"""
import json
import sys

SCHEMA_VERSION = 1


class App:
    def __init__(self):
        self.cols = 80
        self.rows = 24
        self.rev = 0
        self.echo = []

    def handle(self, ev):
        if ev.get("type") == "resize":
            self.cols = ev.get("cols") or self.cols
            self.rows = ev.get("rows") or self.rows
        elif ev.get("type") == "key":
            char = ev.get("char")
            if char:
                self.echo.append(char)
            elif ev.get("key") == "tab":
                self.echo.append(" ")

    def view(self):
        self.rev += 1
        line = "".join(self.echo) or "(type something)"
        return {
            "version": SCHEMA_VERSION,
            "rev": self.rev,
            "root": {
                "flow": "col",
                "children": [
                    {"id": "header", "size": {"height": 1},
                     "content": {"text": " python program · any language works ", "style": "header"}},
                    {"id": "body", "size": {"flex": 1}, "flow": "col",
                     "border": {"title": "hello"},
                     "children": [
                         {"content": {"text": line}},
                         {"content": {"text": "宿主只渲染盒子；逻辑全在 Python 里。", "style": "muted"}},
                     ]},
                    {"id": "footer", "size": {"height": 1},
                     "content": {"text": " anytty-program-harness --program examples/program-hello.py  │  ctrl-c quit",
                                 "style": "footer"}},
                ],
            },
        }


def main():
    app = App()
    sys.stdout.write(json.dumps(app.view()) + "\n")
    sys.stdout.flush()
    for raw in sys.stdin:
        raw = raw.strip()
        if not raw:
            continue
        try:
            event = json.loads(raw)
        except json.JSONDecodeError:
            continue
        app.handle(event)
        sys.stdout.write(json.dumps(app.view()) + "\n")
        sys.stdout.flush()


if __name__ == "__main__":
    main()
