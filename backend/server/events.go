package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/evanofslack/analogdb"
	v1 "github.com/evanofslack/analogdb/internal/gen/proto/analytics/v1"
	"github.com/evanofslack/analogdb/metrics"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/medama-io/go-useragent"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	eventsRoute        = "/v1/events"
	maxEventsBodyBytes = 32 << 10
	maxEventsPerBatch  = 25
	maxEventProps      = 20
	maxEventString     = maxSearchQueryLen
	maxEventPath       = 512
	maxUserAgent       = 1024
	maxEventClockSkew  = 24 * time.Hour
	saltBytes          = 32
	saltTimeout        = 2 * time.Second
	unknownEventName   = "unknown"

	visitorIPHeader = "X-Analogdb-Visitor-IP"
	visitorUAHeader = "X-Analogdb-Visitor-UA"
)

var botPattern = regexp.MustCompile(`(?i)bot|crawl|spider|headless|lighthouse|preview`)

// EventBatch is a batch of UI events forwarded by the web server
type EventBatch struct {
	Events []EventInput `json:"events"`
}

type EventInput struct {
	ID       string         `json:"id" example:"9b2f6c1e-3f4a-4d6b-9a51-2c7e8f0d1a23"`
	Name     string         `json:"name" example:"page_view"`
	V        int            `json:"v" example:"1"`
	Ts       int64          `json:"ts" example:"1760000000000"`
	Path     string         `json:"path" example:"/post/123"`
	Route    string         `json:"route" example:"/post/[id]"`
	Referrer string         `json:"referrer" example:"news.ycombinator.com"`
	Utm      EventUtm       `json:"utm"`
	Vw       int            `json:"vw" example:"1280"`
	SearchID string         `json:"search_id"`
	PostID   int64          `json:"post_id" example:"123"`
	Props    map[string]any `json:"props"`
}

type EventUtm struct {
	Source   string `json:"source"`
	Medium   string `json:"medium"`
	Campaign string `json:"campaign"`
}

type rawEventBatch struct {
	Events []json.RawMessage `json:"events"`
}

type uiEventStats struct {
	events  *prometheus.CounterVec
	batches *prometheus.CounterVec
}

func newUiEventStats() *uiEventStats {
	return &uiEventStats{
		events: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: metrics.AnalogdbNamespace,
			Name:      "ui_events_total",
			Help:      "Number of UI events received, by name and whether they were accepted or dropped",
		}, []string{"name", "result"}),
		batches: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: metrics.AnalogdbNamespace,
			Name:      "ui_event_batches_total",
			Help:      "Number of UI event batches received, by result",
		}, []string{"result"}),
	}
}

