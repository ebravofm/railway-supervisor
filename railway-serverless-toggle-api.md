# Toggle serverless y redeploy vía API (Railway)

Resumen para diseñar un servicio que programe activación/desactivación de serverless y redeploys en contenedores Railway.

## API

- **Endpoint:** `POST https://backboard.railway.com/graphql/v2`
- **Auth:** header `Authorization: Bearer <TOKEN>` (token de cuenta/team; project token no basta para estas mutaciones).

## Mutaciones usadas

### 1. Toggle serverless

**Mutación:** `serviceInstanceUpdate`

- **Campo:** `sleepApplication: Boolean` en el input (`true` = serverless activado, `false` = desactivado).
- **Args:** `environmentId`, `serviceId`, `input: ServiceInstanceUpdateInput`.
- **Respuesta:** `Boolean` (sin subcampos en la query).

Ejemplo (desactivar):

```graphql
mutation($input: ServiceInstanceUpdateInput!) {
  serviceInstanceUpdate(
    environmentId: "<ENVIRONMENT_ID>"
    serviceId: "<SERVICE_ID>"
    input: $input
  )
}
# variables: { "input": { "sleepApplication": false } }
```

### 2. Disparar deploy (para que tome efecto)

**Mutación:** `serviceInstanceDeployV2`

- **Args:** `serviceId`, `environmentId`.
- **Respuesta:** `String!` (ID del nuevo deployment).

Ejemplo:

```graphql
mutation {
  serviceInstanceDeployV2(
    serviceId: "<SERVICE_ID>"
    environmentId: "<ENVIRONMENT_ID>"
  )
}
```

## Flujo recomendado

1. Llamar `serviceInstanceUpdate` con `sleepApplication: true | false`.
2. Llamar `serviceInstanceDeployV2` con el mismo `serviceId` y `environmentId` para aplicar el cambio.

## IDs necesarios

- **Environment ID:** p.ej. en la URL del dashboard: `?environmentId=...`
- **Service ID:** en la URL del servicio: `/service/<SERVICE_ID>`
- **Token:** crear/rotar en Railway (Account/Team) y guardar en variable de entorno (nunca en código).

## Referencia rápida (curl)

```bash
# 1) Toggle (ejemplo: desactivar serverless)
curl -s -X POST https://backboard.railway.com/graphql/v2 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $RAILWAY_TOKEN" \
  -d '{"query":"mutation($input: ServiceInstanceUpdateInput!) { serviceInstanceUpdate(environmentId: \"<ENV_ID>\", serviceId: \"<SVC_ID>\", input: $input) }","variables":{"input":{"sleepApplication":false}}}'

# 2) Deploy
curl -s -X POST https://backboard.railway.com/graphql/v2 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $RAILWAY_TOKEN" \
  -d '{"query":"mutation { serviceInstanceDeployV2(serviceId: \"<SVC_ID>\", environmentId: \"<ENV_ID>\") }"}'
```

Sustituir `<ENV_ID>`, `<SVC_ID>` y `$RAILWAY_TOKEN`. Para activar serverless usar `"sleepApplication":true` en el paso 1.
