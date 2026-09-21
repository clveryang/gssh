# gssh

[English](README.md) | 中文

把 ssh 主机集中放在一个 YAML 文件里，提供可搜索的选择器，同时让 `ssh`、`scp`、
`rsync`、`git` 和 VS Code Remote-SSH 继续照常使用这些配置。

## 为什么要有它

`ssh` 本来就能保存多台主机，也本来就能 Tab 补全。它缺的不是"存"，而是当你有
三十台机器、名字叫 `13`、`62`、`board3`、`杭州备份机` 时，**你没法从里面找出想
连的那台**。

gssh 只在原生 ssh 之上加两样东西，其余一概不动：

- **元数据** —— 分组、标签、备注，这些 `ssh_config` 没有地方放。
- **对中文名有效的搜索** —— `hzbfj` 能找到 `杭州备份机`，`gzsc2` 能找到
  `广州生产2`，`bjgpu` 能找到 `北京GPU`。别的 ssh 管理工具都没做这件事，这也是
  这个项目存在的理由。

gssh 自己不实现 SSH 协议。它生成一个 `ssh_config` 片段，然后 `exec` 真正的
`ssh`，所以 agent 转发、`ControlMaster`、跳板机、信号处理的行为和你直接敲 `ssh`
完全一致。

## 它是怎么接进去的

```
~/.config/gssh/hosts.yaml        你编辑这个
        │  gssh sync
        ▼
~/.ssh/config.d/gssh.conf        自动生成，不要手改
        │  Include
        ▼
~/.ssh/config                    只加一行 Include，其余不动
```

因为产物就是标准 `ssh_config`，所有其他工具都能白嫖——包括 VS Code Remote-SSH。

## 安装

一行装好，不需要 Go 工具链。你自己的电脑和你 ssh 过去的服务器都适用：

```sh
curl -fsSL https://raw.githubusercontent.com/clveryang/gssh/main/install.sh | sh
```

装到 `~/.local/bin`，不需要 sudo。可用 `GSSH_INSTALL_DIR=/usr/local/bin` 换目录，
用 `GSSH_VERSION=v0.1.0` 锁版本。提供 darwin/linux 的 amd64/arm64 预编译版本，
下载后会对照发布的 `checksums.txt` 校验。

<details>
<summary>其他安装方式</summary>

```sh
go install github.com/clveryang/gssh@latest   # 有 Go 的话
git clone https://github.com/clveryang/gssh && cd gssh && make install
```

也可以直接从 [releases 页面](https://github.com/clveryang/gssh/releases)下载二进制。
</details>

## 上手

```sh
gssh import              # 预览：把现有 ~/.ssh/config 转成 YAML 会是什么样
gssh import --write      # 确认后写入（会先备份 ssh_config）
gssh doctor              # 检查重名、拼写、失效的密钥、记不住的名字
gssh sync                # 生成片段并接上 Include
```

`import` 不加 `--write` 不会写任何东西；它不会跟进 `Include` 指令（那些文件属于
别的工具）；在你执行 `sync` 之前，你的 `ssh_config` 不会被改动。

## 用法

```sh
gssh                     # 选择器（M2）
gssh shanghai            # 连接
gssh shanghai uptime     # 执行一条命令
gssh shanghai -- -L 8080:localhost:8080   # -- 之后的参数原样交给 ssh
gssh ls -t prod          # 按标签过滤列表
gssh add                 # 交互式新增主机
gssh add sh --host 10.0.1.13 --user deploy --note "上海机房"   # 也可用参数
gssh edit                # 用 $EDITOR 打开，保存后自动校验并 sync
gssh completion zsh      # 补全脚本，备注会一并显示
```

## 配置

```yaml
defaults:
  identity_file: ~/.ssh/id_rsa
  server_alive_interval: 30

groups:
  - name: 板卡
    tags: [lab]
    hosts:
      - name: board1
        alias: [b1]
        host: 10.0.1.13
        user: deploy
        note: 上海机房 A100
      - name: 杭州备份机
        host: 10.0.2.104
        raw:                       # gssh 没有专门字段的 ssh_config 关键字都能写这儿
          Compression: "yes"
```

分组上的 tags 会被组内主机继承。`defaults` 用来填充主机没有设置的字段；端口转发
这类列表是追加而不是覆盖。

`gssh add` 和 `gssh edit` 会保留你写在文件里的注释和缩进。（受 YAML 库限制，
空行不会被保留。）

## 状态

- **M1 — 已完成。** YAML 模型、渲染器、`import`、`sync`、`ls`、`doctor`、带拼音
  匹配的补全。已在一份真实的 29 台主机配置上验证：`ssh -G` 判定迁移前后全部 29
  台语义完全一致。
- **M2** — Bubble Tea 选择器。
- **M3 — 已完成。** `add` / `edit`。
- **M4 — 已完成。** GoReleaser + 一行安装脚本。
