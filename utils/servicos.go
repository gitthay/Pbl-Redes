package utils

import (
	"fmt"
	"encoding/json"
	"os"
	"sync"
)

// String formata a data para leitura (ex: "25/12/2026")
func (d Data) String() string {
	return fmt.Sprintf("%02d/%02d/%04d", d.Dia, d.Mes, d.Ano)
}

// NovaData é uma função utilitária para criar o tipo a partir de inteiros
func NovaData(dia, mes, ano int) Data {
	return Data{Dia: dia, Mes: mes, Ano: ano}
}

//persistencia de caronas
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