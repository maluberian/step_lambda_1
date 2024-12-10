# Description
Just a simple lambda meant to play with step-functions
a little bit.

# Deploy
```shell
aws lambda update-function-code --function-name transaction-step-2 --architectures amd64 --zip-file fileb://pkg.zip --region us-east-2 --profile AdministratorAccess-724772095124
```

