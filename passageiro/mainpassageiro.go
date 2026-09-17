package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"vaiJunto/utils"
)

var leitor = bufio.NewReader(os.Stdin)

func lerTexto(rotulo string) string {
	fmt.Print(rotulo)
	texto, _ := leitor.ReadString('\n')
	// Remove a quebra de linha (\n ou \r\n) e espaços extras antes e depois
	return strings.TrimSpace(texto)
}

// função auxiliar para consultar as reservas do cliente
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

	//AUTENTICAÇÃO 
	var email string
	for {
		email = lerTexto("Digite seu Email: ")
		senha := lerTexto("Digite sua Senha: ")

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

		n, err := conexao.Read(buf)
		if err != nil {
			log.Fatalln("Erro de comunicação com o servidor:", err)
		}

		var respAuth utils.MensagemResposta
		json.Unmarshal(buf[:n], &respAuth)

		if !respAuth.Sucesso {
			fmt.Printf("\n[ERRO]: %s Tente novamente.\n\n", respAuth.Mensagem)
			continue
		}

		fmt.Println("\n[SUCESSO]:", respAuth.Mensagem)
		break
	}

	// MENU PRINCIPAL
	for {
		fmt.Println("\n------------------------------------------")
		fmt.Println("1. Buscar/Reservar Caronas")
		fmt.Println("2. Minhas Reservas Ativas")
		fmt.Println("3. Cancelar Reserva")
		fmt.Println("0. Sair")
		fmt.Println("------------------------------------------")

		opcao := lerTexto("Escolha uma opção: ")

		switch opcao {
		case "1":
			fmt.Println("\n--- BUSCAR ITINERÁRIOS (Digite 0 em qualquer campo para voltar) ---")
			origem := lerTexto("Origem: ")
			if origem == "0" {
				continue
			}

			destino := lerTexto("Destino: ")
			if destino == "0" {
				continue
			}

			// Leitura e validação da data
			var dataFiltro utils.Data
			cancelarBusca := false
			for {
				dataStr := lerTexto("Data da viagem (DD/MM/AAAA): ")
				if dataStr == "0" {
					cancelarBusca = true
					break
				}
				partes := strings.Split(dataStr, "/")
				if len(partes) != 3 {
					fmt.Println("[ERRO]: Formato inválido. Use DD/MM/AAAA. Tente novamente.")
					continue
				}
				d, errD := strconv.Atoi(partes[0])
				m, errM := strconv.Atoi(partes[1])
				a, errA := strconv.Atoi(partes[2])
				if errD != nil || errM != nil || errA != nil || d < 1 || d > 31 || m < 1 || m > 12 {
					fmt.Println("[ERRO]: Data inválida. Tente novamente.")
					continue
				}
				dataFiltro = utils.NovaData(d, m, a)
				break
			}
			if cancelarBusca {
				continue
			}

			filtro := struct {
				Origem  string     `json:"origem"`
				Destino string     `json:"destino"`
				Data    utils.Data `json:"data"`
			}{
				Origem:  origem,
				Destino: destino,
				Data:    dataFiltro,
			}

			payload, _ := json.Marshal(filtro)
			req := utils.MensagemRequisicao{Acao: utils.AcaoBuscarItinerario, Usuario: email, Payload: payload}
			reqBytes, _ := json.Marshal(req)
			conexao.Write(reqBytes)

			n, _ := conexao.Read(buf)
			var resp utils.MensagemResposta
			json.Unmarshal(buf[:n], &resp)

			var itinerariosEncontrados []utils.Itinerario
			json.Unmarshal(resp.Payload, &itinerariosEncontrados)

			fmt.Println("\n==========================================")
			fmt.Printf("   %s\n", resp.Mensagem)
			fmt.Println("==========================================")

			if len(itinerariosEncontrados) == 0 {
				fmt.Println("Nenhum itinerário encontrado para a rota e data especificadas.")
				continue
			}

			for i, itin := range itinerariosEncontrados {
				fmt.Printf("\nOPÇÃO [%d] - Preço Total: R$ %.2f\n", i+1, itin.PrecoTotal)
				for j, t := range itin.Trechos {
					fmt.Printf("  Trecho %d: %s -> %s | Partida: %s | Chegada: %s (Carona ID: %s | Motorista: %s)\n",
						j+1, t.Origem, t.Destino, t.HorarioPartida.Format("15:04"), t.HorarioChegada.Format("15:04"), t.CaronaID, t.MotoristaID)
				}
			}

			// Pergunta se deseja reservar ou voltar (repete se digitar algo inválido)
			fmt.Println("\n------------------------------------------")
			var desejaReservar string
			for {
				desejaReservar = strings.ToLower(lerTexto("Deseja reservar algum desses itinerários? (s/n): "))
				if desejaReservar == "s" || desejaReservar == "n" {
					break
				}
				fmt.Println("[ERRO]: Digite apenas 's' para sim ou 'n' para não.")
			}

			if desejaReservar == "n" {
				continue
			}

			input := lerTexto("Digite o número da OPÇÃO desejada (ou 0 para cancelar): ")
			num, err := strconv.Atoi(input)
			if err != nil || num == 0 || num > len(itinerariosEncontrados) {
				fmt.Println("Reserva não realizada.")
				continue
			}

			itinerarioEscolhido := itinerariosEncontrados[num-1]
			payloadReserva, _ := json.Marshal(itinerarioEscolhido)

			reqReserva := utils.MensagemRequisicao{Acao: utils.AcaoReservarTrecho, Usuario: email, Payload: payloadReserva}
			reqReservaBytes, _ := json.Marshal(reqReserva)
			conexao.Write(reqReservaBytes)

			n, _ = conexao.Read(buf)
			var respReserva utils.MensagemResposta
			json.Unmarshal(buf[:n], &respReserva)

			fmt.Println("\n>", respReserva.Mensagem)

		case "2":
			reservas := buscarReservas(conexao, email, buf)
			fmt.Println("\n--- MINHAS RESERVAS ---")
			if len(reservas) == 0 {
				fmt.Println("Você não possui reservas ativas.")
				continue
			}
			for i, r := range reservas {
				fmt.Printf("[%d] ID Reserva: %s | Valor Total: R$ %.2f | Trechos: %d\n", i+1, r.ID, r.ValorTotal, len(r.Itinerario))
			}

		case "3":
			reservas := buscarReservas(conexao, email, buf)
			fmt.Println("\n==========================================")
			fmt.Println("           CANCELAR RESERVA               ")
			fmt.Println("==========================================")

			if len(reservas) == 0 {
				fmt.Println("Você não possui reservas ativas para cancelar.")
				continue
			}

			// Lista todas as reservas com detalhes completos e ID da reserva preservado
			for i, r := range reservas {
				origemGeral := "Indefinida"
				destinoGeral := "Indefinido"
				dataHoraStr := "Indefinida"

				if len(r.Itinerario) > 0 {
					origemGeral = r.Itinerario[0].Origem
					destinoGeral = r.Itinerario[len(r.Itinerario)-1].Destino
					dataHoraStr = fmt.Sprintf("%s às %s",
						r.Itinerario[0].HorarioPartida.Format("02/01/2006"),
						r.Itinerario[0].HorarioPartida.Format("15:04"))
				}

				fmt.Printf("\n[%d] ID Reserva: %s | Rota: %s -> %s\n", i+1, r.ID, origemGeral, destinoGeral)
				fmt.Printf("    Partida: %s | Valor Total: R$ %.2f | Trechos: %d\n", dataHoraStr, r.ValorTotal, len(r.Itinerario))

				// Detalha os trechos individuais com ID da carona preservado
				for j, t := range r.Itinerario {
					fmt.Printf("      Trecho %d: %s -> %s (Carona ID: %s | Motorista: %s | Partida: %s | Chegada: %s)\n",
						j+1, t.Origem, t.Destino, t.CaronaID, t.MotoristaID,
						t.HorarioPartida.Format("15:04"), t.HorarioChegada.Format("15:04"))
				}
			}

			fmt.Println("\n[0] Voltar ao menu")

			input := lerTexto("Escolha o número da reserva para cancelar: ")
			num, err := strconv.Atoi(input)
			if err != nil || num == 0 || num > len(reservas) {
				continue
			}

			reservaEscolhida := reservas[num-1]

			// Confirmação antes de enviar o cancelamento
			confirmacao := lerTexto(fmt.Sprintf("Deseja realmente cancelar a reserva %s? (s/n): ", reservaEscolhida.ID))
			if strings.ToLower(confirmacao) != "s" {
				fmt.Println("Operação cancelada.")
				continue
			}

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
