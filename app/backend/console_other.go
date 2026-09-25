//go:build !windows

package main

import (
	"bufio"
	"fmt"
	"os"
)

func setupConsole() {}

// pauseBeforeExit mantém a janela aberta (ao abrir com dois cliques no Mac/Linux) para ler a mensagem de erro.
func pauseBeforeExit() {
	if inContainer() {
		return
	}
	fmt.Println("\nAperte Enter para fechar.")
	bufio.NewReader(os.Stdin).ReadString('\n')
}
