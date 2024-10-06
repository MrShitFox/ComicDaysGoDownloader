# ComicDaysGoDownloader

ComicDaysGoDownloader is an open-source tool written in Go for downloading and deobfuscating manga from the Comic Days website.

## Features

- Download manga from Comic Days website
- Automatic deobfuscation of downloaded pages
- Cookie-based authentication support
- Sequential page processing
- Simple command-line interface

## How It Works

ComicDaysGoDownloader is a console application that allows you to download manga from the Comic Days website to your computer. To use the program, you need to run it and insert a valid URL of the manga page you want to download. All pages will be downloaded into a separate folder created in the same location as the program. The folder name will contain the date and time of the download.

Example URL to insert into the program: `https://comic-days.com/episode/2550689798731966148`

## Installation

### Option 1: Download pre-built binaries

1. Go to the [Releases](https://github.com/your-username/ComicDaysGoDownloader/releases) page of this repository.
2. Download the latest version for your operating system (Windows, macOS, or Linux).
3. Extract the downloaded archive to a folder of your choice.

### Option 2: Build from source

1. Ensure you have Go installed (version 1.16 or higher).
2. Clone the repository:
   ```
   git clone https://github.com/MrShitFox/ComicDaysGoDownloader.git
   ```
3. Navigate to the project directory:
   ```
   cd ComicDaysGoDownloader
   ```
4. Install dependencies:
   ```
   go mod tidy
   ```
5. Build the executable:
   ```
   go build -o ComicDaysGoDownloader
   ```

## Usage

### Using pre-built binaries

1. Open a terminal or command prompt.
2. Navigate to the folder containing the ComicDaysGoDownloader executable.
3. Run the program:
   - On Windows: `ComicDaysGoDownloader.exe`
   - On macOS/Linux: `./ComicDaysGoDownloader`
4. Insert the manga URL when prompted by the program.
5. Wait for the download and deobfuscation process to complete.

### Using source code

1. Open a terminal or command prompt.
2. Navigate to the project directory.
3. Run the program:
   ```
   go run main.go
   ```
4. Insert the manga URL when prompted by the program.
5. Wait for the download and deobfuscation process to complete.

### Using Cookies

If you need to use authentication to access certain manga, create a `cookie.json` file in the same directory as the executable (or in the project root if running from source) with the following content:

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

Replace `YOUR_COOKIE_VALUE` with the actual cookie value.

## Contributing

We welcome contributions to the project! Please read our contributing guidelines before submitting a pull request.

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

This project is inspired by [ComicDaysDownloader](https://github.com/igorquintaes/ComicDaysDownloader) by igorquintaes. We are grateful for the deobfuscation idea that we adapted for our Go project.

## Disclaimer

This tool is for personal use only. Please respect the copyright and terms of service of the Comic Days website. The authors of this tool are not responsible for any misuse or any violations of Comic Days' terms of service.
