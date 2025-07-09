# CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o simple_web_tool
CGO_ENABLED=0 go build -ldflags="-s -w" -o simple_web_tool
