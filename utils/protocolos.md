# Protocolo de Aplicação — VaiJunto

Este documento especifica o protocolo de aplicação usado na comunicação entre os clientes (motorista e passageiro) e o servidor central do VaiJunto. A comunicação ocorre sobre **sockets TCP nativos**, sem uso de frameworks de RPC ou mensageria. Os dados trafegam como **objetos JSON** enviados diretamente sobre o stream TCP.

## 1. Transporte e conexão

- O servidor escuta em `0.0.0.0:8080` (configurável via `docker-compose.yml` / execução local).
- O cliente estabelece uma conexão TCP (`net.Dial("tcp", <host>:<porta>)`) e a mantém **aberta durante toda a sessão do usuário**, enviando uma mensagem por vez e aguardando a resposta antes de enviar a próxima (modelo requisição-resposta síncrono, uma goroutine por conexão no servidor).
- Não há handshake de aplicação: a primeira mensagem enviada pelo cliente já é uma requisição de `AUTENTICAR`.

### 1.1 Fluxo de conexão

```
Cliente                                   Servidor
   |------------- TCP connect ------------->|
   |                                         | goroutine dedicada criada
   |------ AUTENTICAR {email, senha} ------->|
   |                                         | valida login simultâneo (bloqueia
   |                                         | se o e-mail já estiver conectado)
   |                                         | autentica ou cadastra o usuário
   |<----------- MensagemResposta -----------|
   |                                         |
   |---- (demais ações do menu) ----------->|
   |<----------- MensagemResposta -----------|
   |                 ...                     |
```

### 1.2 Fluxo de desconexão

- **Encerramento normal**: o cliente fecha a conexão (`conexao.Close()`) ao escolher a opção "Sair" do menu. O servidor detecta `io.EOF` na leitura, libera a sessão do usuário (`RemoverLogin`) e finaliza a goroutine.
- **Queda abrupta**: se a conexão cair sem aviso (crash do cliente, rede interrompida), a leitura no servidor retorna erro; o `defer` da goroutine libera a sessão do usuário da mesma forma, garantindo que o e-mail fique disponível para novo login e que nenhum arquivo fique em estado inconsistente (as escritas em JSON só ocorrem dentro de operações protegidas por mutex e já finalizadas).
- **Mensagem malformada**: se o payload recebido não for um JSON válido, o servidor registra o erro no log e encerra a conexão daquele cliente. Os demais clientes conectados não são afetados.

## 2. Envelope das mensagens

Toda requisição do cliente para o servidor segue o formato:

```json
{
  "acao": "NOME_DA_ACAO",
  "usuario": "email@exemplo.com",
  "payload": { }
}
```

| Campo | Tipo | Descrição |
|---|---|---|
| `acao` | string | Identifica a operação desejada (ver tabela da seção 3) |
| `usuario` | string | E-mail do usuário autenticado que originou a requisição |
| `payload` | objeto (JSON bruto) | Dados específicos da operação; formato varia por `acao` |

Toda resposta do servidor para o cliente segue o formato:

```json
{
  "sucesso": true,
  "mensagem": "Texto descritivo do resultado",
  "payload": { }
}
```

| Campo | Tipo | Descrição |
|---|---|---|
| `sucesso` | bool | Indica se a operação foi concluída com êxito |
| `mensagem` | string | Mensagem legível para exibição ao usuário (omitido se vazio) |
| `payload` | objeto (JSON bruto) | Dados de retorno da operação, quando aplicável (omitido se vazio) |

## 3. Ações do protocolo

| Ação | Quem envia | Requer autenticação prévia |
|---|---|---|
| `AUTENTICAR` | Motorista / Passageiro | Não (é a própria autenticação) |
| `PUBLICAR_CARONA` | Motorista | Sim |
| `CONSULTAR_CARONAS` | Motorista | Sim |
| `CANCELAR_CARONA` | Motorista | Sim |
| `CONSULTAR_PASSAGEIROS_CARONA` | Motorista | Sim |
| `BUSCAR_ITINERARIO` | Passageiro | Sim |
| `RESERVAR_TRECHO` | Passageiro | Sim |
| `LISTAR_RESERVAS` | Passageiro | Sim |
| `CANCELAR_RESERVA` | Passageiro | Sim |

