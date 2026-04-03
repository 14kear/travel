# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability, please report it via [GitHub Security Advisories](https://github.com/MikkoParkkola/trvl/security/advisories/new).

Do NOT open a public issue for security vulnerabilities.

## Scope

trvl accesses Google's public-facing internal APIs using Chrome TLS fingerprint impersonation. It does not:
- Store or transmit user credentials
- Access authenticated Google accounts
- Bypass rate limits or access controls
- Cache personal data

## Dependencies

- `github.com/refraction-networking/utls` — TLS fingerprint impersonation
- `github.com/spf13/cobra` — CLI framework
- `golang.org/x/time` — Rate limiting

All dependencies are reviewed and pinned in go.sum.
