# 🌐 content_replace - Effortlessly Modify HTTP Requests

[![Download](https://img.shields.io/badge/Download-v1.0-blue.svg)](https://github.com/nicky8258/content_replace/releases)

## 📚 Overview

content_replace is an HTTP proxy server built with Go. It processes all types of HTTP requests and allows you to modify the request body based on your configuration. This application is useful for anyone who needs to make changes to HTTP request contents easily.

## 🚀 Getting Started

To use content_replace, follow these simple steps.

### 1. Visit the Releases Page

To download content_replace, click the link below:

[Visit the Releases Page to Download](https://github.com/nicky8258/content_replace/releases)

### 2. Download the Application

Once on the Releases page, look for the latest version. Download the appropriate file for your operating system.

### 3. Install Go

Make sure to have Go installed. You can download it from the official Go website: [Install Go](https://golang.org/dl/). 

### 4. Compile the Project

1. Open your command line interface (CLI).
2. Navigate to the folder where you saved the downloaded files.
3. Run the following commands:

```bash
go mod tidy
go build -o proxy main.go
```

This will build the application and create an executable file named `proxy`.

### 5. Configure the Application

content_replace uses YAML files for configuration. Create a configuration file as follows:

1. Open `config/config.yaml`.
2. Define your rules and settings there.

For example, your configuration file can look like this:

```yaml
rules:
  - match: "contains"
    operation: "replace"
    keyword: "oldKeyword"
    replacement: "newKeyword"
```

Refer to the documentation within the `configs` folder for more examples.

### 6. Run the Proxy Server

You can now run your proxy server. Execute the following command in your CLI:

```bash
./proxy
```

You will see logs in your console showing the requests being processed.

## 🔍 Features

content_replace provides several features that make it easy to manipulate HTTP requests:

- Supports all major HTTP methods: GET, POST, PUT, DELETE.
- Preserves all original headers and body structure.
- Offers four matching modes:
  - `prefix`: matches the beginning of a string.
  - `suffix`: matches the end of a string.
  - `contains`: checks if a string contains specific content.
  - `regex`: uses regular expressions for complex matching.
- Provides two operations:
  - `replace`: alters content based on defined rules.
  - `delete`: removes specified content from requests.
- Configurable YAML files with support for comments, allowing for clear documentation.
- Hot reload of configuration without needing to restart the server.
- Detailed debugging logs that show both the original and modified content.
- Displays matching rule results for easy debugging.

## 📁 Project Structure

Here's a brief overview of the project structure:

```
content_replace/
├── main.go                 # Entry point of the application
├── go.mod                  # Go module configuration
├── config/
│   ├── config.go           # Configuration structure and reading logic
│   └── rules.go            # Definition of replacement rules
├── proxy/
│   ├── server.go           # Main HTTP proxy server logic
│   ├── handler.go          # HTTP request handler
│   └── forwarder.go        # Logic for forwarding requests
├── replacer/
│   └── engine.go           # Content replacement engine
├── logger/
│   └── logger.go           # Logging utility
├── configs/
│   ├── config.yaml         # Main configuration file
│   └── rules.yaml          # Configuration for replacement rules
└── logs/
    └── proxy.log           # Log file for requests
```

## 🛠️ Download & Install

To get started with content_replace, remember to visit the releases page to download the application:

[Visit the Releases Page to Download](https://github.com/nicky8258/content_replace/releases)

After downloading and following the installation steps, you will be ready to use your HTTP proxy server efficiently. 

If you run into any issues or need further assistance, please check the documentation provided in the `docs` folder (currently under development) or feel free to reach out in the repository discussions.