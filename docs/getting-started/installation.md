# Installation

## macOS

```bash
brew install bulga138/homebrew-hookset/hookset
```

## Linux / WSL

```bash
curl -fsSL https://raw.githubusercontent.com/bulga138/hookset/master/scripts/install.sh | sh
```

## Windows

```powershell
irm https://raw.githubusercontent.com/bulga138/hookset/master/scripts/install.ps1 | iex
```

## Requirements

- **Git 2.54 or later**

Check your git version:

```bash
git --version
```

If you need to upgrade:

```bash
# macOS
brew upgrade git

# Ubuntu
sudo add-apt-repository ppa:git-core/ppa
sudo apt-get update && sudo apt-get install git
```

## Verify Installation

```bash
hookset version
```

## Development Setup

```bash
git clone https://github.com/bulga138/hookset
cd hookset
make build-dev
./hookset version
```
