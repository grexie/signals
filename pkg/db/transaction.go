package db

import (
	"context"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
)

func WithTransaction(db *mongo.Database, ctx context.Context, callback func(ctx context.Context) (any, error)) (any, error) {
	if os.Getenv("MONGO_SUPPORTS_TRANSACTIONS") == "true" {
		client := db.Client()
		session, err := client.StartSession()
		if err != nil {
			return nil, err
		}
		defer session.EndSession(ctx)

		return session.WithTransaction(ctx, func(ctx mongo.SessionContext) (interface{}, error) {
			return callback(ctx)
		});
	} else {
		return callback(ctx)
	}
}