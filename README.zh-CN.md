<div align="center">

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="images/logo.png">
  <img src="images/logo-light.png" alt="gssh" width="200">
</picture>

<h1>gssh</h1>

**按名字、备注、IP 或拼音，找到并连上你的 ssh 主机。**

[![release](https://img.shields.io/github/v/release/clveryang/gssh?color=2ea043)](https://github.com/clveryang/gssh/releases)
[![ci](https://github.com/clveryang/gssh/actions/workflows/ci.yml/badge.svg)](https://github.com/clveryang/gssh/actions/workflows/ci.yml)
[![license](https://img.shields.io/github/license/clveryang/gssh?color=blue)](LICENSE)
[![platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux-lightgrey)](https://github.com/clveryang/gssh/releases)

[English](README.md) | 中文

</div>

***

三十台叫 `13`、`62`、`杭州备份机` 的机器，存起来容易，找起来难。gssh 在你的
`~/.ssh/config` 之上加了备注、标签和一个可搜索的选择器 —— 输入 `hzbfj` 就能找到
`杭州备份机` —— 连接本身仍交给 `ssh`。

## 安装

```sh
curl -fsSL https://raw.githubusercontent.com/clveryang/gssh/main/install.sh | sh
```

就这一行。装完开个新终端，敲 `gssh` 即可，Tab 补全也一并配好了。

## 用法

```sh
gssh                     # 选择器；首次运行会问要不要导入 ~/.ssh/config
gssh shanghai            # 连接
gssh shanghai -- -L 8080:localhost:8080   # 额外参数交给 ssh
gssh write               # 编辑主机（保存时校验并生效）
gssh list                # 列出全部
gssh doctor              # 重名、失效密钥、没写备注的主机
```

Tab 补全主机名：`gssh we<TAB>` → `gssh web-01`，`gssh hzbfj<TAB>` → `gssh 杭州备份机`。
主机名和命令重名时（`ls`、`add`、`sync`…），用 `gssh -- ls` 连接。

## 配置

`~/.config/gssh/hosts.yaml` —— `gssh write` 会打开一份带注释的示例。
每台主机上方的注释是它所替代的 `ssh` 命令。

```yaml
defaults:                          # 所有主机的默认值，主机自己写了就以主机为准
  identity_file: ~/.ssh/id_rsa
  server_alive_interval: 30        # 空闲时保持连接不断

hosts:
  # ssh 0.0.0.0
  - name: dev
    host: 0.0.0.0

  # ssh -i ~/.ssh/vps_ed25519 ubuntu@0.0.0.0
  - name: vps
    host: 0.0.0.0
    user: ubuntu
    identity_file: ~/.ssh/vps_ed25519
    note: 云服务器

  # ssh -p 2222 root@gpu.example.com
  - name: gpu
    host: gpu.example.com
    port: 2222
    user: root

  # ssh -L 8888:localhost:8888 root@0.0.0.0    （本地 localhost:8888 打开 jupyter）
  - name: notebook
    host: 0.0.0.0
    user: root
    local_forward:
      - 8888 localhost:8888

  # ssh -J jump deploy@0.0.0.0               （只能经跳板机访问的内网机器）
  - name: jump
    host: bastion.example.com
    user: ops
  - name: inner
    host: 0.0.0.0
    user: deploy
    proxy_jump: jump

groups:                            # 组的 tags 会被组内主机继承
  - name: 板卡
    tags: [gpu]
    hosts:
      - name: 杭州备份机            # 中文名没问题，输入 hzbfj 就能找到
        host: 0.0.0.0
        user: deploy
        alias: [hz]                # gssh hz 也能连
        note: 每日快照
```

其他 `ssh_config` 关键字写在 `raw:` 下，例如 `raw: {Compression: "yes"}`。

## 原理

连接时会先显示动画并在后台建好连接，再进入会话。这条连接会保留一分钟，所以紧接着
再连同一台机器是瞬间完成的。设 `GSSH_NO_MULTIPLEX=1` 可以关掉这个行为。

gssh 把 YAML 渲染成 `~/.ssh/config.d/gssh.conf`，并在 `~/.ssh/config` 里加一行
`Include`，所以 `ssh`、`scp`、`rsync`、`git`、VS Code Remote-SSH 看到的是同一份主机。
连接就是直接 `exec ssh`。第一次改动前会备份你的 `ssh_config`，其余内容原样保留。

## 许可

[MIT](LICENSE)
