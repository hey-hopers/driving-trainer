# Driving Trainer — Architecture

## Overview

Driving Trainer is an Android application for structured driving practice using real-world routes, route difficulty analysis, progression and gamification.

The system is centered around two main capabilities:

1. Analyze the difficulty of a real driving route.
2. Generate or recommend training routes appropriate for a driver's accumulated experience.

The architecture must keep third-party navigation and routing providers replaceable.

---

# High-Level Architecture

```text
Android Application
        |
        | HTTPS / JSON
        v
Go Backend API
        |
        +--------------------+
        |                    |
        v                    v
PostgreSQL + PostGIS     Routing Layer
                             |
                             v
                          Valhalla
                             |
                       OpenStreetMap
                       + Elevation DEM
```

Future domain components:

```text
Go Backend
    |
    +-- Difficulty Engine
    |
    +-- Training Engine
    |
    +-- Progression
    |
    +-- Training Sessions
```

---

# Android Application

Technology:

* Kotlin
* Jetpack Compose
* Coroutines / Flow
* Hilt
* Room
* DataStore

Future:

* HERE SDK Navigate
* Android for Cars App Library

The Android application is responsible for:

* user interface
* map visualization
* route creation interface
* navigation
* local training state
* GPS/location access
* temporary offline storage
* synchronization with backend

The Android application must not be responsible for:

* authoritative difficulty calculation
* authoritative XP calculation
* route recommendation business rules
* persistent domain truth

---

# Backend API

Technology:

* Go
* `net/http`
* Chi
* pgx
* sqlc

The backend is the authoritative application layer.

Responsibilities include:

* route requests
* route analysis
* route persistence
* difficulty calculation
* training recommendation
* training sessions
* progression
* achievements
* user domain data

The backend should remain modular without being split into microservices prematurely.

---

# Routing Abstraction

Routing must be accessed through an internal abstraction.

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

The domain must not depend directly on Valhalla-specific models.

Current routing provider:

* Valhalla

Future routing providers should be replaceable without changing route domain logic.

---

# Valhalla

Valhalla is treated as infrastructure.

Responsibilities:

* vehicle route calculation
* routing graph
* map matching
* route attributes
* maneuver generation
* route geometry
* elevation integration where applicable

Data sources:

* OpenStreetMap
* local Digital Elevation Model

Valhalla should initially operate on a regional dataset rather than the entire country.

---

# Navigation Provider

Primary planned navigation provider:

* HERE SDK Navigate

HERE is responsible for in-vehicle navigation UX such as:

* turn-by-turn navigation
* route guidance
* voice instructions
* rerouting
* offline navigation where supported

The application domain must not depend directly on HERE models.

The navigation provider is considered replaceable.

---

# PostgreSQL + PostGIS

PostgreSQL is the main application database.

PostGIS is used for geographic data.

Preferred geographic types:

```text
GEOGRAPHY(POINT, 4326)
GEOGRAPHY(LINESTRING, 4326)
```

Core geographic objects must not be represented only by independent latitude and longitude columns when a PostGIS type is more appropriate.

---

# Route Domain

The initial route domain contains:

```text
Route
RouteSegment
RouteEvent
RouteAnalysis
RouteCategoryScore
```

## Route

Represents a physical route.

Contains information such as:

* geometry
* distance
* duration
* source
* global difficulty

Current persistence stores the authoritative route geometry in PostgreSQL/PostGIS as:

```text
GEOGRAPHY(LINESTRING, 4326)
```

For routes produced by Valhalla, the backend decodes the returned polyline6 into coordinates and writes those coordinates as the PostGIS linestring. The original polyline is retained only as auxiliary provider metadata.

## RouteSegment

Represents a meaningful portion of the route.

A segment may be split when there is a relevant change in:

* road
* road class
* slope
* geometry
* speed characteristics
* intersection context

Potential attributes:

* road class
* road name
* speed limit
* elevation
* incline
* curve score
* difficulty

## RouteEvent

Represents a meaningful driving situation.

Examples:

```text
HILL
STEEP_HILL
STOP
TRAFFIC_LIGHT
HILL_STOP
CURVE
SHARP_CURVE
CURVE_SEQUENCE
INTERSECTION
COMPLEX_INTERSECTION
ROUNDABOUT
HIGH_SPEED_ENTRY
HIGHWAY_ENTRY
HIGHWAY_EXIT
LANE_CHANGE
```

