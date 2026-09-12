# Driving Trainer — Agent Guidelines

## Project Goal

Driving Trainer is an Android application focused on helping people practice real-world driving through structured routes, progressive difficulty and gamification.

The product is not a racing application.

The system must never reward:

* driving faster
* aggressive driving
* shortest completion time
* speeding
* risky maneuvers

Progression must be based on:

* completed training sessions
* variety of situations practiced
* distance practiced
* experience accumulated
* route difficulty
* consistency and progression

The central product concept is:

> Find routes that help a driver practice specific real-world driving situations at an appropriate difficulty level.

---

# Official Stack

## Android

* Kotlin
* Jetpack Compose
* Coroutines / Flow
* Hilt
* Room
* DataStore

Future:

* Android for Cars App Library
* Android Auto

## Navigation

Primary navigation SDK:

* HERE SDK Navigate

The navigation SDK must remain replaceable.

Business logic must not depend directly on HERE SDK models.

---

# Routing Engine

* Valhalla
* self-hosted
* OpenStreetMap data
* local DEM elevation data

Valhalla is responsible for:

* route generation
* routing graph
* map matching
* route attributes
* elevation integration where applicable

Valhalla should be treated as infrastructure.

---

# Backend

* Go
* standard `net/http`
* Chi router
* pgx
* sqlc

Avoid introducing large frameworks unless there is a strong technical reason.

---

# Database

* PostgreSQL
* PostGIS

Use PostGIS native geographic types for geographic data.

Prefer:

* GEOGRAPHY(POINT, 4326)
* GEOGRAPHY(LINESTRING, 4326)

Do not store core geographic objects only as latitude/longitude columns when a PostGIS type is more appropriate.

---

# Authentication

* Firebase Auth

Firebase is responsible only for identity/authentication.

Do not use Firestore as the application's primary database.

Application domain data must remain in PostgreSQL/PostGIS.

---

# Future Infrastructure

Only introduce these when there is a real need:

* Redis
* Asynq
* OpenTelemetry
* Prometheus
* Grafana
* Terraform

Do not introduce infrastructure prematurely.

---

# Core Architecture

The system should preserve these boundaries:

Android
→ Go API
→ Domain Services
→ PostgreSQL/PostGIS

Routing:

Go API
→ Routing abstraction
→ Valhalla

Navigation:

Android
→ Navigation abstraction
→ HERE SDK

The domain must not depend directly on third-party SDK-specific models.

---

# Core Domain Systems

There are two distinct systems.

## Difficulty Engine

Responsible for answering:

> How difficult is this route?

Inputs may include:

* road type
* elevation
* inclination
* curves
* intersections
* STOP signs
* traffic lights
* roundabouts
* road speed
* highway sections
* required maneuvers
* compound events

Examples of compound events:

* HILL_STOP
* SHARP_CURVE
* HIGHWAY_ENTRY
* COMPLEX_INTERSECTION

The Difficulty Engine must remain deterministic and testable.

Do not introduce machine learning during the initial versions.

---

## Training Engine

Responsible for answering:

> Which route should this driver practice?

It combines:

* driver experience
* training goal
* target difficulty
* location
* available duration
* candidate routes
* Difficulty Engine results

The Training Engine and Difficulty Engine must remain separate modules.

---

# Driver Progression

Do not describe progression as certified driving skill.

Prefer terms such as:

* experience
* exposure
* training level
* accumulated practice

The application does not certify that a user is a good or safe driver.

---

# Difficulty Model

Route difficulty is not only an average.

The system should consider:

* average route difficulty
* peak difficulty
* difficult events
* diversity of driving situations

Examples:

A hill alone may be moderately difficult.

A hill with:

* mandatory stop
* intersection
* traffic
* poor visibility

must be considered substantially harder.

Difficulty should be modeled from route segments and route events.

---

# Main Geographic Entities

The expected domain includes:

* Route
* RouteSegment
* RouteEvent
* RouteAnalysis
* RouteCategoryScore

Later:

* User
* DriverProfile
* DriverExperience
* TrainingSession
* TrainingFeedback
* ExperienceGain
* Achievement

Do not create all future entities prematurely.

Implement only what the current milestone requires.

---

# Development Strategy

Use vertical slices.

Each milestone must produce something testable.

Do not implement multiple large subsystems in one task unless explicitly requested.

Preferred progression:

1. HTTP API foundation
2. Valhalla integration
3. Route persistence
4. Route segmentation
5. Difficulty Engine
6. Route event detection
7. Android map
8. Route creation
9. Navigation
10. Training sessions
11. Feedback
12. Progression
13. Training Engine
14. Android Auto
15. Advanced telemetry

---

# Current Project State

The Go backend is running.

The following endpoint exists and has been validated:

POST /api/v1/routes/analyze

It currently returns a mocked route analysis.

Do not remove or substantially change working behavior unless the current task explicitly requires it.

PostgreSQL + PostGIS are running locally through Docker and have been validated through DBeaver.

Git versioning is already configured.

---

# Current Milestone

The next major technical milestone is:

> Given an origin and destination, obtain a real route from Valhalla and return real route data through the Go API.

Before implementing this milestone, infrastructure and documentation should be prepared.

---

# Coding Principles

Prefer:

* small modules
* explicit dependencies
* interfaces at external boundaries
* clear domain types
* testable business logic
* context propagation
* structured error handling
* deterministic behavior

Avoid:

* unnecessary abstractions
* generic repository patterns without benefit
* global state
* large service objects
* duplicated DTO/domain models without reason
* premature microservices
* premature distributed systems
* premature optimization

---

# Go Conventions

Use `context.Context` for external calls and request-scoped work.

External integrations should be hidden behind interfaces.

Example concept:

```go
type Router interface {
    CalculateRoute(
        ctx context.Context,
        origin Coordinate,
        destination Coordinate,
    ) (RouteResult, error)
}
```

Valhalla implementation should satisfy this interface.

Domain code should not import Valhalla-specific packages.

---

# Database Rules

Schema changes must be made through migrations.

Never manually depend on production schema changes.

Use SQL directly when PostGIS operations are clearer than abstractions.

sqlc should be preferred for typed database access.

Use JSONB only for variable metadata.

Core searchable domain fields should remain explicit columns.

---

# Testing Rules

Business logic must be unit-testable without:

* real Valhalla
* real Firebase
* external internet
* production database

Use mocks or test servers at integration boundaries.

Important logic such as difficulty scoring must have deterministic tests.

---

# Safety Principles

The app must not encourage risky behavior.

Never introduce:

* speed leaderboards
* fastest route completion rewards
* time attack mechanics
* aggressive-driving rewards

During active driving, the UI should minimize interaction and distraction.

Gamification should appear primarily before or after driving.

---

# Working With Existing Code

Before making changes:

1. inspect the existing repository
2. understand current conventions
3. preserve working code where appropriate
4. make the smallest coherent change
5. run relevant tests
6. report what changed

Do not rewrite modules merely for stylistic preferences.

---

# Task Completion

When completing a development task:

* summarize files changed
* explain important architectural decisions
* mention assumptions
* run tests where possible
* mention any tests not run
* provide commands to validate the change locally

If a requested implementation conflicts with this architecture, explain the conflict before introducing a major architectural change.
