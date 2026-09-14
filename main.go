package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"

	"vaiJunto/utils"
)

func gerenciarConexao(conexao net.Conn) {
	defer conexao.Close()
	clienteAddr := conexao.RemoteAddr().String()
	utils.RegistrarLog("[CONEXAO] Novo cliente conectado de: %s", clienteAddr)

	buf := make([]byte, 2048)
	for {
		n, err := conexao.Read(buf)
		if err != nil {
			if err == io.EOF {
				fmt.Println("Cliente desconectou.")
				utils.RegistrarLog("[DESCONEXAO] Cliente %s desconectou normalmente.", clienteAddr)
				return
			}
			log.Println("Erro ao ler da conexão:", err)
			utils.RegistrarLog("[ERRO CONEXAO] Erro na leitura do cliente %s: %v", clienteAddr, err)
			return
		}

		var req utils.MensagemRequisicao
		if err := json.Unmarshal(buf[:n], &req); err != nil {
			log.Println("JSON inválido:", err)
			utils.RegistrarLog("[ERRO JSON] Cliente %s enviou JSON malformado: %s", clienteAddr, string(buf[:n]))
			return
		}

		// Log da Requisição
		utils.RegistrarLog("[REQUISICAO] De: %s | Usuario: %s | Acao: %s | Payload: %s",
			clienteAddr, req.Usuario, req.Acao, string(req.Payload))

		// Função auxiliar interna para responder e logar
		enviarResposta := func(resp utils.MensagemResposta) {
			respBytes, _ := json.Marshal(resp)
			conexao.Write(respBytes)
			utils.RegistrarLog("[RESPOSTA] Para: %s (Usuario: %s) | Acao: %s | Sucesso: %t | Msg: %s | Payload: %s",
				clienteAddr, req.Usuario, req.Acao, resp.Sucesso, resp.Mensagem, string(resp.Payload))
		}

		switch req.Acao {
		case utils.AcaoPublicarCarona:
			var carona utils.Carona
			if err := json.Unmarshal(req.Payload, &carona); err != nil {
				log.Println("Erro ao ler payload da carona:", err)
				enviarResposta(utils.MensagemResposta{Sucesso: false, Mensagem: "Payload inválido."})
				return
			}

			if err := utils.SalvarCarona(carona); err != nil {
				log.Println("Erro ao salvar carona:", err)
				enviarResposta(utils.MensagemResposta{Sucesso: false, Mensagem: "Erro ao salvar carona no servidor."})
				return
			}

			fmt.Println("Carona cadastrada e gravada no arquivo caronas.json!")
			enviarResposta(utils.MensagemResposta{Sucesso: true, Mensagem: "Carona cadastrada com sucesso!"})

		case utils.AcaoBuscarItinerario:
			var filtro struct {
				Origem  string     `json:"origem"`
				Destino string     `json:"destino"`
				Data    utils.Data `json:"data"`
			}

			if err := json.Unmarshal(req.Payload, &filtro); err != nil {
				log.Println("Erro no payload da busca:", err)
				enviarResposta(utils.MensagemResposta{Sucesso: false, Mensagem: "Payload de busca inválido."})
				return
			}

			caronas, err := utils.CarregarCaronas()
			if err != nil {
				log.Println("Erro ao carregar caronas:", err)
				enviarResposta(utils.MensagemResposta{Sucesso: false, Mensagem: "Erro interno no servidor."})
				return
			}

			itinerarios := utils.BuscarItinerarios(caronas, filtro.Origem, filtro.Destino, filtro.Data)
			payloadBytes, _ := json.Marshal(itinerarios)

			enviarResposta(utils.MensagemResposta{
				Sucesso:  true,
				Mensagem: fmt.Sprintf("Encontrado(s) %d itinerário(s) disponível(is).", len(itinerarios)),
				Payload:  payloadBytes,
			})

		case utils.AcaoReservarTrecho:
			var itinerarioDesejado utils.Itinerario
			if err := json.Unmarshal(req.Payload, &itinerarioDesejado); err != nil {
				log.Println("Erro ao ler payload da reserva:", err)
				enviarResposta(utils.MensagemResposta{Sucesso: false, Mensagem: "Payload de reserva inválido."})
				return
			}

			reserva, err := utils.ReservarItinerario(req.Usuario, itinerarioDesejado)
			if err != nil {
				log.Println("Falha na reserva:", err)
				enviarResposta(utils.MensagemResposta{Sucesso: false, Mensagem: err.Error()})
				return
			}

			fmt.Printf("Reserva %s confirmada para o passageiro %s!\n", reserva.ID, req.Usuario)
			payloadBytes, _ := json.Marshal(reserva)

			enviarResposta(utils.MensagemResposta{
				Sucesso:  true,
				Mensagem: "Reserva confirmada com sucesso!",
				Payload:  payloadBytes,
			})

		case utils.AcaoListarReservas:
			reservas, err := utils.ListarReservasPassageiro(req.Usuario)
			if err != nil {
				enviarResposta(utils.MensagemResposta{Sucesso: false, Mensagem: "Erro ao buscar reservas."})
				continue
			}

			payloadBytes, _ := json.Marshal(reservas)
			enviarResposta(utils.MensagemResposta{
				Sucesso:  true,
				Mensagem: fmt.Sprintf("Encontrada(s) %d reserva(s).", len(reservas)),
				Payload:  payloadBytes,
			})

		case utils.AcaoCancelarReserva:
			var reqCancelamento struct {
				ReservaID string `json:"reserva_id"`
			}
			if err := json.Unmarshal(req.Payload, &reqCancelamento); err != nil {
				enviarResposta(utils.MensagemResposta{Sucesso: false, Mensagem: "Payload de cancelamento inválido."})
				continue
			}

			err := utils.CancelarReserva(reqCancelamento.ReservaID, req.Usuario)
			if err != nil {
				enviarResposta(utils.MensagemResposta{Sucesso: false, Mensagem: err.Error()})
				continue
			}

			enviarResposta(utils.MensagemResposta{
				Sucesso:  true,
				Mensagem: "Reserva cancelada e assentos liberados com sucesso!",
			})

		case utils.AcaoConsultarCaronas:
			caronas, err := utils.ConsultarCaronasMotorista(req.Usuario)
			if err != nil {
				enviarResposta(utils.MensagemResposta{Sucesso: false, Mensagem: "Erro ao buscar caronas do motorista."})
				continue
			}

			payloadBytes, _ := json.Marshal(caronas)
			enviarResposta(utils.MensagemResposta{
				Sucesso:  true,
				Mensagem: fmt.Sprintf("Encontrada(s) %d carona(s) publicada(s).", len(caronas)),
				Payload:  payloadBytes,
			})

		case utils.AcaoCancelarCarona:
			var reqCancelamento struct {
				CaronaID  string `json:"carona_id"`
				Confirmar bool   `json:"confirmar"`
			}

			if err := json.Unmarshal(req.Payload, &reqCancelamento); err != nil {
				enviarResposta(utils.MensagemResposta{Sucesso: false, Mensagem: "Payload de cancelamento inválido."})
				continue
			}

			requerConfirmacao, msg, err := utils.CancelarCaronaMotorista(reqCancelamento.CaronaID, req.Usuario, reqCancelamento.Confirmar)
			if err != nil {
				enviarResposta(utils.MensagemResposta{Sucesso: false, Mensagem: err.Error()})
				continue
			}

			enviarResposta(utils.MensagemResposta{
				Sucesso:  !requerConfirmacao,
				Mensagem: msg,
			})

		case utils.AcaoConsultarPassageirosCarona:
			var reqPassageiros struct {
				CaronaID string `json:"carona_id"`
			}
			json.Unmarshal(req.Payload, &reqPassageiros)

			passageiros, err := utils.ConsultarPassageirosCarona(reqPassageiros.CaronaID, req.Usuario)
			if err != nil {
				enviarResposta(utils.MensagemResposta{Sucesso: false, Mensagem: "Erro ao consultar passageiros."})
				continue
			}

			payloadBytes, _ := json.Marshal(passageiros)
			enviarResposta(utils.MensagemResposta{
				Sucesso:  true,
				Mensagem: fmt.Sprintf("Passageiros na carona %s:", reqPassageiros.CaronaID),
				Payload:  payloadBytes,
			})

		case utils.AcaoAutenticar:
			var reqAuth struct {
				Email string            `json:"email"`
				Senha string            `json:"senha"`
				Tipo  utils.TipoUsuario `json:"tipo"`
			}

			if err := json.Unmarshal(req.Payload, &reqAuth); err != nil {
				enviarResposta(utils.MensagemResposta{Sucesso: false, Mensagem: "Payload de autenticação inválido."})
				continue
			}

			sucesso, msg, err := utils.AutenticarOuCadastrarUsuario(reqAuth.Email, reqAuth.Senha, reqAuth.Tipo)
			if err != nil || !sucesso {
				enviarResposta(utils.MensagemResposta{Sucesso: false, Mensagem: msg})
				continue
			}

			enviarResposta(utils.MensagemResposta{Sucesso: true, Mensagem: msg})
		}
	}
}

func main() {
	ln, err := net.Listen("tcp", ":8080")
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
