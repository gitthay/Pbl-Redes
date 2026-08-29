package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// String formata a data para leitura (ex: "25/12/2026")
func (d Data) String() string {
	return fmt.Sprintf("%02d/%02d/%04d", d.Dia, d.Mes, d.Ano)
}

// NovaData é uma função utilitária para criar o tipo a partir de inteiros
func NovaData(dia, mes, ano int) Data {
	return Data{Dia: dia, Mes: mes, Ano: ano}
}

// Aresta representa um trecho de carona disponível na rede
type ArestaTrecho struct {
	CaronaID       string
	MotoristaID    string
	Origem         string
	Destino        string
	HorarioPartida time.Time
	HorarioChegada time.Time
	Preco          float64
	AssentosLivre  int
}

// Representa um itinerário completo sugerido ao passageiro (uma lista sequencial de trechos)
type Itinerario struct {
	Trechos    []ArestaTrecho `json:"trechos"`
	PrecoTotal float64        `json:"preco_total"`
}

// Grafo de Cidades (Adjacência: CidadeOrigem -> Lista de Trechos Saindo dela)
type GrafoCidades map[string][]ArestaTrecho

// MontarGrafo constrói o grafo a partir das caronas cadastradas no JSON
func MontarGrafo(caronas []Carona, dataDesejada Data) GrafoCidades {
	grafo := make(GrafoCidades)

	for _, c := range caronas {
		if !c.Ativa {
			continue
		}
		for _, t := range c.Trechos {
			// Filtra pela data e assentos disponíveis
			if t.DataPartida == dataDesejada && t.AssentosLivre > 0 {
				aresta := ArestaTrecho{
					CaronaID:       c.ID,
					MotoristaID:    c.MotoristaID,
					Origem:         t.Origem,
					Destino:        t.Destino,
					HorarioPartida: t.HorarioPartida,
					HorarioChegada: t.HorarioChegada,
					Preco:          t.Preco,
					AssentosLivre:  t.AssentosLivre,
				}
				grafo[t.Origem] = append(grafo[t.Origem], aresta)
			}
		}
	}
	return grafo
}

// BuscarItinerarios realiza uma BFS no grafo para encontrar rotas diretas ou com troca de carona
func BuscarItinerarios(caronas []Carona, origem, destino string, data Data) []Itinerario {
	grafo := MontarGrafo(caronas, data)
	var resultados []Itinerario

	// Fila para algoritmo BFS: cada elemento é um caminho de trechos acumulados
	fila := [][]ArestaTrecho{}

	// Inicializa a fila com as arestas que saem da origem
	for _, aresta := range grafo[origem] {
		fila = append(fila, []ArestaTrecho{aresta})
	}

	for len(fila) > 0 {
		caminhoAtual := fila[0]
		fila = fila[1:]

		ultimoTrecho := caminhoAtual[len(caminhoAtual)-1]

		if ultimoTrecho.Destino == destino {
			var precoTotal float64
			for _, t := range caminhoAtual {
				precoTotal += t.Preco
			}

			// Funde os trechos contínuos do mesmo motorista antes de retornar
			caminhoConsolidado := ConsolidarTrechos(caminhoAtual)

			resultados = append(resultados, Itinerario{
				Trechos:    caminhoConsolidado,
				PrecoTotal: precoTotal,
			})
			continue
		}

		// Procura próximas conexões saindo da cidade onde o último trecho desembarcou
		proximasArestas := grafo[ultimoTrecho.Destino]
		for _, proxima := range proximasArestas {
			// Se for a mesma carona, permite a continuidade do percurso.
			// Se forem caronas diferentes, exige que a partida seja posterior à chegada anterior[cite: 1].
			mesmaCarona := proxima.CaronaID == ultimoTrecho.CaronaID
			horarioValido := proxima.HorarioPartida.After(ultimoTrecho.HorarioChegada) || proxima.HorarioPartida.Equal(ultimoTrecho.HorarioChegada)

			if mesmaCarona || horarioValido {
				novoCaminho := append([]ArestaTrecho{}, caminhoAtual...)
				novoCaminho = append(novoCaminho, proxima)
				fila = append(fila, novoCaminho)
			}
		}
	}

	return resultados
}

// ConsolidarTrechos funde segmentos contínuos da mesma carona em um único trecho
func ConsolidarTrechos(trechos []ArestaTrecho) []ArestaTrecho {
	if len(trechos) <= 1 {
		return trechos
	}

	var consolidados []ArestaTrecho
	atual := trechos[0]

	for i := 1; i < len(trechos); i++ {
		proximo := trechos[i]

		// Se for a mesma carona e houver continuidade direta de rota (Destino anterior = Origem do próximo)
		if atual.CaronaID == proximo.CaronaID && atual.Destino == proximo.Origem {
			atual.Destino = proximo.Destino               // Estende o destino para o extremo final
			atual.HorarioChegada = proximo.HorarioChegada // Ajusta a chegada para o horário do último trecho
			atual.Preco += proximo.Preco                  // Soma o valor total da viagem contínua

			// Mantém o menor número de assentos disponíveis no percurso acumulado
			if proximo.AssentosLivre < atual.AssentosLivre {
				atual.AssentosLivre = proximo.AssentosLivre
			}
		} else {
			// Troca de carona/motorista: salva o trecho acumulado e inicia um novo
			consolidados = append(consolidados, atual)
			atual = proximo
		}
	}

	// Adiciona o último segmento processado
	consolidados = append(consolidados, atual)
	return consolidados
}

// persistencia de caronas
var (
	ArquivoCaronas = "caronas.json"
	mu             sync.Mutex // Evita conflito de concorrência entre goroutines do servidor
)

// CarregarCaronas lê todas as caronas salvas no arquivo JSON
func CarregarCaronas() ([]Carona, error) {
	// Se o arquivo não existir, retorna um slice vazio em vez de dar erro
	if _, err := os.Stat(ArquivoCaronas); os.IsNotExist(err) {
		return []Carona{}, nil
	}

	dados, err := os.ReadFile(ArquivoCaronas)
	if err != nil {
		return nil, err
	}

	if len(dados) == 0 {
		return []Carona{}, nil
	}

	var caronas []Carona
	err = json.Unmarshal(dados, &caronas)
	if err != nil {
		return nil, err
	}

	return caronas, nil
}

// SalvarCarona adiciona uma nova carona ao arquivo JSON existente
func SalvarCarona(novaCarona Carona) error {
	mu.Lock()
	defer mu.Unlock() // Libera a trava do arquivo assim que a função terminar

	// 1. Carrega as caronas já existentes
	caronas, err := CarregarCaronas()
	if err != nil {
		return err
	}

	// 2. Adiciona a nova carona à lista
	caronas = append(caronas, novaCarona)

	// 3. Converte a lista completa de volta para JSON
	dadosFormatados, err := json.MarshalIndent(caronas, "", "  ")
	if err != nil {
		return err
	}

	// 4. Salva os dados no arquivo
	return os.WriteFile(ArquivoCaronas, dadosFormatados, 0644)
}
