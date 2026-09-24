<div align="center">

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="images/logo.png">
  <img src="images/logo-light.png" alt="gssh" width="200">
</picture>

<h1>gssh</h1>

**Find and connect to your ssh hosts — by name, note, IP, or pinyin.**

[![release](https://img.shields.io/github/v/release/clveryang/gssh?color=2ea043)](https://github.com/clveryang/gssh/releases)
[![ci](https://github.com/clveryang/gssh/actions/workflows/ci.yml/badge.svg)](https://github.com/clveryang/gssh/actions/workflows/ci.yml)
[![license](https://img.shields.io/github/license/clveryang/gssh?color=blue)](LICENSE)
[![platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux-lightgrey)](https://github.com/clveryang/gssh/releases)

English | [中文](README.zh-CN.md)

</div>

***

Thirty hosts named `13`, `62` and `杭州备份机` are easy to store and hard to find.
gssh adds notes, tags and a searchable picker on top of your `~/.ssh/config` —
`hzbfj` finds `杭州备份机` — and leaves the connecting to `ssh` itself.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/clveryang/gssh/main/install.sh | sh
```

That's it — open a new terminal and run `gssh`. Tab completion is set up too.

## Usage

```sh
gssh                     # picker; on first run, offers to import ~/.ssh/config
gssh shanghai            # connect
gssh shanghai -- -L 8080:localhost:8080   # extra args go to ssh
gssh write               # edit your hosts (checked and applied on save)
gssh list                # list them
gssh doctor              # duplicates, dead keys, names without notes
```

Tab completes hosts: `gssh we<TAB>` → `gssh web-01`, `gssh hzbfj<TAB>` → `gssh 杭州备份机`.
A host named like a command (`ls`, `add`, `sync`…) is reached with `gssh -- ls`.

## Config

`~/.config/gssh/hosts.yaml` — `gssh write` opens it with a commented example.
Each host notes the `ssh` command it replaces.

```yaml
defaults:                          # applies to every host that does not say otherwise
  identity_file: ~/.ssh/id_rsa
  server_alive_interval: 30        # keep idle connections alive

hosts:
  # ssh 0.0.0.0
  - name: dev
    host: 0.0.0.0

  # ssh -i ~/.ssh/vps_ed25519 ubuntu@0.0.0.0
  - name: vps
    host: 0.0.0.0
    user: ubuntu
    identity_file: ~/.ssh/vps_ed25519
    note: cloud vps

  # ssh -p 2222 root@gpu.example.com
  - name: gpu
    host: gpu.example.com
    port: 2222
    user: root

  # ssh -L 8888:localhost:8888 root@0.0.0.0    (jupyter on localhost:8888)
  - name: notebook
    host: 0.0.0.0
    user: root
    local_forward:
      - 8888 localhost:8888

  # ssh -J jump deploy@0.0.0.0               (only reachable through a bastion)
  - name: jump
    host: bastion.example.com
    user: ops
  - name: inner
    host: 0.0.0.0
    user: deploy
    proxy_jump: jump

groups:                            # a group's tags are inherited by its hosts
  - name: lab
    tags: [gpu]
    hosts:
      - name: 杭州备份机            # Chinese names work; find it with hzbfj
        host: 0.0.0.0
        user: deploy
        alias: [hz]                # gssh hz works too
        note: nightly snapshots
```

Any other `ssh_config` keyword goes under `raw:`, e.g. `raw: {Compression: "yes"}`.

## How it works

Connecting shows a spinner while it opens a connection in the background, then
drops you into the session. That connection is kept for a minute, so the next
`gssh` to the same host is instant. `GSSH_NO_MULTIPLEX=1` turns this off.

`gssh` renders your YAML into `~/.ssh/config.d/gssh.conf` and adds one `Include`
line to `~/.ssh/config`, so `ssh`, `scp`, `rsync`, `git` and VS Code Remote-SSH
all see the same hosts. Connecting is a plain `exec ssh`. Your `ssh_config` is
backed up before the first change and otherwise left as it was.

## License

[MIT](LICENSE)
