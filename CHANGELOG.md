# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-07-23

### Added

- **In-Memory SSH Agent Integration:** Securely injects decrypted private keys directly into `ssh-agent`.
- **OS Keyring Master Key:** Native integration with Secret Service API / Keychain.
- **AES-256-GCM Vault:** Military-grade encryption for all host data and private keys.
- **Interactive Prompts:** Built-in terminal UI to add and manage servers (`shield add server`).
- **Arch Linux Support:** Native `PKGBUILD` for smooth installation via `makepkg`.
- **Makefile:** Cross-compilation support for Linux and macOS (amd64 and arm64).
