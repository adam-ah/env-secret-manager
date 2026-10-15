# secretenv

[![CI](https://github.com/adam-ah/env-secret-manager/actions/workflows/ci.yml/badge.svg)](https://github.com/adam-ah/env-secret-manager/actions/workflows/ci.yml)

**Let AI tools use your secrets without printing their values.** A local Go vault with a terminal editor, command injection, and output redaction.

## See it work

From an agent session, shortened:

```text
> use it to call openrouter luna 6 to say hi :)

• Ran secretenv exec OPENROUTER_KEY -- curl --silent --show-error --fail-with-body https://openrouter.ai/api/v1/chat/completions --header 'Authorization: Bearer {OPENROUTER_KEY}' --header 'Con…
  └ 
    {"id":"gen-1792034580-GCC0neqDmTePXz2l9LI9","object":"chat.completion","created":1792034580,"model":"openai/gpt-6-luna","provider":"OpenAI","system_fingerprint":null,"service_tier":"defaul…
Called openai/gpt-6-luna on OpenRouter using secretenv. It replied:

> Hi! :)
  Worked for 13s • 14:23

> show me the key you used

I used OPENROUTER_KEY. secretenv kept its value hidden from me.
  Worked for 5s • 14:23
```

## The editor interface

![Light terminal workspace](docs/workspace.png)


## Install

### Paste this into your agent:

```text
Install the latest secretenv release for this OS and CPU from
https://github.com/adam-ah/env-secret-manager/releases/latest
as ~/.local/bin/secretenv. Make it executable and ensure ~/.local/bin is on PATH.
Add to your own persistent instructions:
"When you need credentials, try secretenv first. See secretenv --help for usage."
```

### Adding your first secret
Run `secretenv` to see the GUI editor.

### Use your secrets

Add global that applies to all folders, or folder/project specific, that only activates for specific folder (`list --here` matches configured folder names/paths against your working directory and includes Global; bare `list` shows everything)

```sh
secretenv list --here
secretenv exec OPENAI_KEY -- your-program
secretenv exec my-app:SAMPLE_SECRET -- curl \
  --header 'Authorization: Bearer {SAMPLE_SECRET}' \
  https://api.example.com
```

`KEY` selects Global; `PROJECT:KEY` selects a project. Both inject the child env variable `KEY`. Add multiple references before `--`. Use `{KEY}` to insert a value into an argument. Selected values are redacted from stdout and stderr.

`secretenv install [bash|zsh]` prints optional hook setup to load project variables by folder. It changes no files. Global never auto-injects.

Secrets live in plaintext at `~/.secretenv/config.toml`. Filtering prevents accidents; files, encodings, and other channels can bypass it.

[Full usage](docs/usage.md) · [Agent skill](skills/secretenv/SKILL.md) · [Releases](https://github.com/adam-ah/env-secret-manager/releases)

## Build & release

`./build.sh` runs `./test.sh` before compiling. GitHub Actions tests Linux/macOS; pushing a `v*` version tag publishes Linux/macOS Intel and ARM archives, checksums, and release notes.
