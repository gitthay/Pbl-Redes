package main

import (
	"encoding/json"
	"fmt"
	"log"
	"io"
	"net"

	"vaiJunto/utils"
)

func gerenciarConexao(conexao net.Conn) {
	defer conexao.Close()

	buf := make([]byte, 2048)
	for {
		n, err := conexao.Read(buf)
		if err != nil {
			// Captura quando o cliente fecha a aplicação normalmente
			if err == io.EOF {
				fmt.Println("Cliente desconectou.")
				return // Sai do loop e encerra a goroutine
			}
			log.Println("Erro ao ler da conexão:", err)
			return
		}

		var req utils.MensagemRequisicao
		if err := json.Unmarshal(buf[:n], &req); err != nil {
			log.Println("JSON inválido:", err)
			return
		}

		switch req.Acao {
		case utils.AcaoPublicarCarona:
			var carona utils.Carona
			if err := json.Unmarshal(req.Payload, &carona); err != nil {
				log.Println("Erro ao ler payload da carona:", err)
				return
			}

			// Chama a função utilitária do pacote utils
			if err := utils.SalvarCarona(carona); err != nil {
				log.Println("Erro ao salvar carona:", err)

				resp := utils.MensagemResposta{
					Sucesso:  false,
					Mensagem: "Erro ao salvar carona no servidor.",
				}
				respBytes, _ := json.Marshal(resp)
				conexao.Write(respBytes)
				return
			}

			fmt.Println("Carona cadastrada e gravada no arquivo caronas.json!")

			resp := utils.MensagemResposta{
				Sucesso:  true,
				Mensagem: "Carona cadastrada com sucesso!",
			}
			respBytes, _ := json.Marshal(resp)
			conexao.Write(respBytes)

		case utils.AcaoBuscarItinerario:
			// 1. Decodifica os parâmetros de busca enviados pelo passageiro
			var filtro struct {
				Origem  string     `json:"origem"`
				Destino string     `json:"destino"`
				Data    utils.Data `json:"data"`
			}

			if err := json.Unmarshal(req.Payload, &filtro); err != nil {
				log.Println("Erro no payload da busca:", err)
				return
			}

			// 2. Carrega as caronas do JSON
			caronas, err := utils.CarregarCaronas()
			if err != nil {
				log.Println("Erro ao carregar caronas:", err)
				resp := utils.MensagemResposta{Sucesso: false, Mensagem: "Erro interno no servidor."}
				respBytes, _ := json.Marshal(resp)
				conexao.Write(respBytes)
				return
			}

			// 3. Executa a busca de rotas baseada em Grafo (BFS)
			itinerarios := utils.BuscarItinerarios(caronas, filtro.Origem, filtro.Destino, filtro.Data)

			payloadBytes, _ := json.Marshal(itinerarios)

			// 4. Devolve o resultado formatado
			resp := utils.MensagemResposta{
				Sucesso:  true,
				Mensagem: fmt.Sprintf("Encontrado(s) %d itinerário(s) disponível(is).", len(itinerarios)),
				Payload:  payloadBytes,
			}

			respBytes, _ := json.Marshal(resp)
			conexao.Write(respBytes)

		case utils.AcaoReservarTrecho:
			var itinerarioDesejado utils.Itinerario
			if err := json.Unmarshal(req.Payload, &itinerarioDesejado); err != nil {
				log.Println("Erro ao ler payload da reserva:", err)
				resp := utils.MensagemResposta{Sucesso: false, Mensagem: "Payload de reserva inválido."}
				respBytes, _ := json.Marshal(resp)
				conexao.Write(respBytes)
				return
			}

			// Executa o processo atômico de reserva
			reserva, err := utils.ReservarItinerario(req.Usuario, itinerarioDesejado)
			if err != nil {
				log.Println("Falha na reserva:", err)
				resp := utils.MensagemResposta{
					Sucesso:  false,
					Mensagem: err.Error(),
				}
				respBytes, _ := json.Marshal(resp)
				conexao.Write(respBytes)
				return
			}

			fmt.Printf("Reserva %s confirmada para o passageiro %s!\n", reserva.ID, req.Usuario)

			payloadBytes, _ := json.Marshal(reserva)
			resp := utils.MensagemResposta{
				Sucesso:  true,
				Mensagem: "Reserva confirmada com sucesso!",
				Payload:  payloadBytes,
			}
			respBytes, _ := json.Marshal(resp)
			conexao.Write(respBytes)

			case utils.AcaoListarReservas:
			reservas, err := utils.ListarReservasPassageiro(req.Usuario)
			if err != nil {
				resp := utils.MensagemResposta{Sucesso: false, Mensagem: "Erro ao buscar reservas."}
				respBytes, _ := json.Marshal(resp)
				conexao.Write(respBytes)
				continue
			}

			payloadBytes, _ := json.Marshal(reservas)
			resp := utils.MensagemResposta{
				Sucesso:  true,
				Mensagem: fmt.Sprintf("Encontrada(s) %d reserva(s).", len(reservas)),
				Payload:  payloadBytes,
			}
			respBytes, _ := json.Marshal(resp)
			conexao.Write(respBytes)

		case utils.AcaoCancelarReserva:
			var reqCancelamento struct {
				ReservaID string `json:"reserva_id"`
			}
			if err := json.Unmarshal(req.Payload, &reqCancelamento); err != nil {
				resp := utils.MensagemResposta{Sucesso: false, Mensagem: "Payload de cancelamento inválido."}
				respBytes, _ := json.Marshal(resp)
				conexao.Write(respBytes)
				continue
			}

			err := utils.CancelarReserva(reqCancelamento.ReservaID, req.Usuario)
			if err != nil {
				resp := utils.MensagemResposta{Sucesso: false, Mensagem: err.Error()}
				respBytes, _ := json.Marshal(resp)
				conexao.Write(respBytes)
				continue
			}

			resp := utils.MensagemResposta{
				Sucesso:  true,
				Mensagem: "Reserva cancelada e assentos liberados com sucesso!",
			}
			respBytes, _ := json.Marshal(resp)
			conexao.Write(respBytes)

			case utils.AcaoConsultarCaronas:
			caronas, err := utils.ConsultarCaronasMotorista(req.Usuario)
			if err != nil {
				resp := utils.MensagemResposta{Sucesso: false, Mensagem: "Erro ao buscar caronas do motorista."}
				respBytes, _ := json.Marshal(resp)
				conexao.Write(respBytes)
				continue
			}

			payloadBytes, _ := json.Marshal(caronas)
			resp := utils.MensagemResposta{
				Sucesso:  true,
				Mensagem: fmt.Sprintf("Encontrada(s) %d carona(s) publicada(s).", len(caronas)),
				Payload:  payloadBytes,
			}
			respBytes, _ := json.Marshal(resp)
			conexao.Write(respBytes)

		case utils.AcaoCancelarCarona:
			var reqCancelamento struct {
				CaronaID  string `json:"carona_id"`
				Confirmar bool   `json:"confirmar"` // Flag enviada se o motorista deu "SIM" na confirmação
			}

			if err := json.Unmarshal(req.Payload, &reqCancelamento); err != nil {
				resp := utils.MensagemResposta{Sucesso: false, Mensagem: "Payload de cancelamento inválido."}
				respBytes, _ := json.Marshal(resp)
				conexao.Write(respBytes)
				continue
			}

			requerConfirmacao, msg, err := utils.CancelarCaronaMotorista(reqCancelamento.CaronaID, req.Usuario, reqCancelamento.Confirmar)
			if err != nil {
				resp := utils.MensagemResposta{Sucesso: false, Mensagem: err.Error()}
				respBytes, _ := json.Marshal(resp)
				conexao.Write(respBytes)
				continue
			}

			// Retorna se exige confirmação ou se o cancelamento foi finalizado
			resp := utils.MensagemResposta{
				Sucesso:  !requerConfirmacao, // Se requer confirmação, vem Sucesso: false para o cliente tratar o aviso
				Mensagem: msg,
			}
			respBytes, _ := json.Marshal(resp)
			conexao.Write(respBytes)

		case utils.AcaoConsultarPassageirosCarona:
			var reqPassageiros struct {
				CaronaID string `json:"carona_id"`
			}
			json.Unmarshal(req.Payload, &reqPassageiros)

			passageiros, err := utils.ConsultarPassageirosCarona(reqPassageiros.CaronaID, req.Usuario)
			if err != nil {
				resp := utils.MensagemResposta{Sucesso: false, Mensagem: "Erro ao consultar passageiros."}
				respBytes, _ := json.Marshal(resp)
				conexao.Write(respBytes)
				continue
			}

			payloadBytes, _ := json.Marshal(passageiros)
			resp := utils.MensagemResposta{
				Sucesso:  true,
				Mensagem: fmt.Sprintf("Passageiros na carona %s:", reqPassageiros.CaronaID),
				Payload:  payloadBytes,
			}
			respBytes, _ := json.Marshal(resp)
			conexao.Write(respBytes)

			case utils.AcaoAutenticar:
			var reqAuth struct {
				Email string            `json:"email"`
				Senha string            `json:"senha"`
				Tipo  utils.TipoUsuario `json:"tipo"`
			}

			if err := json.Unmarshal(req.Payload, &reqAuth); err != nil {
				resp := utils.MensagemResposta{Sucesso: false, Mensagem: "Payload de autenticação inválido."}
				respBytes, _ := json.Marshal(resp)
				conexao.Write(respBytes)
				continue
			}

			sucesso, msg, err := utils.AutenticarOuCadastrarUsuario(reqAuth.Email, reqAuth.Senha, reqAuth.Tipo)
			if err != nil || !sucesso {
				resp := utils.MensagemResposta{Sucesso: false, Mensagem: msg}
				respBytes, _ := json.Marshal(resp)
				conexao.Write(respBytes)
				continue
			}

			resp := utils.MensagemResposta{Sucesso: true, Mensagem: msg}
			respBytes, _ := json.Marshal(resp)
			conexao.Write(respBytes)
		}

		
	}
}

func main() {
	ln, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		log.Fatalln(err)
	}
	defer ln.Close()

	fmt.Println("Servidor rodando na porta 8080...")

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go gerenciarConexao(conn)
	}
}
