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

> [!IMPORTANT]
> Si vas a definir `CONFIG_JSON` en un archivo `.env`, **debes convertirlo a una sola línea (minificado)**. Los archivos `.env` no soportan valores multilínea por defecto.

### 2. Formato de `CONFIG_JSON`
Puedes escribirlo así para mayor claridad (o guardarlo en un archivo `config.json` si prefieres):

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

### Conversión para el archivo `.env`
Para usarlo en tu `.env`, debes "aplastar" el JSON anterior en una sola línea. 

**Ejemplo de cómo debe verse en tu `.env`:**
```env
RAILWAY_TOKEN=tu_token_aqui
CONFIG_JSON={"checkIntervalMinutes":5,"timezone":"America/Santiago","rules":[{"name":"Dormir Staging","environmentId":"TU_ENV_ID","serviceId":null,"sleepWindow":{"start":"02:00","end":"08:00"}}]}
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
