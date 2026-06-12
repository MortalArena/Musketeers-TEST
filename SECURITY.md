# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

**Please do NOT open a public issue for security vulnerabilities.**

Instead, please email us at: **security@musketeers.dev**

Include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

## Response Timeline

- **Acknowledgment:** Within 48 hours
- **Initial assessment:** Within 7 days
- **Fix deployment:** Depends on severity
  - Critical: 24-72 hours
  - High: 1 week
  - Medium: 2 weeks
  - Low: 1 month

## Security Features

Musketeers implements multiple layers of security:

### Cryptography
- Ed25519 signatures
- AES-256-GCM encryption
- NaCl box for E2E messaging
- scrypt key derivation

### Access Control
- ABAC (Attribute-Based Access Control)
- Multi-level approval system
- Domain separation for signatures
- Nonce-based replay protection

### Infrastructure
- Encrypted vault for secrets
- Pluggable key providers (OS Keychain, HSM, KMS)
- Rate limiting on all operations
- Memory-safe bounded caches

### Network
- Commit-reveal for domain registration
- Homograph attack protection
- Fail-closed revocation checks
- TLS for HTTP gateway

## Security Best Practices for Users

1. **Never share your mnemonic** (24-word backup phrase)
2. **Use strong passphrases** for keystore encryption
3. **Rotate keys regularly** for private channels
4. **Monitor audit logs** for suspicious activity
5. **Keep software updated** to the latest version
6. **Use OS keychain** for production deployments
7. **Enable rate limiting** in production
8. **Review policies** before granting capabilities

## Bug Bounty Program

We are planning a bug bounty program. Stay tuned for announcements!

## Security Audits

Musketeers undergoes regular security audits. Audit reports will be published in the `audits/` directory.
