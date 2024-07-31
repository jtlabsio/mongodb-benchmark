package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.jtlabs.io/settings"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	charset     = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	totalRandos = 10000000
)

var (
	baseIndices = []mongo.IndexModel{
		{
			Keys: bson.D{{"createdAt", 1}},
		},
		{
			Keys:    bson.D{{"email", 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{"favoriteColor", -1}},
		},
		{
			Keys: bson.D{{"updatedAt", -1}},
		},
	}
	baseSchema = bson.M{
		"bsonType": "object",
		"properties": bson.M{
			"createdAt": bson.M{
				"bsonType":    "date",
				"description": "Date the user was created",
			},
			"email": bson.M{
				"bsonType":    "string",
				"description": "Email address of the user",
			},
			"favoriteColor": bson.M{
				"bsonType":    "string",
				"description": "Favorite color of the user",
			},
			"firstName": bson.M{
				"bsonType":    "string",
				"description": "First name of the user",
			},
			"lastName": bson.M{
				"bsonType":    "string",
				"description": "Last name of the user",
			},
			"updatedAt": bson.M{
				"bsonType":    "date",
				"description": "Date the user was last updated",
			},
		},
	}
	collectionRandoBase   = "randoBase"
	collectionRandoCustom = "randoCustom"
	colors                = []string{
		"red",
		"orange",
		"yellow",
		"green",
		"blue",
		"indigo",
		"violet",
	}
	emailHosts = []string{
		"gmail.com",
		"hotmail.com",
		"yahoo.com",
		"outlook.com",
		"icloud.com",
		"protonmail.com",
	}
	l zerolog.Logger
	s Settings
)

type RandoBase struct {
	CreatedAt     time.Time `json:"createdAt" bson:"createdAt"`
	Email         string    `json:"email" bson:"email"`
	FavoriteColor string    `json:"favoriteColor" bson:"favoriteColor"`
	FirstName     string    `json:"firstName" bson:"firstName"`
	ID            string    `json:"randoID" bson:"_id"`
	LastName      string    `json:"lastName" bson:"lastName"`
	UpdatedAt     time.Time `json:"updatedAt" bson:"updatedAt"`
}

type RandoCustom struct {
	CreatedAt     time.Time `json:"createdAt" bson:"createdAt"`
	Email         string    `json:"email" bson:"email"`
	FavoriteColor string    `json:"favoriteColor" bson:"favoriteColor"`
	FirstName     string    `json:"firstName" bson:"firstName"`
	ID            string    `json:"randoID" bson:"randoID"`
	LastName      string    `json:"lastName" bson:"lastName"`
	UpdatedAt     time.Time `json:"updatedAt" bson:"updatedAt"`
}

type Settings struct {
	Data struct {
		Database       string `json:"database" yaml:"database"`
		Host           string `json:"host" yaml:"host"`
		Options        string `json:"options" yaml:"options"`
		Password       string `json:"password" yaml:"password"`
		Protocol       string `json:"protocol" yaml:"protocol"`
		TimeoutSeconds int    `json:"timeoutSeconds" yaml:"timeoutSeconds"`
		Username       string `json:"username" yaml:"username"`
	} `json:"data" yaml:"data"`
	Logging struct {
		Level string `json:"level" yaml:"level"`
	} `json:"logging" yaml:"logging"`
	Populate bool `json:"populate" yaml:"populate"`
}

func (s *Settings) GetMongoURI() string {
	if s.Data.Username == "" || s.Data.Password == "" {
		return fmt.Sprintf(
			"%s://%s/%s?%s",
			s.Data.Protocol,
			s.Data.Host,
			s.Data.Database,
			s.Data.Options)
	}

	return fmt.Sprintf(
		"%s://%s:%s@%s/%s?%s",
		s.Data.Protocol,
		s.Data.Username,
		s.Data.Password,
		s.Data.Host,
		s.Data.Database,
		s.Data.Options)
}

func main() {
	if err := settings.Gather(settings.Options().
		SetBasePath("./settings/defaults.yaml").
		SetEnvOverride("ENV", "GO_ENV").
		SetEnvSearchPaths("./", "./settings").
		SetArgsMap(map[string]string{
			"-p":         "Populate",
			"--populate": "Populate",
		}), &s); err != nil {
		log.Fatal().Err(err).Msg("Failed to gather settings")
	}

	// set logging level
	zerolog.SetGlobalLevel(getLogLevel(s.Logging.Level))

	// make logs pretty
	l = zerolog.New(zerolog.ConsoleWriter{
		FormatFieldName: func(i interface{}) string {
			return fmt.Sprintf("%s:", i)
		},
		FormatLevel: func(i interface{}) string {
			return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
		},
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}).With().Timestamp().Logger()

	l.Trace().Interface("settings", s).Msg("Settings gathered")

	// connect to the database
	l.Info().Str("uri", s.GetMongoURI()).Msg("Connecting to MongoDB")
	mc, err := mongo.Connect(
		context.Background(),
		options.Client().ApplyURI(s.GetMongoURI()))
	if err != nil {
		l.Fatal().Err(err).Msg("Failed to connect to MongoDB")
	}

	// ensure the collections exist
	if err := ensureCollections(mc); err != nil {
		l.Fatal().Err(err).Msg("Failed to ensure collections")
	}

	if s.Populate {
		l.Info().Msg("Populating environment with random data")
		if err := populateData(mc); err != nil {
			l.Fatal().Err(err).Msg("Unable to complete operation")
		}
		l.Info().Msg("Data population complete")
		return
	}
}

func createCollection(ctx context.Context, db *mongo.Database, collection string, schema bson.M, indices []mongo.IndexModel) error {
	// create the collection
	l.Trace().Str("collection", collection).Msg("Creating collection")
	opts := options.CreateCollection().SetValidator(bson.M{"$jsonSchema": schema})
	if err := db.CreateCollection(ctx, collection, opts); err != nil {
		l.Error().Err(err).Str("collection", collection).Msg("Failed to create collection")
		return err
	}
	l.Debug().Str("collection", collection).Msg("Collection created")

	// create indices on the collection
	l.Trace().Str("collection", collection).Msg("Creating indices")
	if _, err := db.Collection(collection).Indexes().CreateMany(ctx, indices); err != nil {
		l.Error().Err(err).Str("collection", collection).Msg("Failed to create indices")
		return err
	}
	l.Debug().Str("collection", collection).Msg("Indices created")

	return nil
}

func ensureCollection(ctx context.Context, mc *mongo.Client, collection string) error {
	db := mc.Database(s.Data.Database)
	collections, err := db.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return err
	}

	for _, c := range collections {
		if c == collection {
			return nil
		}
	}

	switch collection {
	case collectionRandoCustom:
		// index the unique identifer for the user
		altIndices := append(baseIndices, mongo.IndexModel{
			Keys: bson.D{{
				Key:   "randoID",
				Value: 1}},
			Options: options.Index().SetUnique(true),
		})

		// start with base schema, but add a unique identifier for the user
		altSchema := bson.M{}
		for k, v := range baseSchema {
			altSchema[k] = v
		}
		altSchema["properties"].(bson.M)["randoID"] = bson.M{
			"bsonType":    "string",
			"description": "Unique identifier for the user",
		}
		return createCollection(ctx, db, collection, altSchema, altIndices)
	case collectionRandoBase:
		return createCollection(ctx, db, collection, baseSchema, baseIndices)
	default:
		return fmt.Errorf("unknown collection: %s", collection)
	}
}

