# learn-pub-sub-starter (Peril)

This is the starter code used in Boot.dev's [Learn Pub/Sub](https://learn.boot.dev/learn-pub-sub) course.

## Tests (Not part of the course)
Currently tests available for
- Generating gamelog entry by generating 3 terminals (1 server 2 client) then instigating a war between them to create a new entry
```
2025-04-23T12:33:19+03:00 washington: napoleon won against washington
```

## Running commands
To run rabbitMQ:
```
./rabbit.sh start
```
To run the server:
```
go build ./cmd/server && ./server
```
To run the client:
```
go build ./cmd/client && ./client
```
To run tests:
```
go test
```

