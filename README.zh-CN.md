<div align="center">

<img src="images/logo.svg" alt="gssh" width="700">

<h1>gssh</h1>

**给 ssh 主机加一个能搜的选择器 —— 但连接这件事仍然交给 `ssh` 本身。**

[![release](https://img.shields.io/github/v/release/clveryang/gssh?color=2ea043)](https://github.com/clveryang/gssh/releases)
[![ci](https://github.com/clveryang/gssh/actions/workflows/ci.yml/badge.svg)](https://github.com/clveryang/gssh/actions/workflows/ci.yml)
[![license](https://img.shields.io/github/license/clveryang/gssh?color=blue)](LICENSE)
[![go](https://img.shields.io/github/go-mod/go-version/clveryang/gssh)](go.mod)
[![platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux-lightgrey)](https://github.com/clveryang/gssh/releases)
[![stars](https://img.shields.io/github/stars/clveryang/gssh?style=social)](https://github.com/clveryang/gssh/stargazers)

[English](README.md) | 中文

</div>

***

## 为什么要有它

`ssh` 本来就能存很多台主机，也本来就能 Tab 补全。**"存不下"从来不是问题。**

问题出现在你有三十台机器、名字叫 `13`、`62`、`board3`、`杭州备份机` 的时候：
**你找不到想连的那台，中文名你还打不出来。**

gssh 只在原生 ssh 之上加两样东西，其余一概不动：

|   | |
|---|---|
| **元数据** | 分组、标签、备注 —— 这些 `ssh_config` 根本没地方放。 |
| **对中文名有效的搜索** | `hzbfj` 找到 `杭州备份机`，`gzsc2` 找到 `广州生产2`，`bjgpu` 找到 `北京GPU`。别的 ssh 管理工具都没做这件事，这也是这个项目存在的理由。 |

gssh 自己不实现 SSH 协议。它生成一个 `ssh_config` 片段，然后 `exec` 真正的
`ssh`，所以 agent 转发、`ControlMaster`、跳板机、信号处理的行为和你直接敲 `ssh`
完全一致。

***

## 安装

```sh
curl -fsSL https://raw.githubusercontent.com/clveryang/gssh/main/install.sh | sh
```

一行装好，不需要 Go 工具链 —— 你自己的电脑和你 ssh 过去的服务器都适用。装到
`~/.local/bin`，不需要 sudo。下载会对照 release 里的 `checksums.txt` 校验；
即使你把下载源指向镜像，**校验和也始终从 GitHub 取**。

<details>
<summary>其他安装方式和可调开关</summary>

```sh
go install github.com/clveryang/gssh@latest             # 有 Go 的话
git clone https://github.com/clveryang/gssh && cd gssh && make install
```

| 变量 | 作用 |
|---|---|
| `GSSH_INSTALL_DIR` | 安装目录（默认 `~/.local/bin`） |
| `GSSH_VERSION` | 锁定版本，例如 `v0.3.0` |
| `GSSH_BASE_URL` | 只把下载搬到镜像，校验和仍来自 GitHub |
| `GSSH_SKIP_CHECKSUM=1` | 明确放弃校验 |

提供 darwin 与 linux 的 amd64 / arm64 预编译版本，也可以直接从
[releases 页面](https://github.com/clveryang/gssh/releases)下载。
</details>

***

## 上手

```sh
gssh import              # 预览：现有 ~/.ssh/config 转成 YAML 会是什么样
gssh import --write      # 确认后写入（会先备份 ssh_config）
gssh doctor              # 检查重名、拼写、失效密钥、记不住的名字
gssh sync                # 生成片段并接上 Include
```

`import` 不加 `--write` 不会写任何东西；它不会跟进 `Include` 指令（那些文件属于
别的工具）；在你执行 `sync` 之前，你的 `ssh_config` 不会被改动。

***

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

因为产物就是标准 `ssh_config`，所有其他工具都能白嫖 —— `scp`、`rsync`、`git`
以及 VS Code Remote-SSH 全都包括在内。

***

## 用法

```sh
gssh                     # 交互式选择器
gssh shanghai            # 连接
gssh shanghai uptime     # 执行一条命令
gssh shanghai -- -L 8080:localhost:8080   # -- 之后的参数原样交给 ssh

gssh ls -t prod          # 按标签过滤列表
gssh add                 # 交互式新增主机
gssh edit                # 用 $EDITOR 打开，存盘即校验并 sync
gssh doctor              # 检查配置文件里的问题
gssh completion zsh      # 补全脚本，备注会一并显示
```

### 选择器

不带参数敲 `gssh` 就打开。输入即过滤（支持拼音），列表默认按**最近使用**排序 ——
三十台机器里你真正常连的其实就五台。这个排序是 shell 补全给不了的。

```
> bo
› board1  deploy@10.0.1.13  [lab]  上海机房 A100
  board2  deploy@10.0.1.14  [lab]
  1/2  ↑↓ 移动  ⏎ 连接  ^e 编辑  esc 退出
```

***

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
        raw:                       # gssh 没有专门字段的 ssh_config 关键字写这儿
          Compression: "yes"
```

配置文件位于 `~/.config/gssh/hosts.yaml`（遵循 `$XDG_CONFIG_HOME`，`GSSH_CONFIG`
可覆盖）。`gssh doctor` 会打印它实际用的路径。

分组上的 tags 会被组内主机继承。`defaults` 用来填充主机没有设置的字段；端口转发
这类列表是追加而不是覆盖。

`gssh add` 和 `gssh edit` 会保留你写在文件里的注释和缩进。（空行不会被保留，
这是 `yaml.v3` 的限制。）

***

## 关于"不弄坏你的 ssh_config"

`~/.ssh/config` 是你怎么连上那些机器的唯一记录，所以：

- `import` 在加 `--write` 之前只预览，写之前先备份
- `edit` 编辑的是临时副本，**只有 YAML 解析通过才写回**，一个手滑的缩进不会让你丢掉配置
- `add` 走 YAML 节点树插入而不是整体重新序列化，所以你的注释不会被抹掉
- 生成的是独立片段文件；你的 `ssh_config` 只多一行 `Include`，其余内容和顺序不变

已在一份真实的 29 台主机配置上验证：用 `ssh -G`（ssh 自己的配置求值器）逐台比对，
迁移前后 **29 台全部语义完全一致**。

***

## 进度

| 里程碑 | |
|---|---|
| 配置模型、`import` / `sync` / `ls` / `doctor`、拼音补全 | ✅ |
| 交互式选择器，支持拼音搜索与最近使用排序 | ✅ |
| `add` / `edit`，均保留注释 | ✅ |
| 一行安装脚本，四平台自动发布 | ✅ |

## 许可

[MIT](LICENSE)
