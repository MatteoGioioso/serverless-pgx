![GitHub](https://img.shields.io/github/license/MatteoGioioso/serverless-pgx)

# PGX Serverless
Pgx-serverless is a wrapper for **[pgx](https://github.com/jackc/pgx)** go package.
It is heavily inspired by Jeremy Daly's **[serverless-mysql](https://github.com/jeremydaly/serverless-mysql)** Nodejs package.

### Why I need this module?
In a serverless application a function can scale almost "infinitely" by creating separate container instances 
for each concurrent user. 
Each container can correspond to a database connection which, for performance purposes, is left opened for further
re-utilization. If we have a sudden spike of concurrent traffic, the available connections can be quickly maxed out
by other competing functions.
If we reach the max connections limit, Postgres will automatically reject any frontend trying to connect to its backend.
This can cause heavy disruption in your application.

### What does it do?
Pgx-serverless adds a connection management component specifically for FaaS based applications.
By calling the method `.Clean()` at the end of your functions, the module will constantly monitor the status of all
the processes running in the PostgreSQL backend and then, based on the configuration provided, 
will garbage collect the "zombie" connections.
If the client fails to connect with `"sorry, too many clients already"` error, the module will retry
using trusted backoff algorithms.

> **NOTE:** This module *should* work with any PostgreSQL server. 
It has been tested with AWS's RDS Postgres, Aurora Postgres, and Aurora Serverless.

Feel free to request additional features and contribute =)

## Install

```
github.com/MatteoGioioso/serverless-pgx/slsPgx
```

## Usage

Declare the ServerlessClient outside the lambda handler

```go
package main

import (
	"context"
	"fmt"
	"github.com/MatteoGioioso/serverless-pgx/slsPgx"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"os"
)

var (
	serverlessClient = slsPgx.New(slsPgx.SlsConnConfigParams{
		Debug: slsPgx.Bool(true),
	})
	user = os.Getenv("DB_USER")
	password = os.Getenv("DB_PASSWORD")
	host = os.Getenv("DB_HOST")
	db = os.Getenv("DB_NAME")
	connectionString = fmt.Sprintf("postgres://%v:%v@%v:5432/%v?sslmode=disable", user, password, host, db)
)

func function(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if err := serverlessClient.Connect(context.Background(), connectionString); err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	rows, err := serverlessClient.Query(context.Background(), "SELECT 1+1 AS result")
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	for rows.Next() {
		var res int
		if err := rows.Scan(&res); err != nil {
			return events.APIGatewayProxyResponse{}, err
		}
		
		fmt.Println(res)
	}
    
	if _, err := serverlessClient.Clean(context.Background()); err != nil {
		return events.APIGatewayProxyResponse{}, err
	}	

	return events.APIGatewayProxyResponse{
		StatusCode:        200,
		Body:              "Done",
	}, nil
}

func main() {
	lambda.Start(function)
}


```

### Currently under development
