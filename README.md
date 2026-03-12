## Installation

```
git clone https://github.com/slnthndr/wallgtk.git
cd wallgtk
make install
```

Binary will be installed to:

```
/usr/local/bin/wallgtk
```

Run:

```
wallgtk
```

---

## Uninstall

```
make uninstall
```

---

## Run without installation

```
go run .
```

---

# WallGTK

WallGTK is a GTK4 desktop application written in Go for searching, previewing and setting wallpapers from Wallhaven.

It supports multi-monitor setup, favorites management, fullscreen zoom preview and automatic wallpaper downloading.

---

## Features

### Search and browsing

* Search wallpapers using Wallhaven API
* Filter by:

  * text query
  * aspect ratio
  * minimum resolution
  * sorting method
* Infinite scrolling (automatic page loading)

### Wallpaper setting

* Set wallpaper with **left mouse click**
* Select target monitor
* Smooth animated transitions using `swww`

### Favorites

* Add / remove wallpapers using ❤️ button
* Favorites tab for quick access
* Favorites list is stored in:

```
~/.cache/wallgtk/favorites.json
```

* When adding to favorites the image is automatically downloaded to:

```
~/Wallpapers/backgrounds/hor
~/Wallpapers/backgrounds/vert
```

(Horizontal and vertical wallpapers are stored separately)

### Zoom preview

* Hold **right mouse button** to open fullscreen preview
* Low-resolution thumbnail is shown first
* Full image loads asynchronously
* Wallpaper tags are displayed
* Zoom closes when RMB is released

### Caching

All thumbnails and images are stored in:

```
~/.cache/wallgtk
```

---

## Monitor selection

The application allows selecting:

* All monitors
* Primary monitor
* Secondary monitor

### IMPORTANT

Monitor output names are hardcoded in `config.go`:

```go
MonitorOutputs = map[string]string{
    "Primary monitor":   "DP-2",
    "Secondary monitor": "DP-1",
}
```

You must change these values to match your system.

### How to get monitor output names

Wayland:

```
hyprctl monitors
```

or

```
swww query
```

X11:

```
xrandr
```

After changing monitor outputs you must rebuild the application:

```
make install
```

---

## Requirements

### Go

Minimum version:

```
Go 1.21+
```

Install: https://go.dev/doc/install

### GTK4

Arch Linux:

```
sudo pacman -S gtk4
```

Ubuntu / Debian:

```
sudo apt install libgtk-4-dev
```

GTK documentation: https://docs.gtk.org/

### swww

Used for setting wallpapers.

Arch:

```
sudo pacman -S swww
```

---

## Controls

* Left click — set wallpaper
* Hold right click — zoom preview
* ❤️ — add / remove favorite
* Scroll down — load more wallpapers

---

## Configuration

Main settings are located in `config.go`:

* Wallhaven API key
* Monitor outputs
* Supported resolutions
* Aspect ratios
* Tile sizes
* Cache directories

Rebuild is required after changing configuration.

---

## License

GNU General Public License v3.

