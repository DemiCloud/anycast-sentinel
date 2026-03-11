package version

import (
	"fmt"
	"runtime"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
	BuiltBy   = "unknown"
)

func Print() {
	fmt.Printf("anycast-sentinel %s\n", Version)
	fmt.Printf("commit:    %s\n", Commit)
	fmt.Printf("built:     %s\n", BuildDate)
	fmt.Printf("builtBy:   %s\n", BuiltBy)
	fmt.Printf("go:        %s\n", runtime.Version())
	fmt.Printf("os/arch:   %s/%s\n", runtime.GOOS, runtime.GOARCH)
}
