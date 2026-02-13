# budsctl

A small daemon and CLI tool for Linux that lets you deliberately connect and disconnect Bluetooth earbuds, preventing unwanted auto-reconnection.

## The problem

Bluetooth earbuds that pair with multiple devices (e.g. Apple AirPods) will try to auto-connect to your Linux machine whenever they're in range — even when you want them connected to your phone. In theory, setting `Trusted = false` in BlueZ should prevent auto-connect, but in practice this isn't reliable. budsctl takes a more heavy-handed approach — **blocking** the device after every disconnect — which reliably prevents reconnection. See [this my post](https://mil.ad/blog/2024/airpods-on-arch.html) for a longer explanation of the problem and the approach budsctl takes.

The key idea is to **block** the device in BlueZ immediately after every disconnect. This prevents unwanted auto-reconnection. When you actually want to connect, budsctl unblocks the device and initiates the connection. This gives you deliberate, explicit control over when your earbuds connect to your computer.

## How it works

budsctl uses a daemon/client architecture: a long-running daemon talks to BlueZ over the system D-Bus and watches for disconnect events (automatically blocking the device), while lightweight client commands communicate with it over a Unix socket.

## Usage

```
budsctl <daemon|status|toggle <device>>
```

**`daemon`** — Start the background daemon. It listens on `$XDG_RUNTIME_DIR/budsctl.sock` and watches for BlueZ property changes (e.g. auto-blocking a device when it disconnects).

**`status`** — Print the current state of the active device as JSON.

**`toggle <device>`** — Toggle the connection state of a device by MAC address. When switching between devices, the previously active device is automatically disconnected and blocked.

The toggle cycle works as follows:

| Current State | Action |
|---|---|
| connected | Disconnect and block |
| connecting | Block |
| idle | Unblock and connect |
| disabled | Power on adapter, unblock, and connect |

## Requirements

- Linux with BlueZ (`bluetooth.service` must be running)
- Go 1.25+

## License

MIT
