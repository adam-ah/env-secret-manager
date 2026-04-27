# env-secret-manager

`env-secret-manager` is a tiny Go CLI that turns one global TOML file into the right environment variables for the directory you are in.

It is for people who are tired of sprinkling secrets through repo-local files, `.envrc` setup, shell glue, and half a dozen special-case tools just to get the right vars into the right project.

## Why it exists

Most secret workflows fail in one of two ways:

1. They live inside the repo and get copied, committed, leaked, or forgotten.
2. They spread across too many tools, so "load the right env" becomes a ritual instead of a default.

`env-secret-manager` keeps the secret mapping in one place:

`~/.secretenv/config.toml`

Then your shell hook asks a simple question every time you change directories:

> "Which project am I in right now?"

If there is a match, it exports that project's variables. If you leave the project, it unsets them.

That makes it useful when you want:

- one source of truth for local secrets
- zero secret files inside project trees
- automatic env switching on `cd`
- a config format you can read at a glance

## Why this is better than direnv, mise, and pass

This project is not trying to be a full platform. It is trying to be the fastest, least annoying way to inject the right env vars into the right shell.

### Better than direnv for this job

`direnv` is excellent when you want per-repo environment logic and are happy to manage `.envrc` files in every project.

`env-secret-manager` is better when you want:

- no repo-local secret files
- no extra setup per project
- no trust prompt choreography just to get env vars
- one central config for many repositories

If the goal is "load secrets when I enter this folder", this tool is simpler.

### Better than mise for this job

`mise` is a broader tool manager. It does runtimes, tasks, tools, and environment management.

That breadth is useful, but it is also overhead if all you need is:

- match the current directory
- export a few variables
- clean them up when you leave

`env-secret-manager` stays narrow on purpose. Fewer concepts means fewer ways to get stuck.

### Better than pass for this job

`pass` is great for encrypted secret storage and shared credential workflows.

This project is different:

- no GPG tree to manage
- no decryption step in your prompt loop
- no secret retrieval ceremony for every shell change
- no dependency on a password-store workflow if all you need is local env injection

If you need encrypted secret distribution, `pass` is the right tool.
If you need "put the right env in my shell right now", `env-secret-manager` is the lighter hammer.

## Example config

Create `~/.secretenv/config.toml`:

```toml
[projects."screeners".env]
APP_ENV = "local"
SMTP_HOST = "localhost"
SMTP_PORT = "1025"
SMTP_USERNAME = "dev_user"
SMTP_PASSWORD = "dev_password"
CLOUDFLARE_TOKEN = "cf_dev_token"

[projects."~/code/project-a".env]
APP_ENV = "local"
API_USERNAME = "project_a_user"
API_PASSWORD = "project_a_password"

[projects."/mnt/e/temp/project-b".env]
APP_ENV = "local"
DATABASE_URL = "postgres://user:pass@localhost:5432/project_b?sslmode=disable"
```

Matching rules:

- `screeners` matches any folder named `screeners`, case-insensitive
- `~/code/project-a` matches that exact path and its children
- `/mnt/e/temp/project-b` matches that exact path and its children

## Shell hooks

### Zsh

Add this to `~/.zshrc`:

```zsh
_secretenv_hook() {
  local out
  local exit_code

  command -v secretenv >/dev/null 2>&1 || return

  out="$(secretenv export 2>&1)"
  exit_code=$?

  if [[ $exit_code -ne 0 ]]; then
    echo "[secretenv] error:" >&2
    echo "$out" >&2
    return
  fi

  eval "$out"
}

autoload -Uz add-zsh-hook
add-zsh-hook chpwd _secretenv_hook

_secretenv_hook
```

### Bash

Add this to `~/.bashrc`:

```bash
_secretenv_hook() {
  command -v secretenv >/dev/null 2>&1 || return
  eval "$(secretenv export)"
}

PROMPT_COMMAND="_secretenv_hook${PROMPT_COMMAND:+;$PROMPT_COMMAND}"
```

## Install

### From a release

```bash
go install github.com/adam-ah/env-secret-manager@v0.1.0
```

### From source

```bash
git clone git@github.com:adam-ah/env-secret-manager.git
cd env-secret-manager
./build.sh
sudo mv bin/secretenv /usr/local/bin/
```

Then create your config:

```bash
mkdir -p ~/.secretenv
$EDITOR ~/.secretenv/config.toml
```

## Performance

This is designed to be boringly fast enough to run on every prompt change.

On this machine, 5,000 runs of `secretenv export` averaged:

- `8.288 ms` when no config file was present
- `8.780 ms` with a representative config and a matching project directory

That is fast enough to keep the shell hook invisible in normal use.

## Security notes

This keeps secrets out of repo-local files, which helps avoid accidental commits and copy-paste leakage.

It does not encrypt `~/.secretenv/config.toml`. Treat it like a secret:

```bash
chmod 700 ~/.secretenv
chmod 600 ~/.secretenv/config.toml
```

## Commands

```bash
secretenv help
secretenv export
```
