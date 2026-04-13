# REST Clients - Comunicacion entre Microservicios

## Table of Contents
1. [RestClient Beans Nombrados](#restclient-beans-nombrados)
2. [ConfigurationProperties para URLs](#configurationproperties-para-urls)
3. [Configuracion SSL/TLS Custom](#configuracion-ssltls-custom)
4. [WebClient con Retry y Timeouts](#webclient-con-retry-y-timeouts)
5. [ExchangeFilterFunction](#exchangefilterfunction)
6. [WebClientTemplate (Clase Base Abstracta)](#webclienttemplate-clase-base-abstracta)
7. [Cuando usar RestClient vs WebClient](#cuando-usar-restclient-vs-webclient)

---

## RestClient Beans Nombrados

Crear un `@Bean` nombrado por cada microservicio externo. Cada bean tiene su propia configuracion de base URL, timeouts y filtros.

```java
@Configuration
@RequiredArgsConstructor
public class RestClientConfiguration {

    private final PaymentsClientProperties paymentsProperties;
    private final NotificationsClientProperties notificationsProperties;

    @Bean(name = "paymentsRestClient")
    RestClient paymentsRestClient() {
        return RestClient.builder()
            .baseUrl(paymentsProperties.baseUrl())
            .defaultHeader(HttpHeaders.CONTENT_TYPE, MediaType.APPLICATION_JSON_VALUE)
            .requestFactory(createRequestFactory(paymentsProperties))
            .build();
    }

    @Bean(name = "notificationsRestClient")
    RestClient notificationsRestClient() {
        return RestClient.builder()
            .baseUrl(notificationsProperties.baseUrl())
            .defaultHeader(HttpHeaders.CONTENT_TYPE, MediaType.APPLICATION_JSON_VALUE)
            .requestFactory(createRequestFactory(notificationsProperties))
            .build();
    }

    private ClientHttpRequestFactory createRequestFactory(final ClientProperties properties) {
        final var factory = new SimpleClientHttpRequestFactory();
        factory.setConnectTimeout(properties.connectTimeout());
        factory.setReadTimeout(properties.readTimeout());
        return factory;
    }
}
```

### Inyeccion con @Qualifier

```java
@Service
@RequiredArgsConstructor
public class PaymentIntegrationService {

    @Qualifier("paymentsRestClient")
    private final RestClient paymentsRestClient;

    public PaymentResponse processPayment(final PaymentRequest request) {
        return paymentsRestClient.post()
            .uri("/api/v1/payments")
            .body(request)
            .retrieve()
            .body(PaymentResponse.class);
    }
}
```

**Reglas:**
- Un `@Bean` por microservicio externo, con nombre descriptivo
- `@Qualifier` para inyeccion sin ambiguedad
- Base URL siempre desde `@ConfigurationProperties`, nunca hardcodeada
- Timeouts obligatorios en cada bean (connect + read)
- `defaultHeader` para headers comunes (Content-Type, Accept)

---

## ConfigurationProperties para URLs

Usar records con `@ConfigurationProperties` para centralizar la configuracion de cada cliente externo.

```java
@ConfigurationProperties(prefix = "client.payments")
public record PaymentsClientProperties(
    @DefaultValue("http://localhost:8081") String baseUrl,
    @DefaultValue("PT5S") Duration connectTimeout,
    @DefaultValue("PT30S") Duration readTimeout
) {}

@ConfigurationProperties(prefix = "client.notifications")
public record NotificationsClientProperties(
    @DefaultValue("http://localhost:8082") String baseUrl,
    @DefaultValue("PT5S") Duration connectTimeout,
    @DefaultValue("PT15S") Duration readTimeout
) {}
```

### application.yml

```yaml
client:
  payments:
    base-url: https://payments-api.internal.company.com
    connect-timeout: PT5S
    read-timeout: PT30S
  notifications:
    base-url: https://notifications-api.internal.company.com
    connect-timeout: PT3S
    read-timeout: PT10S
```

### Habilitar en Application

```java
@SpringBootApplication
@EnableConfigurationProperties({
    PaymentsClientProperties.class,
    NotificationsClientProperties.class
})
public class Application { }
```

**Reglas:**
- Record inmutable con `@DefaultValue` para valores por defecto sensatos
- Prefijo del namespace claro: `client.<nombre-servicio>`
- `Duration` para timeouts (ISO-8601: `PT5S` = 5 segundos)
- Habilitar con `@EnableConfigurationProperties` en la clase principal

---

## Configuracion SSL/TLS Custom

Para APIs externas que requieren certificados custom o verificacion SSL personalizada.

```java
@Configuration
@RequiredArgsConstructor
public class ExternalApiRestClientConfig {

    private final ExternalApiProperties externalApiProperties;

    @Bean(name = "externalApiRestClient")
    RestClient externalApiRestClient() throws GeneralSecurityException {
        final var sslContext = SSLContextBuilder.create()
            .loadTrustMaterial(externalApiProperties.trustStorePath(),
                externalApiProperties.trustStorePassword().toCharArray())
            .build();

        final var httpClient = HttpClientBuilder.create()
            .setSSLContext(sslContext)
            .build();

        final var requestFactory = new HttpComponentsClientHttpRequestFactory(httpClient);
        requestFactory.setConnectTimeout(externalApiProperties.connectTimeout());
        requestFactory.setReadTimeout(externalApiProperties.readTimeout());

        return RestClient.builder()
            .baseUrl(externalApiProperties.baseUrl())
            .requestFactory(requestFactory)
            .build();
    }
}
```

**Reglas:**
- Nunca deshabilitar la verificacion SSL en produccion (`TrustAllStrategy` solo para desarrollo)
- Certificados y passwords desde propiedades externas, nunca en codigo
- Usar `SSLContextBuilder` de Apache HttpClient (no construir SSLContext manualmente)
- Documentar por que se necesita SSL custom (que API lo requiere)

---

## WebClient con Retry y Timeouts

Usar `WebClient` cuando se necesiten llamadas reactivas, retry con backoff, o procesamiento no-bloqueante.

### Configuracion del WebClient

```java
@Configuration
@RequiredArgsConstructor
public class WebClientConfiguration {

    private final KycServiceProperties kycProperties;

    @Bean(name = "kycWebClient")
    WebClient kycWebClient() {
        final var httpClient = reactor.netty.http.client.HttpClient.create()
            .option(ChannelOption.CONNECT_TIMEOUT_MILLIS,
                (int) kycProperties.connectTimeout().toMillis())
            .responseTimeout(kycProperties.readTimeout());

        return WebClient.builder()
            .baseUrl(kycProperties.baseUrl())
            .clientConnector(new ReactorClientHttpConnector(httpClient))
            .defaultHeader(HttpHeaders.CONTENT_TYPE, MediaType.APPLICATION_JSON_VALUE)
            .build();
    }
}
```

### Llamada con Retry

```java
@Service
@RequiredArgsConstructor
public class KycVerificationService {

    @Qualifier("kycWebClient")
    private final WebClient kycWebClient;

    private static final int MAX_RETRIES = 3;
    private static final Duration MIN_BACKOFF = Duration.ofMillis(500);

    public KycResponse verify(final KycRequest request) {
        return kycWebClient.post()
            .uri("/api/v1/verify")
            .bodyValue(request)
            .retrieve()
            .onStatus(HttpStatusCode::is5xxServerError,
                response -> Mono.error(new KycServiceUnavailableException()))
            .bodyToMono(KycResponse.class)
            .retryWhen(Retry.backoff(MAX_RETRIES, MIN_BACKOFF)
                .filter(this::isRetryableException)
                .onRetryExhaustedThrow((spec, signal) ->
                    new KycServiceUnavailableException("KYC service unavailable after retries")))
            .block();
    }

    private boolean isRetryableException(final Throwable throwable) {
        return throwable instanceof ConnectTimeoutException
            || throwable instanceof ReadTimeoutException
            || throwable instanceof KycServiceUnavailableException;
    }
}
```

**Reglas:**
- `Retry.backoff()` para reintentos con exponential backoff
- `.filter()` para reintentar solo excepciones transitorias (timeout, 503, 504)
- `.onRetryExhaustedThrow()` para convertir a excepcion de dominio
- `.block()` solo en contextos no-reactivos (servicios Spring MVC convencionales)
- Nunca reintentar errores 4xx (son errores del cliente, no transitorios)
- Constantes `private static final` para max retries y backoff

---

## ExchangeFilterFunction

Filtros que interceptan peticiones y respuestas del WebClient. Utiles para logging, headers custom, o metricas.

### Filtro de Logging

```java
@Component
@Slf4j
public class WebClientLoggingFilter {

    public ExchangeFilterFunction logRequest() {
        return ExchangeFilterFunction.ofRequestProcessor(request -> {
            log.debug("HTTP {} {}", request.method(), request.url());
            return Mono.just(request);
        });
    }

    public ExchangeFilterFunction logResponse() {
        return ExchangeFilterFunction.ofResponseProcessor(response -> {
            log.debug("HTTP Response: {}", response.statusCode());
            return Mono.just(response);
        });
    }
}
```

### Filtro de Header Custom

```java
@Component
public class TraceIdFilter {

    public ExchangeFilterFunction addTraceId() {
        return ExchangeFilterFunction.ofRequestProcessor(request -> {
            final var traceId = MDC.get("traceId");
            if (StringUtils.isNotBlank(traceId)) {
                return Mono.just(ClientRequest.from(request)
                    .header("X-Trace-Id", traceId)
                    .build());
            }
            return Mono.just(request);
        });
    }
}
```

### Registro de filtros en WebClient

```java
@Bean(name = "kycWebClient")
WebClient kycWebClient(final WebClientLoggingFilter loggingFilter,
                        final TraceIdFilter traceIdFilter) {
    return WebClient.builder()
        .baseUrl(kycProperties.baseUrl())
        .filter(traceIdFilter.addTraceId())
        .filter(loggingFilter.logRequest())
        .filter(loggingFilter.logResponse())
        .build();
}
```

**Reglas:**
- Filtros como `@Component` reutilizable, no lambdas anonimas en la configuracion
- Orden importa: headers primero, luego logging (para capturar los headers inyectados)
- Nunca loguear datos sensibles (Authorization headers, PII) — usar nivel DEBUG para bodies
- `MDC.get()` para propagar trace IDs desde el request entrante al request saliente

---

## WebClientTemplate (Clase Base Abstracta)

Cuando multiples servicios comparten la misma logica de llamadas HTTP (retry, headers, manejo de errores), extraer a una clase base abstracta.

```java
public abstract class WebClientTemplate {

    protected final WebClient webClient;

    private static final int DEFAULT_MAX_RETRIES = 3;
    private static final Duration DEFAULT_MIN_BACKOFF = Duration.ofMillis(500);

    protected WebClientTemplate(final WebClient webClient) {
        this.webClient = webClient;
    }

    protected <T> T post(final String uri, final Object request, final Class<T> responseType) {
        return webClient.post()
            .uri(uri)
            .headers(this::addCustomHeaders)
            .bodyValue(request)
            .retrieve()
            .onStatus(HttpStatusCode::is5xxServerError,
                response -> Mono.error(createServiceException(response)))
            .bodyToMono(responseType)
            .retryWhen(retrySpec())
            .block();
    }

    protected <T> T get(final String uri, final Class<T> responseType) {
        return webClient.get()
            .uri(uri)
            .headers(this::addCustomHeaders)
            .retrieve()
            .onStatus(HttpStatusCode::is5xxServerError,
                response -> Mono.error(createServiceException(response)))
            .bodyToMono(responseType)
            .retryWhen(retrySpec())
            .block();
    }

    protected Retry retrySpec() {
        return Retry.backoff(DEFAULT_MAX_RETRIES, DEFAULT_MIN_BACKOFF)
            .filter(this::isRetryable);
    }

    protected abstract void addCustomHeaders(HttpHeaders headers);

    protected abstract Throwable createServiceException(ClientResponse response);

    protected boolean isRetryable(final Throwable throwable) {
        return throwable instanceof ConnectTimeoutException
            || throwable instanceof ReadTimeoutException;
    }
}
```

### Implementacion concreta

```java
@Component
public class KycClient extends WebClientTemplate {

    public KycClient(@Qualifier("kycWebClient") final WebClient webClient) {
        super(webClient);
    }

    public KycResponse verify(final KycRequest request) {
        return post("/api/v1/verify", request, KycResponse.class);
    }

    @Override
    protected void addCustomHeaders(final HttpHeaders headers) {
        headers.set("X-Api-Key", "configured-api-key");
    }

    @Override
    protected Throwable createServiceException(final ClientResponse response) {
        return new KycServiceUnavailableException("KYC service error: " + response.statusCode());
    }
}
```

**Reglas:**
- Clase abstracta con metodos `post()`, `get()` que encapsulan retry + error handling
- Metodos abstractos para lo que varia por servicio: headers custom, excepcion especifica
- Implementaciones concretas como `@Component` — una por microservicio
- El constructor recibe el `WebClient` ya configurado via `@Qualifier`
- Sobrescribir `retrySpec()` o `isRetryable()` solo si el servicio tiene necesidades especiales

---

## Cuando usar RestClient vs WebClient

| Criterio | RestClient | WebClient |
|----------|-----------|-----------|
| Modelo de programacion | Sincrono (bloqueante) | Reactivo (no-bloqueante) |
| Retry con backoff | Manual (loop + sleep) | `Retry.backoff()` nativo |
| Disponible desde | Spring 6.1 / Spring Boot 3.2 | Spring 5.0 / Spring Boot 2.0 |
| Cuando usarlo | Llamadas simples, sin retry complejo | Retry, circuit breaker, flujos reactivos |
| Complejidad | Baja | Media |

**Regla de decision:**
- Si la llamada es simple y no necesita retry sofisticado → `RestClient`
- Si necesitas retry con backoff, timeout granular, o procesamiento reactivo → `WebClient`
- Si el proyecto ya usa `WebClient` extensivamente → mantener consistencia con `WebClient`
