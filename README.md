# gssh

Keep your ssh hosts in one YAML file, get a searchable picker over them, and
still let `ssh`, `scp`, `rsync`, `git` and VS Code Remote-SSH use them.

## Why

`ssh` already stores many hosts and already tab-completes them. What it does not
give you is a way to *find* the right one when you have thirty of them named
`13`, `62`, `board3` and `杭州备份机`.

gssh adds two things on top of stock ssh, and changes nothing else:

- **Metadata** — groups, tags and notes, which `ssh_config` has no place for.
- **Search that works on Chinese names** — `hzbfj` finds `杭州备份机`, `gzsc2`
  finds `广州生产2`, `bjgpu` finds `北京GPU`. No other ssh manager does this,
  and it is the reason this one exists.

gssh never speaks the SSH protocol itself. It renders an `ssh_config` fragment
and `exec`s the real `ssh`, so agent forwarding, `ControlMaster`, jump hosts and
signal handling behave exactly as they always did.

## How it fits together

```
~/.config/gssh/hosts.yaml        you edit this
        │  gssh sync
        ▼
~/.ssh/config.d/gssh.conf        generated, never hand-edited
        │  Include
        ▼
~/.ssh/config                    one Include line added, nothing else touched
```

Because the result is a plain `ssh_config`, every other tool picks it up for
free — including VS Code Remote-SSH.

## Getting started

```sh
go build -o gssh .

gssh import              # preview the conversion of your existing ~/.ssh/config
gssh import --write      # save it (backs up ssh_config first)
gssh doctor              # duplicates, typos, missing keys, unmemorable names
gssh sync                # generate the fragment and wire up the Include
```

`import` writes nothing without `--write`, does not follow `Include` directives
(those files belong to other tools), and leaves your `ssh_config` untouched
until you run `sync`.

## Usage

```sh
gssh                     # picker (M2)
gssh shanghai            # connect
gssh shanghai uptime     # run a command
gssh shanghai -- -L 8080:localhost:8080   # pass flags through to ssh
gssh ls -t prod          # list, filtered by tag
gssh completion zsh      # completion script, with notes shown inline
```

## Config

```yaml
defaults:
  identity_file: ~/.ssh/id_rsa
  server_alive_interval: 30

groups:
  - name: 板卡
    tags: [lab]
    hosts:
      - name: mrboard1
        alias: [mb1]
        host: 10.0.1.13
        user: deploy
        note: 上海机房 A100
      - name: 杭州备份机
        host: 10.0.2.104
        raw:                       # any ssh_config keyword gssh has no field for
          Compression: "yes"
```

Tags on a group are inherited by its hosts. `defaults` fills in fields a host
leaves unset; port-forward lists are concatenated rather than replaced.

## Status

- **M1 — done.** YAML model, renderer, `import`, `sync`, `ls`, `doctor`,
  completion with pinyin matching. Verified against a real 29-host config:
  `ssh -G` reports all 29 semantically identical before and after.
- **M2** — Bubble Tea picker.
- **M3** — `add` / `edit`.
- **M4** — GoReleaser + homebrew tap.
