# MongoDB Index Benchmark Testing

## Getting Started

### Setup the environment

```bash
git clone https://github.com/jtlabsio/mongodb-benchmark.git
cd mongodb-benchmark
mkdir -p .mongo
docker run -d -p 27017:27017 --name benchmark -v $PWD/.mongo:/data/db mongo
```

#### Populate with test data

