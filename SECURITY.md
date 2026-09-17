# Security Policy

## Supported Versions

Degubase is actively developed. We provide security patches and bug fixes for the current release and main branch.

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |
| < 0.1   | :x:                |

## Reporting a Vulnerability

The Degubase team takes the security of our software seriously. If you believe you have found a security vulnerability in Degubase, please report it to us as described below.

**Please do not report security vulnerabilities through public GitHub issues, discussions, or pull requests.**

### How to Report

Please send an email directly to [christoph.walcher@gmail.com](mailto:christoph.walcher@gmail.com) with the subject line `[SECURITY] Degubase Vulnerability Report` or use the private **GitHub Security Advisory** reporting feature on GitHub.

Please include as much of the following information as possible:
- Type of vulnerability (e.g., authentication bypass, injection, directory traversal, privilege escalation).
- Detailed steps to reproduce the issue (including sample payload, configuration, or curl commands).
- The version or commit hash of Degubase you tested on.
- Any potential impact on users, data, or host systems.
- Whether you have identified a mitigation or fix.

### Response Timeline

- **Acknowledgment**: We will acknowledge receipt of your vulnerability report within 48 hours.
- **Assessment**: We will evaluate the impact and confirm the vulnerability within 5 business days.
- **Resolution**: We will work on a fix in private and keep you informed of our progress.
- **Disclosure**: Once a patch is released, we will publicly credit you for the discovery (unless you prefer to remain anonymous).

## Security Considerations for Degubase

When deploying Degubase, please keep the following architecture details in mind:

- **Single-user / Local Mode**: When running with `disable_auth: true`, any client with network access to the Degubase port can perform administrative operations. Ensure port 8080 is only bound locally or protected behind a reverse proxy.
- **Lua Script Execution**: Degubase features an embedded Lua engine. While scripts operate in restricted execution contexts, scripts written by workspace owners can modify database records. Workspace access should only be granted to trusted collaborators.
- **Workspace Tokens**: Tokens are hashed with SHA-256 before storage, but must be kept confidential by clients.