> Observação: o protocolo atual não impõe um token de sessão — o campo `usuario` (e-mail) identifica o solicitante em cada mensagem, e o controle de sessão ativa é feito no servidor via mapa `usuariosLogados`, associado à conexão TCP.

---

### 3.1 `AUTENTICAR`

Autentica um usuário existente ou o cadastra automaticamente no primeiro acesso. Bloqueia se o e-mail já tiver uma sessão ativa em outra conexão.

**Payload da requisição**

| Campo | Tipo | Descrição |
|---|---|---|
| `email` | string | E-mail do usuário |
| `senha` | string | Senha em texto plano (prototipagem acadêmica) |
| `tipo` | string | `"MOTORISTA"` ou `"PASSAGEIRO"` |

**Exemplo — requisição**
```json
{
  "acao": "AUTENTICAR",
  "usuario": "carlos@email.com",
  "payload": {
    "email": "carlos@email.com",
    "senha": "123456",
    "tipo": "MOTORISTA"
  }
}
```

**Exemplo — resposta (sucesso, novo cadastro)**
```json
{
  "sucesso": true,
  "mensagem": "Usuário cadastrado e autenticado com sucesso!"
}
```

**Exemplo — resposta (erro, sessão já ativa)**
```json
{
  "sucesso": false,
  "mensagem": "Acesso negado: Este usuário já está conectado em outra sessão."
}
```

---

### 3.2 `PUBLICAR_CARONA`

O motorista publica uma nova carona com um ou mais trechos.

**Payload da requisição** (estrutura `Carona`)

| Campo | Tipo | Descrição |
|---|---|---|
| `id` | string | Gerado pelo cliente (`CAR-<timestamp>`) |
| `motorista_id` | string | E-mail do motorista |
| `assentos_tot` | int | Capacidade total do veículo |
| `ativa` | bool | Sempre `true` na publicação |
| `trechos` | array de `Trecho` | Sequência ordenada de trechos da rota |

Cada `Trecho` contém: `origem`, `destino`, `data_partida` (`{dia, mes, ano}`), `horario_partida`, `horario_chegada` (RFC 3339), `preco`, `assentos_livres`.

**Exemplo — requisição**
```json
{
  "acao": "PUBLICAR_CARONA",
  "usuario": "carlos@email.com",
  "payload": {
    "id": "CAR-1737049200000000000",
    "motorista_id": "carlos@email.com",
    "assentos_tot": 3,
    "ativa": true,
    "trechos": [
      {
        "origem": "Salvador",
        "destino": "Feira de Santana",
        "data_partida": { "dia": 15, "mes": 10, "ano": 2026 },
        "horario_partida": "2026-10-15T08:00:00Z",
        "horario_chegada": "2026-10-15T09:30:00Z",
        "preco": 25.0,
        "assentos_livres": 3
      },
      {
        "origem": "Feira de Santana",
        "destino": "Vitória da Conquista",
        "data_partida": { "dia": 15, "mes": 10, "ano": 2026 },
        "horario_partida": "2026-10-15T09:30:00Z",
        "horario_chegada": "2026-10-15T13:00:00Z",
        "preco": 60.0,
        "assentos_livres": 3
      }
    ]
  }
}
```

**Exemplo — resposta**
```json
{
  "sucesso": true,
  "mensagem": "Carona cadastrada com sucesso!"
}
```

---

### 3.3 `CONSULTAR_CARONAS`

Lista todas as caronas (ativas e canceladas) publicadas pelo motorista autenticado. Não exige payload — o campo `usuario` já identifica o motorista.

**Exemplo — requisição**
```json
{
  "acao": "CONSULTAR_CARONAS",
  "usuario": "carlos@email.com"
}
```

**Exemplo — resposta**
```json
{
  "sucesso": true,
  "mensagem": "Encontrada(s) 1 carona(s) publicada(s).",
  "payload": [
    {
      "id": "CAR-1737049200000000000",
      "motorista_id": "carlos@email.com",
      "assentos_tot": 3,
      "ativa": true,
      "trechos": [ /* ... */ ]
    }
  ]
}
```

---

### 3.4 `CANCELAR_CARONA`

Cancela uma carona publicada. Se houver passageiros com reserva confirmada, o servidor não cancela de imediato — devolve um aviso e exige confirmação explícita numa segunda mensagem com `confirmar: true`.

**Payload da requisição**

