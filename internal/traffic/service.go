package traffic

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"icoo_proxy/internal/api"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const (
	defaultLimit = 100
	keyPrefix    = "request:"
)

type PageResult struct {
	Items           []api.RequestView
	Total           int
	Page            int
	PageSize        int
	ProtocolOptions []string
	TokenStats      api.TokenStatsView
	TotalRequests   int
	SuccessCount    int
	ErrorCount      int
	AverageLatency  int
}

type Service struct {
	db *leveldb.DB
}

func NewService(root string) (*Service, error) {
	storeDir := filepath.Join(root, ".data", "traffic.leveldb")
	if err := os.MkdirAll(filepath.Dir(storeDir), 0o755); err != nil {
		return nil, err
	}
	db, err := leveldb.OpenFile(storeDir, &opt.Options{})
	if err != nil {
		return nil, err
	}
	return &Service{db: db}, nil
}

func (s *Service) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Service) RecordRequest(item api.RequestView) error {
	if s == nil || s.db == nil {
		return nil
	}
	if strings.TrimSpace(item.CreatedAt) == "" {
		item.CreatedAt = time.Now().Format(time.RFC3339)
	}
	if strings.TrimSpace(item.RequestID) == "" {
		item.RequestID = fmt.Sprintf("request-%d", time.Now().UnixNano())
	}

	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	return s.db.Put([]byte(requestKey(item)), data, nil)
}

func (s *Service) ListRecent(limit int) []api.RequestView {
	if s == nil || s.db == nil {
		return nil
	}
	if limit <= 0 {
		limit = defaultLimit
	}

	iter := s.db.NewIterator(util.BytesPrefix([]byte(keyPrefix)), nil)
	defer iter.Release()

	items := make([]api.RequestView, 0, limit)
	for iter.Next() {
		var item api.RequestView
		if err := json.Unmarshal(iter.Value(), &item); err != nil {
			continue
		}
		items = append(items, item)
		if len(items) >= limit {
			break
		}
	}
	return items
}

func (s *Service) TokenStats() api.TokenStatsView {
	if s == nil || s.db == nil {
		return api.TokenStatsView{}
	}

	iter := s.db.NewIterator(util.BytesPrefix([]byte(keyPrefix)), nil)
	defer iter.Release()

	stats := api.TokenStatsView{}
	for iter.Next() {
		var item api.RequestView
		if err := json.Unmarshal(iter.Value(), &item); err != nil {
			continue
		}
		stats.InputTokens += item.InputTokens
		stats.OutputTokens += item.OutputTokens
		stats.TotalTokens += item.TotalTokens
	}
	return stats
}

func (s *Service) QueryPage(filter string, page int, pageSize int) PageResult {
	if s == nil || s.db == nil {
		return PageResult{
			Page:            normalizePage(page),
			PageSize:        normalizePageSize(pageSize),
			ProtocolOptions: []string{"all"},
		}
	}

	page = normalizePage(page)
	pageSize = normalizePageSize(pageSize)
	offset := (page - 1) * pageSize
	filter = normalizeFilter(filter)

	iter := s.db.NewIterator(util.BytesPrefix([]byte(keyPrefix)), nil)
	defer iter.Release()

	result := PageResult{
		Items:           make([]api.RequestView, 0, pageSize),
		Page:            page,
		PageSize:        pageSize,
		ProtocolOptions: []string{"all"},
	}

	seenProtocols := map[string]struct{}{
		"all": {},
	}
	var totalDuration int64

	for iter.Next() {
		var item api.RequestView
		if err := json.Unmarshal(iter.Value(), &item); err != nil {
			continue
		}

		result.TotalRequests++
		totalDuration += item.DurationMS
		result.TokenStats.InputTokens += item.InputTokens
		result.TokenStats.OutputTokens += item.OutputTokens
		result.TokenStats.TotalTokens += item.TotalTokens

		if item.StatusCode > 0 && item.StatusCode < 400 {
			result.SuccessCount++
		}
		if item.StatusCode >= 400 {
			result.ErrorCount++
		}

		appendProtocolOption(&result.ProtocolOptions, seenProtocols, item.Downstream)
		appendProtocolOption(&result.ProtocolOptions, seenProtocols, item.Upstream)

		if !matchesFilter(item, filter) {
			continue
		}

		result.Total++
		if result.Total <= offset {
			continue
		}
		if len(result.Items) >= pageSize {
			continue
		}
		result.Items = append(result.Items, item)
	}

	if result.TotalRequests > 0 {
		result.AverageLatency = int(totalDuration / int64(result.TotalRequests))
	}

	return result
}

func requestKey(item api.RequestView) string {
	createdAt := parseCreatedAt(item.CreatedAt)
	reversed := math.MaxInt64 - createdAt.UnixNano()
	return fmt.Sprintf("%s%020d:%s", keyPrefix, reversed, item.RequestID)
}

func parseCreatedAt(value string) time.Time {
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed
	}
	return time.Now()
}

func normalizePage(page int) int {
	if page <= 0 {
		return 1
	}
	return page
}

func normalizePageSize(pageSize int) int {
	if pageSize <= 0 {
		return 10
	}
	return pageSize
}

func normalizeFilter(filter string) string {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return "all"
	}
	return filter
}

func matchesFilter(item api.RequestView, filter string) bool {
	if filter == "all" {
		return true
	}
	return item.Downstream == filter || item.Upstream == filter
}

func appendProtocolOption(options *[]string, seen map[string]struct{}, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if _, ok := seen[value]; ok {
		return
	}
	seen[value] = struct{}{}
	*options = append(*options, value)
}
