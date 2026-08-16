# Governance

AnyTTY currently uses a maintainer-led governance model appropriate for a pre-release project.

## Roles

Contributors propose changes through issues and pull requests. Maintainers review technical direction, security boundaries, releases, repository access, and community enforcement. The current maintainer team is represented by the repository `CODEOWNERS` file.

## Decisions

Routine changes are decided in pull request review. Material protocol, trust-boundary, compatibility, dependency, or governance changes should begin with a public issue and record the accepted rationale in code or stable documentation. Maintainers seek consensus; when consensus is not practical, the responsible code owner makes the decision and documents it.

At least one approving maintainer is required before merge. Changes to authentication, authorization, pairing, public protocols, release automation, or this governance policy should receive explicit review from the repository owner. A contributor may not approve their own change.

## Releases and security

Maintainers decide when a commit is release-ready using `RELEASE_CHECKLIST.md`. Security reports follow `SECURITY.md`; embargoed details remain private until coordinated disclosure is appropriate.

## Evolution

As the contributor base grows, the project may introduce additional maintainers and a documented nomination process. Until then, sustained high-quality contribution and sound judgment around user safety are prerequisites for expanded repository permissions.
