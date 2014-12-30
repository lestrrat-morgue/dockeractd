package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lestrrat/dockeractd"
)

const version = "0.0.1"

func main() {
	os.Exit(_main())
}

func _main() int {
	var exec = flag.String("exec", "", "Command to execute upon receiving an event")
	flag.Parse()

	opts := dockeractd.MakeDefaultOptions()
	if *exec == "" {
		fmt.Fprintf(os.Stderr, "You must supply the -exec option\n")
		return 1
	}
	opts.OptCmd = *exec
	d := dockeractd.New(opts)
	if err := d.Run(); err != nil {
		return 1
	}
	return 0
}
