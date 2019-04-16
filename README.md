
# GoGoDownload
> A tool for downloading anime from GoGoAnime [![Build Status](https://travis-ci.org/MikeModder/GoGoDownload.svg?branch=master)](https://travis-ci.org/MikeModder/GoGoDownload)
## What is GoGoDownload?
As mentioned above GoGoDownload is a tool written to aid users in the process of downloading anime from gogoanime.tv. GGD scrapes each of the episode's pages and extracts the mp4 link for you, then automatically starts downloading them with [aria2](https://aria2.github.io/).
## Usage
> You need [aria2](https://aria2.github.io/) installed on your system for GGD to function properly.

Using GGD is pretty simple. Download the binary for your system and then run is like this
```
./GoGoDownload <series url> 1 12
```
This will download episodes 1 through 12 of the anime you provided. Currently GGD does not support searching for anime, as such a GoGoAnime link is required.
## Todo
These are some things I want to eventually add to this tool
* Searching (ex `./GoGoAnime --anime "Steins;Gate"`)
* Allow use of other downloaders (not just aria2)
* Make it more user friendly (maybe a GUI of some sort?)
