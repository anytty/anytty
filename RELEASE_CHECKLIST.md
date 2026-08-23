# Release Checklist

This checklist is for maintainers preparing an AnyTTY public release from `anytty/anytty`.

## Source and boundary

- [ ] Select a reviewed commit from the public `anytty/anytty` repository and confirm `VERSION` matches the intended tag.
- [ ] Confirm the repository contains only reviewed user-device source and public documentation, with no private service implementation, production configuration, credentials, or private history.
- [ ] Run `npm run public:check` in the public repository.
- [ ] Review generated files, screenshots, fixtures, and notices for secrets and personal data.

## Quality

- [ ] Run `make test`, `make test-clients`, Android validation, and the supported iOS build check.
- [ ] Build the CLI/TUI and verify official Cloud client capability remains compiled in.
- [ ] Build the Pages artifact and inspect desktop, phone, keyboard, and reduced-motion behavior.
- [ ] Confirm README, documentation, issue forms, and security links resolve.
- [ ] Review `CHANGELOG.md`, supported platforms, known limitations, and upgrade notes.

## Legal and supply chain

- [ ] Confirm Apache-2.0 `LICENSE`, `NOTICE`, DCO, trademark policy, and copyright years.
- [ ] Regenerate and review Go, npm, Android, iOS, font, and pinned third-party notices.
- [ ] Review Dependabot and CI results; investigate known vulnerabilities and document accepted risk.
- [ ] Produce checksums and provenance for every release artifact actually published.

## GitHub and publication

- [ ] Enable private vulnerability reporting and verify the maintainer notification path.
- [ ] Configure GitHub Pages to use GitHub Actions and verify the published base path `/anytty/`.
- [ ] Review branch protection, required checks, CODEOWNERS, Discussions, issue permissions, and Actions permissions.
- [ ] Decide whether the existing public history is suitable; only then consider a reviewed clean-history initialization.
- [ ] Create a signed tag; verify the Release workflow publishes only to `anytty/anytty` and produces all expected checksums and attestations.
- [ ] Publish only after a second maintainer or designated reviewer signs off on the boundary audit.
