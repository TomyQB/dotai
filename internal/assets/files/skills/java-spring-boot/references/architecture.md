# Spring Boot Architecture Patterns

## Table of Contents
1. [Estructura de Capas](#estructura-de-capas)
2. [Controller](#controller)
3. [Service](#service)
4. [Validator](#validator)
5. [Mapper](#mapper)
6. [Repository](#repository)
7. [Exception Handling](#exception-handling)
8. [Entidades con Lombok](#entidades-con-lombok)
9. [Configuracion](#configuracion)

---

## Estructura de Capas

```
controller/        -> Recibe HTTP, delega al service, retorna response
  ├── dto/         -> Records con validacion Jakarta (request/response)
service/           -> Logica de negocio, orquesta validator + mapper + repository
  ├── validator/   -> Validacion de reglas de negocio (componentes separados)
  ├── mapper/      -> Transformacion entre DTOs y entidades
repository/        -> Acceso a datos (Spring Data JPA)
model/             -> Entidades JPA, enums, sealed classes de dominio
config/            -> Beans de configuracion, propiedades
exception/         -> Excepciones custom de dominio + GlobalExceptionHandler
```

**Reglas de dependencia:**
- Controller -> Service (nunca Repository directamente)
- Service -> Validator, Mapper, Repository
- Validator, Mapper: sin dependencias entre si
- Repository: sin dependencias a Service o Controller

---

## Controller

```java
@RestController
@RequestMapping("/api/v1/transfers")
@RequiredArgsConstructor
public class TransferController {

    private final TransferService transferService;

    @PostMapping
    @ResponseStatus(HttpStatus.CREATED)
    public TransferResponse create(@Valid @RequestBody final TransferRequest request) {
        return transferService.execute(request);
    }

    @GetMapping("/{id}")
    public TransferResponse findById(@PathVariable final String id) {
        return transferService.findById(id);
    }

    @GetMapping
    public List<TransferResponse> findByAccount(@RequestParam final String accountId) {
        return transferService.findByAccountId(accountId);
    }
}
```

**Reglas:**
- Solo recibir request, delegar al service, retornar response
- `@Valid` en `@RequestBody` para activar validacion Jakarta
- `@RequiredArgsConstructor` para inyeccion por constructor
- Sin logica de negocio, sin try-catch (lo maneja GlobalExceptionHandler)
- Un metodo por endpoint, metodos de maximo 3-5 lineas
- Versionado en la URL: `/api/v1/...`

---

## Service

```java
@Service
@RequiredArgsConstructor
public class TransferService {

    private final TransferRepository transferRepository;
    private final AccountRepository accountRepository;
    private final TransferValidator transferValidator;
    private final TransferMapper transferMapper;

    @Transactional
    public TransferResponse execute(final TransferRequest request) {
        final var source = findAccountOrThrow(request.sourceAccountId());
        final var target = findAccountOrThrow(request.targetAccountId());
        transferValidator.validate(source, target, request.amount());
        final var transfer = transferMapper.toEntity(request);
        return transferMapper.toResponse(transferRepository.save(transfer));
    }

    @Transactional(readOnly = true)
    public TransferResponse findById(final String id) {
        final var transfer = transferRepository.findByIdOrThrow(id);
        return transferMapper.toResponse(transfer);
    }

    private Account findAccountOrThrow(final String accountId) {
        return accountRepository.findByIdOrThrow(accountId);
    }
}
```

**Reglas:**
- `@Transactional` en metodos que modifican datos
- `@Transactional(readOnly = true)` en metodos de solo lectura
- Orquestar validator, mapper, repository. No implementar logica de transformacion ni validacion aqui
- Metodos pequenos (max 10 lineas de logica)
- Nombres de metodos que expresen la intencion de negocio

---

## Validator

Componente separado para validacion de reglas de negocio (no Jakarta).

```java
@Component
public class TransferValidator {

    private static final BigDecimal MAX_SINGLE_TRANSFER = new BigDecimal("10000.00");

    public void validate(final Account source, final Account target, final BigDecimal amount) {
        Objects.requireNonNull(source, "Source account cannot be null");
        Objects.requireNonNull(target, "Target account cannot be null");
        Objects.requireNonNull(amount, "Amount cannot be null");

        if (source.id().equals(target.id())) {
            throw new InvalidTransferException("Source and target must be different");
        }
        if (amount.compareTo(MAX_SINGLE_TRANSFER) > 0) {
            throw new TransferLimitExceededException(amount, MAX_SINGLE_TRANSFER);
        }
        if (source.balance().compareTo(amount) < 0) {
            throw new InsufficientFundsException(source.id(), source.balance(), amount);
        }
    }
}
```

**Reglas:**
- `@Component` independiente, uno por entidad/operacion compleja
- Metodo `validate()` lanza excepcion custom si falla
- `Objects.requireNonNull()` para precondiciones de null
- Constantes `private static final` para limites y configuracion
- Sin retorno (void) — lanza excepciones custom especificas
- Nombre descriptivo: `TransferValidator`, `AccountValidator`

---

## Mapper

Componente separado para transformacion entre DTOs y entidades.

```java
@Component
public class TransferMapper {

    public Transfer toEntity(final TransferRequest request) {
        return Transfer.builder()
            .sourceAccountId(request.sourceAccountId())
            .targetAccountId(request.targetAccountId())
            .amount(request.amount())
            .status(TransferStatus.PENDING)
            .createdAt(Instant.now())
            .build();
    }

    public TransferResponse toResponse(final Transfer transfer) {
        return TransferResponse.builder()
            .id(transfer.getId())
            .sourceAccountId(transfer.getSourceAccountId())
            .targetAccountId(transfer.getTargetAccountId())
            .amount(transfer.getAmount())
            .status(transfer.getStatus())
            .createdAt(transfer.getCreatedAt())
            .build();
    }

    public List<TransferResponse> toResponseList(final List<Transfer> transfers) {
        return transfers.stream()
            .map(this::toResponse)
            .toList();
    }
}
```

**Reglas:**
- `@Component` independiente, uno por entidad principal
- `toEntity()`: DTO -> Entity
- `toResponse()`: Entity -> DTO
- `toResponseList()`: usar stream + method reference
- Sin logica de negocio, solo transformacion de datos
- Usar `@Builder` para construir objetos complejos

---

## Repository

```java
public interface TransferRepository extends JpaRepository<Transfer, String> {

    List<Transfer> findBySourceAccountIdOrTargetAccountId(String sourceId, String targetId);

    @Query("SELECT t FROM Transfer t WHERE t.status = :status AND t.createdAt > :since")
    List<Transfer> findRecentByStatus(
        @Param("status") TransferStatus status,
        @Param("since") Instant since
    );

    Optional<Transfer> findByIdAndSourceAccountId(String id, String sourceAccountId);
}
```

**Reglas:**
- Extender `JpaRepository` (o `CrudRepository` si no se necesita paginacion)
- Derived queries para consultas simples
- `@Query` con JPQL para consultas complejas
- Retornar `Optional` para busquedas por ID unico
- Retornar `List` para busquedas multiples (nunca null)
- Sin `@Repository` (Spring Data lo detecta automaticamente)
- **Resolver Optional en el Repository, no en el Service:** Cuando existe una capa custom (Repository interface + RepositoryImpl), el `orElseThrow()` se hace en el RepositoryImpl. El Service recibe tipos concretos, nunca `Optional`

### Repository custom (interface + impl)

Cuando el acceso a datos requiere logica adicional (transformar entity, queries complejas), crear una capa custom:

```java
// Interfaz — retorna tipos concretos, nunca Optional
public interface TransferRepository {
    Transfer findByIdOrThrow(String id);
    String findUrlByMerchantId(String merchantId);
}

// Implementacion — resuelve Optional internamente
@Repository
@RequiredArgsConstructor
public class TransferRepositoryImpl implements TransferRepository {

    private final TransferJpaRepository jpaRepository;

    @Override
    public Transfer findByIdOrThrow(final String id) {
        return jpaRepository.findById(id)
            .orElseThrow(() -> new TransferNotFoundException(id));
    }

    @Override
    public String findUrlByMerchantId(final String merchantId) {
        return jpaRepository.findByMerchantId(merchantId)
            .map(Entity::getUrl)
            .orElseThrow(() -> new NotFoundException("Not found: " + merchantId));
    }
}
```

El Service llama directamente sin gestionar Optional:
```java
final var transfer = transferRepository.findByIdOrThrow(id);
```

---

## Exception Handling

### Excepciones custom de dominio
```java
public class TransferNotFoundException extends RuntimeException {
    public TransferNotFoundException(final String id) {
        super("Transfer not found with id: %s".formatted(id));
    }
}

public class InsufficientFundsException extends RuntimeException {
    public InsufficientFundsException(final String accountId, final BigDecimal balance,
            final BigDecimal requested) {
        super("Insufficient funds in account %s: balance=%s, requested=%s"
            .formatted(accountId, balance, requested));
    }
}

public class TransferLimitExceededException extends RuntimeException {
    public TransferLimitExceededException(final BigDecimal amount, final BigDecimal limit) {
        super("Transfer amount %s exceeds limit %s".formatted(amount, limit));
    }
}
```

### GlobalExceptionHandler
```java
@RestControllerAdvice
@Slf4j
public class GlobalExceptionHandler {

    @ExceptionHandler(TransferNotFoundException.class)
    @ResponseStatus(HttpStatus.NOT_FOUND)
    public ErrorResponse handleNotFound(final TransferNotFoundException ex) {
        return new ErrorResponse(HttpStatus.NOT_FOUND.value(), ex.getMessage());
    }

    @ExceptionHandler(InsufficientFundsException.class)
    @ResponseStatus(HttpStatus.UNPROCESSABLE_ENTITY)
    public ErrorResponse handleInsufficientFunds(final InsufficientFundsException ex) {
        return new ErrorResponse(HttpStatus.UNPROCESSABLE_ENTITY.value(), ex.getMessage());
    }

    @ExceptionHandler(MethodArgumentNotValidException.class)
    @ResponseStatus(HttpStatus.BAD_REQUEST)
    public ErrorResponse handleValidation(final MethodArgumentNotValidException ex) {
        final var errors = ex.getBindingResult().getFieldErrors().stream()
            .map(error -> "%s: %s".formatted(error.getField(), error.getDefaultMessage()))
            .toList();
        return new ErrorResponse(HttpStatus.BAD_REQUEST.value(), "Validation failed", errors);
    }

    @ExceptionHandler(Exception.class)
    @ResponseStatus(HttpStatus.INTERNAL_SERVER_ERROR)
    public ErrorResponse handleGeneral(final Exception ex) {
        log.error("Unexpected error", ex);
        return new ErrorResponse(HttpStatus.INTERNAL_SERVER_ERROR.value(), "Internal server error");
    }
}

public record ErrorResponse(int status, String message, List<String> errors) {
    public ErrorResponse(final int status, final String message) {
        this(status, message, List.of());
    }
}
```

**Reglas:**
- Una excepcion custom por tipo de error de negocio
- Extender `RuntimeException` (unchecked)
- Mensaje descriptivo con `String.formatted()`
- `@RestControllerAdvice` centraliza el manejo de errores
- Mapear excepcion -> HTTP status code apropiado
- Log en errores inesperados, no en errores de negocio esperados
- `ErrorResponse` como record inmutable

---

## Entidades con Lombok

Usar Lombok para clases de entidad JPA (no records, porque JPA necesita mutabilidad controlada).

```java
@Entity
@Table(name = "transfers")
@Getter
@Builder
@NoArgsConstructor(access = AccessLevel.PROTECTED)
@AllArgsConstructor(access = AccessLevel.PRIVATE)
public class Transfer {

    @Id
    @GeneratedValue(strategy = GenerationType.UUID)
    private String id;

    @Column(nullable = false)
    private String sourceAccountId;

    @Column(nullable = false)
    private String targetAccountId;

    @Column(nullable = false, precision = 17, scale = 2)
    private BigDecimal amount;

    @Enumerated(EnumType.STRING)
    @Column(nullable = false)
    private TransferStatus status;

    @Column(nullable = false, updatable = false)
    private Instant createdAt;
}
```

**Reglas:**
- `@Getter` (nunca `@Setter` — inmutabilidad)
- `@Builder` para construccion fluida
- `@NoArgsConstructor(access = PROTECTED)` requerido por JPA
- `@AllArgsConstructor(access = PRIVATE)` para el Builder
- Sin `@Data` (incluye `@Setter` y `equals/hashCode` problematicos con JPA)
- Sin `@ToString` en entidades (riesgo de lazy loading)
- Mutacion controlada via metodos de negocio especificos, no setters

---

## Configuracion

### Properties con @ConfigurationProperties
```java
@ConfigurationProperties(prefix = "app.transfer")
public record TransferProperties(
    @DefaultValue("10000.00") BigDecimal maxAmount,
    @DefaultValue("100") int dailyLimit,
    @DefaultValue("PT30M") Duration timeout
) {}
```

### Habilitar en Application
```java
@SpringBootApplication
@EnableConfigurationProperties(TransferProperties.class)
public class Application { ... }
```

### Inyectar en servicios
```java
@Service
@RequiredArgsConstructor
public class TransferService {
    private final TransferProperties transferProperties;
    // usar transferProperties.maxAmount(), etc.
}
```

**Reglas:**
- `record` para properties (inmutable)
- `@DefaultValue` para valores por defecto
- Inyectar como dependencia normal
- Agrupar propiedades relacionadas bajo un prefijo comun
