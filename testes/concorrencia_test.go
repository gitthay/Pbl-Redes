package testes

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"vaiJunto/utils"
)

const enderecoServidor = "localhost:8080"

// Função auxiliar genérica para enviar requisições TCP e ler respostas
func enviarRequisicao(conn net.Conn, acao utils.TipoAcao, usuario string, payload interface{}) (utils.MensagemResposta, error) {
	var payloadBytes []byte
	if payload != nil {
		payloadBytes, _ = json.Marshal(payload)
	}

	req := utils.MensagemRequisicao{
		Acao:    acao,
		Usuario: usuario,
		Payload: payloadBytes,
	}

	reqBytes, _ := json.Marshal(req)
	if _, err := conn.Write(reqBytes); err != nil {
		return utils.MensagemResposta{}, err
	}

	buf := make([]byte, 8192)
	n, err := conn.Read(buf)
	if err != nil {
		return utils.MensagemResposta{}, err
	}

	var resp utils.MensagemResposta
	err = json.Unmarshal(buf[:n], &resp)
	return resp, err
}


// 1. Teste Unitário: Normalização de Texto e Insensibilidade a Maiúsculas/Espaços
func TestNormalizacaoDeTexto(t *testing.T) {
	entrada := "   Feira DE Santana   "
	esperado := "feira de santana"

	resultado := utils.NormalizarTexto(entrada)
	if resultado != esperado {
		t.Fatalf("Erro na normalização: esperava '%s', obteve '%s'", esperado, resultado)
	}
	t.Log("[UNITÁRIO]: Normalização de texto validada!")
}

// 2. Teste Unitário: Consolidação de Trechos Contínuos na Mesma Carona
func TestConsolidacaoDeTrechos(t *testing.T) {
	trechosOriginais := []utils.ArestaTrecho{
		{CaronaID: "CAR-1", Origem: "Feira de Santana", Destino: "Santo Amaro", Preco: 15.0, AssentosLivre: 3},
		{CaronaID: "CAR-1", Origem: "Santo Amaro", Destino: "Salvador", Preco: 20.0, AssentosLivre: 2},
	}

	consolidados := utils.ConsolidarTrechos(trechosOriginais)

	if len(consolidados) != 1 {
		t.Fatalf("Deveria ter consolidado em 1 único trecho, mas gerou %d", len(consolidados))
	}

	if consolidados[0].Preco != 35.0 || consolidados[0].AssentosLivre != 2 {
		t.Fatalf("A soma de preço ou o menor limite de assentos falhou na consolidação")
	}
	t.Log("[UNITÁRIO]: Consolidação de trechos contínuos validada!")
}

// 3. Teste Unitário: Formatação de Data e Criação Utilitária
func TestEstruturaData(t *testing.T) {
	data := utils.NovaData(15, 10, 2026)
	esperado := "15/10/2026"

	if data.String() != esperado {
		t.Fatalf("Erro ao formatar data: esperava %s, obteve %s", esperado, data.String())
	}
	t.Log("[UNITÁRIO]: Formatação de data validada!")
}


// 4. AcaoAutenticar: Cadastro e Autenticação Simultânea
func TestConcorrenciaAutenticacaoECadastro(t *testing.T) {
	const totalUsuarios = 15
	var wg sync.WaitGroup
	var mu sync.Mutex

	sucessos := 0
	falhas := 0

	wg.Add(totalUsuarios)
	for i := 1; i <= totalUsuarios; i++ {
		email := fmt.Sprintf("user_stress_%d@email.com", i)
		go func(e string) {
			defer wg.Done()
			conn, err := net.Dial("tcp", enderecoServidor)
			if err != nil {
				mu.Lock()
				falhas++
				mu.Unlock()
				return
			}
			defer conn.Close()

			payloadAuth := map[string]interface{}{
				"email": e,
				"senha": "senha_segura_123",
				"tipo":  utils.TipoPassageiro,
			}

			resp, err := enviarRequisicao(conn, utils.AcaoAutenticar, e, payloadAuth)
			mu.Lock()
			if err == nil && resp.Sucesso {
				sucessos++
			} else {
				falhas++
			}
			mu.Unlock()
		}(email)
	}

	wg.Wait()
	t.Logf("[INTEGRAÇÃO]: %d cadastrados com sucesso | %d falhas", sucessos, falhas)
	if sucessos != totalUsuarios {
		t.Fatalf("Erro no cadastro concorrente: esperava %d sucessos, obteve %d", totalUsuarios, sucessos)
	}
}

