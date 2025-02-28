package internal

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// ListBlockDevices enumerates block devices per OS
func ListBlockDevices() ([]string, error) {
	switch runtime.GOOS {
	case "linux":
		return listBlockDevicesLinux()
	case "darwin":
		return listBlockDevicesDarwin()
	case "windows":
		return listBlockDevicesWindows()
	default:
		return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func listBlockDevicesLinux() ([]string, error) {
	data, err := os.ReadFile("/proc/partitions")
	if err != nil {
		return nil, fmt.Errorf("cannot read /proc/partitions: %v", err)
	}
	var devices []string
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	re := regexp.MustCompile(`\s+\d+\s+\d+\s+\d+\s+(\S+)`)
	for scanner.Scan() {
		line := scanner.Text()
		if matches := re.FindStringSubmatch(line); matches != nil {
			devName := matches[1]
			// skip loop/ram
			if strings.HasPrefix(devName, "loop") || strings.HasPrefix(devName, "ram") {
				continue
			}
			devices = append(devices, "/dev/"+devName)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return devices, nil
}

func listBlockDevicesDarwin() ([]string, error) {
	const volumesDir = "/Volumes"
	entries, err := os.ReadDir(volumesDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %v", volumesDir, err)
	}
	var devices []string
	for _, entry := range entries {
		if entry.IsDir() {
			devices = append(devices, filepath.Join(volumesDir, entry.Name()))
		}
	}
	return devices, nil
}

func listBlockDevicesWindows() ([]string, error) {
	var devices []string
	for c := 'A'; c <= 'Z'; c++ {
		drive := string(c) + ":\\"
		info, err := os.Stat(drive)
		if err == nil && info.IsDir() {
			devices = append(devices, drive)
		}
	}
	return devices, nil
}

// PickDevices prompts user to pick from the discovered block devices
func PickDevices(devices []string) (*string, error) {
	logger.Println("\nAvailable block devices:")
	for i, dev := range devices {
		fmt.Printf("[%d] %s\n", i, dev)
	}
	choice := Prompt("[m] Enter manually or [x] Skip volumes: ")
	if choice == "x" {
		return nil, nil
	}
	if choice == "m" {
		path := Prompt("Enter device path (e.g. /dev/sdb): ")
		if strings.TrimSpace(path) == "" {
			return nil, nil
		}
		return &path, nil
	}

	var selected []string
	parts := strings.Split(choice, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		idx, err := strconv.Atoi(p)
		if err != nil || idx < 0 || idx >= len(devices) {
			fmt.Printf("Invalid index: %s (skipped)\n", p)
			continue
		}
		selected = append(selected, devices[idx])
	}
	return &selected[0], nil
}

// unmountDevice unmounts the given device using external commands based on the OS.
func unmountDevice(volume string) error {
	switch runtime.GOOS {
	case "windows":
		return unmountWindows(volume)
	case "linux":
		return unmountLinux(volume)
	case "darwin":
		return unmountMacOS(volume)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// unmountWindows unmounts the volume on Windows using PowerShell commands,
// but prevents a window from being shown by setting HideWindow = true.
func unmountWindows(volume string) error {
	driveLetter := strings.TrimSuffix(volume, ":")
	psCommand := fmt.Sprintf("Dismount-Volume -DriveLetter %s -Force", driveLetter)

	// Prepare PowerShell command
	cmd := exec.Command("powershell.exe",
		"-noprofile",
		"-nologo",
		"-windowstyle", "Hidden",
		"-command", psCommand,
	)

	//// Prevent spawning a new PowerShell window
	//cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	// (Optional) Redirect output to see errors in your main console or logs
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// unmountLinux unmounts the volume on Linux using the 'umount' command.
func unmountLinux(volume string) error {
	cmd := exec.Command("umount", volume)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// unmountMacOS unmounts the volume on macOS using the 'diskutil unmount' command.
func unmountMacOS(volume string) error {
	cmd := exec.Command("diskutil", "unmount", volume)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// FormatVolumeFat32 formats the given volume with a FAT32 filesystem
func FormatVolumeFat32(volumePath, volumeLabel string, volumeSizeMg int) error {
	fmt.Println("Formatting volume as FAT32...")

	devicePath, err := getDevicePath(volumePath)
	if err != nil {
		return fmt.Errorf("error get device path %v", err)
	}

	if err = unmountDevice(volumePath); err != nil {
		return fmt.Errorf("error unmount device %v", err)
	}

	switch runtime.GOOS {
	case "windows":
		// On Windows, devicePath is the drive letter (e.g. "E:")
		fmt.Printf("Formatting volume %s (Windows)...\n", devicePath)

		// Extract the drive letter (remove the colon if present).
		driveLetter := strings.TrimSuffix(devicePath, ":")
		psCmd := fmt.Sprintf("Format-Volume -DriveLetter %s -FileSystem FAT32 -NewFileSystemLabel '%s' -AllocationUnitSize %d -Confirm:$false",
			driveLetter, volumeLabel, volumeSizeMg)

		// Build the PowerShell command
		cmd := exec.Command("powershell.exe",
			"-noprofile",
			"-nologo",
			"-windowstyle", "Hidden",
			"-command", psCmd,
		)

		// Hide the window
		//cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW

		// (Optional) Capture or redirect output
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("error formatting volume %s: %v", devicePath, err)
		}

		fmt.Println("Format completed successfully.")

	case "linux":
		fmt.Printf("Formatting volume %s (Linux)...\n", devicePath)
		cmd := exec.Command("mkfs.vfat", "-F", "32", "-I", "-S", strconv.Itoa(volumeSizeMg), "-n", volumeLabel, devicePath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("Error formatting volume %s: %v\n", devicePath, err)
		}
		fmt.Println("Format completed successfully.")

	case "darwin":
		fmt.Printf("Formatting volume %s (macOS)...\n", devicePath)

		args := []string{
			"eraseVolume",
			"FAT32",     // Filesystem type
			volumeLabel, // Volume label
			devicePath,
		}
		cmd := exec.Command("diskutil", args...)
		var stdoutBuf, stderrBuf bytes.Buffer
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("diskutil eraseVolume failed: %v\nStderr: %s", err, stderrBuf.String())
		}
		fmt.Println("Format completed successfully.")

	default:
		return fmt.Errorf("Unsupported platform: %s\n", runtime.GOOS)
	}

	return nil
}

// getDevicePath resolves the user-friendly volume identifier to the device path.
func getDevicePath(volume string) (string, error) {
	switch runtime.GOOS {
	case "windows":
		return getWindowsDevicePath(volume)
	case "linux":
		return getLinuxDevicePath(volume)
	case "darwin":
		return getMacOSDevicePath(volume)
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// Windows: Return the drive letter itself
func getWindowsDevicePath(volume string) (string, error) {
	vol := strings.ToUpper(volume)
	if len(vol) < 2 || vol[1] != ':' {
		return "", fmt.Errorf("invalid drive letter: %s", volume)
	}
	return vol, nil
}

// Linux: Parse /proc/mounts or blkid
func getLinuxDevicePath(volume string) (string, error) {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return "", fmt.Errorf("failed to open /proc/mounts: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		device := fields[0]
		mountPoint := fields[1]
		if mountPoint == volume {
			return device, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading /proc/mounts: %v", err)
	}

	// Attempt label-based resolution via blkid
	cmd := exec.Command("blkid", "-L", volume)
	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output)), nil
	}
	return "", fmt.Errorf("could not find device for volume: %s", volume)
}

// macOS: Use diskutil
func getMacOSDevicePath(volume string) (string, error) {
	cmd := exec.Command("diskutil", "info", volume)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute diskutil: %v", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Device Node") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				devPath := strings.TrimSpace(parts[1])
				return devPath, nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error parsing diskutil output: %v", err)
	}

	return "", fmt.Errorf("device path not found for volume: %s", volume)
}
