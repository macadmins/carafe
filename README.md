# Carafe

Carafe is a (fancy) wrapper for [Homebrew](https://brew.sh). It is designed to be run as root; it drops privileges to the currently logged-in user so it can be safely executed by management tools such as Munki or Jamf.

## Example use cases

- Bootstrapping engineering machines with a set of Homebrew formulae.
- Enforcing minimum formulae versions to address security vulnerabilities, only for formulae that are already installed.
- For a full example of using Carafe with Munki, see the [examples](https://github.com/macadmins/carafe/tree/main/examples) directory.

In addition to the basic Homebrew functionality, Carafe provides a few additional features:

## Minimum version enforcement

You can specify a minimum version of a formula that should be installed. If the installed version is lower than the specified minimum version, Carafe will automatically upgrade it to the latest version.

Versions are compared the way Homebrew orders them for a minimum version check. A revision suffix on the installed version (`2.52.0_1`) is ignored; `--min-version` itself cannot carry a revision. A letter suffix marks a later release, so `10.2` < `10.2p1` < `10.2p2` (openssh) and `1.1.1` < `1.1.1w` (openssl@1.1). `alpha`, `beta`, `preview`, `pre`, `rc`, and `a` or `b` followed by a number (`1.0.0b1`, `3.14.0a2`) are pre-releases, ranked in that order, and their numbers compare as numbers (`rc9` < `rc10`). A leading `v` and a `-` before the suffix are ignored (`v10.2p2`, `1.0-rc1`). Homebrew's own order is not transitive in a few mixed cases (for example `1.0rc` against `1.0c1`, or `1.0b` against `1.0b3`); carafe uses the order above there. One known difference: without a number Homebrew puts `1.0pre` before `1.0preview`, while carafe ranks `preview` before `pre` in every case. Upper-case suffixes keep the go-version rules.

```bash
/opt/macadmins/bin/carafe update <formula> --min-version=<version>
```

## Check

Carafe can check whether a formula is installed and whether it meets a minimum version. Use `--skip-not-installed` to ignore formulae that are not installed; this is useful when you only want to enforce minimum versions for installed formulae.

```bash
/opt/macadmins/bin/carafe check <formula> [--min-version=<version>] [--skip-not-installed]
```

### Caching

When running many `check` commands in quick succession (e.g. from multiple Munki `installcheck_script` entries), Carafe caches the output of `brew info --json --installed` on disk for 60 seconds by default. This means only the first `check` call invokes Homebrew; all subsequent calls within the TTL window are served from the cache, significantly reducing the time for a full Munki check run.

The cache is stored at `/var/root/.carafe/brew_info_cache_arm64.json` (Apple Silicon) or `/var/root/.carafe/brew_info_cache_x86_64.json` (Intel). The directory is created with mode `0700` so only root can read or write cache files, preventing symlink and injection attacks.

To disable caching:
```bash
/opt/macadmins/bin/carafe check <formula> --no-cache
```

To use a custom cache TTL:
```bash
/opt/macadmins/bin/carafe check <formula> --cache-ttl=30s
/opt/macadmins/bin/carafe check <formula> --cache-ttl=2m
```

### Munki-specific exit codes

Munki expects an exit code of 0 to indicate that installation is required, and 1 to indicate that no action is needed when using `installcheck_script`. With `--munki-installcheck`, `carafe check` exits 0 if the formula is not installed or fails the `--min-version` check, and 1 if it is installed and meets the requirement.

```bash
/opt/macadmins/bin/carafe check <formula> [--min-version=<version>] [--skip-not-installed] --munki-installcheck
```

## Other supported brew commands

These commands support the same options as the `brew` command. The commands are:

- `cleanup`
- `info`
- `install`
- `tap`
- `uninstall`
- `untap`
- `upgrade`

### Tap-qualified formula names

Commands that take a formula accept either a bare name (`openssl@3`) or a tap-qualified name (`homebrew/core/openssl@3`). Use a tap-qualified name when more than one tap provides a formula with the same name and you need to target a specific one. Casks are not supported.

## Occasionally asked questions

- **Does Carafe install Homebrew if it is not already installed?**: No, Carafe assumes that Homebrew is already installed on the system. We recommend using the [official package from Github](https://github.com/Homebrew/brew/releases).
- **Does Carafe prevent the use of Homebrew outside of Carafe?**: No, Carafe does not restrict the use of Homebrew. If you need to prevent users from using Homebrew directly, or prevent the installation of unauthorized formulae, consider using tools like [Santa](https://github.com/northpolesec/santa).
- **Will Carafe work in a shared deployment, such as an instructional lab?**: Carafe has not been tested in shared deployments, and it is possible there will be issues in those scenarios.
