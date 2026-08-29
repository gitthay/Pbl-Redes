package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"

	"vaiJunto/utils"
)

func gerenciarConexao(conexao net.Conn) {
	defer conexao.Close()

	buf := make([]byte, 2048)
	n, err := conexao.Read(buf)
	if err != nil {
		log.Println("Erro na leitura:", err)
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