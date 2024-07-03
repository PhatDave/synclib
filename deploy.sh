GOOS=windows GOARCH=amd64 go build -o main.exe main && cp main.exe "/c/Program Files/Git/usr/bin/cln.exe"
GOOS=linux GOARCH=amd64 go build -o main_linux main