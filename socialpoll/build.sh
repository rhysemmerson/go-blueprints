#/bin/sh!

export GOOS=linux 
export GOARCH=386

go build -o ./services/bin/twittervotes ./twittervotes
go build -o ./services/bin/twittercounter ./counter

cd ./services
docker-compose up -d