# goduct

> SSH tunnels without the TTY pain.

A dead-simple tunneling tool that speaks native SSH — no agent needed when `sshd` is already running.

## Why?

`ssh -N -f -L 8080:localhost:80 user@host` is annoying to type, dies silently, and hijacks your terminal.  
`goduct` fixes that. Single binary, clean flags, no TTY.

## Install

```bash
git clone https://github.com/v4rm4n/goduct
cd goduct
go build .
```

## Usage

### Forward — bring a remote service to your machine (`ssh -L`)

```bash
goduct forward [localPort]:[remoteHost]:[remotePort] --via user@host
```

```bash
# Reach a web server on the remote host
goduct forward 8080:localhost:80 --via root@jumphost

# Reach a private DB only the jumphost can see
goduct forward 5432:db.internal:5432 --via root@jumphost
```

```
Your machine                     Jumphost (SSH server)
─────────────────                ─────────────────────────────
curl localhost:8080
      ↓
net.Listen(:8080)
accepts connection
      ↓
client.Dial("tcp",
  "db.internal:5432")  ────────→ SSH decrypts channel request
      ↓                                    ↓
SSH encrypted tunnel                server dials db.internal:5432
    (port 22)                        (resolved from jumphost's network!)
      ↑                                    ↓
pipe() copies                       db responds
bytes both ways  ←──────────────────────────
```

### Reverse — expose a local service on the remote server (`ssh -R`)

```bash
goduct reverse [remotePort]:[localHost]:[localPort] --via user@host
```

```bash
# Expose your local dev server on the VPS
goduct reverse 9090:localhost:3000 --via root@vps

# Expose a LAN machine through the VPS
goduct reverse 8080:192.168.1.50:3000 --via root@vps
```

```
Your machine                     VPS (SSH server)
─────────────────                ─────────────────────────────
                                 client.Listen("0.0.0.0:9090")
                                 ← goduct asks SSH server to
                                   open THIS port on the VPS
                                        ↓
                                 someone hits vps:9090
                                        ↓
                         ←────── SSH sends connection back
                                 through the tunnel
      ↓
handleReverse()
net.Dial("192.168.1.50:3000")
← your machine dials this
  (from YOUR network!)
      ↓
pipe() copies
bytes both ways  ──────────────────────────→ response flows back
```

> **Key insight:**
> `forward` → remote host is resolved by the **SSH server**  
> `reverse` → local host is resolved by **your machine**

## Authentication

goduct tries auth methods in this order — no flags needed in the common case:

| Priority | Method | How |
|---|---|---|
| 1 | Explicit key | `--key ~/.ssh/my_key` |
| 2 | SSH agent | automatic if `$SSH_AUTH_SOCK` is set |
| 3 | Default keys | tries `~/.ssh/id_ed25519`, `id_rsa`, `id_ecdsa` |
| 4 | Password | `--password s3cr3t` (last resort) |

```bash
# Uses agent or default keys automatically
goduct forward 8080:localhost:80 --via root@host

# Explicit key
goduct forward 8080:localhost:80 --via root@host --key ~/Downloads/my-key

# Password fallback
goduct forward 8080:localhost:80 --via root@host --password s3cr3t
```

## Roadmap

- [x] CLI skeleton (cobra)
- [x] Spec parsing (`port:host:port`)
- [x] SSH key auth (explicit key, agent, default paths, password)
- [x] SSH forward (`ssh -L`)
- [x] SSH reverse (`ssh -R`)
- [ ] Auto-reconnect on drop
- [ ] `--bind` flag for reverse (control which interface the remote binds on)
- [ ] HTTP tunnel fallback (chisel-style, for when port 22 is blocked)