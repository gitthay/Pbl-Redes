package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"vaiJunto/utils"
)

var leitor = bufio.NewReader(os.Stdin)

func lerTexto(rotulo string) string {
	fmt.Print(rotulo)
	texto, _ := leitor.ReadString('\n')
	return strings.TrimSpace(texto)
}

func main() {
	conexao, err := net.Dial("tcp", "localhost:8080")
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
			origem := lerTexto("Origem: ")
			destino := lerTexto("Destino: ")

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

			var numOpcao int
			fmt.Print("Digite o número da OPÇÃO desejada para reserva: ")
			fmt.Scanln(&numOpcao)

			if numOpcao < 1 || numOpcao > len(ultimosItinerarios) {
				fmt.Println("Opção inválida!")
				continue
			}

			itinerarioEscolhido := ultimosItinerarios[numOpcao-1]
			payload, _ := json.Marshal(itinerarioEscolhido)

			req := utils.MensagemRequisicao{Acao: utils.AcaoReservarTrecho, Usuario: email, Payload: payload}
			reqBytes, _ := json.Marshal(req)
			conexao.Write(reqBytes)

			n, _ := conexao.Read(buf)
			var resp utils.MensagemResposta
			json.Unmarshal(buf[:n], &resp)

			fmt.Println("\n>", resp.Mensagem)

		case "3":
			req := utils.MensagemRequisicao{Acao: utils.AcaoListarReservas, Usuario: email}
			reqBytes, _ := json.Marshal(req)
			conexao.Write(reqBytes)

			n, _ := conexao.Read(buf)
			var resp utils.MensagemResposta
			json.Unmarshal(buf[:n], &resp)

			var reservas []utils.Reserva
			json.Unmarshal(resp.Payload, &reservas)

			fmt.Println("\n--- MINHAS RESERVAS ---")
			for _, r := range reservas {
				fmt.Printf("ID Reserva: %s | Valor Total: R$ %.2f | Trechos: %d\n", r.ID, r.ValorTotal, len(r.Itinerario))
			}

		case "4":
			reservaID := lerTexto("Digite o ID da reserva para cancelar: ")
			payload, _ := json.Marshal(map[string]string{"reserva_id": reservaID})

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