# ComicDaysGoDownloader

**ComicDaysGoDownloader** is an open-source tool written in Go (version **1.23.1** or higher) for downloading and deobfuscating manga from the [Comic Days](https://comic-days.com/) website.

## Features

- **Download Manga:** Easily download manga chapters from the Comic Days website.
- **Automatic Deobfuscation:** Automatically deobfuscates downloaded pages for seamless reading.
- **Cookie-Based Authentication:** Supports authenticated downloads using cookies.
- **Sequential Page Processing:** Downloads and processes pages in order.
- **Simple CLI:** User-friendly command-line interface for effortless operation.

## How It Works

ComicDaysGoDownloader is a console application that downloads manga from the Comic Days website directly to your computer. Follow these steps to use the program:

1. **Run the Program:** Execute the `ComicDaysGoDownloader` binary.
2. **Enter Manga URL:** Provide a valid URL of the manga episode you wish to download.
   - *Example URL:* `https://comic-days.com/episode/2550689798731966148`
3. **Download Process:** The tool will download all pages and save them in a newly created folder named with the current date and time.

## Installation

### Download Pre-Built Binaries

1. Visit the [Releases](https://github.com/MrShitFox/ComicDaysGoDownloader/releases) page of this repository.
2. Download the latest version for your operating system (Windows, macOS, or Linux).
3. Extract the downloaded archive to a desired location.

## Usage

### Running the Downloader

1. **Open Terminal or Command Prompt.**
2. **Navigate to the Executable Folder:**
   - Example:
     ```sh
     cd path/to/ComicDaysGoDownloader
     ```
3. **Run the Program:**
   - On Windows:
     ```sh
     ComicDaysGoDownloader.exe
     ```
   - On macOS/Linux:
     ```sh
     ./ComicDaysGoDownloader
     ```
4. **Enter Manga URL:** When prompted, input the URL of the manga episode you want to download.
5. **Wait for Completion:** The tool will handle downloading and deobfuscating all pages automatically.

### Using Cookies for Authentication

If the manga requires authentication, create a `cookie.json` file in the same directory as the executable with the following structure:

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

- **Replace `YOUR_COOKIE_VALUE`** with your actual cookie value to enable authenticated downloads.

## Contributing

Contributions are welcome! If you'd like to contribute to ComicDaysGoDownloader:

1. Fork the repository.
2. Create a new branch for your feature or bugfix.
3. Commit your changes with clear messages.
4. Submit a pull request.

Please ensure your contributions adhere to the project's coding standards and include necessary tests.

## License

This project is licensed under the **GNU General Public License v3.0**. See the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by [ComicDaysDownloader](https://github.com/igorquintaes/ComicDaysDownloader) by igorquintaes.

## Disclaimer

**ComicDaysGoDownloader** is intended for personal use only. Please respect the copyright and terms of service of the Comic Days website. The authors are not responsible for any misuse or violations of Comic Days' terms of service.
