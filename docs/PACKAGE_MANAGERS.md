# Package manager publishing

The repository contains release-ready source files for three package channels. They are not considered published until the corresponding external repository or registry accepts them.

## Homebrew

`packaging/homebrew/Formula/anytty.rb` installs the four macOS/Linux archives from GitHub Releases. During beta, publish it from an `anytty/homebrew-tap` repository and install it with:

```sh
brew install anytty/tap/anytty
```

After copying the formula into the tap repository, run:

```sh
brew tap anytty/tap
brew audit --strict anytty/tap/anytty
brew install anytty/tap/anytty
brew test anytty/tap/anytty
```

Homebrew core requires a stable tagged release, so the beta belongs in a project tap rather than `homebrew/core`.

## npm

`packaging/npm` is a small `@anytty/cli` package. Its postinstall script selects the current OS and architecture, downloads the matching GitHub Release archive, and checks SHA-256 before exposing the `anytty` command.

```sh
cd packaging/npm
npm test
npm pack --dry-run
npm publish --access public
```

Publishing requires control of the `@anytty` npm organization and npm publishing authentication.

## WinGet

`packaging/winget/0.0.1-beta.0` contains a multi-file manifest for both Windows x64 and ARM64 portable archives. Validate it on Windows before opening a single-version pull request to `microsoft/winget-pkgs`:

```powershell
winget validate --manifest packaging\winget\0.0.1-beta.0
winget install --manifest packaging\winget\0.0.1-beta.0
```

Use `wingetcreate submit` or place the files under `manifests/a/AnyTTY/AnyTTY/0.0.1-beta.0/` in a fork of `microsoft/winget-pkgs`. GitHub Release URLs are version-specific and the manifest hashes match the published beta archives.
