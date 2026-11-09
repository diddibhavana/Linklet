package application

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"linklet/internal/domain"
)

type fakeStore struct {
	mu sync.Mutex
	created domain.Link
	createErr error
	get domain.Link
	getErr error
	list []domain.Link
	listErr error
	key string
}
func (f *fakeStore) Create(_ context.Context,l domain.Link,key,_ string,_ time.Time)(domain.Link,error){f.mu.Lock();defer f.mu.Unlock();f.created=l;f.key=key;if f.createErr!=nil{return domain.Link{},f.createErr};l.ID=7;return l,nil}
func (f *fakeStore) Get(context.Context,string)(domain.Link,error){return f.get,f.getErr}
func (f *fakeStore) List(context.Context,time.Time,int64,int)([]domain.Link,error){return f.list,f.listErr}
func (f *fakeStore) Ready(context.Context)error{return nil}

type fakeCache struct { mu sync.Mutex; value string; getErr,setErr error; sets int }
func (f *fakeCache) Get(context.Context,string)(string,error){f.mu.Lock();defer f.mu.Unlock();return f.value,f.getErr}
func (f *fakeCache) Set(_ context.Context,_,value string,_ time.Duration)error{f.mu.Lock();defer f.mu.Unlock();f.sets++;f.value=value;return f.setErr}
func (f *fakeCache) Ready(context.Context)error{return nil}

func TestCreateValidatesAndPersists(t *testing.T){
	store:=&fakeStore{};service:=New(store,&fakeCache{});service.code=func()(string,error){return "generated_1",nil};service.clock=func()time.Time{return time.Date(2026,1,2,3,4,5,0,time.UTC)}
	got,err:=service.Create(context.Background()," https://example.com/path ","","idempotency-key")
	if err!=nil{t.Fatal(err)};if got.ID!=7||got.Code!="generated_1"||store.key!="idempotency-key"{t.Fatalf("unexpected create result: %+v",got)}
}

func TestCreateRejectsInvalidInputs(t *testing.T){
	service:=New(&fakeStore{},&fakeCache{})
	for _,tc:=range []struct{url,code string;want error}{{"javascript:alert(1)","",domain.ErrInvalidURL},{"https://example.com","bad code",domain.ErrInvalidCode}}{
		if _,err:=service.Create(context.Background(),tc.url,tc.code,"key");!errors.Is(err,tc.want){t.Errorf("Create(%q,%q) error=%v want %v",tc.url,tc.code,err,tc.want)}
	}
}

func TestResolveUsesCacheAndFallsBack(t *testing.T){
	store:=&fakeStore{get:domain.Link{ID:1,Code:"fromdb",URL:"https://db.example"}};cache:=&fakeCache{value:"https://cache.example"};service:=New(store,cache)
	link,err:=service.Resolve(context.Background(),"cached");if err!=nil||link.URL!="https://cache.example"{t.Fatalf("cache resolve: link=%+v err=%v",link,err)}
	cache.value="";link,err=service.Resolve(context.Background(),"fromdb");if err!=nil||link.URL!="https://db.example"||cache.value!="https://db.example"{t.Fatalf("database fallback: link=%+v err=%v",link,err)}
}

func TestResolveReturnsStoreError(t *testing.T){
	want:=domain.ErrNotFound;service:=New(&fakeStore{getErr:want},&fakeCache{getErr:errors.New("cache unavailable")})
	if _,err:=service.Resolve(context.Background(),"missing");!errors.Is(err,want){t.Fatalf("Resolve error=%v want %v",err,want)}
}

func TestListPassesCursorToStore(t *testing.T){
	want:=[]domain.Link{{Code:"one"}};service:=New(&fakeStore{list:want},&fakeCache{});got,err:=service.List(context.Background(),time.Now(),4,25)
	if err!=nil||len(got)!=1||got[0].Code!="one"{t.Fatalf("List=%+v, %v",got,err)}
}

func TestCreateConcurrentCallsAreSafe(t *testing.T){
	store:=&fakeStore{};service:=New(store,&fakeCache{});service.code=func()(string,error){return "samecode",nil};var wg sync.WaitGroup
	for range 32 {wg.Add(1);go func(){defer wg.Done();_,_ = service.Create(context.Background(),"https://example.com","","key")}()};wg.Wait()
}
