# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0-beta] - 2026-04-06

### Added

- `shield setup` - Initialize vault and generate master key
- `shield add` - Add SSH server credentials to vault
- `shield ls` - List all saved servers
- `shield connect` - Connect to a saved server via SSH

### Features

- Secure master key storage using system keyring
- AES-256-GCM encryption for vault data
- Support for SSH key-based authentication
- Cross-platform binaries (Linux, macOS)
- Architecture support: amd64, arm64
