# Usage Accounting Contract

> Date: 2026-07-16
> Scope: internal traffic accounting and cross-protocol usage extraction
> Status: tested project contract; cached-token billing semantics remain pending vendor-document verification

## Purpose

The Bridge needs deterministic token totals for traffic records without mixing similarly named fields from different protocols. This contract defines what the current implementation may safely extract. It does not claim to reproduce vendor billing calculations.

## Extraction contract

Usage is read first from the top-level `usage` object and, when absent, from `response.usage`. Other recursive or ambiguous locations are ignored.

| Upstream protocol | Input | Output | Total |
| --- | --- | --- | --- |
| Anthropic Messages | `input_tokens` | `output_tokens` | `total_tokens`, otherwise input + output |
| OpenAI Chat Completions | `prompt_tokens` | `completion_tokens` | `total_tokens`, otherwise input + output |
| OpenAI Responses | `input_tokens` | `output_tokens` | `total_tokens`, otherwise input + output |

Fields belonging to another protocol must not be added to the selected fields. For example, a Responses payload containing both `input_tokens` and `prompt_tokens` uses only `input_tokens`.

Missing, malformed, or unrecognized usage returns zero values. Callers may normalize a missing total to input plus output.

## Cached-token mapping

The existing Responses-to-Anthropic response converter currently treats Responses `input_tokens` as inclusive of `input_tokens_details.cached_tokens`, maps the cached portion to Anthropic `cache_read_input_tokens`, and maps the remaining non-cached portion to Anthropic `input_tokens`.

This behavior remains unchanged in this phase. The older 2026-07-11 plan proposed retaining the full Responses input count while also setting the Anthropic cache-read count; that could double-represent cached input and must not be implemented without authoritative vendor semantics and billing fixtures.

`cache_creation_input_tokens` is not synthesized because the current Responses usage model has no equivalent source field.

## Verification limitation

During implementation, the OpenAI developer pages and the installed official OpenAI documentation MCP both returned HTTP 403 from the current network region. The Anthropic documentation route redirected to a region-unavailable page. Therefore:

- protocol-specific field extraction is implemented and fixture-tested;
- cached-token conversion is preserved rather than changed from an unverified assumption;
- vendor billing parity remains a follow-up gate requiring accessible official documentation or captured real API fixtures.

## Required follow-up fixtures

1. OpenAI Responses with cached and uncached input in one response.
2. OpenAI Chat Completions with `prompt_tokens_details.cached_tokens`.
3. Anthropic Messages with cache creation, cache read, and uncached input together.
4. Cross-protocol conversions proving totals are neither lost nor counted twice.

