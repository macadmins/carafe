# Carafe

Carafe is a (fancy) wrapper for [Homebrew](https://brew.sh). It is designed to be run as root; it drops privileges to the currently logged-in user so it can be safely executed by management tools such as Munki or Jamf.

## Example use cases

- Bootstrapping engineering machines with a set of Homebrew formulae.
- Enforcing minimum formulae versions to address security vulnerabilities, only for formulae that are already installed.
- For a full example of using Carafe with Munki, see the [examples](https://github.com/macadmins/carafe/tree/main/examples) directory.

In addition to the basic Homebrew functionality, Carafe provides a few additional features:

## Minimum version enforcement

You can specify a minimum version of a formula that should be installed. If the installed version is lower than the specified minimum version, Carafe will automatically upgrade it to the latest version.

```bash
/opt/macadmins/bin/carafe update <formula> --min-version=<version>
```

## Check

Carafe can check whether a formula is installed and whether it meets a minimum version. Use `--skip-not-installed` to ignore formulae that are not installed; this is useful when you only want to enforce minimum versions for installed formulae.

```bash
/opt/macadmins/bin/carafe check <formula> [--min-version=<version>] [--skip-not-installed]
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

## Occasionally asked questions

- **Does Carafe install Homebrew if it is not already installed?**: No, Carafe assumes that Homebrew is already installed on the system. We recommend using the [official package from Github](https://github.com/Homebrew/brew/releases).
- **Does Carafe prevent the use of Homebrew outside of Carafe?**: No, Carafe does not restrict the use of Homebrew. If you need to prevent users from using Homebrew directly, or prevent the installation of unauthorized formulae, consider using tools like [Santa](https://github.com/northpolesec/santa).
- **Will Carafe work in a shared deployment, such as an instructional lab?**: Carafe has not been tested in shared deployments, and it is possible there will be issues in those scenarios.
