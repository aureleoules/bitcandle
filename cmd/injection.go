package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thediveo/enumflag"
)

var (
	filePath        string
	injectionMethod InjectionMethod
)

type InjectionMethod enumflag.Flag

// ② Define the enumeration values for FooMode.
const (
	Auto InjectionMethod = iota
	P2FKH
	P2FK
	P2SH
	OP_RETURN
)

var InjectionMethodIds = map[InjectionMethod][]string{
	Auto:      {"auto"},
	P2FKH:     {"p2fkh"},
	P2FK:      {"p2fk"},
	P2SH:      {"p2sh"},
	OP_RETURN: {"op_return"},
}

func init() {
	injectCmd.PersistentFlags().VarP(
		enumflag.New(&injectionMethod, "method", InjectionMethodIds, enumflag.EnumCaseInsensitive), "method", "m", "injection method; can be 'auto', 'p2fkh', 'p2fk', 'p2sh' or 'op_return'")

	injectCmd.Flags().StringVarP(&filePath, "file", "f", "", "path of the file to inject on Bitcoin")

	rootCmd.AddCommand(injectCmd)
}

var injectCmd = &cobra.Command{
	Use:   "inject",
	Short: "Inject a file on the Bitcoin network",
	Run: func(cmd *cobra.Command, args []string) {
		if filePath == "" {
			errInjectHelp("missing file path")
		}
	},
}

func errInjectHelp(err string) {
	fmt.Println("error: " + err)
	fmt.Println(`Please see "bitcandle inject --help" for more information.`)
	os.Exit(1)
}
