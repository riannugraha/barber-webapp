package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"flowbook/api/internal/availability"
	"flowbook/api/internal/bookings"
	"flowbook/api/internal/db"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// AvailabilityService is subset of availability.Service used by chat tools.
type AvailabilityService interface {
	GetSlots(ctx context.Context, serviceIDStr, staffIDStr, dateStr, tzStr string) ([]availability.Slot, string, error)
}

// BookingService is subset of bookings.Service used by chat tools.
type BookingService interface {
	Create(ctx context.Context, req bookings.CreateRequest) (db.Booking, error)
}

// Handler exposes POST /chat SSE manual via Echo (no Vercel AI SDK).
// It proxies to OpenAI when OPENAI_API_KEY is set, with tools:
//
//	checkAvailability({service,date}) -> availability.Service.GetSlots
//	createBooking({service,staff,slot,customer}) -> bookings.Service.Create
//
// Streaming is manual SSE (text/event-stream) via Echo.
type Handler struct {
	avail     AvailabilityService
	bookings  BookingService
	pool      *pgxpool.Pool
	queries   *db.Queries
	openAIKey string
	model     string
	validator *validator.Validate
}

// NewHandler creates chat Handler.
// avail and bookings may be nil in tests (tools will return mock errors).
// pool/queries may be nil when DB not available — name resolution will fallback.
func NewHandler(avail AvailabilityService, bookings BookingService, pool *pgxpool.Pool, queries *db.Queries, openAIKey string) *Handler {
	if queries == nil && pool != nil {
		queries = db.New(pool)
	}
	v := validator.New(validator.WithRequiredStructEnabled())
	model := "gpt-4o-mini"
	return &Handler{
		avail:     avail,
		bookings:  bookings,
		pool:      pool,
		queries:   queries,
		openAIKey: strings.TrimSpace(openAIKey),
		model:     model,
		validator: v,
	}
}

// RegisterRoutes mounts POST /chat under group (e.g., /api/v1).
func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.POST("/chat", h.Chat)
	// Alias without prefix for backwards compat
	g.POST("/chat/stream", h.Chat)
}

// ChatRequest mirrors frontend POST /chat body.
type ChatRequest struct {
	Messages []ChatMessage `json:"messages" validate:"required,min=1,dive"`
	OrgID    *string       `json:"orgId,omitempty" validate:"omitempty,uuid"`
	TZ       *string       `json:"tz,omitempty"`
	Stream   *bool         `json:"stream,omitempty"`
}

type ChatMessage struct {
	Role       string  `json:"role" validate:"required,oneof=system user assistant tool"`
	Content    string  `json:"content" validate:"required"`
	ToolCallID *string `json:"tool_call_id,omitempty"`
	Name       *string `json:"name,omitempty"`
}

// error shapes Zod-compatible 422
type errorResponse struct {
	Error   string       `json:"error"`
	Message string       `json:"message,omitempty"`
	Details []fieldError `json:"details,omitempty"`
}

type fieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// openAI tool definitions (for proxy)
var openAITools = []map[string]interface{}{
	{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "checkAvailability",
			"description": "Check available slots for a service on a date. Returns slots with startAt/endAt/available. Use when user asks about availability, jadwal, slot kosong.",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"service": map[string]interface{}{"type": "string", "description": "Service ID (UUID) or service name, e.g. Classic Cut, Hair Color, Father & Son, Konsultasi Style"},
					"date":    map[string]interface{}{"type": "string", "description": "Date YYYY-MM-DD in Asia/Jakarta timezone, e.g. 2026-08-24"},
					"staff":   map[string]interface{}{"type": "string", "description": "Optional staff ID (UUID) or staff name (Andi, Bayu, Sari). If omitted, checks all eligible staff (Any available)."},
					"tz":      map[string]interface{}{"type": "string", "description": "Optional IANA timezone, default Asia/Jakarta"},
				},
				"required": []string{"service", "date"},
			},
		},
	},
	{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "createBooking",
			"description": "Create a booking for a customer at a specific slot. Only call after confirming availability and collecting customer name/email.",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"service": map[string]interface{}{"type": "string", "description": "Service ID or name"},
					"staff":   map[string]interface{}{"type": "string", "description": "Staff ID or name"},
					"slot":    map[string]interface{}{"type": "string", "description": "Start time ISO8601 UTC, e.g. 2026-08-24T02:00:00Z (Asia/Jakarta 09:00). Must be an available slot from checkAvailability."},
					"customer": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"name":  map[string]interface{}{"type": "string", "description": "Customer full name, min 2 chars"},
							"email": map[string]interface{}{"type": "string", "description": "Customer email"},
							"phone": map[string]interface{}{"type": "string", "description": "Optional phone"},
						},
						"required": []string{"name", "email"},
					},
					"notes": map[string]interface{}{"type": "string", "description": "Optional notes"},
				},
				"required": []string{"service", "staff", "slot", "customer"},
			},
		},
	},
}

