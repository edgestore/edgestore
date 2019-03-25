// Package version contains version information for this app.
package version

import (
	"fmt"
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

// Version is set by the build scripts.
var (
	BuildTime  = time.Now().In(time.UTC).Format(time.Stamp + " 2006 UTC")
	CommitHash = ""
	Version    = "development"
)

func NewCommand(description string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version and exit",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(description)
			fmt.Printf("Go Version: %s\n", runtime.Version())
			fmt.Printf("Go OS/ARCH: %s %s\n", runtime.GOOS, runtime.GOARCH)
			fmt.Printf("Build Time: %s\n", BuildTime)
			fmt.Printf("Commit: %s\n", CommitHash)
			fmt.Printf("Version: %s\n", Version)
		},
	}
}
