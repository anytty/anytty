# Security Policy

AnyTTY is in beta. The current beta receives best-effort security fixes, but there is no vulnerability bounty or guaranteed response time.

Please use [GitHub private vulnerability reporting](https://github.com/anytty/anytty/security/advisories/new) for security issues. Do not
open a public issue containing exploit details, credentials, pairing material,
terminal content, or private user data.

The daemon is the final authority for terminal, file, device identity, and
client permissions. A connection service can provide reachability but cannot
grant terminal or file capabilities. Review
[`docs/SECURITY_BOUNDARY.md`](docs/SECURITY_BOUNDARY.md) for the user-visible trust model.

Reports should include the affected version, prerequisites, a minimal
reproduction using generated test credentials, impact, and known mitigations.

Maintainers will acknowledge and triage reports as capacity permits, coordinate remediation privately, and agree on disclosure timing with the reporter. Do not assume a fix or disclosure date until a maintainer confirms it. General support and service account questions belong in the channels listed in `SUPPORT.md`.
