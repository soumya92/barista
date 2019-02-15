---
title: Bluetooth
---

Display information about a bluetooth adapter: `bluetooth.Adapter("hci0")`.  
Display information about a specific device: `bluetooth.Device("hci0", "B0:2A:43:AB:CD:EF")`.

The bluetooth module uses D-Bus to communicate with a running bluez instance.

## Configuration

* (Adapter) `Output(func(bluetooth.AdapterInfo) bar.Output)`: Sets the output format.
* (Device) `Output(func(bluetooth.DeviceInfo) bar.Output)`: Sets the output format.

## Data

### `type AdapterInfo struct`

* `Name string`: The name of the local machine as shown over bluetooth. Usually the hostname.
* `Alias string`: A configurable alternative to `Name` that is only used within the system. Not recommended. 
* `Address string`: The MAC address of the adapter.
* `Discoverable bool`: Whether the adapter is discoverable.
* `Pairable bool`: Whether devices can be paired with the adapter.
* `Powered bool`: Whether the adapter is powered on.
* `Discovering bool`: Whether the adapter is actively searching for other discoverable devices.

### `type DeviceInfo struct`

* `Name string`: Name of the device, as set by the remote device.
* `Alias string`: Name of the device, as set by the user (defaults to `Name` if not set).
* `Address string`: MAC address of the device.
* `Adapter string`: The blue-z adapter path used to obtain this device (`/org/bluez/...`).
* `Battery int`: Percentage of battery remaining on the device (0 to 100).
* `Paired bool`: Whether the device is paired.
* `Connected bool`: Whether the device is currently connected.
* `Trusted bool`: Whether the device is trusted.
* `Blocked bool`: Whether the device is blocked. All connections from blocked devices are rejected.
