package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
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

	builder "go.jtlabs.io/mongo"
	query "go.jtlabs.io/query"
)

const (
	charset     = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	totalRandos = 10000000
)

var (
	baseIndices = []mongo.IndexModel{
		{
			Keys: bson.D{{
				Key:   "createdAt",
				Value: 1,
			}},
		},
		{
			Keys: bson.D{{
				Key:   "email",
				Value: 1,
			}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{
				Key:   "favoriteColor",
				Value: -1,
			}},
		},
		{
			Keys: bson.D{{
				Key:   "updatedAt",
				Value: -1,
			}},
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
	customSchema = bson.M{
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
			"randoID": bson.M{
				"bsonType":    "string",
				"description": "Unique identifier for the user",
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

type APIResponse[T any] struct {
	Data    []T           `json:"data"`
	Options query.Options `json:"options"`
	Total   int           `json:"total"`
}

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
		Database        string `json:"database" yaml:"database"`
		DefaultPageSize int    `json:"defaultPageSize" yaml:"defaultPageSize"`
		Host            string `json:"host" yaml:"host"`
		MaxPageSize     int    `json:"maxPageSize" yaml:"maxPageSize"`
		Options         string `json:"options" yaml:"options"`
		Password        string `json:"password" yaml:"password"`
		Protocol        string `json:"protocol" yaml:"protocol"`
		TimeoutSeconds  int    `json:"timeoutSeconds" yaml:"timeoutSeconds"`
		Username        string `json:"username" yaml:"username"`
	} `json:"data" yaml:"data"`
	Logging struct {
		Level string `json:"level" yaml:"level"`
	} `json:"logging" yaml:"logging"`
	Populate bool `json:"populate" yaml:"populate"`
	Server   struct {
		Address string `json:"address" yaml:"address"`
	} `json:"server" yaml:"server"`
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
	// gather settings from environment
	gatherSettings()

	// setup the logger
	l = configureLogger()

	// log the settings
	l.Trace().Interface("settings", s).Msg("Settings gathered")

	// connect to MongoDB
	mc, err := connectToMongo()
	if err != nil {
		l.Fatal().Err(err).Msg("Failed to connect to MongoDB")
	}

	// ensure the collections exist
	if err := ensureCollections(mc); err != nil {
		l.Fatal().Err(err).Msg("Failed to ensure collections")
	}

	// populate the collections with random data if requested
	if s.Populate {
		l.Info().Msg("Populating environment with random data")
		if err := populateData(mc); err != nil {
			l.Fatal().Err(err).Msg("Unable to complete operation")
		}
		l.Info().Msg("Data population complete")
		return
	}

	// start the server
	startServer(mc)
}

func configureLogger() zerolog.Logger {
	// set logging level
	zerolog.SetGlobalLevel(getLogLevel(s.Logging.Level))

	// make logs pretty
	return zerolog.New(zerolog.ConsoleWriter{
		FormatFieldName: func(i interface{}) string {
			return fmt.Sprintf("%s:", i)
		},
		FormatLevel: func(i interface{}) string {
			return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
		},
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}).With().Timestamp().Logger()
}

func connectToMongo() (*mongo.Client, error) {
	// connect to the database
	l.Info().Str("uri", s.GetMongoURI()).Msg("Connecting to MongoDB")
	return mongo.Connect(
		context.Background(),
		options.Client().ApplyURI(s.GetMongoURI()))
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
		customIndices := append(baseIndices, mongo.IndexModel{
			Keys: bson.D{{
				Key:   "randoID",
				Value: 1}},
			Options: options.Index().SetUnique(true),
		})

		return createCollection(ctx, db, collection, customSchema, customIndices)
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

func gatherSettings() {
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

func handleSearch[T any](qb *builder.QueryBuilder, col *mongo.Collection) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// parse the query parameters
		opts, err := query.FromQuerystring(r.URL.RawQuery)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// set default page details if not provided
		if len(opts.Page) == 0 {
			opts.Page = map[string]int{
				"limit":  s.Data.DefaultPageSize,
				"offset": 0,
			}
		}

		// set default limit if not provided
		if _, ok := opts.Page["limit"]; !ok {
			opts.Page["limit"] = s.Data.DefaultPageSize
		}

		// set default offset if not provided
		if _, ok := opts.Page["offset"]; !ok {
			opts.Page["offset"] = 0
		}

		// set limit to max if too large
		if ps := opts.Page["limit"]; ps > s.Data.MaxPageSize {
			opts.Page["limit"] = s.Data.MaxPageSize
		}

		// build the query filter
		f, err := qb.Filter(opts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// build the query options
		o, err := qb.FindOptions(opts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// allow disk use
		o = o.SetAllowDiskUse(true)

		// time the query
		fs := time.Now()

		// create a context with a timeout
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.Data.TimeoutSeconds)*time.Second)
		defer cancel()

		// execute the query
		l.Trace().
			Str("collection", col.Name()).
			Msg("Beginning MongoDB query")
		cur, err := col.Find(ctx, f, o)
		if err != nil {
			l.Error().
				Err(err).
				Str("collection", col.Name()).
				Msg("Failed to execute query")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// defer closing the cursor
		defer func() {
			if err := cur.Close(ctx); err != nil {
				l.Error().Err(err).Msg("Failed to close cursor")
			}
		}()

		l.Debug().
			Dur("duration", time.Since(fs)).
			Str("collection", col.Name()).
			Msg("MongoDB query complete")

		// collect results
		ar := APIResponse[T]{
			Data:    make([]T, 0),
			Options: opts,
		}
		ss := time.Now()
		if err := cur.All(ctx, &ar.Data); err != nil {
			l.Error().
				Err(err).
				Str("collection", col.Name()).
				Msg("Failed to read cursor")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		l.Debug().
			Dur("duration", time.Since(ss)).
			Str("collection", col.Name()).
			Msg("Cursor read / data serialization complete")

		// get the document count
		var count func() (int, error)
		if len(f) == 0 {
			count = func() (int, error) {
				c, err := col.EstimatedDocumentCount(ctx)
				return int(c), err
			}
		}

		if count == nil {
			count = func() (int, error) {
				c, err := col.CountDocuments(ctx, f)
				return int(c), err
			}
		}

		// run the count query
		l.Trace().
			Str("collection", col.Name()).
			Msg("Looking up total count")
		cs := time.Now()
		ar.Total, err = count()
		if err != nil {
			l.Error().
				Err(err).
				Str("collection", col.Name()).
				Msg("Failed to lookup count")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		l.Debug().
			Dur("duration", time.Since(cs)).
			Str("collection", col.Name()).
			Msg("Completed count lookup")

		// return the results
		l.Info().
			Str("method", r.Method).
			Str("url", r.URL.String()).
			Dur("duration", time.Since(fs)).
			Msg("Request complete")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Query-Duration", time.Since(fs).String())
		json.NewEncoder(w).Encode(ar)
	}
}

func newRando(custom bool) any {
	createdAt := randomDate()
	updatedAt := randomDate(createdAt)

	if custom {
		return &RandoCustom{
			CreatedAt:     createdAt,
			Email:         randomEmail(),
			FavoriteColor: randomColor(),
			FirstName:     randomLengthString(5, 10),
			ID:            newUniqueID(),
			LastName:      randomLengthString(5, 10),
			UpdatedAt:     updatedAt,
		}
	}

	return &RandoBase{
		CreatedAt:     createdAt,
		Email:         randomEmail(),
		FavoriteColor: randomColor(),
		FirstName:     randomLengthString(5, 10),
		ID:            newUniqueID(),
		LastName:      randomLengthString(5, 10),
		UpdatedAt:     updatedAt,
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

func randomDate(after ...time.Time) time.Time {
	// if an after date is provided, use it to generate a random date AFTER the provided date
	if len(after) > 0 {
		// return a random date after the provided date but before now
		from := after[0].UnixMilli()
		to := time.Now().UnixMilli()
		return time.UnixMilli(int64(randomInt(int(from), int(to))))
	}

	return time.Now().Add(time.Duration(-randomInt(0, 365*24)) * time.Hour)
}

func randomEmail() string {
	return fmt.Sprintf(
		"%s@%s",
		randomLengthString(10, 15),
		emailHosts[rand.Intn(len(emailHosts))])
}

func randomInt(min, max int) int {
	// if they are the same, just return the min
	if min == max {
		return min
	}

	// swap values to ensure no negative numbers
	if min > max {
		min, max = max, min
	}

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

func startServer(mc *mongo.Client) {
	// create a query builder for finding documents
	qbB := builder.NewQueryBuilder(collectionRandoBase, baseSchema)
	qbC := builder.NewQueryBuilder(collectionRandoCustom, customSchema)

	// specify mongoDB collections
	cB := mc.Database(s.Data.Database).Collection(collectionRandoBase)
	cC := mc.Database(s.Data.Database).Collection(collectionRandoCustom)

	http.HandleFunc("/v0/randos", handleSearch[RandoBase](qbB, cB))
	http.HandleFunc("/v1/randos", handleSearch[RandoCustom](qbC, cC))

	l.Info().Str("addr", s.Server.Address).Msg("Starting HTTP server")
	log.Fatal().
		Err(http.ListenAndServe(s.Server.Address, nil)).
		Msg("Failed to start HTTP server")
}
