# Contributing

AnyTTY welcomes focused fixes, tests, documentation, accessibility improvements, and proposals that preserve the documented security boundary. The project is pre-release, so discuss large changes in an issue before investing in an implementation.

Read `README.md`, `ARCHITECTURE.md`, and the relevant protocol document before
changing behavior. Keep schemas, generated code, implementations, tests, and
documentation consistent in the same change.

```sh
npm ci
make test
make test-clients
npm run public:check
```

Android release validation additionally requires Java 21 and an Android SDK.
Do not commit `.artifacts/`, credentials, pairing claims, enrollment codes,
terminal content, or generated platform build directories.

Use a `Signed-off-by` line as described in `DCO`. Report vulnerabilities
privately according to `SECURITY.md`; do not open a public issue containing
security-sensitive details.

## Pull requests

Keep each pull request to one reviewable behavior. Update schemas, generated code, implementation, tests, docs, and third-party notices together when the change crosses those boundaries. Describe user-visible behavior, security impact, validation performed, and screenshots for UI changes.

By contributing, you certify the Developer Certificate of Origin using `git commit -s`. Contributions are licensed under Apache-2.0. Maintainers may ask for changes, close work that conflicts with project scope, or defer behavior until its protocol and security model are clear.

Do not commit build outputs, credentials, pairing material, terminal content, private service source, production configuration, or personal data. Follow `CODE_OF_CONDUCT.md` in all project spaces.
