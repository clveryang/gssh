<div align="center">

<img src="images/logo.svg" alt="gssh" width="700">

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

Tab completes hosts: `gssh v<TAB>` → `gssh vast.ai.5060`, `gssh myjx<TAB>` → `gssh 美亚镜像`.
A host named like a command (`ls`, `add`, `sync`…) is reached with `gssh -- ls`.

## Config

`~/.config/gssh/hosts.yaml` — `gssh write` opens it with a commented example.

```yaml
defaults:
  identity_file: ~/.ssh/id_rsa

groups:
  - name: lab
    tags: [gpu]              # inherited by every host in the group
    hosts:
      - name: 杭州备份机
        host: 10.0.2.104
        user: deploy
        note: nightly snapshots
        alias: [hz]
```

## How it works

`gssh` renders your YAML into `~/.ssh/config.d/gssh.conf` and adds one `Include`
line to `~/.ssh/config`, so `ssh`, `scp`, `rsync`, `git` and VS Code Remote-SSH
all see the same hosts. Connecting is a plain `exec ssh`. Your `ssh_config` is
backed up before the first change and otherwise left as it was.

## License

[MIT](LICENSE)
