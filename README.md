# age-gitlab-tool-tui

`age-gitlab-tool-tui` is a terminal-based user interface (TUI) written in Go, designed to simplify encrypting data with [`age`](https://github.com/FiloSottile/age). It streamlines the process by fetching SSH public keys directly from a GitLab instance, allowing users to quickly select recipients, input data, and perform encryption seamlessly within a simple, interactive terminal UI.

## Features

- Detects and offers to decrypt age-encrypted files found in the clipboard
- Supports passphrase-protected SSH keys for decryption
- Fetches a list of active GitLab users along with their SSH keys.
- Interactive recipient selection through an intuitive searchable list.
- Encrypt plaintext data directly from the terminal interface.
- Generates ASCII-armored ciphertext compatible with the `age` tool.
- Guided environment setup - prompts for missing configuration values

## Installation

Ensure you have Go installed on your system. You can download it [here](https://golang.org/dl/).


Install the package:

```bash
go install github.com/deathrjj/age-gitlab-tool-tui@v0.2.0
```

## Setup

You can either set environment variables before running the application, or you'll be prompted to enter them interactively when needed:

```bash
# Required for encryption (or enter interactively):
export GITLAB_URL="https://gitlab.example.com"
export GITLAB_TOKEN="your_personal_access_token"

# Required for decryption (or enter interactively):
export AGE_PRIVATE_KEY_PATH="/path/to/your/private/key.txt"
```

- `GITLAB_URL`: URL of your GitLab instance.
- `GITLAB_TOKEN`: GitLab Personal Access Token with sufficient permissions to read user data and SSH keys.
- `AGE_PRIVATE_KEY_PATH`: Path to your SSH private key file for decryption (both regular and passphrase-protected keys are supported).

If any of these values are not set when needed, the application will prompt you to enter them.

## Usage

Run the application directly:

```bash
age-gitlab-tool-tui
```

### Encryption

When started without an age-encrypted file in the clipboard:

- **Recipient List**:
  - `↑ / ↓`: Navigate through the user list.
  - `Enter`: Toggle recipient selection.
  - Type to search and filter users.

- **Data Input**:
  - Type or paste plaintext data into the provided text area.

- **Encrypting Data**:
  - Press `Tab` to navigate between interface elements.
  - Select "Encrypt" to generate encrypted output.

After encryption, the encrypted data will be printed to the terminal in ASCII-armored format.

### Decryption

When you have an age-encrypted file in your clipboard:

1. The application will detect it and ask if you want to decrypt it
2. If you select "Yes", it will:
   - Check if the `AGE_PRIVATE_KEY_PATH` environment variable is set
   - Attempt to use the specified private key to decrypt the message
   - If the key is passphrase-protected, prompt you to enter the passphrase
   - Output the decrypted content to the terminal
3. If you select "No", it will proceed with the normal encryption interface

## Output Example

Encrypted data output follows the standard `age` ASCII-armored format:

```bash
-----BEGIN AGE ENCRYPTED FILE-----
...
-----END AGE ENCRYPTED FILE-----
```

You can decrypt this data by running this program again with the age file in your clipboard. 
Or by using `age` or compatible tools:

```bash
age -d -i <your-private-key-file> encrypted-file.age
```

## Troubleshooting

If the application fails to start or throws an error fetching users, verify:

- Your GitLab URL is reachable and correct.
- Your personal access token is valid and has the required permissions (`read_user` and `read_ssh_keys`).
- Network connectivity between your system and the GitLab instance.

For decryption issues:
- Make sure your `AGE_PRIVATE_KEY_PATH` points to a valid SSH private key file
- If your key is passphrase-protected, ensure you're entering the correct passphrase
- The application uses `ssh-keygen` for decrypting passphrase-protected keys, so ensure this tool is available on your system
- Verify the encrypted content in the clipboard is valid and complete

## Dependencies

- [rivo/tview](https://github.com/rivo/tview) - Terminal UI components.
- [gdamore/tcell](https://github.com/gdamore/tcell) - Terminal cell library for TUI.
- [filippo.io/age](https://github.com/FiloSottile/age) - Encryption library used to perform secure encryption.
- [atotto/clipboard](https://github.com/atotto/clipboard) - Clipboard access for detecting encrypted files.

## Contributing

Contributions are welcome. Feel free to open issues or submit pull requests.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

