# Daikin Skyport API

## Overview

Daikin Skyport API is a library for interacting with Daikin One+ devices/thermostats.

Note: This library uses an undocumented API that is currently used by the [Daikin One Home mobile app](https://www.daikinone.com/product/one-home-mobile-app). As such, the API could change at any moment and break this library. For a more stable, but less feature rich library based on published docs, check out the [Daikin Skyport Integrator API](https://github.com/redgoose/daikin-skyport-integrator).

## Installation

```
go get -u github.com/redgoose/daikin-skyport
```

## Usage

This library requires the email and password associated with your Daikin account.

### Get a list of all devices associated with your account

```
d := daikin.New("your@email.com", "yourPassword")
devices, err := d.GetDevices()
```

### Get info for a given deviceId

```
d := daikin.New("your@email.com", "yourPassword")
deviceInfo, err := d.GetDeviceInfo("0000000-0000-0000-0000-000000000000")
```

## Testing

Run all integration tests from the root folder by running:
```
go test ./... -v
```