func (stats *uiEventStats) register(registerer prometheus.Registerer) error {
	for _, c := range []prometheus.Collector{stats.events, stats.batches} {
		if err := registerer.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// uiEvents holds the state behind the events route
type uiEvents struct {
	stats  *uiEventStats
	parser *useragent.Parser
	now    func() time.Time

	mu      sync.Mutex
	saltDay string
	salt    []byte
}

func newUiEvents() *uiEvents {
	return &uiEvents{
		stats:  newUiEventStats(),
		parser: useragent.NewParser(),
		now:    time.Now,
	}
}

func (s *Server) mountEventHandlers(r chi.Router) {
	if err := s.ui.stats.register(s.metrics.Registry); err != nil {
		s.logger.Error("Fail register ui event metrics", "error", err)
	}
	r.With(s.require(roleWeb)).Post("/events", s.createEvents)
}

// @Summary Record UI events
// @Description Accept a batch of UI events from the web server. Events that fail validation are dropped and the rest are published (requires authentication)
// @Tags events
// @Accept json
// @Param batch body EventBatch true "Event batch, at most 25 events"
// @Param X-Analogdb-Visitor-IP header string false "Visitor IP, only read for the web role"
// @Param X-Analogdb-Visitor-UA header string false "Visitor user agent, only read for the web role"
// @Success 204
// @Failure 400 {object} analogdb.Error "Invalid batch"
// @Failure 401 {object} analogdb.Error "Unauthorized"
// @Failure 403 {object} analogdb.Error "Forbidden"
// @Failure 413 {object} analogdb.Error "Batch too large"
// @Security BasicAuth
// @Router /events [post]
func (s *Server) createEvents(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxEventsBodyBytes))
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			s.ui.stats.batches.WithLabelValues("too_large").Inc()
			s.writeError(w, r, &analogdb.Error{Code: errCodeTooLarge, Message: "request body too large"})
			return
		}
		s.ui.stats.batches.WithLabelValues("bad_request").Inc()
		s.writeError(w, r, badRequest("read event batch"))
		return
	}

	var batch rawEventBatch
	if err := json.Unmarshal(body, &batch); err != nil {
		s.ui.stats.batches.WithLabelValues("bad_request").Inc()
		s.writeError(w, r, badRequest("invalid event batch"))
		return
	}
	if len(batch.Events) > maxEventsPerBatch {
		s.ui.stats.batches.WithLabelValues("bad_request").Inc()
		s.writeError(w, r, badRequest("too many events, max %d", maxEventsPerBatch))
		return
	}
	s.ui.stats.batches.WithLabelValues("ok").Inc()

	if len(batch.Events) > 0 {
		s.publishEvents(r, batch.Events)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) publishEvents(r *http.Request, raw []json.RawMessage) {
	ctx := r.Context()
	now := s.ui.now()

	ip, ua := visitor(r)
	salt := s.dailySalt(ctx, now)
	visitorID := visitorHash(salt, ip, ua)
	device, browser, osName, bot := s.parseUserAgent(ua)

	for _, msg := range raw {
		event, name, ok := buildEvent(msg, now)
		if !ok {
			s.ui.stats.events.WithLabelValues(name, "dropped").Inc()
			continue
		}
		s.ui.stats.events.WithLabelValues(name, "accepted").Inc()

		event.VisitorId = visitorID
		event.DeviceType = device
		event.Browser = browser
		event.Os = osName
		event.IsBot = bot

		if err := s.UiEventService.Write(context.WithoutCancel(ctx), event); err != nil {
			s.logger.DebugContext(ctx, "Fail write ui event to event stream", "error", err)
		}
	}
}

// visitor returns the visitor IP and user agent. The web server forwards them
// in headers since its own requests carry its IP and user agent
func visitor(r *http.Request) (string, string) {
	ip, ua := getRealIP(r), r.UserAgent()
	if p := principalFrom(r); p != nil && p.role == roleWeb {
		if parsed := net.ParseIP(strings.TrimSpace(r.Header.Get(visitorIPHeader))); parsed != nil {
			ip, ua = parsed.String(), r.Header.Get(visitorUAHeader)
		}
	}
	if len(ua) > maxUserAgent {
		ua = ua[:maxUserAgent]
	}
	return ip, ua
}

func visitorHash(salt []byte, ip, ua string) string {
	h := sha256.New()
	h.Write(salt)
	h.Write([]byte(ip))
	h.Write([]byte{0})
	h.Write([]byte(ua))
	return hex.EncodeToString(h.Sum(nil)[:16])
}

// dailySalt returns the random salt for the UTC day of now. With redis every
// instance shares one salt, without it each process keeps its own
func (s *Server) dailySalt(ctx context.Context, now time.Time) []byte {
	day := now.UTC().Format(time.DateOnly)

	s.ui.mu.Lock()
	defer s.ui.mu.Unlock()
	if s.ui.saltDay == day {
		return s.ui.salt
	}

	salt := make([]byte, saltBytes)
	_, _ = rand.Read(salt)
	if s.SaltService == nil {
		s.logger.InfoContext(ctx, "No cache for the daily salt, using an in process salt", "day", day)
	} else {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), saltTimeout)
		defer cancel()
		stored, err := s.SaltService.DailySalt(ctx, day, salt)
		if err != nil {
			s.logger.ErrorContext(ctx, "Fail get daily salt from cache, using an in process salt", "day", day, "error", err)
		} else {
			salt = stored
		}
	}
	s.ui.saltDay, s.ui.salt = day, salt
	return salt
}

