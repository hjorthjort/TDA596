Accessing EC2 instance:

```sh
ssh -i ~/.ssh/lab1-key.pem ec2-user@54.144.165.163
```

Testing:
```
curl -i http://54.144.165.163/index.html
```

# How to run locally

## Using Go

```sh
/Users/hjort/sdk/go1.27.3/bin/go build
/Users/hjort/sdk/go1.27.3/bin/go test
LOG_LEVEL=debug ./http_server 8080
```

Specifying port 0 assign a free port.

## Using Docker

The Docker image always starts the http server on its internal port 8080, so we
must map to it.

```sh
docker build -t lab1:latest .
docker stop lab1
docker run --rm -d \
  --name lab1 \
  -p 127.0.0.1:8080:8080 \ # <ip>:<port on host>:<port in docker image>
  -e LOG_LEVEL=debug \
  lab1:latest
```

NOTE: `http_server` crashes if the port chosen is already in use. However if it
is in use by Docker, the local binary shadows it.

# How to run in the cloud

Go to the following address to start the AWS Lab. This brings your instance
back online.

https://awsacademy.instructure.com/courses/195325/modules/items/19333881

To deploy the latest changes, commit all your work and run

```sh
./run.sh
```

Go to the IP address of the instance, path /index.html

```sh
open http://$IP_ADDRESS/index.html
```

From there you can try subcribing, unsubcribing and publishing. You can also try
uploading and downloading files.
