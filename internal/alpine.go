package internal

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// AlpineVersions represents the structure of alpine_versions.json
type AlpineVersions struct {
	Versions []string `json:"versions"`
}

// PlaceholderReplacement defines the placeholders and their replacement values for specific files.
type PlaceholderReplacement struct {
	Filename     string
	Placeholders map[string]string
}

// ProgressReader wraps an underlying io.Reader to track the
// number of bytes read and print progress.
type ProgressReader struct {
	io.Reader
	total      int64 // total size of the file (for percentage calculation)
	bytesRead  int64 // how many bytes have been read so far
	lastUpdate time.Time
}

// Read implements the io.Reader interface, wrapping the underlying
// reader to count bytes and print progress.
func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.bytesRead += int64(n)

	// Throttle updates so we don't print too often; adjust as needed
	if time.Since(pr.lastUpdate) > 200*time.Millisecond {
		pr.printProgress()
		pr.lastUpdate = time.Now()
	}

	return n, err
}

// GetAlpineVersions return list of alpine versions
func GetAlpineVersions() ([]string, error) {
	var versions AlpineVersions
	// 1. Unmarshal the embedded JSON (alpine_versions.json) into alpineConfig
	if err := json.Unmarshal(VersionsData, &versions); err != nil {
		logger.Fatalf("Failed to parse embedded versions.json: %v\n", err)
		return []string{}, err
	}

	return versions.Versions, nil
}

// printProgress prints the percentage of bytes read relative to total.
func (pr *ProgressReader) printProgress() {
	if pr.total <= 0 {
		// If total is unknown, we can only show bytes read.
		fmt.Printf("\rDownloaded %d bytes...", pr.bytesRead)
		return
	}
	percent := float64(pr.bytesRead) / float64(pr.total) * 100
	fmt.Printf("\rDownloading: %.1f%% (%d / %d bytes)", percent, pr.bytesRead, pr.total)
}

// PickAlpineVersionInteractive shows a list of versions (first 10) and lets the user pick.
func PickAlpineVersionInteractive(cfg AlpineVersions) string {
	if len(cfg.Versions) == 0 {
		logger.Println("No known Alpine versions in config. Please type one manually:")
		return Prompt("Alpine version: ")
	}

	count := len(cfg.Versions)
	if count > 10 {
		count = 10
	}

	logger.Println("Available Alpine versions (pick an index or 'm' for manual):")
	for i := 0; i < count; i++ {
		fmt.Printf("[%d] %s\n", i, cfg.Versions[i])
	}
	fmt.Println("[m] Enter manually")
	choice := Prompt("Select index or 'm': ")

	if choice == "m" {
		manual := Prompt("Type Alpine version manually (e.g. 3.18.2): ")
		return strings.TrimSpace(manual)
	}

	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 0 || idx >= count {
		logger.Println("Invalid selection, defaulting to empty version.")
		return ""
	}
	return cfg.Versions[idx]
}

// DownloadAlpineRelease fetches the chosen Alpine release tarball, and extracts it to the cache dir.
// This version prints progress for the download (by bytes) and also shows extraction progress by file count.
func DownloadAlpineRelease(cliName string, version string) error {
	cacheDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get user cache directory: %v", err)
	}
	myCacheDir := filepath.Join(fmt.Sprintf("%s/%s", cacheDir, cliName), "alpine")

	// Clean up or ensure directory is empty
	if err := os.RemoveAll(myCacheDir); err != nil {
		return fmt.Errorf("could not remove existing extract dir: %v", err)
	}
	if err := os.MkdirAll(myCacheDir, 0o755); err != nil {
		return fmt.Errorf("could not create cache directory %s: %v", myCacheDir, err)
	}

	// Build the URL (example for aarch64)
	firstVersion := fmt.Sprintf("%s.%s", strings.Split(version, ".")[0], strings.Split(version, ".")[1])
	url := fmt.Sprintf("https://dl-cdn.alpinelinux.org/alpine/v%s/releases/aarch64/alpine-rpi-%s-aarch64.tar.gz", firstVersion, version)

	logger.Printf("Downloading Alpine from URL: %s\n", url)

	// Initiate HTTP request
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Parse Content-Length to determine total size for progress
	var totalSize int64
	if clStr := resp.Header.Get("Content-Length"); clStr != "" {
		if size, err := strconv.ParseInt(clStr, 10, 64); err == nil {
			totalSize = size
		}
	}

	// Create a gzip reader from the progressReader
	gzReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("error creating gzip reader: %v", err)
	}
	defer gzReader.Close()

	// Read the tar
	tarReader := tar.NewReader(gzReader)

	// We can do a quick scan of the tar to count total files for progress, if we want.
	// A better way is to read the tar twice (inefficient) or store headers, but let's keep it simple:
	// We'll just count how many TypeReg or TypeDir for an approximate measure of "items" to extract.
	if totalSize > 0 {
		// We only do a quick pass if the stream is seekable or small.
		// For a normal HTTP stream, we can't do that easily unless the server supports Range requests.
		// For demonstration, let's skip the "pre-scan" here, or we do a HEAD request.
		// If you want an item-based progress, do a separate HEAD + length or an index of the tar, etc.
	}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading tar entry: %v", err)
		}

		targetPath := filepath.Join(myCacheDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			f, err := os.Create(targetPath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tarReader); err != nil {
				f.Close()
				return err
			}
			f.Close()
		default:
			// We will just ignore other types
			fmt.Printf("Skipping unsupported file type: %s\n", header.Name)
		}
	}
	logger.Println("Download and extraction completed successfully!")
	return nil
}

