package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"vaiJunto/utils"
)

func main() {
	conexao, err := net.Dial("tcp", "localhost:8080") //cria um socket individual para o cliente
	if err != nil {
		log.Fatalln(err)
	}
	defer conexao.Close()

	novaCarona := utils.Carona{
		ID:          "1",
		MotoristaID: "motorista@email.com",
		AssentosTot: 4,
		Ativa:       true,
		Trechos: []utils.Trecho{
			{
				Origem:         "Feira de Santana",
				Destino:        "Salvador",
				DataPartida:    utils.NovaData(15, 10, 2026),
				HorarioPartida: time.Now(),
				HorarioChegada: time.Now().Add(2 * time.Hour),
				Preco:          35.0,
				AssentosLivre:  4,
			},
		},
	}

	payloadBytes, err := json.Marshal(novaCarona)
	if err != nil {
		log.Println("Erro ao converter payload:", err)
		return
	}

	req := utils.MensagemRequisicao{
		Acao:    utils.AcaoPublicarCarona,
		Usuario: "motorista@email.com",
		Payload: payloadBytes,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		log.Println("Erro ao converter requisição:", err)
		return
	}

	conexao.Write(reqBytes)

	buf := make([]byte, 1024)
	n, err := conexao.Read(buf)
	if err != nil {
		log.Println("Erro ao ler resposta:", err)
		return
	}

	fmt.Println("Resposta do servidor:", string(buf[:n]))
}