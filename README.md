<p align="center">
    <img src="docs/assets/logo.png" alt="drawing" width="200"/>
</p>

# Raspberry Pi Alpine image builder

The repository contains code of command line tool for building alpine image
for Raspberry PI devices. It should provide easier way of deployment.

# Getting started

## Using CLI

First build cmd tool by:

``` go build -o ./build/rpialp ./cmd/cli.go  ```

Run command:

``` ./build/rpialp build  ```

Help command:

``` ./build/rpialp --help  ```

## Using desktop app

To get the app first visit the releases. Then download the base on your platform. 
More info about the development visit tho docs [here](desktop/README.md)

### MacOS

First unzip the file:

``` unzip <download-path>/rpi-alpine-builder-desktop-macos  ```

Open the terminal and run the binary from command line

``` <download-path>/rpi-alpine-builder-desktop-macos  ```

#### Permission issue 

If you are facing issue with the verification of *rpi-alpine-builder-desktop-macos* 
then you have to allow in System Settings -> Security and privacy and here you should enable
to open *rpi-alpine-builder-desktop-macos*

### Windows

> TODO - not yet test it
 
First unzip the file and run the exe file *rpi-alpine-builder-desktop-windows.exe*

### Linux

> TODO - not yet test it

First unzip the file:

``` unzip <download-path>/rpi-alpine-builder-desktop-linux  ```

Open the terminal and run the binary from command line

``` <download-path>/rpi-alpine-builder-desktop-linux  ```