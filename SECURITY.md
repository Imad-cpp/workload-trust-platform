# Security Policy

This project is in pre-release development and is not yet suitable for production use.

## Reporting security issues

Do not publish exploitable security details in a public issue. Use GitHub's private security reporting mechanism once enabled for the repository. Until the repository is configured, no public vulnerability-reporting address is claimed here.

## Security design

The release-blocking security invariants are documented in [`docs/03_SECURITY_INVARIANTS.md`](docs/03_SECURITY_INVARIANTS.md). The threat model is in [`docs/02_THREAT_MODEL.md`](docs/02_THREAT_MODEL.md).

## Cryptography

The project does not intend to implement custom cryptographic primitives. Workload identity is built on SPIFFE/SPIRE and established TLS/X.509 libraries.
