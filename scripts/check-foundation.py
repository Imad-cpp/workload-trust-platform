#!/usr/bin/env python3
from __future__ import annotations

from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parents[1]

REQUIRED_FILES = [
    "README.md",
    "SECURITY.md",
    "CHANGELOG.md",
    ".editorconfig",
    ".gitignore",
    "docs/00_PRODUCT.md",
    "docs/01_ARCHITECTURE.md",
    "docs/02_THREAT_MODEL.md",
    "docs/03_SECURITY_INVARIANTS.md",
    "docs/04_IDENTITY_MODEL.md",
    "docs/05_POLICY_MODEL.md",
    "docs/06_API_BOUNDARIES.md",
    "docs/07_V1_SCOPE.md",
    "docs/08_DEFINITION_OF_DONE.md",
    "docs/09_DECISIONS.md",
    "docs/10_ROADMAP.md",
    "docs/11_OPEN_QUESTIONS.md",
    "docs/12_DATA_MODEL.md",
    "docs/13_IDENTITY_LAB.md",
    "docs/14_PHASE1_EVIDENCE.md",
    "docs/15_CONTROL_PLANE_FOUNDATION.md",
    "docs/16_OPERATOR_AUTH_AND_RECONCILIATION.md",
    "docs/17_OPERATOR_MUTATION_EVIDENCE.md",
    "docs/adr/0001-go-core.md",
    "docs/adr/0002-spiffe-spire.md",
    "docs/adr/0003-modular-control-plane.md",
    "docs/adr/0004-postgresql-source-of-truth.md",
    "docs/adr/0005-v1-lab-attestation.md",
    "docs/adr/0006-read-only-loopback-api-before-auth.md",
    "docs/adr/0007-local-operator-bearer-auth.md",
    "docs/adr/0008-spire-registration-reconciliation.md",
    "docs/adr/0009-authorized-audited-registration-mutations.md",
]

REQUIRED_PHRASES = {
    "docs/00_PRODUCT.md": ["default-deny", "long-lived credentials"],
    "docs/02_THREAT_MODEL.md": ["attestation", "fail closed"],
    "docs/03_SECURITY_INVARIANTS.md": ["default-deny", "Authentication", "Authorization", "No custom cryptographic primitives", "fail closed"],
    "docs/04_IDENTITY_MODEL.md": ["SPIFFE", "X.509-SVID"],
    "docs/05_POLICY_MODEL.md": ["ALLOW", "DENY"],
    "docs/06_API_BOUNDARIES.md": ["loopback-only", "registrations:write", "expected_revision", "same PostgreSQL transaction"],
    "docs/07_V1_SCOPE.md": ["Docker", "Linux"],
    "docs/08_DEFINITION_OF_DONE.md": ["security", "test", "Server-side authorization"],
    "docs/13_IDENTITY_LAB.md": ["Phase 1 complete", "workload-trust.test", "Docker workload-attestation", "insecure_bootstrap", "12-second X.509-SVID TTL", "not a production"],
    "docs/14_PHASE1_EVIDENCE.md": ["Docker workload attestation", "automatic SVID rotation", "does not prove service authorization"],
    "docs/15_CONTROL_PLANE_FOUNDATION.md": ["loopback-only", "append-only", "does not complete Phase 2", "no workload mutation route exists"],
    "docs/16_OPERATOR_AUTH_AND_RECONCILIATION.md": ["constant-time", "wtp-rule:", "foreign", "does not prove service authorization"],
    "docs/17_OPERATOR_MUTATION_EVIDENCE.md": ["transactionally audited", "viewer", "409", "does not"],
    "docs/adr/0005-v1-lab-attestation.md": ["host-native SPIRE", "Docker-label attestation"],
    "docs/adr/0006-read-only-loopback-api-before-auth.md": ["read-only and loopback-only", "operator authentication/authorization"],
    "docs/adr/0007-local-operator-bearer-auth.md": ["loopback-only", "constant-time", "32 bytes"],
    "docs/adr/0008-spire-registration-reconciliation.md": ["wtp-rule:", "foreign", "not a cryptographic ownership proof"],
    "docs/adr/0009-authorized-audited-registration-mutations.md": ["registrations:write", "same PostgreSQL transaction", "expected_revision"],
}

LINK_RE = re.compile(r"\[[^\]]+\]\(([^)]+)\)")


def fail(message: str) -> None:
    print(f"ERROR: {message}", file=sys.stderr)
    raise SystemExit(1)


def check_required_files() -> None:
    missing = [path for path in REQUIRED_FILES if not (ROOT / path).is_file()]
    if missing:
        fail("missing required files: " + ", ".join(missing))


def check_required_phrases() -> None:
    for rel, phrases in REQUIRED_PHRASES.items():
        text = (ROOT / rel).read_text(encoding="utf-8")
        lower = text.lower()
        for phrase in phrases:
            if phrase.lower() not in lower:
                fail(f"{rel} is missing required phrase: {phrase!r}")


def check_local_markdown_links() -> None:
    root = ROOT.resolve()
    for path in ROOT.rglob("*.md"):
        text = path.read_text(encoding="utf-8")
        for target in LINK_RE.findall(text):
            target = target.split("#", 1)[0].strip()
            if not target or target.startswith(("http://", "https://", "mailto:")):
                continue
            resolved = (path.parent / target).resolve()
            try:
                resolved.relative_to(root)
            except ValueError:
                fail(f"{path.relative_to(ROOT)} links outside repository: {target}")
            if not resolved.exists():
                fail(f"broken local link in {path.relative_to(ROOT)}: {target}")


def check_readme_claim_boundary() -> None:
    readme = (ROOT / "README.md").read_text(encoding="utf-8")
    required = "No production deployment or production-readiness claim is made."
    if required not in readme:
        fail(f"README.md must keep the claim boundary: {required}")


def check_ephemeral_runtime_ignored() -> None:
    gitignore = (ROOT / ".gitignore").read_text(encoding="utf-8").splitlines()
    if ".lab/" not in {line.strip() for line in gitignore}:
        fail(".gitignore must exclude .lab/ because SPIRE runtime material is ephemeral and sensitive")


def main() -> None:
    check_required_files()
    check_required_phrases()
    check_local_markdown_links()
    check_readme_claim_boundary()
    check_ephemeral_runtime_ignored()
    print(f"foundation validation passed: {len(REQUIRED_FILES)} required files")


if __name__ == "__main__":
    main()