func ensureCollections(mc *mongo.Client) error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(s.Data.TimeoutSeconds)*time.Second)
	defer cancel()

	collections := []string{collectionRandoBase, collectionRandoCustom}
	for _, collection := range collections {
		if err := ensureCollection(ctx, mc, collection); err != nil {
			return err
		}
	}

	return nil
}

func getLogLevel(level string) zerolog.Level {
	switch level {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}

func newRando(custom bool) any {
	if custom {
		return &RandoCustom{
			CreatedAt:     randomDate(),
			Email:         randomEmail(),
			FavoriteColor: randomColor(),
			FirstName:     randomLengthString(5, 10),
			ID:            newUniqueID(),
			LastName:      randomLengthString(5, 10),
			UpdatedAt:     randomDate(),
		}
	}

	return &RandoBase{
		CreatedAt:     randomDate(),
		Email:         randomEmail(),
		FavoriteColor: randomColor(),
		FirstName:     randomLengthString(5, 10),
		ID:            newUniqueID(),
		LastName:      randomLengthString(5, 10),
		UpdatedAt:     randomDate(),
	}
}

func newUniqueID() string {
	id := uuid.New()
	return strings.ReplaceAll(id.String(), "-", "")
}

func populateData(mc *mongo.Client) error {
	db := mc.Database(s.Data.Database)
	collections := []string{collectionRandoBase, collectionRandoCustom}
	for _, collection := range collections {
		l.Info().
			Str("collection", collection).
			Int("total", totalRandos).
			Msg("Populating collection...")

		start := time.Now()

		for i := 0; i < totalRandos; i++ {
			if _, err := db.
				Collection(collection).
				InsertOne(
					context.Background(),
					newRando(collection == collectionRandoCustom)); err != nil {
				l.Error().Err(err).Int("i", i).Str("collection", collection).Msg("Failed to insert document")
				return err
			}

			// periodic status update (to let folks know it's still running...)
			if i%100000 == 0 {
				l.Debug().
					Str("status", fmt.Sprintf("%.2f", float64(i)/totalRandos*100)).
					Str("collection", collection).Msg("Inserted document")
			}
		}

		l.Info().
			Str("collection", collection).
			Dur("duration", time.Since(start)).
			Msg("Collection populated")
	}

	return nil
}

func randomColor() string {
	return colors[rand.Intn(len(colors))]
}

func randomDate() time.Time {
	return time.Now().Add(time.Duration(-randomInt(0, 365*24)) * time.Hour)
}

func randomEmail() string {
	return fmt.Sprintf(
		"%s@%s",
		randomLengthString(10, 15),
		emailHosts[rand.Intn(len(emailHosts))])
}

func randomInt(min, max int) int {
	return rand.Intn(max-min) + min
}

func randomLengthString(min, max int) string {
	return randomString(randomInt(min, max))
}

func randomString(l int) string {
	b := make([]byte, l)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
