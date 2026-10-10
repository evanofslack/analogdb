package redis

import (
	"context"
	"errors"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	rediscache "github.com/go-redis/cache/v9"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/evanofslack/analogdb"
	"github.com/evanofslack/analogdb/logger"
	"github.com/evanofslack/analogdb/metrics"
)

type fakePostService struct {
	mu        sync.Mutex
	posts     map[int]*analogdb.Post
	findCalls atomic.Int64
	idCalls   atomic.Int64
	// called by find methods after reading, used to hold a lookup open
	afterRead func()
}

func newFakePostService(ids ...int) *fakePostService {
	posts := map[int]*analogdb.Post{}
	for _, id := range ids {
		p := &analogdb.Post{Id: id}
		p.Score = id * 10
		posts[id] = p
	}
	return &fakePostService{posts: posts}
}

func (f *fakePostService) FindPosts(ctx context.Context, filter *analogdb.PostFilter) ([]*analogdb.Post, int, error) {
	f.findCalls.Add(1)
	f.mu.Lock()
	var posts []*analogdb.Post
	for id, p := range f.posts {
		if filter.IDs != nil && !slices.Contains(*filter.IDs, id) {
			continue
		}
		post := *p
		posts = append(posts, &post)
	}
	f.mu.Unlock()
	sort.Slice(posts, func(i, j int) bool { return posts[i].Id < posts[j].Id })
	count := len(posts)
	if filter.Limit != nil && len(posts) > *filter.Limit {
		posts = posts[:*filter.Limit]
	}
	if f.afterRead != nil {
		f.afterRead()
	}
	return posts, count, nil
}

func (f *fakePostService) FindPostByID(ctx context.Context, id int) (*analogdb.Post, error) {
	f.idCalls.Add(1)
	f.mu.Lock()
	p, ok := f.posts[id]
	var post analogdb.Post
	if ok {
		post = *p
	}
	f.mu.Unlock()
	if f.afterRead != nil {
		f.afterRead()
	}
	if !ok {
		return nil, &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: "post not found"}
	}
	return &post, nil
}

func (f *fakePostService) CreatePost(ctx context.Context, post *analogdb.CreatePost) (*analogdb.Post, error) {
	return nil, errors.New("not implemented")
}

func (f *fakePostService) PatchPost(ctx context.Context, patch *analogdb.PatchPost, id int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.posts[id]
	if !ok {
		return &analogdb.Error{Code: analogdb.ERRNOTFOUND, Message: "post not found"}
	}
	if patch.Score != nil {
		post := *p
		post.Score = *patch.Score
		f.posts[id] = &post
	}
	return nil
}

func (f *fakePostService) DeletePost(ctx context.Context, id int, reason string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.posts, id)
	return nil
}

func (f *fakePostService) AllPostIDs(ctx context.Context) ([]int, error) {
	return nil, nil
}

type fakeSimilarityService struct {
	similar map[int][]int
	calls   atomic.Int64
}

func (f *fakeSimilarityService) CreateSchemas(ctx context.Context) error { return nil }

func (f *fakeSimilarityService) EncodePost(ctx context.Context, id int) error { return nil }

func (f *fakeSimilarityService) BatchEncodePosts(ctx context.Context, ids []int, batchSize int) ([]int, error) {
	return nil, nil
}

func (f *fakeSimilarityService) FindSimilarPosts(ctx context.Context, filter *analogdb.PostSimilarityFilter) ([]*analogdb.Post, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeSimilarityService) FindSimilarPostIDs(ctx context.Context, filter *analogdb.PostSimilarityFilter) ([]int, error) {
	f.calls.Add(1)
	return f.similar[*filter.ID], nil
}

func (f *fakeSimilarityService) DeletePost(ctx context.Context, id int) error { return nil }

func newTestRDB(t *testing.T, url string) *RDB {
	t.Helper()
	log, err := logger.New("error", "debug", "analogdb_test")
	if err != nil {
		t.Fatal(err)
	}
	m, err := metrics.New(log)
	if err != nil {
		t.Fatal(err)
	}
	rdb, err := NewRDB(url, log, m, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rdb.Close() })
	return rdb
}

func newMiniRDB(t *testing.T) (*RDB, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	return newTestRDB(t, "redis://"+mr.Addr()), mr
}

