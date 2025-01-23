package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"log"
	_ "softbake.dev/rpialp/internal"
	"strings"
)



// Global variables for CLI usage:
var (
	volumeSizeMg  int    = 4096
	volumeLabel   string = "ALPINE"
	cliName       string = "rpialp"
	hostname      string
	ssid          string
	ssidPass      string
	ssidPsk       string
	rootPass      string
	alpineVersion string
	// Verbose is tied to the --verbose/-v flag; logger prints to /dev/null by default.
	var verbose bool

	// We'll unmarshal JSON into alpineConfig in main.go
	alpineConfig internal.AlpineConfig
)

func init() {
	// Define our Cobra flags
	rootCmd.Flags().StringVar(&alpineVersion, "alpine-version", "",
		"Alpine release version for Raspberry Pi (e.g. 3.18.2). If left empty, shows a list from the embedded config.")
	rootCmd.Flags().StringVar(&hostname, "hostname", "", "Hostname for the system.")
	rootCmd.Flags().StringVar(&ssid, "ssid", "", "Wi-fi name")
	rootCmd.Flags().StringVar(&ssidPass, "ssidPass", "", "Wi-Fi passphrase")
	rootCmd.Flags().StringVar(&rootPass, "rootPass", "", "Root passphrase")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	// Add the build subcommand to rootCmd
	rootCmd.AddCommand(buildCmd)
}

func main() {
	// 1. Unmarshal the embedded JSON (alpine_versions.json) into alpineConfig
	if err := json.Unmarshal(VersionsData, &alpineConfig); err != nil {
		log.Fatalf("Failed to parse embedded versions.json: %v\n", err)
	}

	// 2. Execute the root command (Cobra)
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

// rootCmd is our main Cobra command
var rootCmd = &cobra.Command{
	Use:   cliName,
	Short: "CLI for building alpine image for Raspberry PI devices",
	// Show help if no subcommand or flags are provided
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// buildCmd is the subcommand that handles the "build" logic
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Builds the Alpine release image with specified configuration.",
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		// Turn on verbose logging if --verbose / -v was provided
		EnableVerboseLogging()

		// 1. If user did NOT specify --alpine-version, prompt from embedded config
		if strings.TrimSpace(alpineVersion) == "" {
			alpineVersion = pickAlpineVersionInteractive(alpineConfig)
		}

		// 2. Download the Alpine release
		if strings.TrimSpace(alpineVersion) != "" {
			fmt.Printf("Alpine version selected: %s. Downloading (if not cached)...\n", alpineVersion)
			if err := downloadAlpineRelease(alpineVersion); err != nil {
				return fmt.Errorf("failed downloading Alpine release: %v", err)
			}
		}

		// 3. Prompt for hostname if not provided
		if strings.TrimSpace(hostname) == "" {
			hostname = prompt("Hostname (required): ")
			if strings.TrimSpace(hostname) == "" {
				return errors.New("hostname cannot be empty")
			}
		}

		// 4. Pick a Wi-Fi network
		if strings.TrimSpace(ssid) == "" {
			ssid, err = pickWifiNetwork()
			if err != nil {
				return fmt.Errorf("error picking Wi-Fi network: %v", err)
			}
		}

		// 5. Prompt for Wi-Fi passphrase => derive WPA2-PSK
		if strings.TrimSpace(ssidPass) == "" {
			ssidPass = promptHidden("Enter Wi-Fi passphrase (hidden): ")
		}

		ssidPsk, err = generateWpaPsk(ssid, ssidPass)
		if err != nil {
			return fmt.Errorf("failed generating WPA2-PSK: %v", err)
		}

		// 6. Root password => hashed
		if strings.TrimSpace(rootPass) == "" {
			rootPass = promptHidden("Enter root password to encrypt (hidden): ")
		}
		shadowPass, err := generateShadowPasswordHash(rootPass)
		if err != nil {
			return fmt.Errorf("failed to encrypt root password: %v", err)
		}

		// 7. Process the apkovl
		if err := processApkovl(hostname, ssid, ssidPsk, shadowPass); err != nil {
			return fmt.Errorf("failed to process apkovl tar file %v", err)
		}

		// 8. List block devices
		devices, err := listBlockDevices()
		if err != nil {
			return fmt.Errorf("failed listing block devices: %v", err)
		}

		var chosen []string
		if len(devices) > 0 {
			chosen, err = pickDevices(devices)
			if err != nil {
				return fmt.Errorf("error picking devices: %v", err)
			}
		} else {
			logger.Println("No block devices found in /sys/block.")
		}

		if len(chosen) > 0 {
			// Format FAT32
			if err = formatVolumeFat32(chosen[0]); err != nil {
				return fmt.Errorf("error format fat32 volume %s: %v", chosen[0], err)
			}
			// Build the image
			if err = buildImage(chosen[0]); err != nil {
				return fmt.Errorf("error build image in volume %s: %v", chosen[0], err)
			}
		} else {
			return fmt.Errorf("the volume cannot be empty")
		}
		return nil
	},
}
