---
title: AI Provider Setup
description: Configure encrypted API keys and dynamic model fetching.
---

Zgit supports multiple AI providers for commit messages, PR generation, and other smart features.

## Supported Providers

| Provider | Models | Auth |
|----------|--------|------|
| **OpenAI** | GPT-4o, GPT-4o-mini, o1, o3 | Bearer token |
| **Anthropic** | Claude 3.5 Sonnet, Claude 3 Opus | `x-api-key` header |
| **Gemini** | Gemini 1.5 Pro, Gemini 1.5 Flash | API key (query param) |
| **DeepSeek** | DeepSeek-V2, DeepSeek-Coder | Bearer token |
| **Groq** | Llama 3, Mixtral, Gemma | Bearer token |
| **OpenRouter** | 200+ models via unified API | Bearer token |
| **Ollama** | Any local model | No auth (localhost) |

## Adding a Provider

1. Open **Settings → AI**
2. Click **Add Provider**
3. Select the provider from the dropdown
4. Enter your API key
5. Choose a model (fetch dynamically or type manually)
6. Toggle **Set as Active**

## Encrypted Key Storage

API keys are encrypted with **AES-256-GCM** before being saved to disk.

- Encryption key auto-generated at `~/.config/zgit/.ai_key_encryption` (`0600` permissions)
- Plaintext keys **never sent to the frontend**
- UI shows only masked previews: `sk-...abcd`
- Keys persist per-provider; delete via **Clear** button

## Dynamic Model Fetching

Each provider's available models can be fetched live:

| Provider | Endpoint |
|----------|----------|
| OpenAI | `GET https://api.openai.com/v1/models` |
| Groq | `GET https://api.groq.com/openai/v1/models` |
| OpenRouter | `GET https://openrouter.ai/api/v1/models` |
| Ollama | `GET http://localhost:11434/api/tags` |

Fetched models appear as clickable chips. Select one or type a custom model ID.

## Configuration File

Keys are stored in `~/.config/zgit/config.yaml` under:

```yaml
ai:
  active_provider: openai
  providers:
    openai:
      api_key: <encrypted>
      model: gpt-4o
      endpoint: https://api.openai.com/v1
```

Use `zgit config set-ai --provider openai --model gpt-4o` from the CLI.
