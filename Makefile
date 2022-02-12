http-proxy.linux:
	env GOOS=linux GOARCH=amd64 go build -o http-proxy.linux src/*.go

http-proxy.mac:
	env GOOS=darwin GOARCH=amd64 go build -o http-proxy.mac src/*.go

confignginx.linux:
	env GOOS=linux GOARCH=amd64 go build -o confignginx.linux src/confignginx/*.go

clean:
	rm http-proxy.linux

deploy:
	make http-proxy.linux
	rsync -avz http-proxy.linux root@157.230.234.171:/home/ptn/http-proxy/
