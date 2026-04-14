# Build stage: Compilação do Microkernel com suporte a CGO
FROM golang:1.26.1-bookworm AS builder

# Instalar dependências de compilação C/C++ (necessárias para USearch e TurboQuant)
RUN apt-get update && apt-get install -y \
    g++ \
    make \
    cmake \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copiar arquivos de dependência
COPY go.mod go.sum ./
RUN go mod download

# Copiar o código fonte e os headers do USearch
COPY . .

# Compilar o binário principal para ambiente Linux
# Ativando CGO para suporte à biblioteca vetorial USearch
RUN CGO_CFLAGS="-I/app/internal/storage/db" \
    GOOS=linux GOARCH=amd64 CGO_ENABLED=1 \
    go build -mod=vendor -ldflags="-s -w -extldflags '-static'" -o vectora ./cmd/core

# Final stage: Imagem de produção leve
FROM debian:bookworm-slim

# Instalar certificados CA para conexões HTTPS com provedores LLM (Gemini, Anthropic, etc.)
RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /root/

# Copiar o executável do builder
COPY --from=builder /app/vectora .

# Criar diretório para persistência de dados
RUN mkdir -p /root/.config/vectora

# Expor a porta da API Cloud-Native (Phase 4)
EXPOSE 8080

# Comando padrão: inicia o daemon
ENTRYPOINT ["./vectora"]
CMD ["core", "--detached=false"]
