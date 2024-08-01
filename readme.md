# MongoDB Index Benchmark Testing

Quick and simple Go application to test the performance of two different MongoDB index approaches for the entity ID field.

## Getting Started

```bash
# Clone the repository
git clone https://github.com/jtlabsio/mongodb-benchmark.git
cd mongodb-benchmark
# Start a local MongoDB instance
mkdir -p .mongo
docker run -d -p 27017:27017 --name benchmark -v $PWD/.mongo:/data/db mongo
# Install dependencies
go mod download
```

Now populate the database...

```bash
go run cmd/main.go -p true
```

Now run the server...

```bash
go run cmd/main.go
```

### Setup the environment

The environment can be customized by creating a YAML file in the `./settings` directory. The default environment is `development`, but can be overridden by setting the `GO_ENV` or `ENV` environment variable to the desired environment name.

#### Configuration / Settings

By default, the application will point to a locally running instance of MongoDB on port 27017 (per the Docker command above), and will set the logging level to `info`.

For additional verbosity, or to override any settings, either modify the `./settings/defaults.yaml` or create a new YAML file in either `./` or `./settings` with the name of an environment you wish to use (e.g. `./settings/development.yaml`). See below for an example that will override the default logging level and set it to `trace`.

```yaml
logging:
  level: trace
```

To run the application with the custom settings, use either the `GO_ENV` or `ENV` environment variable to specify the environment name (which should match the name of the file, i.e. `development`).

```bash
GO_ENV=development go run cmd/main.go
```

In addition to adjusting the logging level, the default address and port can be changed by modifying the `server.address` value in the settings file, along with several other settings (i.e. MongoDB connection details, pagination settings, etc.).

#### Populate with test data

To begin with benchmarking, you will need to populate the database with test data. This can be done by running the following command:

```bash
GO_ENV=development go run cmd/main.go -p true
```

The above command will take anywhere from 2 to 4 hours to complete (depending on your machine memory and CPU). The process will insert 10,000,000 documents into the `benchmark` database, `randoBase` collection, and 10,000,000 documents into the `randoCustom` collection.

The population process populates date fields with random dates, string fields with random alpha numeric string values (the email field adds a random domain to the end of the string based on a predefined list of options), and the `favoriteColor` field with a random color name.

For `randoBase` the `_id` field is populated with UUID (with the `-` characters removed), and for `randoCustom` the `randoID` field is populated with UUID (with the `-` characters removed) leaving the `_id` untouched to be assigned automatically by MongoDB.

##### Testing Schema

The schema for the `randoBase` collection is as follows:

```json
{
  "$jsonSchema": {
    "bsonType": "object",
    "properties": {
      "createdAt": {
        "bsonType": "date"
      },
      "email": {
        "bsonType": "string"
      },
      "favoriteColor": {
        "bsonType": "string"
      },
      "firstName": {
        "bsonType": "string"
      },
      "lastName": {
        "bsonType": "string"
      },
      "updatedAt": {
        "bsonType": "date"
      }
    }
  }
  ```

The schema for the `randoCustom` collection is as follows (the same as above, but with an added field for `randoID`):

```json
{
  "$jsonSchema": {
    "bsonType": "object",
    "properties": {
      "createdAt": {
        "bsonType": "date"
      },
      "email": {
        "bsonType": "string"
      },
      "favoriteColor": {
        "bsonType": "string"
      },
      "firstName": {
        "bsonType": "string"
      },
      "lastName": {
        "bsonType": "string"
      },
      "randoID": {
        "bsonType": "string"
      },
      "updatedAt": {
        "bsonType": "date"
      }
    }
  }
}
```

##### Testing Indexes

The following indexes are added for the `randoBase` collection:

```json
[
  {
    "createdAt": 1
  },
  {
    "email": 1,
    "unique": true
  },
  {
    "favoriteColor": 1
  },
  {
    "updatedAt": -1
  }
]
```

The following indexes are added for the `randoCustom` collection:

```json
[
  {
    "createdAt": 1
  },
  {
    "email": 1,
    "unique": true
  },
  {
    "favoriteColor": 1
  },
  {
    "randoID": 1,
    "unique": true
  },
  {
    "updatedAt": -1
  }
]
```

### Running the Benchmark

The benchmarking process is dependent upon k6, a modern load testing tool. Docker can be used to execute the k6 load tests, per the instructions below. Alternative, to install k6, follow the instructions on the [k6 website](https://k6.io/docs/getting-started/installation/).

Before testing, in a separate terminal window, start the server:

```bash
go run cmd/main.go
```

__Note:__ If the port is changed in the settings file, the port number will need to be updated in the k6 scripts (on line 13).

To test the base case (`_id` is used as the ID field):

```bash
docker run --rm --add-host=host.docker.internal:host-gateway -i grafana/k6 run - < ./base.k6.js
```

To test the custom case (`randoID` is used as the ID field, and `_id` is left to be assigned by MongoDB):

```bash
docker run --rm --add-host=host.docker.internal:host-gateway -i grafana/k6 run - < ./custom.k6.js
```
