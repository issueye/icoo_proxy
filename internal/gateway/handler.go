package gateway

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"icoo_proxy/internal/audit"
	"icoo_proxy/internal/protocol"
	"icoo_proxy/internal/provider"
)

// Handler handles gateway HTTP requests.
type Handler struct{}

// NewHandler creates a new Handler.
func NewHandler() *Handler {
	return &Handler{}
}

// ChatCompletions handles POST /v1/chat/completions
func (h *Handler) ChatCompletions(w http.ResponseWriter, r *http.Request) {
	aw := &auditResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
	w = aw
	startedAt := time.Now()
	logEntry := audit.RequestLogInput{
		Method:    r.Method,
		Path:      r.URL.Path,
		ClientIP:  r.RemoteAddr,
		UserAgent: r.UserAgent(),
	}
	defer func() {
		logEntry.StatusCode = aw.statusCode
		logEntry.DurationMs = time.Since(startedAt).Milliseconds()
		logEntry.ResponseHeaders = serializeHeaders(aw.Header())
		logEntry.ResponsePayload = aw.CapturedBody()
		if err := audit.GetService().Add(logEntry); err != nil {
			log.Printf("[Gateway] Failed to persist request log: %v", err)
		}
	}()

	if r.Method != "POST" {
		logEntry.ErrorMessage = "method not allowed"
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logEntry.ErrorMessage = "failed to read request body"
		writeError(w, http.StatusBadRequest, "Failed to read request body", "invalid_request")
		return
	}
	defer r.Body.Close()
	logEntry.Streaming = IsStreamingRequest(body)
	logEntry.RequestPayload = truncateRequestPayload(body)

	// Parse just enough to get the model name
	var partial struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(body, &partial); err != nil {
		logEntry.ErrorMessage = "invalid json"
		writeError(w, http.StatusBadRequest, "Invalid JSON", "invalid_request")
		return
	}
	model := partial.Model
	logEntry.Model = model

	if model == "" {
		logEntry.ErrorMessage = "model is required"
		writeError(w, http.StatusBadRequest, "model is required", "invalid_request")
		return
	}

	// Parse the incoming request using the OpenAI adapter (gateway speaks OpenAI)
	gwAdapter := &protocol.OpenAIAdapter{}
	internalReq, err := gwAdapter.ParseRequest(body)
	if err != nil {
		logEntry.ErrorMessage = fmt.Sprintf("failed to parse request: %v", err)
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Failed to parse request: %v", err), "invalid_request")
		return
	}

	// Find provider for this request
	pm := provider.GetManager()
	decision := pm.ResolveRequest(internalReq)
	if decision == nil || decision.Provider == nil {
		logEntry.ErrorMessage = fmt.Sprintf("no provider found for model: %s", model)
		writeError(w, http.StatusNotFound, fmt.Sprintf("No provider found for model: %s", model), "model_not_found")
		return
	}
	p := decision.Provider
	logEntry.ProviderID = p.Config.ID
	logEntry.ProviderName = p.Config.Name
	logEntry.ProviderType = p.Config.Type

	actualModel := decision.TargetModel
	if actualModel != model {
		log.Printf("[Gateway] Model mapped: %s -> %s (provider: %s)", model, actualModel, p.Config.Name)
	}
	model = actualModel
	logEntry.TargetModel = model

	// Set the mapped model name for the target provider
	internalReq.Model = model

	// Check if we need protocol conversion
	_, isOpenAIChatAdapter := p.Adapter.(*protocol.OpenAIAdapter)
	needsConversion := !isOpenAIChatAdapter

	if !needsConversion {
		// For passthrough, we need to update the model in the request body
		body, err = updateModelInBody(body, model)
		if err != nil {
			logEntry.ErrorMessage = fmt.Sprintf("failed to update model in body: %v", err)
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update model in body: %v", err), "server_error")
			return
		}
		// Direct passthrough for OpenAI-compatible providers
		h.handlePassthrough(w, r, p, body, &logEntry)
		return
	}

	// Need protocol conversion
	if internalReq.Stream {
		h.handleStreamWithConversion(w, r, p, internalReq, &logEntry)
	} else {
		h.handleNonStreamWithConversion(w, r, p, internalReq, &logEntry)
	}
}

