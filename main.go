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