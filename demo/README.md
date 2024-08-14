# Database Setup

(TIP: Run `make setup`)

Stand up a postgres instance:

```bash
docker run --rm -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgres:16-alpine
```

Create a named configuration for the postgres instance ( postgres sqldb provider ):

```bash

wash config put pg-task-db \
 POSTGRES_HOST=127.0.0.1 \
 POSTGRES_PORT=5432 \
 POSTGRES_USERNAME=postgres \
 POSTGRES_PASSWORD=postgres \
 POSTGRES_DATABASE=postgres \
 POSTGRES_TLS_REQUIRED=false

```

Create tables

```bash
cat components/task-manager/tasks.sql | PGPASSWORD=postgres psql -Upostgres -h 127.0.0.1
```

# Application Setup

Compile everything:

```bash
make build
```

Deploy:

```bash
make deploy
```

Reach the application at `http://localhost:8080`

# Components

- http-router: Capability Provider bringing external requests into the system
- task-manager: Keeps track of background tasks, storing state in postgres
- image-processor: Resizes images, stashing them in blobstore
- image-analyzer: Runs LLM on images from blobstore

# Patterns

## Serving User Interfaces

The UI is bundled in the [http-router](./providers/http-router/static/).

## DMZ ( demilitarized zone ) / API Gateway

The HTTP Router is responsible for handling incoming requests and routing them to the appropriate component.
External Requests land in the http-router as json and are converted to wit/wrpc (strong typing) to communicate with components.