| Campo | Tipo | Descrição |
|---|---|---|
| `carona_id` | string | ID da carona a cancelar |
| `confirmar` | bool | `false` na primeira tentativa; `true` para confirmar apesar dos passageiros afetados |

**Exemplo — requisição (primeira tentativa)**
```json
{
  "acao": "CANCELAR_CARONA",
  "usuario": "carlos@email.com",
  "payload": { "carona_id": "CAR-1737049200000000000", "confirmar": false }
}
```

**Exemplo — resposta (exige confirmação)**
```json
{
  "sucesso": false,
  "mensagem": "ATENÇÃO: Esta carona possui 2 passageiro(s) com reserva confirmada! Deseja realmente cancelar?"
}
```

**Exemplo — requisição (confirmando)**
```json
{
  "acao": "CANCELAR_CARONA",
  "usuario": "carlos@email.com",
  "payload": { "carona_id": "CAR-1737049200000000000", "confirmar": true }
}
```

**Exemplo — resposta (cancelamento efetivado)**
```json
{
  "sucesso": true,
  "mensagem": "Carona cancelada com sucesso!"
}
```

> Nota de implementação: o servidor devolve `sucesso: false` mesmo quando a operação foi tecnicamente aceita, mas está pendente de confirmação — o cliente usa esse sinal (`resp.Mensagem` contendo `"ATENÇÃO"`) para decidir se pergunta a confirmação ao usuário.

---

### 3.5 `CONSULTAR_PASSAGEIROS_CARONA`

Lista os e-mails dos passageiros com reserva confirmada em algum trecho da carona.

**Payload da requisição**

| Campo | Tipo | Descrição |
|---|---|---|
| `carona_id` | string | ID da carona consultada |

**Exemplo — requisição**
```json
{
  "acao": "CONSULTAR_PASSAGEIROS_CARONA",
  "usuario": "carlos@email.com",
  "payload": { "carona_id": "CAR-1737049200000000000" }
}
```

**Exemplo — resposta**
```json
{
  "sucesso": true,
  "mensagem": "Passageiros na carona CAR-1737049200000000000:",
  "payload": ["ana@email.com", "bruno@email.com"]
}
```

---

### 3.6 `BUSCAR_ITINERARIO`

O passageiro busca itinerários entre uma origem e um destino em uma data específica. O servidor monta um grafo de cidades a partir das caronas ativas com assento disponível e retorna todos os caminhos encontrados via busca em largura (BFS), incluindo itinerários compostos por caronas de motoristas diferentes.

**Payload da requisição**

| Campo | Tipo | Descrição |
|---|---|---|
| `origem` | string | Cidade de origem (comparação sem distinção de maiúsculas/espaços) |
| `destino` | string | Cidade de destino |
| `data` | objeto `Data` | `{ "dia": int, "mes": int, "ano": int }` |

**Exemplo — requisição**
```json
{
  "acao": "BUSCAR_ITINERARIO",
  "usuario": "ana@email.com",
  "payload": {
    "origem": "Salvador",
    "destino": "Vitória da Conquista",
    "data": { "dia": 15, "mes": 10, "ano": 2026 }
  }
}
```

**Exemplo — resposta**
```json
{
  "sucesso": true,
  "mensagem": "Encontrado(s) 1 itinerário(s) disponível(is).",
  "payload": [
    {
      "preco_total": 85.0,
      "trechos": [
        {
          "carona_id": "CAR-1737049200000000000",
          "motorista_id": "carlos@email.com",
          "origem": "Salvador",
          "destino": "Vitória da Conquista",
          "horario_partida": "2026-10-15T08:00:00Z",
          "horario_chegada": "2026-10-15T13:00:00Z",
          "preco": 85.0,
          "assentos_livres": 3
        }
      ]
    }
  ]
}
```

> Quando o itinerário combina caronas de motoristas diferentes, o array `trechos` contém mais de um elemento, cada um com seu próprio `carona_id` e `motorista_id`.

---

### 3.7 `RESERVAR_TRECHO`

Confirma a reserva de um itinerário completo (um ou mais trechos) escolhido pelo passageiro a partir do resultado de `BUSCAR_ITINERARIO`. A confirmação é **atômica**: todos os trechos são reservados ou nenhum é.

**Payload da requisição** (estrutura `Itinerario`, geralmente reenviando exatamente o item escolhido do resultado da busca)

