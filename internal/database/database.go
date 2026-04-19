package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/joinids/bot/internal/config"
)

type Account struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Type        string             `bson:"type"`
	SessionStr  string             `bson:"session_string"`
	Phone       string             `bson:"phone"`
	TelegramUID int64              `bson:"telegram_uid"`
	AddedBy     int64              `bson:"added_by"`
}

type DBChannel struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Name      string             `bson:"name"`
	Username  string             `bson:"username"`
	IsPrivate bool               `bson:"is_private"`
}

type DB struct {
	client   *mongo.Client
	accounts *mongo.Collection
	users    *mongo.Collection
	sudoers  *mongo.Collection
	channels *mongo.Collection
	settings *mongo.Collection
}

var Instance *DB

func Connect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.C.MongoURI))
	if err != nil {
		return err
	}

	if err = client.Ping(ctx, nil); err != nil {
		return err
	}

	db := client.Database("joinids_bot")
	Instance = &DB{
		client:   client,
		accounts: db.Collection("accounts"),
		users:    db.Collection("users"),
		sudoers:  db.Collection("sudoers"),
		channels: db.Collection("db_channels"),
		settings: db.Collection("settings"),
	}

	Instance.ensureIndexes()
	return nil
}

func (d *DB) ensureIndexes() {
	ctx := context.Background()
	d.users.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.M{"user_id": 1},
		Options: options.Index().SetUnique(true),
	})
	d.sudoers.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.M{"user_id": 1},
		Options: options.Index().SetUnique(true),
	})
}

func ctx() context.Context {
	c, _ := context.WithTimeout(context.Background(), 10*time.Second)
	return c
}

func (d *DB) AddUser(userID int64) error {
	_, err := d.users.UpdateOne(
		ctx(),
		bson.M{"user_id": userID},
		bson.M{"$setOnInsert": bson.M{"user_id": userID}},
		options.Update().SetUpsert(true),
	)
	return err
}

func (d *DB) GetAllUsers() ([]int64, error) {
	cur, err := d.users.Find(ctx(), bson.M{}, options.Find().SetProjection(bson.M{"user_id": 1}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(context.Background())

	var ids []int64
	for cur.Next(context.Background()) {
		var doc struct {
			UserID int64 `bson:"user_id"`
		}
		if err := cur.Decode(&doc); err == nil {
			ids = append(ids, doc.UserID)
		}
	}
	return ids, nil
}

func (d *DB) AddAccount(acc Account) error {
	_, err := d.accounts.InsertOne(ctx(), acc)
	return err
}

func (d *DB) GetAllAccounts() ([]Account, error) {
	cur, err := d.accounts.Find(ctx(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(context.Background())

	var accounts []Account
	if err := cur.All(context.Background(), &accounts); err != nil {
		return nil, err
	}
	return accounts, nil
}

func (d *DB) DeleteAccount(id primitive.ObjectID) error {
	_, err := d.accounts.DeleteOne(ctx(), bson.M{"_id": id})
	return err
}

func (d *DB) AddSudoer(userID int64) error {
	_, err := d.sudoers.UpdateOne(
		ctx(),
		bson.M{"user_id": userID},
		bson.M{"$setOnInsert": bson.M{"user_id": userID}},
		options.Update().SetUpsert(true),
	)
	return err
}

func (d *DB) RemoveSudoer(userID int64) error {
	_, err := d.sudoers.DeleteOne(ctx(), bson.M{"user_id": userID})
	return err
}

func (d *DB) GetSudoers() ([]int64, error) {
	cur, err := d.sudoers.Find(ctx(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(context.Background())

	var ids []int64
	for cur.Next(context.Background()) {
		var doc struct {
			UserID int64 `bson:"user_id"`
		}
		if err := cur.Decode(&doc); err == nil {
			ids = append(ids, doc.UserID)
		}
	}
	return ids, nil
}

func (d *DB) IsSudoer(userID int64) (bool, error) {
	count, err := d.sudoers.CountDocuments(ctx(), bson.M{"user_id": userID})
	return count > 0, err
}

func (d *DB) AddDBChannel(ch DBChannel) error {
	_, err := d.channels.InsertOne(ctx(), ch)
	return err
}

func (d *DB) GetAllDBChannels() ([]DBChannel, error) {
	cur, err := d.channels.Find(ctx(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(context.Background())

	var channels []DBChannel
	if err := cur.All(context.Background(), &channels); err != nil {
		return nil, err
	}
	return channels, nil
}

func (d *DB) DeleteDBChannel(id primitive.ObjectID) error {
	_, err := d.channels.DeleteOne(ctx(), bson.M{"_id": id})
	return err
}

func (d *DB) GetSetting(key string) (interface{}, error) {
	var doc struct {
		Value interface{} `bson:"value"`
	}
	err := d.settings.FindOne(ctx(), bson.M{"key": key}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return doc.Value, err
}

func (d *DB) SetSetting(key string, value interface{}) error {
	_, err := d.settings.UpdateOne(
		ctx(),
		bson.M{"key": key},
		bson.M{"$set": bson.M{"key": key, "value": value}},
		options.Update().SetUpsert(true),
	)
	return err
}
