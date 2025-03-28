package encryption

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"filippo.io/age"
	"filippo.io/age/agessh"
	"filippo.io/age/armor"
	"github.com/atotto/clipboard"
	"github.com/deathrjj/age-gitlab-tool-tui/gitlab"
	"github.com/deathrjj/age-gitlab-tool-tui/models"
)

// CheckClipboardForAgeFile checks if the clipboard contains an age encrypted file
func CheckClipboardForAgeFile() (string, bool) {
	text, err := clipboard.ReadAll()
	if err != nil {
		return "", false
	}

	// Check for age encrypted file markers
	if strings.Contains(text, "-----BEGIN AGE ENCRYPTED FILE-----") &&
		strings.Contains(text, "-----END AGE ENCRYPTED FILE-----") {
		return text, true
	}
	return "", false
}

// DecryptAgeFile decrypts an age encrypted file using a private key file
func DecryptAgeFile(encryptedText, privateKeyPath, passphrase string) (string, error) {
	// Read the private key file
	keyData, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read private key file: %w", err)
	}

	var sshIdentity age.Identity
	if passphrase == "" {
		// Try without passphrase
		sshIdentity, err = agessh.ParseIdentity(keyData)
		if err != nil && strings.Contains(err.Error(), "passphrase") {
			return "", fmt.Errorf("ssh key is passphrase protected, please provide passphrase")
		}
	} else {
		// Create a temporary file for SSH key handling
		sshKeyFile, err := os.CreateTemp("", "ssh-key-*")
		if err != nil {
			return "", fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(sshKeyFile.Name())
		defer sshKeyFile.Close()
		
		// Write the key data to the temp file
		_, err = sshKeyFile.Write(keyData)
		if err != nil {
			return "", fmt.Errorf("failed to write temp key file: %w", err)
		}
		sshKeyFile.Close()
		
		// Run the command to decrypt the key with the passphrase
		decryptCmd := exec.Command("bash", "-c", 
			fmt.Sprintf("ssh-keygen -p -P '%s' -N '' -f '%s'", passphrase, sshKeyFile.Name()))
		if err := decryptCmd.Run(); err != nil {
			return "", fmt.Errorf("failed to decrypt SSH key with passphrase: %w", err)
		}
		
		// Read the decrypted key
		decryptedKeyData, err := ioutil.ReadFile(sshKeyFile.Name())
		if err != nil {
			return "", fmt.Errorf("failed to read decrypted key: %w", err)
		}
		
		// Parse the decrypted key
		sshIdentity, err = agessh.ParseIdentity(decryptedKeyData)
		if err != nil {
			return "", fmt.Errorf("failed to parse decrypted SSH key: %w", err)
		}
	}
	
	if err != nil {
		return "", fmt.Errorf("failed to parse SSH key: %w", err)
	}

	// Decrypt the message using the SSH identity
	r, err := age.Decrypt(armor.NewReader(strings.NewReader(encryptedText)), sshIdentity)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	// Read the decrypted content
	decrypted, err := ioutil.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("failed to read decrypted content: %w", err)
	}

	return string(decrypted), nil
}

// EncryptData encrypts plaintext using age with each selected user's key as a recipient
func EncryptData(plaintext string, selected models.UserSelectionMap, gitlabClient *gitlab.Client) (string, error) {
	var recipients []age.Recipient
	
	for uid := range selected {
		keys, err := gitlabClient.FetchUserKeys(uid)
		if err != nil {
			return "", err
		}
		
		for _, keyStr := range keys {
			// Use agessh.ParseRecipient to parse an SSH key as an age recipient
			rec, err := agessh.ParseRecipient(keyStr)
			if err != nil {
				return "", fmt.Errorf("failed to parse recipient for user %d: %w", uid, err)
			}
			recipients = append(recipients, rec)
		}
	}
	
	var buf bytes.Buffer
	armorWriter := armor.NewWriter(&buf)
	
	w, err := age.Encrypt(armorWriter, recipients...)
	if err != nil {
		return "", err
	}
	
	if _, err := w.Write([]byte(plaintext)); err != nil {
		return "", err
	}
	
	if err := w.Close(); err != nil {
		return "", err
	}
	
	if err := armorWriter.Close(); err != nil {
		return "", err
	}
	
	return buf.String(), nil
} 
