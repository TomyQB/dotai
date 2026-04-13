# Checklist OWASP ASVS 4.0 — Web/Backend

Referencia detallada para auditorías de seguridad. Aplica solo las categorías relevantes al código bajo revisión.

## Índice

- [V1 - Arquitectura y Diseño](#v1---arquitectura-y-diseño)
- [V2 - Autenticación](#v2---autenticación)
- [V3 - Gestión de Sesiones](#v3---gestión-de-sesiones)
- [V4 - Control de Acceso](#v4---control-de-acceso)
- [V5 - Validación y Sanitización](#v5---validación-y-sanitización)
- [V6 - Criptografía](#v6---criptografía)
- [V7 - Errores y Logging](#v7---errores-y-logging)
- [V8 - Protección de Datos](#v8---protección-de-datos)
- [V9 - Comunicaciones](#v9---comunicaciones)
- [V10 - Código Malicioso](#v10---código-malicioso)
- [V11 - Lógica de Negocio](#v11---lógica-de-negocio)
- [V12 - Archivos y Recursos](#v12---archivos-y-recursos)
- [V13 - API y Servicios Web](#v13---api-y-servicios-web)
- [V14 - Configuración](#v14---configuración)

---

## V1 - Arquitectura y Diseño

**Checks universales:**
- Separación de responsabilidades (controladores, servicios, acceso a datos)
- Principio de menor privilegio en componentes y servicios
- Validación de entrada en los límites del sistema (no en capas internas)
- Flujos de datos sensibles identificados y protegidos
- Dependencias de terceros justificadas y actualizadas

**Por stack:**
- **Next.js/Node/Express**: Orden correcto de middlewares, uso de helmet.js, separación API routes vs SSR
- **Python/Django/FastAPI**: MIDDLEWARE settings correctos, SECURITY_* configurados, dependencia de Pydantic/serializers para validación
- **Java/Spring Boot**: Cadena de filtros de seguridad configurada, beans con scope apropiado, perfiles de entorno separados
- **Go/Gin**: Middlewares de seguridad aplicados, panic recovery configurado, graceful shutdown implementado

## V2 - Autenticación

**Checks universales:**
- Contraseñas hasheadas con bcrypt/argon2/scrypt (cost factors apropiados)
- No hay credenciales por defecto o hardcodeadas
- Protección contra brute force (rate limiting, lockout temporal)
- Verificación de fortaleza de contraseña en registro
- MFA disponible para operaciones sensibles

**Por stack:**
- **Next.js/Node**: Secretos JWT con longitud suficiente (≥256 bits), expiración de tokens configurada, refresh tokens implementados
- **Python/Django**: AUTH_PASSWORD_VALIDATORS configurados, django.contrib.auth usado correctamente, no custom crypto
- **Java/Spring**: AuthenticationManager configurado, PasswordEncoder con BCrypt, endpoints de auth protegidos con CSRF
- **Go**: Librería de JWT verificada (no custom), middleware de autenticación aplicado consistentemente

## V3 - Gestión de Sesiones

**Checks universales:**
- Tokens de sesión con entropía suficiente (≥128 bits)
- Sesiones invalidadas al hacer logout
- Timeout de inactividad y timeout absoluto configurados
- Rotación de ID de sesión tras autenticación exitosa
- Cookies con flags: HttpOnly, Secure, SameSite

**Por stack:**
- **Next.js/Node**: express-session o next-auth con store externo (no memoria), cookies configuradas con sameSite strict
- **Python/Django**: SESSION_COOKIE_HTTPONLY=True, SESSION_COOKIE_SECURE=True, SESSION_ENGINE con backend persistente
- **Java/Spring**: HttpSession con timeout, SessionCreationPolicy configurada, CSRF token vinculado a sesión
- **Go**: gorilla/sessions o similar con store seguro, no sesiones en memoria para producción

## V4 - Control de Acceso

**Checks universales:**
- Principio de deny-by-default (todo prohibido salvo lo explícitamente permitido)
- Autorización verificada en el servidor, nunca solo en el cliente
- No hay IDOR (Insecure Direct Object References) — validar propiedad del recurso
- Escalación de privilegios vertical y horizontal prevenida
- Acceso basado en roles (RBAC) o atributos (ABAC) implementado consistentemente

**Por stack:**
- **Next.js/Node**: Middleware de autorización aplicado a todas las rutas protegidas, no solo autenticación
- **Python/Django**: Decoradores @login_required y @permission_required usados, IsAuthenticated/permissions en DRF
- **Java/Spring**: @PreAuthorize/@Secured en métodos, SecurityFilterChain con antMatchers/requestMatchers apropiados
- **Go**: Middleware de autorización separado del de autenticación, checks de permisos antes de acceso a recursos

## V5 - Validación y Sanitización

**Checks universales:**
- Toda entrada del usuario validada en tipo, longitud, rango y formato
- Queries parametrizadas (nunca concatenación de strings para SQL)
- Codificación de salida contextual (HTML, JS, URL, CSS, SQL)
- Protección contra XSS (reflejado, almacenado, DOM-based)
- Protección contra inyecciones: SQL, NoSQL, LDAP, OS command, template injection

**Por stack:**
- **Next.js/Node**: Uso de ORM (Prisma, Sequelize) con queries parametrizadas, DOMPurify o similar para output en SSR
- **Python/Django**: ORM de Django para queries, template engine con auto-escape, bleach para HTML user-generated
- **Java/Spring**: JPA/Hibernate con named parameters, Thymeleaf con escape automático, @Valid con Bean Validation
- **Go**: sqlx o GORM con placeholders, html/template con escape automático, validación con go-playground/validator

## V6 - Criptografía

**Checks universales:**
- Algoritmos de cifrado actuales (AES-256-GCM, ChaCha20-Poly1305) — no DES, 3DES, RC4, MD5
- Claves y secretos nunca en código fuente ni en control de versiones
- Generación de números aleatorios criptográficamente segura (CSPRNG)
- Gestión de claves: rotación, almacenamiento seguro (vault, KMS, variables de entorno)
- Hashing de contraseñas con salt y algoritmo adecuado (bcrypt, argon2id)

**Por stack:**
- **Next.js/Node**: crypto.randomBytes() para aleatorios, no Math.random() para seguridad
- **Python**: secrets module para aleatorios, hashlib con algoritmos seguros, no pickle para datos no confiables
- **Java**: SecureRandom para aleatorios, KeyStore para almacenamiento de claves, JCE con providers actualizados
- **Go**: crypto/rand para aleatorios, no math/rand para seguridad, golang.org/x/crypto para bcrypt/argon2

## V7 - Errores y Logging

**Checks universales:**
- Excepciones manejadas gracefully (no stack traces expuestos al usuario)
- Logging de eventos de seguridad: login exitoso/fallido, cambios de permisos, acceso a datos sensibles
- No loguear datos sensibles (contraseñas, tokens, PII, números de tarjeta)
- Mensajes de error genéricos para el usuario (no revelar detalles internos)
- Logs protegidos contra inyección de logs (sanitizar input antes de loguear)

**Por stack:**
- **Next.js/Node**: Error handler global configurado, winston/pino sin datos sensibles, NODE_ENV=production sin stack traces
- **Python/Django**: DEBUG=False en producción, LOGGING configurado, logging.exception() sin datos de usuario
- **Java/Spring**: @ControllerAdvice para manejo global de errores, SLF4J/Logback configurado, no e.printStackTrace()
- **Go**: Middleware de recovery para panics, structured logging (zerolog/zap), errores envueltos con context

## V8 - Protección de Datos

**Checks universales:**
- Datos sensibles (PII, financieros, salud) identificados y clasificados
- Datos sensibles cifrados en reposo
- No cachear datos sensibles en el cliente (Cache-Control: no-store)
- Datos sensibles no incluidos en URLs (query parameters)
- Cumplimiento con regulaciones aplicables (GDPR, CCPA)

**Por stack:**
- **Next.js/Node**: Variables de entorno para secrets (.env no commiteado), no exponer datos sensibles en getServerSideProps hacia el cliente
- **Python/Django**: SECRET_KEY protegido, SECURE_BROWSER_XSS_FILTER, datos PII con cifrado a nivel de campo si aplica
- **Java/Spring**: @JsonIgnore para campos sensibles en respuestas, jasypt o similar para cifrado de propiedades
- **Go**: Struct tags `json:"-"` para campos sensibles, cifrado de campos en repositorio/DAO layer

## V9 - Comunicaciones

**Checks universales:**
- TLS 1.2+ obligatorio para todas las comunicaciones externas
- Certificados SSL válidos y verificados (no skip de verificación)
- HSTS (HTTP Strict Transport Security) configurado
- No mixed content (HTTP recursos en páginas HTTPS)
- Comunicaciones internas entre servicios cifradas si cruzan redes no confiables

**Por stack:**
- **Next.js/Node**: Redirección HTTP→HTTPS, helmet con HSTS, no process.env.NODE_TLS_REJECT_UNAUTHORIZED='0'
- **Python/Django**: SECURE_SSL_REDIRECT=True, SECURE_HSTS_SECONDS configurado, requests con verify=True
- **Java/Spring**: server.ssl configurado, HttpsURLConnection sin bypass de certificados, HSTS en headers
- **Go**: http.ListenAndServeTLS, no InsecureSkipVerify en tls.Config para producción

## V10 - Código Malicioso

**Checks universales:**
- No hay backdoors, rutas de acceso ocultas o funcionalidad no documentada
- No hay bombas lógicas, time bombs o condiciones de activación sospechosas
- No hay comunicaciones con servidores externos no documentados
- Dependencias de terceros verificadas (checksums, lock files)
- No hay código ofuscado sin justificación clara

## V11 - Lógica de Negocio

**Checks universales:**
- Flujos de negocio siguen el orden esperado (no se pueden saltar pasos)
- Race conditions prevenidas en operaciones críticas (transacciones, pagos)
- Límites y cuotas implementados (cantidad máxima, frecuencia, volumen)
- Validación de invariantes de negocio en el servidor
- Idempotencia en operaciones que lo requieran (pagos, creación de recursos)

**Por stack:**
- **Next.js/Node**: Transacciones de BD para operaciones atómicas, mutex/locks para operaciones concurrentes críticas
- **Python/Django**: select_for_update() para prevenir race conditions, @transaction.atomic para operaciones críticas
- **Java/Spring**: @Transactional con isolation level apropiado, optimistic/pessimistic locking con JPA
- **Go**: sync.Mutex para recursos compartidos, database transactions con serializable isolation si aplica

## V12 - Archivos y Recursos

**Checks universales:**
- Tipos de archivo permitidos validados (whitelist, no blacklist)
- Tamaño máximo de archivo configurado
- Archivos subidos almacenados fuera del webroot
- Path traversal prevenido (validar que la ruta no sale del directorio permitido)
- Nombres de archivo sanitizados (no usar el nombre original del usuario directamente)

**Por stack:**
- **Next.js/Node**: multer con limits y fileFilter, almacenamiento en S3/GCS con URLs presignadas
- **Python/Django**: FILE_UPLOAD_MAX_MEMORY_SIZE, FileExtensionValidator, MEDIA_ROOT fuera de STATIC_ROOT
- **Java/Spring**: MultipartFile con validación, MAX_FILE_SIZE en configuración, storage service separado
- **Go**: r.Body con MaxBytesReader, filepath.Clean() para prevenir traversal, storage en servicio externo

## V13 - API y Servicios Web

**Checks universales:**
- Autenticación requerida en todos los endpoints no públicos
- Rate limiting implementado (por IP, por usuario, por endpoint)
- Paginación en endpoints que devuelven listas (no devolver todo)
- Versionado de API implementado
- CORS configurado restrictivamente (no wildcard * en producción)
- Input validation en todos los parámetros de request (body, query, path, headers)

**Por stack:**
- **Next.js/Node**: CORS con origin específico, express-rate-limit, validación con zod/joi en API routes
- **Python/Django**: DRF throttling configurado, CORS_ALLOWED_ORIGINS específicos, Serializer validation
- **Java/Spring**: @CrossOrigin con origins específicos, RateLimiter (Resilience4j/Bucket4j), @Valid en @RequestBody
- **Go**: CORS middleware con origins específicos, rate limiter (golang.org/x/time/rate), validación de structs en handlers

## V14 - Configuración

**Checks universales:**
- Cabeceras de seguridad configuradas: X-Content-Type-Options, X-Frame-Options, Content-Security-Policy
- Modo debug/desarrollo desactivado en producción
- Secrets gestionados via variables de entorno o vault (no en código)
- Dependencias sin vulnerabilidades conocidas (verificar con herramientas de auditoría)
- Configuraciones por defecto seguras (no puertos abiertos innecesarios, no servicios expuestos sin necesidad)

**Por stack:**
- **Next.js/Node**: helmet.js configurado, next.config.js con headers de seguridad, npm audit / yarn audit limpio
- **Python/Django**: ALLOWED_HOSTS configurado, DEBUG=False, SECURE_* settings habilitados, pip-audit limpio
- **Java/Spring**: Security headers en SecurityFilterChain, actuator endpoints protegidos, dependency-check sin CVEs
- **Go**: Security headers en middleware, go vuln check limpio, configuración de producción separada de desarrollo
