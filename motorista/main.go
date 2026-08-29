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
	"time"

	"vaiJunto/utils"
)

var leitor = bufio.NewReader(os.Stdin)

func lerTexto(rotulo string) string {
	fmt.Print(rotulo)
	texto, _ := leitor.ReadString('\n')
	// Remove a quebra de linha (\n ou \r\n) e espaços extras antes e depois
	return strings.TrimSpace(texto) 
}

// BuscarCaronasDoMotorista é uma função auxiliar para consultar a lista atual de caronas
func buscarCaronas(conexao net.Conn, email string, buf []byte) []utils.Carona {
	req := utils.MensagemRequisicao{Acao: utils.AcaoConsultarCaronas, Usuario: email}
	reqBytes, _ := json.Marshal(req)
	conexao.Write(reqBytes)

	n, _ := conexao.Read(buf)
	var resp utils.MensagemResposta
	json.Unmarshal(buf[:n], &resp)

	var caronas []utils.Carona
	json.Unmarshal(resp.Payload, &caronas)
	return caronas
}

func main() {
	conexao, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		log.Fatalln("Erro ao conectar ao servidor:", err)
	}
	defer conexao.Close()

	buf := make([]byte, 8192)

	fmt.Println("==========================================")
	fmt.Println("        VAIJUNTO - MOTORISTA              ")
	fmt.Println("==========================================")

	email := lerTexto("Digite seu Email: ")
	senha := lerTexto("Digite sua Senha: ")

	// --- AUTENTICAÇÃO ---
	payloadAuth, _ := json.Marshal(map[string]interface{}{
		"email": email,
		"senha": senha,
		"tipo":  utils.TipoMotorista,
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

	// --- MENU PRINCIPAL ---
	for {
		fmt.Println("\n------------------------------------------")
		fmt.Println("1. Publicar Nova Carona")
		fmt.Println("2. Listar Minhas Caronas")
		fmt.Println("3. Consultar Passageiros de uma Carona")
		fmt.Println("4. Cancelar Carona")
		fmt.Println("0. Sair")
		fmt.Println("------------------------------------------")

		opcao := lerTexto("Escolha uma opção: ")

		switch opcao {
		case "1":
			fmt.Println("\n--- PUBLICAR CARONA (Digite 0 em qualquer campo para voltar) ---")
			origem := lerTexto("Cidade de Origem: ")
			if origem == "0" { continue }

			destino := lerTexto("Cidade de Destino: ")
			if destino == "0" { continue }

			preco := 35.0
			assentos := 4

			novaCarona := utils.Carona{
				ID:          fmt.Sprintf("CAR-%d", time.Now().UnixNano()),
				MotoristaID: email,
				AssentosTot: assentos,
				Ativa:       true,
				Trechos: []utils.Trecho{
					{
						Origem:         origem,
						Destino:        destino,
						DataPartida:    utils.NovaData(15, 10, 2026),
						HorarioPartida: time.Now(),
						HorarioChegada: time.Now().Add(2 * time.Hour),
						Preco:          preco,
						AssentosLivre:  assentos,
					},
				},
			}

			payload, _ := json.Marshal(novaCarona)
			req := utils.MensagemRequisicao{Acao: utils.AcaoPublicarCarona, Usuario: email, Payload: payload}
			reqBytes, _ := json.Marshal(req)
			conexao.Write(reqBytes)

			n, _ := conexao.Read(buf)
			var resp utils.MensagemResposta
			json.Unmarshal(buf[:n], &resp)
			fmt.Println("\n>", resp.Mensagem)

		case "2":
			caronas := buscarCaronas(conexao, email, buf)
			fmt.Println("\n--- MINHAS CARONAS ---")
			if len(caronas) == 0 {
				fmt.Println("Nenhuma carona encontrada.")
				continue
			}
			for i, c := range caronas {
				status := "Ativa"
				if !c.Ativa { status = "Cancelada" }
				fmt.Printf("[%d] ID: %s | Status: %s | Trechos: %d\n", i+1, c.ID, status, len(c.Trechos))
			}

		case "3":
			caronas := buscarCaronas(conexao, email, buf)
			fmt.Println("\n--- CONSULTAR PASSAGEIROS ---")
			if len(caronas) == 0 {
				fmt.Println("Nenhuma carona encontrada.")
				continue
			}

			for i, c := range caronas {
				fmt.Printf("[%d] ID: %s | Trechos: %d\n", i+1, c.ID, len(c.Trechos))
			}
			fmt.Println("[0] Voltar ao menu")

			input := lerTexto("Escolha o número da carona: ")
			num, err := strconv.Atoi(input)
			if err != nil || num == 0 || num > len(caronas) {
				continue
			}

			caronaEscolhida := caronas[num-1]
			payload, _ := json.Marshal(map[string]string{"carona_id": caronaEscolhida.ID})
			req := utils.MensagemRequisicao{Acao: utils.AcaoConsultarPassageirosCarona, Usuario: email, Payload: payload}
			reqBytes, _ := json.Marshal(req)
			conexao.Write(reqBytes)

			n, _ := conexao.Read(buf)
			var resp utils.MensagemResposta
			json.Unmarshal(buf[:n], &resp)

			var passageiros []string
			json.Unmarshal(resp.Payload, &passageiros)

			fmt.Println("\n--- PASSAGEIROS CONFIRMADOS ---")
			if len(passageiros) == 0 {
				fmt.Println("Nenhum passageiro reservou esta carona ainda.")
			} else {
				for _, p := range passageiros {
					fmt.Println("- ", p)
				}
			}

		case "4":
			caronas := buscarCaronas(conexao, email, buf)
			fmt.Println("\n--- CANCELAR CARONA ---")
			if len(caronas) == 0 {
				fmt.Println("Nenhuma carona encontrada.")
				continue
			}

			for i, c := range caronas {
				if c.Ativa {
					fmt.Printf("[%d] ID: %s | Trechos: %d\n", i+1, c.ID, len(c.Trechos))
				}
			}
			fmt.Println("[0] Voltar ao menu")

			input := lerTexto("Escolha o número da carona para cancelar: ")
			num, err := strconv.Atoi(input)
			if err != nil || num == 0 || num > len(caronas) {
				continue
			}

			caronaEscolhida := caronas[num-1]
			payload, _ := json.Marshal(map[string]interface{}{"carona_id": caronaEscolhida.ID, "confirmar": false})
			req := utils.MensagemRequisicao{Acao: utils.AcaoCancelarCarona, Usuario: email, Payload: payload}
			reqBytes, _ := json.Marshal(req)
			conexao.Write(reqBytes)

			n, _ := conexao.Read(buf)
			var resp utils.MensagemResposta
			json.Unmarshal(buf[:n], &resp)

			fmt.Println("\n>", resp.Mensagem)

			if !resp.Sucesso && strings.Contains(resp.Mensagem, "ATENÇÃO") {
				confirmar := lerTexto("Deseja realmente cancelar? (s/n ou 0 para voltar): ")
				if strings.ToLower(confirmar) == "s" {
					payloadConf, _ := json.Marshal(map[string]interface{}{"carona_id": caronaEscolhida.ID, "confirmar": true})
					req.Payload = payloadConf
					reqBytes, _ = json.Marshal(req)
					conexao.Write(reqBytes)

					n, _ = conexao.Read(buf)
					json.Unmarshal(buf[:n], &resp)
					fmt.Println("\n>", resp.Mensagem)
				}
			}

		case "0":
			fmt.Println("Saindo...")
			return
		}
	}
}