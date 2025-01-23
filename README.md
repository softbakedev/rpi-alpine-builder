<p align="center">
    <img src="docs/assets/logo.png" alt="drawing" width="200"/>
</p>

# Raspberry Pi Alpine image builder

The repository contains code of command line tool for building alpine image
for Raspberry PI devices. It should provide easier way of deployment.

# Getting started

## Using CLI
First build cmd tool by:

``` go build -o  ./build/rpialp .  ```

Run command (command must be run as sudo):

``` sudo ./build/rpialp build  ```

Help command:

``` sudo ./build/rpialp --help  ```

## Using desktop app

Visit tho docs [here](desktop/README.md)
