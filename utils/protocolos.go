package utils

import (
	"encoding/json"
	"time"
)

// Data representa estritamente o dia, mês e ano
type Data struct {
	Dia int `json:"dia"`
	Mes int `json:"mes"`
	Ano int `json:"ano"`
}

// trechos individuais de uma carona, que podem ser reservados individualmente
type Trecho struct {
	Origem         string    `json:"origem"`
	Destino        string    `json:"destino"`
	DataPartida    Data      `json:"data_partida"`    // Aponta exclusivamente para o Dia/Mês/Ano
	HorarioPartida time.Time `json:"horario_partida"` // Define a hora/minuto da saída
	HorarioChegada time.Time `json:"horario_chegada"` // Define a hora/minuto da chegada
	Preco          float64   `json:"preco"`
	AssentosLivre  int       `json:"assentos_livres"`
}

// estrutura que guarda a carona
type Carona struct {
	ID          string    `json:"id"`           // Identificador único da carona
	MotoristaID string    `json:"motorista_id"` // ID / Email do motorista
	AssentosTot int       `json:"assentos_tot"` // Capacidade total do veículo
	Trechos     []Trecho  `json:"trechos"`      // Sequência de trechos da rota
	Ativa       bool      `json:"ativa"`        // Status se a carona está visível
}

// estrutura de reserva
type Reserva struct {
	ID           string    `json:"id"`
	PassageiroID string    `json:"passageiro_id"`
	CaronaID     string    `json:"carona_id"`
	Trechos      []Trecho  `json:"trechos"`
	ValorTotal   float64   `json:"valor_total"`
	DataCriacao  time.Time `json:"data_criacao"`
}

// TipoAcao representa o tipo de ação que o cliente deseja realizar no servidor

type TipoAcao string

const (
	AcaoAutenticar       TipoAcao = "AUTENTICAR"
	AcaoPublicarCarona   TipoAcao = "PUBLICAR_CARONA"
	AcaoConsultarCaronas TipoAcao = "CONSULTAR_CARONAS"
	AcaoCancelarCarona   TipoAcao = "CANCELAR_CARONA"
	AcaoBuscarItinerario TipoAcao = "BUSCAR_ITINERARIO"
	AcaoReservarTrecho   TipoAcao = "RESERVAR_TRECHO"
	AcaoCancelarReserva  TipoAcao = "CANCELAR_RESERVA"
)

// MensagemRequisicao é o envelope de dados enviado pelo Cliente ao Servidor.
type MensagemRequisicao struct {
	Acao    TipoAcao        `json:"acao"`
	Usuario string          `json:"usuario"`
	Payload json.RawMessage `json:"payload"` //se eu fixar uma struct especifica, teria que crair uma struct para cada tipo de payload, então uso RawMessage 
}
// MensagemResposta é o envelope padrão retornado pelo Servidor.
type MensagemResposta struct {
	Sucesso  bool            `json:"sucesso"`
	Mensagem string          `json:"mensagem,omitempty"` //se o campo estiver vazio, não será incluído no JSON
	Payload  json.RawMessage `json:"payload,omitempty"`
}


