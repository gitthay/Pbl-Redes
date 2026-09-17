# VaiJunto 🚗💨

Sistema de caronas compartilhadas para viagens de média e longa distância, desenvolvido para a disciplina **TEC502 — Concorrência e Conectividade**. O projeto implementa um servidor central em Go que gerencia caronas e reservas via **sockets TCP puros** (sem frameworks de RPC/mensageria), com clientes de terminal para motoristas e passageiros.

## Contexto

Motoristas que já farão determinado trajeto publicam os assentos livres do veículo, e passageiros que precisam se deslocar entre as mesmas cidades reservam esses assentos, dividindo o custo da viagem. A rota é definida pelo motorista, e o sistema é responsável por descobrir e coordenar automaticamente os encontros entre a oferta (caronas publicadas) e a demanda (buscas dos passageiros).

## Principais funcionalidades

- **Autenticação/cadastro automático**: usuário se autentica com email/senha; se não existir, é cadastrado no primeiro acesso.
- **Bloqueio de sessão simultânea**: um mesmo email não pode estar autenticado em duas conexões ao mesmo tempo.
- **Publicação de caronas** com múltiplos trechos (rota composta por várias cidades), data, horários, preço e assentos por trecho.
- **Controle de disponibilidade por trecho**: um assento ocupado entre a cidade A e B continua livre para os trechos seguintes da mesma carona.
- **Busca de itinerários via BFS em grafo**: o servidor monta um grafo de cidades a partir das caronas ativas e busca caminhos que atendam origem/destino/data, combinando trechos de **motoristas diferentes** quando necessário.
- **Reserva atômica de itinerário**: todos os trechos escolhidos são confirmados, ou nenhum é — protegido por *mutex* para evitar duplicidade de assento sob concorrência.
- **Cancelamento de reserva**, com devolução dos assentos ocupados.
- **Cancelamento de carona pelo motorista**, com aviso e confirmação explícita caso existam passageiros já reservados.
- **Consulta de caronas e passageiros** por motorista.
- **Log de auditoria** (`servidor.log`) de conexões, requisições e respostas.

## Arquitetura

```
┌────────────────┐        ┌────────────────┐
│ Cliente         │        │ Cliente         │
│ Motorista       │        │ Passageiro      │
│ (mainmotorista) │        │ (mainpassageiro)│
└────────┬────────┘        └────────┬────────┘
         │        TCP / JSON        │
         └────────────┬─────────────┘
                       │
              ┌────────▼─────────┐
              │  Servidor TCP     │
              │  (main.go)        │
              │  - 1 goroutine/   │
              │    conexão        │
              └────────┬─────────┘
                       │
              ┌────────▼─────────┐
              │  Pacote utils     │
              │  - protocolos.go  │
              │  - servicos.go    │
              │  (grafo, reservas,│
              │   persistência)   │
              └────────┬─────────┘
                       │
        caronas.json · reservas.json · usuarios.json
```

- **`main.go`**: servidor TCP. Aceita conexões, dispara uma goroutine por cliente (`gerenciarConexao`) e roteia as ações do protocolo.
- **`mainmotorista.go`** / **`mainpassageiro.go`**: clientes de terminal (CLI) que se conectam ao servidor via `net.Dial("tcp", ...)`, com endereço configurável pela flag `-server`.
- **`protocolos.go`**: definição do protocolo de aplicação — tipos de mensagem (`MensagemRequisicao` / `MensagemResposta`), ações (`TipoAcao`) e entidades de domínio (`Carona`, `Trecho`, `Reserva`, `Usuario`, `Data`).
- **`servicos.go`**: regras de negócio — montagem do grafo de cidades, busca de itinerários (BFS), consolidação de trechos contínuos da mesma carona, persistência em JSON e controle de concorrência (`sync.Mutex`).
- **`concorrencia_test.go`**: suíte de testes automatizados (unitários + integração/estresse contra o servidor real).

## Protocolo de aplicação

Comunicação via **TCP bruto**, com mensagens serializadas em **JSON**. Toda requisição segue o envelope:

```json
{
  "acao": "BUSCAR_ITINERARIO",
  "usuario": "email@exemplo.com",
  "payload": { "...": "..." }
}
```

E toda resposta:

```json
{
  "sucesso": true,
  "mensagem": "Encontrado(s) 2 itinerário(s) disponível(is).",
  "payload": { "...": "..." }
}
```

### Ações suportadas (`TipoAcao`)

