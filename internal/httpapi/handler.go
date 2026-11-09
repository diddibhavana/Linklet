package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"linklet/internal/application"
	"linklet/internal/domain"
	"linklet/internal/ports"
)

type Handler struct { Service *application.Service; Store ports.Store; Cache ports.Cache; FrontendOrigin string; requests atomic.Uint64; creates atomic.Uint64; created atomic.Uint64; redirects atomic.Uint64 }
type problem struct { Type string `json:"type"`; Title string `json:"title"`; Status int `json:"status"`; Detail string `json:"detail"`; RequestID string `json:"request_id"` }
type requestIDKey struct{}

func New(service *application.Service, store ports.Store, cache ports.Cache, origin string) *Handler { return &Handler{Service:service,Store:store,Cache:cache,FrontendOrigin:origin} }

func (h *Handler) Routes() http.Handler {
	mux:=http.NewServeMux()
	mux.HandleFunc("GET /healthz",func(w http.ResponseWriter,r *http.Request){writeJSON(w,200,map[string]string{"status":"ok"})})
	mux.HandleFunc("GET /readyz",h.ready)
	mux.HandleFunc("GET /metrics",h.metrics)
	mux.HandleFunc("POST /api/v1/links",h.create)
	mux.HandleFunc("GET /api/v1/links",h.list)
	mux.HandleFunc("GET /r/{code}",h.redirect)
	return h.middleware(mux)
}

func (h *Handler) middleware(next http.Handler) http.Handler {
	var inflight atomic.Int64
	return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		id:=r.Header.Get("X-Request-ID");if id==""{id=fmt.Sprintf("req-%d",time.Now().UnixNano())};w.Header().Set("X-Request-ID",id)
		start:=time.Now();inflight.Add(1);defer inflight.Add(-1);h.requests.Add(1)
		defer func(){if recovered:=recover();recovered!=nil{slog.Error("request panic","request_id",id,"error",fmt.Sprint(recovered));writeProblem(w,r,500,"Internal Server Error","The request could not be completed.")};slog.Info("http request","request_id",id,"method",r.Method,"path",r.URL.Path,"duration_ms",time.Since(start).Milliseconds())}()
		ctx:=context.WithValue(r.Context(),requestIDKey{},id)
		r=r.WithContext(ctx)
		origin:=r.Header.Get("Origin")
		if origin!="" && origin==h.FrontendOrigin {
			w.Header().Set("Access-Control-Allow-Origin",origin)
			w.Header().Set("Access-Control-Allow-Methods","GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers","Content-Type, Idempotency-Key, X-Request-ID")
			w.Header().Set("Vary","Origin")
			if r.Method==http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
		}
		next.ServeHTTP(w,r)
	})
}

func (h *Handler) create(w http.ResponseWriter,r *http.Request){
	h.creates.Add(1)
	mediaType,_,mediaErr:=mime.ParseMediaType(r.Header.Get("Content-Type"));if mediaErr!=nil||mediaType!="application/json"{writeProblem(w,r,415,"Unsupported Media Type","Send application/json.");return}
	key:=strings.TrimSpace(r.Header.Get("Idempotency-Key"));if len(key)<8||len(key)>128{writeProblem(w,r,400,"Invalid Request","Idempotency-Key must contain 8 to 128 characters.");return}
	r.Body=http.MaxBytesReader(w,r.Body,4096)
	var input struct{URL string `json:"url"`;Code string `json:"code"`}
	decoder:=json.NewDecoder(r.Body);decoder.DisallowUnknownFields();if err:=decoder.Decode(&input);err!=nil{writeProblem(w,r,400,"Invalid Request","Request body must contain a URL and optional code.");return}
	link,err:=h.Service.Create(r.Context(),input.URL,input.Code,key);if err!=nil{if errors.Is(err,domain.ErrInvalidURL)||errors.Is(err,domain.ErrInvalidCode){writeProblem(w,r,422,"Validation Failed",err.Error());return};if strings.Contains(strings.ToLower(err.Error()),"unique")||strings.Contains(strings.ToLower(err.Error()),"duplicate"){writeProblem(w,r,409,"Conflict","That short code is already in use.");return};slog.Error("create link failed","request_id",requestID(r),"error",err);writeProblem(w,r,503,"Service Unavailable","The link could not be saved.");return}
	writeJSON(w,201,link)
	h.created.Add(1)
}