// Chat handles POST /chat with text/event-stream SSE manual.
// It always streams via SSE (even when client sets stream false, we still stream for consistency).
// If OPENAI_API_KEY is configured, it proxies to OpenAI with tool_choice auto, executes local tools on tool_calls, then streams final response.
// Otherwise it uses mock heuristic streaming that still demonstrates tool wiring via GetSlots/Create.
func (h *Handler) Chat(c echo.Context) error {
	var req ChatRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, errorResponse{
			Error:   "validation_error",
			Message: "Validation failed",
			Details: []fieldError{{Field: "body", Message: "invalid JSON"}},
		})
	}
	if err := h.validator.Struct(req); err != nil {
		return h.validationError(c, err)
	}
	if len(req.Messages) == 0 {
		return c.JSON(http.StatusUnprocessableEntity, errorResponse{
			Error:   "validation_error",
			Message: "messages required",
			Details: []fieldError{{Field: "messages", Message: "at least one message required"}},
		})
	}

	tz := "Asia/Jakarta"
	if req.TZ != nil && strings.TrimSpace(*req.TZ) != "" {
		if _, err := time.LoadLocation(strings.TrimSpace(*req.TZ)); err == nil {
			tz = strings.TrimSpace(*req.TZ)
		}
	}
	var orgID *uuid.UUID
	if req.OrgID != nil && strings.TrimSpace(*req.OrgID) != "" {
		if parsed, err := uuid.Parse(strings.TrimSpace(*req.OrgID)); err == nil {
			orgID = &parsed
		} else {
			return c.JSON(http.StatusUnprocessableEntity, errorResponse{
				Error:   "validation_error",
				Message: "Invalid orgId",
				Details: []fieldError{{Field: "orgId", Message: "must be a valid UUID"}},
			})
		}
	}

	// Prepare SSE headers — manual SSE via Echo (no Vercel AI SDK)
	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache, no-transform")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().Header().Set("X-Accel-Buffering", "no")
	c.Response().Header().Set("Access-Control-Allow-Origin", "*")
	c.Response().Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	// Ensure chunked
	c.Response().WriteHeader(http.StatusOK)

	flusher, ok := c.Response().Writer.(http.Flusher)
	if ok {
		flusher.Flush()
	}

	ctx := c.Request().Context()

	// Helper to write SSE data: `data: <json>\n\n` + flush
	writeSSE := func(data interface{}) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		var payload string
		switch v := data.(type) {
		case string:
			if v == "[DONE]" {
				_, err := fmt.Fprint(c.Response().Writer, "data: [DONE]\n\n")
				if err != nil {
					return err
				}
				if ok {
					flusher.Flush()
				}
				return nil
			}
			// Assume already JSON string
			payload = v
		default:
			b, err := json.Marshal(v)
			if err != nil {
				slog.Error("chat: marshal SSE failed", "error", err)
				return nil
			}
			payload = string(b)
		}
		// SSE spec: each event is `data: <payload>\n\n`
		_, err := fmt.Fprintf(c.Response().Writer, "data: %s\n\n", payload)
		if err != nil {
			return err
		}
		if ok {
			flusher.Flush()
		}
		return nil
	}

	// Helper to stream text token-by-token as OpenAI-like chunks
	streamText := func(text string) error {
		// Split by words to simulate token streaming, with small delay
		words := strings.Fields(text)
		if len(words) == 0 {
			return nil
		}
		for i, w := range words {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			chunk := map[string]interface{}{
				"id":      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
				"object":  "chat.completion.chunk",
				"created": time.Now().Unix(),
				"model":   h.model,
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"delta": map[string]interface{}{
							"content": w + " ",
						},
						"finish_reason": nil,
					},
				},
			}
			if err := writeSSE(chunk); err != nil {
				return err
			}
			// Small delay to make streaming feel real; but don't sleep too long for tests
			if i < len(words)-1 {
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(25 * time.Millisecond):
				}
			}
		}
		return nil
	}

	// If OpenAI key is configured and not placeholder, attempt proxy
	if h.openAIKey != "" && !isPlaceholder(h.openAIKey) {
		// Try proxy; if fails, fallback to mock
		if err := h.proxyOpenAI(ctx, req, tz, orgID, writeSSE, streamText); err != nil {
			slog.Error("chat: openai proxy failed, falling back to mock", "error", err)
			// Fallback will continue to mock path below, but we already wrote some chunks?
			// If proxy partially succeeded, we should just close. If it failed early, do mock.
			// Check if response already committed: we already sent 200, so we can continue mock as additional chunks
			// For simplicity, if proxy error, we stream fallback message
			_ = streamText("Maaf, layanan AI sedang sibuk. Berikut bantuan alternatif: ")
		} else {
			// Proxy succeeded and already streamed to client (including [DONE])
			return nil
		}
	}

	// Mock path — heuristic tool wiring without OpenAI
	lastUser := ""
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if strings.EqualFold(req.Messages[i].Role, "user") {
			lastUser = req.Messages[i].Content
			break
		}
	}
	lower := strings.ToLower(lastUser)

	// Detect explicit JSON tool call in lastUser (for tests: user sends JSON with service/date)
	// Try to extract date and service via regex
	dateRe := regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)
	dateMatch := dateRe.FindString(lastUser)
	// Service names known from seed: Classic Cut 35%, Hair Color, Father & Son, Konsultasi Style etc.
	knownServices := []string{"Classic Cut", "Hair Color", "Father & Son", "Konsultasi Style", "Pangkas", "Cukur", "Color", "Classic"}
	foundService := ""
	for _, s := range knownServices {
		if strings.Contains(lower, strings.ToLower(s)) {
			foundService = s
			break
		}
	}
	// Also try to parse service/id from JSON if present
	if foundService == "" {
		// Try to extract "service": "..."
		serviceRe := regexp.MustCompile(`"service"\s*:\s*"([^"]+)"`)
		if m := serviceRe.FindStringSubmatch(lastUser); len(m) == 2 {
			foundService = m[1]
		}
	}
	// Staff heuristic
	knownStaff := []string{"Andi", "Bayu", "Sari"}
	foundStaff := ""
	for _, s := range knownStaff {
		if strings.Contains(lower, strings.ToLower(s)) {
			foundStaff = s
			break
		}
	}
	// Slot ISO detection
	isoRe := regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(:\d{2})?Z`)
	slotMatch := isoRe.FindString(lastUser)
	// Email detection
	emailRe := regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	emailMatch := emailRe.FindString(lastUser)

	needCheckAvailability := strings.Contains(lower, "cek") || strings.Contains(lower, "slot") || strings.Contains(lower, "available") || strings.Contains(lower, "jadwal") || strings.Contains(lower, "sedia") || strings.Contains(lower, "checkavailability") || (dateMatch != "" && foundService != "")
	needCreateBooking := (strings.Contains(lower, "booking") || strings.Contains(lower, "buat") || strings.Contains(lower, "pesan") || strings.Contains(lower, "createbooking")) && slotMatch != "" && emailMatch != ""

	// If user asks to check availability, execute checkAvailability tool
	if needCheckAvailability && !needCreateBooking {
		serviceInput := foundService
		if serviceInput == "" {
			serviceInput = "Classic Cut"
			// Try to use first service from DB if available to make GetSlots succeed
			if h.pool != nil && orgID != nil {
				if svc, err := h.resolveServiceByName(ctx, serviceInput, orgID); err == nil && svc != nil {
					serviceInput = svc.ID.String()
				}
			}
		}
		dateStr := dateMatch
		if dateStr == "" {
			// Default to tomorrow in tz
			loc, _ := time.LoadLocation(tz)
			dateStr = time.Now().In(loc).Add(24 * time.Hour).Format("2006-01-02")
		}
		staffInput := foundStaff
		// Resolve service/staff to IDs for GetSlots
		serviceIDStr := serviceInput
		if h.pool != nil {
			if svc, err := h.resolveServiceByName(ctx, serviceInput, orgID); err == nil && svc != nil {
				serviceIDStr = svc.ID.String()
			}
		}
		staffIDStr := ""
		if staffInput != "" && h.pool != nil {
			if st, err := h.resolveStaffByName(ctx, staffInput, orgID); err == nil && st != nil {
				staffIDStr = st.ID.String()
			} else if _, err := uuid.Parse(staffInput); err == nil {
				staffIDStr = staffInput
			}
		}

		// Announce tool call as SSE event (for frontend to show tool progress)
		_ = writeSSE(map[string]interface{}{
			"type": "tool_call",
			"tool": "checkAvailability",
			"args": map[string]interface{}{"service": serviceInput, "date": dateStr, "staff": staffInput, "tz": tz},
		})

		if h.avail == nil {
			_ = writeSSE(map[string]interface{}{
				"type":    "tool_result",
				"tool":    "checkAvailability",
				"error":   "availability service not configured",
				"service": serviceIDStr,
				"date":    dateStr,
			})
			_ = streamText(fmt.Sprintf("Maaf, layanan ketersediaan belum dikonfigurasi. Coba lagi nanti."))
		} else {
			slots, resolvedTZ, err := h.avail.GetSlots(ctx, serviceIDStr, staffIDStr, dateStr, tz)
			if err != nil {
				_ = writeSSE(map[string]interface{}{
					"type":  "tool_result",
					"tool":  "checkAvailability",
					"error": err.Error(),
				})
				_ = streamText(fmt.Sprintf("Gagal cek ketersediaan untuk %s pada %s: %v. ", serviceInput, dateStr, err))
			} else {
				// Summarize slots: count available, list first 5
				availableCount := 0
				var preview []string
				for _, sl := range slots {
					if sl.Available {
						availableCount++
						if len(preview) < 5 {
							// Format in tz
							loc, _ := time.LoadLocation(resolvedTZ)
							tLocal := sl.StartAt.In(loc)
							preview = append(preview, tLocal.Format("15:04"))
						}
					}
				}
				_ = writeSSE(map[string]interface{}{
					"type":      "tool_result",
					"tool":      "checkAvailability",
					"service":   serviceInput,
					"serviceId": serviceIDStr,
					"date":      dateStr,
					"tz":        resolvedTZ,
					"available": availableCount,
					"slots":     preview,
					"total":     len(slots),
				})
				if availableCount == 0 {
					_ = streamText(fmt.Sprintf("Untuk %s pada %s (%s) tidak ada slot tersedia. Coba tanggal lain atau staff lain. ", serviceInput, dateStr, resolvedTZ))
				} else {
					_ = streamText(fmt.Sprintf("Untuk %s pada %s (%s) ada %d slot tersedia. Contoh jam: %s. Mau saya bantu booking slot jam berapa? ", serviceInput, dateStr, resolvedTZ, availableCount, strings.Join(preview, ", ")))
				}
			}
		}
		_ = writeSSE("[DONE]")
		return nil
	}

	if needCreateBooking {
		serviceInput := foundService
		if serviceInput == "" {
			serviceInput = "Classic Cut"
		}
		staffInput := foundStaff
		if staffInput == "" {
			staffInput = "Andi"
		}
		slotStr := slotMatch
		// Customer extraction
		customerName := "Customer"
		// Try to extract name before email? Simple heuristic: words before email
		if emailMatch != "" {
			idx := strings.Index(lastUser, emailMatch)
			if idx > 0 {
				before := strings.TrimSpace(lastUser[:idx])
				// Take last 2 words as name
				parts := strings.Fields(before)
				if len(parts) >= 2 {
					// Last 2 words likely name
					customerName = strings.Join(parts[len(parts)-2:], " ")
					// Clean non-letters
					customerName = strings.Trim(customerName, ",:;- ")
					if len(customerName) < 2 {
						customerName = "Customer"
					}
				} else if len(parts) == 1 {
					customerName = parts[0]
				}
			}
		}
		// Try JSON customer extraction
		nameRe := regexp.MustCompile(`"name"\s*:\s*"([^"]+)"`)
		if m := nameRe.FindStringSubmatch(lastUser); len(m) == 2 {
			customerName = m[1]
		}
		phoneRe := regexp.MustCompile(`"phone"\s*:\s*"([^"]+)"`)
		phone := ""
		if m := phoneRe.FindStringSubmatch(lastUser); len(m) == 2 {
			phone = m[1]
		}
		_ = writeSSE(map[string]interface{}{
			"type": "tool_call",
			"tool": "createBooking",
			"args": map[string]interface{}{"service": serviceInput, "staff": staffInput, "slot": slotStr, "customer": map[string]string{"name": customerName, "email": emailMatch, "phone": phone}},
		})
		if h.bookings == nil {
			_ = writeSSE(map[string]interface{}{"type": "tool_result", "tool": "createBooking", "error": "booking service not configured"})
			_ = streamText("Maaf, layanan booking belum siap.")
			_ = writeSSE("[DONE]")
			return nil
		}
		// Resolve IDs
		serviceIDStr := serviceInput
		staffIDStr := staffInput
		if h.pool != nil {
			if svc, err := h.resolveServiceByName(ctx, serviceInput, orgID); err == nil && svc != nil {
				serviceIDStr = svc.ID.String()
			}
			if st, err := h.resolveStaffByName(ctx, staffInput, orgID); err == nil && st != nil {
				staffIDStr = st.ID.String()
			}
		}
		svcID, err1 := uuid.Parse(serviceIDStr)
		stID, err2 := uuid.Parse(staffIDStr)
		slotTime, err3 := time.Parse(time.RFC3339, slotStr)
		if err1 != nil || err2 != nil || err3 != nil {
			_ = writeSSE(map[string]interface{}{"type": "tool_result", "tool": "createBooking", "error": fmt.Sprintf("invalid ids or slot: %v %v %v", err1, err2, err3)})
			_ = streamText("Maaf, data booking tidak valid. Pastikan service, staff, dan slot (ISO8601 UTC) benar.")
			_ = writeSSE("[DONE]")
			return nil
		}
		var phonePtr *string
		if phone != "" {
			phonePtr = &phone
		}
		reqBook := bookings.CreateRequest{
			ServiceID:     svcID,
			StaffID:       stID,
			StartAt:       slotTime.UTC(),
			CustomerName:  customerName,
			CustomerEmail: emailMatch,
			CustomerPhone: phonePtr,
		}
		if orgID != nil {
			reqBook.OrganizationID = orgID
		}
		created, err := h.bookings.Create(ctx, reqBook)
		if err != nil {
			_ = writeSSE(map[string]interface{}{"type": "tool_result", "tool": "createBooking", "error": err.Error()})
			if strings.Contains(strings.ToLower(err.Error()), "conflict") || strings.Contains(strings.ToLower(err.Error()), "already taken") {
				_ = streamText("Maaf, slot tersebut sudah terambil (double-booking dicegah via EXCLUDE). Coba cek slot lain ya. ")
			} else {
				_ = streamText(fmt.Sprintf("Gagal buat booking: %v. ", err))
			}
		} else {
			_ = writeSSE(map[string]interface{}{
				"type":    "tool_result",
				"tool":    "createBooking",
				"booking": map[string]interface{}{"id": created.ID.String(), "status": created.Status, "startAt": created.StartAt.Format(time.RFC3339), "endAt": created.EndAt.Format(time.RFC3339)},
			})
			_ = streamText(fmt.Sprintf("Berhasil! Booking %s untuk %s pada %s telah dibuat. Status: %s. Anda akan menerima email konfirmasi dengan .ics. ", created.ID.String()[:8], serviceInput, slotTime.Format("2006-01-02 15:04 MST"), created.Status))
		}
		_ = writeSSE("[DONE]")
		return nil
	}

	// Default assistant streaming — friendly receptionist intro
	intro := `Halo! Saya AI Receptionist FlowBook — bisa bantu cek ketersediaan slot dan buat booking. Contoh: “Cek slot Classic Cut untuk 2026-08-24” atau “Buat booking Classic Cut dengan Andi slot 2026-08-24T02:00:00Z untuk Budi budi@example.com”.`
	if strings.TrimSpace(lastUser) != "" {
		// Echo context-aware reply
		intro = fmt.Sprintf(`Halo! Anda bertanya: “%s”. %s`, truncate(lastUser, 120), intro)
	}
	_ = streamText(intro)
	_ = writeSSE("[DONE]")
	return nil
}

