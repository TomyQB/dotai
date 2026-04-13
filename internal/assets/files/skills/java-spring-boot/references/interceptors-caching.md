# Interceptors y Caching

## Table of Contents
1. [HandlerInterceptor](#handlerinterceptor)
2. [Control de Concurrencia con Semaphore](#control-de-concurrencia-con-semaphore)
3. [Lectura de Headers Custom y MDC](#lectura-de-headers-custom-y-mdc)
4. [ClientHttpRequestInterceptor](#clienthttprequestinterceptor)
5. [Cacheable con Cache Manager](#cacheable-con-cache-manager)
6. [Eviccion de Cache](#eviccion-de-cache)

---

## HandlerInterceptor

Intercepta peticiones HTTP entrantes a nivel de Spring MVC (despues de DispatcherServlet, antes del Controller). Diferente de los `Filter` de Servlet que operan a nivel mas bajo.

### Implementacion

```java
@Component
@Slf4j
public class RequestAuditInterceptor implements HandlerInterceptor {

    @Override
    public boolean preHandle(final HttpServletRequest request,
                             final HttpServletResponse response,
                             final Object handler) {
        final var startTime = System.currentTimeMillis();
        request.setAttribute("startTime", startTime);
        log.info("Request: {} {}", request.getMethod(), request.getRequestURI());
        return true;
    }

    @Override
    public void afterCompletion(final HttpServletRequest request,
                                final HttpServletResponse response,
                                final Object handler,
                                final Exception ex) {
        final var startTime = (Long) request.getAttribute("startTime");
        final var duration = System.currentTimeMillis() - startTime;
        log.info("Response: {} {} - {}ms - Status {}",
            request.getMethod(), request.getRequestURI(), duration, response.getStatus());
    }
}
```

### Registro en WebMvcConfigurer

```java
@Configuration
@RequiredArgsConstructor
public class WebMvcConfig implements WebMvcConfigurer {

    private final RequestAuditInterceptor requestAuditInterceptor;
    private final RequestLockInterceptor requestLockInterceptor;
    private final TraceIdInterceptor traceIdInterceptor;

    @Override
    public void addInterceptors(final InterceptorRegistry registry) {
        registry.addInterceptor(traceIdInterceptor)
            .addPathPatterns("/api/**");

        registry.addInterceptor(requestAuditInterceptor)
            .addPathPatterns("/api/**")
            .excludePathPatterns("/api/health", "/api/actuator/**");

        registry.addInterceptor(requestLockInterceptor)
            .addPathPatterns("/api/v1/register/**");
    }
}
```

**Reglas:**
- `HandlerInterceptor` para logica a nivel de Spring MVC (tiene acceso al handler)
- `preHandle()` retorna `true` para continuar, `false` para bloquear
- `afterCompletion()` se ejecuta SIEMPRE, incluso si hay excepcion — ideal para cleanup
- Registrar en `WebMvcConfigurer.addInterceptors()` con `addPathPatterns`/`excludePathPatterns`
- Orden de registro = orden de ejecucion (el primero registrado es el primero en ejecutarse)
- Interceptors como `@Component` inyectados en la configuracion

---

## Control de Concurrencia con Semaphore

Patron para limitar peticiones concurrentes a endpoints criticos usando `Semaphore` dentro de un interceptor.

```java
@Component
@Slf4j
public class RequestLockInterceptor implements HandlerInterceptor {

    private final ConcurrentMap<String, Semaphore> endpointSemaphores = new ConcurrentHashMap<>();
    private static final int MAX_CONCURRENT_REQUESTS = 1;

    @Override
    public boolean preHandle(final HttpServletRequest request,
                             final HttpServletResponse response,
                             final Object handler) throws IOException {
        final var traceId = request.getHeader("X-Openpay-Trace-Id");
        final var lockKey = "%s_%s".formatted(request.getRequestURI(), traceId);

        final var semaphore = endpointSemaphores
            .computeIfAbsent(lockKey, key -> new Semaphore(MAX_CONCURRENT_REQUESTS));

        if (!semaphore.tryAcquire()) {
            log.warn("Concurrent request blocked: {} traceId={}", request.getRequestURI(), traceId);
            response.sendError(HttpStatus.TOO_MANY_REQUESTS.value(),
                "Concurrent request in progress");
            return false;
        }

        request.setAttribute("lockKey", lockKey);
        return true;
    }

    @Override
    public void afterCompletion(final HttpServletRequest request,
                                final HttpServletResponse response,
                                final Object handler,
                                final Exception ex) {
        final var lockKey = (String) request.getAttribute("lockKey");
        if (StringUtils.isNotBlank(lockKey)) {
            final var semaphore = endpointSemaphores.get(lockKey);
            if (Objects.nonNull(semaphore)) {
                semaphore.release();
            }
        }
    }
}
```

**Reglas:**
- `Semaphore.tryAcquire()` (sin bloqueo) en `preHandle()`, `release()` en `afterCompletion()`
- SIEMPRE liberar en `afterCompletion()` — se ejecuta incluso con excepciones
- Usar `ConcurrentHashMap` para thread-safety del mapa de semaforos
- `computeIfAbsent()` para creacion atomica del semaforo
- Considerar limpieza periodica del mapa para evitar memory leaks en larga ejecucion

---

## Lectura de Headers Custom y MDC

Patron para capturar headers de tracing y propagarlos a traves del MDC (Mapped Diagnostic Context) de SLF4J.

```java
@Component
@Slf4j
public class TraceIdInterceptor implements HandlerInterceptor {

    private static final String TRACE_HEADER = "X-Openpay-Trace-Id";
    private static final String MDC_TRACE_KEY = "traceId";

    @Override
    public boolean preHandle(final HttpServletRequest request,
                             final HttpServletResponse response,
                             final Object handler) {
        final var traceId = request.getHeader(TRACE_HEADER);
        if (StringUtils.isNotBlank(traceId)) {
            MDC.put(MDC_TRACE_KEY, traceId);
        } else {
            MDC.put(MDC_TRACE_KEY, UUID.randomUUID().toString());
        }
        return true;
    }

    @Override
    public void afterCompletion(final HttpServletRequest request,
                                final HttpServletResponse response,
                                final Object handler,
                                final Exception ex) {
        MDC.remove(MDC_TRACE_KEY);
    }
}
```

### Uso en logback pattern

```xml
<pattern>%d{yyyy-MM-dd HH:mm:ss} [%X{traceId}] %-5level %logger{36} - %msg%n</pattern>
```

**Reglas:**
- SIEMPRE limpiar MDC en `afterCompletion()` para evitar leaks entre requests
- Si no viene el header, generar un UUID como fallback (nunca dejar el traceId vacio)
- Nombres de header y MDC key como constantes `private static final`
- `%X{traceId}` en el patron de logback para incluir el trace ID en todos los logs

---

## ClientHttpRequestInterceptor

Intercepta peticiones HTTP **salientes** de `RestTemplate` o `RestClient`. Util para logging de llamadas a otros microservicios.

```java
@Component
@Slf4j
public class OutboundLoggingInterceptor implements ClientHttpRequestInterceptor {

    @Override
    public ClientHttpResponse intercept(final HttpRequest request,
                                         final byte[] body,
                                         final ClientHttpRequestExecution execution)
            throws IOException {
        logRequest(request, body);
        final var response = execution.execute(request, body);
        logResponse(response);
        return response;
    }

    private void logRequest(final HttpRequest request, final byte[] body) {
        log.debug("Outbound request: {} {}", request.getMethod(), request.getURI());
        if (body.length > 0) {
            log.debug("Request body: {}", new String(body, StandardCharsets.UTF_8));
        }
    }

    private void logResponse(final ClientHttpResponse response) throws IOException {
        log.debug("Outbound response: {}", response.getStatusCode());
    }
}
```

### Registro en RestClient

```java
@Bean(name = "paymentsRestClient")
RestClient paymentsRestClient(final OutboundLoggingInterceptor loggingInterceptor) {
    return RestClient.builder()
        .baseUrl(paymentsProperties.baseUrl())
        .requestInterceptor(loggingInterceptor)
        .build();
}
```

**Reglas:**
- `ClientHttpRequestInterceptor` para peticiones salientes (RestTemplate/RestClient)
- `HandlerInterceptor` para peticiones entrantes (Spring MVC)
- No confundirlos — son interfaces diferentes con propositos diferentes
- Nivel `DEBUG` para logging de body (evitar ruido en produccion)
- Nunca loguear headers de autorizacion ni datos sensibles

---

## Cacheable con Cache Manager

Usar `@Cacheable` para cachear resultados de operaciones costosas (consultas frecuentes, configuraciones, datos que cambian poco).

### Configuracion del CacheManager

```java
@Configuration
@EnableCaching
public class CacheConfig {

    @Bean
    @Primary
    public CacheManager defaultCacheManager() {
        final var caffeine = Caffeine.newBuilder()
            .maximumSize(500)
            .expireAfterWrite(Duration.ofMinutes(10));

        return new CaffeineCacheManager() {{
            setCaffeine(caffeine);
            setCacheNames(List.of("merchants", "configurations"));
        }};
    }

    @Bean(name = "longLivedCacheManager")
    public CacheManager longLivedCacheManager() {
        final var caffeine = Caffeine.newBuilder()
            .maximumSize(100)
            .expireAfterWrite(Duration.ofHours(1));

        return new CaffeineCacheManager() {{
            setCaffeine(caffeine);
            setCacheNames(List.of("roles", "permissions"));
        }};
    }
}
```

### Uso en servicios

```java
@Service
@RequiredArgsConstructor
public class MerchantConfigService {

    private final MerchantConfigRepository merchantConfigRepository;
    private final MerchantConfigMapper merchantConfigMapper;

    @Cacheable(value = "merchants", key = "#merchantId")
    @Transactional(readOnly = true)
    public MerchantConfigResponse findByMerchantId(final String merchantId) {
        final var config = merchantConfigRepository.findByMerchantIdOrThrow(merchantId);
        return merchantConfigMapper.toResponse(config);
    }

    @Cacheable(value = "roles", cacheManager = "longLivedCacheManager")
    @Transactional(readOnly = true)
    public List<String> getRolesForPath(final String path) {
        return roleRepository.findRolesByPath(path);
    }
}
```

**Reglas:**
- `@EnableCaching` en la clase de configuracion
- `@Cacheable(value = "cacheName")` — el valor es el nombre del cache, NO la key
- `key = "#parametro"` para definir la key (SpEL). Si el metodo tiene un solo parametro, es la key por defecto
- `cacheManager = "nombreBean"` cuando existen multiples cache managers
- Cachear SOLO datos que cambian poco (configuraciones, roles, catalgos)
- Nunca cachear datos transaccionales (saldos, estados de pedidos)
- Siempre definir `expireAfterWrite` — cache sin expiracion es un bug en espera

---

## Eviccion de Cache

Toda entrada `@Cacheable` DEBE tener su correspondiente estrategia de eviccion. Sin eviccion, el cache servira datos obsoletos indefinidamente.

### @CacheEvict en operaciones de escritura

```java
@Service
@RequiredArgsConstructor
public class MerchantConfigService {

    @CacheEvict(value = "merchants", key = "#merchantId")
    @Transactional
    public MerchantConfigResponse update(final String merchantId,
                                          final UpdateMerchantConfigRequest request) {
        final var config = merchantConfigRepository.findByMerchantIdOrThrow(merchantId);
        merchantConfigMapper.updateEntity(config, request);
        return merchantConfigMapper.toResponse(merchantConfigRepository.save(config));
    }

    @CacheEvict(value = "merchants", allEntries = true)
    @Transactional
    public void refreshAllConfigs() {
        log.info("All merchant configs evicted from cache");
    }
}
```

### Eviccion programatica

```java
@Service
@RequiredArgsConstructor
public class CacheAdminService {

    private final CacheManager cacheManager;

    public void evictSingle(final String cacheName, final String key) {
        final var cache = cacheManager.getCache(cacheName);
        if (Objects.nonNull(cache)) {
            cache.evict(key);
        }
    }

    public void evictAll(final String cacheName) {
        final var cache = cacheManager.getCache(cacheName);
        if (Objects.nonNull(cache)) {
            cache.clear();
        }
    }
}
```

**Reglas:**
- `@CacheEvict` en toda operacion que modifique datos cacheados (update, delete)
- `allEntries = true` para invalidar todo el cache de golpe (usar con prudencia)
- `key = "#parametro"` debe coincidir con la key usada en `@Cacheable`
- Eviccion programatica solo cuando la logica de invalidacion es compleja
- Patron: `@Cacheable` en lectura, `@CacheEvict` en escritura — siempre en pares
