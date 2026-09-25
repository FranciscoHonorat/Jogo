//go:build windows

package main

import (
	"bufio"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// setupConsole liga UTF-8 no terminal do Windows (senão os acentos saem quebrados).
func setupConsole() {
	k := syscall.NewLazyDLL("kernel32.dll")
	k.NewProc("SetConsoleOutputCP").Call(65001)
	k.NewProc("SetConsoleCP").Call(65001)
	if title, err := syscall.UTF16PtrFromString("Ordem & Caos - não feche esta janela"); err == nil {
		k.NewProc("SetConsoleTitleW").Call(uintptr(unsafe.Pointer(title)))
	}
}

// pauseBeforeExit mantém a janela aberta para a pessoa conseguir ler a mensagem de erro.
func pauseBeforeExit() {
	fmt.Println("\nAperte Enter para fechar.")
	bufio.NewReader(os.Stdin).ReadString('\n')
}
