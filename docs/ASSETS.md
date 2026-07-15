# Assets

El servidor sirve `public/` bajo `/public/` y `/assets/`. En desarrollo también expone assets detectados dentro de `node_modules` mediante `/assets/vendor/`.

El detector lee únicamente `dependencies` de `package.json`. Para cada paquete instalado busca `style`, `main` terminado en `.js` y, como fallback, archivos `*.min.css`/`*.min.js` en la raíz o `dist/`. No garantiza detectar todos los formatos modernos de paquetes.

Las etiquetas CSS se insertan antes de `</head>` y las JS antes de `</body>` al renderizar vistas. En build VFS, `/assets/vendor/` no sirve directamente desde `node_modules`; incluye en el build cualquier asset que la aplicación necesite.

Los `.scss` de `assets/css/` se recompilan durante reload. El compilador incluido implementa un subconjunto de SCSS y resolución básica de imports; no sustituye toda la semántica de Sass.
