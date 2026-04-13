# JUnit & Mockito Patterns

## Table of Contents
1. [Configuracion Base](#configuracion-base)
2. [Estructura BDD](#estructura-bdd)
3. [Object Mothers](#object-mothers)
4. [Mocking con BDDMockito](#mocking-con-bddmockito)
5. [Assertions de Calidad](#assertions-de-calidad)
6. [Verificacion de Interacciones](#verificacion-de-interacciones)
7. [Parameterized Tests](#parameterized-tests)
8. [Nested Tests](#nested-tests)

---

## Configuracion Base

Siempre usar `MockitoExtension`. Nunca `@SpringBootTest` para tests unitarios.

```java
@ExtendWith(MockitoExtension.class)
class TransferServiceTest {

    @Mock
    private TransferRepository transferRepository;

    @Mock
    private AccountRepository accountRepository;

    @Mock
    private TransferValidator transferValidator;

    @Mock
    private TransferMapper transferMapper;

    @InjectMocks
    private TransferService transferService;
}
```

**Reglas:**
- `@ExtendWith(MockitoExtension.class)` para tests unitarios (no `@SpringBootTest`)
- `@Mock` para dependencias
- `@InjectMocks` para el SUT (System Under Test)
- `@Spy` solo cuando se necesite comportamiento parcial real
- `@Captor` para capturar argumentos complejos

---

## Estructura BDD

Todo test sigue Given/When/Then. Separar visualmente con comentarios.

```java
@Test
@DisplayName("Should complete transfer when accounts are valid and have sufficient funds")
void shouldCompleteTransfer_WhenAccountsValidAndSufficientFunds() {
    // Given
    final var request = TransferRequestMother.createValidRequest();
    final var sourceAccount = AccountMother.createWithBalance(new BigDecimal("5000.00"));
    final var targetAccount = AccountMother.createDefault();
    final var transfer = TransferMother.createPending();
    final var savedTransfer = TransferMother.createCompleted();
    final var expectedResponse = TransferResponseMother.createCompleted();

    given(accountRepository.findById(request.sourceAccountId()))
        .willReturn(Optional.of(sourceAccount));
    given(accountRepository.findById(request.targetAccountId()))
        .willReturn(Optional.of(targetAccount));
    given(transferMapper.toEntity(request)).willReturn(transfer);
    given(transferRepository.save(transfer)).willReturn(savedTransfer);
    given(transferMapper.toResponse(savedTransfer)).willReturn(expectedResponse);

    // When
    final var result = transferService.execute(request);

    // Then
    assertThat(result).isEqualTo(expectedResponse);
    then(transferValidator).should().validate(sourceAccount, targetAccount, request.amount());
    then(transferRepository).should().save(transfer);
}
```

**Reglas:**
- `// Given` — preparar datos y configurar mocks
- `// When` — ejecutar la accion (una sola linea, idealmente)
- `// Then` — verificar resultado y/o interacciones
- `@DisplayName` descriptivo en lenguaje de negocio
- Nombre del metodo: `shouldExpectedBehavior_WhenCondition`

---

## Object Mothers

Factoria de objetos de test. Un Mother por entidad/DTO.

```java
public final class TransferRequestMother {

    private static final String DEFAULT_SOURCE = "ACC-001";
    private static final String DEFAULT_TARGET = "ACC-002";
    private static final BigDecimal DEFAULT_AMOUNT = new BigDecimal("1000.00");

    private TransferRequestMother() {}

    public static TransferRequest createValidRequest() {
        return new TransferRequest(DEFAULT_SOURCE, DEFAULT_TARGET, DEFAULT_AMOUNT, "Test transfer");
    }

    public static TransferRequest createWithAmount(final BigDecimal amount) {
        return new TransferRequest(DEFAULT_SOURCE, DEFAULT_TARGET, amount, "Test transfer");
    }

    public static TransferRequest createWithSameSourceAndTarget() {
        return new TransferRequest(DEFAULT_SOURCE, DEFAULT_SOURCE, DEFAULT_AMOUNT, "Test");
    }

    public static TransferRequest createWithNullDescription() {
        return new TransferRequest(DEFAULT_SOURCE, DEFAULT_TARGET, DEFAULT_AMOUNT, null);
    }
}

public final class AccountMother {

    private static final String DEFAULT_ID = "ACC-001";
    private static final BigDecimal DEFAULT_BALANCE = new BigDecimal("5000.00");

    private AccountMother() {}

    public static Account createDefault() {
        return Account.builder()
            .id(DEFAULT_ID)
            .balance(DEFAULT_BALANCE)
            .status(AccountStatus.ACTIVE)
            .build();
    }

    public static Account createWithBalance(final BigDecimal balance) {
        return Account.builder()
            .id(DEFAULT_ID)
            .balance(balance)
            .status(AccountStatus.ACTIVE)
            .build();
    }

    public static Account createInactive() {
        return Account.builder()
            .id(DEFAULT_ID)
            .balance(DEFAULT_BALANCE)
            .status(AccountStatus.INACTIVE)
            .build();
    }
}
```

**Reglas:**
- Clase `final` con constructor privado (utility class)
- Constantes `private static final` para valores por defecto
- `createDefault()` / `createValid*()` para el caso feliz
- `createWith*()` para variaciones especificas
- `createInvalid*()` / `createWithNull*()` para casos de error
- Ubicar en paquete `mother` o `fixture` dentro de test
- Nombres: `{Entidad}Mother` — nunca `{Entidad}Factory` (evitar confusion con patron Factory de produccion)

---

## Mocking con BDDMockito

Usar siempre `BDDMockito` (`given`/`then`) en lugar de `Mockito` (`when`/`verify`).

### Configurar comportamiento (Given)
```java
// Retornar valor
given(repository.findById(anyString())).willReturn(Optional.of(account));

// Retornar diferentes valores en llamadas sucesivas
given(repository.findById(anyString()))
    .willReturn(Optional.of(sourceAccount))
    .willReturn(Optional.of(targetAccount));

// Lanzar excepcion
given(repository.findById("INVALID")).willThrow(new AccountNotFoundException("INVALID"));

// Metodo void que lanza excepcion
willThrow(new ValidationException("Invalid")).given(validator).validate(any());

// Respuesta dinamica
given(repository.save(any(Transfer.class)))
    .willAnswer(invocation -> {
        final var transfer = invocation.getArgument(0, Transfer.class);
        return transfer.toBuilder().id("GENERATED-ID").build();
    });
```

### Matchers
```java
// Basicos
any(), any(Transfer.class), anyString(), anyLong(), anyList()

// Especificos
eq("value"), eq(BigDecimal.ZERO)

// Argumentos custom
argThat(transfer -> transfer.amount().compareTo(BigDecimal.ZERO) > 0)

// Captura
@Captor ArgumentCaptor<Transfer> transferCaptor;
then(repository).should().save(transferCaptor.capture());
final var captured = transferCaptor.getValue();
assertThat(captured.amount()).isEqualByComparingTo(expectedAmount);
```

**Reglas:**
- `given()` de BDDMockito, nunca `when()` de Mockito
- `then().should()` de BDDMockito, nunca `verify()` de Mockito
- No mezclar matchers con valores literales: `eq("value")` si se usan matchers
- `@Captor` para argumentos complejos que necesitan assertions detalladas

---

## Assertions de Calidad

Usar AssertJ como libreria principal de assertions.

### Assertions basicas
```java
assertThat(result).isNotNull();
assertThat(result.name()).isEqualTo("expected");
assertThat(result.amount()).isEqualByComparingTo(new BigDecimal("100.00"));
assertThat(result.items()).hasSize(3);
assertThat(result.status()).isEqualTo(Status.ACTIVE);
```

### assertAll para multiples verificaciones
```java
assertAll(
    () -> assertThat(result.id()).isNotNull(),
    () -> assertThat(result.sourceAccountId()).isEqualTo(SOURCE_ACCOUNT_ID),
    () -> assertThat(result.amount()).isEqualByComparingTo(TRANSFER_AMOUNT),
    () -> assertThat(result.status()).isEqualTo(TransferStatus.COMPLETED)
);
```

### Excepciones con AssertJ
```java
// Verificar tipo y mensaje
assertThatThrownBy(() -> service.execute(invalidRequest))
    .isInstanceOf(InsufficientFundsException.class)
    .hasMessageContaining("Insufficient funds");

// Verificar que NO lanza excepcion
assertThatCode(() -> service.execute(validRequest))
    .doesNotThrowAnyException();
```

### Colecciones
```java
assertThat(results)
    .hasSize(3)
    .extracting(TransferResponse::status)
    .containsExactly(Status.COMPLETED, Status.PENDING, Status.FAILED);

assertThat(results)
    .filteredOn(r -> r.amount().compareTo(BigDecimal.TEN) > 0)
    .hasSize(2)
    .allMatch(r -> r.status() == Status.COMPLETED);
```

**Reglas:**
- AssertJ (`assertThat`) como assertion principal
- `assertAll` de JUnit para agrupar verificaciones relacionadas (todas se ejecutan aunque falle una)
- `isEqualByComparingTo()` para BigDecimal (ignora escala)
- `assertThatThrownBy()` para excepciones (nunca `@Test(expected=...)`)
- `extracting()` + `containsExactly()` para verificar propiedades de colecciones

---

## Verificacion de Interacciones

### Verificar llamadas
```java
// Se llamo exactamente 1 vez
then(repository).should().save(any(Transfer.class));
then(repository).should(times(1)).save(any());

// Nunca se llamo
then(repository).should(never()).delete(any());

// Al menos N veces
then(repository).should(atLeast(2)).findById(anyString());
```

### Verificar orden
```java
final var inOrder = inOrder(validator, repository, eventPublisher);
then(validator).should(inOrder).validate(any(), any(), any());
then(repository).should(inOrder).save(any());
then(eventPublisher).should(inOrder).publishEvent(any());
```

### No mas interacciones
```java
then(repository).should().save(any());
then(repository).shouldHaveNoMoreInteractions();
```

**Reglas:**
- Verificar interacciones solo cuando es relevante para el contrato del metodo
- `inOrder` cuando el orden de las operaciones importa (ej: validar antes de guardar)
- No sobre-verificar: un test no debe verificar cada llamada interna

---

## Parameterized Tests

Para probar la misma logica con diferentes datos de entrada.

```java
@ParameterizedTest
@DisplayName("Should reject transfer when amount exceeds limit")
@ValueSource(strings = {"10000.01", "50000.00", "999999.99"})
void shouldRejectTransfer_WhenAmountExceedsLimit(final String amount) {
    // Given
    final var request = TransferRequestMother.createWithAmount(new BigDecimal(amount));

    // When / Then
    assertThatThrownBy(() -> validator.validate(source, target, request.amount()))
        .isInstanceOf(TransferLimitExceededException.class);
}

@ParameterizedTest
@DisplayName("Should accept transfer for valid amounts")
@CsvSource({"100.00, COMPLETED", "5000.00, COMPLETED", "9999.99, COMPLETED"})
void shouldAcceptTransfer_ForValidAmounts(final String amount, final String expectedStatus) {
    // Given
    final var request = TransferRequestMother.createWithAmount(new BigDecimal(amount));
    // ...
}

@ParameterizedTest
@NullAndEmptySource
@ValueSource(strings = {" ", "  "})
void shouldRejectBlankAccountId(final String accountId) {
    // ...
}
```

**Reglas:**
- `@ParameterizedTest` + `@DisplayName` (no `@Test`)
- `@ValueSource` para un solo parametro
- `@CsvSource` para multiples parametros
- `@NullAndEmptySource` para null y empty
- `@MethodSource` para datos complejos
- Cada parametrizacion debe testear la misma logica con diferente input

---

## Nested Tests

Agrupar tests por escenario o metodo bajo test.

```java
@ExtendWith(MockitoExtension.class)
@DisplayName("TransferService")
class TransferServiceTest {

    @Mock private TransferRepository transferRepository;
    @Mock private TransferValidator transferValidator;
    @Mock private TransferMapper transferMapper;
    @InjectMocks private TransferService transferService;

    @Nested
    @DisplayName("execute")
    class Execute {

        @Test
        @DisplayName("Should complete transfer when all validations pass")
        void shouldCompleteTransfer_WhenAllValidationsPass() { ... }

        @Test
        @DisplayName("Should throw when source account not found")
        void shouldThrow_WhenSourceAccountNotFound() { ... }
    }

    @Nested
    @DisplayName("findById")
    class FindById {

        @Test
        @DisplayName("Should return transfer when exists")
        void shouldReturnTransfer_WhenExists() { ... }

        @Test
        @DisplayName("Should throw when transfer not found")
        void shouldThrow_WhenTransferNotFound() { ... }
    }
}
```

**Reglas:**
- `@Nested` por metodo publico del SUT
- `@DisplayName` en la clase nested con el nombre del metodo
- Tests del caso feliz primero, luego casos de error
- Los mocks del padre se comparten con los nested