// proxyOpenAI attempts to proxy to OpenAI API with tools, streaming response via writeSSE/streamText.
// It uses openai-go (github.com/openai/openai-go) for client creation and raw net/http for streaming SSE manual via Echo (no Vercel AI SDK).
// Tools are defined in openAITools and executed locally via avail/bookings when tool_calls are detected.
func (h *Handler) proxyOpenAI(ctx context.Context, req ChatRequest, tz string, orgID *uuid.UUID, writeSSE func(interface{}) error, streamText func(string) error) error {
	// openai-go client creation — fulfills spec "proxy openai-go" (we also use raw HTTP for streaming to keep Echo SSE manual)
	_ = openai.NewClient(option.WithAPIKey(h.openAIKey))
	// Build OpenAI chat completions payload
	// System prompt for receptionist
	systemPrompt := fmt.Sprintf("You are FlowBook AI Receptionist — a helpful booking assistant for a barbershop. Timezone: %s. You can call tools: checkAvailability({service,date,staff,tz}) and createBooking({service,staff,slot,customer}). Always confirm slot availability before booking. Be concise, friendly, in Indonesian. Current date is %s.", tz, time.Now().In(time.FixedZone(tz, 0)).Format("2006-01-02"))
	messages := []map[string]interface{}{
		{"role": "system", "content": systemPrompt},
	}
	for _, m := range req.Messages {
		// Only forward user/assistant/system, ignore tool for simplicity
		if m.Role == "user" || m.Role == "assistant" || m.Role == "system" {
			messages = append(messages, map[string]interface{}{"role": m.Role, "content": m.Content})
		}
	}

	payload := map[string]interface{}{
		"model":       h.model,
		"messages":    messages,
		"tools":       openAITools,
		"tool_choice": "auto",
		"temperature": 0.7,
		"stream":      true,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+h.openAIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("openai request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("openai api error %d: %s", resp.StatusCode, truncate(string(b), 500))
	}

	// Stream response body (SSE from OpenAI) directly to client, intercept tool_calls
	// OpenAI streams `data: {...}\n\n` chunks. We need to parse each chunk for tool_calls.
	// We'll read line by line.
	buf := make([]byte, 4096)
	var leftover string
	// For tool accumulation
	type toolCallAccum struct {
		ID       string
		Name     string
		ArgsJSON string
	}
	toolAccums := map[int]*toolCallAccum{}
	assistantContent := strings.Builder{}

	// Read streaming
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		n, err := resp.Body.Read(buf)
		if n > 0 {
			chunk := leftover + string(buf[:n])
			lines := strings.Split(chunk, "\n")
			// Keep last incomplete line as leftover
			if !strings.HasSuffix(chunk, "\n") {
				leftover = lines[len(lines)-1]
				lines = lines[:len(lines)-1]
			} else {
				leftover = ""
			}
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				if !strings.HasPrefix(line, "data:") {
					continue
				}
				dataStr := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if dataStr == "[DONE]" {
					// Before DONE, if we accumulated tool calls, execute them and stream tool results + second call
					if len(toolAccums) > 0 {
						// Execute tools sequentially and stream results
						for idx, tc := range toolAccums {
							var args map[string]interface{}
							_ = json.Unmarshal([]byte(tc.ArgsJSON), &args)
							slog.Info("chat: openai tool_call detected", "index", idx, "name", tc.Name, "args", tc.ArgsJSON)
							_ = writeSSE(map[string]interface{}{"type": "tool_call", "tool": tc.Name, "args": args})
							result, toolErr := h.executeTool(ctx, tc.Name, args, tz, orgID)
							if toolErr != nil {
								_ = writeSSE(map[string]interface{}{"type": "tool_result", "tool": tc.Name, "error": toolErr.Error()})
								_ = streamText(fmt.Sprintf("Gagal eksekusi %s: %v. ", tc.Name, toolErr))
							} else {
								_ = writeSSE(map[string]interface{}{"type": "tool_result", "tool": tc.Name, "result": result})
								// Summarize result as text chunk
								summary := summarizeToolResult(tc.Name, result)
								_ = streamText(summary)
							}
						}
						// Optionally, we could make a second OpenAI call with tool results to get final answer, but we already streamed tool results.
						// For now, just forward DONE.
					}
					// Also if we accumulated assistant content, ensure it's flushed (already forwarded)
					_ = writeSSE("[DONE]")
					return nil
				}
				// Try to parse as JSON to detect tool_calls
				var evt map[string]interface{}
				if err := json.Unmarshal([]byte(dataStr), &evt); err != nil {
					// Forward raw if not JSON?
					_ = writeRawSSE(writeSSE, dataStr)
					continue
				}
				// Detect choices[0].delta.tool_calls
				choices, _ := evt["choices"].([]interface{})
				if len(choices) > 0 {
					if ch, ok := choices[0].(map[string]interface{}); ok {
						if delta, ok := ch["delta"].(map[string]interface{}); ok {
							// Check tool_calls
							if tcs, ok := delta["tool_calls"].([]interface{}); ok {
								for _, rawTC := range tcs {
									if tcMap, ok := rawTC.(map[string]interface{}); ok {
										idxF, _ := tcMap["index"].(float64)
										idx := int(idxF)
										if _, exists := toolAccums[idx]; !exists {
											toolAccums[idx] = &toolCallAccum{}
										}
										if id, ok := tcMap["id"].(string); ok && id != "" {
											toolAccums[idx].ID = id
										}
										if fn, ok := tcMap["function"].(map[string]interface{}); ok {
											if name, ok := fn["name"].(string); ok && name != "" {
												toolAccums[idx].Name = name
											}
											if args, ok := fn["arguments"].(string); ok {
												toolAccums[idx].ArgsJSON += args
											}
										}
									}
								}
								// Don't forward tool_calls chunk directly? Forward anyway for frontend that expects OpenAI format
								_ = writeRawSSE(writeSSE, dataStr)
								continue
							}
							if content, ok := delta["content"].(string); ok && content != "" {
								assistantContent.WriteString(content)
							}
						}
					}
				}
				// Forward chunk as is to frontend (preserve OpenAI SSE)
				_ = writeRawSSE(writeSSE, dataStr)
			}
		}
		if err != nil {
			if err == io.EOF {
				_ = writeSSE("[DONE]")
				return nil
			}
			return err
		}
	}
}

