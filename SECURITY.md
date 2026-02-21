# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability in bc, please report it responsibly.

### How to Report

1. **GitHub Security Advisories** (Preferred): Use [GitHub's Security Advisory feature](../../security/advisories/new) to report vulnerabilities privately.

2. **Email**: Send details to the repository maintainers via GitHub.

### What to Include

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Any suggested fixes (optional)

### What to Expect

- **Acknowledgment**: Within 48 hours
- **Initial Assessment**: Within 7 days
- **Resolution Timeline**: Depends on severity
  - Critical: 24-48 hours
  - High: 7 days
  - Medium: 30 days
  - Low: 90 days

### Disclosure Process

1. Report received and acknowledged
2. Vulnerability confirmed and assessed
3. Fix developed and tested
4. Security advisory published with fix
5. Public disclosure after patch is available

### Scope

This policy applies to:
- The bc CLI tool
- The TUI interface
- Agent communication protocols
- Configuration handling

### Out of Scope

- Issues in third-party dependencies (report to upstream)
- Social engineering attacks
- Physical security issues

## Security Best Practices

When using bc:

- Keep bc updated to the latest version
- Review agent prompts before execution
- Use environment variables for sensitive data (never hardcode)
- Restrict agent capabilities to minimum required
- Monitor agent costs and set appropriate budgets
