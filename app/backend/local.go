package main

// Modo "clicar duas vezes": descobre o endereço na rede local, abre o navegador
// e mostra instruções amigáveis na janela do terminal.

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
)

// interfaces virtuais (Docker, VMs, VPN) que os alunos não conseguem alcançar
var virtualIface = []string{"docker", "veth", "br-", "virbr", "vmnet", "vbox", "virtualbox", "utun", "tun", "tap", "zt", "tailscale", "wg", "hyper-v", "vethernet", "lo"}

// lanURLs devolve os endereços pelos quais outros aparelhos da mesma rede
// acessam o jogo, do mais provável para o menos provável.
func lanURLs(port string) []string {
	type cand struct {
		ip   string
		rank int
	}
	var cs []cand
	ifaces, _ := net.Interfaces()
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		name := strings.ToLower(ifc.Name)
		skip := false
		for _, v := range virtualIface {
			if strings.HasPrefix(name, v) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			ipn, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipn.IP.To4()
			if ip == nil || !ip.IsPrivate() {
				continue
			}
			rank := 2 // 172.16–31: o menos comum em Wi-Fi de casa/escola
			switch {
			case ip[0] == 192 && ip[1] == 168:
				rank = 0
			case ip[0] == 10:
				rank = 1
			}
			cs = append(cs, cand{ip.String(), rank})
		}
	}
	sort.SliceStable(cs, func(i, j int) bool { return cs[i].rank < cs[j].rank })
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, "http://"+c.ip+":"+port)
	}
	return out
}

// inContainer: dentro do Docker (servidor na nuvem) não abrimos navegador nem mostramos o endereço local.
func inContainer() bool {
	_, err := os.Stat("/.dockerenv")
	return err == nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

func printWelcome(port string, lan []string) {
	line := strings.Repeat("=", 62)
	fmt.Println()
	fmt.Println(line)
	fmt.Println("   ORDEM & CAOS  -  A Metamorfose")
	fmt.Println(line)
	fmt.Println()
	fmt.Println("   O jogo esta rodando! O navegador vai abrir sozinho.")
	fmt.Println()
	fmt.Println("   Neste computador:   http://localhost:" + port)
	if len(lan) > 0 {
		fmt.Println()
		fmt.Println("   Para os alunos (conectados na MESMA rede Wi-Fi):")
		fmt.Println()
		fmt.Println("       " + lan[0])
		for _, u := range lan[1:] {
			fmt.Println("       (ou tente: " + u + ")")
		}
		fmt.Println()
		fmt.Println("   Ou mostre o QR code que aparece na tela do jogo.")
	} else {
		fmt.Println()
		fmt.Println("   Atencao: este computador nao parece estar conectado")
		fmt.Println("   a uma rede. Conecte no Wi-Fi para os alunos entrarem.")
	}
	fmt.Println()
	fmt.Println("   NAO FECHE ESTA JANELA durante o jogo.")
	fmt.Println("   Para encerrar, e so fechar esta janela.")
	fmt.Println(line)
	fmt.Println()
}
