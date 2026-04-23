# Security Policy

## Supported Versions
Only the latest release receives security fixes.

## Reporting a Vulnerability
Please **do not** open a public GitHub issue for security vulnerabilities.

Report privately via GitHub's Security Advisory feature:
https://github.com/Om-Rohilla/recall/security/advisories/new

You will receive a response within 72 hours. Critical vulnerabilities are
patched and released within 14 days.

## Scope
In-scope: vault key management, secret scrubbing gaps, shell hook injection,
FTS5 injection, export file format, WASM sandbox escapes.

Out of scope: Theoretical attacks requiring physical machine access.

## macOS Distribution

Recall binaries are not currently Apple-notarized (requires a paid Apple Developer
account). The install script automatically removes the Gatekeeper quarantine flag
via `xattr -d com.apple.quarantine`. If you install manually, run:

```sh
xattr -d com.apple.quarantine /usr/local/bin/recall
```

Full notarization is planned for a future release. The notarization workflow is
already prepared in `.github/workflows/release.yml` (commented out, awaiting
Apple Developer ID certificate).

## Secure Key Deletion

When migrating from legacy file-based key storage to the OS keyring, Recall
performs a multi-pass overwrite (zeros, ones, random) of the key file before
removal. This is best-effort: on copy-on-write filesystems (APFS, btrfs, ZFS)
and SSDs with hardware wear-leveling, physical data recovery may still be
possible at the hardware level.

Using the OS keyring (the default) avoids this issue entirely since no key file
is persisted to disk.

## Plugin Security

Recall executes plugins inside a WASM sandbox. The sandbox provides a minimal
set of WASI stubs: only `fd_write` (stdout), `proc_exit`, and ENOSYS stubs for
all other syscalls. Plugins cannot access the filesystem, environment variables,
network, clocks, or random numbers through the sandbox.

**Plugin signature verification is not implemented.** The `recall plugin install`
command requires `--accept-risk` to acknowledge this limitation. Only install
plugins from sources you trust explicitly.

## GitHub Sync Token Scope

For `recall sync`, the `RECALL_GITHUB_TOKEN` environment variable requires only
the `gist` OAuth scope. Do not grant broader scopes (repo, admin, etc.).
Prefer using `--token-file` over the environment variable to avoid token exposure
in `ps aux` output.
