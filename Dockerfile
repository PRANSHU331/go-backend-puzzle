FROM golang:1.21-alpine

WORKDIR /app

# we can COPY go modues here
COPY go.mod go.sum ./
RUN go mod download

# this will Copy all the sourse code here 
COPY . .

# run this command to build the app 
RUN go build -o app ./cmd/server

# this the Default end point
CMD ["./app"]
