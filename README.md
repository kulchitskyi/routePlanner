# RoutePlanner

The core idea behind RoutePlanner is to provide users the opportunity to write their preferences in human language and receive a tailored travel route in return. Behind this idea lies backend built with Fiber that orchestrates complex geospatial operations, local LLM inference, and secure user management. It prioritizes clean code architecture and efficient resource management.

## Architecture & Design Principles

RoutePlanner follows the principles of **Clean Architecture**, enforcing a strict separation of concerns to ensure maintainability and testability.

-   **Layered Architecture**: the application is structured into separate layers:
    -   **Transport Layer (Handlers)**: manages HTTP requests/responses, input validation, and rate limiting integration.
    -   **Business Logic Layer (Services)**: contains the core domain logic, independent of database or HTTP concerns.
    -   **Data Access Layer (Repositories)**: abstracts database interactions, allowing for easy swapping of storage mechanisms.
-   **Dependency Injection**: core components (Loggers, Services, Repositories) are injected at runtime, facilitating easy testing and loose coupling.
-   **Configurable Validation**: utilizes struct-tag-based configuration validation to foster fail-fast startup behavior and runtime safety.

## Key Backend Features

### Intelligent Intent Analysis (LLM Integration)
-   **Local Inference**: a primary challenge was optimizing of the prompt to "squeeze" out maximum reasoning capability from the constraint of a small, local model. So **Ollama** (llama3.2:1b) was used to perform offline Natural Language Processing. 
-   **Semantic Tag Extraction**: analyses unstructured user prompts (e.g., "I want a quiet place to read and drink coffee") to extract standardized search tags, enabling semantic search over geospatial data.

### High-Performance Geospatial Engine
-   **PostGIS Powered**: leverages the power of PostgreSQL's **PostGIS** extension for advanced spatial queries, indexing, and radius-based lookups.
-   **Routing Integration**: interfaces with **Geoapify** to calculate optimized travel paths between discovered points of interest.

### Resilience & Scalability
-   **Distributed Caching**: uses **Redis** for caching frequently accessed data (allowed tags, ratelimiting data, etc.), reducing latency and load on the primary database.
-   **Rate Limiting**: implements sliding-window rate limiters as fiber middleware to protect important endpoints (LLM analysis, Route Building, Place Creation) from abuse.
-   **Connection Pooling**: efficient database and Redis connection management to handle concurrent request loads.

### Observability & Metrics
-   **Instrumentation**: **Prometheus** metrics export key performance indicators (HTTP request duration, potential error rates, count of calls to external API).
-   **Structured Logging**: uses Go's `slog` for structured, leveled logging, making it easier to ingest and analyze logs in aggregation systems (e.g., Loki).

## Tech Stack

-   **Golang**
-   **Fiber**
-   **GORM**

**Data & Infrastructure**

-   **PostgreSQL 16 + PostGIS**
-   **Redis 7**
-   **Docker & Docker Compose**

**AI & External Services**
-   **Ollama**
-   **Geoapify**

**Frontend (only for demonstration)**
-   **Vanilla JavaScript**
-   **Leaflet.js**

### Quick Start
1.  **Make .env file** <br>
    Bare minimum is (I provide default values where it is possible):
    ```
    ALLOW_ORIGINS=*
    JWT_SECRET= YOUR_SECRET

    OLLAMA_HOST=llm-service
    OLLAMA_PORT=11434
    OLLAMA_URL="http://${OLLAMA_HOST}:${OLLAMA_PORT}"

    DB_HOST=db
    DB_PORT=5432
    DB_USERNAME=postgres
    DB_PASSWORD= YOUR_PASSWORD
    DB_NAME=postgres
    DATABASE_URL="host=${DB_HOST} user=${DB_USERNAME} password=${DB_PASSWORD} dbname=${DB_NAME} port=${DB_PORT}"

    REDIS_HOST=redis
    REDIS_PORT=6379
    REDIS_PASSWORD= YOUR_REDIS_PASSWORD

    GEOAPIFY_API_KEY= YOUR_GEOAPIFY_KEY
    ```

2.  **Start Infrastructure**:
    ```bash
    docker-compose up -build
    ```

3. **load actual llm model into ollama volume**
    ```
    docker exec -it ollama ollama pull llama3.2:1b
    ```
4. **Now you can go to http://localhost:YOUR_SERVER_PORT/ and test it and check metrics ain grafana dashbord at http://localhost:3000**:


### All .env variables:
```
SERVER_PORT=
ALLOW_ORIGINS=
MAX_CONNECTIONS=
READ_TIMEOUT=
WRITE_TIMEOUT=
IDLE_TIMEOUT=

LOG_LEVEL=

JWT_SECRET=
JWT_EXPIRATION_HOURS=

#RATELIMITS
PLACE_CREATE_LIMIT_PER_HOUR=
ROUTE_BUILD_TOTAL_PER_DAY=
ROUTE_BUILD_USER_PER_30S=

OLLAMA_HOST=llm-service
OLLAMA_PORT=
OLLAMA_URL="http://${OLLAMA_HOST}:${OLLAMA_PORT}"

DB_HOST=
DB_PORT=
DB_USERNAME=
DB_PASSWORD=
DB_NAME=
DATABASE_URL="host=${DB_HOST} user=${DB_USERNAME} password=${DB_PASSWORD} dbname=${DB_NAME} port=${DB_PORT}"

REDIS_HOST=
REDIS_PORT=
REDIS_PASSWORD=

CAHCE_TAGS_REFRESH_TIME_MIN=

GEOAPIFY_API_KEY=
GEOAPIFY_REQ_TIMEOUT=
```