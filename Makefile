.PHONY: clean

clean:
	docker-compose stop redis
	docker-compose rm -f redis
	docker-compose scale redis=9
	docker ps -q --filter 'name=redis' | xargs docker inspect --format '{{ .NetworkSettings.IPAddress  }}' | tr "\n" " "

crosscompile:
	GOARCH="amd64" GOOS="linux" go build redis-cluster.go
