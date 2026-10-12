// Command vern manages canon, apparatus, and working spaces around a Git repository.
package main

import (
	"os"

	"github.com/grosspoetrysystems/vernacular/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
