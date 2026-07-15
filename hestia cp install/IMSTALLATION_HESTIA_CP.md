# Integración experimental con HestiaCP

Esta carpeta contiene `setup_hestia_cp.sh`, un script independiente para un servidor Linux que ya tenga HestiaCP y el comando `joss` instalado. No forma parte del instalador general ni del runtime.

## Qué modifica

El script debe ejecutarse como `root` y escribe configuración del sistema:

- plantillas Nginx de Hestia llamadas `joss`;
- `/usr/local/bin/joss-launcher`;
- `/etc/systemd/system/joss@.service`;
- `/usr/local/bin/deploy-joss`.

El servicio systemd arranca como `root`, pero el launcher localiza `/home/*/web/<dominio>/public_html`, obtiene el propietario de Hestia y ejecuta `joss run main.joss` mediante `runuser`. Revisa el script antes de usarlo en producción: sobrescribe esos archivos y recarga systemd/Nginx.

## Preparación

```bash
joss version
sudo bash setup_hestia_cp.sh
```

Después, crea el dominio en HestiaCP y selecciona la plantilla Nginx `joss`.

En el proyecto local:

```bash
joss build web
```

Ese comando copia el proyecto a `build/`, excluye `env.joss`, cifra el entorno como `env.enc` y crea `nginx_port.conf` con el valor de `PORT` —o `8000` para el build si no está definido—. No compila el programa a un binario autónomo.

Sube el contenido de `build/` a `/home/<usuario>/web/<dominio>/public_html/` y ejecuta:

```bash
sudo deploy-joss ejemplo.com
```

El despliegue cambia el propietario de `public_html`, reinicia y habilita `joss@ejemplo.com`, y recarga Nginx.

## Diagnóstico

```bash
systemctl status joss@ejemplo.com
journalctl -u joss@ejemplo.com -f
```

Limitaciones: la búsqueda del dominio usa la primera coincidencia bajo `/home/*/web`; no hay rollback, aislamiento por sandbox de systemd ni gestión automática de secretos. Esta integración debe endurecerse y probarse en un entorno de staging antes de producción.
