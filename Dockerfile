# Golang image
FROM golang:1.24-alpine

# Ish papkasiga o'tish
WORKDIR /app

# Modullarni yuklash
COPY go.mod ./
RUN go mod tidy

# Appni copy qilish
COPY . ./

# Appni build qilish
RUN go build -o server ./cmd/server/main.go

# Portni expose qilish
EXPOSE 9000

# Appni ishga tushirish
CMD ["./server", "-port=9000"]