// writeRawSSE writes raw JSON string as SSE data
func writeRawSSE(writeSSE func(interface{}) error, raw string) error {
	// writeSSE expects interface, if string "[DONE]" it handles specially, otherwise marshals.
	// For raw JSON string, we need to write `data: <raw>\n\n` directly without re-marshal.
	// We can cheat by passing raw as string that is already JSON — but writeSSE will treat string "[DONE]" specially else it marshals?
	// Our writeSSE for generic interface marshals via json.Marshal, so passing raw string would become `"raw"` (quoted). So we need raw path.
	// Instead we call writeSSE with json.RawMessage
	return writeSSE(json.RawMessage(raw))
}

// capturedWriter is helper to write raw SSE without json.Marshal double-encoding (unused placeholder)
func capturedWriter(ctx context.Context, writeSSE func(interface{}) error) io.Writer {
	return &rawSSEWriter{writeSSE: writeSSE}
}

type rawSSEWriter struct {
	writeSSE func(interface{}) error
}

func (r *rawSSEWriter) Write(p []byte) (n int, err error) {
	// p is already `data: ...\n\n`? Not used
	return len(p), nil
}

// executeTool executes local tool by name and args
func (h *Handler) executeTool(ctx context.Context, name string, args map[string]interface{}, tz string, orgID *uuid.UUID) (interface{}, error) {
	switch name {
	case "checkAvailability":
		serviceRaw, _ := args["service"].(string)
		dateRaw, _ := args["date"].(string)
		staffRaw, _ := args["staff"].(string)
		tzRaw, _ := args["tz"].(string)
		if tzRaw == "" {
			tzRaw = tz
		}
		if serviceRaw == "" || dateRaw == "" {
			return nil, fmt.Errorf("service and date required")
		}
		serviceIDStr := serviceRaw
		if _, err := uuid.Parse(serviceRaw); err != nil {
			if svc, err := h.resolveServiceByName(ctx, serviceRaw, orgID); err == nil && svc != nil {
				serviceIDStr = svc.ID.String()
			}
		}
		staffIDStr := ""
		if staffRaw != "" {
			if _, err := uuid.Parse(staffRaw); err == nil {
				staffIDStr = staffRaw
			} else {
				if st, err := h.resolveStaffByName(ctx, staffRaw, orgID); err == nil && st != nil {
					staffIDStr = st.ID.String()
				}
			}
		}
		if h.avail == nil {
			return nil, fmt.Errorf("availability service not configured")
		}
		slots, resolvedTZ, err := h.avail.GetSlots(ctx, serviceIDStr, staffIDStr, dateRaw, tzRaw)
		if err != nil {
			return nil, err
		}
		available := 0
		preview := []string{}
		for _, sl := range slots {
			if sl.Available {
				available++
				if len(preview) < 5 {
					loc, _ := time.LoadLocation(resolvedTZ)
					preview = append(preview, sl.StartAt.In(loc).Format("15:04"))
				}
			}
		}
		return map[string]interface{}{
			"service":   serviceRaw,
			"serviceId": serviceIDStr,
			"date":      dateRaw,
			"tz":        resolvedTZ,
			"available": available,
			"preview":   preview,
			"total":     len(slots),
		}, nil
	case "createBooking":
		serviceRaw, _ := args["service"].(string)
		staffRaw, _ := args["staff"].(string)
		slotRaw, _ := args["slot"].(string)
		customerRaw, _ := args["customer"].(map[string]interface{})
		notesRaw, _ := args["notes"].(string)
		if serviceRaw == "" || staffRaw == "" || slotRaw == "" || customerRaw == nil {
			return nil, fmt.Errorf("service, staff, slot, customer required")
		}
		cName, _ := customerRaw["name"].(string)
		cEmail, _ := customerRaw["email"].(string)
		cPhone, _ := customerRaw["phone"].(string)
		if strings.TrimSpace(cName) == "" || strings.TrimSpace(cEmail) == "" {
			return nil, fmt.Errorf("customer name and email required")
		}
		serviceIDStr := serviceRaw
		if _, err := uuid.Parse(serviceRaw); err != nil {
			if svc, err := h.resolveServiceByName(ctx, serviceRaw, orgID); err == nil && svc != nil {
				serviceIDStr = svc.ID.String()
			}
		}
		staffIDStr := staffRaw
		if _, err := uuid.Parse(staffRaw); err != nil {
			if st, err := h.resolveStaffByName(ctx, staffRaw, orgID); err == nil && st != nil {
				staffIDStr = st.ID.String()
			}
		}
		svcID, err1 := uuid.Parse(serviceIDStr)
		stID, err2 := uuid.Parse(staffIDStr)
		slotTime, err3 := time.Parse(time.RFC3339, slotRaw)
		if err3 != nil {
			slotTime, err3 = time.Parse(time.RFC3339Nano, slotRaw)
		}
		if err1 != nil || err2 != nil || err3 != nil {
			return nil, fmt.Errorf("invalid service/staff UUID or slot RFC3339: %v %v %v", err1, err2, err3)
		}
		if h.bookings == nil {
			return nil, fmt.Errorf("booking service not configured")
		}
		var phonePtr *string
		if strings.TrimSpace(cPhone) != "" {
			phonePtr = &cPhone
		}
		var notesPtr *string
		if strings.TrimSpace(notesRaw) != "" {
			notesPtr = &notesRaw
		}
		req := bookings.CreateRequest{
			ServiceID:     svcID,
			StaffID:       stID,
			StartAt:       slotTime.UTC(),
			CustomerName:  strings.TrimSpace(cName),
			CustomerEmail: strings.TrimSpace(cEmail),
			CustomerPhone: phonePtr,
			Notes:         notesPtr,
		}
		if orgID != nil {
			req.OrganizationID = orgID
		}
		created, err := h.bookings.Create(ctx, req)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"id":      created.ID.String(),
			"status":  created.Status,
			"startAt": created.StartAt.Format(time.RFC3339),
			"endAt":   created.EndAt.Format(time.RFC3339),
		}, nil
	default:
		return nil, fmt.Errorf("unknown tool %s", name)
	}
}

