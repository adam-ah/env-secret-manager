---
name: secretenv
description: Use locally stored secretenv credentials in commands and API requests without exposing secret values in agent output.
---

# Use secrets with secretenv

Discover names and exact references before executing a command:

```sh
secretenv list --current
```

This returns Global plus the project matching the working directory. Use `secretenv list` to find other projects. Output contains names and references, never values. Copy exact entries from `references`; ask the user if the intended credential is ambiguous or missing. Example names below are placeholders, not credentials to assume exist.

`KEY` addresses Global from anywhere; `PROJECT:KEY` addresses that exact project. Both inject the child environment variable `KEY`, without the project prefix. Global never auto-exports into the parent environment.

## Inject as an env variable

Available as `$SAMPLE_SECRET` in the child process:

```sh
secretenv exec my-app:SAMPLE_SECRET -- your-program
```

Replace `your-program` with the executable that needs the credential. Add multiple references before `--`, for example `secretenv exec OPENAI_KEY my-app:SAMPLE_SECRET -- your-program`. Duplicate key names are rejected. Exported parent variables and the working directory are inherited; selected values override variables in the child only.

## Curl using the env variable

```sh
secretenv exec my-app:SAMPLE_SECRET -- sh -c 'exec curl --silent --show-error \
  --header "Authorization: Bearer $SAMPLE_SECRET" \
  https://api.example.com'
```

The single quotes defer `$SAMPLE_SECRET` expansion until the child shell receives the secret. `exec` runs programs directly; explicitly use `sh -c` for variable expansion, shell operators, or builtins. Replace the endpoint with the destination authorized by the user.

## Inject into curl

```sh
secretenv exec my-app:SAMPLE_SECRET -- curl \
  --header 'Authorization: Bearer {SAMPLE_SECRET}' \
  https://api.example.com
```

`{KEY}` inserts the selected value into an argument literally, without recursive substitution. Values are still available as child env variables. Prefer the environment when the target program supports it, since argument values can appear in process listings.

## Keep values out of agent output

Use `list` and `exec` for agent access. Never read `~/.secretenv/config.toml`, invoke the internal `secretenv _hook` command, or retrieve values through files or other channels. There is no raw `get` or public `export` command. `install [bash|zsh]` only prints folder-based shell hook setup for humans and changes no files. Avoid inherited secrets and human shell hooks in agent sessions.

Literal selected values are redacted from child stdout and stderr, including split writes. This prevents accidental disclosure; files, encodings, direct terminal access, and unselected inherited secrets can bypass it. Do not treat filtering as isolation. Empty values remain empty. Child exit codes are preserved; stop and rediscover references when selection fails.

Without a terminal, bare `secretenv`, `menu`, and `edit` show concise help. `list` and `exec` work normally for agents.
