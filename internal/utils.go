package internal

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"github.com/GehirnInc/crypt"
	"io"
	"os"
	"path/filepath"
	"time"

	_ "github.com/GehirnInc/crypt/sha512_crypt"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/term"
)

// Prompt reads a line from stdin (non-hidden)
func Prompt(label string) string {
	fmt.Print(label)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return scanner.Text()
	}
	return ""
}

// PromptHidden reads a line from stdin without echoing it.
func PromptHidden(label string) string {
	fmt.Print(label)
	bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		fmt.Println("Error reading hidden input:", err)
		return Prompt("Please enter again (not hidden): ")
	}
	return string(bytePassword)
}

// GenerateWpaPsk derives a WPA2-PSK from SSID + passphrase using PBKDF2(HMAC-SHA1)
func GenerateWpaPsk(ssid, passphrase string) (string, error) {
	key := pbkdf2.Key([]byte(passphrase), []byte(ssid), 4096, 32, sha1.New)
	return fmt.Sprintf("%x", key), nil
}

// GenerateShadowPasswordHash produces a /etc/shadow-compatible SHA-512 hash with a random salt.
func GenerateShadowPasswordHash(password string) (string, error) {
	// Generate a random salt (16 bytes for a strong salt)
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("error generating salt: %v", err)
	}

	// Convert the salt to a base64-encoded string
	saltSha512 := fmt.Sprintf("$6$%s", base64.StdEncoding.EncodeToString(salt))

	// Hash the password with the salt using SHA-512
	crypt := crypt.SHA512.New()
	cryptFormat, err := crypt.Generate([]byte(password), []byte(saltSha512))
	if err != nil {
		fmt.Errorf("error hashed password by salt: %v", err)
	}

	return cryptFormat, nil
}

// compress takes a source dir and creates a .tar.gz at dest
func compress(src, dest string) error {
	tarGzFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create tar.gz file: %v", err)
	}
	defer tarGzFile.Close()

	gzipWriter := gzip.NewWriter(tarGzFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	return filepath.Walk(src, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, filePath)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath)

		// Zero out timestamps to keep consistent
		header.ModTime = time.Unix(0, 0)
		header.AccessTime = time.Unix(0, 0)
		header.ChangeTime = time.Unix(0, 0)

		switch {
		case info.Mode()&os.ModeSymlink != 0:
			linkTarget, err := os.Readlink(filePath)
			if err != nil {
				return fmt.Errorf("failed to read symlink target of %s: %v", filePath, err)
			}
			header.Typeflag = tar.TypeSymlink
			header.Linkname = linkTarget
			if err := tarWriter.WriteHeader(header); err != nil {
				return err
			}
		case info.Mode().IsRegular():
			header.Typeflag = tar.TypeReg
			if err := tarWriter.WriteHeader(header); err != nil {
				return err
			}
			file, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer file.Close()
			if _, err := io.Copy(tarWriter, file); err != nil {
				return err
			}
		case info.IsDir():
			header.Typeflag = tar.TypeDir
			if err := tarWriter.WriteHeader(header); err != nil {
				return err
			}
		default:
			fmt.Printf("Skipping unhandled file type: %s\n", filePath)
		}

		return nil
	})
}
