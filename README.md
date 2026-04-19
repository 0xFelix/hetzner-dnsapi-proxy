# hetzner-dnsapi-proxy

hetzner-dnsapi-proxy proxies DNS API update requests to the [Hetzner Cloud API](https://docs.hetzner.cloud).

> **Note:** Support for the old Hetzner DNS API has been removed since it has
> been [shut down](https://docs.hetzner.com/networking/dns/faq/beta/#timeline).
> If upgrading from a setup that used the old DNS API, update your `API_TOKEN`
> (or `token` in the config file) to a Hetzner Cloud API token. The `cloudAPI`
> config option and `CLOUD_API` environment variable are no longer recognized
> and can be removed from existing configurations.

## Container image

Get the container image from [ghcr.io](https://github.com/0xFelix/hetzner-dnsapi-proxy/pkgs/container/hetzner-dnsapi-proxy)

## Supported DNS APIs

| API                | Endpoint                                                                                                                                                                                                                                                                                                                                                           |
|--------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| lego HTTP request  | POST `/httpreq/present`<br>POST `/httpreq/cleanup`<br>(see https://go-acme.github.io/lego/dns/httpreq/)                                                                                                                                  |
| ACMEDNS            | POST `/acmedns/update`<br>(see https://github.com/joohoi/acme-dns#update-endpoint)                                                                                                                                                                                 |
| DirectAdmin Legacy | GET `/directadmin/CMD_API_SHOW_DOMAINS`<br>GET `/directadmin/CMD_API_DNS_CONTROL` (only adding A/AAAA/TXT records, everything else always returns `200 OK`)<br>GET `/directadmin/CMD_API_DOMAIN_POINTER` (only a stub, always returns `200 OK`)<br>(see https://docs.directadmin.com/developer/api/legacy-api.html and https://www.directadmin.com/features.php?id=504) |
| plain HTTP         | GET `/plain/update` (query params `hostname` and `ip` (can be ipv4 for A or ipv6 for AAAA records), if auth method is `users` then HTTP Basic auth is used) <br/>                                                                                                                                                                                                               |
| DynDNS2            | GET `/nic/update` (query params `hostname` and optional `myip` (falls back to client IP, ipv4 or ipv6), HTTP Basic auth, responses follow the DynDNS2 token spec)                                                                                                                                                                                                             |

## Configuration

Configuration can be passed by environment variables or from a file (with 
the `-c` flag).

> **Security notes:**
> - The server speaks plaintext HTTP only. Terminate TLS in front of it
>   (e.g. with a reverse proxy) whenever it is exposed beyond a trusted
>   network - credentials and update values would otherwise travel in clear
>   text.
> - The config file holds the Hetzner API token and, optionally, user
>   passwords. Restrict it to the service account (e.g. `chmod 600`) and
>   keep it out of version control and container images.

### Authorization

Authorization takes place via a list of domains and ip networks allowed
to update them or from a list of users. Both can be provided in a config
file while when parsing the configuration from environment variables only
the former is supported.

The supported authorization methods are:
- `allowedDomains`: Define ip networks allowed to update specific domains or 
  subdomains
- `users`: Define users allowed to update specific domains or subdomains
- `both`: Combination of `allowedDomains` and `users`, **both** must be
  satisfied
- `any`: Combination of `allowedDomains` and `users`, **any** of the two must
  be satisfied

To authorize a domain and all of its subdomains, prefix the entry with `*.`
(for example `*.example.com` matches `example.com`'s subdomains like
`foo.example.com` and `bar.foo.example.com`). A bare `example.com` entry only
authorizes that exact name - subdomains will be rejected.

> **Note:** The `/nic/update` endpoint follows the DynDNS2 response spec and
> returns `200 OK` with a `nohost` token on authorization failure in
> `allowedDomains` mode (a `401 badauth` is only returned when HTTP Basic auth
> is actively being used). The `lockout` feature still applies so repeated
> `nohost` responses from the same client IP eventually trigger a lockout.

> **Note:** Caller-supplied IPs (`myip` on `/nic/update`, `ip` on
> `/plain/update`, JSON `value` on `/httpreq/*` and `/acmedns/update`) are
> taken from the request at face value. They are only as trustworthy as the
> authenticated client submitting them - there is no server-side verification
> that the value actually belongs to the caller.

### Rate limiting and auth-failure lockout

Both features per-client-IP defenses:

- `rateLimit` is a token-bucket throttle applied to every endpoint. Requests
  above `burst` refill at `rps` tokens per second. Excess requests get
  HTTP 429 (or the DynDNS2 `abuse` token on `/nic/update`).
- `lockout` tracks consecutive auth failures. After `maxAttempts` failures
  within `windowSeconds`, the client IP is locked out for `durationSeconds`.
  A successful auth clears the counter. Partial failures outside the window
  are forgotten.

Client IPs are determined after `trustedProxies` resolution, so requests
traversing a trusted reverse proxy are counted against the real client.

### Enabled endpoints

By default all endpoint groups are enabled. You can restrict which groups are
active by listing only the ones you want:

- `plain` — `/plain/update`
- `nic` — `/nic/update`
- `acmedns` — `/acmedns/update`
- `httpreq` — `/httpreq/present`, `/httpreq/cleanup`
- `directadmin` — `/directadmin/CMD_API_*`

Via config file set the `endpoints` key; via environment variable set
`ENDPOINTS` to a comma-separated list (e.g. `ENDPOINTS=plain,nic`). Listing
any endpoint disables all others not listed.

### Security headers

Every response includes `X-Content-Type-Options: nosniff`,
`X-Frame-Options: DENY`, `Content-Security-Policy: default-src 'none'`, and
`Cache-Control: no-store`.

### Configuration file

```yaml
token: verysecrettoken
timeout: 60
auth:
  method: both
  allowedDomains:
    example.com:
      - ip: 127.0.0.1
        mask:
          - 255
          - 255
          - 255
          - 255
  users:
    - username: user
      password: pass
      domains:
        - example.com
endpoints:
  plain: true
  nic: true
  acmedns: true
  httpreq: true
  directadmin: true
recordTTL: 60
listenAddr: :8081
trustedProxies:
  - 127.0.0.1
rateLimit:
  rps: 5
  burst: 10
  idleSeconds: 600
lockout:
  maxAttempts: 10
  durationSeconds: 3600
  windowSeconds: 900
debug: false
```

### Environment variables

| Variable                   | Type   | Description                                                                                                                                | Required | Default                        |
|:---------------------------|--------|--------------------------------------------------------------------------------------------------------------------------------------------|----------|--------------------------------|
| `API_BASE_URL`             | string | Base URL of the API                                                                                                                        | N        | `https://api.hetzner.cloud/v1` |
| `API_TOKEN`                | string | Auth token for the API                                                                                                                     | Y        |                                |
| `API_TIMEOUT`              | int    | Timeout for calls to the API in seconds                                                                                                    | N        | 15 seconds                     |
| `RECORD_TTL`               | int    | TTL that is set when creating/updating records                                                                                             | N        | 60 seconds                     |
| `ALLOWED_DOMAINS`          | string | Combination of domains and CIDRs allowed to update them, example:<br>`example1.com,127.0.0.1/32;_acme-challenge.example2.com,127.0.0.1/32` | Y        |                                |
| `LISTEN_ADDR`              | string | Listen address of hetzner-dnsapi-proxy                                                                                                     | N        | `:8081`                        |
| `TRUSTED_PROXIES`          | string | Comma-separated list of trusted proxy IPs or CIDR ranges (e.g. `10.0.0.1,192.168.0.0/24`). When empty, `X-Real-Ip` / `X-Forwarded-For` are ignored. | N        | Trust no proxies               |
| `RATE_LIMIT_RPS`           | float  | Tokens per second refilled per client IP                                                                                                   | N        | `5`                            |
| `RATE_LIMIT_BURST`         | int    | Maximum burst size per client IP                                                                                                           | N        | `10`                           |
| `RATE_LIMIT_IDLE_SECONDS`  | int    | Seconds of inactivity before a client's rate limit bucket is removed                                                                       | N        | `600`                          |
| `LOCKOUT_MAX_ATTEMPTS`     | int    | Failures before lockout                                                                                                                    | N        | `10`                           |
| `LOCKOUT_DURATION_SECONDS` | int    | Lockout duration in seconds                                                                                                                | N        | `3600`                         |
| `LOCKOUT_WINDOW_SECONDS`   | int    | Window in seconds during which consecutive failures accumulate                                                                             | N        | `900`                          |
| `ENDPOINTS`                | string | Comma-separated list of endpoint groups to enable: `plain`, `nic`, `acmedns`, `httpreq`, `directadmin`. All enabled when unset.            | N        | All enabled                    |
| `DEBUG`                    | bool   | Output debug logs of received requests                                                                                                     | N        | `false`                        |
