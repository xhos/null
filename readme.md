# ariand

ariand is a high-performance gRPC (or rather [connect-go](https://github.com/connectrpc/connect-go)) API built in Go that handles core financial operations including user management, account management, transaction processing, and categorization, among other things.

## ⚙️ config

### cli params

| param     | description       | default  |
|-----------|-------------------|----------|
| `--port`  | gRPC server port  | `55555`  |

### environment variables

| variable                  | description                                | default  | required?  |
|---------------------------|--------------------------------------------|----------|------------|
| `API_KEY`                 | Authentication key for gRPC API access     |          | [x]        |
| `DATABASE_URL`            | PostgreSQL connection string               |          | [x]        |
| `ARIAN_RECEIPTS_URL`      | gRPC endpoint for receipt parsing service  |          | [x]        |
| `BETTER_AUTH_URL`         | URL for BetterAuth service                 |          | [x]        |
| `RECEIPT_PARSER_TIMEOUT`  | Timeout for receipt parser requests        |          | [ ]        |
| `LOG_LEVEL`               | Log level: debug, info, warn, error        | `info`   | [ ]        |
| `OPENAI_API_KEY`          | OpenAI API access                          |          | [ ]        |
| `ANTHROPIC_API_KEY`       | Anthropic API access                       |          | [ ]        |
| `OLLAMA_API_KEY`          | Ollama API access                          |          | [ ]        |
| `GOOGLE_API_KEY`          | Google/Gemini API access                   |          | [ ]        |

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