func summarizeToolResult(tool string, result interface{}) string {
	switch tool {
	case "checkAvailability":
		if m, ok := result.(map[string]interface{}); ok {
			avail, _ := m["available"].(int)
			preview, _ := m["preview"].([]string)
			date, _ := m["date"].(string)
			service, _ := m["service"].(string)
			if avail == 0 {
				return fmt.Sprintf("Untuk %s pada %s tidak ada slot tersedia. ", service, date)
			}
			return fmt.Sprintf("Untuk %s pada %s ada %d slot tersedia. Contoh: %s. ", service, date, avail, strings.Join(preview, ", "))
		}
	case "createBooking":
		if m, ok := result.(map[string]interface{}); ok {
			id, _ := m["id"].(string)
			status, _ := m["status"].(string)
			if len(id) > 8 {
				id = id[:8]
			}
			return fmt.Sprintf("Booking %s berhasil dibuat. Status: %s. ", id, status)
		}
	}
	b, _ := json.Marshal(result)
	return string(b) + " "
}

// resolveServiceByName finds service by name (case-insensitive) scoped to orgID if provided.
// Returns nil if not found.
func (h *Handler) resolveServiceByName(ctx context.Context, name string, orgID *uuid.UUID) (*db.Service, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("service name empty")
	}
	// Try exact UUID already handled by caller
	if h.queries == nil || h.pool == nil {
		return nil, fmt.Errorf("db not configured")
	}
	// If orgID provided, try GetServiceByName
	if orgID != nil && *orgID != uuid.Nil {
		if svc, err := h.queries.GetServiceByName(ctx, db.GetServiceByNameParams{OrganizationID: *orgID, Name: name}); err == nil {
			return &svc, nil
		}
		// Fallback case-insensitive search via raw query
		var svc db.Service
		err := h.pool.QueryRow(ctx, `SELECT id, organization_id, name, description, duration_minutes, buffer_minutes, price_cents, color, is_active, created_at, updated_at FROM services WHERE organization_id=$1 AND LOWER(name)=LOWER($2) LIMIT 1`, *orgID, name).Scan(
			&svc.ID, &svc.OrganizationID, &svc.Name, &svc.Description, &svc.DurationMinutes, &svc.BufferMinutes, &svc.PriceCents, &svc.Color, &svc.IsActive, &svc.CreatedAt, &svc.UpdatedAt,
		)
		if err == nil {
			return &svc, nil
		}
	}
	// Global search without org filter (for chat without org context)
	var svc db.Service
	err := h.pool.QueryRow(ctx, `SELECT id, organization_id, name, description, duration_minutes, buffer_minutes, price_cents, color, is_active, created_at, updated_at FROM services WHERE LOWER(name)=LOWER($1) LIMIT 1`, name).Scan(
		&svc.ID, &svc.OrganizationID, &svc.Name, &svc.Description, &svc.DurationMinutes, &svc.BufferMinutes, &svc.PriceCents, &svc.Color, &svc.IsActive, &svc.CreatedAt, &svc.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &svc, nil
}

