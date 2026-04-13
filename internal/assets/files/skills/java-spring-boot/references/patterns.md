# Java Modern Patterns

## Table of Contents
1. [Records para DTOs](#records-para-dtos)
2. [Sealed Classes](#sealed-classes)
3. [Pattern Matching](#pattern-matching)
4. [Inmutabilidad](#inmutabilidad)
5. [Programacion Funcional](#programacion-funcional)
6. [Virtual Threads](#virtual-threads)

---

## Records para DTOs

Usar `record` para todos los DTOs. Combinar con `@Builder` de Lombok para facilitar la creacion.

```java
@Builder
public record TransferRequest(
    @NotBlank(message = "Source account ID is required") String sourceAccountId,
    @NotBlank(message = "Target account ID is required") String targetAccountId,
    @NotNull @Positive @Digits(integer = 15, fraction = 2) BigDecimal amount,
    @Size(max = 140) String description
) {}

@Builder
public record TransferResponse(
    String id,
    String sourceAccountId,
    String targetAccountId,
    BigDecimal amount,
    TransferStatus status,
    Instant createdAt
) {}
```

**Reglas:**
- Siempre `record` para request/response DTOs
- `@Builder` de Lombok para records con 3+ campos
- Validacion Jakarta en los campos del request record
- Response records sin validacion (datos ya validados)
- Sin logica de negocio dentro del record

---

## Sealed Classes

Usar `sealed` para jerarquias cerradas donde se conocen todos los subtipos.

```java
public sealed interface Account permits SavingsAccount, CheckingAccount, InvestmentAccount {
    String id();
    BigDecimal balance();
    AccountType type();
}

public record SavingsAccount(String id, BigDecimal balance, BigDecimal interestRate)
    implements Account {
    @Override
    public AccountType type() { return AccountType.SAVINGS; }
}

public record CheckingAccount(String id, BigDecimal balance, BigDecimal overdraftLimit)
    implements Account {
    @Override
    public AccountType type() { return AccountType.CHECKING; }
}

public record InvestmentAccount(String id, BigDecimal balance, RiskLevel riskLevel)
    implements Account {
    @Override
    public AccountType type() { return AccountType.INVESTMENT; }
}
```

**Reglas:**
- `sealed interface` + `permits` para listar subtipos explicitamente
- Subtipos como `record` cuando son inmutables (preferido)
- Subtipos como `final class` cuando necesitan mutabilidad controlada
- Combinar con pattern matching en `switch`

---

## Pattern Matching

Usar pattern matching con `switch` expressions para logica basada en tipos.

```java
public BigDecimal calculateFee(final Account account) {
    return switch (account) {
        case SavingsAccount s -> s.balance().multiply(SAVINGS_FEE_RATE);
        case CheckingAccount c -> c.balance().multiply(CHECKING_FEE_RATE);
        case InvestmentAccount i -> i.balance().multiply(INVESTMENT_FEE_RATE);
    };
}

public String formatNotification(final Notification notification) {
    return switch (notification) {
        case EmailNotification e -> "Email to %s: %s".formatted(e.recipient(), e.subject());
        case SmsNotification s -> "SMS to %s: %s".formatted(s.phoneNumber(), s.message());
        case PushNotification p -> "Push to %s: %s".formatted(p.deviceToken(), p.title());
    };
}
```

**Reglas:**
- `switch` expression (con `->`) en lugar de `switch` statement
- Sin `default` cuando sealed classes cubren todos los casos (el compilador lo verifica)
- Usar `default` solo cuando el conjunto de casos no es exhaustivo
- Pattern variables con nombres descriptivos cortos (`s`, `c`, `i` para 1 linea; nombre completo para logica compleja)

---

## Inmutabilidad

Todo debe ser inmutable por defecto. La mutabilidad es la excepcion y debe justificarse.

### Variables siempre `final`
```java
public TransferResponse execute(final TransferRequest request) {
    final var source = accountRepository.findById(request.sourceAccountId());
    final var target = accountRepository.findById(request.targetAccountId());
    final var transfer = transferMapper.toEntity(request);
    return transferMapper.toResponse(transferRepository.save(transfer));
}
```

### Colecciones inmutables
```java
// Crear listas inmutables
final var accounts = List.of(account1, account2, account3);
final var filtered = accounts.stream()
    .filter(a -> a.balance().compareTo(BigDecimal.ZERO) > 0)
    .toList(); // Returns unmodifiable list

// Crear maps inmutables
final var config = Map.of("key1", "value1", "key2", "value2");

// Copiar coleccion haciendola inmutable
final var immutableCopy = List.copyOf(mutableList);
final var immutableMap = Map.copyOf(mutableMap);
```

### Parametros siempre `final`
```java
public Account findById(final String accountId) { ... }
public void validate(final Account source, final Account target, final BigDecimal amount) { ... }
```

**Reglas:**
- Todo parametro es `final`
- Toda variable local es `final`
- Usar `var` con `final` para inferencia de tipos: `final var x = ...`
- Colecciones: `List.of()`, `Map.of()`, `Set.of()`, `.toList()`, `Collections.unmodifiable*()`
- Records para objetos de datos (inmutables por naturaleza)
- Clases de entidad: campos `final` donde sea posible, Lombok `@Builder` para construccion

---

## Programacion Funcional

Preferir estilo funcional sobre imperativo. Usar streams, Optional, y composicion de funciones.

### Streams sobre loops
```java
// Correcto: funcional
final var activeAccounts = accounts.stream()
    .filter(Account::isActive)
    .map(transferMapper::toResponse)
    .toList();

// Incorrecto: imperativo
final var result = new ArrayList<AccountResponse>();
for (final var account : accounts) {
    if (account.isActive()) {
        result.add(transferMapper.toResponse(account));
    }
}
```

### Optional en lugar de null
```java
public Optional<Account> findById(final String id) {
    return Optional.ofNullable(accountRepository.findById(id));
}

// Encadenar operaciones
final var accountName = findById(id)
    .filter(Account::isActive)
    .map(Account::name)
    .orElseThrow(() -> new AccountNotFoundException(id));
```

### Composicion de funciones
```java
final Function<String, Account> findAndValidate = ((Function<String, Account>) this::findAccount)
    .andThen(this::validateAccount);

// Predicados compuestos
final Predicate<Account> isEligible = Account::isActive
    .and(a -> a.balance().compareTo(minimumBalance) >= 0)
    .and(a -> Objects.isNull(a.suspendedAt()));
```

### Utilities: Objects y StringUtils
```java
// Objects para null checks
Objects.requireNonNull(account, "Account cannot be null");
Objects.isNull(account.suspendedAt());
Objects.nonNull(account.email());
Objects.requireNonNullElse(name, "Unknown");

// StringUtils (Apache Commons / Spring)
StringUtils.isNotBlank(account.email());
StringUtils.hasText(request.description());
StringUtils.trimWhitespace(input);
```

**Reglas:**
- `Stream` sobre `for`/`while` para transformaciones de colecciones
- `Optional` como retorno, nunca como parametro ni campo
- `Objects.requireNonNull()` para validacion de precondiciones
- `Objects.isNull()` / `Objects.nonNull()` sobre `== null` / `!= null`
- `StringUtils.isNotBlank()` sobre `str != null && !str.isEmpty()`
- Method references (`Class::method`) sobre lambdas cuando sea posible
- Evitar side-effects dentro de streams

---

## Virtual Threads

Usar Virtual Threads (Project Loom) para operaciones I/O-bound.

### Configuracion del executor
```java
@Configuration
public class AsyncConfig {

    @Bean
    public ExecutorService virtualThreadExecutor() {
        return Executors.newVirtualThreadPerTaskExecutor();
    }
}
```

### Uso en servicios
```java
@Service
@RequiredArgsConstructor
public class NotificationService {

    private final ExecutorService virtualThreadExecutor;

    public CompletableFuture<Void> sendNotificationAsync(final Notification notification) {
        return CompletableFuture.runAsync(
            () -> sendNotification(notification),
            virtualThreadExecutor
        );
    }

    public <T> CompletableFuture<T> executeAsync(final Supplier<T> task) {
        return CompletableFuture.supplyAsync(task, virtualThreadExecutor);
    }
}
```

### Spring Boot con Virtual Threads
```yaml
# application.yml
spring:
  threads:
    virtual:
      enabled: true
```

**Reglas:**
- Activar `spring.threads.virtual.enabled: true` en Spring Boot 3.2+
- `Executors.newVirtualThreadPerTaskExecutor()` para tareas async custom
- Virtual threads para I/O (HTTP calls, DB queries, file operations)
- NO usar virtual threads para CPU-bound (usar platform threads)
- No usar `synchronized` con virtual threads (usar `ReentrantLock`)
