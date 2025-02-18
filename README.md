# hetzner-dnsapi-proxy

hetzner-dnsapi-proxy proxies DNS API update requests to the [Hetzner](https://dns.hetzner.com/api-docs) DNS API.

## Container image

Get the container image from [ghcr.io](https://github.com/0xFelix/hetzner-dnsapi-proxy/pkgs/container/hetzner-dnsapi-proxy)

## Supported DNS APIs

| API                | Endpoint                                                                                                                                                                                                                                                                                                                                                           |
|--------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| lego HTTP request  | POST `/httpreq/present`<br>POST `/httpreq/cleanup` (always returns `200 OK`)<br>(see https://go-acme.github.io/lego/dns/httpreq/)                                                                                                                                                                                                                                  |
| ACMEDNS            | POST `/acmedns/update`<br>(see https://github.com/joohoi/acme-dns#update-endpoint)                                                                                                                                                                                                                                                                                 |
| DirectAdmin Legacy | GET `/directadmin/CMD_API_SHOW_DOMAINS`<br>GET `/directadmin/CMD_API_DNS_CONTROL` (only adding A/TXT records, everything else always returns `200 OK`)<br>GET `/directadmin/CMD_API_DOMAIN_POINTER` (only a stub, always returns `200 OK`)<br>(see https://docs.directadmin.com/developer/api/legacy-api.html and https://www.directadmin.com/features.php?id=504) |
| plain HTTP         | GET `/plain/update` (query params `hostname` and `ip`, if auth method is `users` also `username` and `password`) <br/>                                                                                                                                                                                                                                               |

## Configuration

Configuration can be passed by environment variables or from a file (with 
the `-c` flag).

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

### Configuration file

```yaml
token: verysecrettoken
timeout: 15
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
recordTTL: 60
listenAddr: :8081
trustedProxies:
  - 127.0.0.1
debug: false
```

### Environment variables

| Variable          | Type   | Description                                                                                                                                | Required | Default                          |
|:------------------|--------|--------------------------------------------------------------------------------------------------------------------------------------------|----------|----------------------------------|
| `API_BASE_URL`    | string | Base URL of the DNS API                                                                                                                    | n        | `https://dns.hetzner.com/api/v1` |
| `API_TOKEN`       | string | Auth token for the API                                                                                                                     | Y        |                                  |
| `API_TIMEOUT`     | int    | Timeout for calls to the API in seconds                                                                                                    | N        | 15 seconds                       |
| `RECORD_TTL`      | int    | TTL that is set when creating/updating records                                                                                             | N        | 60 seconds                       |
| `ALLOWED_DOMAINS` | string | Combination of domains and CIDRs allowed to update them, example:<br>`example1.com,127.0.0.1/32;_acme-challenge.example2.com,127.0.0.1/32` | Y        |                                  |
| `LISTEN_ADDR`     | string | Listen address of hetzner-dnsapi-proxy                                                                                                     | N        | `:8081`                          |
| `TRUSTED_PROXIES` | string | List of trusted proxy host addresses separated by comma                                                                                    | N        | Trust all proxies                |
| `DEBUG`           | bool   | Output debug logs of received requests                                                                                                     | N        | `false`                          |
