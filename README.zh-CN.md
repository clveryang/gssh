<div align="center">

<img src="images/logo.svg" alt="gssh" width="700">

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

支持 macOS / Linux，amd64 / arm64。装到 `~/.local/bin`，不需要 sudo，下载后校验 checksum。
也可以 `go install github.com/clveryang/gssh@latest`。

## 用法

```sh
gssh                     # 选择器；首次运行会问要不要导入 ~/.ssh/config
gssh shanghai            # 连接
gssh shanghai -- -L 8080:localhost:8080   # 额外参数交给 ssh
gssh write               # 编辑主机（保存时校验并生效）
gssh list                # 列出全部
gssh doctor              # 重名、失效密钥、没写备注的主机
```

Tab 补全 —— `gssh v<TAB>` → `gssh vast.ai.5060`，`gssh myjx<TAB>` → `gssh 美亚镜像`：

```sh
echo 'source <(gssh completion zsh)' >> ~/.zshrc && exec zsh
```

主机名和命令重名时（`ls`、`add`、`sync`…），用 `gssh -- ls` 连接。

## 配置

`~/.config/gssh/hosts.yaml` —— `gssh write` 会打开一份带注释的示例。

```yaml
defaults:
  identity_file: ~/.ssh/id_rsa

groups:
  - name: 板卡
    tags: [gpu]              # 组内主机都会继承
    hosts:
      - name: 杭州备份机
        host: 10.0.2.104
        user: deploy
        note: 每日快照
        alias: [hz]
```

## 原理

gssh 把 YAML 渲染成 `~/.ssh/config.d/gssh.conf`，并在 `~/.ssh/config` 里加一行
`Include`，所以 `ssh`、`scp`、`rsync`、`git`、VS Code Remote-SSH 看到的是同一份主机。
连接就是直接 `exec ssh`。第一次改动前会备份你的 `ssh_config`，其余内容原样保留。

## 许可

[MIT](LICENSE)
