package main

import (
	"cabourotte/healthcheck"
	"fmt"
)

func main() {
	i := healthcheck.Protocol(1)
	fmt.Printf("%d", i)
}