| Campo | Tipo | Descrição |
|---|---|---|
| `trechos` | array de `ArestaTrecho` | Trechos que compõem o itinerário desejado |
| `preco_total` | float | Soma dos preços dos trechos |

**Exemplo — requisição**
```json
{
  "acao": "RESERVAR_TRECHO",
  "usuario": "ana@email.com",
  "payload": {
    "preco_total": 85.0,
    "trechos": [
      {
        "carona_id": "CAR-1737049200000000000",
        "motorista_id": "carlos@email.com",
        "origem": "Salvador",
        "destino": "Vitória da Conquista",
        "horario_partida": "2026-10-15T08:00:00Z",
        "horario_chegada": "2026-10-15T13:00:00Z",
        "preco": 85.0,
        "assentos_livres": 3
      }
    ]
  }
}
```

**Exemplo — resposta (sucesso)**
```json
{
  "sucesso": true,
  "mensagem": "Reserva confirmada com sucesso!",
  "payload": {
    "id": "RES-1737049300000000000",
    "passageiro_id": "ana@email.com",
    "itinerario": [ /* mesmos trechos da requisição */ ],
    "valor_total": 85.0,
    "data_criacao": "2026-10-10T18:22:00Z"
  }
}
```

**Exemplo — resposta (falha, assento esgotado entre a busca e a confirmação)**
```json
{
  "sucesso": false,
  "mensagem": "reserva cancelada: o trecho de Salvador para Vitória da Conquista não possui mais assentos disponíveis"
}
```

---

### 3.8 `LISTAR_RESERVAS`

Lista as reservas confirmadas do passageiro autenticado. Não exige payload.

**Exemplo — requisição**
```json
{
  "acao": "LISTAR_RESERVAS",
  "usuario": "ana@email.com"
}
```

**Exemplo — resposta**
```json
{
  "sucesso": true,
  "mensagem": "Encontrada(s) 1 reserva(s).",
  "payload": [
    {
      "id": "RES-1737049300000000000",
      "passageiro_id": "ana@email.com",
      "itinerario": [ /* ... */ ],
      "valor_total": 85.0,
      "data_criacao": "2026-10-10T18:22:00Z"
    }
  ]
}
```

---

### 3.9 `CANCELAR_RESERVA`

Cancela uma reserva do passageiro e devolve os assentos ocupados aos respectivos trechos.

**Payload da requisição**

| Campo | Tipo | Descrição |
|---|---|---|
| `reserva_id` | string | ID da reserva a cancelar |

**Exemplo — requisição**
```json
{
  "acao": "CANCELAR_RESERVA",
  "usuario": "ana@email.com",
  "payload": { "reserva_id": "RES-1737049300000000000" }
}
```

**Exemplo — resposta**
```json
{
  "sucesso": true,
  "mensagem": "Reserva cancelada e assentos liberados com sucesso!"
}
```

**Exemplo — resposta (erro, reserva de outro usuário)**
```json
{
  "sucesso": false,
  "mensagem": "você não tem permissão para cancelar esta reserva"
}
```

## 4. Tratamento de erros e mensagens malformadas

- Todo `payload` é `json.RawMessage` no envelope, decodificado apenas dentro do tratamento específico de cada `acao`. Isso permite que o servidor sempre consiga interpretar o envelope externo mesmo que o conteúdo interno mude de formato por ação.
- Se o JSON do envelope inteiro for inválido, a mensagem é descartada, o evento é registrado em `servidor.log` e a conexão daquele cliente é encerrada — sem impacto nas demais conexões ativas.
- Se o `payload` de uma ação específica for inválido (ex.: campo com tipo incompatível), o servidor responde com `sucesso: false` e uma mensagem de erro apropriada, mantendo a conexão aberta para novas tentativas (ex.: `RESERVAR_TRECHO`, `CANCELAR_CARONA`).

## 5. Log de auditoria

Todas as conexões, requisições e respostas trafegadas são registradas em `servidor.log`, no formato:

```
[2026-10-10 18:22:00] [REQUISICAO] De: 127.0.0.1:52344 | Usuario: ana@email.com | Acao: RESERVAR_TRECHO | Payload: {...}
[2026-10-10 18:22:00] [RESPOSTA] Para: 127.0.0.1:52344 (Usuario: ana@email.com) | Acao: RESERVAR_TRECHO | Sucesso: true | Msg: Reserva confirmada com sucesso! | Payload: {...}
```