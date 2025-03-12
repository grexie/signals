package db

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func EnsureIndex(db *mongo.Database, ctx context.Context, collectionName string, model mongo.IndexModel) error {
	c := db.Collection(collectionName)

	idxs := c.Indexes()

	v := model.Options.Name
	if v == nil {
		return fmt.Errorf("must provide a name for index")
	}
	expectedName := *v

	cur, err := idxs.List(ctx)
	if err != nil {
		return fmt.Errorf("unable to list indexes: %s", err)
	}

	found := false
	for cur.Next(ctx) {
		var d bson.M

		if err := cur.Decode(&d); err != nil {
			return fmt.Errorf("unable to decode bson index document: %s", err)
		}

		v := d["name"]
		if v != nil && v.(string) == expectedName {
			found = true
			break
		}
	}

	if found {
		return nil
	}

	_, err = idxs.CreateOne(ctx, model)
	return err
}

func ConnectMongo() (*mongo.Database, error) {
	registry := bson.NewRegistry()
	registry.RegisterTypeMapEntry(0x03, reflect.TypeOf(bson.M{}))

	mongoUrl := os.Getenv("MONGO_URL")
	if mongoUrl == "" {
		mongoUrl = "mongodb://localhost:27017/signals"
	}

	uri, err := url.Parse(mongoUrl)
	if err != nil {
		return nil, err
	}

	if client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoUrl).SetRegistry(registry)); err != nil {
		return nil, err
	} else {
		dbName := strings.Trim(uri.Path, "/")
		if dbName == "" {
			dbName = "signals"
		}
		return client.Database(dbName), nil
	}
}
