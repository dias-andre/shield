echo "Building shield service..."
go build -o target/shldd cmd/shieldd/main.go
echo "Ok!"
echo "Building shield CLI..."
go build -o target/cli cmd/shield/main.go
echo "Ok!"
