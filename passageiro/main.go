package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"flag"

	"vaiJunto/utils"
)

var leitor = bufio.NewReader(os.Stdin)

func lerTexto(rotulo string) string {
	fmt.Print(rotulo)
	texto, _ := leitor.ReadString('\n')
	// Remove a quebra de linha (\n ou \r\n) e espaços extras antes e depois
	return strings.TrimSpace(texto)
}

// BuscarReservasDoPassageiro é uma função auxiliar para consultar as reservas do cliente
func buscarReservas(conexao net.Conn, email string, buf []byte) []utils.Reserva {
	req := utils.MensagemRequisicao{Acao: utils.AcaoListarReservas, Usuario: email}
	reqBytes, _ := json.Marshal(req)
	conexao.Write(reqBytes)

	n, _ := conexao.Read(buf)
	var resp utils.MensagemResposta
	json.Unmarshal(buf[:n], &resp)

	var reservas []utils.Reserva
	json.Unmarshal(resp.Payload, &reservas)
	return reservas
}

func main() {
	// Flag para aceitar IP dinâmico na execução do terminal
	serverAddr := flag.String("server", "localhost:8080", "Endereço do servidor TCP (ex: 192.168.1.15:8080)")
	flag.Parse()

	// Conecta ao endereço configurado
	conexao, err := net.Dial("tcp", *serverAddr)
	if err != nil {
		log.Fatalln("Erro ao conectar ao servidor:", err)
	}
	defer conexao.Close()

	buf := make([]byte, 8192)

	fmt.Println("==========================================")
	fmt.Println("        VAIJUNTO - PASSAGEIRO             ")
	fmt.Println("==========================================")

	email := lerTexto("Digite seu Email: ")
	senha := lerTexto("Digite sua Senha: ")

	// --- AUTENTICAÇÃO ---
	payloadAuth, _ := json.Marshal(map[string]interface{}{
		"email": email,
		"senha": senha,
		"tipo":  utils.TipoPassageiro,
	})

	reqAuth := utils.MensagemRequisicao{
		Acao:    utils.AcaoAutenticar,
		Usuario: email,
		Payload: payloadAuth,
	}
	reqAuthBytes, _ := json.Marshal(reqAuth)
	conexao.Write(reqAuthBytes)

	n, _ := conexao.Read(buf)
	var respAuth utils.MensagemResposta
	json.Unmarshal(buf[:n], &respAuth)

	if !respAuth.Sucesso {
		fmt.Println("\n[ERRO]:", respAuth.Mensagem)
		return
	}
	fmt.Println("\n[SUCESSO]: Login realizado com sucesso!")

	var ultimosItinerarios []utils.Itinerario

	// --- MENU PRINCIPAL ---
	for {
		fmt.Println("\n------------------------------------------")
		fmt.Println("1. Buscar Itinerários/Caronas")
		fmt.Println("2. Reservar um Itinerário")
		fmt.Println("3. Minhas Reservas Ativas")
		fmt.Println("4. Cancelar Reserva")
		fmt.Println("0. Sair")
		fmt.Println("------------------------------------------")

		opcao := lerTexto("Escolha uma opção: ")

		switch opcao {
		case "1":
			fmt.Println("\n--- BUSCAR ITINERÁRIOS (Digite 0 para voltar) ---")
			origem := lerTexto("Origem: ")
			if origem == "0" {
				continue
			}

			destino := lerTexto("Destino: ")
			if destino == "0" {
				continue
			}

			filtro := struct {
				Origem  string     `json:"origem"`
				Destino string     `json:"destino"`
				Data    utils.Data `json:"data"`
			}{
				Origem:  origem,
				Destino: destino,
				Data:    utils.NovaData(15, 10, 2026),
			}

			payload, _ := json.Marshal(filtro)
			req := utils.MensagemRequisicao{Acao: utils.AcaoBuscarItinerario, Usuario: email, Payload: payload}
			reqBytes, _ := json.Marshal(req)
			conexao.Write(reqBytes)

			n, _ := conexao.Read(buf)
			var resp utils.MensagemResposta
			json.Unmarshal(buf[:n], &resp)

			json.Unmarshal(resp.Payload, &ultimosItinerarios)

			fmt.Println("\n==========================================")
			fmt.Printf("   %s\n", resp.Mensagem)
			fmt.Println("==========================================")

			for i, itin := range ultimosItinerarios {
				fmt.Printf("\nOPÇÃO %d - Preço Total: R$ %.2f\n", i+1, itin.PrecoTotal)
				for j, t := range itin.Trechos {
					fmt.Printf("  Trecho %d: %s -> %s (Carona: %s | Motorista: %s)\n", j+1, t.Origem, t.Destino, t.CaronaID, t.MotoristaID)
				}
			}

		case "2":
			if len(ultimosItinerarios) == 0 {
				fmt.Println("\nFaça uma busca (Opção 1) antes de solicitar a reserva!")
				continue
			}

			fmt.Println("\nDigite o número da OPÇÃO desejada (ou 0 para voltar):")
			input := lerTexto("Opção: ")
			num, err := strconv.Atoi(input)
			if err != nil || num == 0 || num > len(ultimosItinerarios) {
				continue
			}

			itinerarioEscolhido := ultimosItinerarios[num-1]
			payload, _ := json.Marshal(itinerarioEscolhido)

			req := utils.MensagemRequisicao{Acao: utils.AcaoReservarTrecho, Usuario: email, Payload: payload}
			reqBytes, _ := json.Marshal(req)
			conexao.Write(reqBytes)

			n, _ := conexao.Read(buf)
			var resp utils.MensagemResposta
			json.Unmarshal(buf[:n], &resp)

			fmt.Println("\n>", resp.Mensagem)

		case "3":
			reservas := buscarReservas(conexao, email, buf)
			fmt.Println("\n--- MINHAS RESERVAS ---")
			if len(reservas) == 0 {
				fmt.Println("Você não possui reservas ativas.")
				continue
			}
			for i, r := range reservas {
				fmt.Printf("[%d] ID Reserva: %s | Valor Total: R$ %.2f | Trechos: %d\n", i+1, r.ID, r.ValorTotal, len(r.Itinerario))
			}

		case "4":
			reservas := buscarReservas(conexao, email, buf)
			fmt.Println("\n--- CANCELAR RESERVA ---")
			if len(reservas) == 0 {
				fmt.Println("Você não possui reservas para cancelar.")
				continue
			}

			for i, r := range reservas {
				fmt.Printf("[%d] ID Reserva: %s | Valor Total: R$ %.2f\n", i+1, r.ID, r.ValorTotal)
			}
			fmt.Println("[0] Voltar ao menu")

			input := lerTexto("Escolha o número da reserva para cancelar: ")
			num, err := strconv.Atoi(input)
			if err != nil || num == 0 || num > len(reservas) {
				continue
			}

			reservaEscolhida := reservas[num-1]
			payload, _ := json.Marshal(map[string]string{"reserva_id": reservaEscolhida.ID})

			req := utils.MensagemRequisicao{Acao: utils.AcaoCancelarReserva, Usuario: email, Payload: payload}
			reqBytes, _ := json.Marshal(req)
			conexao.Write(reqBytes)

			n, _ := conexao.Read(buf)
			var resp utils.MensagemResposta
			json.Unmarshal(buf[:n], &resp)

			fmt.Println("\n>", resp.Mensagem)

		case "0":
			fmt.Println("Saindo...")
			return
		}
	}
}