| Ação | Descrição |
|---|---|
| `AUTENTICAR` | Autentica o usuário ou cadastra automaticamente no primeiro acesso |
| `PUBLICAR_CARONA` | Motorista publica uma nova carona com um ou mais trechos |
| `CONSULTAR_CARONAS` | Lista as caronas publicadas por um motorista |
| `CANCELAR_CARONA` | Cancela uma carona (com confirmação se houver passageiros) |
| `BUSCAR_ITINERARIO` | Passageiro busca itinerários entre origem/destino/data |
| `RESERVAR_TRECHO` | Confirma a reserva atômica de um itinerário |
| `LISTAR_RESERVAS` | Lista as reservas ativas de um passageiro |
| `CANCELAR_RESERVA` | Cancela uma reserva e libera os assentos |
| `CONSULTAR_PASSAGEIROS_CARONA` | Lista os passageiros confirmados em uma carona |

Mensagens malformadas (JSON inválido) são detectadas e descartadas pelo servidor, que encerra a conexão do remetente sem afetar os demais clientes.

## Controle de concorrência

- Um `sync.Mutex` global protege as operações de leitura/escrita dos arquivos `caronas.json` e `reservas.json`, garantindo que a validação e a confirmação de um itinerário aconteçam de forma atômica.
- O passageiro que confirmar a reserva primeiro mantém a preferência sobre os assentos disputados; os demais recebem erro e podem tentar outro itinerário.
- Um `map` protegido por mutex (`usuariosLogados`) impede login simultâneo do mesmo email em mais de uma conexão, liberando a sessão automaticamente se o cliente cair (`io.EOF` ou erro de leitura).
- Não é usado nenhum banco de dados ou serviço externo de coordenação — a exclusão mútua é feita inteiramente pela aplicação.

## Como executar

### Pré-requisitos
- Go 1.22+
- Docker e Docker Compose (para rodar o servidor em contêiner)

### Subindo o servidor com Docker

```bash
docker compose up --build
```

O servidor sobe na porta `8080` e persiste `caronas.json`, `reservas.json`, `usuarios.json` e `servidor.log` na raiz do projeto (via volumes).

### Rodando o servidor localmente (sem Docker)

```bash
go run main.go
```

### Rodando os clientes

```bash
# Cliente motorista
go run ./motorista -server localhost:8080

# Cliente passageiro
go run ./passageiro -server localhost:8080
```

A flag `-server` permite apontar para o IP de outra máquina na rede (útil para testar servidor e clientes em computadores distintos do laboratório).

### Rodando os testes automatizados

Com o servidor já em execução (Docker ou `go run main.go`):

```bash
go test ./testes/... -v -args -server localhost:8080
```

A suíte cobre:
- **Testes unitários**: normalização de texto, consolidação de trechos, formatação de data.
- **Testes de concorrência/integração** contra o servidor real: cadastro/autenticação simultânea de múltiplos usuários, buscas concorrentes de itinerários, reserva de itinerários compostos por caronas de motoristas diferentes, disputa pelo mesmo assento (garantindo que nenhum trecho é vendido duas vezes), cancelamento de carona com passageiros, bloqueio de login duplicado, queda abrupta de cliente e envio de mensagens malformadas.
- **Teste de desempenho sob carga** (TestDesempenhoTempoResposta): dispara 50 requisições concorrentes de busca de itinerário, cada uma em sua própria conexão TCP, medindo o tempo de resposta (round-trip) individual de cada uma e reportando o tempo médio, mínimo e máximo observados.

## Estrutura de arquivos

```
pbl_redes/
├── motorista/
│   └── mainmotorista.go      # Cliente CLI do motorista
├── passageiro/
│   └── mainpassageiro.go     # Cliente CLI do passageiro
├── testes/
│   └── concorrencia_test.go   # Testes automatizados
├── utils/
│   ├── protocolos.go          # Protocolo e entidades de domínio
│   └── servicos.go            # Regras de negócio, grafo/BFS e persistência
├── main.go                    # Servidor TCP
├── go.mod
├── Dockerfile
├── docker-compose.yml
├── README.md
├── caronas.json                # gerado em tempo de execução
├── reservas.json                # gerado em tempo de execução
├── usuarios.json                # gerado em tempo de execução
└── servidor.log                 # gerado em tempo de execução
```

## Restrições atendidas

- Comunicação implementada com a **API socket TCP nativa**, sem frameworks de RPC ou mensageria.
- Estado do sistema mantido em um **único servidor central**, sem réplicas.
- **Controle de concorrência implementado na aplicação**, sem delegar a SGBD ou serviço externo.
- Servidor tolerante à queda abrupta de clientes, sem corromper o estado das reservas.
- Mensagens trocadas em **JSON**, com validação e descarte de payloads malformados pelo receptor.
- Backend containerizado via **Docker**.

## Autor

Trabalho desenvolvido por Thaylane da Silva para a disciplina TEC502 — Problema 1: VaiJunto, Caronas Compartilhadas.
