#!/usr/bin/env python3
"""Entry point for ``tui2-sdk-verify --cmd "python3 clients/tui/sdk/python/conformance.py"``."""

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from tui2sdk.conformance import main  # noqa: E402

if __name__ == "__main__":
    sys.exit(main())
