```
./goduct forward 8080:localhost:80 --via root@129.212.244.172 --key ~/Downloads/some-ssh-key
```

**What's happening inside, step by step:**
```
You curl localhost:8080
      ↓
net.Listen() accepts the connection
      ↓
client.Dial("tcp", "localhost:80")   ← SSH asks server to open channel
      ↓
SSH encrypted tunnel (port 22)
      ↓
Server dials localhost:80 on its side
      ↓
pipe() — goroutines copy bytes both ways
```