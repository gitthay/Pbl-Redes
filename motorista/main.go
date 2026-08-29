package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

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
			origem := lerTexto("Cidade de Origem: ")
			destino := lerTexto("Cidade de Destino: ")
			preco := 35.0 // Exemplo fixo ou pode usar fmt.Sscanf
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
			req := utils.MensagemRequisicao{Acao: utils.AcaoConsultarCaronas, Usuario: email}
			reqBytes, _ := json.Marshal(req)
			conexao.Write(reqBytes)

			n, _ := conexao.Read(buf)
			var resp utils.MensagemResposta
			json.Unmarshal(buf[:n], &resp)

			var caronas []utils.Carona
			json.Unmarshal(resp.Payload, &caronas)

			fmt.Println("\n--- MINHAS CARONAS ---")
			for _, c := range caronas {
				fmt.Printf("ID: %s | Status Ativo: %t | Trechos: %d\n", c.ID, c.Ativa, len(c.Trechos))
			}

		case "3":
			caronaID := lerTexto("Digite o ID da Carona: ")
			payload, _ := json.Marshal(map[string]string{"carona_id": caronaID})
			req := utils.MensagemRequisicao{Acao: utils.AcaoConsultarPassageirosCarona, Usuario: email, Payload: payload}
			reqBytes, _ := json.Marshal(req)
			conexao.Write(reqBytes)

			n, _ := conexao.Read(buf)
			var resp utils.MensagemResposta
			json.Unmarshal(buf[:n], &resp)

			var passageiros []string
			json.Unmarshal(resp.Payload, &passageiros)

			fmt.Println("\n--- PASSAGEIROS CONFIRMADOS ---")
			for _, p := range passageiros {
				fmt.Println("- ", p)
			}

		case "4":
			caronaID := lerTexto("Digite o ID da carona para cancelar: ")
			payload, _ := json.Marshal(map[string]interface{}{"carona_id": caronaID, "confirmar": false})
			req := utils.MensagemRequisicao{Acao: utils.AcaoCancelarCarona, Usuario: email, Payload: payload}
			reqBytes, _ := json.Marshal(req)
			conexao.Write(reqBytes)

			n, _ := conexao.Read(buf)
			var resp utils.MensagemResposta
			json.Unmarshal(buf[:n], &resp)

			fmt.Println("\n>", resp.Mensagem)

			// Se houver passageiros afetados, pede confirmação do motorista
			if !resp.Sucesso && strings.Contains(resp.Mensagem, "ATENÇÃO") {
				confirmar := lerTexto("Deseja realmente cancelar? (s/n): ")
				if strings.ToLower(confirmar) == "s" {
					payloadConf, _ := json.Marshal(map[string]interface{}{"carona_id": caronaID, "confirmar": true})
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