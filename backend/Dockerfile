FROM golang:alpine

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go mod tidy
RUN go build -o /cinema-backend
EXPOSE 8080
CMD [ "/cinema-backend" ]