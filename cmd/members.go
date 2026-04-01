package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jtsilverman/council/internal/persona"
)

var membersCmd = &cobra.Command{
	Use:   "members",
	Short: "List available code review council members",
	RunE:  runMembers,
}

func init() {
	rootCmd.AddCommand(membersCmd)
}

func runMembers(cmd *cobra.Command, args []string) error {
	infos := persona.AllMemberInfo()

	fmt.Println("Code review council members:")
	fmt.Println()

	for _, info := range infos {
		setBadge := ""
		switch info.Set {
		case "core, light":
			setBadge = "\033[32m[core+light]\033[0m"
		case "core":
			setBadge = "\033[36m[core]\033[0m"
		case "extended":
			setBadge = "\033[33m[extended]\033[0m"
		}

		fmt.Printf("  \033[1m%-18s\033[0m %s  %s\n", info.ShortName, setBadge, info.FullName)
		fmt.Printf("    %s\n\n", info.Focus)
	}

	fmt.Println("Sets:")
	fmt.Println("  --light   security, bugs (2 members)")
	fmt.Println("  default   security, bugs, performance, maintainability (4 members)")
	fmt.Println("  --all     all 10 members")
	fmt.Println("  --with    pick specific: --with \"security,concurrency,data\"")

	return nil
}
