# gitrip

gitrip is a simple CLI tool for quickly downloading specific files or directories from a Git repository. It uses git sparse-checkout under the hood, making it fast and bandwidth-efficient—no need to clone the entire repository!

## Features

- Download only the files or directories you need from any GitHub repo
- uses git sparse-checkout for speed and minimal downloads
- Easy, single-command installation

## Installation

Make sure you have [git](https://git-scm.com/) installed on your system.

Then, install gitrip with:

```bash
go install github.com/alireza-karampour/gitrip@latest
```

## Usage

```bash
gitrip [flags] -r <repo-url> -t <tree> -p <paths> -d <dest>
```

- `<repo-url>`: Full URL of the target Git repository
- `<paths>`: comma separated list of paths to files or directories to download, relative to the root of the repo
- `<tree>`: name of a branch or hash of a commit that files should be downloaded from
- `<dest>`: path to the directory that files should be downloaded into

_Example:_

```bash
gitrip -r https://github.com/some-repo.git -t develop -p "/Taskfile.yml,/cmd" -d /home
```

## Requirements

- [git](https://git-scm.com/) must be installed on your system
- Go 1.18 or newer (for building from source)

## Contributing

Contributions, issues, and feature requests are welcome! Feel free to fork the repository and submit a pull request.

## License

This project is licensed under the MIT License.
