# Shield CLI

A secure, fast, and interactive command-line interface written in Go to manage your SSH servers, private keys, and passwords. 

Shield acts as a local encrypted vault for your infrastructure credentials. Instead of leaving private `.pem` or `id_rsa` files lying around in plaintext, Shield encrypts everything using **AES-GCM 256-bit** encryption. The master key is never stored in files; it is securely managed by your Operating System's native Keyring (Secret Service API via D-Bus, Keychain, or Credential Manager).

## Features

* **OS Keyring Integration:** Secure master key storage natively integrated with your OS (works flawlessly on Linux/Wayland, macOS, and Windows).
* **Military-Grade Encryption:** All host data, usernames, passwords, and private key contents are encrypted at rest using AES-GCM.
* **Interactive Prompts:** Beautiful and user-friendly terminal UI for adding resources, powered by `survey`.
* **Zero Plaintext Files:** Private keys are ingested into the encrypted vault, keeping your `~/.ssh/` directory clean and safe.

## Installation

Make sure you have [Go](https://go.dev/) installed. Clone the repository and build the binary:

```bash
git clone https://github.com/dias-andre/shield.git
cd shield
go build -o shield main.go

# Move the binary to your PATH
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


