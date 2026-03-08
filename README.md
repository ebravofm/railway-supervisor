# Railway Serverless Supervisor 🤖

Servicio ultraliviano en Go para automatizar el modo **Serverless** (`sleepApplication`) en servicios de Railway.

## Estructura del Proyecto
El proyecto sigue el estándar de organización de Go para mayor escalabilidad:
- `cmd/supervisor/`: Punto de entrada de la aplicación.
- `pkg/railway/`: Cliente para la API de GraphQL de Railway.
- `pkg/supervisor/`: Lógica central de monitoreo y reglas de horarios.
- `Dockerfile`: Construcción multi-etapa para una imagen final mínima (~15MB).

## Configuración

### 1. Variables de Entorno
Crea un archivo `.env` o configúralas en el dashboard de Railway:

- `RAILWAY_TOKEN`: Token de cuenta/equipo.
- `CONFIG_JSON`: Configuración de reglas en formato JSON.

### 2. Formato de `CONFIG_JSON`
```json
{
  "checkIntervalMinutes": 5,
  "timezone": "America/Santiago",
  "rules": [
    {
      "name": "Dormir Staging",
      "environmentId": "TU_ENV_ID",
      "serviceId": null,
      "sleepWindow": { "start": "02:00", "end": "08:00" }
    }
  ]
}
```
*Si `serviceId` es `null`, aplicará la regla a **todos** los servicios del entorno indicado.*

## Comandos Útiles

### Compilar Localmente
```bash
go build -o supervisor ./cmd/supervisor/main.go
```

### Ejecutar Localmente
```bash
./supervisor
```

### Docker
```bash
docker build -t railway-supervisor .
docker run --env-file .env railway-supervisor
```
