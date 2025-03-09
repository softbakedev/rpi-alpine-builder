package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"softbake.dev/rpialp/internal"
	"strings"
)

// Global variables for CLI usage:
var (
	volumeSizeMg  int    = 4096
	volumeLabel   string = "ALPINE"
	cliName       string = "rpialp"
	cacheFolder   string = fmt.Sprintf(".%s", cliName)
	hostname      string
	ssid          string
	ssidPass      string
	ssidPsk       string
	rootPass      string
	alpineVersion string
	// Verbose is tied to the --verbose/-v flag; logger prints to /dev/null by default.
	verbose bool

	// We'll unmarshal JSON into alpineConfig in main.go
	alpineVersions internal.AlpineVersions
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
	if err := json.Unmarshal(internal.VersionsData, &alpineVersions); err != nil {
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
	Short: "Builds the Alpine release image with specified configuration for Raspberry Pi",
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		// Turn on verbose logging if --verbose / -v was provided
		internal.EnableVerboseLogging(verbose)
		spinner := internal.NewSpinner()
		// 1. If user did NOT specify --alpine-version, prompt from embedded config
		if strings.TrimSpace(alpineVersion) == "" {
			alpineVersion = internal.PickAlpineVersionInteractive(alpineVersions)
		}

		// 2. Download the Alpine release
		if strings.TrimSpace(alpineVersion) != "" {
			spinner.Start()
			fmt.Printf("Alpine version selected: %s. Downloading (if not cached)...\n", alpineVersion)
			if err := internal.DownloadAlpineRelease(cacheFolder, alpineVersion); err != nil {
				spinner.Stop()
				return fmt.Errorf("failed downloading Alpine release: %v", err)
			}
		}

		spinner.Stop()

		// 3. Prompt for hostname if not provided
		if strings.TrimSpace(hostname) == "" {
			hostname = internal.Prompt("Hostname (required): ")
			if strings.TrimSpace(hostname) == "" {
				return errors.New("hostname cannot be empty")
			}
		}

		// 4. Pick a Wi-Fi network
		spinner.Start()
		if strings.TrimSpace(ssid) == "" {
			fmt.Printf("Finding available wifi around")
			ssids, err := internal.PickWifiNetwork()

			if err != nil {
				spinner.Stop()
				return fmt.Errorf("error picking Wi-Fi network: %v", err)
			}
			spinner.Stop()
			internal.PromptForSSID(ssids)
		}

		// 5. Prompt for Wi-Fi passphrase => derive WPA2-PSK
		if strings.TrimSpace(ssidPass) == "" {
			ssidPass = internal.PromptHidden("Enter Wi-Fi passphrase (hidden): ")
		}

		ssidPsk, err = internal.GenerateWpaPsk(ssid, ssidPass)
		if err != nil {
			return fmt.Errorf("failed generating WPA2-PSK: %v", err)
		}

		// 6. Root password => hashed
		if strings.TrimSpace(rootPass) == "" {
			rootPass = internal.PromptHidden("Enter root password to encrypt (hidden): ")
		}
		shadowPass, err := internal.GenerateShadowPasswordHash(rootPass)
		if err != nil {
			return fmt.Errorf("failed to encrypt root password: %v", err)
		}

		// 7. Process the apkovl
		spinner.Start()
		if err := internal.ProcessApkovl(cacheFolder, hostname, ssid, ssidPsk, shadowPass); err != nil {
			spinner.Stop()
			return fmt.Errorf("failed to process apkovl tar file %v", err)
		}

		spinner.Stop()

		// 8. List block devices
		devices, err := internal.ListBlockDevices()
		if err != nil {
			return fmt.Errorf("failed listing block devices: %v", err)
		}

		var chosen *string
		if len(devices) > 0 {
			chosen, err = internal.PickDevices(devices)
			if err != nil {
				return fmt.Errorf("error picking devices: %v", err)
			}
		}

		if chosen != nil {
			spinner.Start()
			// Format FAT32
			if err = internal.FormatVolumeFat32(*chosen, volumeLabel, volumeSizeMg); err != nil {
				spinner.Stop()
				return fmt.Errorf("error format fat32 volume %s: %v", *chosen, err)
			}
			// Build the image
			if err = internal.BuildImage(cacheFolder, *chosen, hostname); err != nil {
				spinner.Stop()
				return fmt.Errorf("error build image in volume %s: %v", *chosen, err)
			}
		} else {
			return fmt.Errorf("the volume cannot be empty")
		}

		spinner.Stop()

		return nil
	},
}