// 5. AcaoPublicarCarona + AcaoBuscarItinerario: Busca Simultânea no Grafo (BFS)
func TestConcorrenciaBuscaItinerarios(t *testing.T) {
	conn, err := net.Dial("tcp", enderecoServidor)
	if err != nil {
		t.Fatalf("Servidor offline: %v", err)
	}
	defer conn.Close()

	caronaID := fmt.Sprintf("CAR-BUSCA-%d", time.Now().UnixNano())
	c := utils.Carona{
		ID:          caronaID,
		MotoristaID: "m_busca@email.com",
		AssentosTot: 4,
		Ativa:       true,
		Trechos: []utils.Trecho{{
			Origem: "Feira de Santana", Destino: "Salvador",
			DataPartida: utils.NovaData(15, 10, 2026), Preco: 25.0, AssentosLivre: 4,
		}},
	}
	enviarRequisicao(conn, utils.AcaoPublicarCarona, "m_busca@email.com", c)

	var wg sync.WaitGroup
	var mu sync.Mutex
	buscasSucesso := 0

	filtroBusca := map[string]interface{}{
		"origem":  "  FEIRA de santana ",
		"destino": "salvador  ",
		"data":    utils.NovaData(15, 10, 2026),
	}

	wg.Add(20)
	for i := 0; i < 20; i++ {
		go func() {
			defer wg.Done()
			cli, err := net.Dial("tcp", enderecoServidor)
			if err != nil {
				return
			}
			defer cli.Close()

			resp, err := enviarRequisicao(cli, utils.AcaoBuscarItinerario, "passageiro_busca@email.com", filtroBusca)
			if err == nil && resp.Sucesso {
				mu.Lock()
				buscasSucesso++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	t.Logf("[INTEGRAÇÃO]: %d buscas concorrentes no Grafo concluídas", buscasSucesso)
	if buscasSucesso != 20 {
		t.Fatalf("Falha na consulta concorrente ao Grafo: esperava 20, obteve %d", buscasSucesso)
	}
}

// 6. AcaoReservarTrecho: Disparo Concorrente em Itinerário Composto
func TestConcorrenciaItinerarioComposto(t *testing.T) {
	conn, err := net.Dial("tcp", enderecoServidor)
	if err != nil {
		t.Fatalf("Servidor offline: %v", err)
	}
	defer conn.Close()

	carona1ID := fmt.Sprintf("CAR-COMP1-%d", time.Now().UnixNano())
	c1 := utils.Carona{
		ID:          carona1ID,
		MotoristaID: "m1@email.com",
		AssentosTot: 1,
		Ativa:       true,
		Trechos: []utils.Trecho{{
			Origem: "Feira de Santana", Destino: "Santo Amaro",
			DataPartida: utils.NovaData(15, 10, 2026), HorarioPartida: time.Now(), HorarioChegada: time.Now().Add(time.Hour),
			Preco: 15.0, AssentosLivre: 1,
		}},
	}
	enviarRequisicao(conn, utils.AcaoPublicarCarona, "m1@email.com", c1)

	carona2ID := fmt.Sprintf("CAR-COMP2-%d", time.Now().UnixNano())
	c2 := utils.Carona{
		ID:          carona2ID,
		MotoristaID: "m2@email.com",
		AssentosTot: 1,
		Ativa:       true,
		Trechos: []utils.Trecho{{
			Origem: "Santo Amaro", Destino: "Salvador",
			DataPartida: utils.NovaData(15, 10, 2026), HorarioPartida: time.Now().Add(2 * time.Hour), HorarioChegada: time.Now().Add(3 * time.Hour),
			Preco: 20.0, AssentosLivre: 1,
		}},
	}
	enviarRequisicao(conn, utils.AcaoPublicarCarona, "m2@email.com", c2)

	itinerarioComposto := utils.Itinerario{
		PrecoTotal: 35.0,
		Trechos: []utils.ArestaTrecho{
			{CaronaID: carona1ID, Origem: "Feira de Santana", Destino: "Santo Amaro", Preco: 15.0},
			{CaronaID: carona2ID, Origem: "Santo Amaro", Destino: "Salvador", Preco: 20.0},
		},
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	reservasAprovadas := 0

	wg.Add(10)
	for i := 1; i <= 10; i++ {
		email := fmt.Sprintf("p_comp_%d@email.com", i)
		go func(e string) {
			defer wg.Done()
			c, err := net.Dial("tcp", enderecoServidor)
			if err != nil {
				return
			}
			defer c.Close()

			resp, err := enviarRequisicao(c, utils.AcaoReservarTrecho, e, itinerarioComposto)
			if err == nil && resp.Sucesso {
				mu.Lock()
				reservasAprovadas++
				mu.Unlock()
			}
		}(email)
	}

	wg.Wait()
	t.Logf("[INTEGRAÇÃO]: %d aprovado(s) para itinerário atômico", reservasAprovadas)
	if reservasAprovadas != 1 {
		t.Fatalf("Erro na atomicidade do itinerário: esperava 1, obteve %d", reservasAprovadas)
	}
}

// 7. AcaoListarReservas + AcaoCancelarReserva: Gestão do Passageiro
func TestConcorrenciaListarECancelarReserva(t *testing.T) {
	conn, err := net.Dial("tcp", enderecoServidor)
	if err != nil {
		t.Fatalf("Servidor offline: %v", err)
	}
	defer conn.Close()

	caronaID := fmt.Sprintf("CAR-GEST-%d", time.Now().UnixNano())
	c := utils.Carona{
		ID:          caronaID,
		MotoristaID: "m_gestao@email.com",
		AssentosTot: 2,
		Ativa:       true,
		Trechos: []utils.Trecho{{
			Origem: "Feira de Santana", Destino: "Valença",
			DataPartida: utils.NovaData(15, 10, 2026), Preco: 40.0, AssentosLivre: 2,
		}},
	}
	enviarRequisicao(conn, utils.AcaoPublicarCarona, "m_gestao@email.com", c)

	itin := utils.Itinerario{
		PrecoTotal: 40.0,
		Trechos:    []utils.ArestaTrecho{{CaronaID: caronaID, Origem: "Feira de Santana", Destino: "Valença"}},
	}

	respRes, _ := enviarRequisicao(conn, utils.AcaoReservarTrecho, "passageiro_gestao@email.com", itin)
	var resObj utils.Reserva
	json.Unmarshal(respRes.Payload, &resObj)

	respLista, err := enviarRequisicao(conn, utils.AcaoListarReservas, "passageiro_gestao@email.com", nil)
	if err != nil || !respLista.Sucesso {
		t.Fatalf("Falha ao listar reservas do passageiro")
	}

	payloadCanc := map[string]string{"reserva_id": resObj.ID}
	respCanc, err := enviarRequisicao(conn, utils.AcaoCancelarReserva, "passageiro_gestao@email.com", payloadCanc)
	if err != nil || !respCanc.Sucesso {
		t.Fatalf("Falha ao cancelar reserva: %s", respCanc.Mensagem)
	}

	t.Log("[INTEGRAÇÃO]: Listagem e cancelamento de reservas validados")
}

// 8. AcaoConsultarCaronas + AcaoConsultarPassageirosCarona
func TestConcorrenciaConsultasMotorista(t *testing.T) {
	conn, err := net.Dial("tcp", enderecoServidor)
	if err != nil {
		t.Fatalf("Servidor offline: %v", err)
	}
	defer conn.Close()

	emailMotorista := "motorista_consultas@email.com"
	caronaID := fmt.Sprintf("CAR-CONS-%d", time.Now().UnixNano())

	c := utils.Carona{
		ID:          caronaID,
		MotoristaID: emailMotorista,
		AssentosTot: 3,
		Ativa:       true,
		Trechos: []utils.Trecho{{
			Origem: "Feira de Santana", Destino: "Ilhéus",
			DataPartida: utils.NovaData(15, 10, 2026), Preco: 80.0, AssentosLivre: 3,
		}},
	}
	enviarRequisicao(conn, utils.AcaoPublicarCarona, emailMotorista, c)

	itin := utils.Itinerario{
		PrecoTotal: 80.0,
		Trechos:    []utils.ArestaTrecho{{CaronaID: caronaID, Origem: "Feira de Santana", Destino: "Ilhéus"}},
	}
	enviarRequisicao(conn, utils.AcaoReservarTrecho, "p_ilheus@email.com", itin)

	respConsult, _ := enviarRequisicao(conn, utils.AcaoConsultarCaronas, emailMotorista, nil)
	if !respConsult.Sucesso {
		t.Fatalf("Motorista falhou ao consultar suas caronas")
	}

	respPass, _ := enviarRequisicao(conn, utils.AcaoConsultarPassageirosCarona, emailMotorista, map[string]string{"carona_id": caronaID})
	if !respPass.Sucesso {
		t.Fatalf("Motorista falhou ao consultar passageiros da carona")
	}

	t.Log("[INTEGRAÇÃO]: Consultas exclusivas do motorista validadas!")
}

// 9. AcaoCancelarCarona: Cancelamento pelo Motorista com Confirmação Expressa
func TestCancelarCaronaMotorista(t *testing.T) {
	conn, err := net.Dial("tcp", enderecoServidor)
	if err != nil {
		t.Fatalf("Servidor offline: %v", err)
	}
	defer conn.Close()

	emailMotorista := "motorista_cancela@email.com"
	caronaID := fmt.Sprintf("CAR-CANC-MOT-%d", time.Now().UnixNano())

	c := utils.Carona{
		ID:          caronaID,
		MotoristaID: emailMotorista,
		AssentosTot: 2,
		Ativa:       true,
		Trechos: []utils.Trecho{{
			Origem: "Feira de Santana", Destino: "Itabuna",
			DataPartida: utils.NovaData(15, 10, 2026), Preco: 60.0, AssentosLivre: 2,
		}},
	}
	enviarRequisicao(conn, utils.AcaoPublicarCarona, emailMotorista, c)

	payloadCanc := map[string]interface{}{"carona_id": caronaID, "confirmar": true}
	respCancMot, _ := enviarRequisicao(conn, utils.AcaoCancelarCarona, emailMotorista, payloadCanc)
	if !respCancMot.Sucesso {
		t.Fatalf("Falha ao cancelar carona pelo motorista: %s", respCancMot.Mensagem)
	}

	t.Log("[INTEGRAÇÃO]: Cancelamento de carona pelo motorista validado!")
}

// 10. Encerramento Inesperado do Socket TCP (EOF)
func TestQuedaAbruptaDeCliente(t *testing.T) {
	for i := 0; i < 5; i++ {
		conn, err := net.Dial("tcp", enderecoServidor)
		if err == nil {
			conn.Close() // Fecha o socket abruptamente sem protocolo
		}
	}

	connOk, err := net.Dial("tcp", enderecoServidor)
	if err != nil {
		t.Fatalf("Servidor caiu após quedas abruptas de conexão!")
	}
	defer connOk.Close()

	resp, err := enviarRequisicao(connOk, utils.AcaoConsultarCaronas, "teste_resiliencia@email.com", nil)
	if err != nil || !resp.Sucesso {
		t.Fatalf("Servidor deixou de responder requisições após falhas nos sockets")
	}

	t.Log("O servidor tratou os erros de socket (EOF) e permanece online!")
}
