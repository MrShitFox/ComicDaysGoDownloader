# ComicDaysGoDownloader

[![Go Version](https://img.shields.io/badge/Go-1.23.1%2B-blue)](https://golang.org/) [![License: GPLv3](https://img.shields.io/badge/License-GPLv3-red)](LICENSE)

Tool for downloading and deobfuscating manga from ComicDays. Bypasses DRM protection for offline reading and usage.

## 🚀 Key Features

- **Direct download** from ComicDays URLs
- **DRM deobfuscation** for clean images
- **Session maintenance** via cookie authentication
- Sequential processing with progress tracking
- Cross-platform CLI (Win/macOS/Linux)

## ⚡ Quick Start

### Pre-built Binaries

1. Download latest release from [Releases page](https://github.com/MrShitFox/ComicDaysGoDownloader/releases)
1. Unzip archive
1. Run:

    <details open>
    <summary><strong>On Windows:</strong></summary>

    ```powershell
    .\ComicDaysGoDownloader.exe
    ```

    </details>

    <details open>
    <summary><strong>On Unix/macOS:</strong></summary>

    ```bash
    chmod +x ComicDaysGoDownloader
    ./ComicDaysGoDownloader
    ```

    </details>

### From Source

1. Clone the repository:

   ```bash
   git clone https://github.com/MrShitFox/ComicDaysGoDownloader.git
   cd ComicDaysGoDownloader
   ```

1. Build the binary:

   <details open>
   <summary><strong>For Unix/macOS:</strong></summary>

   ```bash
   go build -o ComicDaysGoDownloader .
   # Note: The dot (.) tells Go to build the package in the current directory.
   # Do not replace it with "main.go"
   ```

   </details>

   <details open>
   <summary><strong>For Windows (Command Prompt or PowerShell):</strong></summary>

   ```powershell
   go build -o ComicDaysGoDownloader.exe .
   # Note: The dot (.) tells Go to build the package in the current directory.
   # Do not replace it with "main.go"
   ```

   </details>

## 🔑 Auth Setup

Create `cookie.json` in root directory:

```json
[
    {
        "domain": "comic-days.com",
        "expirationDate": 1759754874.804194,
        "hostOnly": true,
        "httpOnly": true,
        "name": "glsc",
        "path": "/",
        "sameSite": null,
        "secure": true,
        "session": false,
        "storeId": null,
        "value": "YOUR_COOKIE_VALUE"
    }
]
```

*Use browser devtools to extract fresh cookie values, I recommend this [extension](https://cookie-editor.com), just hit export and select json.*

## 📚 Usage Example

Download single chapter:

```bash
./ComicDaysGoDownloader
```

<sub>(On Windows, use `.\ComicDaysGoDownloader.exe` instead.)</sub>

Next, enter the URL of the desired manga, wait a bit, and you're done.

## ⚖️ Legal Notice

**ComicDaysGoDownloader** is intended for personal use only. Please respect the copyright and terms of service of the Comic Days website. The authors are not responsible for any misuse or violations of Comic Days' terms of service, and blah blah blah.

---

[![Star this Repo](https://img.shields.io/github/stars/MrShitFox/ComicDaysGoDownloader?style=social)](https://github.com/MrShitFox/ComicDaysGoDownloader/stargazers) *Consider giving a star if you find this useful*
