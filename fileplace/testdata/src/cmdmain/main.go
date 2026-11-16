// Command cmdmain declares subcommands three ways.
package main

import "os"

// Command stands in for a command library's type.
type Command struct {
	Name string
	Use  string
}

var commands = []*Command{{Name: "ask"}, {Use: "export [file]"}}

func main() {
	switch os.Args[1] {
	case "serve":
	}
	switch os.Getenv("MODE") {
	case "debug":
	}
	_ = &Command{Name: "prior"}
	runLimit()
}

func runLimit() {}
