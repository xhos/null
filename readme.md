# ariand

ariand is a high-performance gRPC (or rather [connect-go](https://github.com/connectrpc/connect-go)) API built in Go that handles core financial operations including user management, account management, transaction processing, and categorization, among other things.

## ⚙️ config

### environment variables

| variable                  | description                                | default              | required?  |
|---------------------------|--------------------------------------------|----------------------|------------|
| `API_KEY`                 | Authentication key for gRPC API access     |                      | [x]        |
| `DATABASE_URL`            | PostgreSQL connection string               |                      | [x]        |
| `ARIAN_WEB_URL`           | URL for arian-web frontend                 |                      | [x]        |
| `ARIAN_RECEIPTS_URL`      | gRPC endpoint for receipt parsing service  |                      | [x]        |
| `EXCHANGE_API_URL`        | Exchange rate API endpoint                 |                      | [x]        |
| `LISTEN_ADDRESS`          | Server listen address (port or host:port)  | `127.0.0.1:55555`    | [ ]        |
| `LOG_LEVEL`               | Log level: debug, info, warn, error        | `info`               | [ ]        |
| `LOG_FORMAT`              | Log format: json, text                     | `text`               | [ ]        |

## 🌱 ecosystem

```definition
arian (n.) /ˈarjan/ [Welsh] Silver; money; wealth.  
```

- [ariand](https://github.com/xhos/ariand) - main backend service
- [arian-web](https://github.com/xhos/arian-web) - frontend web application
- [arian-mobile](https://github.com/xhos/arian-mobile) - mobile appplication
- [arian-protos](https://github.com/xhos/arian-protos) - shared protobuf definitions
- [arian-receipts](https://github.com/xhos/arian-receipts) - receipt parsing microservice
- [arian-email-parser](https://github.com/xhos/arian-email-parser) - email parsing service