func (h *Handler) redirect(w http.ResponseWriter,r *http.Request){
	code:=r.PathValue("code");if !domain.ValidateCode(code){writeProblem(w,r,404,"Not Found","No link exists for this code.");return}
	link,err:=h.Service.Resolve(r.Context(),code);if errors.Is(err,domain.ErrNotFound){writeProblem(w,r,404,"Not Found","No link exists for this code.");return};if err!=nil{slog.Error("resolve link failed","request_id",requestID(r),"error",err);writeProblem(w,r,503,"Service Unavailable","The link could not be resolved.");return};h.redirects.Add(1);w.Header().Set("Cache-Control","no-store");http.Redirect(w,r,link.URL,http.StatusFound)
}

func (h *Handler) list(w http.ResponseWriter,r *http.Request){
	limit:=25;if raw:=r.URL.Query().Get("limit");raw!=""{n,err:=strconv.Atoi(raw);if err!=nil||n<1||n>100{writeProblem(w,r,400,"Invalid Request","limit must be between 1 and 100.");return};limit=n}
	var before time.Time;var id int64
	if cursor:=r.URL.Query().Get("cursor");cursor!=""{decoded,err:=base64.RawURLEncoding.DecodeString(cursor);parts:=strings.Split(string(decoded),"|");if err!=nil||len(parts)!=2{writeProblem(w,r,400,"Invalid Request","cursor is invalid.");return};before,err=time.Parse(time.RFC3339Nano,parts[0]);if err!=nil{writeProblem(w,r,400,"Invalid Request","cursor is invalid.");return};id,err=strconv.ParseInt(parts[1],10,64);if err!=nil||id<1{writeProblem(w,r,400,"Invalid Request","cursor is invalid.");return}}
	links,err:=h.Service.List(r.Context(),before,id,limit+1);if err!=nil{slog.Error("list links failed","request_id",requestID(r),"error",err);writeProblem(w,r,503,"Service Unavailable","Links could not be loaded.");return}
	var next string;if len(links)>limit{last:=links[limit-1];next=base64.RawURLEncoding.EncodeToString([]byte(last.CreatedAt.UTC().Format(time.RFC3339Nano)+"|"+strconv.FormatInt(last.ID,10)));links=links[:limit]}
	writeJSON(w,200,map[string]any{"items":links,"next_cursor":next})
}

func (h *Handler) ready(w http.ResponseWriter,r *http.Request){ctx,cancel:=context.WithTimeout(r.Context(),2*time.Second);defer cancel();if err:=h.Store.Ready(ctx);err!=nil{writeProblem(w,r,503,"Not Ready","Database is unavailable.");return};if err:=h.Cache.Ready(ctx);err!=nil{writeProblem(w,r,503,"Not Ready","Cache is unavailable.");return};writeJSON(w,200,map[string]string{"status":"ready"})}
func (h *Handler) metrics(w http.ResponseWriter,r *http.Request){
	w.Header().Set("Content-Type","text/plain; version=0.0.4")
	fmt.Fprintf(w,"linklet_http_requests_total %d\n",h.requests.Load())
	fmt.Fprintf(w,"linklet_create_attempts_total %d\n",h.creates.Load())
	fmt.Fprintf(w,"linklet_links_created_total %d\n",h.created.Load())
	fmt.Fprintf(w,"linklet_redirects_total %d\n",h.redirects.Load())
	if source,ok:=h.Store.(interface{OutboxBacklog(context.Context)(int64,error)});ok { ctx,cancel:=context.WithTimeout(r.Context(),time.Second);defer cancel();if count,err:=source.OutboxBacklog(ctx);err==nil{fmt.Fprintf(w,"linklet_outbox_backlog %d\n",count)} }
}
func requestID(r *http.Request)string{id,_:=r.Context().Value(requestIDKey{}).(string);return id}
func writeJSON(w http.ResponseWriter,status int,value any){w.Header().Set("Content-Type","application/json");w.WriteHeader(status);_ = json.NewEncoder(w).Encode(value)}
func writeProblem(w http.ResponseWriter,r *http.Request,status int,title,detail string){w.Header().Set("Content-Type","application/problem+json");w.WriteHeader(status);_ = json.NewEncoder(w).Encode(problem{Type:"about:blank",Title:title,Status:status,Detail:detail,RequestID:requestID(r)})}
