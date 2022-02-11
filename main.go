package main

import "github.com/netrixframework/netrix/cmd"

func main() {
	c := cmd.RootCmd()
	c.Execute()
}
