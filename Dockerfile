# Usa a imagem oficial do Go para compilar
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copia o gerenciador de módulos do Go
COPY go.mod ./
RUN go mod download

# Copia todo o seu código para dentro do contêiner
COPY . .

# Compila o seu servidor
RUN CGO_ENABLED=0 GOOS=linux go build -o server_bin main.go

# Cria a imagem final leve para rodar o sistema
FROM alpine:latest
WORKDIR /app

# Copia o servidor compilado
COPY --from=builder /app/server_bin .

# Libera a porta 8080 do servidor TCP
EXPOSE 8080

# Executa o servidor quando o contêiner ligar
CMD ["./server_bin"]