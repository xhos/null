# ariand config

## cli params

| param      | description            | default  |
|------------|------------------------|----------|
| `--port`   | gRPC server port       | `55555`  |

## env's

| variable                | description                               | default  | required? |
|------------------------ |-------------------------------------------|----------|-----------|
| `API_KEY`               | Authentication key for gRPC API access    |          | [x]       |
| `DATABASE_URL`          | PostgreSQL connection string              |          | [x]       |
| `ARIAN_RECEIPTS_URL`    | gRPC endpoint for receipt parsing service |          | [x]       |
| `RECEIPT_PARSER_TIMEOUT`| Timeout for receipt parser requests       |          | [x]       |
| `LOG_LEVEL`             | Log level: debug, info, warn, error       | `info`   | [ ]       |
| `OPENAI_API_KEY`        | OpenAI API access                         |          | [ ]       |
| `ANTHROPIC_API_KEY`     | Anthropic API access                      |          | [ ]       |
| `OLLAMA_API_KEY`        | Ollama API access                         |          | [ ]       |
| `GOOGLE_API_KEY`        | Google/Gemini API access                  |          | [ ]       |
