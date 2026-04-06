# Shield CLI

A secure, fast, and interactive command-line interface written in Go to manage your SSH servers, and private keys.

Shield acts as a local encrypted vault for your infrastructure credentials. Instead of leaving private `.pem` or `id_rsa` files lying around in plaintext, Shield encrypts everything using **AES-GCM 256-bit** encryption. The master key is never stored in files; it is securely managed by your Operating System's native Keyring (Secret Service API via D-Bus, Keychain, or Credential Manager).

## Features

* **OS Keyring Integration:** Secure master key storage natively integrated with your OS (works flawlessly on Linux/Wayland, and macOS).
* **Military-Grade Encryption:** All host data, usernames, and private key contents are encrypted at rest using AES-GCM.
* **Interactive Prompts:** Beautiful and user-friendly terminal UI for adding resources, powered by `survey`.
* **Zero Plaintext Files:** Private keys are ingested into the encrypted vault, keeping your `~/.ssh/` directory clean and safe.

## Installation

### Pre-built Binaries

Download the latest release from the [Releases](https://github.com/dias-andre/shield/releases) page:

| Platform | Architecture | File |
|----------|--------------|------|
| Linux    | x86_64       | `shield_1.0.0_linux_amd64.tar.gz` |
| Linux    | ARM64        | `shield_1.0.0_linux_arm64.tar.gz` |
| macOS    | x86_64       | `shield_1.0.0_darwin_amd64.tar.gz` |
| macOS    | ARM64        | `shield_1.0.0_darwin_arm64.tar.gz` |

Extract and install:

```bash
tar -xzf shield_1.0.0_*.tar.gz
mv shield ~/.local/bin/
```

### Build from Source

Make sure you have [Go](https://go.dev/) installed:

```bash
git clone https://github.com/dias-andre/shield.git
cd shield
go build -o shield main.go
mv shield ~/.local/bin/
```

## Quick Start

Initialize your vault (this generates a secure master key in your OS Keyring)

```bash
shield setup
```

Add a new server interactively:

```bash
shield add server
```

*Or via positional arguments:*

```bash
shield add server aws-prod ubuntu 10.0.0.5 file:~/Downloads/ubuntu.pem
```

## Roadmap

Shield is actively evolving. Here is what is planned for the future:

- **Native SSH Spawning:** Launch secure SSH sessions directly from the CLI using
your encrypted credentials (no need to decrypt files manually).

- **SCP Support:** Securely copy files to and from your servers using the CLI.

- **Export/Import Tools:** Safely export your vault data and keys for backups or migrate them to another
machine.

- **Ansible Integration:** Automatically generate dynamic Ansible inventories from your Shield vault.

## Contributing

Contributions, issues, and feature requests are welcome! Feel free to check the issues
page.


