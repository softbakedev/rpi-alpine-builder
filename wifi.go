package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// getCurrentlyConnectedWifi attempts to get the SSID of the network
// the user is currently connected to, using OS-specific external tools.
//
// Linux:   Uses "nmcli"
// macOS:   Uses "/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport -I"
//
//	(with a fallback to "networksetup -getairportnetwork <device>" if you prefer.)
//
// Windows: Uses "netsh wlan show interfaces"
func getCurrentlyConnectedWifi() (string, error) {
	switch runtime.GOOS {
	case "linux":
		return getWifiLinux()
	case "darwin":
		return getWifiMac()
	case "windows":
		return getWifiWindows()
	default:
		return "", fmt.Errorf("getCurrentlyConnectedWifi not implemented for OS: %s", runtime.GOOS)
	}
}

// getWifiLinux executes `nmcli -t -f active,ssid dev wifi` and parses
// the output to find the row with active == "yes".
func getWifiLinux() (string, error) {
	out, err := exec.Command("nmcli", "-t", "-f", "active,ssid", "dev", "wifi").Output()
	if err != nil {
		return "", fmt.Errorf("failed to run nmcli: %v", err)
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		parts := strings.Split(line, ":")
		// Expecting lines in format: "yes:<SSID>" or "no:<SSID>"
		if len(parts) == 2 && parts[0] == "yes" {
			return parts[1], nil
		}
	}
	return "", errors.New("no connected Wi-Fi found on Linux")
}

// getWifiMac tries to parse output from the `airport -I` command.
// Example output snippet:
//
//	agrCtlRSSI: -42
//	SSID: MyWifiNetwork
//	BSSID: ...
//
// We look for a line starting with "SSID:".
func getWifiMac() (string, error) {
	// Execute the system_profiler command
	cmd := exec.Command("system_profiler", "SPAirPortDataType")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute system_profiler: %v", err)
	}

	// Initialize a scanner to read the output line by line
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Check if the line contains "Current Network"
		if strings.Contains(line, "Current Network") {
			// Depending on the system_profiler output, the network name might be on the same line or the next line.
			// We'll handle both cases.

			// Attempt to extract from the current line
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				network := strings.TrimSpace(parts[1])
				// Remove any colons from the network name if present
				network = strings.ReplaceAll(network, ":", "")
				if network != "" {
					return network, nil
				}
			}

			// If the network name is on the next line, read it
			if scanner.Scan() {
				nextLine := strings.TrimSpace(scanner.Text())
				// Remove any colons from the network name
				network := strings.ReplaceAll(nextLine, ":", "")
				network = strings.TrimSpace(network)
				if network != "" {
					return network, nil
				}
			}

			// If neither method yields a result, continue searching
		}
	}

	// Check for any scanning errors
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading system_profiler output: %v", err)
	}

	return "", errors.New("no connected Wi-Fi found on macOS")
}

// getWifiWindows executes `netsh wlan show interfaces` and looks for a line starting with "SSID"
//
// Example snippet from `netsh wlan show interfaces`:
//
//	There is 1 interface on the system:
//
//	Name                   : Wi-Fi
//	Description            : Intel(R) Wireless blah
//	GUID                   : ...
//	Physical address       : ...
//	State                 : connected
//	SSID                  : MyWifiNetwork
//	BSSID                 : ...
//	...
func getWifiWindows() (string, error) {
	out, err := exec.Command("netsh", "wlan", "show", "interfaces").Output()
	if err != nil {
		return "", fmt.Errorf("failed to run netsh: %v", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Typically lines may say something like: "SSID                   : MyWifiNetwork"
		if strings.HasPrefix(strings.ToLower(line), "ssid") {
			// Remove "SSID" (or "SSID:") and any extra punctuation
			// Some outputs look like: "SSID                   : MyWifiNetwork"
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				ssid := strings.TrimSpace(parts[1])
				if ssid != "" && !strings.EqualFold(ssid, "SSID") {
					return ssid, nil
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading netsh output: %v", err)
	}
	return "", errors.New("no connected Wi-Fi found on Windows")
}

// pickWifiNetwork orchestrates the OS-specific Wi-Fi scan, then prompts user to pick an SSID.
func pickWifiNetwork() (string, error) {
	var ssids []string
	var err error

	switch runtime.GOOS {
	case "linux":
		ssids, err = scanWifiLinux()
	case "darwin":
		ssids, err = scanWifiMac()
	case "windows":
		ssids, err = scanWifiWindows()
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	if err != nil {
		// Try to get the SSID of the currently connected network:
		connectedSSID, errConn := getCurrentlyConnectedWifi()
		if errConn == nil && connectedSSID != "" {
			fmt.Printf("Detected currently connected Wi-Fi: %s\n", connectedSSID)
			// Prompt user with the connected SSID as the default
			return promptWithDefault("Enter Wi-Fi SSID", connectedSSID), nil
		}

		fmt.Printf("Error scanning Wi-Fi networks: %v\n", err)

		// If we can't detect or retrieve the current SSID, fallback to a manual entry:
		return prompt("Enter Wi-Fi SSID: "), nil
	}

	return promptForSSID(ssids)
}

// promptWithDefault prints a prompt and includes a default value. If the user just hits Enter,
// the default value is returned; otherwise, user input is returned.
func promptWithDefault(promptText, defaultVal string) string {
	fmt.Printf("%s [%s]: ", promptText, defaultVal)
	var userInput string
	_, _ = fmt.Scanln(&userInput)
	input := strings.TrimSpace(userInput)
	if input == "" {
		return defaultVal
	}
	return input
}

func scanWifiLinux() ([]string, error) {
	cmd := exec.Command("nmcli", "-f", "SSID", "-t", "device", "wifi", "list")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run nmcli: %v", err)
	}
	lines := strings.Split(string(out), "\n")
	var ssids []string
	for _, line := range lines {
		if ssid := strings.TrimSpace(line); ssid != "" {
			ssids = append(ssids, ssid)
		}
	}
	return ssids, nil
}

func scanWifiMac() ([]string, error) {
	cmd := exec.Command("/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport", "-s")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to scan Wi-Fi on macOS: %v", err)
	}
	re := regexp.MustCompile(`SSID:\s+(.*)`)
	var ssids []string
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			ssids = append(ssids, strings.TrimSpace(matches[1]))
		}
	}
	if len(ssids) == 0 {
		return nil, errors.New("no Wi-Fi networks found on macOS")
	}
	return ssids, nil
}

func scanWifiWindows() ([]string, error) {
	cmd := exec.Command("netsh", "wlan", "show", "networks")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run netsh: %v", err)
	}
	re := regexp.MustCompile(`^SSID\s+\d+\s+:\s+(.*)$`)
	var ssids []string
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			ssids = append(ssids, strings.TrimSpace(matches[1]))
		}
	}
	return ssids, nil
}

func promptForSSID(ssids []string) (string, error) {
	if len(ssids) == 0 {
		logger.Println("No Wi-Fi networks found. Enter SSID manually:")
		return prompt("Wi-Fi SSID: "), nil
	}
	fmt.Println("Available Wi-Fi networks:")
	for i, ssid := range ssids {
		fmt.Printf("[%d] %s\n", i, ssid)
	}
	fmt.Println("[m] Enter manually")
	choice := prompt("Select index or 'm': ")
	if choice == "m" {
		return prompt("Wi-Fi SSID (manual): "), nil
	}
	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 0 || idx >= len(ssids) {
		logger.Println("Invalid selection. Enter SSID manually:")
		return prompt("Wi-Fi SSID: "), nil
	}
	return ssids[idx], nil
}
