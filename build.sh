echo "Building shield service..."
go build -o target/shldd ./cmd/shieldd
echo "Ok!"
echo "Building shield CLI..."
go build -o target/cli ./cmd/shield
echo "Ok!"
