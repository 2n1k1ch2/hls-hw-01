FROM golang:1.27.2

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /dashboard-generator ./cmd/dashboard-generator

CMD ["/dashboard-generator"]