// Responses handles POST /v1/responses
func (h *Handler) Responses(w http.ResponseWriter, r *http.Request) {
	aw := &auditResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
	w = aw
	startedAt := time.Now()
	logEntry := audit.RequestLogInput{
		Method:    r.Method,
		Path:      r.URL.Path,
		ClientIP:  r.RemoteAddr,
		UserAgent: r.UserAgent(),
	}
	defer func() {
		logEntry.StatusCode = aw.statusCode
		logEntry.DurationMs = time.Since(startedAt).Milliseconds()
		logEntry.ResponseHeaders = serializeHeaders(aw.Header())
		logEntry.ResponsePayload = aw.CapturedBody()
		if err := audit.GetService().Add(logEntry); err != nil {
			log.Printf("[Gateway] Failed to persist request log: %v", err)
		}
	}()

	if r.Method != "POST" {
		logEntry.ErrorMessage = "method not allowed"
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logEntry.ErrorMessage = "failed to read request body"
		writeError(w, http.StatusBadRequest, "Failed to read request body", "invalid_request")
		return
	}
	defer r.Body.Close()
	logEntry.Streaming = IsStreamingRequest(body)
	logEntry.RequestPayload = truncateRequestPayload(body)

	gwAdapter := &protocol.OpenAIAdapter{}
	internalReq, err := gwAdapter.ParseResponsesRequest(body)
	if err != nil {
		logEntry.ErrorMessage = fmt.Sprintf("failed to parse responses request: %v", err)
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Failed to parse request: %v", err), "invalid_request")
		return
	}

	model := internalReq.Model
	logEntry.Model = model
	if model == "" {
		logEntry.ErrorMessage = "model is required"
		writeError(w, http.StatusBadRequest, "model is required", "invalid_request")
		return
	}

	pm := provider.GetManager()
	decision := pm.ResolveRequest(internalReq)
	if decision == nil || decision.Provider == nil {
		logEntry.ErrorMessage = fmt.Sprintf("no provider found for model: %s", model)
		writeError(w, http.StatusNotFound, fmt.Sprintf("No provider found for model: %s", model), "model_not_found")
		return
	}
	p := decision.Provider
	logEntry.ProviderID = p.Config.ID
	logEntry.ProviderName = p.Config.Name
	logEntry.ProviderType = p.Config.Type

	actualModel := decision.TargetModel
	if actualModel != model {
		log.Printf("[Gateway] Responses model mapped: %s -> %s (provider: %s)", model, actualModel, p.Config.Name)
	}
	internalReq.Model = actualModel
	logEntry.TargetModel = actualModel

	if internalReq.Stream {
		h.handleResponsesStream(w, r, p, internalReq, &logEntry)
		return
	}
	h.handleResponsesNonStream(w, r, p, internalReq, &logEntry)
}

