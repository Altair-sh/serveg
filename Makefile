build:
	go build -trimpath -ldflags="-s -w" -o bin/ ./cmd/serveg
	ls -lh bin

run: build
	cd bin && ./serveg

clean:
	rm -rf bin
