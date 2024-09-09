<h1 align="center">torgo</h1>
<p align="center">
  <img alt="GitHub commit activity" src="https://img.shields.io/github/commit-activity/m/stl3/torgo"/>
  <img alt="Github Last Commit" src="https://img.shields.io/github/last-commit/stl3/torgo"/>  
  <img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/stl3/torgo"/>  
</p>
<p align="center"><strong><i>A mighty all-in-one magnet scraper & streamer</i></strong></p>

<p align="center"><img src="demo.gif" width=100%></p>

<p align="center">
     <a href="./LICENSE.txt"><img alt="license" src="https://img.shields.io/github/license/tnychn/torrodle.svg"></a>
</p>

<div align="center">
    <strong>
    <a href="#features">Features</a> |
    <a href="#installation">Install</a> |
    <a href="providers.md">Providers</a> |
    <a href="CLI.md">CLI</a> |
    <a href="API.md">API</a> 
    </strong>
</div>
[![Go Report Card](https://goreportcard.com/badge/github.com/stl3/torgo)](https://goreportcard.com/report/github.com/stl3/torgo)  
---

**torgo** is a command-line program which searches and gathers magnet links of movies, tv shows, anime, porn & documentary videos from a variety of [providers](providers.md).
It then streams the torrent via HTTP (along with its subtitles) and plays it with a user preferred video player (such as *vlc* and *mpv*).
Currently I am only getting this to be in a working state. 

<sub>Note: The original Torrodle repo has been archived since April 2023 and has not been updated in a while, I liked the project (thanks for inspiring me tnychn!) so I just wanted to get it in a working state again.
I have also made modifications so that it deletes the downloads from your `os.TempDir()` or the one you set as `DataDir` in `~/.torgo.json` when quitting the program. Since I am only trying to get it in a working state, options will be added later to toggle keeping/deleting downloads.
I personally do it this way because I have mapped my gpu vram as a disk so that my mechanical drive suffers no wear and tear.
> If you don't know what BitTorrent is, you shouldn't be using **torgo**.
> There are some copyrighted content which might be illegal downloading them in your country.

## Table of Contents
- [Features](#features)
- [Installation](#installation)
  - [Binary](#binary)
  - [Go Get](#go-get)
  - [Build From Source](#build-from-source)
    - [Dependencies](#dependencies)
- [Usage](#usage)
- [Configuration File](#configuration-file)
- [Caveats](#caveeats)
- [Contributing](#contributing)
- [Credit](#credit)

    
## Features

* 🔥 Blazing fast
* 🚸 User-friendly (debatable)
* 🤖 Built-in torrent streaming client via HTTP (refined from [`go-peerflix`](https://github.com/Sioro-Neoku/go-peerflix))
* 🔰 Watch the video while it is being downloaded / allows going to any point in the video
* 🔎 Query multiple providers in a single search
* 🚀 Sorted results from 13 different providers at once 
* 📄 Along with subtitles fetching for the video (using [`osdb`](https://github.com/oz/osdb))
* 💥 Select individual episode from complete seasons/packs
* Support for playing with mpv/vlc on Win/Linux/Mac(?)/Android (Termux) *Android does not have subtitle support yet unless embedded*

## Installation

### Binary

Download the latest stable release of the binary at [releases](https://github.com/stl3/torgo/releases).

### Go Get

Make sure you have **Go 1.12+** installed on your machine.

`$ go get github.com/stl3/torgo/cmd/...`

### Build From Source

Make sure you have **Go 1.12+** installed on your machine.

```shell script
$ git clone github.com/stl3/torgo.git
$ cd torgo
$ go build cmd/torgo/main.go
```
If you have upx you can make a smaller executable:
```shell script
$ go build -o torgo.exe -ldflags="-s -w" .\cmd\torgo\main.go ; upx -9 -k torgo.exe
```

### Dependencies

See [`go.mod`](./go.mod).

This project depends on the following Go libraries:

Direct Dependencies:

    github.com/PuerkitoBio/goquery - v1.8.1 (Web scraping)
    github.com/anacrolix/torrent - v1.53.2 (BitTorrent client)
    github.com/avast/retry-go - v3.0.0+incompatible (Retrying operations with backoff)
    github.com/briandowns/spinner - v1.23.0 (Terminal spinner for visual feedback)
    github.com/dustin/go-humanize - v1.0.1 (Formats numbers as human-readable strings)
    github.com/fatih/color - v1.16.0 (Terminal color manipulation)
    github.com/olekukonko/tablewriter - v0.0.5 (Pretty table printing)
    github.com/oz/osdb - v0.0.0-20221214175751-f169057712ec (OS database information)
    github.com/sirupsen/logrus - v1.9.3 (Structured logging library)
    golang.org/x/net - v0.19.0 (Standard library network extensions)
    golang.org/x/time - v0.5.0 (Time manipulation extensions)
    
## Usage

For command-line (CLI) usage, see [`CLI.md`](CLI.md).

For API usage, see [`API.md`](API.md).

## Configuration file
Please check [`example.torgo.json`](example.torgo.json)
##### OS PATHS for configuration file
###### Linux
${XDG_CONFIG_HOME:-${HOME}/.config}/torgo/config
###### MacOS
${HOME}/Library/Application Support/torgo/config
###### Termux
$HOME/.config/torgo/config
###### Windows
%APPDATA%\torgo\config
%APPDATA%\.config\torgo\config

```
"DataDir": "E:/exampledir",
"ResultsLimit": 150,
"TorrentPort": 56666,
"HostPort": 8080,
"Proxy": "https://example_proxy_address:8008",
"EstablishedConnsPerTorrent": 25,
"HalfOpenConnsPerTorrent": 25,
"TotalHalfOpenConns": 50,
"eztv_cookie": "cookie_value"
"Debug": false
```
Change `DataDir` if you want a custom path for where it downloads files
Change `TorrentPort` to the port you open/forwarded to use with torrents
Change `HostPort` to the port you want to host the file from
Change `Proxy` to whatever proxy you want to use ([anacrolix/torrent](https://pkg.go.dev/github.com/anacrolix/torrent?utm_source=godoc#ClientConfig.HTTPProxy) - seems to use this only for fetching metainfo and webtorrent seeds but not for the transport of down/up traffic itself

## Caveeats

In Android, when quitting the player you have to manually stop the server. On Windows this is not an issue.

## Contributing

If you have any ideas on how to improve this project or if you think there is a lack of features,
feel free to open an issue, or even better, open a pull request. All contributions are welcome!

## Credit

Original repo [now archived since April 2023]: Made with ♥︎ by tnychn MIT © 2019 Tony Chan 
[@tnychn](https://github.com/tnychn)'s [torrodle](https://github.com/tnychn/torrodle).

This project is inspired by [@Fabio Spampinato](https://github.com/fabiospampinato)'s [cliflix](https://github.com/fabiospampinato/cliflix).

Torrent streaming technique adapted from [@Sioro Neoku](https://github.com/Sioro-Neoku)'s [go-peerflix](https://github.com/Sioro-Neoku/go-peerflix).