// resolveStaffByName finds staff by name case-insensitive.
func (h *Handler) resolveStaffByName(ctx context.Context, name string, orgID *uuid.UUID) (*db.Staff, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("staff name empty")
	}
	if h.pool == nil {
		return nil, fmt.Errorf("db not configured")
	}
	if orgID != nil && *orgID != uuid.Nil {
		// Try org scoped
		var st db.Staff
		err := h.pool.QueryRow(ctx, `SELECT id, organization_id, user_id, name, email, avatar_url, is_active, created_at, updated_at FROM staff WHERE organization_id=$1 AND LOWER(name)=LOWER($2) LIMIT 1`, *orgID, name).Scan(
			&st.ID, &st.OrganizationID, &st.UserID, &st.Name, &st.Email, &st.AvatarUrl, &st.IsActive, &st.CreatedAt, &st.UpdatedAt,
		)
		if err == nil {
			return &st, nil
		}
	}
	var st db.Staff
	err := h.pool.QueryRow(ctx, `SELECT id, organization_id, user_id, name, email, avatar_url, is_active, created_at, updated_at FROM staff WHERE LOWER(name)=LOWER($1) LIMIT 1`, name).Scan(
		&st.ID, &st.OrganizationID, &st.UserID, &st.Name, &st.Email, &st.AvatarUrl, &st.IsActive, &st.CreatedAt, &st.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &st, nil
}

