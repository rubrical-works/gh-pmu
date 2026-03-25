 # Security Policy

  ## Supported Versions

  | Version | Supported          |
  |---------|--------------------|
  | Latest minor release | :white_check_mark: |
  | Previous releases    | :x:                |

  Only the latest minor release receives security patches. Upgrade to the latest version before reporting.

  ## Reporting a Vulnerability

  **Do not open a public issue for security vulnerabilities.**

  Use [GitHub Private Vulnerability Reporting](https://github.com/rubrical-works/gh-pmu/security/advisories/new) to submit a report. This keeps the details confidential until a fix is available.

  ### What to include

  - Description of the vulnerability
  - Steps to reproduce
  - Affected version(s)
  - Impact assessment (if known)

  ### Response Timeline

  - **Acknowledgment:** Within 48 hours
  - **Target patch:** Within 30 days
  - **Disclosure:** Coordinated with reporter after fix is released

  ## Scope

  ### In scope

  - Command injection via crafted input
  - Token or credential leakage (e.g., GitHub tokens exposed in logs or temp files)
  - Config file permission issues allowing unauthorized access
  - Dependency vulnerabilities in direct dependencies

  ### Out of scope

  - Issues requiring local access beyond normal CLI usage
  - Social engineering
  - Denial of service against GitHub APIs (rate limiting is GitHub's domain)
  - Vulnerabilities in dependencies not used by gh-pmu

  ## Security Measures

  This project uses the following automated security tooling:

  | Tool | Purpose |
  |------|---------|
  | [gosec](https://github.com/securego/gosec) | Static analysis for Go security issues |
  | [CodeQL](https://codeql.github.com/) | Semantic code analysis via GitHub |
  | `go vet` | Go source code analysis |
  | `golangci-lint` | Aggregated linter suite |

  All security scans run on every push and pull request via GitHub Actions.
