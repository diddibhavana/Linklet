package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
	"linklet/internal/application"
	"linklet/internal/httpapi"
	"linklet/internal/kafkapub"
	"linklet/internal/postgres"
	"linklet/internal/redisstore"
)

type config struct{ databaseURL,redisURL,kafkaBrokers,listen,frontendOrigin string; maxOpen,maxIdle int }
func required(name string)string{value:=strings.TrimSpace(os.Getenv(name));if value==""{slog.Error("required environment variable is missing","name",name);os.Exit(2)};return value}
func loadConfig()config{return config{databaseURL:required("DATABASE_URL"),redisURL:required("REDIS_URL"),kafkaBrokers:required("KAFKA_BROKERS"),listen:env("HTTP_ADDR",":8080"),frontendOrigin:required("FRONTEND_ORIGIN"),maxOpen:integer("DB_MAX_OPEN",10),maxIdle:integer("DB_MAX_IDLE",5)}}
func env(k,d string)string{if v:=os.Getenv(k);v!=""{return v};return d}
func integer(k string,d int)int{v,err:=strconv.Atoi(env(k,strconv.Itoa(d)));if err!=nil||v<1{slog.Error("invalid positive integer environment variable","name",k);os.Exit(2)};return v}

func main(){
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout,&slog.HandlerOptions{Level:slog.LevelInfo})))
	cfg:=loadConfig()
	db,err:=sql.Open("pgx",cfg.databaseURL);if err!=nil{fatal(err)};defer db.Close()
	db.SetMaxOpenConns(cfg.maxOpen);db.SetMaxIdleConns(cfg.maxIdle);db.SetConnMaxLifetime(5*time.Minute);db.SetConnMaxIdleTime(1*time.Minute)
	startupCtx,cancel:=context.WithTimeout(context.Background(),30*time.Second);defer cancel()
	if err=db.PingContext(startupCtx);err!=nil{fatal(err)}
	if err=postgres.Migrate(startupCtx,db);err!=nil{fatal(err)}
	redisOpts,err:=redis.ParseURL(cfg.redisURL);if err!=nil{fatal(err)};redisClient:=redis.NewClient(redisOpts);defer redisClient.Close()
	store:=&postgres.Store{DB:db};cache:=&redisstore.Cache{Client:redisClient};service:=application.New(store,cache);handler:=httpapi.New(service,store,cache,cfg.frontendOrigin)
	publisher:=kafkapub.New(strings.Split(cfg.kafkaBrokers,","));defer publisher.Close()
	rootCtx,stop:=signal.NotifyContext(context.Background(),syscall.SIGINT,syscall.SIGTERM);defer stop()
	go postgres.RunRelay(rootCtx,store,publisher,2*time.Second)
	server:=&http.Server{Addr:cfg.listen,Handler:handler.Routes(),ReadHeaderTimeout:5*time.Second,ReadTimeout:10*time.Second,WriteTimeout:15*time.Second,IdleTimeout:60*time.Second}
	go func(){slog.Info("http server starting","address",cfg.listen);if err:=server.ListenAndServe();err!=nil&&!errors.Is(err,http.ErrServerClosed){slog.Error("http server failed","error",err);stop()}}()
	<-rootCtx.Done();shutdownCtx,shutdownCancel:=context.WithTimeout(context.Background(),15*time.Second);defer shutdownCancel();if err:=server.Shutdown(shutdownCtx);err!=nil{slog.Error("http shutdown failed","error",err)}
}
func fatal(err error){slog.Error("startup failed","error",err);os.Exit(1)}

