package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"

	"vaiJunto/utils"
)

func main() {
	conexao, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		log.Fatalln("Erro ao conectar ao servidor:", err)
	}
	defer conexao.Close()

	filtroBusca := struct {
		Origem  string     `json:"origem"`
		Destino string     `json:"destino"`
		Data    utils.Data `json:"data"`
	}{
		Origem:  "Feira de Santana",
		Destino: "Paris",
		Data:    utils.NovaData(15, 10, 2026),
	}

	payloadBytes, _ := json.Marshal(filtroBusca)

	req := utils.MensagemRequisicao{
		Acao:    utils.AcaoBuscarItinerario,
		Usuario: "passageiro@email.com",
		Payload: payloadBytes,
	}

	reqBytes, _ := json.Marshal(req)
	conexao.Write(reqBytes)

	buf := make([]byte, 4096)
	n, err := conexao.Read(buf)
	if err != nil {
		log.Fatalln("Erro ao ler resposta:", err)
	}

	var resp utils.MensagemResposta
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		log.Fatalln("Erro ao decodificar envelope de resposta:", err)
	}

	if !resp.Sucesso {
		fmt.Println("Erro do servidor:", resp.Mensagem)
		return
	}

	// 1. Decodifica o Payload bruto para o tipo []utils.Itinerario
	var itinerarios []utils.Itinerario
	if err := json.Unmarshal(resp.Payload, &itinerarios); err != nil {
		log.Fatalln("Erro ao processar lista de itinerários:", err)
	}

	// 2. Formata a exibição amigável no terminal
	fmt.Println("\n==================================================")
	fmt.Printf("   %s\n", resp.Mensagem)
	fmt.Println("==================================================")

	for i, itinerario := range itinerarios {
		fmt.Printf("\nOPÇÃO %d - Preço Total: R$ %.2f\n", i+1, itinerario.PrecoTotal)
		fmt.Println("--------------------------------------------------")

		for j, trecho := range itinerario.Trechos {
			fmt.Printf("  Trecho %d: %s -> %s\n", j+1, trecho.Origem, trecho.Destino)
			fmt.Printf("    * Motorista: %s (Carona ID: %s)\n", trecho.MotoristaID, trecho.CaronaID)
			fmt.Printf("    * Horários:  Saída %s | Chegada %s\n",
				trecho.HorarioPartida.Format("15:04"),
				trecho.HorarioChegada.Format("15:04"))
			fmt.Printf("    * Preço:     R$ %.2f (%d assentos livres)\n", trecho.Preco, trecho.AssentosLivre)
		}
	}
	fmt.Println("\n==================================================")
}
