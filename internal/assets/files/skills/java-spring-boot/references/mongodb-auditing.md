# MongoDB y Auditing

## Table of Contents
1. [MongoRepository y @Query](#mongorepository-y-query)
2. [Entidades con @Document](#entidades-con-document)
3. [Indices con @Indexed](#indices-con-indexed)
4. [AuditMetadata (Clase Base Abstracta)](#auditmetadata-clase-base-abstracta)
5. [Configuracion de Auditing](#configuracion-de-auditing)
6. [AuditorAware Implementacion](#auditoraware-implementacion)

---

## MongoRepository y @Query

`MongoRepository` funciona de forma similar a `JpaRepository`, pero las queries custom usan sintaxis nativa MongoDB (JSON) en lugar de JPQL.

### Repository basico

```java
public interface PartnerRepository extends MongoRepository<PartnerEntity, String> {

    Optional<PartnerEntity> findByMerchantId(String merchantId);

    List<PartnerEntity> findByStatusAndCountry(PartnerStatus status, String country);

    boolean existsByMerchantId(String merchantId);
}
```

### @Query con sintaxis MongoDB

```java
public interface PartnerRepository extends MongoRepository<PartnerEntity, String> {

    @Query("{ 'merchantId': ?0 }")
    Optional<PartnerEntity> findByMerchantId(String merchantId);

    @Query("{ 'status': ?0, 'createdDate': { '$gte': ?1 } }")
    List<PartnerEntity> findByStatusSince(PartnerStatus status, LocalDateTime since);

    @Query(value = "{ 'country': ?0 }", fields = "{ 'merchantId': 1, 'status': 1 }")
    List<PartnerEntity> findSummaryByCountry(String country);

    @Query("{ 'tags': { '$in': ?0 } }")
    List<PartnerEntity> findByTagsIn(List<String> tags);
}
```

### Diferencias con JPA

| Aspecto | JPA (@Query) | MongoDB (@Query) |
|---------|-------------|-----------------|
| Sintaxis | JPQL (`SELECT t FROM Transfer t WHERE...`) | JSON (`{ 'field': ?0 }`) |
| Proyecciones | `SELECT t.id, t.name` | `fields = "{ 'id': 1, 'name': 1 }"` |
| Parametros | `:paramName` o `?1` | `?0`, `?1` (zero-based) |
| Null handling | `IS NULL` | `{ '$exists': false }` o `null` |
| Like | `LIKE %valor%` | `{ '$regex': '.*valor.*', '$options': 'i' }` |

### Repository custom (interface + impl)

Igual que con JPA, cuando se necesita logica adicional:

```java
public interface PartnerRepository {
    PartnerEntity findByMerchantIdOrThrow(String merchantId);
    List<PartnerEntity> findActiveByCountry(String country);
}

@Repository
@RequiredArgsConstructor
public class PartnerRepositoryImpl implements PartnerRepository {

    private final PartnerMongoRepository mongoRepository;

    @Override
    public PartnerEntity findByMerchantIdOrThrow(final String merchantId) {
        return mongoRepository.findByMerchantId(merchantId)
            .orElseThrow(() -> new PartnerNotFoundException(merchantId));
    }

    @Override
    public List<PartnerEntity> findActiveByCountry(final String country) {
        return mongoRepository.findByStatusAndCountry(PartnerStatus.ACTIVE, country);
    }
}
```

**Reglas:**
- Sin `@Repository` en la interfaz (Spring Data la detecta automaticamente)
- `Optional` para busquedas por ID unico, `List` para busquedas multiples
- `@Query` con sintaxis JSON nativa de MongoDB para consultas no triviales
- Los indices de parametros en `@Query` son zero-based (`?0`, `?1`)
- Resolver `Optional` en el Repository custom, el Service recibe tipos concretos
- `fields` para proyecciones cuando no se necesitan todos los campos

---

## Entidades con @Document

Las entidades MongoDB usan `@Document` en lugar de `@Entity`. No necesitan las mismas restricciones de JPA (constructor sin argumentos es opcional si se usa Lombok correctamente).

### Entidad basica

```java
@Document(collection = "partners")
@Getter
@Builder
@NoArgsConstructor(access = AccessLevel.PROTECTED)
@AllArgsConstructor(access = AccessLevel.PRIVATE)
public class PartnerEntity extends AuditMetadata {

    @Id
    private String id;

    @Indexed(unique = true)
    private String merchantId;

    private String name;

    @Indexed
    private PartnerStatus status;

    private String country;

    private ContactInfo contactInfo;

    private List<String> tags;
}
```

### Documentos embebidos

```java
@Getter
@Builder
@NoArgsConstructor(access = AccessLevel.PROTECTED)
@AllArgsConstructor(access = AccessLevel.PRIVATE)
public class ContactInfo {

    private String email;
    private String phone;
    private String address;
}
```

**Reglas:**
- `@Document(collection = "nombre")` con nombre de coleccion SIEMPRE explicito
- `@Id` con tipo `String` (MongoDB genera ObjectId automaticamente)
- Documentos embebidos como clases separadas sin `@Document` (se anidan directamente)
- Preferir embedding sobre `@DBRef` (las referencias son anti-patron en MongoDB)
- `@Field("nombre")` solo si el nombre del campo en MongoDB difiere del campo Java
- Mismas anotaciones Lombok que JPA: `@Getter`, `@Builder`, sin `@Setter`, sin `@Data`
- Extender `AuditMetadata` para auditoria automatica (ver seccion dedicada)

---

## Indices con @Indexed

Los indices son criticos en MongoDB — sin indices, toda consulta es un full collection scan.

### Indice simple

```java
@Indexed
private PartnerStatus status;

@Indexed(unique = true)
private String merchantId;
```

### Indice compuesto

```java
@Document(collection = "partners")
@CompoundIndex(name = "status_country_idx", def = "{ 'status': 1, 'country': 1 }")
public class PartnerEntity extends AuditMetadata {
    // ...
}
```

### Indice con TTL (auto-expiracion)

```java
@Indexed(expireAfterSeconds = 86400) // 24 horas
private LocalDateTime expiresAt;
```

**Reglas:**
- `@Indexed` en CADA campo usado en `@Query` o en derived queries
- `@CompoundIndex` para queries que filtran por multiples campos (el orden importa)
- `unique = true` para constraints de unicidad (reemplaza la logica de validacion manual)
- `@Indexed(expireAfterSeconds = N)` para datos temporales (tokens, sesiones, logs)
- El orden de campos en `@CompoundIndex` debe seguir el patron de consultas mas frecuente
- Demasiados indices degradan las escrituras — indexar solo lo que se consulta

---

## AuditMetadata (Clase Base Abstracta)

Clase base que todas las entidades MongoDB deben extender. Captura automaticamente quien y cuando creo/modifico cada documento.

```java
@Getter
@SuperBuilder
@NoArgsConstructor(access = AccessLevel.PROTECTED)
@AllArgsConstructor(access = AccessLevel.PROTECTED)
public abstract class AuditMetadata {

    @CreatedDate
    private LocalDateTime createdDate;

    @LastModifiedDate
    private LocalDateTime lastModifiedDate;

    @CreatedBy
    private String createdByUser;

    @LastModifiedBy
    private String modifiedByUser;

    @Version
    private Long version;
}
```

### Entidad que extiende AuditMetadata

```java
@Document(collection = "partners")
@Getter
@SuperBuilder
@NoArgsConstructor(access = AccessLevel.PROTECTED)
@AllArgsConstructor(access = AccessLevel.PRIVATE)
public class PartnerEntity extends AuditMetadata {

    @Id
    private String id;

    @Indexed(unique = true)
    private String merchantId;

    private String name;

    @Indexed
    private PartnerStatus status;
}
```

**Reglas:**
- `@SuperBuilder` (no `@Builder`) en AMBAS clases: la abstracta y la hija. Es obligatorio para que la herencia funcione con Lombok Builder
- `@CreatedDate` y `@LastModifiedDate` se llenan automaticamente via Spring Data Auditing
- `@CreatedBy` y `@LastModifiedBy` se llenan via `AuditorAware` (ver seccion siguiente)
- `@Version` para optimistic locking — previene escrituras concurrentes conflictivas
- TODA entidad MongoDB DEBE extender `AuditMetadata` — sin excepciones
- `@NoArgsConstructor(access = PROTECTED)` en la clase abstracta para que Spring Data pueda instanciarla

---

## Configuracion de Auditing

Habilitar la auditoria automatica de Spring Data MongoDB con una unica clase de configuracion.

```java
@Configuration
@EnableMongoAuditing
public class MongoAuditingConfig {

    @Bean
    public AuditorAware<String> auditorProvider() {
        return new AuditorAwareImpl();
    }
}
```

Para stacks reactivos:

```java
@Configuration
@EnableReactiveMongoAuditing
public class ReactiveMongoAuditingConfig {

    @Bean
    public ReactiveAuditorAware<String> reactiveAuditorProvider() {
        return () -> ReactiveSecurityContextHolder.getContext()
            .map(ctx -> ctx.getAuthentication().getName())
            .defaultIfEmpty("system");
    }
}
```

**Reglas:**
- Una sola clase `@Configuration` para toda la configuracion de auditing
- `@EnableMongoAuditing` para stacks imperativos, `@EnableReactiveMongoAuditing` para reactivos
- El `AuditorAware<String>` bean se registra aqui mismo
- El tipo generico (`String`) debe coincidir con el tipo de `@CreatedBy` y `@LastModifiedBy`

---

## AuditorAware Implementacion

`AuditorAware` proporciona el usuario actual que se escribe en `@CreatedBy` y `@LastModifiedBy`.

### Desde SecurityContext

```java
@Component
public class AuditorAwareImpl implements AuditorAware<String> {

    @Override
    public Optional<String> getCurrentAuditor() {
        return Optional.ofNullable(SecurityContextHolder.getContext())
            .map(SecurityContext::getAuthentication)
            .filter(Authentication::isAuthenticated)
            .map(Authentication::getName)
            .or(() -> Optional.of("system"));
    }
}
```

### Desde header custom

```java
@Component
public class AuditorAwareImpl implements AuditorAware<String> {

    private static final String USER_HEADER = "X-Authenticated-User";
    private static final String DEFAULT_USER = "system";

    @Override
    public Optional<String> getCurrentAuditor() {
        return Optional.ofNullable(RequestContextHolder.getRequestAttributes())
            .filter(ServletRequestAttributes.class::isInstance)
            .map(ServletRequestAttributes.class::cast)
            .map(ServletRequestAttributes::getRequest)
            .map(request -> request.getHeader(USER_HEADER))
            .filter(StringUtils::isNotBlank)
            .or(() -> Optional.of(DEFAULT_USER));
    }
}
```

**Reglas:**
- NUNCA retornar `Optional.empty()` — siempre proporcionar fallback (`"system"`) para procesos batch, schedulers, o peticiones sin autenticacion
- Si el usuario viene de SecurityContext: validar que `isAuthenticated()` sea true
- Si el usuario viene de un header: validar que no este blank
- Un solo `AuditorAwareImpl` por aplicacion (Spring Data solo acepta un bean de `AuditorAware`)
- La implementacion depende de como el proyecto gestiona la autenticacion — elegir entre SecurityContext o header segun el caso