Events can contain flexible metadata through JSONB.

Current event sources:

* Slope analysis produces `HILL` and `STEEP_HILL`.
* Geometry analysis produces `CURVE`, `SHARP_CURVE` and `CURVE_SEQUENCE`.
* Valhalla road attributes and segment transitions produce `INTERSECTION`, `COMPLEX_INTERSECTION`, `ROUNDABOUT`, `HIGHWAY_ENTRY` and `HIGHWAY_EXIT`.
* Compound event analysis associates nearby route events and route-boundary stop contexts to produce `HILL_STOP` and additional `COMPLEX_INTERSECTION` events.

Known data-source limitation:

* `STOP` and `TRAFFIC_LIGHT` remain valid domain event types, but reliable detection may require enrichment beyond the current Valhalla `trace_attributes` integration, such as Valhalla tiles, `/locate`, custom Valhalla attributes or direct OSM/PostGIS data.

## RouteAnalysis

Represents the output of one Difficulty Engine version.

Important fields include:

* engine version
* overall difficulty
* average difficulty
* peak difficulty
* complexity score

Analyses should be versioned so future algorithm changes do not destroy historical interpretation.

## RouteCategoryScore

Difficulty by category.

Examples:

```text
HILLS
CURVES
INTERSECTIONS
HIGH_SPEED
HIGHWAY
ROUNDABOUTS
```

Two routes with the same overall difficulty may have very different category profiles.

---

# Difficulty Engine

The Difficulty Engine answers:

> How difficult is this route?

It should be deterministic and independently testable.

Potential inputs include:

* road type
* speed characteristics
* incline
* elevation
* curves
* intersections
* mandatory stops
* traffic lights
* roundabouts
* highway sections
* maneuver complexity
* compound events

Example:

```text
Hill
+
mandatory stop
+
intersection
=
HILL_STOP
```

A route difficulty score should not simply be the arithmetic average of all segments.

The engine should consider:

* average difficulty
* peak events
* frequency of difficult events
* diversity of driving situations

---

# Training Engine

The Training Engine answers:

> Which route should this driver practice next?

Inputs may include:

* driver's accumulated experience
* desired training category
* target difficulty zone
* available time
* current location
* candidate routes

Flow:

```text
Driver Profile
      +
Training Goal
      +
Target Difficulty
      |
      v
Generate Candidate Routes
      |
      v
Difficulty Engine
      |
      v
Rank Candidates
      |
      v
Recommended Training Route
```

The Training Engine and Difficulty Engine must remain separate systems.

---

# Driver Progression

The platform measures accumulated practice and exposure.

It must not claim to certify driving skill.

Preferred terminology:

* experience
* training level
* accumulated practice
* exposure

Avoid presenting metrics as verified driving competency.

---

# Training Session

A TrainingSession represents a user driving a Route.

Future flow:

```text
Route selected
     |
Training started
     |
Navigation
     |
Training completed
     |
Feedback
     |
Experience gain
     |
Progression update
```

The client must never be authoritative over XP or achievements.

---

# Offline Strategy

Driving should not require constant backend connectivity.

The Android application should eventually be capable of storing locally:

* active route
* route events
* active training session
* telemetry buffer

Synchronization can occur when connectivity returns.

---

# Authentication

Authentication provider:

* Firebase Auth

Firebase is responsible for identity only.

Domain data remains in PostgreSQL.

The backend validates Firebase tokens and maps them to internal users.

Firestore is not the primary application database.

---

# Future Telemetry

Potential telemetry includes:

* GPS
* speed
* heading
* timestamps
* accelerometer
* gyroscope

Telemetry is not part of the first milestones.

Do not prematurely introduce a high-volume telemetry architecture.

---

# Architectural Priorities

1. Correct domain boundaries.
2. Testability.
3. Geographic accuracy.
4. Safety.
5. Replaceable external providers.
6. Simple deployment.
7. Incremental evolution.

Avoid premature:

* microservices
* event-driven architecture
* distributed systems
* Kubernetes
* machine learning
* large abstractions