// handlePassthrough forwards requests directly to OpenAI-compatible providers.
func (h *Handler) handlePassthrough(w http.ResponseWriter, r *http.Request, p *provider.ProviderRuntime, body []byte, logEntry *audit.RequestLogInput) {
	// Use the model from the parsed request (already mapped)
	model := extractModel(body)

	_, path, err := p.Adapter.BuildRequest(&protocol.InternalRequest{Model: model})
	if err != nil {
		logEntry.ErrorMessage = err.Error()
		writeError(w, http.StatusInternalServerError, err.Error(), "server_error")
		return
	}

	resp, err := provider.GetManager().DoRequestRaw(r.Context(), p, "POST", path, body)
	if err != nil {
		logEntry.ErrorMessage = fmt.Sprintf("provider request failed: %v", err)
		writeError(w, http.StatusBadGateway, fmt.Sprintf("Provider request failed: %v", err), "server_error")
		return
	}
	defer resp.Body.Close()

	// Copy headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// handleNonStreamWithConversion handles non-streaming requests with protocol conversion.
func (h *Handler) handleNonStreamWithConversion(w http.ResponseWriter, r *http.Request, p *provider.ProviderRuntime, internalReq *protocol.InternalRequest, logEntry *audit.RequestLogInput) {
	resp, err := provider.GetManager().DoRequestWithRetry(r.Context(), p, internalReq)
	if err != nil {
		logEntry.ErrorMessage = fmt.Sprintf("provider request failed: %v", err)
		writeError(w, http.StatusBadGateway, fmt.Sprintf("Provider request failed: %v", err), "server_error")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		logEntry.ErrorMessage = string(body)
		// Try to parse as provider error and convert to OpenAI format
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logEntry.ErrorMessage = "failed to read provider response"
		writeError(w, http.StatusInternalServerError, "Failed to read provider response", "server_error")
		return
	}

	// Parse provider response into internal format
	internalResp, err := p.Adapter.ParseResponse(respBody)
	if err != nil {
		logEntry.ErrorMessage = fmt.Sprintf("failed to parse provider response: %v", err)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to parse provider response: %v", err), "server_error")
		return
	}

	// Ensure the model name is preserved
	internalResp.Model = internalReq.Model

	// Convert to OpenAI response format
	gwAdapter := &protocol.OpenAIAdapter{}
	openaiResp, err := gwAdapter.BuildResponse(internalResp)
	if err != nil {
		logEntry.ErrorMessage = fmt.Sprintf("failed to build response: %v", err)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to build response: %v", err), "server_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(openaiResp)
}

// handleStreamWithConversion handles streaming requests with protocol conversion.
func (h *Handler) handleStreamWithConversion(w http.ResponseWriter, r *http.Request, p *provider.ProviderRuntime, internalReq *protocol.InternalRequest, logEntry *audit.RequestLogInput) {
	resp, err := provider.GetManager().DoRequestWithRetry(r.Context(), p, internalReq)
	if err != nil {
		logEntry.ErrorMessage = fmt.Sprintf("provider request failed: %v", err)
		writeError(w, http.StatusBadGateway, fmt.Sprintf("Provider request failed: %v", err), "server_error")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		logEntry.ErrorMessage = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	// Set SSE headers for OpenAI format
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, canFlush := w.(http.Flusher)
	gwAdapter := &protocol.OpenAIAdapter{}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		// Handle different SSE formats
		if p.Config.Type == "gemini" {
			// Gemini returns JSON array chunks, not SSE
			line = strings.TrimSpace(line)
			if line == "" || line == "[" || line == "]" || line == "," {
				continue
			}
			// Strip leading/trailing brackets and commas
			line = strings.TrimPrefix(line, "[")
			line = strings.TrimPrefix(line, ",")
			line = strings.TrimSuffix(line, "]")
			line = strings.TrimSuffix(line, ",")
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// Parse Gemini chunk and convert
			chunk, err := p.Adapter.ParseStreamEvent("", line)
			if err != nil {
				log.Printf("[Gateway] Failed to parse Gemini stream chunk: %v", err)
				continue
			}
			if chunk.Model == "" {
				chunk.Model = internalReq.Model
			}
			if chunk.StreamDone {
				fmt.Fprintf(w, "data: [DONE]\n\n")
				if canFlush {
					flusher.Flush()
				}
				break
			}

			// Convert to OpenAI SSE
			_, data, err := gwAdapter.BuildStreamEvent(chunk)
			if err != nil || data == "" {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			if canFlush {
				flusher.Flush()
			}
			continue
		}

		// Anthropic SSE format: "event: xxx\ndata: xxx"
		if strings.HasPrefix(line, "event: ") {
			// Store event type, next line will have data
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Check for stream done
		if data == p.Adapter.StreamDone() || data == "" {
			fmt.Fprintf(w, "data: [DONE]\n\n")
			if canFlush {
				flusher.Flush()
			}
			break
		}

		// Parse provider SSE event
		chunk, err := p.Adapter.ParseStreamEvent("", data)
		if err != nil {
			log.Printf("[Gateway] Failed to parse stream event: %v", err)
			continue
		}
		if chunk.Model == "" {
			chunk.Model = internalReq.Model
		}
		if len(chunk.Choices) == 0 && !chunk.StreamDone && chunk.Usage == nil {
			continue
		}
		if chunk.StreamDone {
			fmt.Fprintf(w, "data: [DONE]\n\n")
			if canFlush {
				flusher.Flush()
			}
			break
		}

		// Convert to OpenAI SSE
		_, openaiData, err := gwAdapter.BuildStreamEvent(chunk)
		if err != nil || openaiData == "" {
			continue
		}
		fmt.Fprintf(w, "data: %s\n\n", openaiData)
		if canFlush {
			flusher.Flush()
		}
	}
}

func (h *Handler) handleResponsesNonStream(w http.ResponseWriter, r *http.Request, p *provider.ProviderRuntime, internalReq *protocol.InternalRequest, logEntry *audit.RequestLogInput) {
	resp, err := provider.GetManager().DoRequestWithRetry(r.Context(), p, internalReq)
	if err != nil {
		logEntry.ErrorMessage = fmt.Sprintf("provider request failed: %v", err)
		writeError(w, http.StatusBadGateway, fmt.Sprintf("Provider request failed: %v", err), "server_error")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		logEntry.ErrorMessage = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logEntry.ErrorMessage = "failed to read provider response"
		writeError(w, http.StatusInternalServerError, "Failed to read provider response", "server_error")
		return
	}

	internalResp, err := p.Adapter.ParseResponse(respBody)
	if err != nil {
		logEntry.ErrorMessage = fmt.Sprintf("failed to parse provider response: %v", err)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to parse provider response: %v", err), "server_error")
		return
	}
	internalResp.Model = internalReq.Model

	gwAdapter := &protocol.OpenAIAdapter{}
	responsesResp, err := gwAdapter.BuildResponsesResponse(internalResp)
	if err != nil {
		logEntry.ErrorMessage = fmt.Sprintf("failed to build responses response: %v", err)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to build response: %v", err), "server_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responsesResp)
}

func (h *Handler) handleResponsesStream(w http.ResponseWriter, r *http.Request, p *provider.ProviderRuntime, internalReq *protocol.InternalRequest, logEntry *audit.RequestLogInput) {
	resp, err := provider.GetManager().DoRequestWithRetry(r.Context(), p, internalReq)
	if err != nil {
		logEntry.ErrorMessage = fmt.Sprintf("provider request failed: %v", err)
		writeError(w, http.StatusBadGateway, fmt.Sprintf("Provider request failed: %v", err), "server_error")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		logEntry.ErrorMessage = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, canFlush := w.(http.Flusher)
	gwAdapter := &protocol.OpenAIAdapter{}
	state := &protocol.OpenAIResponsesStreamState{}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if p.Config.Type == "gemini" {
			line = strings.TrimSpace(line)
			if line == "" || line == "[" || line == "]" || line == "," {
				continue
			}
			line = strings.TrimPrefix(line, "[")
			line = strings.TrimPrefix(line, ",")
			line = strings.TrimSuffix(line, "]")
			line = strings.TrimSuffix(line, ",")
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			chunk, err := p.Adapter.ParseStreamEvent("", line)
			if err != nil {
				log.Printf("[Gateway] Failed to parse Gemini stream chunk: %v", err)
				continue
			}
			events, err := gwAdapter.BuildResponsesStreamEvents(chunk, state)
			if err != nil {
				log.Printf("[Gateway] Failed to build responses stream events: %v", err)
				continue
			}
			for _, event := range events {
				fmt.Fprintf(w, "data: %s\n\n", event)
			}
			if canFlush && len(events) > 0 {
				flusher.Flush()
			}
			if chunk.StreamDone {
				break
			}
			continue
		}

		if strings.HasPrefix(line, "event: ") {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == p.Adapter.StreamDone() || data == "" {
			chunk := &protocol.InternalStreamChunk{Model: internalReq.Model, StreamDone: true}
			events, err := gwAdapter.BuildResponsesStreamEvents(chunk, state)
			if err == nil {
				for _, event := range events {
					fmt.Fprintf(w, "data: %s\n\n", event)
				}
			}
			if canFlush {
				flusher.Flush()
			}
			break
		}

		chunk, err := p.Adapter.ParseStreamEvent("", data)
		if err != nil {
			log.Printf("[Gateway] Failed to parse stream event: %v", err)
			continue
		}
		chunk.Model = internalReq.Model
		events, err := gwAdapter.BuildResponsesStreamEvents(chunk, state)
		if err != nil {
			log.Printf("[Gateway] Failed to build responses stream events: %v", err)
			continue
		}
		for _, event := range events {
			fmt.Fprintf(w, "data: %s\n\n", event)
		}
		if canFlush && len(events) > 0 {
			flusher.Flush()
		}
		if chunk.StreamDone {
			break
		}
	}
}

// Models handles GET /v1/models
func (h *Handler) Models(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request")
		return
	}

	pm := provider.GetManager()
	models := pm.GetAllModels()

	type modelObject struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	}

	result := struct {
		Object string        `json:"object"`
		Data   []modelObject `json:"data"`
	}{
		Object: "list",
		Data:   make([]modelObject, 0, len(models)),
	}

	for _, m := range models {
		result.Data = append(result.Data, modelObject{
			ID:      m.ID,
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: m.OwnedBy,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Health handles GET /v1/health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request")
		return
	}

	s := GetServer()
	pm := provider.GetManager()
	allProviders := pm.GetAll()
	healthyCount := 0
	for _, p := range allProviders {
		if p.Healthy {
			healthyCount++
		}
	}

	response := struct {
		Status        string `json:"status"`
		Port          int    `json:"port"`
		ProviderCount int    `json:"providerCount"`
		HealthyCount  int    `json:"healthyCount"`
	}{
		Status:        "ok",
		Port:          s.GetPort(),
		ProviderCount: len(allProviders),
		HealthyCount:  healthyCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// writeError writes an OpenAI-compatible error response.
func writeError(w http.ResponseWriter, statusCode int, message, errType string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"message": message,
			"type":    errType,
		},
	})
}

// extractModel extracts the model name from a request body.
func extractModel(body []byte) string {
	var partial struct {
		Model string `json:"model"`
	}
	json.Unmarshal(body, &partial)
	return partial.Model
}

// updateModelInBody updates the model field in the request body and returns the modified body.
func updateModelInBody(body []byte, model string) ([]byte, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return body, err
	}
	data["model"] = model
	return json.Marshal(data)
}

type auditResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

func (rw *auditResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *auditResponseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	const maxLen = 2048
	if len(b) > 0 && len(rw.body) < maxLen {
		remaining := maxLen - len(rw.body)
		if len(b) > remaining {
			rw.body = append(rw.body, b[:remaining]...)
		} else {
			rw.body = append(rw.body, b...)
		}
	}
	return rw.ResponseWriter.Write(b)
}

func (rw *auditResponseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (rw *auditResponseWriter) CapturedBody() string {
	if len(rw.body) == 0 {
		return ""
	}
	return truncatePayload(rw.body)
}

func truncateRequestPayload(body []byte) string {
	return truncatePayload(body)
}

func truncatePayload(body []byte) string {
	const maxLen = 2048
	if len(body) <= maxLen {
		return string(body)
	}
	return string(body[:maxLen]) + "...(truncated)"
}

func serializeHeaders(header http.Header) string {
	if len(header) == 0 {
		return ""
	}
	payload, err := json.Marshal(header)
	if err != nil {
		return ""
	}
	return truncatePayload(payload)
}
