---
name: pinata-nft-upload
description: Sube imágenes y metadatos NFT a Pinata IPFS usando curl. Flujo completo end-to-end - sube imágenes individuales desde images/, obtiene CIDs, actualiza los JSON de metadata/ con los CIDs reales de las imágenes, y sube la carpeta metadata/ completa. Usa este skill siempre que el usuario pida subir NFTs a IPFS, subir imágenes a Pinata, subir metadata a IPFS, hacer upload de una colección NFT, o cualquier variación como "sube esto a Pinata", "upload to IPFS", "sube las imágenes y metadata", "pin to IPFS", "publica los NFTs en IPFS". También se activa cuando el usuario dice "actualiza los CIDs de las imágenes en los JSON" o "necesito los IPFS hashes de mis NFTs".
---

# Pinata NFT Upload

Automatiza el upload de colecciones NFT a Pinata IPFS via API REST con curl.

## Requisitos previos

1. **PINATA_JWT** debe estar definido como variable de entorno o en un archivo `.env` en la raíz del proyecto
2. Las imágenes deben estar en la carpeta `images/` del proyecto
3. Los archivos JSON de metadata deben estar en la carpeta `metadata/` del proyecto
4. Los JSON de metadata deben tener un campo `"image"` que se actualizará con el CID real

Si falta alguno de estos requisitos, informar al usuario y ayudarle a configurarlo antes de continuar.

## Flujo de ejecución

### Paso 1: Verificar autenticación

Cargar el JWT y verificar que es válido:

```bash
# Cargar desde .env si existe
source .env 2>/dev/null
curl -s -H "Authorization: Bearer $PINATA_JWT" \
  https://api.pinata.cloud/data/testAuthentication
```

Si falla, pedir al usuario que configure su `PINATA_JWT`.

### Paso 2: Subir imágenes individuales

Para cada archivo en `images/`, subir a Pinata V3 y capturar el CID:

```bash
curl -s -X POST https://uploads.pinata.cloud/v3/files \
  -H "Authorization: Bearer $PINATA_JWT" \
  -F "file=@images/<filename>" \
  -F "network=public"
```

La respuesta devuelve el CID en `data.cid`. Guardar un mapeo de `nombre_archivo → CID`.

Formato de URI IPFS para las imágenes: `ipfs://<CID>`

### Paso 3: Actualizar los JSON de metadata

Para cada archivo `metadata/N.json`, actualizar el campo `"image"` con el CID correspondiente de la imagen subida en el paso anterior.

La correspondencia entre imagen y metadata se hace por nombre de archivo:
- `images/0.png` → `metadata/0.json`
- `images/1.jpg` → `metadata/1.json`
- Si los nombres no son numéricos, buscar coincidencia por nombre base (sin extensión)

Usar `jq` o python para actualizar el campo image de forma segura (sin corromper el JSON):

```bash
jq --arg img "ipfs://<CID>" '.image = $img' metadata/0.json > tmp.json && mv tmp.json metadata/0.json
```

### Paso 4: Subir la carpeta metadata/

Usar el endpoint legacy (V3 no soporta carpetas) para subir toda la carpeta metadata/:

```bash
# Construir dinámicamente los -F para cada archivo en metadata/
curl -s -X POST https://api.pinata.cloud/pinning/pinFileToIPFS \
  -H "Authorization: Bearer $PINATA_JWT" \
  -F "file=@metadata/0.json;filename=metadata/0.json" \
  -F "file=@metadata/1.json;filename=metadata/1.json" \
  -F 'pinataMetadata={"name":"<project-name>-metadata"}' \
  -F 'pinataOptions={"cidVersion":1}'
```

La respuesta devuelve el hash base en `IpfsHash`. La URI base de la colección es: `ipfs://<IpfsHash>/metadata/`

### Paso 5: Reportar resultados

Al finalizar, mostrar un resumen claro al usuario:

```
Upload completado:

Imágenes:
  - 0.png → ipfs://bafy...abc
  - 1.jpg → ipfs://bafy...def

Metadata:
  - Base URI: ipfs://bafy...xyz/metadata/
  - Token 0: ipfs://bafy...xyz/metadata/0.json
  - Token 1: ipfs://bafy...xyz/metadata/1.json
```

## Manejo de errores

- Si un upload falla, reintentar una vez antes de reportar el error
- Si `jq` no está disponible, usar `python3 -c` como fallback para manipular JSON
- Nunca commitear el archivo `.env` — verificar que está en `.gitignore`

## Seguridad

- NUNCA mostrar o loguear el valor completo del JWT en la salida
- Verificar que `.env` está en `.gitignore` antes de cualquier operación
- Si `.env` no está en `.gitignore`, añadirlo automáticamente y avisar al usuario
