package main

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/macadmins/carafe/brew"
	"github.com/macadmins/carafe/exec"

	"github.com/spf13/cobra"
)

var formulaRe = regexp.MustCompile(`^[A-Za-z0-9+@._-]{1,128}$`) //nolint:gochecknoglobals

func validateFormulaArg(arg string) error {
	if !formulaRe.MatchString(arg) {
		return fmt.Errorf("invalid formula name")
	}
	return nil
}

func completionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "completion",
		Short: "Generate the autocompletion script for the specified shell",
	}
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "carafe",
		Short: "A CLI tool for managing homebrew packages",
	}

	c, err := exec.NewConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var cleanupCmd = &cobra.Command{
		Use:   "cleanup [package]",
		Short: "Cleanup the desired package",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateFormulaArg(args[0]); err != nil {
				return err
			}
			return brew.Cleanup(c, args[0])
		},
	}

	var installCmd = &cobra.Command{
		Use:   "install [package]",
		Short: "Install the desired package",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateFormulaArg(args[0]); err != nil {
				return err
			}
			return brew.Install(c, args[0])

		},
	}

	var uninstallCmd = &cobra.Command{
		Use:   "uninstall [package]",
		Short: "Uninstall the desired package",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateFormulaArg(args[0]); err != nil {
				return err
			}
			return brew.Uninstall(c, args[0])
		},
	}

	var tapCmd = &cobra.Command{
		Use:   "tap [tapname]",
		Short: "Add the desired tap",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return brew.Tap(c, args[0])
		},
	}

	var untapCmd = &cobra.Command{
		Use:   "untap [tapname]",
		Short: "Remove the desired tap",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return brew.Untap(c, args[0])
		},
	}

	var infoCmd = &cobra.Command{
		Use:   "info [package]",
		Short: "List information about installed packages or a specific package",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return brew.AllInfo(c)
			} else {
				if err := validateFormulaArg(args[0]); err != nil {
					return err
				}
				return brew.Info(c, args[0])
			}
		},
	}

	var minVersion string
	var upgradeCmd = &cobra.Command{
		Use:   "upgrade [package]",
		Short: "Upgrade the package if its version is less than the specified minimum version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateFormulaArg(args[0]); err != nil {
				return err
			}
			// This would require additional logic to check the version and compare it with minVersion
			// For simplicity, we are assuming the package is updated directly
			if minVersion != "" {
				return brew.EnsureMinimumVersion(c, args[0], minVersion)
			}
			return brew.Upgrade(c, args[0])
		},
	}
	upgradeCmd.Flags().StringVar(&minVersion, "min-version", "", "Minimum version to update the package to")

	// check command
	var munkiInstallCheck bool
	var skipNotInstalled bool
	var noCache bool
	var cacheTTL time.Duration
	var checkCmd = &cobra.Command{
		Use:   "check [package]",
		Short: "Check if the package is installed, and optionally at or above a specific version. Use --min-version to specify a minimum version. Use --munki-installcheck to reverse the exit codes.", //nolint:lll
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateFormulaArg(args[0]); err != nil {
				return err
			}
			if noCache {
				cacheTTL = 0
			}
			exitCode, err := brew.Check(c, args[0], minVersion, munkiInstallCheck, skipNotInstalled, cacheTTL)
			if err != nil {
				return err
			}
			os.Exit(exitCode)
			return nil
		},
	}
	checkCmd.Flags().StringVar(&minVersion, "min-version", "", "Minimum version to check the package against")
	checkCmd.Flags().BoolVar(
		&munkiInstallCheck,
		"munki-installcheck",
		false,
		"Flag for munki installcheck which reverses the exit codes",
	)
	checkCmd.Flags().BoolVar(
		&skipNotInstalled,
		"skip-not-installed",
		false,
		"Exits with success if the package is not installed. Must be used with --min-version flag. For use when checking to upgrade for security reasons", //nolint:lll
	)
	checkCmd.Flags().BoolVar(
		&noCache,
		"no-cache",
		false,
		"Disable the brew info cache and call brew directly for each check",
	)
	checkCmd.Flags().DurationVar(
		&cacheTTL,
		"cache-ttl",
		60*time.Second,
		"How long the brew info cache is considered valid (e.g. 30s, 2m)",
	)

	var Version = "dev" // Set at build time using -ldflags
	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version of the carafe CLI tool",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(Version)
			os.Exit(0)
		},
	}

	completion := completionCommand()
	completion.Hidden = true
	rootCmd.AddCommand(completion)
	rootCmd.AddCommand(
		installCmd,
		uninstallCmd,
		cleanupCmd,
		infoCmd,
		tapCmd,
		untapCmd,
		upgradeCmd,
		checkCmd,
		versionCmd,
	)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
