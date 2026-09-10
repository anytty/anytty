#!/usr/bin/env python3
"""Verify that an iOS executable exports every symbol used by Dart FFI."""

import argparse
from pathlib import Path
import re
import subprocess
import sys


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("executable", type=Path, help="Path to Runner.app/Runner")
    args = parser.parse_args()
    bindings = (
        Path(__file__).resolve().parents[1]
        / "clients/flutter/lib/src/native/generated"
    )
    expected = set()
    for name in ("anytty_client_bindings.dart", "anytty_terminal_input_bindings.dart"):
        expected.update(re.findall(r"'(anytty_[a-z0-9_]+)'", (bindings / name).read_text()))
    if not expected:
        parser.error("No Dart FFI symbols found")
    # Read the dynamic export trie, not just nm's static symbol table: these
    # exports are what DynamicLibrary.process().lookup() can actually resolve.
    result = subprocess.run(
        ["xcrun", "dyld_info", "-exports", str(args.executable)],
        capture_output=True,
        text=True,
    )
    if result.returncode:
        sys.exit(result.stderr or result.stdout)
    exported = set(re.findall(r"\b_(anytty_[a-z0-9_]+)\b", result.stdout))
    missing = sorted(expected - exported)
    if missing:
        sys.exit("Missing iOS FFI exports:\n" + "\n".join(missing))
    print(f"Verified {len(expected)} iOS FFI exports in {args.executable}")


if __name__ == "__main__":
    main()