// ProcessApkovl extracts the embedded apkovl tar, replaces placeholders, etc.
func ProcessApkovl(cliName, hostname, ssid, psk, shadowPass string) error {
	cacheDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get user cache directory: %v", err)
	}
	extractDir := filepath.Join(fmt.Sprintf("%s/%s", cacheDir, cliName), "apkovl")
	logger.Println("Extracting embedded tar to:", extractDir)

	if err := os.RemoveAll(extractDir); err != nil {
		return fmt.Errorf("could not remove existing extract dir: %v", err)
	}
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return fmt.Errorf("could not create extract directory: %v", err)
	}

	if err := extractTarToDir(ApkovlTar, extractDir); err != nil {
		return fmt.Errorf("failed to extract tar: %v", err)
	}

	if err := updatePlaceholdersInExtracted(extractDir, hostname, ssid, psk, shadowPass); err != nil {
		return fmt.Errorf("failed updating placeholders: %v", err)
	}
	return nil
}

// ExtractTarToDir unpacks a gzip'd tar (in []byte form) to destDir
func extractTarToDir(tarData []byte, destDir string) error {
	gzipReader, err := gzip.NewReader(bytes.NewReader(tarData))
	if err != nil {
		return fmt.Errorf("error creating gzip reader: %v", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading tar entry: %v", err)
		}

		targetPath := filepath.Join(destDir, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return fmt.Errorf("error creating directory: %v", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return fmt.Errorf("could not create parent dir: %v", err)
			}
			outFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("error creating file %s: %v", targetPath, err)
			}
			defer outFile.Close()
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return fmt.Errorf("error writing to file %s: %v", targetPath, err)
			}
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return fmt.Errorf("could not create parent dir for symlink: %v", err)
			}
			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				return fmt.Errorf("creating symlink %s -> %s: %v", targetPath, header.Linkname, err)
			}
		default:
			logger.Printf("Skipping unsupported file type: %s (typeflag=%d)\n", header.Name, header.Typeflag)
		}
	}
	return nil
}

// updatePlaceholdersInExtracted walks through the extractDir and replaces placeholders
// in specified files with provided values. It skips symlinks and processes only regular files.
func updatePlaceholdersInExtracted(extractDir, hostname, ssid, psk, shadowPass string) error {
	// Define the mapping of filenames to their respective placeholders and replacements.
	replacements := []PlaceholderReplacement{
		{
			Filename: "wpa_supplicant.conf",
			Placeholders: map[string]string{
				"{{ ssid_name }}": ssid,
				"{{ ssid_psk }}":  psk,
			},
		},
		{
			Filename: "hostname",
			Placeholders: map[string]string{
				"{{ hostname }}": hostname,
			},
		},
		{
			Filename: "hostname",
			Placeholders: map[string]string{
				"{{ hostname }}": hostname,
			},
		},
		{
			Filename: "shadow",
			Placeholders: map[string]string{
				"{{ shadow_pass }}": shadowPass,
			},
		},
		{
			Filename: "shadow-",
			Placeholders: map[string]string{
				"{{ shadow_pass }}": shadowPass,
			},
		},
	}

	// Walk through the directory
	return filepath.Walk(extractDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %q: %v", path, err)
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip symbolic links
		if info.Mode()&os.ModeSymlink != 0 {
			logger.Printf("Skipping symlink: %s\n", path)
			return nil
		}

		// Get the base filename
		baseName := filepath.Base(path)

		// Iterate through the replacements to find a match
		for _, rep := range replacements {
			if baseName == rep.Filename {
				logger.Printf("Found %s at: %s\n", rep.Filename, path)

				// Read the file content
				data, readErr := os.ReadFile(path)
				if readErr != nil {
					return fmt.Errorf("error reading %s: %v", path, readErr)
				}

				content := string(data)

				// Replace all placeholders
				for placeholder, replacement := range rep.Placeholders {
					content = strings.ReplaceAll(content, placeholder, replacement)
				}

				// Validate that replacements occurred (optional)
				for placeholder := range rep.Placeholders {
					if strings.Contains(content, placeholder) {
						logger.Printf("Warning: Placeholder %s not replaced in %s\n", placeholder, path)
					}
				}

				// Write the updated content back to the file
				if writeErr := os.WriteFile(path, []byte(content), info.Mode()); writeErr != nil {
					return fmt.Errorf("error writing back to %s: %v", path, writeErr)
				}

				logger.Printf("Updated placeholders in: %s\n", path)
				break // No need to check other replacements for this file
			}
		}

		return nil
	})
}

// BuildImage copies Alpine data and creates a .tar.gz for the apkovl
func BuildImage(cliName, volumeDir, hostname string) error {
	cacheDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get user cache directory: %v", err)
	}

	alpinePath := filepath.Join(cacheDir, cliName, "alpine")
	apkovlPath := filepath.Join(cacheDir, cliName, "apkovl")

	// Copy Alpine data
	if err := os.CopyFS(alpinePath, os.DirFS(volumeDir)); err != nil {
		return fmt.Errorf("error copying alpine data into volumeDir %s: %v", volumeDir, err)
	}

	tarGzPath := filepath.Join(volumeDir, fmt.Sprintf("%s.apkovl.tar.gz", hostname))
	if err := compress(apkovlPath, tarGzPath); err != nil {
		return fmt.Errorf("failed to compress apkovl data: %v", err)
	}

	logger.Printf("apkovl.tar.gz created successfully at %s\n", tarGzPath)
	return nil
}
