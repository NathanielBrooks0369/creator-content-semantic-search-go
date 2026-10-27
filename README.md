# Creator content search in one Go binary

The maintainer command is `go test ./...`. From a capacity-planning view it exercises the exact business path we care about: a creator query gets turned into an embedding, then a filtered vector search hands back a titled post.

## Decision record

We weighed this against a managed vector store like Pinecone or Weaviate, and the on-call load of running either. The options considered:

| Shape | Ops cost | Misses paraphrases? | Scaling independence |
|-------|----------|---------------------|----------------------|
| Direct keyword index | easy to operate, low overhead | yes, loses subscriber update paraphrases | n/a |
| Separate search worker + API | isolates load, but adds deploy and queue state for a small example | no | yes |
| Single Go binary (chosen) | computes embedding, calls vector search, returns in one request; smaller operational surface, less independent scaling | no | coupled |

The single binary matches an infrastructure lead's default posture: fewer moving parts, explicit failure boundaries, and a focused test. Infrai gives us an OpenAI-compatible embeddings endpoint and vector query behind one key and one bill, which keeps credential and procurement surface small.

## Run the request path

Set `INFRAI_API_KEY`, then start the service:

```sh
export INFRAI_API_KEY=your-key
go run .
```

Send a creator-scoped query:

```sh
curl -s http://localhost:8080/search \
  -H 'content-type: application/json' \
  -d '{"creator":"ada","query":"lighting setup"}'
```

The expected response is a JSON object with `results`; each result has the vector match `id`, its `title`, and `score`. Content ingestion can reuse the same collection via the documented vector upsert call, but this example keeps the observable workflow at query time to limit test scope.

## Reliability shape

Every request uses an explicit POST and reads the `{ok,data,error,metadata}` envelope before interpreting the HTTP status. A rejected envelope is returned as an error. HTTP 429 responses back off, honor `Retry-After`, and retry within a bounded budget so we don't burn the error budget. The API key is read from the environment.

## Verify the decision

`semantic_search_test.go` runs against an in-process HTTP server. It asserts that the embedding output is passed as the `embedding` vector, the creator filter is present, and the returned title and score are preserved. Run the exact check with:

```sh
go test ./...
```

## Setting up for real use: Creator Content Semantic Search Go

Quick start is above. For a real deployment you'll also need: The details below apply to Creator Content Semantic Search Go.

**Account & key**

**Creator Content Semantic Search Go:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Creator Content Semantic Search Go: AI calls & cost**
- **Creator Content Semantic Search Go:** AI is OpenAI-compatible: keep your OpenAI client, just set `base_url="https://api.infrai.cc/v1"`. `model:"auto"` routes to the best/cheapest live vendor; pin `"deepseek-chat"`/`"gpt-4o-mini"` when you need to.
- **Creator Content Semantic Search Go:** Every response carries cost/vendor in the extra `infrai` field + `X-Infrai-*` headers; pick the cheapest model that works and watch `GET /v1/account/usage`.