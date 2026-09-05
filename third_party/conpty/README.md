# Windows Terminal ConPTY runtime

AnyTTY embeds the matching Windows Terminal ConPTY runtime into each Windows
executable so Windows 10 receives the same lossless VT passthrough used on
newer Windows releases. At runtime the files are verified and materialized in
the current user's AnyTTY cache before loading.

- Runtime version: `1.25.260303002`
- Source package: `microsoft/node-pty` commit `2beb5d6ef2d9cc5e76cfd9dafc172e35aa3cbeb1`
- Upstream project: https://github.com/microsoft/terminal
- License: MIT; see `LICENSE`

SHA-256:

| Target | File | SHA-256 |
| --- | --- | --- |
| windows-amd64 | `conpty.dll` | `3319b484b80bb53d1f4d0a9eb0ea60fd0f61da69db7280ca43b84215f19245ff` |
| windows-amd64 | `OpenConsole.exe` | `7f68c840226505004215c0b82d4e502c24b5bc3f4b93c4baaaa19bd679c0def8` |
| windows-arm64 | `conpty.dll` | `b3a9f975c51d5b9b96d290bf784e990e3f85203b063c21614457353f566266ff` |
| windows-arm64 | `OpenConsole.exe` | `7fa560be0c9b6c81db5d1b855a4a6e0f3ee5a2ea75d36eebb3ae44443b8d4866` |
