# Extended Flow Management System

This module implements the "Extended Flow Management System" with Connector support.

## Features
- **Connector Management**: API to register and configure external service connectors.
- **Execution Engine**: Supports triggering flows and async execution (simulated).
- **API Gateway**: Unified REST API for management and execution.

## Running
```bash
go run cmd/server/main.go
```

## API Documentation (Swagger)
Once the server is running, you can access the Swagger UI at:
http://localhost:8080/swagger/index.html

## API Endpoints
- `POST /api/v1/connectors`: Create a connector
- `GET /api/v1/connectors`: List connectors
- `POST /api/v1/connectors/config`: Create connector configuration
- `POST /api/v1/flows/{id}/execute`: Trigger a flow
