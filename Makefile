all: build

run-local: build
	AWS_LAMBDA_RUNTIME_API=localhost:8080 ./bin/aws-lambda-rie ./bootstrap

pkg: build
	zip pkg.zip bootstrap

build: main.go
	GOOS=linux GOARCH=amd64 go build -o bootstrap --buildvcs=false

clean:
	rm -f main pkg.zip