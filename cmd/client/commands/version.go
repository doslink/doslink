package commands

import (
	"runtime"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/doslink/doslink/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Client",
	Run: func(cmd *cobra.Command, args []string) {
		jww.FEEDBACK.Printf("Client v%s %s/%s\n", version.Version, runtime.GOOS, runtime.GOARCH)
	},
}
