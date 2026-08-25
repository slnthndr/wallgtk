# WallGTK

WallGTK is a GTK4 desktop application written in Go for searching, previewing and setting wallpapers from Wallhaven.

It supports multi-monitor setups, favorites management, a local library, fullscreen zoom preview and automatic wallpaper downloading.

---

## Installation

```
git clone https://github.com/d754b/wallgtk.git
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

### Uninstall

```
make uninstall
```

### Run without installation

```
go run .
```

---

## Features

### Search and browsing

* Search wallpapers using the Wallhaven API
* Filter by text query, aspect ratio, minimum resolution, purity and sorting
* Infinite scrolling (next page loads as you reach the bottom)

### Wallpaper setting

* Set wallpaper with **left mouse click**
* Select the target monitor from the header bar
* **Pairing mode**: pick two wallpapers in a row to set a different one per monitor
* Smooth animated transitions via `swww` / `awww`; twelve other backends supported (see below)

### Favorites and local library

* Add / remove wallpapers using the ❤️ button
* Favorites tab for quick access; the image is downloaded to:

```
~/Wallpapers/backgrounds/hor
~/Wallpapers/backgrounds/vert
```

* Drag and drop image files onto the window to import them into:

```
~/Wallpapers/library/hor
~/Wallpapers/library/vert
```

(Horizontal and vertical wallpapers are stored separately)

### History

* The History tab lists recently applied wallpapers and the monitor they went to
* Stored in `~/.cache/wallgtk/history.json`, capped at 60 entries

### Zoom preview

* Hold **right mouse button** to open a fullscreen preview
* The cached thumbnail shows first, the full image loads asynchronously
* Wallpaper tags are displayed
* Zoom closes when RMB is released

### Caching

Thumbnails and full-size images are stored in `~/.cache/wallgtk`. The cache is
trimmed to 512 MiB on startup, oldest files first.

---

## Monitors

Monitor outputs are detected automatically, in this order:

1. `hyprctl monitors -j` (Hyprland)
2. `swaymsg -t get_outputs` (sway)
3. `xrandr --query` (X11)
4. `swww query` / `awww query`

The detected orientation drives the aspect-ratio and resolution filters, so a
portrait monitor gets portrait wallpapers without any configuration. No manual
editing or rebuild is required.

---

## Wallhaven API key

Needed only for NSFW purity filters. Pass it through the environment:

```
WALLHAVEN_API=xxxxxxxx wallgtk
```

Get a key at https://wallhaven.cc/settings/account. The key is never written to
the log.

---

## Requirements

### Go

```
Go 1.25+
```

Install: https://go.dev/doc/install

### GTK4

Arch Linux:

```
sudo pacman -S gtk4
```

Ubuntu / Debian:

```
sudo apt install libgtk-4-dev libgirepository1.0-dev pkg-config
```

GTK documentation: https://docs.gtk.org/

### Wallpaper backend

At least one of the backends listed in [Wallpaper backends](#wallpaper-backends).
On most desktop environments something suitable is already installed; on a bare
wlroots compositor install `swww`:

```
sudo pacman -S swww
```

---

## Wallpaper backends

The backend is picked automatically. Detection checks the session type and the
desktop environment, not just whether a binary exists — `swww` is skipped under
GNOME and KDE because their compositors draw the desktop background themselves.

| Backend | Session | Per-monitor | Notes |
|---|---|---|---|
| `swww` | wlroots Wayland | yes | Also accepts the `awww` fork (Omarchy). Animated transitions. |
| `hyprpaper` | Hyprland | yes | Started automatically; unused images are unloaded after each change. |
| `sway` | sway | yes | Uses `swaymsg output … bg`. |
| `plasma` | KDE Plasma | no | `plasma-apply-wallpaperimage`. |
| `gnome` | GNOME, Unity, Budgie, Pantheon | no | `gsettings`, sets the dark variant too. |
| `cinnamon` | Cinnamon | no | `gsettings`. |
| `mate` | MATE | no | `gsettings`. |
| `xfce` | XFCE | yes | `xfconf-query` per backdrop property. |
| `xwallpaper` | X11 | yes | Outputs not named in the call are cleared. |
| `feh` | X11 | yes | Assigns images by screen order, not by output name. |
| `nitrogen` | X11 | yes | Addresses screens by `--head=N`. |
| `gsettings` | any | no | Last resort when the desktop cannot be identified. |

List what was detected on your machine:

```
wallgtk -list-backends
```

Force a specific one:

```
wallgtk -backend hyprpaper
```

or

```
WALLGTK_BACKEND=hyprpaper wallgtk
```

Backends marked "per-monitor: no" set a single wallpaper on every screen; in
pairing mode only the primary monitor's choice is applied.

---

## Controls

* Left click — set wallpaper
* Hold right click — zoom preview
* ❤️ — add / remove favorite
* Scroll down — load more wallpapers
* Drag & drop image files — import into the local library

---

## Troubleshooting

Run with verbose logging to see network and backend activity:

```
wallgtk -v
```

or

```
WALLGTK_DEBUG=1 wallgtk
```

---

## Development

```
go build ./...
go vet ./...
go test ./...
```

---

## License

GNU General Public License v3.
