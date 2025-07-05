<p align="center">
  <a href="https://github.com/blacktop/go-gitfamous"><img alt="go-gitfamous Logo" src="https://raw.githubusercontent.com/blacktop/go-gitfamous/main/docs/logo.webp" /></a>
  <h1 align="center">go-gitfamous</h1>
  <h4><p align="center">Github Event Tracker TUI</p></h4>
  <p align="center">
    <a href="https://github.com/blacktop/go-gitfamous/actions" alt="Actions">
          <img src="https://github.com/blacktop/go-gitfamous/actions/workflows/go.yml/badge.svg" /></a>
    <a href="https://github.com/blacktop/go-gitfamous/releases/latest" alt="Downloads">
          <img src="https://img.shields.io/github/downloads/blacktop/go-gitfamous/total.svg" /></a>
    <a href="https://github.com/blacktop/go-gitfamous/releases" alt="GitHub Release">
          <img src="https://img.shields.io/github/release/blacktop/go-gitfamous.svg" /></a>
    <a href="http://doge.mit-license.org" alt="LICENSE">
          <img src="https://img.shields.io/:license-mit-blue.svg" /></a>
</p>
<br>

## Why? 🤔

There are many *kinds* of being famous. This is just a very nerdy one. 🤓

## Getting Started

### Install

```bash
brew install blacktop/tap/gitfamous
```

Or

```bash
go install github.com/blacktop/go-gitfamous@latest
```

Or download the latest [release](https://github.com/blacktop/go-gitfamous/releases/latest)

### Run

#### Single User Mode

```bash
> gitfamous --help

Github Event Tracker TUI

Usage:
  gitfamous <username> [flags]

Flags:
  -t, --api string       Github API Token
  -c, --count int        Number of events to fetch
  -f, --filter strings   Comma-separated list of event types to display
  -h, --help             help for gitfamous
  -s, --since string     Limit events to those after the specified amount of time (e.g. 1h, 1d, 1w)
  -V, --verbose          Verbose output
```   

#### Multi-User Mode

Create a configuration file at `~/.config/gitfamous/config.yml`:

```yaml
users:
  - username: "user1"
  - username: "user2"
    token: "optional_user_specific_token"
    
default_settings:
  count: 50
  since: "1w"
  filter: ["PushEvent", "PullRequestEvent", "CreateEvent"]
```

Then run gitfamous without any arguments:

```bash
> gitfamous
```

This will launch a tabbed interface where you can switch between users using arrow keys or j/l.

![demo](vhs.gif)

## License

MIT Copyright (c) 2024-2025 **blacktop**