# Notificaciones durables con FCM HTTP v1

Joss puede utilizar FCM como transporte privilegiado sin delegar el backend a Firebase o Supabase. El runtime mantiene la fuente de verdad en la base de datos:

1. La app registra su token en `push_devices`.
2. El backend inserta el mensaje en `notifications` con estado `pending`.
3. El dispatcher crea una entrega en `notification_deliveries` y llama a FCM HTTP v1.
4. Flutter muestra la notificación o el modal in-app.
5. El dispositivo confirma la entrega y Joss cambia el mensaje a `sent`.

## Configuración del servidor

Activa Firebase Cloud Messaging API en el proyecto de Firebase usado por la app. Genera una cuenta de servicio desde Firebase Console y guarda el JSON únicamente en el servidor.

```ini
FCM_CREDENTIALS_PATH="storage/secrets/firebase-service-account.json"
```

No subas la cuenta de servicio a Git ni la incluyas como asset de Flutter. El runtime obtiene tokens OAuth 2.0 de corta duración con el scope `https://www.googleapis.com/auth/firebase.messaging` y llama a:

```text
POST https://fcm.googleapis.com/v1/projects/PROJECT_ID/messages:send
```

## Tablas administradas por Joss

- `push_devices`: tokens por usuario, aplicación y plataforma.
- `notifications`: buzón durable y estado confirmado por el dispositivo.
- `notification_deliveries`: outbox por dispositivo, intentos, error y mensaje asignado por FCM.

El dispatcher reintenta fallos hasta cinco veces y desactiva tokens que FCM reporta como no registrados.

## Límites reales

- En Android la app debe abrirse al menos una vez y aceptar el permiso de notificaciones.
- Si el usuario fuerza la detención desde Ajustes, Android no vuelve a entregar mensajes hasta que abra la app.
- FCM confirma que aceptó el mensaje, no que el usuario lo vio. Por eso Joss conserva el ACK de aplicación y el buzón durable.
