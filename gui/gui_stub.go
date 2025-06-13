//go:build !gui
// +build !gui

package gui

import "fmt"

type GUI struct{}

func NewGUI() *GUI {
	return &GUI{}
}

func (g *GUI) Run() {
	fmt.Println("GUI support not compiled in. Please build with -tags gui flag.")
	fmt.Println("Example: go build -tags gui -o go-socks5-chain")
}