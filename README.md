# RP Chat Logger

A simple and efficient chat message logging tool that captures messages from your game or application and sends them to Discord webhooks and/or saves them to local files.

## Features

- **Discord Integration**: Send messages directly to Discord channels via webhooks
- **Local File Logging**: Save messages to text, CSV, JSON, or DOCX files
- **Web UI**: User-friendly configuration interface accessible via browser
- **Live Monitoring**: Real-time log viewer and failure tracking (debug mode)
- **Rate Limiting**: Automatic retry mechanism for Discord rate-limited requests
- **Auto-Start**: Optionally start the server automatically on launch
- **Configuration Management**: All settings saved and persist between sessions

## Installation & Running

### Download & Run

1. Go to the [Releases](https://github.com/ragaz-zo/rp-chat-logger/releases) page
2. Download the latest `rp-chat-logger.exe` file
3. Run the executable
4. The web UI will automatically open in your browser at `http://127.0.0.1:8080`

### Build from Source (Optional)

Requirements: Go 1.21 or later

```bash
git clone https://github.com/ragaz-zo/rp-chat-logger.git
cd rp-chat-logger
go build
rp-chat-logger.exe
```

**Optional: Reduce Antivirus False Positives**

To embed a Windows manifest (reduces heuristic detection):

1. Install rsrc: `go install github.com/akavel/rsrc@latest`
2. Build with manifest: `rsrc -manifest rp-chat-logger.manifest -o rsrc.syso && go build && rm rsrc.syso`

Or use the release script: `./release.sh 1.0.0` (manifest embedding is automatic if rsrc is available)

## Configuration

Access the web UI to configure the application:

### Discord Notifications
1. **Enable Discord Notifications**: Toggle to enable Discord integration
2. **Webhook URL**: Get a webhook URL from your Discord server settings
   - Right-click channel → Edit Channel → Integrations → Webhooks → New Webhook
   - Copy the webhook URL into the configuration

### File Logging
1. **Enable File Logging**: Toggle to enable local file storage
2. **File Path**: Directory where log files will be saved (e.g., `C:\Logs` or `C:\Users\YourName\Documents\Logs`)
3. **Format**: Choose file format:
   - `txt`: Plain text, human-readable format
   - `csv`: Comma-separated values for spreadsheets
   - `json`: JSON format for programmatic access
   - `docx`: Microsoft Word document format

### Server Settings
- **Listen Address**: The address the message receiver listens on (default: `0.0.0.0:3000`)
- **Auto Start Server**: Automatically start the ingestion server when the app launches
- **Debug Mode**: Shows live server logs and failed messages in the web UI

## Sending Messages

Send POST requests to the ingestion server with this format:

```bash
curl -X POST http://localhost:3000/message \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "sender=PlayerName&message=Hello%20World"
```

Or as form data:
- `sender`: The name/ID of the message sender
- `message`: The message content to log

## Important Notes

- **At least one output option** (Discord or File Logging) must be enabled to run the server
- **Configuration is required** before the ingestion server can start:
  - Discord: Need a valid webhook URL if enabled
  - File Logging: Need a valid directory path if enabled
- The **Web UI** always runs on the configured port, even if the ingestion server fails to start
- Changes to configuration take effect immediately

## Troubleshooting

**Server won't start**
- Check that at least one output option (Discord or File Logging) is enabled
- If Discord is enabled, ensure the webhook URL is valid
- If File Logging is enabled, ensure the directory path exists or is writable

**Discord messages not sending**
- Verify the webhook URL is correct and still valid
- Check that Discord notifications are enabled in the configuration
- Enable Debug Mode to see detailed error messages

**No logs visible**
- Enable Debug Mode to see the Live Server Logs section
- Make sure the ingestion server is running (green status indicator)
- Check that messages are being sent to the correct server address and port

**Antivirus warns about IDP.Generic or other threats**
- This is a false positive. The executable is unsigned and performs network operations, which triggers heuristic detection
- **Solution**: Whitelist the executable in your antivirus
  - Windows Defender: Right-click file → Scan with Windows Defender → See detailed results → Allow on device
  - Other antivirus: Look for "whitelist" or "exclude" options in your antivirus settings
- **Long-term fix**: A code signing certificate would eliminate this, but isn't essential for personal use

## Support

- Report issues on [GitHub Issues](https://github.com/ragaz-zo/rp-chat-logger/issues)
- Check existing discussions and documentation on the repository

## License

See LICENSE file for details.
