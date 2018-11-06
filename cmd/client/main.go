package main

import (
	"runtime"

	cmd "github.com/doslink/doslink/cmd/client/commands"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	cmd.Execute()
}
