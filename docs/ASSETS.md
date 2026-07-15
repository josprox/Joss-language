# Gestión de Assets y Node.js en Joss

Joss incluye un sistema moderno de gestión de assets que se integra automáticamente con el ecosistema de Node.js.

## Integración con Node.js (Auto-Vendor)

El sistema detecta automáticamente los paquetes instalados vía `npm` en tu proyecto y sirve sus archivos CSS y JS sin configuración adicional.

### Cómo funciona

1. **Instalación**: Instala cualquier librería de frontend usando npm.
   ```bash
   npm install bootstrap
   npm install jquery
   npm install animate.css
   ```

2. **Detección Automática**:
   El servidor monitorea tu archivo `package.json` y la carpeta `node_modules`.
   - Al detectar cambios (instalación/desinstalación), escanea los paquetes.
   - Busca archivos principales (.css, .js) definidos en el `package.json` de la librería o en carpetas estándar (`dist/`).

3. **Inyección Automática**:
   Los assets detectados se inyectan automáticamente en tus vistas HTML.
   - **CSS**: Se inserta antes de `</head>`.
   - **JS**: Se inserta antes de `</body>`.

   Si necesitas control manual, puedes usar el placeholder `<!-- JOSS_ASSETS -->` en tu layout principal.

### Hot Reload

El servidor soporta recarga en caliente para dependencias:
- Si ejecutas `npm install <paquete>` mientras el servidor corre, el navegador se recargará automáticamente con los nuevos estilos/scripts aplicados.
- Si borras `node_modules`, el servidor limpiará los assets cacheados al instante.

### Rutas Virtuales

Los archivos de `node_modules` no se exponen públicamente por defecto por seguridad. Joss crea rutas virtuales seguras solo para los assets detectados:

- **Ruta**: `/assets/vendor/<paquete>/<archivo>`
- **Mapeo Real**: `node_modules/<paquete>/<archivo>`

Ejemplo:
Si instalas bootstrap, el servidor genera automáticamente:
`<link rel="stylesheet" href="/assets/vendor/bootstrap/dist/css/bootstrap.min.css">`

## CSS Personalizado (SCSS)

Tus estilos personalizados viven en `assets/css/app.scss`.
El servidor los compila automáticamente a `public/css/app.css` en cada cambio.

### Estructura Recomendada

```
assets/
├── css/
│   ├── app.scss       # Importa otros módulos
│   ├── _variables.scss
│   └── _components.scss
```

## Solución de Problemas

**Mi paquete no aparece:**
1. Revisa que exista en `package.json`.
2. Verifica que el paquete tenga un campo `style` o `main` en su `package.json`, o una carpeta `dist/` con archivos `.min.css` o `.min.js`.
3. Reinicia el servidor para forzar un re-escaneo completo (aunque el Hot Reload debería detectarlo).

**Los estilos de Node se ven rotos:**
Asegúrate de no tener reglas CSS globales en tu `app.scss` que sobrescriban las librerías.
