# Use a imagem base oficial do Go
FROM golang:1.19-alpine

# Instalar curl e outras dependências necessárias
RUN apk add --no-cache curl gcc musl-dev

# Configurar o diretório de trabalho dentro do container
WORKDIR /app

# Copiar os arquivos go.mod e go.sum e baixar as dependências
COPY go.mod go.sum ./
RUN go mod download

# Copiar o restante dos arquivos da aplicação
COPY . .

# Construir o binário da aplicação
RUN go build -o main .

# Tornar o script de inicialização executável
RUN chmod +x scripts/start.sh

# Definir o entrypoint
ENTRYPOINT ["./scripts/start.sh"]
