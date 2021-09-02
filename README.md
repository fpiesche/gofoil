# Containerised sftp server

![build](https://github.com/fpiesche/gofoil/actions/workflows/main.yml/badge.svg)

Gofoil is a fork of [`Orygin/gofoil`](https://github.cm/Orygin/gofoil) with added functionality to
make it function for multiple target clients and both Tinfoil and FBI installers.

This means Gofoil can run on e.g. a NAS device (via Docker support) and keep polling, making your
files ready to install on any of your devices once they run the relevant installer tool.

# Quick reference

-   **Image Repositories**:
    - Docker Hub: [`florianpiesche/gofoil`](https://hub.docker.com/r/florianpiesche/gofoil)
    - GitHub Packages: [`ghcr.io/fpiesche/gofoil`](https://ghcr.io/fpiesche/gofoil)  

-   **Maintained by**:  
	[Florian Piesche](https://github.com/fpiesche)

-	**Where to file issues**:  
    [https://github.com/fpiesche/gofoil/issues](https://github.com/fpiesche/gofoil/issues)

-   **Dockerfile**:
    [https://github.com/fpiesche/gofoil/blob/main/Dockerfile](https://github.com/fpiesche/gofoil/blob/main/Dockerfile)

-	**Supported architectures**:
    Each image is a multi-arch manifest for the following architectures:
    `amd64`, `arm64`, `armv7`, `armv6`

-	**Source of this description**: [Github README](https://github.com/fpiesche/gofoil/tree/main/README.md) ([history](https://github.com/fpiesche/gofoil/commits/main/README.md))

# How to use this image

  * Set the `GOFOIL_EXTERNALADDRESS` environment variable to an IP address or hostname (and port, if necessary; by default the http server runs on port 8000) that clients will be able to reach externally.
  * Set the `GOFOIL_CLIENTS` environment variable to a comma-separated list of IP addresses or host names of the devices that will be running the installers. You may want to assign a fixed host name or IP address to each device in your Wifi router's settings.
  * Mount directories of your files into the `/files/` directory in the container.

```console
$ docker run \
  --rm -it \
  -e GOFOIL_EXTERNALADDRESS=192.168.0.42:8000 \
  -e GOFOIL_CLIENTS=192.168.0.43,switch,another-switch
  -p 8000:8000 \
  -v /path/to/files:/files/source_1/ \
  -v /path/to/some/other/files:/files/source_2/ \
  florianpiesche/gofoil
```

# Building

## Docker (recommended)
Simply run `docker build . -t gofoil` in the repository directory. This will spin up a `golang` Docker container,
build gofoil in it and copy it to a new image you can then run as e.g. `docker run gofoil`.

## Local/native

Use the regular `go build` command to build a `gofoil` executable for your current system:
```shell script
go build github.com/fpiesche/gofoil
```

You can build for another OS/Arch target using the env vars `GOOS` and `GOARCH`:

```shell script
GOOS="linux" GOARCH="arm" GOARM=5 go build github.com/fpiesche/gofoil
```
