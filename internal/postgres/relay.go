package postgres

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

)

func RunRelay(ctx context.Context,store *Store,publisher interface{Publish(context.Context,string,string,[]byte)error},interval time.Duration){
	ticker:=time.NewTicker(interval);defer ticker.Stop()
	for{select{case<-ctx.Done():return;case<-ticker.C:}
		events,err:=store.Pending(ctx,100);if err!=nil{slog.Error("read outbox failed","error",err);continue}
		for _,event:=range events{if err:=publisher.Publish(ctx,event.Topic,event.Key,event.Payload);err!=nil{slog.Error("publish outbox event failed","event_id",event.ID,"error",err);wait:=time.Duration(rand.Int64N(int64(time.Minute)));select{case<-ctx.Done():return;case<-time.After(wait):};break};if err:=store.MarkPublished(ctx,event.ID);err!=nil{slog.Error("mark outbox event failed","event_id",event.ID,"error",err)}}
	}
}