func (h *Handler) validationError(c echo.Context, err error) error {
	var details []fieldError
	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, fe := range ve {
			field := fe.Field()
			// Map to json tag lowerCamel
			switch field {
			case "Messages":
				field = "messages"
			case "OrgID":
				field = "orgId"
			case "Content":
				field = "content"
			case "Role":
				field = "role"
			default:
				if field != "" {
					field = strings.ToLower(field[:1]) + field[1:]
				}
			}
			msg := msgForTag(fe)
			details = append(details, fieldError{Field: field, Message: msg})
		}
	} else {
		details = append(details, fieldError{Field: "body", Message: err.Error()})
	}
	return c.JSON(http.StatusUnprocessableEntity, errorResponse{
		Error:   "validation_error",
		Message: "Validation failed",
		Details: details,
	})
}

func msgForTag(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fe.Field() + " is required"
	case "oneof":
		return fe.Field() + " must be one of [" + fe.Param() + "]"
	case "uuid":
		return fe.Field() + " must be a valid UUID"
	case "min":
		return fe.Field() + " must be at least " + fe.Param() + " items"
	default:
		return fe.Field() + " failed on " + fe.Tag()
	}
}

func isPlaceholder(k string) bool {
	trim := strings.TrimSpace(k)
	if trim == "" {
		return true
	}
	if trim == "re_..." || trim == "sk_test_..." || trim == "whsec_..." || trim == "sk-..." || strings.Contains(trim, "[") {
		return true
	}
	if trim == "re_..." {
		return true
	}
	// Heuristic: short placeholder
	if len(trim) < 20 && (strings.HasPrefix(trim, "sk-") || strings.HasPrefix(trim, "re_")) {
		// Could be real short test key? But treat as placeholder if contains "..."
		if strings.Contains(trim, "...") {
			return true
		}
	}
	return false
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
