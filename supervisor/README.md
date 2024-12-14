# Supervisor

## Knowledge base

- All binaries are stored in `~/bin` directory (directory is automatically created during build step)
- All environment files are stored in `~/bin` directory
- All logs are stores in `~/log` directory (directory must be manually created)

## Prerequisities

All prerequisities are supposed to be done only once when setting up the VM.

### Install gcc

```shell
sudo apt install gcc
```

### Install Go (go1.21.5)

```shell
cd ~
mkdir go
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz

sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
```

Set path to go binary

```shell
echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.bashrc
```

### Install supervisor

Ubuntu

```shell
sudo apt install supervisor
```

MacOS

```shell
brew install supervisor
```

### Create log directory

```shell
mkdir ~/log
```

## First-time deployment

1. Fetch git tag representing the version of the service you are going to deploy.

```shell
git fetch --all --tags --prune
git checkout tags/<tag_name>
```

2. Build binary of the service to be deployed (e.g. call `task local:dal-build` to create `dal` in `~/bin/`)
3. Create symlink from `supervisor/*.conf` file to `/etc/supervisor/conf.d/*.conf` (MacOS uses `/opt/homebrew/etc/supervisor.d/*.conf`)
4. Create symlink to (or copy) `.env.*` file (`.env` file is in format `.env.service_name`) to `~/bin`
5. Launch service through supervisor

```shell
supervisorctl
reread
update
```

## Update

1. Fetch git tag representing the version of the service you are going to deploy.

```shell
git fetch --all --tags --prune
git checkout tags/<tag_name>
```

2. Build binary of the service to be deployed (e.g. call `task local:dal-build` to create `dal` in `~/bin/`)
3. Update `*.conf` if necessary
4. Update `.env.*` if necessary
5. Launch service through supervisor

```shell
supervisorctl
reread
update
```
