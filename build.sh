cd ui
bun run build > /dev/null

cd ..
go build -o ./bin/sqlite-gui ./cmd/sqlite-gui/main.go
