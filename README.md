# Railway Serverless Supervisor

Un servicio dockerizado en Go ultraliviano (~15MB RAM) para prender y apagar el modo "Serverless" (`sleepApplication`) de otros servicios en tu cuenta de Railway.

## ¿Por qué existe esto?
Para ahorrar dinero. Railway cobra por uso de RAM/CPU de los contenedores encendidos. Con este supervisor puedes programar, a través de reglas JSON, qué horas del día tus entornos (ej: Staging) o servicios específicos deben entrar en modo _Serverless_ (apagado con scale-to-zero) y cuándo deben volver a encenderse.

## Cómo implementar en Railway

### 1. Variables de Entorno Necesarias
Para que este supervisor funcione correctamente en tu proyecto de Railway, debes configurar las siguientes variables de entorno:

- `RAILWAY_TOKEN`: Un token de API a nivel Account/Team (no a nivel de Proyecto). Puedes generarlo en la configuración de tu cuenta o equipo en el Dashboard de Railway.
- `CONFIG_JSON` (opcional pero **muy recomendado**): Pega el contenido de tu configuración JSON aquí. Esto te permite cambiar horas y reglas sin tocar el código nuevamente. Alternativamente el servicio leerá el archivo `config.json` en disco si no se provee.

Ejemplo de `CONFIG_JSON`:
```json
{
  "checkIntervalMinutes": 5,
  "timezone": "America/Santiago",
  "rules": [
    {
      "name": "Apagar todo Staging",
      "environmentId": "UUID_DEL_ENTORNO",
      "serviceId": null,
      "sleepWindow": {
        "start": "02:00",
        "end": "08:00"
      }
    }
  ]
}
```

*Nota:* Los UUID de `environmentId` y `serviceId` los puedes sacar de la URL en el dashboard de Railway.

### 2. Despliegue (Deploy)
1. Haz un commit y push de esta carpeta (`Dockerfile`, `main.go`, `go.mod`, etc.) a tu repositorio de GitHub conectado a Railway.
2. Opcionalmente puedes usar el CLI: `railway up`.
3. ¡Asegúrate de agregar las variables de entorno mencionadas arriba en la configuración del servicio dentro del panel de Railway!

## Lógica del Supervisor
El script verificará cada X minutos (configurado en `checkIntervalMinutes`) si la hora local (configurado en `timezone`) se encuentra dentro o fuera del periodo de gracia (`sleepWindow.start` y `sleepWindow.end`).

- Si solo provees el `environmentId` y pones `serviceId: null`: El supervisor hará una llamada previa a la API para obtener dinámicamente **todos** los servicios alojados en el entorno y forzará Sleep/Wakeover en cada uno de ellos.
- Cada vez que se actualiza el estado, se emite un redespliegue (`deployV2`) automáticamente, para que Railway actualice la asignación de réplicas en su infrastructura.
