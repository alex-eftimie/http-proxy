http-proxy.linux:
	env GOOS=linux GOARCH=amd64 go build -o http-proxy.linux src/*.go

clean:
	rm http-proxy.linux

deploy:
	make http-proxy.linux
	rsync -avz http-proxy.linux root@157.230.234.171:/home/ptn/http-proxy/
