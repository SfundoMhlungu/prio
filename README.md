# Install the packages 


```bash
go mod tidy 
go get


```

## Build the tool 


example exe:

```bash 

go build -o myapp.exe ./main.go

```

Put it in the environment variables and use away 


```bash
Usage: cli [add|score|recommend|done]
```

e.g

```bash 
myapp add "task" "description"
myapp score 

```