func (s *Server) parseUserAgent(ua string) (device, browser, osName string, bot bool) {
	agent := s.ui.parser.Parse(ua)
	bot = agent.IsBot() || botPattern.MatchString(ua)
	switch {
	case bot:
		device = "bot"
	case agent.IsMobile():
		device = "mobile"
	case agent.IsTablet():
		device = "tablet"
	case agent.IsDesktop():
		device = "desktop"
	default:
		device = "other"
	}
	return device, string(agent.Browser()), string(agent.OS()), bot
}

// buildEvent validates one event against the allowlist. It returns the name
// to count it under and false when the event should be dropped
func buildEvent(msg json.RawMessage, now time.Time) (*v1.UiEvent, string, bool) {
	var in EventInput
	dec := json.NewDecoder(bytes.NewReader(msg))
	dec.UseNumber()
	if err := dec.Decode(&in); err != nil {
		return nil, unknownEventName, false
	}

	schema, ok := eventSchemas[in.Name]
	if !ok {
		return nil, unknownEventName, false
	}
	if in.V != schema.version {
		return nil, in.Name, false
	}

	id, err := uuid.Parse(in.ID)
	if err != nil || len(in.ID) != 36 {
		return nil, in.Name, false
	}

	for _, str := range []string{in.Route, in.Referrer, in.Utm.Source, in.Utm.Medium, in.Utm.Campaign, in.SearchID} {
		if utf8.RuneCountInString(str) > maxEventString {
			return nil, in.Name, false
		}
	}
	if in.Route != "" && !strings.HasPrefix(in.Route, "/") {
		return nil, in.Name, false
	}
	if in.PostID < 0 || in.PostID > math.MaxInt32 {
		return nil, in.Name, false
	}
	if in.Vw < 0 || in.Vw > math.MaxInt32 {
		return nil, in.Name, false
	}
	if !validProps(in.Props, schema.props) {
		return nil, in.Name, false
	}

	props := in.Props
	if props == nil {
		props = map[string]any{}
	}
	propsJSON, err := json.Marshal(props)
	if err != nil {
		return nil, in.Name, false
	}

	received := now.UnixMilli()
	clientTs := in.Ts
	if skew := time.Duration(received-clientTs) * time.Millisecond; skew > maxEventClockSkew || skew < -maxEventClockSkew {
		clientTs = received
	}

	return &v1.UiEvent{
		EventId:       id.String(),
		EventName:     in.Name,
		SchemaVersion: int32(in.V),
		ClientTs:      clientTs,
		ReceivedTs:    received,
		Path:          truncateBytes(in.Path, maxEventPath),
		Route:         in.Route,
		ReferrerHost:  in.Referrer,
		UtmSource:     in.Utm.Source,
		UtmMedium:     in.Utm.Medium,
		UtmCampaign:   in.Utm.Campaign,
		ViewportWidth: int32(in.Vw),
		SearchId:      in.SearchID,
		PostId:        in.PostID,
		PropsJson:     string(propsJSON),
	}, in.Name, true
}

func validProps(props map[string]any, allowed map[string]propKind) bool {
	if len(props) > maxEventProps {
		return false
	}
	for key, value := range props {
		kind, ok := allowed[key]
		if !ok || !validProp(kind, value) {
			return false
		}
	}
	return true
}

func validProp(kind propKind, value any) bool {
	switch kind {
	case propString:
		str, ok := value.(string)
		return ok && utf8.RuneCountInString(str) <= maxEventString
	case propInt:
		num, ok := value.(json.Number)
		if !ok {
			return false
		}
		_, err := num.Int64()
		return err == nil
	case propNumber:
		num, ok := value.(json.Number)
		if !ok {
			return false
		}
		_, err := num.Float64()
		return err == nil
	case propBool:
		_, ok := value.(bool)
		return ok
	case propStringMap:
		m, ok := value.(map[string]any)
		if !ok || len(m) > maxEventProps {
			return false
		}
		for key, v := range m {
			str, ok := v.(string)
			if !ok || utf8.RuneCountInString(key) > maxEventString || utf8.RuneCountInString(str) > maxEventString {
				return false
			}
		}
		return true
	}
	return false
}

// truncateBytes cuts str to at most n bytes without splitting a rune
func truncateBytes(str string, n int) string {
	if len(str) <= n {
		return str
	}
	for n > 0 && !utf8.RuneStart(str[n]) {
		n--
	}
	return str[:n]
}
