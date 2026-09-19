"""TUI v2 Python SDK: core client, view builder and program-side chrome widgets.

    from tui2sdk import Client, App, builder

    class MyProgram(App):
        def on_hello(self, hello):
            self.client.commit(builder.text("hello").build(), ["ctrl-p"])

    Client(sys.stdin.buffer, sys.stdout.buffer).run(MyProgram())

Layers:
  - ``tui2sdk.wire``     protobuf payloads + binary frames (PB-01);
  - ``tui2sdk.core``     Client/App: event loop, typed events, Emit/Commit;
  - ``tui2sdk.builder``  chainable box builder + text metrics;
  - ``tui2sdk.widgets``  program-side chrome (card/tab bar/footer/picker/
                         floating/split tree/toast).
"""

from . import builder, wire
from .core import App, Client
from .wire import WireError

__all__ = ["App", "Client", "WireError", "builder", "wire"]
