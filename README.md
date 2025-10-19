[Download page](https://github.com/leon-knoke/wago-init/releases)
-

**Launch instructions**
- **Windows:** download the Windows build, double-click the `.exe`, choose **More info**, then **Run anyway** when SmartScreen appears.
- **Linux:** download the Linux build, open the file properties, enable **Allow executing file as program**, and launch the binary.
- **macOS:** download the darwin bundle, open it once to trigger the warning, close the dialog, then go to **System Settings → Privacy & Security**, click **Open Anyway**, and confirm.

# wago-init

`wago-init` is a desktop provisioning assistant for WAGO CC100 controllers. It guides the operator through credential capture, firmware updates, service configuration, Docker container deployment, and configuration file transfer — all in a single workflow.

The application is built with Go and Fyne, features live logging and progress tracking, and persists environment settings for quick repeat runs on the production line.

## Key capabilities
- **Device discovery:** scan IP ranges, wildcards, or CIDR blocks to locate supported devices and auto-fill the target IP.
- **Credential management:** prompt for AWS ECR credentials, Docker flags, and firmware sources, storing them securely in the local env config file.
- **Firmware automation:** upload `.wup` packages, monitor status, and verify the applied revision.
- **Service & container setup:** configure required system services, authenticate to AWS, and create the target Docker container with stored runtime flags.
- **Config delivery:** copy prepared configuration directories to the controller over SSH using a tar-over-stdin transport.
- **Operator UX:** live, timestamped log pane with replaceable status lines, progress bar animation, and clear error handling.

## Prerequisites
- A workstation with basic OpenGL support.
- Operator knowledge of target WAGO CC100 network settings.
- Valid AWS ECR credentials for container image pulls.
- Firmware `.wup` package and configuration directory available locally.

## Quick start
1. Launch the application (see platform instructions above).
2. (Optional) Click **Device discovery** to scan for controllers and auto-fill the IP address field.
3. Open **AWS settings** and provide Region, Account ID, Access ID, and Access Key.
4. Configure container settings (image URI and optional `docker run` flags) via **Container settings**.
5. Configure firmware source (revision target and `.wup` path) through **Firmware settings** if updates are required.
6. Select the configuration folder to copy via the **Search** button next to the config path entry.
7. Click **Start**, supply device passwords when prompted, and monitor the log output while the workflow runs.
8. When the progress bar reaches 100% and the log reports **Done.**, disconnect the device or move to the next unit.

## What happens when you click “Start”
1. Connection and MAC validation of the target controller.
2. Interactive password update prompts and credential storage for subsequent SSH calls.
3. Firmware upload, extraction, `fwupdate` activation, progress polling, and post-reboot reconnection.
4. System service configuration and Docker container creation using the saved AWS token and flags.
5. Recursive copy of the chosen config directory to `/root` on the device.
6. Final verification of firmware revision and overall success reporting.

## Logs and troubleshooting
- The log pane timestamps each message and supports inline replacement for periodic status updates.
- Progress bar animates smoothly between reported checkpoints; if it stalls, review the log for SSH or firmware messages.
- Errors present a dialog and reset the UI to the idle state so the operator can adjust inputs and retry.

## Building from source

```bash
cd src
go build ./cmd/wago-init
```

## License

This project is released under the MIT License. See [`LICENSE`](LICENSE) for details.