func postsFilter(limit int) *analogdb.PostFilter {
	return analogdb.NewPostFilter(&limit, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
}

func postIDs(posts []*analogdb.Post) []int {
	ids := []int{}
	for _, p := range posts {
		ids = append(ids, p.Id)
	}
	return ids
}

func testPatchRefreshesCache(t *testing.T, rdb *RDB) {
	ctx := context.Background()
	db := newFakePostService(1, 2)
	s := NewCachePostService(rdb, db)

	// warm both caches
	for range 2 {
		if post, err := s.FindPostByID(ctx, 1); err != nil || post.Score != 10 {
			t.Fatalf("find post: %v %v", post, err)
		}
		if posts, _, err := s.FindPosts(ctx, postsFilter(10)); err != nil || posts[0].Score != 10 {
			t.Fatalf("find posts: %v %v", posts, err)
		}
	}
	if got := db.idCalls.Load(); got != 1 {
		t.Fatalf("expected post by id to be cached, got %d db calls", got)
	}
	if got := db.findCalls.Load(); got != 1 {
		t.Fatalf("expected posts to be cached, got %d db calls", got)
	}

	score := 99
	if err := s.PatchPost(ctx, &analogdb.PatchPost{Score: &score}, 1); err != nil {
		t.Fatal(err)
	}

	post, err := s.FindPostByID(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if post.Score != score {
		t.Errorf("post by id score = %d, want %d", post.Score, score)
	}
	posts, _, err := s.FindPosts(ctx, postsFilter(10))
	if err != nil {
		t.Fatal(err)
	}
	if posts[0].Score != score {
		t.Errorf("posts score = %d, want %d", posts[0].Score, score)
	}
}

func TestPatchRefreshesCache(t *testing.T) {
	rdb, _ := newMiniRDB(t)
	testPatchRefreshesCache(t, rdb)
}

func TestDeleteRemovesFromListsAndSimilar(t *testing.T) {
	ctx := context.Background()
	rdb, _ := newMiniRDB(t)
	db := newFakePostService(1, 2, 3)
	posts := NewCachePostService(rdb, db)
	vec := &fakeSimilarityService{similar: map[int][]int{1: {3, 2}}}
	similar := NewCacheSimilarityService(rdb, vec, posts)

	id := 1
	filter := &analogdb.PostSimilarityFilter{ID: &id}
	got, err := similar.FindSimilarPosts(ctx, filter)
	if err != nil {
		t.Fatal(err)
	}
	if ids := postIDs(got); !slices.Equal(ids, []int{3, 2}) {
		t.Fatalf("similar ids = %v, want [3 2]", ids)
	}
	if _, _, err := posts.FindPosts(ctx, postsFilter(10)); err != nil {
		t.Fatal(err)
	}

	if err := posts.DeletePost(ctx, 3, ""); err != nil {
		t.Fatal(err)
	}

	list, count, err := posts.FindPosts(ctx, postsFilter(10))
	if err != nil {
		t.Fatal(err)
	}
	if ids := postIDs(list); !slices.Equal(ids, []int{1, 2}) || count != 2 {
		t.Errorf("posts = %v count %d, want [1 2] count 2", ids, count)
	}

	got, err = similar.FindSimilarPosts(ctx, filter)
	if err != nil {
		t.Fatal(err)
	}
	if ids := postIDs(got); !slices.Equal(ids, []int{2}) {
		t.Errorf("similar ids = %v, want [2]", ids)
	}
	if calls := vec.calls.Load(); calls != 1 {
		t.Errorf("expected similar ids to be cached, got %d calls", calls)
	}
}

func TestReadDuringPatchDoesNotCacheOldPost(t *testing.T) {
	ctx := context.Background()
	rdb, _ := newMiniRDB(t)
	db := newFakePostService(1)
	s := NewCachePostService(rdb, db)

	read := make(chan struct{})
	release := make(chan struct{})
	db.afterRead = func() {
		read <- struct{}{}
		<-release
	}

	done := make(chan *analogdb.Post)
	go func() {
		post, err := s.FindPostByID(ctx, 1)
		if err != nil {
			t.Error(err)
		}
		done <- post
	}()

	// the reader has the old post, now patch before it can cache it
	<-read
	db.afterRead = nil
	score := 99
	if err := s.PatchPost(ctx, &analogdb.PatchPost{Score: &score}, 1); err != nil {
		t.Fatal(err)
	}
	close(release)
	if old := <-done; old.Score != 10 {
		t.Fatalf("expected reader to see the old post, got score %d", old.Score)
	}

	post, err := s.FindPostByID(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if post.Score != score {
		t.Errorf("score = %d, want %d", post.Score, score)
	}
}

func TestConcurrentReadsDuringPatch(t *testing.T) {
	ctx := context.Background()
	rdb, _ := newMiniRDB(t)
	db := newFakePostService(1)
	s := NewCachePostService(rdb, db)

	var wg sync.WaitGroup
	stop := make(chan struct{})
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				if _, err := s.FindPostByID(ctx, 1); err != nil {
					t.Error(err)
					return
				}
				if _, _, err := s.FindPosts(ctx, postsFilter(10)); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}

	score := 99
	if err := s.PatchPost(ctx, &analogdb.PatchPost{Score: &score}, 1); err != nil {
		t.Fatal(err)
	}
	close(stop)
	wg.Wait()

	post, err := s.FindPostByID(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if post.Score != score {
		t.Errorf("score = %d, want %d", post.Score, score)
	}
}

func TestDecodeErrorIsMiss(t *testing.T) {
	ctx := context.Background()
	rdb, mr := newMiniRDB(t)
	cache := rdb.NewCache("test", 10, time.Hour)

	if err := mr.Set("key", "not msgpack"); err != nil {
		t.Fatal(err)
	}

	var value []int
	err := cache.get(ctx, "key", &value)
	if !errors.Is(err, rediscache.ErrCacheMiss) {
		t.Fatalf("expected a miss, got %v", err)
	}
	if mr.Exists("key") {
		t.Error("expected the key to be deleted")
	}
	if got := cache.stats.getErrors(errorKindDecode); got != 1 {
		t.Errorf("decode errors = %d, want 1", got)
	}
}

func TestConcurrentMissesShareOneLoad(t *testing.T) {
	ctx := context.Background()
	rdb, _ := newMiniRDB(t)
	db := newFakePostService(1, 2)
	s := NewCachePostService(rdb, db)

	release := make(chan struct{})
	db.afterRead = func() { <-release }

	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, _, err := s.FindPosts(ctx, postsFilter(10)); err != nil {
				t.Error(err)
			}
		}()
	}

	for db.findCalls.Load() == 0 {
		time.Sleep(time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	if got := db.findCalls.Load(); got != 1 {
		t.Errorf("db calls = %d, want 1", got)
	}
}

func TestGenIncrFailureKeepsWrite(t *testing.T) {
	ctx := context.Background()
	rdb, mr := newMiniRDB(t)
	db := newFakePostService(1)
	s := NewCachePostService(rdb, db)

	mr.Close()

	score := 99
	if err := s.PatchPost(ctx, &analogdb.PatchPost{Score: &score}, 1); err != nil {
		t.Fatalf("expected write to succeed, got %v", err)
	}
	if got := testutil.ToFloat64(rdb.stats.incrErrors); got != 3 {
		t.Errorf("gen incr errors = %v, want 3", got)
	}
	post, err := s.FindPostByID(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if post.Score != score {
		t.Errorf("score = %d, want %d", post.Score, score)
	}
}

func TestCountSharedAcrossPages(t *testing.T) {
	ctx := context.Background()
	rdb, mr := newMiniRDB(t)
	db := newFakePostService(1, 2, 3)
	s := NewCachePostService(rdb, db)

	first := postsFilter(2)
	next := postsFilter(3)
	random := analogdb.PostSortRandom
	seed := 7
	next.Sort = &random
	next.Seed = &seed
	next.Cursor = &analogdb.Cursor{Hash: "8f", ID: 2}

	for _, f := range []*analogdb.PostFilter{first, next} {
		if _, count, err := s.FindPosts(ctx, f); err != nil || count != 3 {
			t.Fatalf("count = %d err %v, want 3", count, err)
		}
	}

	var countKeys, postKeys int
	for _, k := range mr.Keys() {
		switch {
		case strings.HasPrefix(k, "count:"):
			countKeys++
		case strings.HasPrefix(k, "gen:"):
		default:
			postKeys++
		}
	}
	if countKeys != 1 || postKeys != 2 {
		t.Errorf("count keys = %d post keys = %d, want 1 and 2", countKeys, postKeys)
	}
}

func TestGenerationMemo(t *testing.T) {
	ctx := context.Background()
	rdb, mr := newMiniRDB(t)
	now := time.Unix(1000, 0)
	rdb.now = func() time.Time { return now }

	if g := rdb.gen(ctx, postsEntity); g != 0 {
		t.Fatalf("gen = %d, want 0", g)
	}

	// another instance writes
	if _, err := mr.Incr(genKey(postsEntity), 1); err != nil {
		t.Fatal(err)
	}
	if g := rdb.gen(ctx, postsEntity); g != 0 {
		t.Errorf("gen within memo ttl = %d, want 0", g)
	}
	now = now.Add(genMemoTTL)
	if g := rdb.gen(ctx, postsEntity); g != 1 {
		t.Errorf("gen after memo ttl = %d, want 1", g)
	}

	// a local write is seen right away
	rdb.bumpGen(ctx, postsEntity)
	if g := rdb.gen(ctx, postsEntity); g != 2 {
		t.Errorf("gen after local write = %d, want 2", g)
	}
	if got := testutil.ToFloat64(rdb.stats.invalidations.WithLabelValues(postsEntity)); got != 1 {
		t.Errorf("invalidations = %v, want 1", got)
	}
}

func TestSimilarNeverReturnsDuplicates(t *testing.T) {
	ctx := context.Background()
	rdb, _ := newMiniRDB(t)
	db := newFakePostService(1, 2, 3, 4)
	posts := NewCachePostService(rdb, db)
	vec := &fakeSimilarityService{similar: map[int][]int{1: {3, 2, 3, 4, 2, 3}}}
	similar := NewCacheSimilarityService(rdb, vec, posts)

	id := 1
	filter := &analogdb.PostSimilarityFilter{ID: &id}
	for range 2 {
		ids, err := similar.FindSimilarPostIDs(ctx, filter)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(ids, []int{3, 2, 4}) {
			t.Errorf("similar ids = %v, want [3 2 4]", ids)
		}

		got, err := similar.FindSimilarPosts(ctx, filter)
		if err != nil {
			t.Fatal(err)
		}
		if ids := postIDs(got); !slices.Equal(ids, []int{3, 2, 4}) {
			t.Errorf("similar posts = %v, want [3 2 4]", ids)
		}
	}
	if calls := vec.calls.Load(); calls != 1 {
		t.Errorf("expected similar ids to be cached, got %d calls", calls)
	}
}
