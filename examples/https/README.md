# HTTPS example

Serves `GET /hello` in one of three modes, selected with the `MODE` environment variable.

## Create a local certificate

```sh
openssl req -x509 -newkey rsa:2048 -nodes -keyout tls.key -out tls.crt -days 30 \
  -subj "/CN=localhost" -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
```

`*.crt` and `*.key` are ignored by git; never commit key material.

## Run

| `MODE`     | Listeners                                                 |
|------------|-----------------------------------------------------------|
| `both`     | plain on `10000`, TLS on `10443` (default)                |
| `https`    | TLS on `8443` only                                        |
| `redirect` | plain on `10000` redirects to TLS on `10443`, probes excluded |

```sh
MODE=both go run .
MODE=https go run .
MODE=redirect go run .
```

The certificate paths default to `tls.crt` and `tls.key` in the working directory; override them with
`TLS_CERT` and `TLS_KEY`. Requests are in [https.http](https.http); with curl use `-k` for the
self-signed certificate.
