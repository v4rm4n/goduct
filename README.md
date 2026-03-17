# goduct

> SSH tunnels without the TTY pain.

A dead-simple tunneling tool that speaks native SSH — no agent needed when `sshd` is already running.

## Why?

`ssh -N -f -L 8080:localhost:80 user@host` is annoying to type, dies silently, and hijacks your terminal.  
`goduct` fixes that by using an intuitive connector syntax (`-f-` and `-r-`). Single binary, clean CLI, no TTY.

## Install

```bash
git clone [https://github.com/v4rm4n/goduct](https://github.com/v4rm4n/goduct)
cd goduct
go build .
```

## Usage

**Syntax:** `goduct [source] [-f- or -r-] [destination] [user@host]`

**Note:** You can use IP addresses (10.0.1.5:80), hostnames (localhost:8080), or network interface names (eth0:80, Wi-Fi:3000) for the endpoints.

**Forward (-f-)** — bring a remote service to your machine (ssh -L)
Listen on your local machine, and route traffic through the SSH server to the destination.
```
# Listen locally on 8080, forward to the jumphost's own port 80
goduct localhost:8080 -f- localhost:80 root@jumphost

# Listen locally on all interfaces, reach a private DB only the jumphost can see
goduct 0.0.0.0:5432 -f- db.internal:5432 root@jumphost

# Bind to a specific network interface
goduct eth0:8080 -f- 10.0.0.5:80 root@jumphost
```

```
Your machine                         Jumphost (SSH server)
─────────────────                    ─────────────────────────────
curl localhost:8080
      ↓
net.Listen(:8080) accepts
      ↓
client.Dial("db.internal:5432") →    SSH decrypts channel request
                                             ↓
                                      server dials db.internal:5432
                                      (resolved from jumphost's side)
                                             ↓
                                      db responds
      ↓                    ←──────────────────
pipe() copies bytes both ways
```

**Reverse (-r-**) — expose a local service on the remote server (ssh -R)
Ask the remote SSH server to listen on a port, and route traffic back to a destination from your machine.

```
# Expose your local dev server on the VPS 
goduct 0.0.0.0:9090 -r- localhost:3000 root@vps

# Expose a LAN machine through the VPS
goduct 8080 -r- 192.168.1.50:3000 root@vps
```

```
Your machine                         VPS (SSH server)
─────────────────                    ─────────────────────────────
                                     client.Listen("0.0.0.0:9090")
                       ←─────────── goduct asks SSH server to
                                     open THIS port on the VPS
                                             ↓
                                     someone hits vps:9090
                                             ↓
                       ←─────────── SSH sends connection back
      ↓
net.Dial("192.168.1.50:3000")
(resolved from your machine's side)
      ↓
pipe() copies bytes both ways
      ↓                    ────────────────────→ response flows back
```


## Key insight:
**-f- (Forward)** → destination is resolved by the SSH server

**-r- (Reverse)** → destination is resolved by your machine

## Authentication

goduct tries auth methods in this order — no flags needed in the common case:

| Priority | Method | How |
|---|---|---|
| 1 | Explicit key | `--key ~/.ssh/my_key` |
| 2 | SSH agent | automatic if `$SSH_AUTH_SOCK` is set |
| 3 | Default keys | tries `~/.ssh/id_ed25519`, `id_rsa`, `id_ecdsa` |
| 4 | Password | Secure interactive prompt (hidden input) |

```bash
# Uses agent or default keys automatically (prompts for password if they fail)
goduct localhost:8080 -f- localhost:80 root@host

# Explicit key
goduct localhost:8080 -f- localhost:80 root@host --key ~/Downloads/my-key
```

Roadmap
[x] CLI skeleton (cobra)

[x] Spec parsing and interface resolution (eth0:80, etc.)

[x] Intuitive connector syntax (-f- and -r-)

[x] SSH key auth (explicit key, agent, default paths, secure password prompt)

[x] SSH forward (ssh -L)

[x] SSH reverse (ssh -R)

[ ] Auto-reconnect on drop

[ ] HTTP tunnel fallback (chisel-style, for when port 22 is blocked)