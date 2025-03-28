# age-gitlab-tool-tui

`age-gitlab-tool-tui` is a terminal-based user interface (TUI) written in Go, designed to simplify encrypting data with [`age`](https://github.com/FiloSottile/age). It streamlines the process by fetching SSH public keys directly from a GitLab instance, allowing users to quickly select recipients, input data, and perform encryption seamlessly within a simple, interactive terminal UI.

## Features

- Fetches a list of active GitLab users along with their SSH keys.
- Interactive recipient selection through an intuitive searchable list.
- Encrypt plaintext data directly from the terminal interface.
- Generates ASCII-armored ciphertext compatible with the `age` tool.

## Installation

Ensure you have Go installed on your system. You can download it [here](https://golang.org/dl/).

Clone the repository:

```bash
git clone <repository-url>
cd age-gitlab-tool-tui
```

Install required Go dependencies:

```bash
go mod tidy
```

## Setup

Set the necessary environment variables to connect to your GitLab instance:

```bash
export GITLAB_URL="https://gitlab.example.com"
export GITLAB_TOKEN="your_personal_access_token"
```

- `GITLAB_URL`: URL of your GitLab instance.
- `GITLAB_TOKEN`: GitLab Personal Access Token with sufficient permissions to read user data and SSH keys.

## Usage

Run the application directly with Go:

```bash
go run main.go
```

### Interface Controls

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

## Output Example

Encrypted data output follows the standard `age` ASCII-armored format:

```bash
-----BEGIN AGE ENCRYPTED FILE-----
...
-----END AGE ENCRYPTED FILE-----
```

You can decrypt this data using `age` or compatible tools:

```bash
age -d -i <your-private-key-file> encrypted-file.age
```

## Troubleshooting

If the application fails to start or throws an error fetching users, verify:

- Your GitLab URL is reachable and correct.
- Your personal access token is valid and has the required permissions (`read_user` and `read_ssh_keys`).
- Network connectivity between your system and the GitLab instance.

## Dependencies

- [rivo/tview](https://github.com/rivo/tview) - Terminal UI components.
- [gdamore/tcell](https://github.com/gdamore/tcell) - Terminal cell library for TUI.
- [filippo.io/age](https://github.com/FiloSottile/age) - Encryption library used to perform secure encryption.

## Contributing

Contributions are welcome. Feel free to open issues or submit pull requests.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

