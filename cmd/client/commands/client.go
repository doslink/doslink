package commands

import (
	"fmt"
	"os"
	"regexp"

	"github.com/spf13/cobra"

	"github.com/doslink/doslink/util"
)

// client usage template
var usageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:
    {{range .Commands}}{{if (and .IsAvailableCommand (.Name | WalletDisable))}}
    {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}

  available with wallet enable:
    {{range .Commands}}{{if (and .IsAvailableCommand (.Name | WalletEnable))}}
    {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

// commandError is an error used to signal different error situations in command handling.
type commandError struct {
	s         string
	userError bool
}

func (c commandError) Error() string {
	return c.s
}

func (c commandError) isUserError() bool {
	return c.userError
}

func newUserError(a ...interface{}) commandError {
	return commandError{s: fmt.Sprintln(a...), userError: true}
}

func newSystemError(a ...interface{}) commandError {
	return commandError{s: fmt.Sprintln(a...), userError: false}
}

func newSystemErrorF(format string, a ...interface{}) commandError {
	return commandError{s: fmt.Sprintf(format, a...), userError: false}
}

// Catch some of the obvious user errors from Cobra.
// We don't want to show the usage message for every error.
// The below may be to generic. Time will show.
var userErrorRegexp = regexp.MustCompile("argument|flag|shorthand")

func isUserError(err error) bool {
	if cErr, ok := err.(commandError); ok && cErr.isUserError() {
		return true
	}

	return userErrorRegexp.MatchString(err.Error())
}

// ClientCmd is Client's root command.
// Every other command attached to ClientCmd is a child command to it.
var ClientCmd = &cobra.Command{
	Use:   "client",
	Short: "Client is a commond line client for chain core (a.k.a. server)",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.SetUsageTemplate(usageTemplate)
			cmd.Usage()
		}
	},
}

// Execute adds all child commands to the root command ClientCmd and sets flags appropriately.
func Execute() {

	AddCommands()
	AddTemplateFunc()

	if _, err := ClientCmd.ExecuteC(); err != nil {
		os.Exit(util.ErrLocalExe)
	}
}

// AddCommands adds child commands to the root command ClientCmd.
func AddCommands() {
	ClientCmd.AddCommand(createAccessTokenCmd)
	ClientCmd.AddCommand(listAccessTokenCmd)
	ClientCmd.AddCommand(deleteAccessTokenCmd)
	ClientCmd.AddCommand(checkAccessTokenCmd)

	ClientCmd.AddCommand(createAccountCmd)
	ClientCmd.AddCommand(deleteAccountCmd)
	ClientCmd.AddCommand(listAccountsCmd)
	ClientCmd.AddCommand(createAccountReceiverCmd)
	ClientCmd.AddCommand(listAddressesCmd)
	ClientCmd.AddCommand(validateAddressCmd)
	ClientCmd.AddCommand(listPubKeysCmd)

	ClientCmd.AddCommand(createAssetCmd)
	ClientCmd.AddCommand(getAssetCmd)
	ClientCmd.AddCommand(listAssetsCmd)
	ClientCmd.AddCommand(updateAssetAliasCmd)

	ClientCmd.AddCommand(getTransactionCmd)
	ClientCmd.AddCommand(listTransactionsCmd)

	ClientCmd.AddCommand(getUnconfirmedTransactionCmd)
	ClientCmd.AddCommand(listUnconfirmedTransactionsCmd)
	ClientCmd.AddCommand(decodeRawTransactionCmd)

	ClientCmd.AddCommand(listUnspentOutputsCmd)
	ClientCmd.AddCommand(listBalancesCmd)

	ClientCmd.AddCommand(rescanWalletCmd)
	ClientCmd.AddCommand(walletInfoCmd)

	ClientCmd.AddCommand(buildTransactionCmd)
	ClientCmd.AddCommand(signTransactionCmd)
	ClientCmd.AddCommand(submitTransactionCmd)
	ClientCmd.AddCommand(estimateTransactionGasCmd)

	ClientCmd.AddCommand(getBlockCountCmd)
	ClientCmd.AddCommand(getBlockHashCmd)
	ClientCmd.AddCommand(getBlockCmd)
	ClientCmd.AddCommand(getBlockHeaderCmd)
	ClientCmd.AddCommand(getDifficultyCmd)
	ClientCmd.AddCommand(getHashRateCmd)

	ClientCmd.AddCommand(createKeyCmd)
	ClientCmd.AddCommand(deleteKeyCmd)
	ClientCmd.AddCommand(listKeysCmd)
	ClientCmd.AddCommand(resetKeyPwdCmd)
	ClientCmd.AddCommand(checkKeyPwdCmd)

	ClientCmd.AddCommand(signMsgCmd)
	ClientCmd.AddCommand(verifyMsgCmd)
	ClientCmd.AddCommand(decodeProgCmd)

	ClientCmd.AddCommand(createTransactionFeedCmd)
	ClientCmd.AddCommand(listTransactionFeedsCmd)
	ClientCmd.AddCommand(deleteTransactionFeedCmd)
	ClientCmd.AddCommand(getTransactionFeedCmd)
	ClientCmd.AddCommand(updateTransactionFeedCmd)

	ClientCmd.AddCommand(isMiningCmd)
	ClientCmd.AddCommand(setMiningCmd)

	ClientCmd.AddCommand(netInfoCmd)
	ClientCmd.AddCommand(gasRateCmd)

	ClientCmd.AddCommand(versionCmd)
}

// AddTemplateFunc adds usage template to the root command ClientCmd.
func AddTemplateFunc() {
	walletEnableCmd := []string{
		createAccountCmd.Name(),
		listAccountsCmd.Name(),
		deleteAccountCmd.Name(),
		createAccountReceiverCmd.Name(),
		listAddressesCmd.Name(),
		validateAddressCmd.Name(),
		listPubKeysCmd.Name(),

		createAssetCmd.Name(),
		getAssetCmd.Name(),
		listAssetsCmd.Name(),
		updateAssetAliasCmd.Name(),

		createKeyCmd.Name(),
		deleteKeyCmd.Name(),
		listKeysCmd.Name(),
		resetKeyPwdCmd.Name(),
		checkKeyPwdCmd.Name(),
		signMsgCmd.Name(),

		buildTransactionCmd.Name(),
		signTransactionCmd.Name(),

		getTransactionCmd.Name(),
		listTransactionsCmd.Name(),
		listUnspentOutputsCmd.Name(),
		listBalancesCmd.Name(),

		rescanWalletCmd.Name(),
		walletInfoCmd.Name(),
	}

	cobra.AddTemplateFunc("WalletEnable", func(cmdName string) bool {
		for _, name := range walletEnableCmd {
			if name == cmdName {
				return true
			}
		}
		return false
	})

	cobra.AddTemplateFunc("WalletDisable", func(cmdName string) bool {
		for _, name := range walletEnableCmd {
			if name == cmdName {
				return false
			}
		}
		return true
	})
}
